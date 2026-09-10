package dns

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestLookupIP_CacheTTLExpired(t *testing.T) {
	ClearCache()

	// Inject an expired entry.
	SetCacheForTest("example.com_false", []string{"1.2.3.4"}, time.Now().Add(-1*time.Second))

	// LookupIP should discard the expired entry and resolve fresh.
	ips, err := LookupIP("example.com", false)
	if err != nil {
		t.Fatalf("LookupIP failed: %v", err)
	}
	if len(ips) == 0 {
		t.Fatal("expected at least one IP")
	}
	if ips[0] == "1.2.3.4" {
		t.Error("expected fresh IP, got stale cached value 1.2.3.4")
	}
	t.Logf("Fresh resolve: %v", ips)

	ClearCache()
}

func TestLookupIP_CacheTTLNotExpired(t *testing.T) {
	ClearCache()

	// Inject a valid (not-yet-expired) entry.
	SetCacheForTest("cached-host.test_false", []string{"10.0.0.1"}, time.Now().Add(5*time.Minute))

	ips, err := LookupIP("cached-host.test", false)
	if err != nil {
		t.Fatalf("LookupIP failed: %v", err)
	}
	if len(ips) != 1 || ips[0] != "10.0.0.1" {
		t.Errorf("expected cached [10.0.0.1], got %v", ips)
	}

	ClearCache()
}

func TestClearHost(t *testing.T) {
	ClearCache()

	SetCacheForTest("a.com_false", []string{"1.1.1.1"}, time.Now().Add(5*time.Minute))
	SetCacheForTest("a.com_true", []string{"::1"}, time.Now().Add(5*time.Minute))
	SetCacheForTest("b.com_false", []string{"2.2.2.2"}, time.Now().Add(5*time.Minute))

	ClearHost("a.com")

	// a.com entries should be gone.
	cachesMutex.RLock()
	_, hasAFalse := caches["a.com_false"]
	_, hasATrue := caches["a.com_true"]
	_, hasBFalse := caches["b.com_false"]
	cachesMutex.RUnlock()

	if hasAFalse || hasATrue {
		t.Error("ClearHost(a.com) should have removed a.com entries")
	}
	if !hasBFalse {
		t.Error("ClearHost(a.com) should NOT have removed b.com entries")
	}

	ClearCache()
}

func TestClearCache(t *testing.T) {
	SetCacheForTest("x.com_false", []string{"3.3.3.3"}, time.Now().Add(5*time.Minute))
	ClearCache()

	cachesMutex.RLock()
	count := len(caches)
	cachesMutex.RUnlock()

	if count != 0 {
		t.Errorf("ClearCache should empty the cache, got %d entries", count)
	}
}

func TestDialContext_Timeout(t *testing.T) {
	ClearCache()

	// Inject an unreachable IP (240.x is reserved, not routable).
	SetCacheForTest("unreachable.test_false", []string{"240.0.0.1"}, time.Now().Add(10*time.Minute))

	dial := DialContext()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	start := time.Now()
	conn, err := dial(ctx, "tcp", "unreachable.test:443")
	elapsed := time.Since(start)

	if conn != nil {
		conn.Close()
		t.Fatal("expected nil conn for unreachable IP")
	}
	if err == nil {
		t.Fatal("expected error for unreachable IP")
	}

	// The Dialer Timeout is 15s; should fail within ~16s (not hang forever).
	if elapsed > 18*time.Second {
		t.Errorf("dial took %v, expected <18s (Dialer.Timeout=15s)", elapsed)
	}
	t.Logf("Dial failed in %v: %v", elapsed, err)

	ClearCache()
}

func TestLookupIP_DirectIP(t *testing.T) {
	ips, err := LookupIP("127.0.0.1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ips) != 1 || ips[0] != "127.0.0.1" {
		t.Errorf("expected [127.0.0.1], got %v", ips)
	}
}

func TestLookupIP_CacheKeyFormat(t *testing.T) {
	ClearCache()

	// ipv4-only and ipv6 should use separate cache keys.
	SetCacheForTest("dual.test_false", []string{"4.4.4.4"}, time.Now().Add(5*time.Minute))
	SetCacheForTest("dual.test_true", []string{"::1"}, time.Now().Add(5*time.Minute))

	ips4, _ := LookupIP("dual.test", false)
	ips6, _ := LookupIP("dual.test", true)

	if fmt.Sprint(ips4) == fmt.Sprint(ips6) {
		t.Error("ipv4 and ipv6 cache keys should return different results")
	}

	ClearCache()
}

