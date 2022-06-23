package encoding

import (
	"encoding/hex"
	"testing"

	"github.com/go-playground/assert/v2"
	"github.com/yaoapp/gou"
)

func TestHexEncode(t *testing.T) {
	data := "SomeData"
	res, err := gou.NewProcess("encoding.hex.Encode", data).Exec()
	if err != nil {
		t.Fatal(err)
	}
	v, err := hex.DecodeString(res.(string))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, string(v), data)
}

func TestHexDecode(t *testing.T) {
	data := hex.EncodeToString([]byte("SomeData"))
	res, err := gou.NewProcess("encoding.hex.Decode", data).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "SomeData", string(res.(string)))
}
