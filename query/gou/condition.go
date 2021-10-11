package gou

import (
	"strings"

	"github.com/go-errors/errors"
	jsoniter "github.com/json-iterator/go"
)

func condShouldHaveValue(i int, cond Condition) error {
	if cond.Value == nil && cond.Query == nil {
		return errors.Errorf("缺少 value 或 query")
	}
	return nil
}

func condIgnoreValue(i int, cond Condition) error {
	return nil
}

// OPs 可用的操作符
var OPs = map[string]func(i int, where Condition) error{
	"=":     condShouldHaveValue,
	">":     condShouldHaveValue,
	">=":    condShouldHaveValue,
	"<":     condShouldHaveValue,
	"<=":    condShouldHaveValue,
	"like":  condShouldHaveValue,
	"match": condShouldHaveValue,
	"in":    condShouldHaveValue,
	"is":    condShouldHaveValue,
}

// UnmarshalJSON for json marshalJSON
func (cond *Condition) UnmarshalJSON(data []byte) error {
	origin := map[string]interface{}{}
	err := jsoniter.Unmarshal(data, &origin)
	if err != nil {
		return err
	}

	for key, val := range origin {

		key = strings.TrimSpace(key)

		if _, has := OPs[key]; has { // 解析操作符
			cond.OP = key
			cond.SetValue(val)
			continue
		}

		if key == "field" {
			if field, ok := val.(string); ok {
				cond.Field = NewExpression(field)
			}
			continue

		} else if key == "op" {
			if op, ok := val.(string); ok {
				cond.OP = op
			}
			continue

		} else if key == "value" {
			cond.SetValue(val)
			continue

		} else if key == "or" {
			if or, ok := val.(bool); ok {
				cond.OR = or
			}
			continue

		} else if key == "comment" {
			if comment, ok := val.(string); ok {
				cond.Comment = comment
			}
			continue
		} else if key == "query" { // Query
			cond.SetQuery(val)
		}

		if strings.HasPrefix(key, ":") { // 字段 {":field":"名称"}
			cond.Field = NewExpression(strings.TrimPrefix(key, ":"))
			if comment, ok := val.(string); ok {
				cond.Comment = comment
			}
			continue
		}

		if strings.HasPrefix(key, "or :") { // 字段 {"or :field":"名称"}
			cond.Field = NewExpression(strings.TrimPrefix(key, "or :"))
			cond.OR = true
			if comment, ok := val.(string); ok {
				cond.Comment = comment
			}
		}

	}

	return nil
}

// MarshalJSON for json marshalJSON
func (cond Condition) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(cond.ToMap())
}

// ToMap Order 转换为 map[string]interface{}
func (cond Condition) ToMap() map[string]interface{} {

	res := map[string]interface{}{}
	if _, has := OPs[cond.OP]; has {
		res["op"] = cond.OP
	}

	if cond.Field != nil {
		res["field"] = cond.Field.ToString()
	}

	if cond.Query != nil {
		res["query"] = cond.Query
	} else if cond.Value != nil {
		res["value"] = cond.Value
	}

	if cond.OR {
		res["or"] = cond.OR
	}

	if cond.Comment != "" {
		res["comment"] = cond.Comment
	}

	return res
}

// SetQuery 设定子查询
func (cond *Condition) SetQuery(v interface{}) {
}

// SetValue 设定数值
func (cond *Condition) SetValue(v interface{}) {

	if v == nil {
		return
	}

	cond.Value = v
	if value, ok := cond.Value.(string); ok {
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {
			value = strings.TrimPrefix(value, "{")
			value = strings.TrimSuffix(value, "}")
			cond.ValueExpression = NewExpression(value)
		}
	}
}

// ValidateWheres 校验 wheres
func (gou QueryDSL) ValidateWheres() []error {
	errs := []error{}
	if gou.Wheres == nil {
		return errs
	}

	for i, where := range gou.Wheres {
		if where.Field == nil {
			errs = append(errs, errors.Errorf("参数错误: 第 %d 个 where 查询条件, 缺少 field", i+1))
		}

		// 验证条件
		if where.OP == "" {
			where.OP = "="
		}

		check, has := OPs[where.OP]
		if !has {
			errs = append(errs, errors.Errorf("参数错误: 第 %d 个 where 查询条件,  匹配关系运算符(%s)不合法", i+1, where.OP))
		} else {
			err := check(i, where.Condition)
			if err != nil {
				errs = append(errs, err)
			}
		}

	}

	return errs
}
