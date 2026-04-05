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

		if gou.driver == "postgres" {
			if path == "" || path == "." {
				join := fmt.Sprintf(
					"LATERAL jsonb_array_elements_text(%s::jsonb) WITH ORDINALITY AS %s(%s, __ord)",
					gou.WrapNameOf(*field), gou.Quote(table), gou.Quote(name))
				joins = append(joins, join)
			} else {
				path = strings.TrimPrefix(path, ".")
				join := fmt.Sprintf(
					"LATERAL jsonb_array_elements(%s::jsonb) WITH ORDINALITY AS %s(__elem, __ord)",
					gou.WrapNameOf(*field), gou.Quote(table))
				joins = append(joins, join)
				_ = typ
				_ = path
				castType := gou.pgCastType(*field)
				field = NewExpression(fmt.Sprintf("(%s.__elem->>'%s')::%s AS %s", table, path, castType, nameGroup))
				update[node.Index] = *field
			}
			if _, ok := update[node.Index]; !ok {
				field = NewExpression(fmt.Sprintf("%s.%s AS %s", table, name, nameGroup))
				update[node.Index] = *field
			}
		} else {
			join := fmt.Sprintf(
				"JSON_TABLE(%s, '$[*]' columns (`%s` %s path '$%s') ) AS `%s`",
				gou.WrapNameOf(*field), name, typ, path, table,
			)
			joins = append(joins, join)
			field = NewExpression(fmt.Sprintf("%s.%s AS %s", table, name, nameGroup))
			update[node.Index] = *field
		}
	}

	// 是否为 roll up
	rollup := ""
	if group.Rollup != "" {
		fieldName := gou.NameOf(*field)
		alias := fmt.Sprintf(" AS %s", fieldName)
		if field.Alias != "" {
			alias = fmt.Sprintf(" AS %s", field.Alias)
		}

		if gou.driver == "postgres" {
			field = NewExpression(fmt.Sprintf(
				":CASEGROUPING(:GROUPING(%s), '%s', %s)%s",
				fieldName, group.Rollup, fieldName, alias,
			))
			update[node.Index] = *field
			rollup = " __ROLLUP__"
		} else {
			field = NewExpression(fmt.Sprintf(
				":IF(:GROUPING(%s), '%s', %s)%s",
				fieldName, group.Rollup, fieldName, alias,
			))
			update[node.Index] = *field
			rollup = " WITH ROLLUP"
		}
	}

	return gou.Quote(nameGroup) + rollup, joins, update
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
		alias = " AS " + gou.Quote(alias)
	}

	if exp.Table != "" {
		if exp.IsModel {
			table = gou.GetTableName(exp.Table)
		}
		table = gou.Quote(table) + "."
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
		return exp.Field
		// if value, has := gou.Bindings[exp.Field]; has {
		// 	return value
		// }
		// return nil
	}

	if exp.IsAES {
		if defaultAlias && alias == "" {
			alias = fmt.Sprintf(" AS %s", exp.Field)
		}
		safeKey := strings.ReplaceAll(gou.AESKey, "'", "''")
		if gou.driver == "postgres" {
			return dbal.Raw(fmt.Sprintf("pgp_sym_decrypt(decode(%s%s, 'hex'), '%s')%s", table, gou.Quote(exp.Field), safeKey, alias))
		}
		return dbal.Raw(fmt.Sprintf("AES_DECRYPT(UNHEX(%s%s), '%s')%s", table, gou.Quote(exp.Field), safeKey, alias))
	}

	if exp.IsFun {
		args := []string{}
		for _, arg := range exp.FunArgs {
			exp := gou.sqlExpression(arg)
			if argstr, ok := exp.(string); ok {
				args = append(args, argstr)
			} else if argraw, ok := exp.(dbal.Expression); ok {
				args = append(args, argraw.GetValue())
			}
		}
		funName := exp.FunName
		if gou.driver == "postgres" {
			upper := strings.ToUpper(funName)
			if upper == "IF" && len(args) == 3 {
				return dbal.Raw(fmt.Sprintf("CASE WHEN %s = 1 THEN %s ELSE %s END%s", args[0], args[1], args[2], alias))
			}
			if upper == "CASEGROUPING" && len(args) == 3 {
				return dbal.Raw(fmt.Sprintf("CASE WHEN %s = 1 THEN %s ELSE %s END%s", args[0], args[1], args[2], alias))
			}
		}
		return dbal.Raw(fmt.Sprintf("%s(%s)%s", funName, strings.Join(args, ","), alias))
	}

	if exp.IsObject {
		if defaultAlias && alias == "" {
			alias = fmt.Sprintf(" AS %s", exp.Field)
		}
		key := exp.Key
		if key != "" {
			key = strings.ReplaceAll(key, "'", `\'`)
			if gou.driver == "postgres" {
				return dbal.Raw(fmt.Sprintf("%s%s::jsonb->>'%s'%s", table, gou.Quote(exp.Field), key, alias))
			}
			return dbal.Raw(fmt.Sprintf("JSON_EXTRACT(%s`%s`, '$.%s')%s", table, exp.Field, key, alias))
		}
		return dbal.Raw(fmt.Sprintf("%s%s%s", table, exp.Field, alias))
	}

	if exp.IsArray {
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
			key = strings.ReplaceAll(key, "'", `\'`)
		}

		if key != "" {
			key = "." + key
		}

		if index == "" && key == "" {
			return dbal.Raw(fmt.Sprintf("%s%s%s", table, exp.Field, alias))
		}

		if gou.driver == "postgres" {
			jsonPath := strings.TrimPrefix(index+key, ".")
			jsonPath = strings.ReplaceAll(jsonPath, "[*]", "")
			jsonPath = strings.ReplaceAll(jsonPath, ".", ",")
			jsonPath = strings.ReplaceAll(jsonPath, "[", "")
			jsonPath = strings.ReplaceAll(jsonPath, "]", "")
			if jsonPath != "" {
				return dbal.Raw(fmt.Sprintf("%s%s::jsonb#>>'{%s}'%s", table, gou.Quote(exp.Field), jsonPath, alias))
			}
			return dbal.Raw(fmt.Sprintf("%s%s::jsonb%s", table, gou.Quote(exp.Field), alias))
		}

		return dbal.Raw(fmt.Sprintf("JSON_EXTRACT(%s`%s`, '$%s%s')%s", table, exp.Field, index, key, alias))
	}

	return dbal.Raw(fmt.Sprintf("%s%s%s", table, gou.Quote(exp.Field), alias))
}

