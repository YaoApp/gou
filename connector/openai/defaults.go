package openai

import "strings"

// DefaultModelCapabilities defines default capabilities for common model patterns
// Based on top models from https://artificialanalysis.ai/leaderboards/models (Intelligence Index > 50)
var DefaultModelCapabilities = map[string]Capabilities{
	// Google Gemini Models (Index: 73, 65, 60, 54)
	"gemini-3-pro": {
		Vision:     "openai",
		ToolCalls:  true,
		Audio:      true,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: true,
	},
	"gemini-2.5-pro": {
		Vision:     "openai",
		ToolCalls:  true,
		Audio:      true,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: true,
	},
	"gemini-2.5-flash": {
		Vision:     "openai",
		ToolCalls:  true,
		Audio:      true,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: true,
	},

	// Anthropic Claude Models (Index: 70, 63, 60, 55)
	"claude-opus-4.5": {
		Vision:     "claude",
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: true,
	},
	"claude-4.5-sonnet": {
		Vision:     "claude",
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: true,
	},
	"claude-4.5-haiku": {
		Vision:     "claude",
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: true,
	},

	// OpenAI Models (Index: 70, 68, 65, 64, 62)
	"gpt-5.1": {
		Vision:     "openai",
		ToolCalls:  true,
		Audio:      true,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: true,
	},
	"gpt-5": {
		Vision:     "openai",
		ToolCalls:  true,
		Audio:      true,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: true,
	},
	"gpt-5-mini": {
		Vision:     "openai",
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: true,
	},
	"gpt-4o": {
		Vision:     "openai",
		ToolCalls:  true,
		Audio:      true, // GPT-4o supports audio via Realtime API
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: true,
	},
	"o3": {
		Vision:     "openai",
		ToolCalls:  true, // o3 supports tool calls
		Audio:      false,
		Reasoning:  true,  // Reasoning model
		Streaming:  false, // No streaming for reasoning models
		JSON:       false, // No JSON mode
		Multimodal: true,
	},
	"o1": {
		Vision:     "openai",
		ToolCalls:  true, // o1 now supports tool calls (updated)
		Audio:      false,
		Reasoning:  true,  // Reasoning model
		Streaming:  false, // No streaming for reasoning models
		JSON:       false, // No JSON mode
		Multimodal: true,  // Supports vision
	},

	// xAI Grok Models (Index: 65, 64, 60)
	"grok-4": {
		Vision:     "openai",
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: true,
	},
	"grok-4.1-fast": {
		Vision:     "openai",
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: true,
	},

	// Moonshot Kimi Models (Index: 67)
	"kimi-k2": {
		Vision:     false,
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  true,
		Streaming:  true,
		JSON:       true,
		Multimodal: false,
	},

	// DeepSeek Models (Index: 58, 57, 52)
	"deepseek-v3": {
		Vision:     false,
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: false,
	},
	"deepseek-r1": {
		Vision:     false,
		ToolCalls:  false, // R1 does NOT support tool calling
		Audio:      false,
		Reasoning:  true,
		Streaming:  true,
		JSON:       true,
		Multimodal: false,
	},
	"deepseek-chat": {
		Vision:     false,
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: false,
	},

	// Alibaba Qwen Models (Index: 57, 56, 55, 54)
	"qwen3-235b": {
		Vision:     false,
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: false,
	},
	"qwen3-max": {
		Vision:     "openai",
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: true,
	},
	"qwen3-vl": {
		Vision:     "openai",
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: true,
	},
	"qwen3-omni": {
		Vision:     "openai",
		ToolCalls:  true,
		Audio:      true, // Qwen3 Omni supports audio
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: true,
	},
	"qwen2.5": {
		Vision:     false,
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: false,
	},

	// ByteDance Models (Index: 57)
	"doubao-seed": {
		Vision:     false,
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: false,
	},

	// Z AI GLM Models (Index: 56)
	"glm-4.6": {
		Vision:     false,
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: false,
	},
	"glm-4v": {
		Vision:     "openai",
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: true, // GLM-4V is vision model
	},

	// MiniMax Models (Index: 61)
	"minimax-m2": {
		Vision:     false,
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  false,
		Streaming:  true,
		JSON:       true,
		Multimodal: false,
	},

	// Mistral Models (Index: 52)
	"mistral-large": {
		Vision:     false,
		ToolCalls:  true,
		Audio:      false,
		Reasoning:  false,
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

	// Convert to lowercase for case-insensitive matching
	modelLower := strings.ToLower(model)

	// Try exact match first (for performance)
	for pattern, caps := range DefaultModelCapabilities {
		if strings.Contains(modelLower, pattern) {
			capsCopy := caps
			return &capsCopy
		}
	}

	return nil
}
