package llm

import (
	"encoding/json"
	"testing"
)

func TestHasConfidence_Nil(t *testing.T) {
	var a *DecisionAnswer
	if a.HasConfidence() {
		t.Error("nil receiver should return false")
	}
}

func TestHasConfidence_NoulNoConfidence(t *testing.T) {
	a := &DecisionAnswer{Type: "noul", Noul: ptrFloat(0.98)}
	if a.HasConfidence() {
		t.Error("noul answer without Confidence should return false")
	}
}

func TestHasConfidence_ChoiceWithConfidence(t *testing.T) {
	a := &DecisionAnswer{
		Type:       "choice",
		Choice:     "billing",
		Confidence: ptrFloat(0.67),
	}
	if !a.HasConfidence() {
		t.Error("choice answer with Confidence should return true")
	}
}

func TestHasConfidence_Zero(t *testing.T) {
	a := &DecisionAnswer{
		Type:       "score",
		Score:      ptrFloat(1.28),
		Confidence: ptrFloat(0.0),
	}
	if !a.HasConfidence() {
		t.Error("Confidence=0.0 (pointer non-nil) should return true")
	}
}

func TestDecisionAnswer_JSONRoundTrip_Noul(t *testing.T) {
	raw := `{"type":"noul","noul":0.98}`
	var a DecisionAnswer
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if a.Type != "noul" {
		t.Errorf("Type = %q, want noul", a.Type)
	}
	if a.Noul == nil || *a.Noul != 0.98 {
		t.Errorf("Noul = %v, want 0.98", a.Noul)
	}
	if a.Confidence != nil {
		t.Errorf("Confidence should be nil for noul, got %v", *a.Confidence)
	}
}

func TestDecisionAnswer_JSONRoundTrip_Choice(t *testing.T) {
	raw := `{
		"type":"choice",
		"choice":"billing",
		"confidence":0.67,
		"probabilities":{"billing":0.78,"sales":0.0,"technical":0.22}
	}`
	var a DecisionAnswer
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if a.Choice != "billing" {
		t.Errorf("Choice = %q, want billing", a.Choice)
	}
	if a.Confidence == nil || *a.Confidence != 0.67 {
		t.Errorf("Confidence = %v, want 0.67", a.Confidence)
	}
	if len(a.Probabilities) != 3 {
		t.Errorf("Probabilities len = %d, want 3", len(a.Probabilities))
	}
}

func TestDecisionAnswer_JSONRoundTrip_Score(t *testing.T) {
	raw := `{
		"type":"score",
		"score":1.55,
		"confidence":0.32,
		"legend":{"0":"Calm","1":"Frustrated","2":"Very angry"},
		"probabilities":{"0":0.0,"1":0.45,"2":0.55}
	}`
	var a DecisionAnswer
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if a.Score == nil || *a.Score != 1.55 {
		t.Errorf("Score = %v, want 1.55", a.Score)
	}
	if a.Legend["0"] != "Calm" {
		t.Errorf("Legend[0] = %q, want Calm", a.Legend["0"])
	}
}

func TestDecisionResponse_JSONRoundTrip(t *testing.T) {
	raw := `{
		"model":"jev-1.13.0",
		"answers":{
			"is_urgent":{"type":"noul","noul":0.98},
			"department":{"type":"choice","choice":"billing","confidence":0.67,
				"probabilities":{"billing":0.78,"sales":0.0,"technical":0.22}}
		},
		"usage":{"input_tokens":408,"output_tokens":73}
	}`
	var resp DecisionResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Model != "jev-1.13.0" {
		t.Errorf("Model = %q, want jev-1.13.0", resp.Model)
	}
	if len(resp.Answers) != 2 {
		t.Fatalf("Answers len = %d, want 2", len(resp.Answers))
	}

	urgent := resp.Answers["is_urgent"]
	if urgent.HasConfidence() {
		t.Error("is_urgent (noul) should not have confidence")
	}

	dept := resp.Answers["department"]
	if !dept.HasConfidence() {
		t.Error("department (choice) should have confidence")
	}

	if resp.Usage == nil {
		t.Fatal("Usage should not be nil")
	}
	if resp.Usage.InputTokens != 408 {
		t.Errorf("InputTokens = %d, want 408", resp.Usage.InputTokens)
	}
}

func TestDecisionRequest_Marshal(t *testing.T) {
	req := DecisionRequest{
		Model: "jev-latest",
		State: "Customer says billing is wrong",
		Questions: map[string]DecisionQuestion{
			"dept": {
				Type:         "choice",
				Instructions: "Route to department",
				Criteria:     map[string]string{"billing": "Billing issues", "tech": "Technical"},
			},
		},
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal check: %v", err)
	}
	if decoded["model"] != "jev-latest" {
		t.Errorf("model = %v, want jev-latest", decoded["model"])
	}
	if decoded["state"] != "Customer says billing is wrong" {
		t.Errorf("state mismatch")
	}
}

func ptrFloat(f float64) *float64 { return &f }
