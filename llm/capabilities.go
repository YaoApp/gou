package llm

// Capabilities defines the capabilities of a language model.
// This is the universal capability definition shared by all LLM connectors (OpenAI, Anthropic, etc.).
// Configure via the connector's options.capabilities field.
type Capabilities struct {
	Vision                interface{} `json:"vision,omitempty" yaml:"vision,omitempty"`                                 // Supports vision/image input: bool or VisionFormat string ("openai", "claude"/"base64", "default")
	Audio                 bool        `json:"audio,omitempty" yaml:"audio,omitempty"`                                   // Supports audio input/output (multimodal audio understanding)
	STT                   bool        `json:"stt,omitempty" yaml:"stt,omitempty"`                                       // Speech-to-Text / audio transcription model (e.g. Whisper)
	ToolCalls             bool        `json:"tool_calls,omitempty" yaml:"tool_calls,omitempty"`                         // Supports tool/function calling
	Reasoning             bool        `json:"reasoning,omitempty" yaml:"reasoning,omitempty"`                           // Supports reasoning/thinking mode (o1, DeepSeek R1)
	Streaming             bool        `json:"streaming,omitempty" yaml:"streaming,omitempty"`                           // Supports streaming responses
	JSON                  bool        `json:"json,omitempty" yaml:"json,omitempty"`                                     // Supports JSON mode
	Multimodal            bool        `json:"multimodal,omitempty" yaml:"multimodal,omitempty"`                         // Supports multimodal input
	Embedding             bool        `json:"embedding,omitempty" yaml:"embedding,omitempty"`                           // Text embedding model (e.g. text-embedding-3-large)
	ImageGeneration       bool        `json:"image_generation,omitempty" yaml:"image_generation,omitempty"`             // Image generation model (e.g. DALL-E, Seedream)
	TemperatureAdjustable bool        `json:"temperature_adjustable,omitempty" yaml:"temperature_adjustable,omitempty"` // Supports temperature adjustment (reasoning models typically don't)
	MaxInputTokens        int         `json:"max_input_tokens,omitempty" yaml:"max_input_tokens,omitempty"`             // Maximum input context window size (aligned with Anthropic Models API)
	MaxOutputTokens       int         `json:"max_output_tokens,omitempty" yaml:"max_output_tokens,omitempty"`           // Maximum output tokens the model can generate
}

// HasVision reports whether the model supports vision/image input.
// Vision is interface{} and may be bool, string (format name), or nil.
func (c *Capabilities) HasVision() bool {
	if c == nil || c.Vision == nil {
		return false
	}
	switch v := c.Vision.(type) {
	case bool:
		return v
	case string:
		return v != ""
	default:
		return true
	}
}

// HasReasoning reports whether the model supports thinking/reasoning mode.
func (c *Capabilities) HasReasoning() bool {
	return c != nil && c.Reasoning
}

// HasToolCalls reports whether the model supports tool/function calling.
func (c *Capabilities) HasToolCalls() bool {
	return c != nil && c.ToolCalls
}

// ToMap converts Capabilities to map[string]interface{} for API responses.
// This is the canonical implementation; yao-layer wrappers should delegate here.
func (c *Capabilities) ToMap() map[string]interface{} {
	if c == nil {
		return nil
	}
	result := make(map[string]interface{})
	if c.Vision != nil {
		result["vision"] = c.Vision
	}
	result["audio"] = c.Audio
	result["stt"] = c.STT
	result["tool_calls"] = c.ToolCalls
	result["reasoning"] = c.Reasoning
	result["streaming"] = c.Streaming
	result["json"] = c.JSON
	result["multimodal"] = c.Multimodal
	result["embedding"] = c.Embedding
	result["image_generation"] = c.ImageGeneration
	result["temperature_adjustable"] = c.TemperatureAdjustable
	return result
}
