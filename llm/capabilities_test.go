package llm

import "testing"

func TestHasVision_Nil(t *testing.T) {
	var c *Capabilities
	if c.HasVision() {
		t.Error("nil receiver should return false")
	}
	c = &Capabilities{}
	if c.HasVision() {
		t.Error("nil Vision field should return false")
	}
}

func TestHasVision_Bool(t *testing.T) {
	c := &Capabilities{Vision: true}
	if !c.HasVision() {
		t.Error("Vision=true should return true")
	}
	c.Vision = false
	if c.HasVision() {
		t.Error("Vision=false should return false")
	}
}

func TestHasVision_String(t *testing.T) {
	for _, format := range []string{"openai", "claude", "base64", "default"} {
		c := &Capabilities{Vision: format}
		if !c.HasVision() {
			t.Errorf("Vision=%q should return true", format)
		}
	}
	c := &Capabilities{Vision: ""}
	if c.HasVision() {
		t.Error("Vision=\"\" should return false")
	}
}

func TestHasReasoning(t *testing.T) {
	var c *Capabilities
	if c.HasReasoning() {
		t.Error("nil receiver should return false")
	}
	c = &Capabilities{Reasoning: false}
	if c.HasReasoning() {
		t.Error("Reasoning=false should return false")
	}
	c.Reasoning = true
	if !c.HasReasoning() {
		t.Error("Reasoning=true should return true")
	}
}

func TestHasToolCalls(t *testing.T) {
	var c *Capabilities
	if c.HasToolCalls() {
		t.Error("nil receiver should return false")
	}
	c = &Capabilities{ToolCalls: true}
	if !c.HasToolCalls() {
		t.Error("ToolCalls=true should return true")
	}
}
