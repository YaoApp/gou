package gou

import (
	"fmt"
	"strings"

	"github.com/go-errors/errors"
)

// UnmarshalJSON for json marshalJSON
func (tab *Table) UnmarshalJSON(data []byte) error {
	input := strings.ToLower(string(data))
	array := strings.Split(input, " as ")
	if len(array) == 1 {
		tab.Name = strings.TrimSpace(array[0])
		return nil
	} else if len(array) == 2 {
		tab.Name = strings.TrimSpace(array[0])
		tab.Alias = strings.TrimSpace(array[1])
		return nil
	}
	return errors.Errorf("%s 格式错误", input)
}

// MarshalJSON for json marshalJSON
func (tab *Table) MarshalJSON() ([]byte, error) {
	if tab.Alias == "" {
		return []byte(tab.Name), nil
	}
	return []byte(fmt.Sprintf("%s as %s", tab.Name, tab.Alias)), nil
}