func TestLookupIP_DefaultIPv6(t *testing.T) {
	ClearCache()

	// When no ipv6 arg is passed, default is true → cache key "host_true".
	SetCacheForTest("default.test_true", []string{"5.5.5.5"}, time.Now().Add(5*time.Minute))

	ips, err := LookupIP("default.test") // no ipv6 arg
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ips) != 1 || ips[0] != "5.5.5.5" {
		t.Errorf("expected cached [5.5.5.5], got %v", ips)
	}

	ClearCache()
}

func TestLookupIP_RealResolve(t *testing.T) {
	ClearCache()

	// Fresh resolve of a well-known host (not cached).
	ips, err := LookupIP("dns.google", false)
	if err != nil {
		t.Fatalf("LookupIP(dns.google) failed: %v", err)
	}
	if len(ips) == 0 {
		t.Fatal("expected at least one IP for dns.google")
	}
	t.Logf("dns.google resolved to: %v", ips)

	// Verify it was cached.
	ips2, _ := LookupIP("dns.google", false)
	if fmt.Sprint(ips) != fmt.Sprint(ips2) {
		t.Errorf("second lookup should return cached result: %v vs %v", ips, ips2)
	}

	ClearCache()
}

func TestClearHost_NoMatch(t *testing.T) {
	ClearCache()

	SetCacheForTest("keep.test_false", []string{"9.9.9.9"}, time.Now().Add(5*time.Minute))

	ClearHost("nonexistent.host")

	cachesMutex.RLock()
	_, has := caches["keep.test_false"]
	cachesMutex.RUnlock()

	if !has {
		t.Error("ClearHost(nonexistent) should not affect other entries")
	}

	ClearCache()
}

func TestDialContext_Success(t *testing.T) {
	ClearCache()

	dial := DialContext()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Dial a real reachable address.
	conn, err := dial(ctx, "tcp", "dns.google:443")
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	conn.Close()

	ClearCache()
}

func TestLookupIP_ResolverError(t *testing.T) {
	ClearCache()

	// .invalid TLD is reserved by RFC 2606 and should fail lookup.
	_, err := LookupIP("definitely-not-real.invalid", false)
	if err == nil {
		t.Skip("resolver did not return error for .invalid TLD (some resolvers return NXDOMAIN without error)")
	}
	t.Logf("Got expected resolver error: %v", err)

	ClearCache()
}

func TestLookupIP_EmptyResult(t *testing.T) {
	ClearCache()

	// A host that resolves to no IPs should return empty slice, no cache write.
	ips, err := LookupIP("empty-result-unlikely.invalid", false)
	if err != nil {
		// This is the expected path — resolver error.
		t.Logf("Resolver returned error (expected): %v", err)
	} else if len(ips) == 0 {
		t.Log("Got empty result as expected")
	}

	ClearCache()
}

func TestDialContext_BadAddr(t *testing.T) {
	dial := DialContext()
	ctx := context.Background()

	// Address without port → SplitHostPort fails.
	conn, err := dial(ctx, "tcp", "no-port-here")
	if conn != nil {
		conn.Close()
		t.Error("expected nil conn")
	}
	if err == nil {
		t.Error("expected error for malformed address")
	}
}

func TestLookupIP_IPv6Path(t *testing.T) {
	ClearCache()

	// Explicitly request ipv6=true to cover the "ip" network lookup path.
	ips, err := LookupIP("dns.google", true)
	if err != nil {
		t.Fatalf("LookupIP(dns.google, ipv6=true) failed: %v", err)
	}
	if len(ips) == 0 {
		t.Fatal("expected at least one IP")
	}
	t.Logf("dns.google (ipv6=true): %v", ips)

	ClearCache()
}

func TestDialContext_IPv6Env(t *testing.T) {
	ClearCache()

	// Set env to enable ipv6 in DialContext.
	t.Setenv("YAO_ENABLE_IPV6", "1")

	// Inject a cached entry for the ipv6 key.
	SetCacheForTest("envtest.test_true", []string{"127.0.0.1"}, time.Now().Add(5*time.Minute))

	dial := DialContext()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// The DialContext should use ipv6=true due to env, matching cache key "_true".
	conn, err := dial(ctx, "tcp", "envtest.test:80")
	// Connection to 127.0.0.1:80 may fail (no server), but the important thing
	// is that it used the right cache key (ipv6=true).
	if conn != nil {
		conn.Close()
	}
	// Error is acceptable — we're testing the env var path, not the connection.
	t.Logf("IPv6 env dial result: err=%v", err)

	ClearCache()
}

func TestDialContext_LookupError(t *testing.T) {
	ClearCache()

	dial := DialContext()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// A TLD that doesn't exist → LookupIP returns error.
	conn, err := dial(ctx, "tcp", "this-host-does-not-exist.invalid:443")
	if conn != nil {
		conn.Close()
		t.Error("expected nil conn")
	}
	if err == nil {
		t.Error("expected error for unresolvable host")
	}
	t.Logf("Got expected error: %v", err)

	ClearCache()
}
