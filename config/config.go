// config/config.go
package config

import (
    "bufio"
    "fmt"

    "net"
    "os"
    "strings"

    "gopkg.in/yaml.v3"
)

// -----------------------------
// 配置结构体定义
// -----------------------------
type ServerConfig struct {
    DNSPort  string `yaml:"dnsport"`
    HTTPPort string `yaml:"httpport"`
}

type Upstream struct {
    Server  string   `yaml:"server"`
    V6      bool     `yaml:"v6"`
    File    string   `yaml:"file,omitempty"`
    Entries []string `yaml:"-"`
    Trie    *Trie    `yaml:"-"`
}

type DefaultConfig struct {
    ChinaIPCidrsFile string       `yaml:"china_ip_cidrs_file"`
    Domestic         []Upstream   `yaml:"domestic"`
    Foreign          []Upstream   `yaml:"foreign"`
    CIDRs            []*net.IPNet `yaml:"-"`
}

type HostConfig struct {
    File  string            `yaml:"file"`
    Hosts map[string]string `yaml:"-"`
}

type BlacklistConfig struct {
    File    string   `yaml:"file"`
    Entries []string `yaml:"-"`
}

type APIConfig struct {
    Token string `yaml:"token"`
}

type LogConfig struct {
    Level string `yaml:"level"`
    File  string `yaml:"file"`
}

type Config struct {
    Server    ServerConfig    `yaml:"server"`
    Default   DefaultConfig   `yaml:"default"`
    Host      HostConfig      `yaml:"host"`
    Blacklist BlacklistConfig `yaml:"blacklist"`
    API       APIConfig       `yaml:"api"`
    Log       LogConfig       `yaml:"log"`
    Forward   []Upstream      `yaml:"forward"`
}

// -----------------------------
// LoadConfig 读取并解析配置，并保证 host/blacklist/forward 均可为空
// -----------------------------
func LoadConfig(path string) (*Config, error) {
    // 1. 从磁盘读 YAML
    raw, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var cfg Config
    if err := yaml.Unmarshal(raw, &cfg); err != nil {
        return nil, err
    }

    // 2. 初始化为「空」，避免后续 nil map 或 nil slice
    cfg.Host.Hosts = make(map[string]string)
    cfg.Blacklist.Entries = make([]string, 0)
    if cfg.Forward == nil {
        cfg.Forward = make([]Upstream, 0)
    }
    if cfg.Default.Domestic == nil {
        cfg.Default.Domestic = make([]Upstream, 0)
    }
    if cfg.Default.Foreign == nil {
        cfg.Default.Foreign = make([]Upstream, 0)
    }

    // 3. 只有在 file 字段非空时，才真正去加载文件
    if cfg.Host.File != "" {
        hosts, err := readHostFile(cfg.Host.File)
        if err != nil {
            return nil, fmt.Errorf("解析 host 文件失败: %w", err)
        }
        cfg.Host.Hosts = hosts
    }

    if cfg.Blacklist.File != "" {
        lines, err := readLines(cfg.Blacklist.File)
        if err != nil {
            return nil, fmt.Errorf("解析 blacklist 文件失败: %w", err)
        }
        cfg.Blacklist.Entries = lines
    }

    for i, up := range cfg.Forward {
        if up.File != "" {
            lines, err := readLines(up.File)
            if err != nil {
                return nil, fmt.Errorf("解析 forward[%d].file 失败: %w", i, err)
            }
            cfg.Forward[i].Entries = lines
        } else {
            cfg.Forward[i].Entries = make([]string, 0)
        }
        trie := NewTrie()
        for _, entry := range cfg.Forward[i].Entries {
            trie.Insert(entry)
        }
        cfg.Forward[i].Trie = trie
    }

    if cfg.Default.ChinaIPCidrsFile != "" {
        cidrs, err := readCIDRs(cfg.Default.ChinaIPCidrsFile)
        if err != nil {
            return nil, fmt.Errorf("解析 china_ip_cidrs_file 失败: %w", err)
        }
        cfg.Default.CIDRs = cidrs
    } else {
        cfg.Default.CIDRs = make([]*net.IPNet, 0)
    }

    return &cfg, nil
}

// -----------------------------
// readHostFile 支持 “IP 域名1 域名2…” 格式，加载到 map
// -----------------------------
func readHostFile(path string) (map[string]string, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    hosts := make(map[string]string)
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        fields := strings.Fields(line)
        if len(fields) < 2 {
            continue
        }
        ip := fields[0]
        for _, domain := range fields[1:] {
            hosts[domain] = ip
        }
    }
    if err := scanner.Err(); err != nil {
        return nil, err
    }
    return hosts, nil
}

// -----------------------------
// readLines 去除空行和注释，返回字符串切片
// -----------------------------
func readLines(path string) ([]string, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    var lines []string
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        lines = append(lines, line)
    }
    if err := scanner.Err(); err != nil {
        return nil, err
    }
    return lines, nil
}

// readCIDRs parses CIDR blocks from file
func readCIDRs(path string) ([]*net.IPNet, error) {
    lines, err := readLines(path)
    if err != nil {
        return nil, err
    }
    var cidrs []*net.IPNet
    for _, line := range lines {
        if _, ipnet, err := net.ParseCIDR(line); err == nil {
            cidrs = append(cidrs, ipnet)
        } else {
            return nil, fmt.Errorf("invalid CIDR %s: %w", line, err)
        }
    }
    return cidrs, nil
}
