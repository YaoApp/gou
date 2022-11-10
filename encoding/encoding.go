package encoding

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/encoding/base64"
	"github.com/yaoapp/gou/encoding/hex"
	"github.com/yaoapp/gou/encoding/json"
)

func init() {
	gou.RegisterProcessHandler("encoding.base64.Encode", base64.ProcessEncode)
	gou.RegisterProcessHandler("encoding.base64.Decode", base64.ProcessDecode)
	gou.RegisterProcessHandler("encoding.hex.Encode", hex.ProcessEncode)
	gou.RegisterProcessHandler("encoding.hex.Decode", hex.ProcessDecode)
	gou.RegisterProcessHandler("encoding.json.Encode", json.ProcessEncode)
	gou.RegisterProcessHandler("encoding.json.Decode", json.ProcessDecode)
}
