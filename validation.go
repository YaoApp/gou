package gou

import (
	"fmt"
	"net/mail"
	"regexp"
	"time"

	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/kun/str"
)

// Validations 数据校验函数
var Validations = map[string]func(value interface{}, row maps.MapStrAny, args ...interface{}) bool{
	"typof":     ValidationTypeof,    // 校验数值类型 string, integer, float, number, datetime, timestamp,
	"min":       ValidationMin,       // 最小值
	"max":       ValidationMax,       // 最大值
	"enum":      ValidationEnum,      // 枚举型
	"pattern":   ValidationPattern,   // 正则匹配
	"minLength": ValidationMinLength, // 最小长度
	"maxLength": ValidationMaxLength, // 最大长度
	"email":     ValidationEmail,     // 邮箱地址
	"mobile":    ValidationMobile,    // 手机号
}

// ValidationTypeof 校验数值类型
func ValidationTypeof(value interface{}, _ maps.MapStrAny, args ...interface{}) bool {

	if len(args) < 1 {
		return false
	}

	typ := str.Of(args[0])

	switch typ {
	case "string":
		_, ok := value.(string)
		return ok

	case "integer":
		return any.Of(value).IsInt()

	case "float":
		return any.Of(value).IsFloat()

	case "number":
		return any.Of(value).IsNumber()

	case "datetime":
		isDatetime := false
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
		}
		valueStr := fmt.Sprintf("%v", value)
		for _, format := range formats {
			_, err := time.Parse(format, valueStr)
			if err == nil {
				isDatetime = true
			}
		}
		return isDatetime

	case "bool":
		v := any.Of(value)
		if v.IsInt() {
			return v.Int() == 1 || v.Int() == 0
		}
		return v.IsBool()
	}

	return false
}

// ValidationMin 最小值
func ValidationMin(value interface{}, _ maps.MapStrAny, args ...interface{}) bool {
	if len(args) < 1 {
		return true
	}
	v := any.Of(value)
	if v.IsInt() {
		return v.Int() >= any.Of(args[0]).CInt()
	} else if v.IsFloat() {
		return v.Float() >= any.Of(args[0]).CFloat()
	}
	return false
}

// ValidationMax 最大值
func ValidationMax(value interface{}, _ maps.MapStrAny, args ...interface{}) bool {
	if len(args) < 1 {
		return true
	}
	v := any.Of(value)
	if v.IsInt() {
		return v.Int() <= any.Of(args[0]).CInt()
	} else if v.IsFloat() {
		return v.Float() <= any.Of(args[0]).CFloat()
	}
	return false
}

// ValidationEnum 枚举型
func ValidationEnum(value interface{}, _ maps.MapStrAny, args ...interface{}) bool {
	if len(args) < 1 {
		return true
	}

	for _, arg := range args {
		if arg == value {
			return true
		}
	}

	return false
}

// ValidationPattern 正则匹配
func ValidationPattern(value interface{}, _ maps.MapStrAny, args ...interface{}) bool {
	if len(args) < 1 {
		return true
	}

	re, err := regexp.Compile(string(str.Of(args[0])))
	if err != nil {
		return false
	}
	return re.Match([]byte(str.Of(value)))
}

// ValidationMinLength 最小长度
func ValidationMinLength(value interface{}, _ maps.MapStrAny, args ...interface{}) bool {
	if len(args) < 1 {
		return true
	}
	return str.Of(value).Length() >= any.Of(args[0]).CInt()
}

// ValidationMaxLength 最大长度
func ValidationMaxLength(value interface{}, _ maps.MapStrAny, args ...interface{}) bool {
	if len(args) < 1 {
		return true
	}
	return str.Of(value).Length() <= any.Of(args[0]).CInt()
}

// ValidationEmail 验证邮箱
func ValidationEmail(value interface{}, _ maps.MapStrAny, _ ...interface{}) bool {
	v := any.Of(value)
	if !v.IsString() {
		return false
	}
	_, err := mail.ParseAddress(v.String())
	return err == nil
}

// ValidationMobile 验证手机号
func ValidationMobile(value interface{}, _ maps.MapStrAny, args ...interface{}) bool {

	v := any.Of(value)
	if !v.IsString() {
		return false
	}

	zone := "cn"
	if len(args) > 0 {
		zone = any.Of(args[0]).String()
	}
	reg := regexp.MustCompile("^1[3-9]\\d{9}$")
	switch zone {
	case "us":
		reg = regexp.MustCompile(`^[0-9]{3}-[0-9]{3}-[0-9]{4}$`)
	}
	return reg.MatchString(v.String())
}
