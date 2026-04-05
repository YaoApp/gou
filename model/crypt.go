package model

import (
	"fmt"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"golang.org/x/crypto/bcrypt"
)

// escapeSQLString escapes single quotes for safe embedding in SQL string literals.
func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

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
	return fmt.Sprintf("HEX(AES_ENCRYPT('%s', '%s'))", escapeSQLString(value), escapeSQLString(aes.Key)), nil
}

// Decode AES Decode
func (aes EncryptorAES) Decode(field string) (string, error) {
	if strings.Contains(field, ".") {
		namer := strings.Split(field, ".")
		return fmt.Sprintf("AES_DECRYPT(UNHEX(`%s`.`%s`), '%s')", namer[0], namer[1], escapeSQLString(aes.Key)), nil
	}
	return fmt.Sprintf("AES_DECRYPT(UNHEX(`%s`), '%s')", field, escapeSQLString(aes.Key)), nil
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

// EncryptorPGCrypto uses PostgreSQL pgcrypto extension for AES-equivalent encryption
type EncryptorPGCrypto struct{ *Encryptor }

// Set PGCrypto Encryptor
func (pg *EncryptorPGCrypto) Set(crypt *Encryptor) {
	pg.Encryptor = crypt
}

// Encode PGCrypto Encode — returns SQL expression using pgp_sym_encrypt + hex encoding
func (pg EncryptorPGCrypto) Encode(value string) (string, error) {
	return fmt.Sprintf("encode(pgp_sym_encrypt('%s', '%s'), 'hex')", escapeSQLString(value), escapeSQLString(pg.Key)), nil
}

// Decode PGCrypto Decode — returns SQL expression using pgp_sym_decrypt + hex decoding
func (pg EncryptorPGCrypto) Decode(field string) (string, error) {
	if strings.Contains(field, ".") {
		namer := strings.Split(field, ".")
		return fmt.Sprintf(`pgp_sym_decrypt(decode("%s"."%s", 'hex'), '%s')`, namer[0], namer[1], escapeSQLString(pg.Key)), nil
	}
	return fmt.Sprintf(`pgp_sym_decrypt(decode("%s", 'hex'), '%s')`, field, escapeSQLString(pg.Key)), nil
}

// Validate PGCrypto Validate
func (pg EncryptorPGCrypto) Validate(hash string, field string) bool {
	plain, err := pg.Decode(hash)
	if err != nil {
		return false
	}
	return plain == field
}
