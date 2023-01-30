package encoding

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestHexEncode(t *testing.T) {
	data := "SomeData"
	res, err := process.New("encoding.hex.Encode", data).Exec()
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
	res, err := process.New("encoding.hex.Decode", data).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "SomeData", string(res.(string)))
}
