package typesafe

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/yaoapp/gou/llm"
)

func TestEndpoint_Default(t *testing.T) {
	c := &Connector{}
	if e := c.endpoint(); e != "/v1/systemone" {
		t.Errorf("default endpoint = %q, want /v1/systemone", e)
	}
}

func TestEndpoint_Custom(t *testing.T) {
	c := &Connector{Options: Options{Endpoint: "/v1/decisions"}}
	if e := c.endpoint(); e != "/v1/decisions" {
		t.Errorf("custom endpoint = %q, want /v1/decisions", e)
	}
}

func TestGetURL_Default(t *testing.T) {
	c := &Connector{}
	if u := c.GetURL(); u != "https://api.typesafe.ai" {
		t.Errorf("default URL = %q, want https://api.typesafe.ai", u)
	}
}

func TestGetURL_Custom(t *testing.T) {
	c := &Connector{Options: Options{Host: "https://tao.example.com"}}
	if u := c.GetURL(); u != "https://tao.example.com" {
		t.Errorf("custom URL = %q, want https://tao.example.com", u)
	}
}

func TestGetModel_Default(t *testing.T) {
	c := &Connector{}
	if m := c.GetModel(); m != "jev-latest" {
		t.Errorf("default model = %q, want jev-latest", m)
	}
}

func TestGetModel_Custom(t *testing.T) {
	c := &Connector{Options: Options{Model: "jev-1.13.0"}}
	if m := c.GetModel(); m != "jev-1.13.0" {
		t.Errorf("custom model = %q, want jev-1.13.0", m)
	}
}

func TestGetCapabilities_Default(t *testing.T) {
	c := &Connector{}
	caps := c.GetCapabilities()
	if caps == nil {
		t.Fatal("capabilities should not be nil")
	}
	if !caps.Decision {
		t.Error("default capabilities should have Decision=true")
	}
}

func TestGetCapabilities_Custom(t *testing.T) {
	c := &Connector{Options: Options{
		Capabilities: &llm.Capabilities{Decision: true, Streaming: true},
	}}
	caps := c.GetCapabilities()
	if !caps.Decision || !caps.Streaming {
		t.Error("custom capabilities should preserve Decision and Streaming")
	}
}

func TestGetKey(t *testing.T) {
	c := &Connector{Options: Options{Key: "sk-test-123"}}
	if k := c.GetKey(); k != "sk-test-123" {
		t.Errorf("key = %q, want sk-test-123", k)
	}
}

func TestGetAuthMode(t *testing.T) {
	c := &Connector{}
	if m := c.GetAuthMode(); m != llm.AuthBearer {
		t.Errorf("auth mode = %v, want AuthBearer", m)
	}
}

func TestSetting(t *testing.T) {
	c := &Connector{Options: Options{
		Host:     "https://tao.example.com",
		Key:      "sk-xxx",
		Model:    "jev-1.13.0",
		Endpoint: "/v1/decisions",
	}}
	s := c.Setting()
	if s["host"] != "https://tao.example.com" {
		t.Errorf("setting host = %v", s["host"])
	}
	if s["key"] != "sk-xxx" {
		t.Errorf("setting key = %v", s["key"])
	}
	if s["model"] != "jev-1.13.0" {
		t.Errorf("setting model = %v", s["model"])
	}
	if s["endpoint"] != "/v1/decisions" {
		t.Errorf("setting endpoint = %v", s["endpoint"])
	}
}

func TestIs(t *testing.T) {
	c := &Connector{}
	if !c.Is(12) {
		t.Error("Is(12) should be true for TYPESAFE")
	}
	if c.Is(1) {
		t.Error("Is(1) should be false for TYPESAFE")
	}
}

func TestID(t *testing.T) {
	c := &Connector{id: "test-conn"}
	if c.ID() != "test-conn" {
		t.Errorf("ID = %q, want test-conn", c.ID())
	}
}

func TestClose(t *testing.T) {
	c := &Connector{}
	if err := c.Close(); err != nil {
		t.Errorf("Close should return nil, got %v", err)
	}
}

