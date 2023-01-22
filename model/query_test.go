package model

// func TestQueryWhere(t *testing.T) {
// 	param := QueryParam{
// 		Model: "user",
// 		Wheres: []QueryWhere{
// 			{
// 				Column: "mobile",
// 				Value:  "13900001111",
// 			},
// 			{
// 				Column: "type",
// 				Value:  "admin",
// 			},
// 		},
// 	}
// 	stack := NewQueryStack(param)
// 	stack.Run()

// }

// func TestQueryOrWhere(t *testing.T) {
// 	param := QueryParam{
// 		Model: "user",
// 		Wheres: []QueryWhere{
// 			{
// 				Column: "status",
// 				Value:  "enabled",
// 			},
// 			{
// 				Wheres: []QueryWhere{
// 					{
// 						Column: "type",
// 						Method: "where",
// 						Value:  "admin",
// 					},
// 					{
// 						Column: "type",
// 						Method: "orWhere",
// 						Value:  "staff",
// 					},
// 				},
// 			}, {
// 				Column: "mobile",
// 				Value:  "13900002222",
// 			},
// 		},
// 	}
// 	stack := NewQueryStack(param)
// 	stack.Run()
// }

// func TestQueryHasOne(t *testing.T) {
// 	param := QueryParam{
// 		Model: "user",
// 		Withs: map[string]With{
// 			"manu": {
// 				Name: "manu",
// 				Query: QueryParam{
// 					Select: []interface{}{"name", "status", "short_name"},
// 				},
// 			},
// 		},
// 		Select: []interface{}{"name", "secret", "status", "type"},
// 		Wheres: []QueryWhere{
// 			{
// 				Column: "status",
// 				Value:  "enabled",
// 			},
// 			{
// 				Wheres: []QueryWhere{
// 					{
// 						Column: "type",
// 						Method: "where",
// 						Value:  "admin",
// 					},
// 					{
// 						Column: "type",
// 						Method: "orWhere",
// 						Value:  "staff",
// 					},
// 				},
// 			}, {
// 				Column: "mobile",
// 				Value:  "13900002222",
// 			},
// 		},
// 	}
// 	stack := NewQueryStack(param)
// 	res := stack.Run()
// 	utils.Dump(res)
// }
// func TestQueryHasOneWhere(t *testing.T) {
// 	param := QueryParam{
// 		Model: "user",
// 		Withs: map[string]With{
// 			"manu": {
// 				Name: "manu",
// 				Query: QueryParam{
// 					Select: []interface{}{"name", "status", "short_name"},
// 					Wheres: []QueryWhere{
// 						{
// 							Column: "status",
// 							Method: "where",
// 							Value:  "disabled",
// 						}},
// 				},
// 			},
// 		},
// 		Select: []interface{}{"name", "secret", "status", "type"},
// 		Wheres: []QueryWhere{
// 			{
// 				Column: "status",
// 				Value:  "enabled",
// 			},
// 			{
// 				Wheres: []QueryWhere{
// 					{
// 						Column: "type",
// 						Method: "where",
// 						Value:  "admin",
// 					},
// 					{
// 						Column: "type",
// 						Method: "orWhere",
// 						Value:  "staff",
// 					},
// 				},
// 			}, {
// 				Column: "mobile",
// 				Value:  "13900002222",
// 			},
// 		},
// 	}
// 	stack := NewQueryStack(param)
// 	stack.Run()
// 	res := stack.Run()
// 	utils.Dump(res)
// }

// func TestQueryHasOneRel(t *testing.T) {
// 	param := QueryParam{
// 		Model: "user",
// 		Withs: map[string]With{
// 			"manu": {
// 				Name: "manu",
// 				Query: QueryParam{
// 					Select: []interface{}{"name", "status", "short_name"},
// 				},
// 			},
// 		},
// 		Select: []interface{}{"name", "secret", "status", "type"},
// 		Wheres: []QueryWhere{
// 			{
// 				Column: "status",
// 				Value:  "enabled",
// 			},
// 			{
// 				Wheres: []QueryWhere{
// 					{
// 						Column: "type",
// 						Method: "where",
// 						Value:  "admin",
// 					},
// 					{
// 						Column: "type",
// 						Method: "orWhere",
// 						Value:  "staff",
// 					},
// 				},
// 			}, {
// 				Rel:    "manu",
// 				Column: "short_name",
// 				Value:  "云道天成",
// 			},
// 		},
// 	}
// 	stack := NewQueryStack(param)
// 	res := stack.Run()
// 	utils.Dump(res)

// }

// func TestQueryHasOneThrough(t *testing.T) {
// 	param := QueryParam{
// 		Model: "user",
// 		Withs: map[string]With{
// 			"mother": {Name: "mother"},
// 		},
// 		Select: []interface{}{"name", "secret", "status", "type", "id"},
// 		Wheres: []QueryWhere{
// 			{
// 				Column: "status",
// 				Value:  "enabled",
// 			},
// 			{
// 				Wheres: []QueryWhere{
// 					{
// 						Column: "type",
// 						Method: "where",
// 						Value:  "admin",
// 					},
// 					{
// 						Column: "type",
// 						Method: "orWhere",
// 						Value:  "staff",
// 					},
// 				},
// 			},
// 		},
// 	}
// 	stack := NewQueryStack(param)
// 	res := stack.Run()
// 	utils.Dump(res)
// }

