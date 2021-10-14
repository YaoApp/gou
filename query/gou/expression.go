package gou

import (
	"strings"

	"github.com/go-errors/errors"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
)

// UnmarshalJSON for json marshalJSON
func (exp *Expression) UnmarshalJSON(data []byte) error {
	var v string
	err := jsoniter.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	*exp = *NewExpression(v)
	return nil
}

// // MarshalJSON for json marshalJSON
// func (exp *Expression) MarshalJSON() ([]byte, error) {
// 	return []byte(exp.string), nil
// }

// NewExpression 创建一个表达式
func NewExpression(s string) *Expression {
	exp, err := MakeExpression(s)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return &exp
}

// MakeExpression 解析表达式
func MakeExpression(s string) (Expression, error) {
	if s == "" {
		return Expression{}, errors.Errorf("字段表达式不能为空")
	}

	s = strings.TrimSpace(s)
	exp := Expression{
		Origin: s, Field: s,
	}

	// 别名
	names := RegAlias.Split(exp.Field, -1)
	if len(names) == 2 {
		exp.Field = strings.TrimSpace(names[0])
		exp.Alias = strings.TrimSpace(names[1])
	}

	// 解析字段
	err := exp.parseExpField(exp.Field)
	return exp, err
}

// parseExpField 解析字段
func (exp *Expression) parseExpField(s string) error {

	// 函数
	if strings.HasPrefix(s, ":") {
		if err := exp.parseExpFunc(); err != nil {
			return err
		}
		return nil
	}

	// 绑定动态数值
	if strings.HasPrefix(s, "?:") {
		if err := exp.parseExpBindings(); err != nil {
			return err
		}
		return nil
	}

	// 表格、模型
	exp.parseExpTable()

	// 数组字段
	if RegFieldIsArray.MatchString(exp.Field) {
		if err := exp.parseExpArray(); err != nil {
			return err
		}
		return nil
	}

	// 对象字段
	if strings.Contains(exp.Field, "$") {
		exp.parseExpObject()
		return nil
	}

	// 加密字段
	if strings.HasSuffix(exp.Field, "*") {
		exp.Field = strings.TrimSuffix(exp.Field, "*")
		exp.IsAES = true
		return nil
	}

	// 数字常量
	if RegIsNumber.MatchString(exp.Field) {
		exp.parseExpNumber()
		return nil
	}

	// 字符串常量
	if strings.HasPrefix(exp.Field, "'") && strings.HasSuffix(exp.Field, "'") {
		exp.Value = strings.Trim(exp.Field, "'")
		exp.Field = ""
		return nil
	}

	return nil
}

// parseExpTable 解析表格、模型
func (exp *Expression) parseExpTable() error {
	matches := RegFieldTable.FindStringSubmatch(exp.Field)
	if matches != nil {
		exp.Table = matches[1]
		exp.IsModel = strings.HasPrefix(exp.Field, "$")
		exp.Field = strings.TrimPrefix(exp.Field, matches[0])
	}
	return nil
}

// parseExpNumber 解析字段, 数字常量
func (exp *Expression) parseExpNumber() error {
	v := any.Of(exp.Field)
	exp.Field = ""
	if strings.Contains(exp.Field, ".") {
		exp.Value = v.CFloat64()
		return nil
	}
	exp.Value = v.CInt()
	return nil
}

// parseExpObject 解析字段, 对象
func (exp *Expression) parseExpObject() error {
	names := strings.Split(exp.Field, ".")
	exp.Field = strings.TrimSuffix(names[0], "$")
	exp.IsObject = true
	if len(names) > 1 {
		exp.Key = strings.Join(names[1:], ".")
	}
	return nil
}

// parseExpArray 解析字段, 数组
func (exp *Expression) parseExpArray() error {
	names := strings.Split(exp.Field, ".")
	exp.Field = strings.ReplaceAll(names[0], "@", "")
	exp.IsArray = true

	if matches := RegArrayIndex.FindStringSubmatch(exp.Field); matches != nil {
		exp.Field = strings.TrimSpace(strings.ReplaceAll(exp.Field, matches[0], ""))
		if matches[1] == "*" {
			exp.Index = Star
		} else {
			exp.Index = any.Of(matches[1]).CInt()
		}
	}

	if len(names) > 1 {
		exp.Key = strings.Join(names[1:], ".")
		exp.IsArrayObject = RegFieldIsArrayObject.MatchString(exp.Key)
	}
	return nil
}

// parseFunc 解析函数
func (exp *Expression) parseExpFunc() error {
	matches := RegFieldFun.FindStringSubmatch(exp.Field)
	if matches == nil {
		return errors.Errorf("字段表达式函数格式不正确(%s)", exp.Field)
	}
	exp.FunName = matches[1]
	exp.FunArgs = []Expression{}
	args := strings.Split(matches[2], ",")
	for _, arg := range args {
		argexp, err := MakeExpression(arg)
		if err != nil {
			return errors.Errorf(" %s 参数错误: %s", exp.Field, err.Error())
		}
		exp.FunArgs = append(exp.FunArgs, argexp)
	}
	exp.Field = ""
	return nil
}

// parseExpBindings 解析动态数值
func (exp *Expression) parseExpBindings() error {
	exp.Field = strings.TrimPrefix(exp.Field, "?:")
	exp.IsBinding = true
	return nil
}

// ToString for json marshalJSON
func (exp Expression) ToString() string {
	return exp.Field
}

// Validate 校验表达式格式
func (exp Expression) Validate() error {
	if strings.Contains(exp.Field, " ") {
		return errors.Errorf("字段表达式格式不正确(%s)", exp.Field)
	}
	return nil
}
