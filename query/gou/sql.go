package gou

import (
	"fmt"
	"strings"

	"github.com/yaoapp/xun/dbal"
)

// sqlSelect Select 转换为SQL (MySQL8.0)
func (gou Query) sqlExpression(exp Expression, withDefaultAlias ...bool) interface{} {

	table := exp.Table
	alias := exp.Alias
	defaultAlias := false
	if len(withDefaultAlias) > 0 && withDefaultAlias[0] {
		defaultAlias = true
	}

	if alias != "" {
		alias = fmt.Sprintf(" AS `%s`", alias)
	}

	if exp.Table != "" {
		if exp.IsModel {
			table = gou.GetTableName(exp.Table)
		}
		table = fmt.Sprintf("`%s`.", table)
	}

	if exp.IsString {
		if value, ok := exp.Value.(string); ok {
			value = strings.ReplaceAll(value, "'", `\'`) // 防注入安全过滤
			return dbal.Raw(fmt.Sprintf("'%s'%s", value, alias))
		}
		return nil
	}

	if exp.IsNumber {
		return dbal.Raw(fmt.Sprintf("%s%v%s", table, exp.Value, alias))
	}

	if exp.IsBinding {
		if value, has := gou.Bindings[exp.Field]; has {
			return value
		}
		return nil
	}

	if exp.IsAES { // MySQL Only()

		if defaultAlias && alias == "" {
			alias = fmt.Sprintf(" AS %s", exp.Field)
		}

		return dbal.Raw(fmt.Sprintf("AES_DECRYPT(UNHEX(%s%s), '%s')%s", table, exp.Field, gou.AESKey, alias))
	}

	if exp.IsObject { // MySQL Only()

		if defaultAlias && alias == "" {
			alias = fmt.Sprintf(" AS %s", exp.Field)
		}

		key := exp.Key
		if key != "" {
			key = strings.ReplaceAll(key, "'", `\'`) // 防注入安全过滤
			return dbal.Raw(fmt.Sprintf("JSON_EXTRACT(%s`%s`, '$.%s')%s", table, exp.Field, key, alias))
		}
		return dbal.Raw(fmt.Sprintf("%s%s%s", table, exp.Field, alias))
	}

	if exp.IsArray { // MySQL Only()

		if defaultAlias && alias == "" {
			alias = fmt.Sprintf(" AS %s", exp.Field)
		}

		index := ""
		if exp.Index != Star {
			index = fmt.Sprintf("[%d]", exp.Index)
		}

		key := exp.Key
		if !strings.HasPrefix(key, "[") {
			key = strings.ReplaceAll(key, "'", `\'`) // 防注入安全过滤
			key = "." + key
		}

		if index == "" && key == "" {
			return dbal.Raw(fmt.Sprintf("%s%s%s", table, exp.Field, alias))
		}

		return dbal.Raw(fmt.Sprintf("JSON_EXTRACT(%s`%s`, '$%s%s')%s", table, exp.Field, index, key, alias))
	}

	if exp.IsFun { // MySQL Only()
		args := []string{}
		for _, arg := range exp.FunArgs {
			exp := gou.sqlExpression(arg)
			if argstr, ok := exp.(string); ok {
				args = append(args, argstr)
			} else if argraw, ok := exp.(dbal.Expression); ok {
				args = append(args, argraw.GetValue())
			}
		}
		return dbal.Raw(fmt.Sprintf("%s(%s)%s", exp.FunName, strings.Join(args, ","), alias))
	}

	return fmt.Sprintf("%s`%s`%s", table, exp.Field, alias)
}
