package gou

import "github.com/go-errors/errors"

// Validate 校验DSL格式
func (gou QueryDSL) Validate() []error {

	errs := []error{}
	if gou.SQL != nil {
		errs = append(errs, gou.ValidateSQL()...)
		return errs
	}

	errs = append(errs, gou.ValidateSelect()...)  // select
	errs = append(errs, gou.ValidateWheres()...)  // wheres
	errs = append(errs, gou.ValidateOrders()...)  // orders
	errs = append(errs, gou.ValidateGroups()...)  // groups
	errs = append(errs, gou.ValidateHavings()...) // havings
	errs = append(errs, gou.ValidateUnions()...)  // unions
	errs = append(errs, gou.ValidateQuery()...)   // query
	errs = append(errs, gou.ValidateJoins()...)   // joins
	errs = append(errs, gou.ValidateSQL()...)     // sql

	return errs
}

// ValidateSelect 校验 select
func (gou QueryDSL) ValidateSelect() []error {
	errs := []error{}
	if gou.Select == nil {
		errs = append(errs, errors.Errorf("参数错误: select 和 sql 必须填写一项"))
	}

	if gou.Query == nil && gou.From == nil {
		errs = append(errs, errors.Errorf("参数错误: from 和 query 必须填写一项"))
	}

	return errs
}

// ValidateWheres 校验 wheres
func (gou QueryDSL) ValidateWheres() []error {
	errs := []error{}
	if gou.Wheres == nil {
		return errs
	}

	for i, where := range gou.Wheres {
		errs := where.Condition.Validate()
		for _, err := range errs {
			errs = append(errs, errors.Errorf("参数错误: 第 %d 个 where 查询条件,  %s", i+1, err.Error()))
		}
	}

	return errs
}

// ValidateOrders 校验 orders
func (gou QueryDSL) ValidateOrders() []error {
	errs := []error{}
	if gou.Orders == nil {
		return errs
	}
	return gou.Orders.Validate()
}

// ValidateGroups 校验 groups
func (gou QueryDSL) ValidateGroups() []error {
	if gou.Groups == nil {
		return []error{}
	}
	return gou.Groups.Validate()
}

// ValidateHavings 校验 havings
func (gou QueryDSL) ValidateHavings() []error {
	errs := []error{}
	if len(gou.Havings) > 0 && gou.Groups == nil {
		errs = append(errs, errors.Errorf("参数错误: 缺少 groups, havings 仅对 groups 有效"))
	}

	for i, having := range gou.Havings {
		errs := having.Condition.Validate()
		for _, err := range errs {
			errs = append(errs, errors.Errorf("参数错误: 第 %d 个 having 查询条件,  %s", i+1, err.Error()))
		}
	}
	return errs
}

// ValidateUnions 校验 unions
func (gou QueryDSL) ValidateUnions() []error {
	if gou.Unions == nil {
		return []error{}
	}
	errs := []error{}
	for i, union := range gou.Unions {
		errs := union.Validate()
		for _, err := range errs {
			errs = append(errs, errors.Errorf("参数错误: 第 %d 个 union 查询,  %s", i+1, err.Error()))
		}
	}
	return errs
}

// ValidateQuery 校验 query
func (gou QueryDSL) ValidateQuery() []error {
	if gou.Query == nil {
		return []error{}
	}
	return gou.Query.Validate()
}

// ValidateJoins 校验 joins
func (gou QueryDSL) ValidateJoins() []error {

	errs := []error{}
	if gou.Joins == nil {
		return []error{}
	}

	for i, join := range gou.Joins {

		if join.Key == nil {
			errs = append(errs, errors.Errorf("参数错误: 第 %d 个 join 查询, 缺少 key", i+1))
		}

		if join.Foreign == nil {
			errs = append(errs, errors.Errorf("参数错误: 第 %d 个 join 查询, 缺少 foreign", i+1))
		}

		if join.From == nil {
			errs = append(errs, errors.Errorf("参数错误: 第 %d 个 join 查询, 缺少 from", i+1))
		}
	}

	return errs
}

// ValidateSQL 校验 sql
func (gou QueryDSL) ValidateSQL() []error {
	errs := []error{}
	if gou.SQL.STMT == "" {
		errs = append(errs, errors.Errorf("参数错误: sql.stmt 必须填写"))
	}
	return errs
}