// sqlTypeOf 字段类型 (用于 JSON_TABLE 列定义)
func (gou *Query) sqlTypeOf(exp Expression) string {
	if exp.Type == nil {
		return "VARCHAR(255)"
	}

	name := strings.ToLower(exp.Type.Name)
	isPG := gou.driver == "postgres"

	switch name {
	case "string", "char":
		if exp.Type.Length > 0 {
			return fmt.Sprintf("VARCHAR(%d)", exp.Type.Length)
		}
		return "VARCHAR(255)"

	case "integer":
		if isPG {
			return "INTEGER"
		}
		return "INT"

	case "boolean":
		return "BOOLEAN"

	case "date":
		return "DATE"

	case "time":
		return "TIME"

	case "datetime":
		if isPG {
			return "TIMESTAMP"
		}
		return "DATETIME"

	case "timestamp":
		return "TIMESTAMP"

	case "double":
		if isPG {
			return "DOUBLE PRECISION"
		}
		if exp.Type.Precision > 0 {
			return fmt.Sprintf("DOUBLE(%d,%d)", exp.Type.Precision, exp.Type.Scale)
		}
		return "DOUBLE(10,2)"

	case "float":
		if isPG {
			return "REAL"
		}
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

// pgCastType returns the PG cast type for LATERAL jsonb element extraction
func (gou *Query) pgCastType(exp Expression) string {
	if exp.Type == nil {
		return "text"
	}
	name := strings.ToLower(exp.Type.Name)
	switch name {
	case "string", "char":
		return "text"
	case "integer":
		return "integer"
	case "boolean":
		return "boolean"
	case "double":
		return "double precision"
	case "float":
		return "real"
	case "decimal":
		if exp.Type.Precision > 0 {
			return fmt.Sprintf("numeric(%d,%d)", exp.Type.Precision, exp.Type.Scale)
		}
		return "numeric(10,2)"
	}
	return "text"
}
