package gou

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/utils"
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
	qbs := param.Query(nil, "")
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
	qbs := param.Query(nil, "")
	for _, qb := range qbs {
		rows := qb.MustGet()
		assert.Equal(t, len(rows), 1)
		for _, row := range rows {
			assert.Equal(t, row.Get("mobile"), "13900002222")
			assert.Equal(t, row.Get("type"), "staff")
		}
	}
}

func TestQueryHasOne(t *testing.T) {
	param := QueryParam{
		Model: "user",
		Withs: map[string]With{
			"manu": {
				Name: "manu",
				Query: QueryParam{
					Select: []interface{}{"name", "status", "short_name"},
				},
			},
		},
		Select: []interface{}{"name", "secret", "status", "type"},
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
	qbs := param.Query(nil, "")
	for _, qb := range qbs {
		utils.Dump(qb.ToSQL())
		rows := qb.MustGet()
		utils.Dump(rows)
	}
}
func TestQueryHasOneWhere(t *testing.T) {
	param := QueryParam{
		Model: "user",
		Withs: map[string]With{
			"manu": {
				Name: "manu",
				Query: QueryParam{
					Select: []interface{}{"name", "status", "short_name"},
					Wheres: []QueryWhere{
						{
							Column: "status",
							Method: "where",
							Value:  "disabled",
						}},
				},
			},
		},
		Select: []interface{}{"name", "secret", "status", "type"},
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
	qbs := param.Query(nil, "")
	for _, qb := range qbs {
		utils.Dump(qb.ToSQL())
		rows := qb.MustGet()
		utils.Dump(rows)
	}
}

func TestQueryHasOneRel(t *testing.T) {
	param := QueryParam{
		Model: "user",
		Withs: map[string]With{
			"manu": {
				Name: "manu",
				Query: QueryParam{
					Select: []interface{}{"name", "status", "short_name"},
				},
			},
		},
		Select: []interface{}{"name", "secret", "status", "type"},
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
				Rel:    "manu",
				Column: "short_name",
				Value:  "云道天成",
			},
		},
	}
	qbs := param.Query(nil, "")
	for _, qb := range qbs {
		utils.Dump(qb.ToSQL())
		rows := qb.MustGet()
		utils.Dump(rows)
	}
}

func TestQueryHasOneThrough(t *testing.T) {
	param := QueryParam{
		Model: "user",
		Withs: map[string]With{
			"mother": {Name: "mother"},
		},
		Select: []interface{}{"name", "secret", "status", "type", "id"},
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
			},
		},
	}
	qbs := param.Query(nil, "")
	for _, qb := range qbs {
		utils.Dump(qb.ToSQL())
		rows := qb.MustGet()
		utils.Dump(rows)
	}
}
