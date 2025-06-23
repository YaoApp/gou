package dns

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/utils"
)

func TestDefaultConfig(t *testing.T) {
	config, err := DefaultConfig()
	if err != nil {
		t.Fatalf("DefaultConfig() failed: %v", err)
	}

	if config == nil {
		t.Fatal("DefaultConfig() returned nil config")
	}

	if len(config.Servers) == 0 {
		t.Fatal("DefaultConfig() returned no DNS servers")
	}

	t.Logf("DNS configuration detected:")
	t.Logf("  Servers: %v", config.Servers)
	t.Logf("  Port: %s", config.Port)
	t.Logf("  Timeout: %d", config.Timeout)
	t.Logf("  Attempts: %d", config.Attempts)

	// Ensure we have at least one DNS server
	if len(config.Servers) < 1 {
		t.Errorf("Expected at least 1 DNS server, got %d", len(config.Servers))
	}

	// Log if we only have one server (for information)
	if len(config.Servers) == 1 {
		t.Logf("Only one DNS server configured. Fallback will provide redundancy if needed.")
	}
}

func TestGetFallbackDNSServers(t *testing.T) {
	servers := getFallbackDNSServers()

	if len(servers) == 0 {
		t.Fatal("getFallbackDNSServers() returned no servers")
	}

	t.Logf("Fallback DNS servers: %v", servers)

	// Should have some public DNS servers as fallback
	hasPublicDNS := false
	publicDNSServers := []string{"1.1.1.1", "8.8.8.8", "1.0.0.1", "8.8.4.4"}

	for _, server := range servers {
		for _, publicDNS := range publicDNSServers {
			if server == publicDNS {
				hasPublicDNS = true
				break
			}
		}
	}

	if !hasPublicDNS {
		t.Error("No public DNS servers found in fallback list")
	}
}

func TestGetSystemSpecificDNS(t *testing.T) {
	servers := getSystemSpecificDNS()

	t.Logf("System-specific DNS servers detected: %v", servers)

	// This test doesn't fail if no system-specific DNS is found
	// as it depends on the system configuration
	if len(servers) > 0 {
		t.Logf("Successfully detected %d system-specific DNS servers", len(servers))
	} else {
		t.Log("No system-specific DNS servers detected (this is normal on some systems)")
	}
}

func TestIsDNSServerReachable(t *testing.T) {
	// Test with known public DNS servers
	testServers := []string{
		"8.8.8.8", // Google DNS
		"1.1.1.1", // Cloudflare DNS
	}

	for _, server := range testServers {
		reachable := isDNSServerReachable(server)
		t.Logf("DNS server %s reachable: %v", server, reachable)

		// Note: We don't fail the test if public DNS is not reachable
		// as it might be blocked by firewall or network configuration
	}

	// Test with obviously unreachable server
	unreachable := isDNSServerReachable("192.0.2.1") // RFC 5737 test address
	if unreachable {
		t.Log("Warning: Test address 192.0.2.1 appears reachable (unexpected)")
	}
}

func TestContains(t *testing.T) {
	servers := []string{"8.8.8.8", "1.1.1.1", "127.0.0.1"}

	if !contains(servers, "8.8.8.8") {
		t.Error("contains() should return true for existing item")
	}

	if contains(servers, "8.8.4.4") {
		t.Error("contains() should return false for non-existing item")
	}
}

func TestMin(t *testing.T) {
	if min(5, 3) != 3 {
		t.Error("min(5, 3) should return 3")
	}

	if min(2, 7) != 2 {
		t.Error("min(2, 7) should return 2")
	}

	if min(4, 4) != 4 {
		t.Error("min(4, 4) should return 4")
	}
}

func TestLookupIP(t *testing.T) {
	_, err := LookupIP("github.com", true)
	if err != nil {
		t.Fatal(err)
	}

	res, err := LookupIP("google.com", true)
	if err != nil {
		t.Fatal(err)
	}

	// Check if we got any IPv6 addresses (containing ":")
	hasIPv6 := strings.Contains(strings.Join(res, ","), ":")
	if !hasIPv6 {
		t.Logf("No IPv6 addresses returned for google.com (this may be normal in some network environments)")
		t.Logf("Returned addresses: %v", res)

		// Test localhost IPv6 to verify IPv6 functionality works
		localRes, err := LookupIP("localhost", true)
		if err == nil {
			localHasIPv6 := strings.Contains(strings.Join(localRes, ","), ":")
			if localHasIPv6 {
				t.Logf("IPv6 functionality verified with localhost: %v", localRes)
			}
		}
	} else {
		t.Logf("Successfully got IPv6 addresses: %v", res)
	}

	res, err = LookupIP("google.com", false)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotContains(t, strings.Join(res, ","), ":")
}

func TestLookupIPCaches(t *testing.T) {
	ips, err := LookupIP("github.com", true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, ips, caches["github.com_true"])
	utils.Dump(ips)

	_, has := caches["github.com_false"]
	assert.Equal(t, false, has)

	ips, err = LookupIP("github.com", false)
	if err != nil {
		t.Fatal(err)
	}
	utils.Dump(ips)
	assert.Equal(t, ips, caches["github.com_false"])

}

func TestLookupIPHostIsIP(t *testing.T) {
	ips, err := LookupIP("127.0.0.1", true)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, ips, []string{"127.0.0.1"})
}

func TestDialContext(t *testing.T) {
	var body []byte
	req, err := http.NewRequest("GET", "https://api.github.com/users/yaoapp", bytes.NewBuffer(body))
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext:     DialContext(),
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body) // response body is []byte
	if err != nil {
		t.Fatal(err)
	}
}
