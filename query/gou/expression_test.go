package gou

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

func TestNewExpressionBase(t *testing.T) {
	var exps []Expression
	bytes := ReadFile("expressions/base.json")
	err := jsoniter.Unmarshal(bytes, &exps)
	assert.Nil(t, err)
	// utils.Dump(exps)
	// 需要验证数据
	for _, exp := range exps {
		// utils.Dump(exp.ToString())
		assert.True(t, len(exp.ToString()) > 0)
	}
}

func TestNewExpressionField(t *testing.T) {
	var exps []Expression
	bytes := ReadFile("expressions/fields.json")
	err := jsoniter.Unmarshal(bytes, &exps)
	assert.Nil(t, err)
	// 需要验证数据
	for _, exp := range exps {
		// utils.Dump(exp.ToString())
		assert.True(t, len(exp.ToString()) > 0)
	}
}

func TestNewExpressionType(t *testing.T) {
	var exps []Expression
	bytes := ReadFile("expressions/type.json")
	err := jsoniter.Unmarshal(bytes, &exps)
	assert.Nil(t, err)
	// 需要验证数据
	for _, exp := range exps {
		// utils.Dump(exp.ToString())
		assert.True(t, len(exp.ToString()) > 0)
	}
}
