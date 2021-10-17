package gou

import (
	"fmt"
	"strings"

	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xun/dbal"
)

func (gou Query) sqlGroup(group Group, selects map[string]FieldNode) (string, []string, map[int]Expression) {

	var node *FieldNode = nil          // 关联 Select 字段节点
	var field *Expression = nil        // 关联 Select 字段信息
	fieldGroup := group.Field          // GroupBy 字段表达式
	nameGroup := fieldGroup.ToString() // GroupBy 字段名称
	id := gou.ID(*fieldGroup)

	// 是否为别名
	if n, has := selects[nameGroup]; has {
		node = &n
	} else if n, has := selects[id]; has {
		node = &n
	}

	if node == nil {
		exception.New("%s 不在 SELECT 列表中", 400, nameGroup).Throw()
	}

	field = node.Field
	update := map[int]Expression{}

	// 是否为 JSON Array 字段
	joins := []string{}
	if field.IsArray {
		table := fmt.Sprintf("__JSON_T%d", node.Index)
		name := fmt.Sprintf("F%d", node.Index)
		typ := gou.sqlTypeOf(*field)
		path := strings.TrimPrefix(field.FullPath(), "$[*]")

		join := fmt.Sprintf(
			"JSON_TABLE(%s, '$[*]' columns (`%s` %s path '$%s') ) AS `%s`",
			gou.WrapNameOf(*field), name, typ, path, table,
		)
		joins = append(joins, join)

		// 更新 SELECT 字段
		field = NewExpression(fmt.Sprintf("%s.%s AS %s", table, name, nameGroup))
		update[node.Index] = *field
	}

	// 是否为 roll up
	rollup := ""
	if group.Rollup != "" {
		fieldName := gou.NameOf(*field)
		alias := fmt.Sprintf(" AS %s", fieldName)
		if field.Alias != "" {
			alias = fmt.Sprintf(" AS %s", field.Alias)
		}

		// 更新 SELECT 字段
		field = NewExpression(fmt.Sprintf(
			":IF(:GROUPING(%s), '%s', %s)%s",
			fieldName, group.Rollup, fieldName, alias,
		))
		update[node.Index] = *field
		rollup = " WITH ROLLUP"
	}

	return fmt.Sprintf("`%s`%s", nameGroup, rollup), joins, update
}

// sqlExpression 字段表达式转换为 SQL (MySQL8.0)
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
		if exp.Index > Star {
			index = fmt.Sprintf("[%d]", exp.Index)
		} else if exp.Index == Star && exp.Key != "" {
			index = "[*]"
		}

		key := exp.Key
		if !strings.HasPrefix(key, "[") {
			key = strings.ReplaceAll(key, "'", `\'`) // 防注入安全过滤
		}

		if key != "" {
			key = "." + key
		}

		if index == "" && key == "" {
			return dbal.Raw(fmt.Sprintf("%s%s%s", table, exp.Field, alias))
		}

		return dbal.Raw(fmt.Sprintf("JSON_EXTRACT(%s`%s`, '$%s%s')%s", table, exp.Field, index, key, alias))
	}

	return fmt.Sprintf("%s`%s`%s", table, exp.Field, alias)
}

// sqlTypeOf 字段类型
func (gou *Query) sqlTypeOf(exp Expression) string {
	if exp.Type == nil {
		return "VARCHAR(255)"
	}

	name := strings.ToLower(exp.Type.Name)
	switch name {
	case "string", "char":
		if exp.Type.Length > 0 {
			return fmt.Sprintf("VARCHAR(%d)", exp.Type.Length)
		}
		return "VARCHAR(255)"

	case "integer":
		return "INT"

	case "boolean":
		return "BOOLEAN"

	case "date":
		return "DATE"

	case "time":
		return "TIME"

	case "datetime":
		return "DATETIME"

	case "timestamp":
		return "TIMESTAMP"

	case "double":
		if exp.Type.Precision > 0 {
			return fmt.Sprintf("DOUBLE(%d,%d)", exp.Type.Precision, exp.Type.Scale)
		}
		return "DOUBLE(10,2)"

	case "float":
		if exp.Type.Precision > 0 {
			return fmt.Sprintf("FLOAT(%d,%d)", exp.Type.Precision, exp.Type.Scale)
		}
		return "FLOAT(10,2)"

	case "decimal":
		if exp.Type.Precision > 0 {
			return fmt.Sprintf("DECIMAL(%d,%d)", exp.Type.Precision, exp.Type.Scale)
		}
		return "DECIMAL(10,2)"
	}

	exception.New("暂不支持 %s 类型转换", 400, exp.Type.Name).Throw()
	return ""
}
