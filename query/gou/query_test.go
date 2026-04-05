package gou

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/query/share"
)

func TestGet(t *testing.T) {
	q := Open(GetFileName("queries/test_get.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	q.Build()
	q.STMT = q.ToSQL()
	q.Bindings = q.GetBindings()
	q.Selects = q.mapOfSelect()

	rows := q.Get(nil)
	assert.Equal(t, 3, len(rows))
	assert.Equal(t, "Alice", rows[0]["name"])
	assert.Equal(t, "alice@test.com", rows[0]["email"])
	assert.Equal(t, "Bob", rows[1]["name"])
	assert.Equal(t, "Charlie", rows[2]["name"])
}

func TestFirst(t *testing.T) {
	q := Open(GetFileName("queries/test_first.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	q.Build()
	q.STMT = q.ToSQL()
	q.Bindings = q.GetBindings()
	q.Selects = q.mapOfSelect()

	row := q.First(nil)
	assert.NotNil(t, row)
	assert.Equal(t, "Alice", row["name"])
	assert.Equal(t, "alice@test.com", row["email"])
}

func TestPaginate(t *testing.T) {
	q := Open(GetFileName("queries/test_paginate.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	q.Build()
	q.STMT = q.ToSQL()
	q.Bindings = q.GetBindings()
	q.Selects = q.mapOfSelect()

	res := q.Paginate(nil)
	assert.Equal(t, 3, res.Total)
	assert.Equal(t, 1, res.Page)
	assert.Equal(t, 2, res.PageSize)
	assert.Equal(t, 2, res.PageCount)
	assert.Equal(t, 2, res.Next)
	assert.Equal(t, -1, res.Prev)
	assert.Equal(t, 2, len(res.Items))
	assert.Equal(t, "Alice", res.Items[0]["name"])
	assert.Equal(t, "Bob", res.Items[1]["name"])
}

func TestRun(t *testing.T) {
	q := Open(GetFileName("queries/test_get.json")).
		With(qb, TableName).
		SetAESKey(TestAESKey)
	q.Build()
	q.STMT = q.ToSQL()
	q.Bindings = q.GetBindings()
	q.Selects = q.mapOfSelect()

	result := q.Run(nil)
	rows, ok := result.([]share.Record)
	assert.True(t, ok)
	assert.Equal(t, 3, len(rows))
	assert.Equal(t, "Alice", rows[0]["name"])
}
