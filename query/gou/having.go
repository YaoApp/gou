package gou

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xun/dbal/query"
)

type havingArgs struct {
	Method string
	OR     bool
	Field  interface{}
	Args   []interface{}
}

// UnmarshalJSON for json marshalJSON
func (having *Having) UnmarshalJSON(data []byte) error {
	origin := map[string]interface{}{}
	err := jsoniter.Unmarshal(data, &origin)
	if err != nil {
		return err
	}
	*having = HavingOf(origin)
	return nil
}

// MarshalJSON for json marshalJSON
func (having Having) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(having.ToMap())
}

// HavingOf 从 maps 载入 having
func HavingOf(input map[string]interface{}) Having {
	having := Having{}
	having.Condition = ConditionOf(input)

	// Havings
	if havings, has := input["havings"]; has {
		if havings, ok := havings.([]interface{}); ok {
			having.Havings = []Having{}
			for i := range havings {
				if w, ok := havings[i].(map[string]interface{}); ok {
					having.Havings = append(having.Havings, HavingOf(w))
				}
			}
		}
	}
	return having
}

// ToMap 转换为 map[string]interface{}
func (having Having) ToMap() map[string]interface{} {
	res := having.Condition.ToMap()
	if len(having.Havings) > 0 {
		havings := []map[string]interface{}{}
		for _, w := range having.Havings {
			havings = append(havings, w.ToMap())
		}
		res["havings"] = havings
	}
	return res
}

func (gou *Query) parseHavingArgs(having Having) havingArgs {

	var field interface{} = nil
	if having.Field != nil {
		field = gou.sqlExpression(*having.Field)
	}
	value := having.Value
	mehtod := "having"

	// 下一版支持全文检索
	if having.OP == "match" {
		if v, ok := value.(string); ok {
			value = "%" + v + "%"
			having.OP = "like"
		}
	} else if having.OP == "is" {
		exception.New("is 操作暂不支持", 400).Throw()
		mehtod = "havingNull"
		if v, ok := having.Value.(string); ok && v == "not null" {
			mehtod = "havingNotNull"
		}
	} else if having.OP == "in" {
		exception.New("in 操作暂不支持", 400).Throw()
		mehtod = "havingIn"
	}

	// 数值表达式
	if having.ValueExpression != nil {
		value = gou.sqlExpression(*having.ValueExpression)
	}

	// 子查询
	if having.Query != nil {
		gouSub := New()
		gouSub.QueryDSL = *having.Query
		value = func(qb query.Query) {
			gouSub.Query = qb
			gouSub.Build()
		}
	}

	// 分组查询
	if having.Havings != nil {
		mehtod = "havings"
		havings := [][]interface{}{}
		for i := range having.Havings {
			w := gou.parseHavingArgs(having.Havings[i])
			args := []interface{}{w.Field}
			args = append(args, w.Args...)
			havings = append(havings, args)
		}
		field = havings
	}

	return havingArgs{
		Method: mehtod,
		OR:     having.OR,
		Field:  field,
		Args:   []interface{}{having.OP, value},
	}
}

func (gou *Query) setHaving(or bool, field interface{}, args ...interface{}) {
	if field == nil {
		return
	}
	if or {
		gou.Query.OrHaving(field, args...)
		return
	}
	gou.Query.Having(field, args...)
}

// func (gou *Query) setHavingIn(or bool, field interface{}, value interface{}) {
// 	if field == nil {
// 		return
// 	}
// 	if or {
// 		gou.Query.OrHavingIn(field, value)
// 		return
// 	}

// 	gou.Query.HavingIn(field, value)
// }

// func (gou *Query) setHavingNull(or bool, field interface{}) {
// 	if or {
// 		gou.Query.OrHavingNull(field)
// 		return
// 	}
// 	gou.Query.HavingNull(field)
// }

// func (gou *Query) setHavingNotNull(or bool, field interface{}) {
// 	if or {
// 		gou.Query.OrHavingNotNull(field)
// 		return
// 	}
// 	gou.Query.HavingNotNull(field)
// }
