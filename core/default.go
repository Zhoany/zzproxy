package core

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/miekg/dns"
	"zzproxy/config"
)

// DefaultQuery sends the request to domestic DNS servers first. If the returned
// IPs fall within the configured China CIDR list, their response is used.
// Otherwise it falls back to foreign DNS servers.
func DefaultQuery(name string, qtype uint16, cfg *config.Config) (*dns.Msg, error) {
	// try domestic servers
	for i := range cfg.Default.Domestic {
		resp, err := queryUpstream(&cfg.Default.Domestic[i], name, qtype)
		if err != nil {
			continue
		}
		if qtype == dns.TypeA || qtype == dns.TypeAAAA {
			if resp != nil && responseInCIDRs(resp.Answer, cfg.Default.CIDRs) {
				return resp, nil
			}
		} else if resp != nil {
			return resp, nil
		}
	}

	// fallback to foreign servers
	for i := range cfg.Default.Foreign {
		resp, err := queryUpstream(&cfg.Default.Foreign[i], name, qtype)
		if err != nil {
			continue
		}
		if resp != nil {
			return resp, nil
		}
	}
	return nil, fmt.Errorf("all default upstreams failed")
}

func queryUpstream(up *config.Upstream, name string, qtype uint16) (*dns.Msg, error) {
	if up.Server == "" {
		return nil, fmt.Errorf("empty upstream server")
	}
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(name), qtype)

	u, err := url.Parse(up.Server)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream URI '%s': %w", up.Server, err)
	}

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

	resp, _, err := client.Exchange(msg, address)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func responseInCIDRs(rrs []dns.RR, cidrs []*net.IPNet) bool {
	for _, rr := range rrs {
		switch v := rr.(type) {
		case *dns.A:
			if ipInCIDRs(v.A, cidrs) {
				return true
			}
		case *dns.AAAA:
			if ipInCIDRs(v.AAAA, cidrs) {
				return true
			}
		}
	}
	return false
}

func ipInCIDRs(ip net.IP, cidrs []*net.IPNet) bool {
	for _, c := range cidrs {
		if c.Contains(ip) {
			return true
		}
	}
	return false
}
