package model

import (
	"net/url"
	"regexp"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

const reURLWhereStr = "(where|orwhere|wherein|orwherein)\\.(.+)\\.(eq|gt|lt|ge|le|like|match|in|null|notnull)"

var reURLWhere = regexp.MustCompile("^" + reURLWhereStr + "$")
var reURLGroupWhere = regexp.MustCompile("^group\\.([a-zA-Z_]{1}[0-9a-zA-Z_]+)\\." + reURLWhereStr + "$")

// AnyToQueryParam interface 转换为 QueryParams
func AnyToQueryParam(v interface{}) (QueryParam, bool) {
	params := QueryParam{}
	params, ok := v.(QueryParam)
	if ok {
		return params, true
	}

	bytes, err := jsoniter.Marshal(v)
	if err != nil {
		return params, false
	}

	err = jsoniter.Unmarshal(bytes, &params)
	if err != nil {
		return params, false
	}

	return params, true
}

// URLToQueryParam url.Values 转换为 QueryParams
func URLToQueryParam(values url.Values) QueryParam {
	param := QueryParam{
		Withs:  map[string]With{},
		Wheres: []QueryWhere{},
	}

	whereGroups := map[string][]QueryWhere{}
	for name := range values {
		if name == "select" {
			param.setSelect(values.Get(name))
			continue
		} else if name == "order" {
			param.setOrder(name, values.Get(name))
			continue
		} else if reURLWhere.MatchString(name) {
			param.setWhere(name, getURLValue(values, name))
			continue
		} else if strings.HasPrefix(name, "group.") {
			param.setGroupWhere(whereGroups, name, getURLValue(values, name))
			continue
		} else if name == "with" {
			param.setWith(values.Get(name))
			continue
		} else if strings.HasSuffix(name, ".select") {
			param.setWithSelect(name, values.Get(name))
			continue
		}
	}

	// WhereGroups
	for _, wheres := range whereGroups {
		param.Wheres = append(param.Wheres, QueryWhere{
			Wheres: wheres,
		})
	}
	return param
}

// getURLValue 读取URLvalues 数值 return []string | string
func getURLValue(values url.Values, name string) interface{} {
	if value, has := values[name]; has {
		if len(value) == 1 {
			return value[0]
		}
		return value
	}
	return ""
}

// "select", "name,secret,status,type" -> []interface{"name","secret"...}
func (param *QueryParam) setSelect(value string) {
	selects := []interface{}{}
	colmns := strings.Split(value, ",")
	for _, column := range colmns {
		selects = append(selects, strings.TrimSpace(column))
	}
	param.Select = selects
}

// "group.types.where.type.eq", "admin"
func (param *QueryParam) setGroupWhere(groups map[string][]QueryWhere, name string, value interface{}) {

	matches := reURLGroupWhere.FindStringSubmatch(name)
	group := matches[1]
	method := matches[2]
	colinfo := strings.Split(matches[3], ".")
	length := len(colinfo)
	column := colinfo[length-1]
	rel := ""
	if length > 1 {
		rel = strings.Join(colinfo[0:length-1], ".")
	}
	op := matches[4]

	where := QueryWhere{
		Method: method,
		OP:     op,
		Column: column,
		Rel:    rel,
		Value:  value,
	}
	if _, has := groups[group]; !has {
		groups[group] = []QueryWhere{}
	}
	groups[group] = append(groups[group], where)
}

// "where.status.eq" , "enabled" -> []Wheres{...}
func (param *QueryParam) setWhere(name string, value interface{}) {
	matches := reURLWhere.FindStringSubmatch(name)
	method := matches[1]
	colinfo := strings.Split(matches[2], ".")
	length := len(colinfo)
	column := colinfo[length-1]
	rel := ""
	if length > 1 {
		rel = strings.Join(colinfo[0:length-1], ".")
	}

	op := matches[3]

	where := QueryWhere{
		Method: method,
		OP:     op,
		Column: column,
		Rel:    rel,
		Value:  value,
	}
	param.Wheres = append(param.Wheres, where)
}

// "order.id" , "desc"
func (param *QueryParam) setOrder(name string, value string) {

	orders := strings.Split(value, ",")
	for _, order := range orders {
		column := order
		option := "asc"

		if strings.Contains(order, ".") {
			colinfo := strings.Split(order, ".")
			last := colinfo[len(colinfo)-1]
			if last == "asc" || last == "desc" {
				option = last
				column = strings.Join(colinfo[:1], ".")
			}
		}

		param.Orders = append(param.Orders, QueryOrder{
			Column: column,
			Option: option,
		})

	}

	// colinfo := strings.Split(name, ".")
	// column := strings.Join(colinfo[1:], ".")
	// param.Orders = append(param.Orders, QueryOrder{
	// 	Column: column,
	// 	Option: value,
	// })
}

// "with", "mother,addresses" -> map[string]With
func (param *QueryParam) setWith(value string) {
	withs := strings.Split(value, ",")
	for _, with := range withs {
		name := strings.TrimSpace(with)
		if _, has := param.Withs[name]; !has {
			param.Withs[name] = With{Name: name, Query: QueryParam{}}
		}
	}
}

// "mother.select", "name,mobile,type,status" -> map[string]With
func (param *QueryParam) setWithSelect(name string, value string) {
	namer := strings.Split(name, ".")
	withName := namer[0]
	if _, has := param.Withs[withName]; !has {
		param.setWith(withName)
	}

	selects := []interface{}{}
	colmns := strings.Split(value, ",")
	for _, column := range colmns {
		selects = append(selects, strings.TrimSpace(column))
	}

	with := param.Withs[withName]
	with.Query.Select = selects
	param.Withs[withName] = with
}
