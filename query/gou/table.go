package gou

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-errors/errors"
	jsoniter "github.com/json-iterator/go"
)

// UnmarshalJSON for json marshalJSON
func (tab *Table) UnmarshalJSON(data []byte) error {
	var input string
	err := jsoniter.Unmarshal(data, &input)
	if err != nil {
		return err
	}

	array := regexp.MustCompile("[ ]+[Aa][Ss][ ]+").Split(input, -1)

	if len(array) == 1 {
		tab.Name = strings.TrimSpace(array[0])
	} else if len(array) == 2 {
		tab.Name = strings.TrimSpace(array[0])
		tab.Alias = strings.TrimSpace(array[1])
	} else {
		return errors.Errorf("%s 格式错误", input)
	}

	// 数据模型
	if strings.HasPrefix(tab.Name, "$") {
		tab.Name = strings.TrimPrefix(tab.Name, "$")
		tab.IsModel = true
	}

	return nil
}

// MarshalJSON for json marshalJSON
func (tab Table) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", tab.ToString())), nil
}

// ToString for json marshalJSON
func (tab Table) ToString() string {
	name := tab.Name

	// 数据模型
	if tab.IsModel {
		name = "$" + name
	}

	if tab.Alias != "" {
		name = name + " as " + tab.Alias
	}

	return name
}

// Validate 校验表达式格式
func (tab Table) Validate() error {

	reg := regexp.MustCompile("^[A-Za-z0-9_\u4e00-\u9fa5]+$")
	mreg := regexp.MustCompile("^[a-zA-Z\\.]+$")
	if !tab.IsModel && !reg.MatchString(tab.Name) {
		return errors.Errorf("数据表名称格式不正确(%s)", tab.Name)
	}

	if tab.IsModel && !mreg.MatchString(tab.Name) {
		return errors.Errorf("模型名称格式不正确(%s)", tab.Name)
	}

	if tab.Alias != "" && !reg.MatchString(tab.Alias) {
		return errors.Errorf("别名格式不正确(%s)", tab.Name)
	}

	return nil
}