// func TestQueryHasOneThroughWhere(t *testing.T) {
// 	param := QueryParam{
// 		Model: "user",
// 		Withs: map[string]With{
// 			"mother": {},
// 		},
// 		Select: []interface{}{"name", "secret", "status", "type", "id"},
// 		Wheres: []QueryWhere{
// 			{
// 				Rel:    "mother.friends",
// 				Column: "status",
// 				Value:  "enabled",
// 			},
// 			{
// 				Column: "status",
// 				Value:  "enabled",
// 			},
// 			{
// 				Wheres: []QueryWhere{
// 					{
// 						Column: "type",
// 						Method: "where",
// 						Value:  "admin",
// 					},
// 					{
// 						Column: "type",
// 						Method: "orWhere",
// 						Value:  "staff",
// 					},
// 				},
// 			},
// 		},
// 	}
// 	stack := NewQueryStack(param)
// 	res := stack.Run()
// 	utils.Dump(res)
// }

// func TestQueryHasMany(t *testing.T) {
// 	param := QueryParam{
// 		Model: "user",
// 		Withs: map[string]With{
// 			"addresses": {
// 				Query: QueryParam{
// 					Select:   []interface{}{"province", "city", "location", "status"},
// 					PageSize: 20,
// 				},
// 			},
// 		},
// 		Select: []interface{}{"name", "secret", "status", "type", "extra"},
// 		Wheres: []QueryWhere{
// 			{
// 				Column: "status",
// 				Value:  "enabled",
// 			},
// 			{
// 				Wheres: []QueryWhere{
// 					{
// 						Column: "type",
// 						Method: "where",
// 						Value:  "admin",
// 					},
// 					{
// 						Column: "type",
// 						Method: "orWhere",
// 						Value:  "staff",
// 					},
// 				},
// 			}, {
// 				Column: "mobile",
// 				Value:  "13900002222",
// 			},
// 		},
// 	}
// 	stack := NewQueryStack(param)
// 	res := stack.Run()
// 	utils.Dump(res)

// }

// func TestQueryHasManyAndOne(t *testing.T) {
// 	param := QueryParam{
// 		Model: "user",
// 		Withs: map[string]With{
// 			"manu": {
// 				Query: QueryParam{
// 					Select: []interface{}{"name", "status", "short_name"},
// 				},
// 			},
// 			"addresses": {
// 				Query: QueryParam{
// 					Select:   []interface{}{"province", "city", "location", "status"},
// 					PageSize: 20,
// 				},
// 			},
// 		},
// 		Select: []interface{}{"name", "secret", "status", "type", "extra"},
// 		Wheres: []QueryWhere{
// 			{
// 				Column: "status",
// 				Value:  "enabled",
// 			},
// 			{
// 				Wheres: []QueryWhere{
// 					{
// 						Column: "type",
// 						Method: "where",
// 						Value:  "admin",
// 					},
// 					{
// 						Column: "type",
// 						Method: "orWhere",
// 						Value:  "staff",
// 					},
// 				},
// 			}, {
// 				Column: "mobile",
// 				Value:  "13900002222",
// 			},
// 		},
// 	}
// 	stack := NewQueryStack(param)
// 	res := stack.Run()
// 	utils.Dump(res)
// }

// func TestQueryHasManyAndOnePaginate(t *testing.T) {
// 	param := QueryParam{
// 		Model: "user",
// 		Withs: map[string]With{
// 			"manu": {
// 				Query: QueryParam{
// 					Select: []interface{}{"name", "status", "short_name"},
// 				},
// 			},
// 			"addresses": {
// 				Query: QueryParam{
// 					Select:   []interface{}{"province", "city", "location", "status"},
// 					PageSize: 20,
// 				},
// 			},
// 		},
// 		Select: []interface{}{"name", "secret", "status", "type", "extra"},
// 		Wheres: []QueryWhere{
// 			{
// 				Column: "status",
// 				Value:  "enabled",
// 			},
// 		},
// 	}
// 	stack := NewQueryStack(param)
// 	res := stack.Paginate(1, 2)
// 	utils.Dump(res)
// }

// func TestQueryHasManyAndOnePaginateOrder(t *testing.T) {
// 	param := QueryParam{
// 		Model: "user",
// 		Withs: map[string]With{
// 			"manu": {
// 				Query: QueryParam{
// 					Select: []interface{}{"name", "status", "short_name"},
// 				},
// 			},
// 			"addresses": {
// 				Query: QueryParam{
// 					Select:   []interface{}{"province", "city", "location", "status"},
// 					PageSize: 20,
// 				},
// 			},
// 		},
// 		Select: []interface{}{"name", "secret", "status", "type", "extra"},
// 		Orders: []QueryOrder{
// 			{
// 				Column: "id",
// 				Option: "desc",
// 			},
// 		},
// 		Wheres: []QueryWhere{
// 			{
// 				Column: "status",
// 				Value:  "enabled",
// 			},
// 		},
// 	}
// 	stack := NewQueryStack(param)
// 	res := stack.Paginate(1, 2)
// 	utils.Dump(res)
// }
