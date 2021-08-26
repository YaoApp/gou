package gou

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/kun/exception"
	"golang.org/x/crypto/bcrypt"
)

// Encryptors 已加载加密器
var Encryptors = map[string]Encryptor{}

// IEncryptors 加密码器接口映射
var IEncryptors = map[string]IEncryptor{
	"AES":      &EncryptorAES{},
	"PASSWORD": &EncryptorPassword{},
}

// LoadCrypt 载入数据模型
func LoadCrypt(source string, name string) Encryptor {
	var input io.Reader = nil
	if strings.HasPrefix(source, "file://") {
		filename := strings.TrimPrefix(source, "file://")
		file, err := os.Open(filename)
		if err != nil {
			exception.Err(err, 400).Throw()
		}
		defer file.Close()
		input = file
	} else {
		input = strings.NewReader(source)
	}

	encryptor := Encryptor{}
	err := helper.UnmarshalFile(input, &encryptor)
	if err != nil {
		panic(err)
	}
	encryptor.Name = name
	encryptor.Source = source
	Encryptors[name] = encryptor
	return encryptor
}

// SelectCrypt 选择加密器
func SelectCrypt(name string) IEncryptor {
	encryptor, has := Encryptors[name]
	if !has {
		exception.New(
			fmt.Sprintf("加密器:%s; 尚未加载", name),
			400,
		).Throw()
	}

	iencryptor, has := IEncryptors[name]
	if !has {
		exception.New(
			fmt.Sprintf("加密器:%s; 上未定义", name),
			400,
		).Throw()
	}

	iencryptor.Set(encryptor)
	return iencryptor
}

// EncryptorAES AES
type EncryptorAES struct{ Encryptor }

// EncryptorAES256 AES 256
type EncryptorAES256 struct{ Encryptor }

// EncryptorAES128 AES 128
type EncryptorAES128 struct{ Encryptor }

// EncryptorPassword 密码加密
type EncryptorPassword struct{ Encryptor }

// Set AES Encode
func (aes *EncryptorAES) Set(crypt Encryptor) {
	aes.Encryptor = crypt
}

// Encode AES Encode
func (aes EncryptorAES) Encode(value string) (string, error) {
	return fmt.Sprintf("HEX(AES_ENCRYPT('%s', '%s'))", value, aes.Key), nil
}

// Decode AES Decode
func (aes EncryptorAES) Decode(field string) (string, error) {
	return fmt.Sprintf("AES_DECRYPT(UNHEX(`%s`), '%s') as `%s`", field, aes.Key, field), nil
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
func (pwd *EncryptorPassword) Set(crypt Encryptor) {
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
