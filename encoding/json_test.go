package encoding

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestJSONEncode(t *testing.T) {
	data := []string{"foo", "bar"}
	res, err := process.New("encoding.json.Encode", data).Exec()
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
	res, err := process.New("encoding.json.Decode", data).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []interface{}{"foo", "bar"}, res)
}
