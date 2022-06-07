package u

import (
	"errors"
	"fmt"
	"os"

	jsoniter "github.com/json-iterator/go"
)

// FileGetMap read the json file and return map[string]interface{}
func FileGetMap(file string) (map[string]interface{}, error) {
	_, err := os.Stat(file)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%s not exists", file)
	} else if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	bytes := ToJSON(data, nil)
	return ToMap(bytes)
}

// FileGetJSON trans JSONC to JSON
func FileGetJSON(file string) ([]byte, error) {
	_, err := os.Stat(file)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%s not exists", file)
	} else if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return ToJSON(data, nil), nil
}

// ToMap parse and  return map[string]interface{}
func ToMap(data []byte) (map[string]interface{}, error) {

	bytes := ToJSON(data, nil)
	res := map[string]interface{}{}
	err := jsoniter.Unmarshal(bytes, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ToJSON strips out comments and trailing commas and convert the input to a
// valid JSON per the official spec: https://tools.ietf.org/html/rfc8259
//
// The resulting JSON will always be the same length as the input and it will
// include all of the same line breaks at matching offsets. This is to ensure
// the result can be later processed by a external parser and that that
// parser will report messages or errors with the correct offsets.
func ToJSON(src, dst []byte) []byte {
	dst = dst[:0]
	for i := 0; i < len(src); i++ {
		if src[i] == '/' {
			if i < len(src)-1 {
				if src[i+1] == '/' {
					dst = append(dst, ' ', ' ')
					i += 2
					for ; i < len(src); i++ {
						if src[i] == '\n' {
							dst = append(dst, '\n')
							break
						} else if src[i] == '\t' || src[i] == '\r' {
							dst = append(dst, src[i])
						} else {
							dst = append(dst, ' ')
						}
					}
					continue
				}
				if src[i+1] == '*' {
					dst = append(dst, ' ', ' ')
					i += 2
					for ; i < len(src)-1; i++ {
						if src[i] == '*' && src[i+1] == '/' {
							dst = append(dst, ' ', ' ')
							i++
							break
						} else if src[i] == '\n' || src[i] == '\t' ||
							src[i] == '\r' {
							dst = append(dst, src[i])
						} else {
							dst = append(dst, ' ')
						}
					}
					continue
				}
			}
		}
		dst = append(dst, src[i])
		if src[i] == '"' {
			for i = i + 1; i < len(src); i++ {
				dst = append(dst, src[i])
				if src[i] == '"' {
					j := i - 1
					for ; ; j-- {
						if src[j] != '\\' {
							break
						}
					}
					if (j-i)%2 != 0 {
						break
					}
				}
			}
		} else if src[i] == '}' || src[i] == ']' {
			for j := len(dst) - 2; j >= 0; j-- {
				if dst[j] <= ' ' {
					continue
				}
				if dst[j] == ',' {
					dst[j] = ' '
				}
				break
			}
		}
	}
	return dst
}
