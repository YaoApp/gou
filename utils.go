package gou

import (
	"os"
	"strings"

	"github.com/yaoapp/kun/any"
)

// EnvString replace $ENV.xxx with the env
func EnvString(key interface{}, defaults ...string) string {
	k, ok := key.(string)
	if !ok {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return ""
	}

	if ok && strings.HasPrefix(k, "$ENV.") {
		k = strings.TrimPrefix(k, "$ENV.")
		v := os.Getenv(k)
		if v == "" && len(defaults) > 0 {
			return defaults[0]
		}
		return v
	}
	return key.(string)
}

// EnvInt replace $ENV.xxx with the env and cast to the integer
func EnvInt(key interface{}, defaults ...int) int {
	if k, ok := key.(string); ok && strings.HasPrefix(k, "$ENV.") {
		k = strings.TrimPrefix(k, "$ENV.")
		v := os.Getenv(k)
		if v == "" {
			if len(defaults) > 0 {
				return defaults[0]
			}
			return 0
		}
		return any.Of(v).CInt()
	}

	v, ok := key.(int)
	if !ok {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return any.Of(key).CInt()
	}
	return v
}
