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
		Model: "pet",
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
		Model: "pet",
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
		Model: "pet",
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
		Model: "pet",
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
		Model: "pet",
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
