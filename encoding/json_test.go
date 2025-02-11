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

func TestJSONRepair(t *testing.T) {
	data := `{"name": "foo", "items": ["bar" "baz"]}` // Invalid JSON missing comma
	res, err := process.New("encoding.json.Repair", data).Exec()
	if err != nil {
		t.Fatal(err)
	}

	// Verify the repaired JSON is valid
	var obj map[string]interface{}
	err = jsoniter.UnmarshalFromString(res.(string), &obj)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "foo", obj["name"])
	assert.Equal(t, []interface{}{"bar", "baz"}, obj["items"])
}

func TestJSONParse(t *testing.T) {
	// Test case 1: Valid JSON
	data1 := `{"name": "foo", "items": ["bar", "baz"]}`
	res1, err := process.New("encoding.json.Parse", data1).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "foo", res1.(map[string]interface{})["name"])
	assert.Equal(t, []interface{}{"bar", "baz"}, res1.(map[string]interface{})["items"])

	// Test case 2: Invalid JSON that needs repair
	data2 := `{"name": "foo", "items": ["bar" "baz"]}`
	res2, err := process.New("encoding.json.Parse", data2).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "foo", res2.(map[string]interface{})["name"])
	assert.Equal(t, []interface{}{"bar", "baz"}, res2.(map[string]interface{})["items"])
}
