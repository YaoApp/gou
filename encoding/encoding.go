package encoding

import (
	"github.com/yaoapp/gou/encoding/base64"
	"github.com/yaoapp/gou/encoding/hex"
	"github.com/yaoapp/gou/encoding/json"
	"github.com/yaoapp/gou/encoding/xml"
	"github.com/yaoapp/gou/encoding/yaml"
	"github.com/yaoapp/gou/process"
)

func init() {
	process.Register("encoding.base64.Encode", base64.ProcessEncode)
	process.Register("encoding.base64.Decode", base64.ProcessDecode)
	process.Register("encoding.hex.Encode", hex.ProcessEncode)
	process.Register("encoding.hex.Decode", hex.ProcessDecode)
	process.Register("encoding.json.Encode", json.ProcessEncode)
	process.Register("encoding.json.Decode", json.ProcessDecode)
	process.Register("encoding.yaml.Encode", yaml.ProcessEncode)
	process.Register("encoding.yaml.Decode", yaml.ProcessDecode)
	process.Register("encoding.xml.Encode", xml.ProcessEncode)
	process.Register("encoding.xml.Decode", xml.ProcessDecode)
}
