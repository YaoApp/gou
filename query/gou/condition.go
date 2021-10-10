package gou

import (
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

// OPS 可用字段
var OPS = map[string]func(i int, where Condition) error{
	"=":       condShouldHaveValue,
	">":       condShouldHaveValue,
	">=":      condShouldHaveValue,
	"<":       condShouldHaveValue,
	"<=":      condShouldHaveValue,
	"like":    condShouldHaveValue,
	"match":   condShouldHaveValue,
	"in":      condShouldHaveValue,
	"null":    condIgnoreValue,
	"notnull": condIgnoreValue,
}

// UnmarshalJSON for json marshalJSON
func (cond *Condition) UnmarshalJSON(data []byte) error {
	res := &Cond{
		OR: false,
		OP: "=",
	}
	err := jsoniter.Unmarshal(data, res)
	if err != nil {
		return err
	}
	*cond = Condition(*res)
	return nil
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

		check, has := OPS[where.OP]
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
