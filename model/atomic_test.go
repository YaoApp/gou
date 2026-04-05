package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/maps"
)

func TestQuoteIdentifier(t *testing.T) {
	prepare(t)
	defer clean()
	user := Select("user")

	// Simple column name
	q := user.QuoteIdentifier("name")
	switch user.Driver {
	case "postgres":
		assert.Equal(t, `"name"`, q)
	default:
		assert.Equal(t, "`name`", q)
	}

	// Dotted name (alias.column)
	q = user.QuoteIdentifier("t.name")
	switch user.Driver {
	case "postgres":
		assert.Equal(t, `"t"."name"`, q)
	default:
		assert.Equal(t, "`t`.`name`", q)
	}
}

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
