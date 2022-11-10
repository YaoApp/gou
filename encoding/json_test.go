package encoding

import (
	"testing"

	"github.com/go-playground/assert/v2"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou"
)

func TestJSONEncode(t *testing.T) {
	data := []string{"foo", "bar"}
	res, err := gou.NewProcess("encoding.json.Encode", data).Exec()
	if err != nil {
		t.Fatal(err)
	}

	new := []string{}
	err = jsoniter.Unmarshal([]byte(res.(string)), &new)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, new, data)
}

func TestJSONDecode(t *testing.T) {
	data := `["foo", "bar"]`
	res, err := gou.NewProcess("encoding.json.Decode", data).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []interface{}{"foo", "bar"}, res)
}
