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
}

func TestNewExpressionField(t *testing.T) {
	var exps []Expression
	bytes := ReadFile("expressions/fields.json")
	err := jsoniter.Unmarshal(bytes, &exps)
	assert.Nil(t, err)
}
