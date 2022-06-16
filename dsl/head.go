package dsl

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/blang/semver/v4"
)

// NewHead create a new head of yao
func NewHead() *Head {
	return &Head{
		Run: &Command{},
	}
}

// SetFrom set the from
func (head *Head) SetFrom(from interface{}) bool {
	if from == nil {
		return false
	}

	if from, ok := from.(string); ok {
		head.From = strings.ToLower(from)
		return true
	}

	return false
}

// SetLang set the YAO DSL version
func (head *Head) SetLang(lang interface{}) error {
	ver, err := head.NewVersion(lang)
	if err != nil {
		return err
	}
	head.Lang = ver
	return nil
}

// SetVersion set the CURRENT DSL version
func (head *Head) SetVersion(version interface{}) error {
	ver, err := head.NewVersion(version)
	if err != nil {
		return err
	}
	head.Version = ver
	return nil
}

// SetType set the type of DSL
func (head *Head) SetType(kind int) {
	head.Type = kind
}

// SetName set the name of DSL
func (head *Head) SetName(name string) {
	head.Name = strings.ToLower(name)
}

// SetFile parse file path and set the name and kind
func (head *Head) SetFile(file string) error {

	base := filepath.Base(file)
	uri := strings.Split(base, ".")
	if len(uri) != 3 { // name.type.yao
		return fmt.Errorf("the file name should be \"name.type.yao\", but got: %s ", base)
	}

	if uri[2] != "yao" && uri[2] != "json" && uri[2] != "jsonc" {
		return fmt.Errorf("the file extension should be yao or jsonc or json, but got: %s ", uri[2])
	}

	typ, has := ExtensionTypes[uri[1]]
	if !has {
		return fmt.Errorf("the type \"%s\" of YAO DSL does not support", uri[1])
	}

	if !filepath.IsAbs(file) {
		file, err := filepath.Abs(file)
		if err != nil {
			return err
		}
		head.File = file
	} else {
		head.File = file
	}

	head.SetType(typ)
	head.SetName(uri[0])
	return nil
}

// SetCommand set the command of DSL
func (head *Head) SetCommand(cmd interface{}) error {
	if cmd == nil {
		return nil
	}

	if cmd, ok := cmd.(map[string]interface{}); ok {
		if err := head.setAppend(cmd["APPEND"]); err != nil {
			return err
		}
		if err := head.setDelete(cmd["DELETE"]); err != nil {
			return err
		}
		if err := head.setReplace(cmd["REPLACE"]); err != nil {
			return err
		}
		if err := head.setMerge(cmd["MERGE"]); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("the RUN should be map of string, but got: %v", cmd)

}

// NewVersion any to version
func (head *Head) NewVersion(ver interface{}) (*semver.Version, error) {
	if ver == nil {
		return semver.New("1.0.0")
	}

	version, ok := ver.(string)
	if !ok || version == "" {
		return semver.New("1.0.0")
	}

	return semver.New(strings.TrimLeft(strings.ToLower(version), "v"))
}

// setAppend set the APPEND of DSL
func (head *Head) setAppend(input interface{}) error {
	if input == nil {
		return nil
	}

	if value, ok := input.([]interface{}); ok {
		cmds := []map[string][]interface{}{}
		for i, val := range value {
			icmd, ok := val.(map[string]interface{})
			if !ok {
				return fmt.Errorf("the APPEND should be array of map string array, but got APPEND.%d: %#v", i, val)
			}

			cmd := map[string][]interface{}{}
			for key, v := range icmd {
				c, ok := v.([]interface{})
				if !ok {
					return fmt.Errorf("the APPEND should be array of map string array, but got APPEND.%d: %#v", i, val)
				}
				cmd[key] = c
			}

			cmds = append(cmds, cmd)
		}
		head.Run.APPEND = cmds
		return nil
	}

	return fmt.Errorf("the APPEND should be array of map string array, but got: %#v", input)
}

// setDelete set the DELETE of DSL
func (head *Head) setDelete(input interface{}) error {
	if input == nil {
		return nil
	}

	if value, ok := input.([]interface{}); ok {
		cmds := []string{}
		for i, val := range value {
			cmd, ok := val.(string)
			if !ok {
				return fmt.Errorf("the DELETE should be array of string, but got DELETE.%d: %#v", i, val)
			}
			cmds = append(cmds, cmd)
		}
		head.Run.DELETE = cmds
		return nil
	}

	return fmt.Errorf("the DELETE should be array of string, but got: %#v", input)
}

// setReplace set the REPLACE of DSL
func (head *Head) setReplace(input interface{}) error {
	if input == nil {
		return nil
	}

	if value, ok := input.([]interface{}); ok {
		cmds := []map[string]interface{}{}
		for i, val := range value {
			cmd, ok := val.(map[string]interface{})
			if !ok {
				return fmt.Errorf("the REPLACE should be array of map, but got REPLACE.%d: %#v", i, val)
			}
			cmds = append(cmds, cmd)
		}
		head.Run.REPLACE = cmds
		return nil
	}
	return fmt.Errorf("the REPLACE should be array of string, but got: %v", input)
}

// setMerge set the MERGE of DSL
func (head *Head) setMerge(merge interface{}) error {
	if merge == nil {
		return nil
	}

	if values, ok := merge.([]interface{}); ok {
		cmds := []map[string]interface{}{}
		for i, val := range values {
			cmd, ok := val.(map[string]interface{})
			if !ok {
				return fmt.Errorf("the MERGE should be array of map, but got MERGE.%d: %v", i, val)
			}
			cmds = append(cmds, cmd)
		}
		head.Run.MERGE = cmds
		return nil
	}
	return fmt.Errorf("the MERGE should be array of map string, but got: %v", merge)
}
