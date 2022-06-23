package encoding

import (
	"encoding/base64"
	"testing"

	"github.com/go-playground/assert/v2"
	"github.com/yaoapp/gou"
)

func TestBase64Encode(t *testing.T) {
	data := "SomeData"
	res, err := gou.NewProcess("encoding.base64.Encode", data).Exec()
	if err != nil {
		t.Fatal(err)
	}
	v, err := base64.StdEncoding.DecodeString(res.(string))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, string(v), data)
}

func TestBase64Decode(t *testing.T) {
	data := base64.StdEncoding.EncodeToString([]byte("SomeData"))
	res, err := gou.NewProcess("encoding.base64.Decode", data).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "SomeData", string(res.(string)))
}
