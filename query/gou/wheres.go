package gou

import "github.com/go-errors/errors"

func whereShouldHaveValue(i int, where Where) error {
	if where.Value == nil && where.Query == nil {
		return errors.Errorf("参数错误: 第 %d 个 where 查询条件, 缺少 value 或 query", i+1)
	}
	return nil
}

func whereIgnoreValue(i int, where Where) error {
	return nil
}

// OPS 可用字段
var OPS = map[string]func(i int, where Where) error{
	"=":       whereShouldHaveValue,
	">":       whereShouldHaveValue,
	">=":      whereShouldHaveValue,
	"<":       whereShouldHaveValue,
	"<=":      whereShouldHaveValue,
	"like":    whereShouldHaveValue,
	"match":   whereShouldHaveValue,
	"in":      whereShouldHaveValue,
	"null":    whereIgnoreValue,
	"notnull": whereIgnoreValue,
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
			err := check(i, where)
			if err != nil {
				errs = append(errs, err)
			}
		}

	}

	return errs
}
