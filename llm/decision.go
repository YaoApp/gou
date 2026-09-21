package llm

// DecisionQuestion defines a single typed question in a decision request.
// Criteria shape is rigid: choice requires map[string]string (option→description),
// score requires []string (level labels). Mismatched types cause 422 from the API.
type DecisionQuestion struct {
	Type         string      `json:"type"`               // "choice" | "score" | "noul"
	Instructions string      `json:"instructions"`       // natural-language instructions for the model
	Criteria     interface{} `json:"criteria,omitempty"` // choice: map[string]string, score: []string
}

// DecisionRequest is the payload sent to the decision endpoint (POST /v1/systemone or proxy equivalent).
// State is required (empty string is valid, missing field is not).
type DecisionRequest struct {
	Model     string                      `json:"model"`
	State     interface{}                 `json:"state"` // string, object, or array — required
	Questions map[string]DecisionQuestion `json:"questions"`
}

// DecisionAnswer is one answer returned by the decision model.
//
// Field presence varies by answer type:
//   - choice: Choice + Probabilities + Confidence (calibrated)
//   - score:  Score + Legend + Probabilities + Confidence (calibrated)
//   - noul:   Noul only — Confidence is nil (not returned by the API)
//
// Confidence uses *float64 so nil distinguishes "field absent" (noul)
// from confidence=0 (a valid low-confidence choice/score response).
type DecisionAnswer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice,omitempty"`
	Score         *float64           `json:"score,omitempty"` // expected level, may be fractional (e.g. 1.28)
	Noul          *float64           `json:"noul,omitempty"`  // 0–1 probability
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	Confidence    *float64           `json:"confidence,omitempty"` // nil for noul; calibrated value for choice/score
	Legend        map[string]string  `json:"legend,omitempty"`     // score level descriptions (e.g. {"0":"Calm","1":"Frustrated"})
}

// HasConfidence reports whether this answer includes a calibrated confidence value.
// Returns false for noul answers (which never carry confidence).
func (a *DecisionAnswer) HasConfidence() bool {
	return a != nil && a.Confidence != nil
}

// DecisionResponse is the response from the decision endpoint.
type DecisionResponse struct {
	Model   string                    `json:"model"`
	Answers map[string]DecisionAnswer `json:"answers"`
	Usage   *DecisionUsage            `json:"usage,omitempty"`
}

// DecisionUsage reports token consumption for a decision request.
// Only input tokens are billed; output tokens are reported but free on TypeSafe.
type DecisionUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// DecisionConnector extends LLMConnector for TypeSafe-style decision models.
// Use type assertion: if dc, ok := conn.(llm.DecisionConnector); ok { ... }
type DecisionConnector interface {
	LLMConnector
	// Decide sends a typed decision request and returns structured answers.
	Decide(req *DecisionRequest) (*DecisionResponse, error)
}
