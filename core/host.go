package core

import (
    "net"
    "strings"

    "github.com/miekg/dns"
)

// Lookup checks if the queried name exists in the hosts map.
// If found, it returns the constructed RR record and true, otherwise (nil, false).
func HostLookup(name string, hosts map[string]string, q dns.Question) (dns.RR, bool) {
    if ipStr, ok := hosts[name]; ok {
        hdr := dns.RR_Header{
            Name:   dns.Fqdn(name),
            Class:  dns.ClassINET,
            Ttl:     0,
        }
        var rr dns.RR
        if strings.Contains(ipStr, ":") {
            // IPv6
            hdr.Rrtype = dns.TypeAAAA
            rr = &dns.AAAA{Hdr: hdr, AAAA: net.ParseIP(ipStr)}
        } else {
            // IPv4
            hdr.Rrtype = dns.TypeA
            rr = &dns.A{Hdr: hdr, A: net.ParseIP(ipStr)}
        }
        return rr, true
    }
    return nil, false
}
