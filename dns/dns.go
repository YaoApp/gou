package dns

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"sync"

	"github.com/miekg/dns"
	"github.com/yaoapp/kun/log"
)

// DNS cache
var caches = map[string][]string{}

// LookupIP looks up host using the local resolver. It returns a slice of that host's IPv4 and IPv6 addresses.
func LookupIP(host string, ipv6 ...bool) ([]string, error) {

	if ip := net.ParseIP(host); ip != nil { // the given host is ip
		return []string{ip.String()}, nil
	}

	if ipv6 == nil {
		ipv6 = []bool{true}
	}

	// the host was cached
	cache := fmt.Sprintf("%s_%v", host, ipv6[0])
	if ips, has := caches[cache]; has {
		return ips, nil
	}

	if runtime.GOOS == "linux" {
		conf, err := DefaultConfig()
		if err != nil {
			return []string{}, err
		}

		res, err := linuxLookupIP(host, conf.Servers, conf.Port, ipv6[0])
		if err != nil {
			return nil, err
		}

		// cache the host resolved result ( for linux )
		if len(res) > 0 {
			caches[cache] = res
		}

		return res, nil
	}

	var ips = []net.IP{}
	var err error

	if !ipv6[0] {
		ips, err = net.DefaultResolver.LookupIP(context.Background(), "ip4", host)
	} else {
		ips, err = net.LookupIP(host)
	}

	if err != nil {
		return nil, err
	}

	res := []string{}
	for _, ip := range ips {
		res = append(res, ip.String())
	}

	// cache the host resolved result
	if len(res) > 0 {
		caches[cache] = res
	}

	return res, nil
}

// DialContext return a DialContext function for http.Transport, using the local resolver
func DialContext() func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := LookupIP(host, true)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			var dialer net.Dialer
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
			if err == nil {
				return conn, err
			}
		}

		log.Error("DNS resolve fail: %v", ips)
		return nil, fmt.Errorf("DNS resolve fail: %v", ips)
	}
}

// LookupIPAt looks up host using the given resolvers. It returns a slice of that host's IPv4 and IPv6 addresses.
func LookupIPAt(servers []string, host string, ipv6 ...bool) ([]string, error) {
	return nil, nil
}

// DialContextAt return a DialContext function for http.Transport, using the given resolvers
func DialContextAt(servers []string, host string) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return nil
}

// DefaultConfig get the local dns server
func DefaultConfig() (*dns.ClientConfig, error) {
	conf, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil || len(conf.Servers) == 0 {
		log.Error("error making client from default file: %v, using 127.0.0.53:53", err)
		return &dns.ClientConfig{Servers: []string{"127.0.0.53"}, Port: "53", Timeout: 1, Attempts: 2}, nil
	}
	return conf, nil
}

// linuxLookupIP
func linuxLookupIP(host string, servers []string, port string, ipv6 bool, attempts ...int) ([]string, error) {

	if servers == nil || len(servers) == 0 {
		return []string{}, fmt.Errorf("error query servers is nil")
	}

	if attempts == nil {
		attempts = []int{0}
	}

	attempt := attempts[0]
	delta := 2
	if ipv6 == false {
		delta = 1
	}

	c := new(dns.Client)
	ips := []string{}

	t := make(chan *dns.Msg, 2)
	wg := new(sync.WaitGroup)
	wg.Add(delta)
	m4 := new(dns.Msg)
	m4.SetQuestion(dns.Fqdn(host), dns.TypeA)
	go do(t, wg, c, m4, net.JoinHostPort(servers[attempt], port))
	if ipv6 {
		m6 := new(dns.Msg)
		m6.SetQuestion(dns.Fqdn(host), dns.TypeAAAA)
		go do(t, wg, c, m6, net.JoinHostPort(servers[attempt], port))
	}
	wg.Wait()
	close(t)

	for d := range t {
		if d.Rcode == dns.RcodeSuccess {
			for _, a := range d.Answer {
				switch t := a.(type) {
				case *dns.A:
					ips = append(ips, t.A.String())
				case *dns.AAAA:
					if ipv6 {
						ips = append(ips, t.AAAA.String())
					}
				}
			}
		}
	}

	if len(ips) == 0 {
		next := attempt + 1
		if next < len(servers) {
			return linuxLookupIP(host, servers, port, ipv6, next)
		}
	}

	return ips, nil
}

func do(t chan *dns.Msg, wg *sync.WaitGroup, c *dns.Client, m *dns.Msg, addr string) {
	defer wg.Done()
	r, _, err := c.Exchange(m, addr)
	if err != nil {
		log.Error("error Exchange: %s", err.Error())
		return
	}
	t <- r
}
