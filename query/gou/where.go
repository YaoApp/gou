package gou

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/xun/dbal/query"
)

type whereArgs struct {
	Method string
	OR     bool
	Field  interface{}
	Args   []interface{}
}

// UnmarshalJSON for json marshalJSON
func (where *Where) UnmarshalJSON(data []byte) error {
	origin := map[string]interface{}{}
	err := jsoniter.Unmarshal(data, &origin)
	if err != nil {
		return err
	}
	*where = WhereOf(origin)
	return nil
}

// MarshalJSON for json marshalJSON
func (where Where) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(where.ToMap())
}

// WhereOf 从 maps 载入 where
func WhereOf(input map[string]interface{}) Where {
	where := Where{}
	where.Condition = ConditionOf(input)

	// Wheres
	if wheres, has := input["wheres"]; has {
		if wheres, ok := wheres.([]interface{}); ok {
			where.Wheres = []Where{}
			for i := range wheres {
				if w, ok := wheres[i].(map[string]interface{}); ok {
					where.Wheres = append(where.Wheres, WhereOf(w))
				}
			}
		}
	}
	return where
}

// ToMap 转换为 map[string]interface{}
func (where Where) ToMap() map[string]interface{} {
	res := where.Condition.ToMap()
	if len(where.Wheres) > 0 {
		wheres := []map[string]interface{}{}
		for _, w := range where.Wheres {
			wheres = append(wheres, w.ToMap())
		}
		res["wheres"] = wheres
	}
	return res
}

func (gou *Query) parseWhereArgs(where Where) whereArgs {

	var field interface{} = nil
	if where.Field != nil {
		field = gou.sqlExpression(*where.Field)
	}
	value := where.Value
	mehtod := "where"

	// 下一版支持全文检索
	if where.OP == "match" {
		if v, ok := value.(string); ok {
			value = "%" + v + "%"
			where.OP = "like"
		}
	} else if where.OP == "is" {
		mehtod = "whereNull"
		if v, ok := where.Value.(string); ok && v == "not null" {
			mehtod = "whereNotNull"
		}
	} else if where.OP == "in" {
		mehtod = "whereIn"
	}

	// 数值表达式
	if where.ValueExpression != nil {
		value = gou.sqlExpression(*where.ValueExpression)
		if mehtod == "where" {
			mehtod = "whereColumn"
		}
	}

	// 子查询
	if where.Query != nil {
		gouSub := gou.Clone()
		gouSub.QueryDSL = *where.Query
		value = func(qb query.Query) {
			gouSub.Query = qb
			gouSub.Build()
		}
	}

	// 分组查询
	if where.Wheres != nil {
		mehtod = "wheres"
		field = where.Wheres
	}

	return whereArgs{
		Method: mehtod,
		OR:     where.OR,
		Field:  field,
		Args:   []interface{}{where.OP, value},
	}
}

func (gou *Query) setWheres(or bool, wheres []Where) {
	if wheres == nil {
		return
	}

	var whereFun = func(qb query.Query) {
		gouWheres := New()
		gouWheres.Query = qb
		for _, where := range wheres {
			gouWheres.buildWhere(where)
		}
	}

	if or {
		gou.Query.OrWhere(whereFun)
		return
	}

	gou.Query.Where(whereFun)

}

func (gou *Query) setWhere(or bool, field interface{}, args ...interface{}) {
	if field == nil {
		return
	}
	if or {
		gou.Query.OrWhere(field, args...)
		return
	}

	gou.Query.Where(field, args...)
}

func (gou *Query) setWhereColumn(or bool, field interface{}, args ...interface{}) {
	if field == nil {
		return
	}
	if or {
		gou.Query.OrWhereColumn(field, args...)
		return
	}

	gou.Query.WhereColumn(field, args...)
}

func (gou *Query) setWhereIn(or bool, field interface{}, value interface{}) {
	if field == nil {
		return
	}
	if or {
		gou.Query.OrWhereIn(field, value)
		return
	}

	gou.Query.WhereIn(field, value)
}

func (gou *Query) setWhereNull(or bool, field interface{}) {
	if or {
		gou.Query.OrWhereNull(field)
		return
	}
	gou.Query.WhereNull(field)
}

func (gou *Query) setWhereNotNull(or bool, field interface{}) {
	if or {
		gou.Query.OrWhereNotNull(field)
		return
	}
	gou.Query.WhereNotNull(field)
}
