package gou

import (
	"strings"

	"github.com/go-errors/errors"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/exception"
)

// OPs 可用的操作符
var OPs = map[string]func(cond Condition) error{
	"=":     condShouldHaveValue,
	">":     condShouldHaveValue,
	">=":    condShouldHaveValue,
	"<":     condShouldHaveValue,
	"<=":    condShouldHaveValue,
	"like":  condShouldHaveValue,
	"match": condShouldHaveValue,
	"in":    condShouldHaveValue,
	"is":    condShouldHaveNull,
}

func condShouldHaveValue(cond Condition) error {
	if cond.Value == nil && cond.Query == nil {
		return errors.Errorf("缺少 value 或 query")
	}
	return nil
}

func condShouldHaveNull(cond Condition) error {
	if cond.Value == nil && cond.Query == nil {
		return errors.Errorf("缺少 value 或 query")
	}

	value, ok := cond.Value.(string)
	if !ok || (value != "null" && value != "not null") {
		return errors.Errorf("%s 应该为 null 或 not null", value)
	}

	return nil
}

// UnmarshalJSON for json marshalJSON
func (cond *Condition) UnmarshalJSON(data []byte) error {
	origin := map[string]interface{}{}
	err := jsoniter.Unmarshal(data, &origin)
	if err != nil {
		return err
	}
	*cond = ConditionOf(origin)
	return nil
}

// MarshalJSON for json marshalJSON
func (cond Condition) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(cond.ToMap())
}

// ConditionOf 从 map[string]interface{}
func ConditionOf(input map[string]interface{}) Condition {
	cond := Condition{}
	for key, val := range input {

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
	return cond
}

// ToMap Condition 转换为 map[string]interface{}
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

	if query, ok := v.(QueryDSL); ok {
		cond.Query = &query
		return
	}

	data, err := jsoniter.Marshal(v)
	if err != nil {
		exception.New("设定子查询错误(%s)", 400, err.Error()).Throw()
	}

	var dsl QueryDSL
	err = jsoniter.Unmarshal(data, &dsl)
	if err != nil {
		exception.New("设定子查询错误(%s)", 400, err.Error()).Throw()
	}

	cond.Query = &dsl
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

// Validate 校验数据
func (cond Condition) Validate() []error {

	errs := []error{}
	if cond.Field == nil {
		errs = append(errs, errors.Errorf("缺少 field"))
	} else if err := cond.Field.Validate(); err != nil {
		errs = append(errs, errors.Errorf("field %s", err.Error()))
	}

	if cond.OP == "" {
		errs = append(errs, errors.Errorf("缺少 op"))
	} else if _, has := OPs[cond.OP]; !has {
		errs = append(errs, errors.Errorf("%s 操作符暂不支持", cond.OP))
	}

	if cond.ValueExpression != nil {
		if err := cond.ValueExpression.Validate(); err != nil {
			errs = append(errs, errors.Errorf("value %s", err.Error()))
		}
	}

	if valueValidate, ok := OPs[cond.OP]; ok {
		err := valueValidate(cond)
		if err != nil {
			errs = append(errs, errors.Errorf("value %s", err.Error()))
		}
	}

	if cond.Query != nil {
		if suberrs := cond.Query.Validate(); len(suberrs) > 0 {
			for _, err := range suberrs {
				errs = append(errs, errors.Errorf("query %s", err.Error()))
			}
		}
	}

	return errs
}
