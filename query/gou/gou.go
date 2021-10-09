package gou

import "github.com/go-errors/errors"

// Validate 校验DSL格式
func (gou QueryDSL) Validate() []error {

	errs := []error{}
	if gou.SQL != nil {
		errs = append(errs, gou.ValidateSQL()...)
		return errs
	}

	errs = append(errs, gou.ValidateSelect()...) // select
	errs = append(errs, gou.ValidateWheres()...) // wheres
	errs = append(errs, gou.ValidateOrders()...) // orders
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

// ValidateSQL 校验 sql
func (gou QueryDSL) ValidateSQL() []error {
	errs := []error{}
	if gou.SQL.STMT == "" {
		errs = append(errs, errors.Errorf("参数错误: sql.stmt 必须填写"))
	}
	return errs
}
