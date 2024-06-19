package encoding

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"gopkg.in/yaml.v3"
)

func TestYAMLEncode(t *testing.T) {
	data := []string{"foo", "bar"}
	res, err := process.New("encoding.yaml.Encode", data).Exec()
	if err != nil {
		t.Fatal(err)
	}

	new := []string{}
	err = yaml.Unmarshal([]byte(res.(string)), &new)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, new, data)
}

func TestYAMLDecode(t *testing.T) {
	data := fmt.Sprintf("\n- foo\n- bar\n")
	res, err := process.New("encoding.yaml.Decode", data).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []interface{}{"foo", "bar"}, res)
}
