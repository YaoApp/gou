package connector

import "github.com/yaoapp/gou/llm"

// defaultParamsByType lists API request-body parameters accepted by each
// provider type. Used as fallback when a connector has no explicit
// supported_params configuration.
var defaultParamsByType = map[string]map[string]*llm.ParamSpec{
	"openai": {
		"temperature": nil, "max_tokens": nil, "max_completion_tokens": nil,
		"top_p": nil, "n": nil, "stop": nil, "stream": nil,
		"presence_penalty": nil, "frequency_penalty": nil,
		"logit_bias": nil, "user": nil,
		"response_format": nil, "seed": nil,
		"tools": nil, "tool_choice": nil,
		"reasoning_effort": nil, "audio": nil, "thinking": nil,
		"stream_options": nil,
	},
	"anthropic": {
		"temperature": nil, "max_tokens": nil,
		"top_p": nil, "top_k": nil, "stop_sequences": nil,
		"tools": nil, "tool_choice": nil,
		"thinking": nil, "stream": nil,
	},
}

// connectorMetadataKeys are fields that should NEVER enter an API request body.
var connectorMetadataKeys = map[string]bool{
	"host": true, "key": true, "model": true, "proxy": true,
	"type": true, "azure": true, "capabilities": true,
	"protocols": true, "supported_params": true, "auth_mode": true,
	"version": true, "organization": true,
	"max_input_tokens": true, "max_output_tokens": true,
}

// FilterRequestBodyParams filters Setting() output to only keep parameters
// that belong in an API request body, applying value constraints when available.
//
// Priority:
//  1. conn implements llm.LLMConnector with non-nil GetSupportedParams() → strict whitelist + clamp
//  2. conn type matches a known provider → provider-type default set
//  3. neither → strip known metadata keys, pass through the rest
func FilterRequestBodyParams(settings map[string]interface{}, conn Connector) map[string]interface{} {
	if settings == nil {
		return make(map[string]interface{})
	}

	// 1. Try LLMConnector interface
	if conn != nil {
		if lc, ok := conn.(llm.LLMConnector); ok {
			if specs := lc.GetSupportedParams(); specs != nil {
				return filterAndClamp(settings, specs)
			}
		}
	}

	// 2. Fall back to provider-type defaults
	if conn != nil {
		providerType := connectorTypeName(conn)
		if defaults, ok := defaultParamsByType[providerType]; ok {
			return filterAndClamp(settings, defaults)
		}
	}

	// 3. Unknown type: strip metadata keys, pass everything else
	return stripMetadataKeys(settings)
}

// filterAndClamp keeps only whitelisted keys and applies value constraints.
func filterAndClamp(settings map[string]interface{}, specs map[string]*llm.ParamSpec) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range settings {
		spec, allowed := specs[k]
		if !allowed {
			continue
		}
		if spec != nil {
			v = clampValue(v, spec)
		}
		result[k] = v
	}
	return result
}

// stripMetadataKeys removes known connector metadata, keeping everything else.
func stripMetadataKeys(settings map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range settings {
		if connectorMetadataKeys[k] {
			continue
		}
		result[k] = v
	}
	return result
}

// clampValue applies min/max/allowedValues constraints to a value.
func clampValue(v interface{}, spec *llm.ParamSpec) interface{} {
	f, ok := toFloat64(v)
	if !ok {
		if len(spec.AllowedValues) > 0 {
			return clampAllowed(v, spec.AllowedValues)
		}
		return v
	}

	if spec.Min != nil && f < *spec.Min {
		f = *spec.Min
	}
	if spec.Max != nil && f > *spec.Max {
		f = *spec.Max
	}

	if len(spec.AllowedValues) > 0 {
		return clampAllowed(f, spec.AllowedValues)
	}

	// Preserve original integer type if the value was integral
	if _, wasInt := v.(int); wasInt {
		return int(f)
	}
	return f
}

// clampAllowed returns v if it matches any allowed value, otherwise returns the first allowed value.
func clampAllowed(v interface{}, allowed []interface{}) interface{} {
	for _, a := range allowed {
		if v == a {
			return v
		}
		// Handle numeric comparisons (json numbers are float64)
		vf, vok := toFloat64(v)
		af, aok := toFloat64(a)
		if vok && aok && vf == af {
			return v
		}
	}
	if len(allowed) > 0 {
		return allowed[0]
	}
	return v
}

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	}
	return 0, false
}

// connectorTypeName maps a Connector to a provider type string
// for looking up default parameter sets.
func connectorTypeName(conn Connector) string {
	if conn == nil {
		return ""
	}
	switch {
	case conn.Is(OPENAI):
		return "openai"
	case conn.Is(ANTHROPIC):
		return "anthropic"
	default:
		return ""
	}
}
