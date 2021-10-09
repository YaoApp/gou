package gou

import (
	"regexp"
	"strings"

	"github.com/go-errors/errors"
	jsoniter "github.com/json-iterator/go"
)

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

// MarshalJSON for json marshalJSON
func (orders Orders) MarshalJSON() ([]byte, error) {
	res := []map[string]interface{}{}
	for _, order := range orders {
		res = append(res, order.ToMap())
	}
	return jsoniter.Marshal(res)
}

// ValidateOrders 校验 orders
func (gou QueryDSL) ValidateOrders() []error {
	errs := []error{}
	if gou.Orders == nil {
		return errs
	}

	for i, order := range gou.Orders {
		if order.Field == nil {
			errs = append(errs, errors.Errorf("参数错误: 第 %d 个 order 排序条件, 缺少 field", i+1))
		}

		if order.Sort != "desc" && order.Sort != "asc" {
			errs = append(errs, errors.Errorf("参数错误: 第 %d 个 order 排序条件, 排序方式(%s)不合法", i+1, order.Sort))
		}
	}

	return errs
}

// ToMap 转换为 map[string]interface{}
func (order Order) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"field": order.Field.ToString(),
		"sort":  order.Sort,
	}
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

	orders.Push(Order{
		Field: NewExpression(strings.TrimSpace(field)),
		Sort:  sort,
	})

	return nil
}

// PushString 添加一个排序条件 string
func (orders *Orders) PushString(order string) error {
	order = strings.TrimSpace(order)
	arr := regexp.MustCompile("[ ]+").Split(order, -1)
	if len(arr) >= 2 {
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
