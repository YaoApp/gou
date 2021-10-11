package gou

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

func TestTableBase(t *testing.T) {
	var tables []Table
	bytes := ReadFile("tables/base.json")
	err := jsoniter.Unmarshal(bytes, &tables)
	assert.Nil(t, err)
	assert.Equal(t, 7, len(tables))

	// "name",
	assert.Equal(t, "name", tables[0].Name)
	assert.Equal(t, "", tables[0].Alias)
	assert.False(t, tables[0].IsModel)
	assert.Nil(t, tables[0].Validate())

	// "$user",
	assert.Equal(t, "user", tables[1].Name)
	assert.Equal(t, "", tables[1].Alias)
	assert.True(t, tables[1].IsModel)
	assert.Nil(t, tables[1].Validate())

	// "name as n",
	assert.Equal(t, "name", tables[2].Name)
	assert.Equal(t, "n", tables[2].Alias)
	assert.False(t, tables[2].IsModel)
	assert.Nil(t, tables[2].Validate())

	// "$user as u",
	assert.Equal(t, "user", tables[3].Name)
	assert.Equal(t, "u", tables[3].Alias)
	assert.True(t, tables[3].IsModel)
	assert.Nil(t, tables[3].Validate())

	// "name   as   bar",
	assert.Equal(t, "name", tables[4].Name)
	assert.Equal(t, "bar", tables[4].Alias)
	assert.False(t, tables[4].IsModel)
	assert.Nil(t, tables[4].Validate())

	// "$user  as    foo"
	assert.Equal(t, "user", tables[5].Name)
	assert.Equal(t, "foo", tables[5].Alias)
	assert.True(t, tables[5].IsModel)
	assert.Nil(t, tables[5].Validate())

	// "$xiang.user as 用户_表"
	assert.Equal(t, "xiang.user", tables[6].Name)
	assert.Equal(t, "用户_表", tables[6].Alias)
	assert.True(t, tables[6].IsModel)
	assert.Nil(t, tables[6].Validate())
}

func TestTableValidate(t *testing.T) {
	var errs []Table
	bytes := ReadFile("tables/error.json")

	err := jsoniter.Unmarshal(bytes, &errs)
	assert.Nil(t, err)
	assert.Equal(t, 5, len(errs))

	// "name.d",
	assert.Error(t, errs[0].Validate())

	// "$name中文",
	assert.Error(t, errs[1].Validate())

	// "name as ,",
	assert.Error(t, errs[2].Validate())

	// "$model.user as |",
	assert.Error(t, errs[3].Validate())

	// "$model.user as foo bar"
	assert.Error(t, errs[4].Validate())
}
