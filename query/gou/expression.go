package gou

import (
	"strings"

	"github.com/go-errors/errors"
)

// UnmarshalJSON for json marshalJSON
func (exp *Expression) UnmarshalJSON(data []byte) error {
	exp.string = string(data)
	return nil
}

// MarshalJSON for json marshalJSON
func (exp *Expression) MarshalJSON() ([]byte, error) {
	return []byte(exp.string), nil
}

// NewExpression 创建一个表达式
func NewExpression(s string) *Expression {
	return &Expression{
		string: s,
	}
}

// ToString for json marshalJSON
func (exp Expression) ToString() string {
	return exp.string
}

// Validate 校验表达式格式
func (exp Expression) Validate() error {
	if strings.Contains(exp.string, " ") {
		return errors.Errorf("字段表达式格式不正确(%s)", exp.string)
	}
	return nil
}
