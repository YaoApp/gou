package cast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnyToURLValues(t *testing.T) {

	values, err := AnyToURLValues("k1=v1&k2=v2&k3=v3")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "v1", values.Get("k1"))
	assert.Equal(t, "v2", values.Get("k2"))
	assert.Equal(t, "v3", values.Get("k3"))

	values, err = AnyToURLValues("?k1=v1&k2=v2&k3=v3")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "v1", values.Get("k1"))
	assert.Equal(t, "v2", values.Get("k2"))
	assert.Equal(t, "v3", values.Get("k3"))

	values, err = AnyToURLValues(map[string]interface{}{"k1": "v1", "k2": 1, "k3": true, "k4": 0.618})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "v1", values.Get("k1"))
	assert.Equal(t, "1", values.Get("k2"))
	assert.Equal(t, "true", values.Get("k3"))
	assert.Equal(t, "0.618", values.Get("k4"))

	values, err = AnyToURLValues(map[string]string{"k1": "v1", "k2": "v2"})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "v1", values.Get("k1"))
	assert.Equal(t, "v2", values.Get("k2"))

	values, err = AnyToURLValues([]map[string]interface{}{{"k1": "v1"}, {"k1": "v11"}, {"k2": 1}, {"k3": true}, {"k4": 0.618}})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "v1", values.Get("k1"))
	assert.Equal(t, []string{"v1", "v11"}, values["k1"])
	assert.Equal(t, "1", values.Get("k2"))
	assert.Equal(t, "true", values.Get("k3"))
	assert.Equal(t, "0.618", values.Get("k4"))

	values, err = AnyToURLValues([]map[string]string{{"k1": "v1"}, {"k1": "v11"}, {"k2": "v2"}})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "v1", values.Get("k1"))
	assert.Equal(t, []string{"v1", "v11"}, values["k1"])
	assert.Equal(t, "v2", values.Get("k2"))

	values, err = AnyToURLValues([]string{"k1=v1", "k1=v11", "k2=v2"})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "v1", values.Get("k1"))
	assert.Equal(t, []string{"v1", "v11"}, values["k1"])
	assert.Equal(t, "v2", values.Get("k2"))

	values, err = AnyToURLValues([]interface{}{"k1=v1", "k1=v11", map[string]interface{}{"k2": 1}, map[string]interface{}{"k3": true}, "k2=v2"})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "v1", values.Get("k1"))
	assert.Equal(t, []string{"v1", "v11"}, values["k1"])
	assert.Equal(t, "1", values.Get("k2"))
	assert.Equal(t, []string{"1", "v2"}, values["k2"])
	assert.Equal(t, "true", values.Get("k3"))

	values, err = AnyToURLValues([]int{1, 2, 3})
	assert.NotNil(t, err)
}

func TestAnyToHeaders(t *testing.T) {

	headers, err := AnyToHeaders(map[string]interface{}{"Content-Type": "application/json"})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "application/json", headers.Get("Content-Type"))

	headers, err = AnyToHeaders([]interface{}{
		map[string]string{"Content-Type": "application/json"},
		map[string]string{"Content-Type": "text/html"},
	})

	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "application/json", headers.Get("Content-Type"))
	assert.Equal(t, []string{"application/json", "text/html"}, headers["Content-Type"])
}
