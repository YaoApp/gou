package anthropic

import "strings"

// DefaultModelCapabilities defines default capabilities for Anthropic models
var DefaultModelCapabilities = map[string]Capabilities{
	// Claude 4.x models
	"claude-opus-4": {
		Vision:     "claude",
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  true,
		Streaming:  true,
		JSON:       true,
		Multimodal: true,
	},
	"claude-sonnet-4": {
		Vision:     "claude",
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: true,
	},
	"claude-haiku-4": {
		Vision:     "claude",
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: true,
	},

	// Claude 3.5 models
	"claude-3-5-sonnet": {
		Vision:     "claude",
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: true,
	},
	"claude-3-5-haiku": {
		Vision:     "claude",
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: true,
	},

	// Kimi K2 Coding models (Anthropic-compatible endpoint)
	"kimi-k2": {
		Vision:     false,
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  true,
		Streaming:  true,
		JSON:       true,
		Multimodal: false,
	},
	"kimi-k2.5": {
		Vision:     "claude",
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  true,
		Streaming:  true,
		JSON:       true,
		Multimodal: true,
	},
	"kimi-for-coding": {
		Vision:     false,
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  true,
		Streaming:  true,
		JSON:       true,
		Multimodal: false,
	},
}

// GetDefaultCapabilities returns default capabilities based on model name
// Returns nil if no matching pattern is found
func GetDefaultCapabilities(model string) *Capabilities {
	if model == "" {
		return nil
	}

	modelLower := strings.ToLower(model)

	for pattern, caps := range DefaultModelCapabilities {
		if strings.Contains(modelLower, pattern) {
			capsCopy := caps
			return &capsCopy
		}
	}

	return nil
}
