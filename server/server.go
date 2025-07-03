package server

import (
	"log"
	
	"net/http"
	"strings"
	"time"

	"zzproxy/config"
	"zzproxy/core"

	"github.com/miekg/dns"
	gocache "github.com/patrickmn/go-cache"
)
var cache     = gocache.New(60*time.Second, 120*time.Second) // 默认缓存 TTL 60s, 清理间隔 120s
// Run 启动 DNS 服务和 API HTTP 服务
func Run(cfg *config.Config) {
	// 注册 DNS 请求处理器
	dns.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		handleDNS(w, r, cfg)
	})

	addr := cfg.Server.DNSPort
	// 并发启动 UDP4/UDP6/TCP4/TCP6
	for _, netType := range []string{"udp4", "udp6", "tcp4", "tcp6"} {
		go func(nt string) {
			if err := StartServer(nt, addr); err != nil {
				log.Printf("[DNS %s] 启动失败: %v", nt, err)
			}
		}(netType)
	}
	log.Printf("启动 DNS 服务器，监听 %s (UDP4/UDP6/TCP4/TCP6)", addr)

	// 启动 API HTTP 服务
	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		// 简单 Token 验证
		auth := r.Header.Get("Authorization")
		expected := "Bearer " + cfg.API.Token
		if auth != expected {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		// TODO: 根据具体 API 路径分发逻辑
		w.Write([]byte("API server is running"))
	})
	go func() {
		log.Printf("启动 API 服务器，监听 %s", cfg.Server.HTTPPort)
		if err := http.ListenAndServe(cfg.Server.HTTPPort, nil); err != nil {
			log.Fatalf("API 服务启动失败: %v", err)
		}
	}()
}

// StartServer 启动单个网络类型的 DNS 服务
func StartServer(netType, addr string) error {
	srv := &dns.Server{Addr: addr, Net: netType}
	log.Printf("Starting %s server on %s (IPv6 enabled: %v)", netType, addr, strings.HasSuffix(netType, "6"))
	return srv.ListenAndServe()
}

func handleDNS(w dns.ResponseWriter, r *dns.Msg, cfg *config.Config) {
    m := new(dns.Msg)
    m.SetReply(r)
    q := r.Question[0]
    name := strings.TrimSuffix(q.Name, ".")

    // 1. 缓存
    key := q.Name + "|" + dns.TypeToString[q.Qtype]
    if cached, found := cache.Get(key); found {
        if msg, ok := cached.(*dns.Msg); ok {
            msg.Id = r.Id
            w.WriteMsg(msg)
            return
        }
    }

    // 2. 黑名单
    if core.BlacklistCheck(name, cfg.Blacklist.Entries) {
        m.Rcode = dns.RcodeRefused
        w.WriteMsg(m)
        return
    }

    // 3. Host 配置
    if rr, ok := core.HostLookup(name, cfg.Host.Hosts, q); ok {
        m.Answer = append(m.Answer, rr)
        cache.Set(key, m.Copy(), gocache.DefaultExpiration)
        w.WriteMsg(m)
        return
    }

    // 4. 上游转发（调用 core.ForwardQuery）
    resp, err := core.ForwardQuery(name, q.Qtype, cfg)
    if err != nil {
        log.Printf("转发失败: %v", err)
        m.Rcode = dns.RcodeServerFailure
        w.WriteMsg(m)
        return
    }
    // 如果 resp == nil，表示未匹配任何上游，交由后续逻辑处理
    if resp == nil {
        // 交给下一个处理器或默认逻辑（例如 NXDOMAIN）
        // 这里简单返回 NameError
        
    }
	
    // 5. 填充并缓存
    m.Answer = resp.Answer
    m.Extra = resp.Extra
    m.Rcode = resp.Rcode
    cache.Set(key, m.Copy(), gocache.DefaultExpiration)
    w.WriteMsg(m)
}
