package gou

import (
	"regexp"
	"strings"

	"github.com/go-errors/errors"
	jsoniter "github.com/json-iterator/go"
)

// MarshalJSON for json marshalJSON
func (order Order) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(order.ToMap())
}

// ToMap Order 转换为 map[string]interface{}
func (order Order) ToMap() map[string]interface{} {
	res := map[string]interface{}{
		"field": order.Field.ToString(),
	}

	if order.Sort != "asc" {
		res["sort"] = order.Sort
	}

	if order.Comment != "" {
		res["comment"] = order.Comment
	}

	return res
}

// Validate 校验 order
func (order Order) Validate() error {
	if order.Field == nil {
		return errors.Errorf("缺少 field")
	} else if err := order.Field.Validate(); err != nil {
		return errors.Errorf("Field %s", err.Error())
	}

	if order.Sort != "desc" && order.Sort != "asc" {
		return errors.Errorf("排序方式(%s)不合法", order.Sort)
	}
	return nil
}

// UnmarshalJSON for json marshalJSON
func (orders *Orders) UnmarshalJSON(data []byte) error {

	var v interface{}
	err := jsoniter.Unmarshal(data, &v)
	if err != nil {
		return err
	}

	values := []interface{}{}
	switch v.(type) {
	case string: // "name, id desc"
		strarr := strings.Split(v.(string), ",")
		for _, str := range strarr {
			orders.PushString(str)
		}
		break
	case []interface{}: // ["name", {"field":"foo"}, "id desc"]
		values = v.([]interface{})
		break
	}

	for _, value := range values {
		switch value.(type) {
		case string: // "name asc"
			orders.PushString(value.(string))
			break
		case map[string]interface{}: // {"field":"foo"}
			orders.PushMap(value.(map[string]interface{}))
			break
		}
	}

	return nil
}

// Validate 校验数据
func (orders Orders) Validate() []error {
	errs := []error{}
	for i, order := range orders {
		if err := order.Validate(); err != nil {
			errs = append(errs, errors.Errorf("参数错误: 第 %d 个 order 排序条件, %s", i+1, err.Error()))
		}
	}
	return errs
}

// Push 添加一个排序条件 Order
func (orders *Orders) Push(order Order) {
	*orders = append(*orders, order)
}

// PushMap 添加一个排序条件 map[string]inteface{}
func (orders *Orders) PushMap(order map[string]interface{}) error {

	field, ok := order["field"].(string)
	if !ok {
		return errors.Errorf("参数错误:缺少 field 字段")
	}

	sort, ok := order["sort"].(string)
	if !ok {
		sort = "asc"
	}

	o := Order{
		Field: NewExpression(strings.TrimSpace(field)),
		Sort:  sort,
	}

	comment, ok := order["comment"].(string)
	if ok && comment != "" {
		o.Comment = comment
	}

	orders.Push(o)
	return nil
}

// PushString 添加一个排序条件 string
func (orders *Orders) PushString(order string) error {
	order = strings.TrimSpace(order)
	arr := regexp.MustCompile("[ ]+").Split(order, -1)
	if len(arr) == 2 {
		sort := strings.TrimSpace(arr[1])
		orders.Push(Order{
			Field: NewExpression(strings.TrimSpace(arr[0])),
			Sort:  sort,
		})
		return nil
	} else if len(arr) == 1 {
		orders.Push(Order{
			Field: NewExpression(strings.TrimSpace(arr[0])),
			Sort:  "asc",
		})
		return nil
	}

	return errors.Errorf("参数错误: %s 格式错误", order)
}
