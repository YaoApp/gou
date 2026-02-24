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
	TemperatureAdjustable bool        `json:"temperature_adjustable,omitempty" yaml:"temperature_adjustable,omitempty"` // Supports temperature adjustment (reasoning models typically don't)
}
