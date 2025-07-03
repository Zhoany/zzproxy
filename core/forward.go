package core

import (
    "crypto/tls"
    "fmt"
    "net/url"
    "strings"
    "time"

    "github.com/miekg/dns"
    "zzproxy/config"
)

// ForwardQuery takes a domain name and type, finds the matching upstream by suffix,
// checks IPv6 support for AAAA queries, then forwards the query over UDP or TLS.
// If no upstream matches, it returns (nil, nil) so the caller can handle fallback.
func ForwardQuery(name string, qtype uint16, cfg *config.Config) (*dns.Msg, error) {
    // Prepare DNS message template
    msg := new(dns.Msg)
    msg.SetQuestion(dns.Fqdn(name), qtype)

    // 1) Match specific upstream by longest suffix
    var selected *config.Upstream
    for i := range cfg.Forward {
        up := &cfg.Forward[i]
        if _, ok := up.Trie.MatchLongest(name); ok {
            selected = up
            break
        }
    }
    // 2) If no upstream matched, hand off to caller
    if selected == nil || selected.Server == "" {
        return nil, nil
    }

    // 3) IPv6 support check for AAAA queries
    if qtype == dns.TypeAAAA && !selected.V6 {
        reply := new(dns.Msg)
        reply.SetQuestion(dns.Fqdn(name), qtype)
        reply.Rcode = dns.RcodeNameError
        return reply, nil
    }

    // Parse upstream URI
    u, err := url.Parse(selected.Server)
    if err != nil {
        return nil, fmt.Errorf("invalid upstream URI '%s': %w", selected.Server, err)
    }

    // Setup DNS client with timeouts
    client := new(dns.Client)
    client.ReadTimeout = 2 * time.Second
    client.WriteTimeout = 2 * time.Second

    var network, address string
    switch strings.ToLower(u.Scheme) {
    case "udp":
        network = "udp"
        address = u.Host
    case "tls", "dot":
        network = "tcp-tls"
        address = u.Host
        client.TLSConfig = &tls.Config{
            InsecureSkipVerify: false,
            ServerName:         u.Hostname(),
        }
    default:
        return nil, fmt.Errorf("unsupported protocol '%s'", u.Scheme)
    }

    // Forward the query
    resp, _, err := client.Exchange(msg, address)
    if err != nil {
        return nil, fmt.Errorf("forward to %s failed: %w", selected.Server, err)
    }
    return resp, nil
}
