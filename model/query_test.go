package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryWhere(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	param := QueryParam{
		Model: "user",
		Wheres: []QueryWhere{
			{Column: "name", Value: "John Doe"},
		},
	}
	stack := NewQueryStack(param)
	res := stack.Run()
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "John Doe", res[0].Get("name"))
}

func TestQueryOrWhere(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	param := QueryParam{
		Model: "user",
		Wheres: []QueryWhere{
			{
				Wheres: []QueryWhere{
					{Column: "name", Method: "where", Value: "John Doe"},
					{Column: "name", Method: "orWhere", Value: "Lucy Queen"},
				},
			},
		},
	}
	stack := NewQueryStack(param)
	res := stack.Run()
	assert.Equal(t, 2, len(res))
}

func TestQueryHasOne(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	param := QueryParam{
		Model:  "pet",
		Select: []interface{}{"name", "category_id", "owner_id"},
		Withs: map[string]With{
			"category": {
				Query: QueryParam{
					Select: []interface{}{"id", "name"},
				},
			},
		},
		Wheres: []QueryWhere{
			{Column: "name", Value: "Tommy"},
		},
	}
	stack := NewQueryStack(param)
	res := stack.Run()
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "Tommy", res[0].Get("name"))
	assert.Equal(t, "Dog", res[0].Dot().Get("category.name"))
}

func TestQueryHasOneWhere(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	param := QueryParam{
		Model:  "pet",
		Select: []interface{}{"name", "owner_id"},
		Withs: map[string]With{
			"owner": {
				Query: QueryParam{
					Select: []interface{}{"id", "name", "email"},
				},
			},
		},
		Wheres: []QueryWhere{
			{Column: "owner_id", Value: 1},
		},
	}
	stack := NewQueryStack(param)
	res := stack.Run()
	assert.Equal(t, 2, len(res))
	for _, row := range res {
		assert.Equal(t, "John Doe", row.Dot().Get("owner.name"))
	}
}

func TestQueryHasOneMultipleRelations(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	param := QueryParam{
		Model:  "pet",
		Select: []interface{}{"name", "category_id", "owner_id", "doctor_id"},
		Withs: map[string]With{
			"category": {
				Query: QueryParam{Select: []interface{}{"id", "name"}},
			},
			"owner": {
				Query: QueryParam{Select: []interface{}{"id", "name"}},
			},
			"doctor": {
				Query: QueryParam{Select: []interface{}{"id", "name"}},
			},
		},
		Wheres: []QueryWhere{
			{Column: "name", Value: "Tommy"},
		},
	}
	stack := NewQueryStack(param)
	res := stack.Run()
	assert.Equal(t, 1, len(res))
	dot := res[0].Dot()
	assert.Equal(t, "Dog", dot.Get("category.name"))
	assert.Equal(t, "John Doe", dot.Get("owner.name"))
	assert.Equal(t, "Lucy Queen", dot.Get("doctor.name"))
}

func TestQueryPaginate(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	param := QueryParam{
		Model:  "pet",
		Select: []interface{}{"name", "category_id"},
		Withs: map[string]With{
			"category": {
				Query: QueryParam{Select: []interface{}{"id", "name"}},
			},
		},
	}
	stack := NewQueryStack(param)
	res := stack.Paginate(1, 2)
	dot := res.Dot()
	assert.Equal(t, 4, dot.Get("total"))
	assert.Equal(t, 1, dot.Get("page"))
}

func TestQueryPaginateOrder(t *testing.T) {
	prepare(t)
	defer clean()
	prepareTestData(t)

	param := QueryParam{
		Model:  "pet",
		Select: []interface{}{"name"},
		Orders: []QueryOrder{
			{Column: "id", Option: "desc"},
		},
	}
	stack := NewQueryStack(param)
	res := stack.Paginate(1, 2)
	dot := res.Dot()
	assert.Equal(t, 4, dot.Get("total"))
	assert.Equal(t, "Nemo", dot.Get("data.0.name"))
}

func TestQueryJSONColumnLike(t *testing.T) {
	prepare(t)
	defer clean()

	user := Select("user")

	_, err := user.Create(map[string]interface{}{
		"name":   "JSONLikeTest",
		"mobile": "13900000001",
		"type":   "admin",
		"roles":  []string{"admin", "editor"},
	})
	assert.NoError(t, err)

	// like OP on JSON array column → WhereJSONContains
	res, err := user.Get(QueryParam{
		Wheres: []QueryWhere{
			{Column: "roles", Value: "%admin%", OP: "like"},
		},
	})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(res), 1)

	// match OP on JSON array column → WhereJSONContains
	res, err = user.Get(QueryParam{
		Wheres: []QueryWhere{
			{Column: "roles", Value: "admin", OP: "match"},
		},
	})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(res), 1)
}

func TestQueryJSONColumnLikeOrWhere(t *testing.T) {
	prepare(t)
	defer clean()

	user := Select("user")
	_, err := user.Create(map[string]interface{}{
		"name":   "JSONOrWhereTest",
		"mobile": "13900000002",
		"type":   "admin",
		"roles":  []string{"tester"},
	})
	assert.NoError(t, err)

	// Nested where with OR on JSON array column
	res, err := user.Get(QueryParam{
		Wheres: []QueryWhere{
			{
				Wheres: []QueryWhere{
					{Column: "name", Value: "nonexistent_user_xyz"},
					{Column: "roles", Value: "%tester%", OP: "like", Method: "orwhere"},
				},
			},
		},
	})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(res), 1)
}

func TestQueryJSONColumnMatchNonJSON(t *testing.T) {
	prepare(t)
	defer clean()

	user := Select("user")
	_, err := user.Create(map[string]interface{}{
		"name":   "MatchNonJSONTest",
		"mobile": "13900000003",
		"type":   "admin",
	})
	assert.NoError(t, err)

	// match OP on non-JSON column uses regular LIKE
	res, err := user.Get(QueryParam{
		Wheres: []QueryWhere{
			{Column: "name", Value: "MatchNonJSON", OP: "match"},
		},
	})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(res), 1)
}

func TestQueryJSONContainsMultipleConditions(t *testing.T) {
	prepare(t)
	defer clean()

	user := Select("user")
	_, err := user.Create(map[string]interface{}{
		"name":   "JSONMultiTest",
		"mobile": "13900000004",
		"type":   "admin",
		"roles":  []string{"admin", "editor"},
	})
	assert.NoError(t, err)

	// Multiple JSON column conditions combined
	res, err := user.Get(QueryParam{
		Wheres: []QueryWhere{
			{Column: "roles", Value: "admin", OP: "match"},
			{Column: "name", Value: "JSONMultiTest"},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(res))
}
