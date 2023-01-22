package model

// Encryptor 加密器
type Encryptor struct {
	Name   string `json:"-"`                // 名称
	Source string `json:"-"`                // 来源
	Salt   string `json:"salt,omitempty"`   // 盐
	Key    string `json:"key,omitempty"`    // 钥匙
	Secret string `json:"secret,omitempty"` // 密钥
}

// IEncryptor 加密器接口
type IEncryptor interface {
	Set(crypt Encryptor)
	Encode(value string) (string, error)
	Decode(value string) (string, error)
	Validate(hash string, value string) bool
}