func TestDecide_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/systemone" {
			t.Errorf("path = %s, want /v1/systemone", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("auth = %q, want Bearer test-key", auth)
		}

		body, _ := io.ReadAll(r.Body)
		var req llm.DecisionRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}
		if req.Model != "jev-latest" {
			t.Errorf("request model = %q, want jev-latest", req.Model)
		}

		resp := llm.DecisionResponse{
			Model: "jev-1.13.0",
			Answers: map[string]llm.DecisionAnswer{
				"is_urgent": {Type: "noul", Noul: ptrFloat(0.98)},
			},
			Usage: &llm.DecisionUsage{InputTokens: 100, OutputTokens: 20},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := &Connector{Options: Options{
		Host: server.URL,
		Key:  "test-key",
	}}

	req := &llm.DecisionRequest{
		State: "Customer says billing is wrong",
		Questions: map[string]llm.DecisionQuestion{
			"is_urgent": {Type: "noul", Instructions: "Is this urgent?"},
		},
	}

	resp, err := c.Decide(req)
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if resp.Model != "jev-1.13.0" {
		t.Errorf("response model = %q, want jev-1.13.0", resp.Model)
	}
	a, ok := resp.Answers["is_urgent"]
	if !ok {
		t.Fatal("missing is_urgent answer")
	}
	if a.Noul == nil || *a.Noul != 0.98 {
		t.Errorf("noul = %v, want 0.98", a.Noul)
	}
	if resp.Usage.InputTokens != 100 {
		t.Errorf("input tokens = %d, want 100", resp.Usage.InputTokens)
	}
}

func TestDecide_CustomEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/decisions" {
			t.Errorf("path = %s, want /v1/decisions", r.URL.Path)
		}
		resp := llm.DecisionResponse{
			Model:   "jev-1.13.0",
			Answers: map[string]llm.DecisionAnswer{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := &Connector{Options: Options{
		Host:     server.URL,
		Key:      "test-key",
		Endpoint: "/v1/decisions",
	}}

	resp, err := c.Decide(&llm.DecisionRequest{State: "test"})
	if err != nil {
		t.Fatalf("Decide with custom endpoint: %v", err)
	}
	if resp.Model != "jev-1.13.0" {
		t.Errorf("model = %q", resp.Model)
	}
}

func TestDecide_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid key"}`))
	}))
	defer server.Close()

	c := &Connector{Options: Options{Host: server.URL, Key: "bad-key"}}
	_, err := c.Decide(&llm.DecisionRequest{State: "test"})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestDecide_RealAPI(t *testing.T) {
	key := os.Getenv("TYPESAFE_API_KEY")
	if key == "" {
		t.Skip("TYPESAFE_API_KEY not set, skipping real API test")
	}

	// Pre-flight: verify api.typesafe.ai is reachable from this host
	if _, err := net.DialTimeout("tcp", "api.typesafe.ai:443", 5*time.Second); err != nil {
		t.Skipf("api.typesafe.ai unreachable, skipping real API test: %v", err)
	}

	c := &Connector{Options: Options{
		Host: "https://api.typesafe.ai",
		Key:  key,
	}}

	req := &llm.DecisionRequest{
		Model: "jev-latest",
		State: "Customer says: my billing is wrong, I was charged twice for the same order",
		Questions: map[string]llm.DecisionQuestion{
			"is_urgent": {
				Type:         "noul",
				Instructions: "Is this issue urgent and needs immediate attention?",
			},
			"department": {
				Type:         "choice",
				Instructions: "Route this ticket to the appropriate department",
				Criteria: map[string]string{
					"billing":   "Billing and payment related issues",
					"technical": "Technical product issues",
					"sales":     "Pre-sales and purchasing inquiries",
				},
			},
		},
	}

	resp, err := c.Decide(req)
	if err != nil {
		t.Fatalf("real API Decide: %v", err)
	}
	if resp.Model == "" {
		t.Error("response model should not be empty")
	}
	if len(resp.Answers) != 2 {
		t.Errorf("expected 2 answers, got %d", len(resp.Answers))
	}

	urgent, ok := resp.Answers["is_urgent"]
	if !ok {
		t.Fatal("missing is_urgent answer")
	}
	if urgent.Type != "noul" {
		t.Errorf("is_urgent type = %q, want noul", urgent.Type)
	}
	if urgent.Noul == nil {
		t.Error("is_urgent noul should not be nil")
	}

	dept, ok := resp.Answers["department"]
	if !ok {
		t.Fatal("missing department answer")
	}
	if dept.Type != "choice" {
		t.Errorf("department type = %q, want choice", dept.Type)
	}
	if dept.Choice == "" {
		t.Error("department choice should not be empty")
	}

	if resp.Usage == nil {
		t.Error("usage should not be nil")
	} else {
		t.Logf("Real API: model=%s, urgent_noul=%.2f, dept=%s, tokens=%d+%d",
			resp.Model, *urgent.Noul, dept.Choice,
			resp.Usage.InputTokens, resp.Usage.OutputTokens)
	}
}

func ptrFloat(f float64) *float64 { return &f }
