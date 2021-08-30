package gou

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryWhere(t *testing.T) {
	param := QueryParam{
		Model: "user",
		Wheres: []QueryWhere{
			{
				Column: "mobile",
				Value:  "13900001111",
			},
			{
				Column: "type",
				Value:  "admin",
			},
		},
	}
	qbs := param.Query(nil)
	for _, qb := range qbs {
		rows := qb.MustGet()
		assert.Equal(t, len(rows), 1)
		for _, row := range rows {
			assert.Equal(t, row.Get("mobile"), "13900001111")
			assert.Equal(t, row.Get("type"), "admin")
		}
	}
}

func TestQueryOrWhere(t *testing.T) {
	param := QueryParam{
		Model: "user",
		Wheres: []QueryWhere{
			{
				Column: "status",
				Value:  "enabled",
			},
			{
				Wheres: []QueryWhere{
					{
						Column: "type",
						Method: "where",
						Value:  "admin",
					},
					{
						Column: "type",
						Method: "orWhere",
						Value:  "staff",
					},
				},
			}, {
				Column: "mobile",
				Value:  "13900002222",
			},
		},
	}
	qbs := param.Query(nil)
	for _, qb := range qbs {
		rows := qb.MustGet()
		assert.Equal(t, len(rows), 1)
		for _, row := range rows {
			assert.Equal(t, row.Get("mobile"), "13900002222")
			assert.Equal(t, row.Get("type"), "staff")
		}
	}
}
