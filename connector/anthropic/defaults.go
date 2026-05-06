package anthropic

import "strings"

// DefaultModelCapabilities defines default capabilities for Anthropic models
var DefaultModelCapabilities = map[string]Capabilities{
	// Claude 4.x models
	"claude-opus-4": {
		Vision:          "claude",
		ToolCalls:       true,
		Audio:           false,
		Reasoning:       true,
		Streaming:       true,
		JSON:            true,
		Multimodal:      true,
		MaxInputTokens:  200000,
		MaxOutputTokens: 32768,
	},
	"claude-sonnet-4": {
		Vision:          "claude",
		ToolCalls:       true,
		Audio:           false,
		Reasoning:       false,
		Streaming:       true,
		JSON:            true,
		Multimodal:      true,
		MaxInputTokens:  200000,
		MaxOutputTokens: 16384,
	},
	"claude-haiku-4": {
		Vision:          "claude",
		ToolCalls:       true,
		Audio:           false,
		Reasoning:       false,
		Streaming:       true,
		JSON:            true,
		Multimodal:      true,
		MaxInputTokens:  200000,
		MaxOutputTokens: 8192,
	},

	// Claude 3.5 models
	"claude-3-5-sonnet": {
		Vision:          "claude",
		ToolCalls:       true,
		Audio:           false,
		Reasoning:       false,
		Streaming:       true,
		JSON:            true,
		Multimodal:      true,
		MaxInputTokens:  200000,
		MaxOutputTokens: 8192,
	},
	"claude-3-5-haiku": {
		Vision:          "claude",
		ToolCalls:       true,
		Audio:           false,
		Reasoning:       false,
		Streaming:       true,
		JSON:            true,
		Multimodal:      true,
		MaxInputTokens:  200000,
		MaxOutputTokens: 8192,
	},

	// Kimi K2 Coding models (Anthropic-compatible endpoint)
	"kimi-k2": {
		Vision:          false,
		ToolCalls:       true,
		Audio:           false,
		Reasoning:       true,
		Streaming:       true,
		JSON:            true,
		Multimodal:      false,
		MaxInputTokens:  131072,
		MaxOutputTokens: 131072,
	},
	"kimi-k2.5": {
		Vision:          "claude",
		ToolCalls:       true,
		Audio:           false,
		Reasoning:       true,
		Streaming:       true,
		JSON:            true,
		Multimodal:      true,
		MaxInputTokens:  262142,
		MaxOutputTokens: 262142,
	},
	"kimi-for-coding": {
		Vision:          false,
		ToolCalls:       true,
		Audio:           false,
		Reasoning:       true,
		Streaming:       true,
		JSON:            true,
		Multimodal:      false,
		MaxInputTokens:  131072,
		MaxOutputTokens: 131072,
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
