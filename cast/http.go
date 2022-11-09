package cast

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// AnyToHeaders cast to http.Header
func AnyToHeaders(v interface{}) (http.Header, error) {
	values, err := AnyToURLValues(v)
	if err != nil {
		return nil, err
	}
	return http.Header(values), nil
}

// AnyToURLValues cast to url.Values
func AnyToURLValues(v interface{}) (url.Values, error) {

	values := url.Values{}
	if v == nil {
		return values, nil
	}

	switch input := v.(type) {

	// k1=v1&k2=v2&k3=v3, ?k1=v1&k2=v2&k3=v3
	case string:
		input = strings.TrimPrefix(input, "?")
		query, err := url.ParseQuery(input)
		if err != nil {
			return nil, err
		}
		return query, nil

	// {"k1":"v1", "k2":1, "k3":true, "k4":0.618}
	case map[string]interface{}:
		for key, val := range input {
			values.Add(key, fmt.Sprintf("%v", val))
		}
		return values, nil

	// {"k1": "v1", "k2": "v2"},
	case map[string]string:
		for key, val := range input {
			values.Add(key, val)
		}
		return values, nil

	// [{"k1": "v1"}, {"k1": "v11"}, {"k2": 1}, {"k3": true}, {"k4": 0.618}]
	case []map[string]interface{}:
		err := ArrayToURLValues(values, input)
		if err != nil {
			return nil, err
		}
		return values, nil

	// [{"k1": "v1"}, {"k1": "v11"}, {"k2": "v2"}]
	case []map[string]string:
		err := ArrayToURLValues(values, input)
		if err != nil {
			return nil, err
		}
		return values, nil

	// ["k1=v1","k1"="v11","k2"="v2"]
	case []string:
		err := ArrayToURLValues(values, input)
		if err != nil {
			return nil, err
		}
		return values, nil

	// ["k1=v1", "k1=v11", {"k2": 1}, {"k3": true}, "k2=v2"]
	case []interface{}:
		err := ArrayToURLValues(values, input)
		if err != nil {
			return nil, err
		}
		return values, nil
	}

	return nil, fmt.Errorf("Unknown type %#v", v)
}

// ArrayToURLValues cast array to url values
func ArrayToURLValues[T string | interface{} | map[string]interface{} | map[string]string](values url.Values, input []T) error {
	for _, item := range input {
		vals, err := AnyToURLValues(item)
		if err != nil {
			return err
		}
		MergeURLValues(values, vals)
	}
	return nil
}

// MergeURLValues merge URL Values
func MergeURLValues(values, new url.Values) {
	for k, val := range new {
		for _, v := range val {
			values.Add(k, v)
		}
	}
}
