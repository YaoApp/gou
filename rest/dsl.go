package rest

import jsoniter "github.com/json-iterator/go"

// compile the source
func compile(name string, source []byte) error {
	rest := REST{}
	err := jsoniter.Unmarshal(source, &rest)
	if err != nil {
		return err
	}
	return nil
}
