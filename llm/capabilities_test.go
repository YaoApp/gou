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

func TestHasImageEditing_Nil(t *testing.T) {
	var c *Capabilities
	if c.HasImageEditing() {
		t.Error("nil receiver should return false")
	}
	c = &Capabilities{}
	if c.HasImageEditing() {
		t.Error("nil ImageEditing field should return false")
	}
}

func TestHasImageEditing_Bool(t *testing.T) {
	c := &Capabilities{ImageEditing: true}
	if !c.HasImageEditing() {
		t.Error("ImageEditing=true should return true")
	}
	c.ImageEditing = false
	if c.HasImageEditing() {
		t.Error("ImageEditing=false should return false")
	}
}

func TestHasImageEditing_String(t *testing.T) {
	for _, proto := range []string{"multipart", "json"} {
		c := &Capabilities{ImageEditing: proto}
		if !c.HasImageEditing() {
			t.Errorf("ImageEditing=%q should return true", proto)
		}
	}
	c := &Capabilities{ImageEditing: ""}
	if c.HasImageEditing() {
		t.Error("ImageEditing=\"\" should return false")
	}
}

func TestHasOCR(t *testing.T) {
	var c *Capabilities
	if c.HasOCR() {
		t.Error("nil receiver should return false")
	}
	c = &Capabilities{OCR: false}
	if c.HasOCR() {
		t.Error("OCR=false should return false")
	}
	c.OCR = true
	if !c.HasOCR() {
		t.Error("OCR=true should return true")
	}
}

func TestToMap_Nil(t *testing.T) {
	var c *Capabilities
	if m := c.ToMap(); m != nil {
		t.Error("nil receiver should return nil")
	}
}

func TestToMap(t *testing.T) {
	c := &Capabilities{
		Vision:          "openai",
		OCR:             true,
		Streaming:       true,
		ImageGeneration: true,
		Embedding:       false,
	}
	m := c.ToMap()
	if m == nil {
		t.Fatal("ToMap should not return nil for non-nil receiver")
	}

	if v, ok := m["ocr"].(bool); !ok || !v {
		t.Error("ToMap must include ocr=true")
	}
	if v, ok := m["vision"]; !ok || v != "openai" {
		t.Error("ToMap must include vision=\"openai\"")
	}
	if v, ok := m["streaming"].(bool); !ok || !v {
		t.Error("ToMap must include streaming=true")
	}
	if v, ok := m["image_generation"].(bool); !ok || !v {
		t.Error("ToMap must include image_generation=true")
	}
	if v, ok := m["embedding"].(bool); !ok || v {
		t.Error("ToMap must include embedding=false")
	}

	// Zero-value Capabilities: all bools should be present as false
	zero := &Capabilities{}
	zm := zero.ToMap()
	if v, ok := zm["ocr"].(bool); !ok || v {
		t.Error("zero Capabilities: ocr should be false")
	}
}

func TestGetImageEditingFormat(t *testing.T) {
	var c *Capabilities
	if f := c.GetImageEditingFormat(); f != "" {
		t.Errorf("nil receiver should return empty, got %q", f)
	}

	c = &Capabilities{ImageEditing: "multipart"}
	if f := c.GetImageEditingFormat(); f != "multipart" {
		t.Errorf("expected \"multipart\", got %q", f)
	}

	c.ImageEditing = "json"
	if f := c.GetImageEditingFormat(); f != "json" {
		t.Errorf("expected \"json\", got %q", f)
	}

	c.ImageEditing = true
	if f := c.GetImageEditingFormat(); f != "" {
		t.Errorf("bool true should return empty, got %q", f)
	}
}
