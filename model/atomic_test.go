package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/maps"
)

func TestSave(t *testing.T) {
	prepare(t)
	defer clean()
	pet := Select("pet")
	id, err := pet.Save(maps.Map{"name": "Cookie"})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, any.Of(id).CInt())
}
