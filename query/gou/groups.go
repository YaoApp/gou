package gou

import (
	"regexp"
	"strings"

	"github.com/go-errors/errors"
	jsoniter "github.com/json-iterator/go"
)

// MarshalJSON for json marshalJSON
func (group Group) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(group.ToMap())
}

// ToMap Order 转换为 map[string]interface{}
func (group Group) ToMap() map[string]interface{} {
	res := map[string]interface{}{
		"field": group.Field.ToString(),
	}

	if group.Rollup != "" {
		res["rollup"] = group.Rollup
	}

	if group.Comment != "" {
		res["comment"] = group.Comment
	}

	return res
}

// UnmarshalJSON for json marshalJSON
func (groups *Groups) UnmarshalJSON(data []byte) error {

	var v interface{}
	err := jsoniter.Unmarshal(data, &v)
	if err != nil {
		return err
	}

	values := []interface{}{}
	switch v.(type) {
	case string: // "kind rollup 所有类型, city"
		strarr := strings.Split(v.(string), ",")
		for _, str := range strarr {
			groups.PushString(str)
		}
		break
	case []interface{}: // ["name", {"field":"foo"}, "id rollup 所有类型"]
		values = v.([]interface{})
		break
	}

	for _, value := range values {
		switch value.(type) {
		case string: // "kind rollup 所有类型"
			groups.PushString(value.(string))
			break
		case map[string]interface{}: // {"field":"foo"}
			groups.PushMap(value.(map[string]interface{}))
			break
		}
	}

	return nil
}

// Validate 校验数据
func (groups Groups) Validate() []error {
	errs := []error{}
	for i, group := range groups {
		if group.Field == nil {
			errs = append(errs, errors.Errorf("参数错误: 第 %d 个 group 聚合条件, 缺少 field", i+1))
			continue
		}
		err := group.Field.Validate()
		if err != nil {
			errs = append(errs, errors.Errorf("参数错误: 第 %d 个 group 聚合条件, Field %s", i+1, err.Error()))
		}
	}
	return errs
}

// Push 添加一个排序条件 Group
func (groups *Groups) Push(group Group) {
	*groups = append(*groups, group)
}

// PushMap 添加一个排序条件 map[string]inteface{}
func (groups *Groups) PushMap(group map[string]interface{}) error {

	field, ok := group["field"].(string)
	if !ok {
		return errors.Errorf("参数错误:缺少 field 字段")
	}

	g := Group{
		Field: NewExpression(strings.TrimSpace(field)),
	}

	rollup, ok := group["rollup"].(string)
	if ok && rollup != "" {
		g.Rollup = rollup
	}

	comment, ok := group["comment"].(string)
	if ok && comment != "" {
		g.Comment = comment
	}

	groups.Push(g)
	return nil
}

// PushString 添加一个排序条件 string
func (groups *Groups) PushString(group string) error {
	group = strings.TrimSpace(group)
	arr := regexp.MustCompile("[ ]+[Rr][Oo][Ll][Ll][Uu][Pp][ ]+").Split(group, -1)
	if len(arr) == 2 {
		rollup := strings.TrimSpace(arr[1])
		groups.Push(Group{
			Field:  NewExpression(strings.TrimSpace(arr[0])),
			Rollup: rollup,
		})
		return nil
	} else if len(arr) == 1 {
		groups.Push(Group{
			Field: NewExpression(strings.TrimSpace(arr[0])),
		})
		return nil
	}

	return errors.Errorf("参数错误: %s 格式错误", group)
}
