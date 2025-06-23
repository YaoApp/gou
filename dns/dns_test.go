package dns

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"
	"testing"
	"time"

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

// =============================================================================
// Goroutine and Memory Leak Detection Tests
// =============================================================================

func TestGoroutineLeakDetection(t *testing.T) {
	initialGoroutines := runtime.NumGoroutine()

	// Test multiple DNS lookups to check for goroutine leaks
	hosts := []string{"google.com", "github.com", "cloudflare.com"}

	for i := 0; i < 10; i++ {
		for _, host := range hosts {
			_, err := LookupIP(host, true)
			if err != nil {
				t.Logf("DNS lookup failed for %s: %v (this may be normal in some network environments)", host, err)
			}
		}
	}

	// Give goroutines time to clean up
	time.Sleep(200 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	goroutineIncrease := finalGoroutines - initialGoroutines

	// Allow some tolerance for system goroutines
	tolerance := 3
	if goroutineIncrease > tolerance {
		buf := make([]byte, 16384)
		runtime.Stack(buf, true)
		t.Errorf("Potential goroutine leak detected: started with %d, ended with %d goroutines (+%d)\nStack trace:\n%s",
			initialGoroutines, finalGoroutines, goroutineIncrease, string(buf))
	} else {
		t.Logf("Goroutine count: %d -> %d (+%d), within tolerance (%d)",
			initialGoroutines, finalGoroutines, goroutineIncrease, tolerance)
	}
}

func TestMemoryLeakDetection(t *testing.T) {
	// Force garbage collection before starting
	runtime.GC()
	runtime.GC()

	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Test operations that might leak memory
	hosts := []string{"google.com", "github.com", "cloudflare.com", "stackoverflow.com", "reddit.com"}

	// Perform many DNS lookups to stress test the cache
	for i := 0; i < 100; i++ {
		for _, host := range hosts {
			// Test both IPv4 and IPv6 lookups
			_, err := LookupIP(host, false)
			if err != nil {
				t.Logf("IPv4 lookup failed for %s: %v", host, err)
			}

			_, err = LookupIP(host, true)
			if err != nil {
				t.Logf("IPv6 lookup failed for %s: %v", host, err)
			}
		}

		// Force garbage collection periodically
		if i%20 == 0 {
			runtime.GC()
		}
	}

	// Force final garbage collection
	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Check memory growth
	var memGrowth, heapGrowth int64
	if m2.Alloc >= m1.Alloc {
		memGrowth = int64(m2.Alloc - m1.Alloc)
	} else {
		memGrowth = -int64(m1.Alloc - m2.Alloc)
	}
	if m2.HeapAlloc >= m1.HeapAlloc {
		heapGrowth = int64(m2.HeapAlloc - m1.HeapAlloc)
	} else {
		heapGrowth = -int64(m1.HeapAlloc - m2.HeapAlloc)
	}

	t.Logf("Memory stats for DNS operations:")
	t.Logf("  Alloc growth: %d bytes (%.2f MB)", memGrowth, float64(memGrowth)/(1024*1024))
	t.Logf("  Heap growth: %d bytes (%.2f MB)", heapGrowth, float64(heapGrowth)/(1024*1024))
	t.Logf("  Sys growth: %d bytes", int64(m2.Sys)-int64(m1.Sys))
	t.Logf("  NumGC: %d", m2.NumGC-m1.NumGC)
	t.Logf("  Cache size: %d entries", len(caches))

	// Check cache growth - this is the main concern for memory leaks
	if len(caches) > len(hosts)*2+10 { // hosts * 2 (IPv4 + IPv6) + some tolerance
		t.Errorf("DNS cache grew excessively: %d entries (expected ~%d)", len(caches), len(hosts)*2)
	}

	// Allow reasonable memory growth for DNS operations and caching
	maxAllowedGrowth := int64(1024 * 1024) // 1MB threshold
	if memGrowth > maxAllowedGrowth {
		t.Errorf("Possible memory leak detected: alloc grew by %d bytes (%.2f MB), threshold: %d bytes (%.2f MB)",
			memGrowth, float64(memGrowth)/(1024*1024), maxAllowedGrowth, float64(maxAllowedGrowth)/(1024*1024))
	}
}

func TestCacheGrowthControl(t *testing.T) {
	// Clear existing cache
	cachesMutex.Lock()
	initialCacheSize := len(caches)
	cachesMutex.Unlock()

	// Test that cache doesn't grow indefinitely
	uniqueHosts := make([]string, 50)
	for i := 0; i < 50; i++ {
		uniqueHosts[i] = fmt.Sprintf("test%d.example.com", i)
	}

	for _, host := range uniqueHosts {
		// These will likely fail to resolve, but should still be cached
		LookupIP(host, false)
		LookupIP(host, true)
	}

	cachesMutex.RLock()
	finalCacheSize := len(caches)
	cachesMutex.RUnlock()

	cacheGrowth := finalCacheSize - initialCacheSize
	t.Logf("Cache growth: %d -> %d (+%d entries)", initialCacheSize, finalCacheSize, cacheGrowth)

	// The cache should grow, but not excessively
	if cacheGrowth > 100 { // Allow some growth but not unlimited
		t.Errorf("Cache grew too much: +%d entries (threshold: 100)", cacheGrowth)
	}
}

func TestConcurrentDNSLookups(t *testing.T) {
	// Test concurrent access to check for race conditions and goroutine leaks
	initialGoroutines := runtime.NumGoroutine()

	hosts := []string{"google.com", "github.com", "stackoverflow.com"}
	numWorkers := 10
	lookupsPerWorker := 5

	done := make(chan bool, numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer func() { done <- true }()

			for j := 0; j < lookupsPerWorker; j++ {
				host := hosts[j%len(hosts)]
				_, err := LookupIP(host, j%2 == 0) // Alternate between IPv4 and IPv6
				if err != nil {
					t.Logf("Worker %d lookup %d failed for %s: %v", workerID, j, host, err)
				}
			}
		}(i)
	}

	// Wait for all workers to complete
	for i := 0; i < numWorkers; i++ {
		<-done
	}

	// Give time for cleanup
	time.Sleep(200 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	goroutineIncrease := finalGoroutines - initialGoroutines

	if goroutineIncrease > 5 { // Allow some tolerance for concurrent operations
		t.Errorf("Potential goroutine leak in concurrent test: started with %d, ended with %d goroutines (+%d)",
			initialGoroutines, finalGoroutines, goroutineIncrease)
	} else {
		t.Logf("Concurrent test passed: goroutine count %d -> %d (+%d)",
			initialGoroutines, finalGoroutines, goroutineIncrease)
	}
}

func TestLinuxLookupIPGoroutineCleanup(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}

	initialGoroutines := runtime.NumGoroutine()

	// Test the Linux-specific DNS lookup function
	config, err := DefaultConfig()
	if err != nil {
		t.Fatalf("Failed to get DNS config: %v", err)
	}

	// Perform multiple lookups to test goroutine cleanup
	for i := 0; i < 20; i++ {
		_, err := linuxLookupIP("google.com", config.Servers, config.Port, true)
		if err != nil {
			t.Logf("Linux DNS lookup failed: %v (this may be normal)", err)
		}
	}

	// Give time for cleanup
	time.Sleep(300 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	goroutineIncrease := finalGoroutines - initialGoroutines

	if goroutineIncrease > 3 {
		buf := make([]byte, 16384)
		runtime.Stack(buf, true)
		t.Errorf("Potential goroutine leak in linuxLookupIP: started with %d, ended with %d goroutines (+%d)\nStack trace:\n%s",
			initialGoroutines, finalGoroutines, goroutineIncrease, string(buf))
	} else {
		t.Logf("Linux DNS lookup test passed: goroutine count %d -> %d (+%d)",
			initialGoroutines, finalGoroutines, goroutineIncrease)
	}
}
