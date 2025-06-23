package dns

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/yaoapp/kun/log"
)

// DNS cache with thread-safe access
var caches = map[string][]string{}
var cachesMutex sync.RWMutex

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
	cachesMutex.RLock()
	if ips, has := caches[cache]; has {
		cachesMutex.RUnlock()
		return ips, nil
	}
	cachesMutex.RUnlock()

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
			cachesMutex.Lock()
			caches[cache] = res
			cachesMutex.Unlock()
		}

		return res, nil
	}

	var ips = []net.IP{}
	var err error

	if !ipv6[0] {
		ips, err = net.DefaultResolver.LookupIP(context.Background(), "ip4", host)
	} else {
		// Try to get both IPv4 and IPv6, but prioritize IPv6 if available
		ips, err = net.DefaultResolver.LookupIP(context.Background(), "ip", host)
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
		cachesMutex.Lock()
		caches[cache] = res
		cachesMutex.Unlock()
	}

	return res, nil
}

// DialContext return a DialContext function for http.Transport, using the local resolver
func DialContext() func(ctx context.Context, network, addr string) (net.Conn, error) {

	// Read the ipv6 support from env YAO_ENABLE_IPV6 (default false)
	ipv6 := false
	if v := os.Getenv("YAO_ENABLE_IPV6"); v != "" {
		ipv6 = true
	}

	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		ips, err := LookupIP(host, ipv6)
		if err != nil {
			return nil, err
		}

		for _, ip := range ips {
			var dialer net.Dialer
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
			if err == nil {
				return conn, nil
			}
		}

		log.Error("DNS resolve fail: %v %s", ips, addr)
		return nil, fmt.Errorf("DNS resolve fail: %v %s", ips, addr)
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

// DefaultConfig get the local dns server with better Linux distribution compatibility
func DefaultConfig() (*dns.ClientConfig, error) {
	// Try to read system DNS configuration
	conf, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err == nil && len(conf.Servers) > 0 {
		// Successfully read DNS servers from resolv.conf
		return conf, nil
	}

	log.Error("error reading /etc/resolv.conf: %v, trying fallback DNS servers", err)

	// Fallback DNS servers for different Linux distributions and configurations
	fallbackServers := getFallbackDNSServers()

	return &dns.ClientConfig{
		Servers:  fallbackServers,
		Port:     "53",
		Timeout:  2, // Increased timeout for better reliability
		Attempts: 3, // Increased attempts for better reliability
	}, nil
}

// getFallbackDNSServers returns a list of fallback DNS servers for different Linux systems
func getFallbackDNSServers() []string {
	servers := []string{}

	// Try to detect and use system-specific DNS configurations
	systemServers := getSystemSpecificDNS()
	servers = append(servers, systemServers...)

	// Test local DNS servers (avoid duplicates)
	localDNSCandidates := []string{
		"127.0.0.1",  // localhost (common for dnsmasq, unbound, pihole)
		"127.0.0.53", // systemd-resolved (Ubuntu 18.04+, some systemd distros)
	}

	addedLocal := false
	for _, dns := range localDNSCandidates {
		if isDNSServerReachable(dns) && !contains(servers, dns) {
			servers = append(servers, dns)
			addedLocal = true
			break // Only add one working local DNS to avoid duplicates
		}
	}

	// If no local DNS found, try systemd-resolved anyway (might work)
	if !addedLocal && !contains(servers, "127.0.0.53") {
		if _, err := os.Stat("/run/systemd/resolve/resolv.conf"); err == nil {
			servers = append(servers, "127.0.0.53")
		}
	}

	// Public DNS servers as final fallback (prioritized for reliability)
	publicDNS := []string{
		"1.1.1.1",        // Cloudflare DNS Primary (fastest)
		"8.8.8.8",        // Google DNS Primary
		"1.0.0.1",        // Cloudflare DNS Secondary
		"8.8.4.4",        // Google DNS Secondary
		"208.67.222.222", // OpenDNS Primary
		"9.9.9.9",        // Quad9 DNS
	}

	servers = append(servers, publicDNS...)

	log.Info("Using fallback DNS servers: %v", servers[:min(len(servers), 4)]) // Log only first 4 for brevity
	return servers
}

// getSystemSpecificDNS detects system-specific DNS configurations
func getSystemSpecificDNS() []string {
	servers := []string{}

	// Check for systemd-resolved configurations (Ubuntu 18.04+, Debian 10+, Fedora, Arch, etc.)
	systemdPaths := []string{
		"/run/systemd/resolve/resolv.conf",
		"/run/systemd/resolve/stub-resolv.conf",
		"/usr/lib/systemd/resolv.conf",
	}

	for _, path := range systemdPaths {
		if _, err := os.Stat(path); err == nil {
			if conf, err := dns.ClientConfigFromFile(path); err == nil && len(conf.Servers) > 0 {
				servers = append(servers, conf.Servers...)
				log.Info("Found systemd-resolved DNS servers from %s: %v", path, conf.Servers)
				break
			}
		}
	}

	// Check for NetworkManager DNS configuration (RHEL, CentOS, Fedora)
	nmPaths := []string{
		"/var/run/NetworkManager/resolv.conf",
		"/etc/NetworkManager/resolv.conf",
	}

	for _, path := range nmPaths {
		if _, err := os.Stat(path); err == nil {
			if conf, err := dns.ClientConfigFromFile(path); err == nil && len(conf.Servers) > 0 {
				for _, server := range conf.Servers {
					if !contains(servers, server) {
						servers = append(servers, server)
					}
				}
				log.Info("Found NetworkManager DNS servers from %s: %v", path, conf.Servers)
				break
			}
		}
	}

	// Check for dhcpcd configuration (Arch Linux, some embedded systems)
	if _, err := os.Stat("/etc/dhcpcd.conf"); err == nil {
		if isDNSServerReachable("127.0.0.1") && !contains(servers, "127.0.0.1") {
			servers = append(servers, "127.0.0.1")
			log.Info("Found dhcpcd DNS configuration using localhost")
		}
	}

	// Check for dnsmasq configuration (OpenWrt, some custom setups)
	if _, err := os.Stat("/etc/dnsmasq.conf"); err == nil {
		if isDNSServerReachable("127.0.0.1") && !contains(servers, "127.0.0.1") {
			servers = append(servers, "127.0.0.1")
			log.Info("Found dnsmasq DNS configuration using localhost")
		}
	}

	// Check for resolvconf (older Debian/Ubuntu systems)
	if _, err := os.Stat("/etc/resolvconf/run/resolv.conf"); err == nil {
		if conf, err := dns.ClientConfigFromFile("/etc/resolvconf/run/resolv.conf"); err == nil && len(conf.Servers) > 0 {
			for _, server := range conf.Servers {
				if !contains(servers, server) {
					servers = append(servers, server)
				}
			}
			log.Info("Found resolvconf DNS servers: %v", conf.Servers)
		}
	}

	return servers
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// isDNSServerReachable checks if a DNS server is reachable with a simple test
func isDNSServerReachable(server string) bool {
	// Try to establish a connection to the DNS server
	conn, err := net.DialTimeout("udp", net.JoinHostPort(server, "53"), 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// linuxLookupIP
func linuxLookupIP(host string, servers []string, port string, ipv6 bool, attempts ...int) ([]string, error) {

	if len(servers) == 0 {
		return []string{}, fmt.Errorf("error query servers is empty")
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
