package openai

import "testing"

func TestGetDefaultCapabilities_OCRModels(t *testing.T) {
	tests := []struct {
		model   string
		wantOCR bool
		wantVis bool
	}{
		{"qwen3.5-ocr", true, true},
		{"Qwen3.5-OCR", true, true},
		{"qwen-vl-ocr", true, true},
		{"qwen-vl-ocr-latest", true, true},
	}
	for _, tt := range tests {
		caps := GetDefaultCapabilities(tt.model)
		if caps == nil {
			t.Errorf("GetDefaultCapabilities(%q) = nil, want match", tt.model)
			continue
		}
		if caps.OCR != tt.wantOCR {
			t.Errorf("GetDefaultCapabilities(%q).OCR = %v, want %v", tt.model, caps.OCR, tt.wantOCR)
		}
		if caps.HasVision() != tt.wantVis {
			t.Errorf("GetDefaultCapabilities(%q).HasVision() = %v, want %v", tt.model, caps.HasVision(), tt.wantVis)
		}
	}
}

func TestGetDefaultCapabilities_NonOCRModels(t *testing.T) {
	nonOCR := []string{"gpt-5", "deepseek-chat", "qwen3-max", "claude-4.5-sonnet"}
	for _, model := range nonOCR {
		caps := GetDefaultCapabilities(model)
		if caps == nil {
			continue
		}
		if caps.OCR {
			t.Errorf("GetDefaultCapabilities(%q).OCR = true, non-OCR model should be false", model)
		}
	}
}

func TestGetDefaultCapabilities_NoMatch(t *testing.T) {
	caps := GetDefaultCapabilities("totally-unknown-model-xyz")
	if caps != nil {
		t.Error("unknown model should return nil")
	}
}

func TestGetDefaultCapabilities_Empty(t *testing.T) {
	caps := GetDefaultCapabilities("")
	if caps != nil {
		t.Error("empty model should return nil")
	}
}
