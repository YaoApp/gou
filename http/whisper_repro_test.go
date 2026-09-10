package http

import (
	"context"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/yaoapp/gou/dns"
)

// TestWhisperReproduction reproduces the exact code path used by
// openai.AudioTranscriptionsFile to diagnose "OpenAI Error".
//
// Run:  OPENAI_KEY=sk-proj-... go test -v -run TestWhisperReproduction ./http/
func TestWhisperReproduction(t *testing.T) {
	key := os.Getenv("OPENAI_KEY")
	if key == "" {
		t.Skip("OPENAI_KEY not set")
	}

	// --- 1. Check DNS resolution (same path as gou/dns) ---
	ips, err := dns.LookupIP("api.openai.com", false)
	t.Logf("dns.LookupIP(api.openai.com, ipv6=false): ips=%v err=%v", ips, err)
	if err != nil {
		t.Fatalf("DNS lookup failed: %v", err)
	}

	// --- 2. Generate a minimal valid WAV (1s 440Hz tone, 16kHz mono 16-bit) ---
	wavPath := t.TempDir() + "/tone.wav"
	writeTestWAV(t, wavPath)
	info, _ := os.Stat(wavPath)
	t.Logf("WAV file: %s  size=%d", wavPath, info.Size())

	// --- 3. Build the request exactly like AudioTranscriptionsFile ---
	url := "https://api.openai.com/v1/audio/transcriptions"
	req := New(url)
	req.WithHeader(map[string][]string{
		"Content-Type":  {"multipart/form-data"},
		"Authorization": {fmt.Sprintf("Bearer %s", key)},
	})
	req.AddFile("file", wavPath)

	option := map[string]interface{}{
		"model": "whisper-1",
	}

	// --- 4. Send (same as openai.go L286) ---
	res := req.Send("POST", option)

	// --- 5. Dump ALL response fields (the info isError() discards) ---
	t.Logf("res.Status  = %d", res.Status)
	t.Logf("res.Code    = %d", res.Code)
	t.Logf("res.Message = %q", res.Message)
	t.Logf("res.Data    = (%T) %v", res.Data, truncate(res.Data, 500))

	if res.Status != 200 {
		t.Errorf("NON-200: Status=%d  Message=%q  DataType=%T", res.Status, res.Message, res.Data)
	} else {
		t.Logf("SUCCESS: %v", res.Data)
	}
}

func TestDNSCacheStale(t *testing.T) {
	// Call LookupIP twice to verify the cache returns the same result.
	ips1, err1 := dns.LookupIP("api.openai.com", false)
	t.Logf("1st lookup: ips=%v err=%v", ips1, err1)

	ips2, err2 := dns.LookupIP("api.openai.com", false)
	t.Logf("2nd lookup (cached): ips=%v err=%v", ips2, err2)

	if err1 != nil {
		t.Fatalf("DNS lookup failed: %v", err1)
	}
	if fmt.Sprint(ips1) != fmt.Sprint(ips2) {
		t.Errorf("cache mismatch: %v vs %v", ips1, ips2)
	}
}

// TestWhisperWithBadIP simulates what happens when gou/dns has cached a
// stale/unreachable IP.  We poison the cache and replay the exact same
// request to see the REAL error that isError() swallows as "OpenAI Error".
func TestWhisperWithBadIP(t *testing.T) {
	key := os.Getenv("OPENAI_KEY")
	if key == "" {
		t.Skip("OPENAI_KEY not set")
	}

	// Clear transport pool so a new dial is forced.
	CloseAllTransports()

	// Poison the DNS cache with an unreachable IP (240.x is reserved, truly unreachable).
	dns.SetCacheForTest("api.openai.com_false", []string{"240.0.0.1"}, time.Now().Add(10*time.Minute))

	ips, _ := dns.LookupIP("api.openai.com", false)
	t.Logf("After poisoning, LookupIP returns: %v", ips)

	wavPath := t.TempDir() + "/tone.wav"
	writeTestWAV(t, wavPath)

	url := "https://api.openai.com/v1/audio/transcriptions"
	req := New(url)

	// Set a 5s context to avoid hanging forever — in production, there is NO
	// context at all, so a bad DNS entry causes the request to hang until the
	// caller (gRPC sandbox) times out.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req.WithContext(ctx)

	req.WithHeader(map[string][]string{
		"Content-Type":  {"multipart/form-data"},
		"Authorization": {fmt.Sprintf("Bearer %s", key)},
	})
	req.AddFile("file", wavPath)

	start := time.Now()
	res := req.Send("POST", map[string]interface{}{"model": "whisper-1"})
	elapsed := time.Since(start)

	t.Logf("Request took: %v", elapsed)
	t.Logf("res.Status  = %d", res.Status)
	t.Logf("res.Code    = %d", res.Code)
	t.Logf("res.Message = %q", res.Message)
	t.Logf("res.Data    = (%T) %v", res.Data, truncate(res.Data, 500))

	// Simulate isError() behavior — this is how the real code loses the error:
	message := "OpenAI Error"
	if v, ok := res.Data.(string); ok {
		message = v
	}
	if data, ok := res.Data.(map[string]interface{}); ok {
		if errObj, has := data["error"]; has {
			if errMap, ok := errObj.(map[string]interface{}); ok {
				if msg, has := errMap["message"].(string); has {
					message = msg
				}
			}
		}
	}
	t.Logf("isError() would return: %q", message)
	t.Logf("REAL error (in res.Message): %q ← this is what isError() DISCARDS", res.Message)

	if res.Status == 0 && res.Message != "" {
		t.Logf("✅ REPRODUCED: DNS cache stale → connection fails → isError() hides real error")
	}

	dns.ClearCache()
}

func writeTestWAV(t *testing.T, path string) {
	t.Helper()
	const (
		sampleRate = 16000
		duration   = 1
		freq       = 440.0
		numSamples = sampleRate * duration
	)
	samples := make([]byte, numSamples*2)
	for i := 0; i < numSamples; i++ {
		v := int16(16000 * math.Sin(2*math.Pi*freq*float64(i)/sampleRate))
		samples[i*2] = byte(v)
		samples[i*2+1] = byte(v >> 8)
	}

	dataSize := uint32(len(samples))
	fmtSize := uint32(16)
	fileSize := 4 + (8 + fmtSize) + (8 + dataSize)

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	w := func(b []byte) { f.Write(b) }
	u32 := func(v uint32) { w([]byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)}) }
	u16 := func(v uint16) { w([]byte{byte(v), byte(v >> 8)}) }

	w([]byte("RIFF"))
	u32(fileSize)
	w([]byte("WAVE"))
	w([]byte("fmt "))
	u32(fmtSize)
	u16(1)     // PCM
	u16(1)     // mono
	u32(16000) // sample rate
	u32(32000) // byte rate
	u16(2)     // block align
	u16(16)    // bits per sample
	w([]byte("data"))
	u32(dataSize)
	w(samples)
}

func truncate(v interface{}, maxLen int) interface{} {
	switch val := v.(type) {
	case []byte:
		if len(val) > maxLen {
			return string(val[:maxLen]) + "..."
		}
		return string(val)
	case string:
		if len(val) > maxLen {
			return val[:maxLen] + "..."
		}
		return val
	default:
		s := fmt.Sprintf("%v", v)
		if len(s) > maxLen {
			return s[:maxLen] + "..."
		}
		return s
	}
}
