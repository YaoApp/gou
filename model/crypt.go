package model

import (
	"fmt"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"golang.org/x/crypto/bcrypt"
)

// Encryptors 已加载加密器
var Encryptors = map[string]*Encryptor{}

// IEncryptors 加密码器接口映射
var IEncryptors = map[string]IEncryptor{
	"AES":      &EncryptorAES{},
	"PASSWORD": &EncryptorPassword{},
}

// WithCrypt 载入数据加密器
func WithCrypt(data []byte, name string) (*Encryptor, error) {
	encryptor := Encryptor{}
	err := jsoniter.Unmarshal(data, &encryptor)
	if err != nil {
		return nil, err

	}
	encryptor.Name = name
	Encryptors[name] = &encryptor
	return Encryptors[name], nil
}

// SelectCrypt 选择加密器
func SelectCrypt(name string) (IEncryptor, error) {
	encryptor, has := Encryptors[name]
	if !has {
		return nil, fmt.Errorf("加密器:%s; 尚未加载", name)
	}

	iencryptor, has := IEncryptors[name]
	if !has {
		return nil, fmt.Errorf("加密器:%s; 尚不支持", name)
	}

	iencryptor.Set(encryptor)
	return iencryptor, nil
}

// EncryptorAES AES
type EncryptorAES struct{ *Encryptor }

// EncryptorAES256 AES 256
type EncryptorAES256 struct{ Encryptor }

// EncryptorAES128 AES 128
type EncryptorAES128 struct{ Encryptor }

// EncryptorPassword 密码加密
type EncryptorPassword struct{ *Encryptor }

// Set AES Encode
func (aes *EncryptorAES) Set(crypt *Encryptor) {
	aes.Encryptor = crypt
}

// Encode AES Encode
func (aes EncryptorAES) Encode(value string) (string, error) {
	return fmt.Sprintf("HEX(AES_ENCRYPT('%s', '%s'))", value, aes.Key), nil
}

// Decode AES Decode
func (aes EncryptorAES) Decode(field string) (string, error) {
	if strings.Contains(field, ".") {
		namer := strings.Split(field, ".")
		return fmt.Sprintf("AES_DECRYPT(UNHEX(`%s`.`%s`), '%s')", namer[0], namer[1], aes.Key), nil
	}
	return fmt.Sprintf("AES_DECRYPT(UNHEX(`%s`), '%s'", field, aes.Key), nil
}

// Validate AES Decode
func (aes EncryptorAES) Validate(hash string, field string) bool {
	plain, err := aes.Decode(hash)
	if err != nil {
		return false
	}
	return plain == field
}

// Set AES Encode
func (pwd *EncryptorPassword) Set(crypt *Encryptor) {
	pwd.Encryptor = crypt
}

// Encode PASSWORD Encode
func (pwd EncryptorPassword) Encode(value string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(value), bcrypt.MinCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Validate PASSWORD Decode
func (pwd EncryptorPassword) Validate(hash string, value string) bool {
	byteHash := []byte(hash)
	err := bcrypt.CompareHashAndPassword(byteHash, []byte(value))
	if err != nil {
		return false
	}
	return true
}

// Decode PASSWORD Decode
func (pwd EncryptorPassword) Decode(value string) (string, error) {
	return value, nil
}
