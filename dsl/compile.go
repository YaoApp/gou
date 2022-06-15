package dsl

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yaoapp/gou/dsl/workshop"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
)

var regArr, _ = regexp.Compile(`([a-zA-Z0-9_-]+)\[([0-9]+)\]+`)
var keepFields = map[string]bool{"FROM": true, "RUN": true}

// compile compile the content
func (yao *YAO) compile() error {

	// COPY Content
	yao.Compiled = yao.Content

	// Compile From
	err := yao.compileFrom()
	if err != nil {
		return err
	}

	return nil
}

// compileFrom FROM
func (yao *YAO) compileFrom() error {
	if yao.Head.From == "" {
		return nil
	}

	if strings.HasPrefix(yao.Head.From, "@") {
		return yao.compileFromRemote()
	}

	return yao.compileFromLocal()
}

// compileFromRemote FROM the remote package
func (yao *YAO) compileFromRemote() error {

	remoteWorkshop, file, err := yao.fromRemoteFile()
	if err != nil {
		return err
	}

	// Trace
	yao.Trace = append(yao.Trace, file)

	// Limit
	if len(yao.Trace) > 32 {
		return fmt.Errorf("Too many layers, the max layer count is 32")
	}

	// Create remote DSL
	remote := New(remoteWorkshop)
	err = remote.Open(file)
	if err != nil {
		return err
	}

	// Compile remote file
	err = remote.Compile()
	if err != nil {
		return err
	}

	// Append the remote trace
	yao.Trace = append(yao.Trace, remote.Trace...)

	// Merge Remote Content
	return yao.merge(remote.Compiled)
}

// fromPath get the remote file
func (yao *YAO) fromRemoteFile() (remoteWorkshop *workshop.Workshop, file string, err error) {

	// VALIDATE THE FROM
	if yao.Head.From == "" {
		return nil, "", fmt.Errorf("FROM is null")
	}

	if yao.Head.From[0] != '@' {
		return nil, "", fmt.Errorf("FROM is not remote")
	}

	// COMPUTE THE PACKAGE NAME
	from := yao.Head.From[1:]
	fromArr := strings.Split(from, "/")
	if len(fromArr) < 4 {
		return nil, "", fmt.Errorf("FROM is error %s", from)
	}
	name := strings.Join(fromArr[:3], "/")

	// AUTO GET PACKAGE
	if !yao.Workshop.Has(name) {
		err = yao.Workshop.Get(name, "", func(total uint64, pkg *workshop.Package, message string) {
			log.Trace("GET %s %d ... %s", pkg.Unique, total, message)
		})
		if err != nil {
			return nil, "", fmt.Errorf("The package %s does not loaded. %s", name, err.Error())
		}
	}

	// AUTO DOWNLOAD
	isDownload, err := yao.Workshop.Mapping[name].IsDownload()
	if err != nil {
		return nil, "", fmt.Errorf("download the package %s error. %s", name, err.Error())
	}

	if !isDownload {
		_, err = yao.Workshop.Download(yao.Workshop.Mapping[name], func(total uint64, pkg *workshop.Package, message string) {
			log.Trace("Download %s %d ... %s", pkg.Unique, total, message)
		})
		if err != nil {
			return nil, "", fmt.Errorf("download the package %s error. %s", name, err.Error())
		}
	}

	// OPEN REMOTE WORKSHOP
	remoteWorkshop, err = workshop.Open(yao.Workshop.Mapping[name].LocalPath)
	if err != nil {
		return nil, "", err
	}

	// Extra file
	pathArr := []string{yao.Workshop.Mapping[name].LocalPath}
	pathArr = append(pathArr, fromArr[3:]...)
	file = filepath.Join(pathArr...) + fmt.Sprintf(".%s.yao", TypeExtensions[yao.Head.Type])

	return remoteWorkshop, file, nil
}

// merge mege the content
func (yao *YAO) merge(content map[string]interface{}) error {

	c := maps.MapOf(content)
	new := maps.MapOf(yao.Content).Dot()

	// STEP1: REPLACE
	err := yao.runReplace(c, new)
	if err != nil {
		return err
	}

	// STEP2: MERGE
	err = yao.runMerge(c, new)
	if err != nil {
		return err
	}

	// STEP3: APPEND
	err = yao.runAppend(c, new)
	if err != nil {
		return err
	}

	// STEP4: DEEP MERGE
	err = yao.runDeepMerge(c)
	if err != nil {
		return err
	}

	// STEP5: DELETE
	err = yao.runDelete(c)
	if err != nil {
		return err
	}

	yao.Compiled = c
	return nil
}

func (yao *YAO) runReplace(content, new maps.MapStr) error {

	if yao.Head.Run.REPLACE == nil {
		return nil
	}

	for _, replace := range yao.Head.Run.REPLACE {
		for key, value := range replace {
			value = yao.getValue(new, value)
			err := yao.setValue(content, key, value)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (yao *YAO) runMerge(content, new maps.MapStr) error {
	if yao.Head.Run.MERGE == nil {
		return nil
	}

	dot := content.Dot()
	for _, merge := range yao.Head.Run.MERGE {
		for key, value := range merge {
			newValue, ok := yao.getValue(new, value).(map[string]interface{})
			if !ok {
				return fmt.Errorf("The %s value is %v, not an object", key, value)
			}

			mergeValue, ok := dot.Get(key).(map[string]interface{})
			if !ok {
				return fmt.Errorf("The %s value is %v, not an object", key, dot.Get(key))
			}

			for k, v := range newValue {
				mergeValue[k] = v
			}

			err := yao.setValue(content, key, mergeValue)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (yao *YAO) runAppend(content, new maps.MapStr) error {
	if yao.Head.Run.APPEND == nil {
		return nil
	}

	dot := content.Dot()
	for _, appends := range yao.Head.Run.APPEND {
		for key, value := range appends {
			items, ok := yao.getValue(new, value).([]interface{})
			if !ok {
				return fmt.Errorf("The %s value is %v, not an array", key, value)
			}

			itemsAny := dot.Get(key)
			appendItems := []interface{}{}
			if itemsAny != nil {
				itemsArr, ok := itemsAny.([]interface{})
				if !ok {
					return fmt.Errorf("The %s value is %v, not an array", key, value)
				}
				appendItems = itemsArr
			}
			appendItems = append(appendItems, items...)
			err := yao.setValue(content, key, appendItems)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (yao *YAO) runDeepMerge(content maps.MapStr) error {
	return yao.deepMerge(content, yao.Content)
}

func (yao *YAO) runDelete(content maps.MapStr) error {
	if yao.Head.Run.DELETE == nil {
		return nil
	}

	for _, key := range yao.Head.Run.DELETE {
		err := yao.deleteValue(content, key)
		if err != nil {
			return err
		}
	}

	return nil
}

func (yao *YAO) runCopy() error { return nil }

func (yao *YAO) setValue(input maps.MapStr, key string, value interface{}) error {

	keys := strings.Split(key, ".")
	if len(keys) == 1 {

		// columns[0]
		if ok, key, idx := yao.isArrayKey(key); ok {
			return yao.setArrayValue(input, key, idx, value)
		}

		// label
		input.Set(key, value)
		return nil
	}

	// columns[0].label
	if ok, key, idx := yao.isArrayKey(keys[0]); ok {

		// columns
		arr := any.Of(input.Get(key)).CArray()
		if len(arr) <= idx {
			return fmt.Errorf("%s[%d] does not existed", key, idx)
		}

		// columns[0].label
		item := any.Of(arr[idx]).MapStr()
		err := yao.setValue(item, strings.Join(keys[1:], "."), value)
		if err != nil {
			return err
		}
		arr[idx] = item

		// set columns
		input.Set(key, arr)
		return nil
	}

	// table.name
	// table
	mapstr := any.Of(input.Get(keys[0])).MapStr()

	// table.name
	err := yao.setValue(mapstr, strings.Join(keys[1:], "."), value)
	if err != nil {
		return err
	}

	// set table
	input.Set(keys[0], mapstr)
	return nil
}

func (yao *YAO) isArrayKey(key string) (bool, string, int) {
	matches := regArr.FindStringSubmatch(key)
	if matches == nil {
		return false, "", -1
	}
	return true, matches[1], any.Of(matches[2]).CInt()
}

func (yao *YAO) setArrayValue(content maps.MapStr, key string, idx int, value interface{}) error {
	arr := content.Get(key)
	if !any.Of(arr).IsSlice() {
		return fmt.Errorf("%s is not array", key)
	}

	v := any.Of(arr).CArray()
	if len(v) <= idx {
		return fmt.Errorf("%s[%d] does not existed", key, idx)
	}

	v[idx] = value
	content.Set(key, v)
	return nil
}

func (yao *YAO) deepMerge(content, merge map[string]interface{}) error {

	for key, value := range merge {

		if keepFields[key] {
			continue
		}

		// map[string]interface{}
		if mapstr, ok := value.(map[string]interface{}); ok {

			valueAny := content[key]
			if valueAny == nil {
				content[key] = mapstr
				continue
			}

			valueMap, ok := valueAny.(map[string]interface{})
			if !ok {
				return fmt.Errorf("The %s value is %v, not a map", key, value)
			}

			err := yao.deepMerge(valueMap, mapstr)
			if err != nil {
				return err
			}

			content[key] = valueMap

			// []interface{}
		} else if arr, ok := value.([]interface{}); ok {
			valueAny := content[key]
			if valueAny == nil {
				content[key] = arr
				continue
			}

			valueArr, ok := valueAny.([]interface{})
			if !ok {
				return fmt.Errorf("The %s value is %v, not a array", key, value)
			}
			valueArr = append(valueArr, arr...)
			content[key] = valueArr

			// string, int, float, etc...
		} else {
			content[key] = value
		}
	}

	return nil
}

func (yao *YAO) deleteValue(input maps.MapStr, key string) error {

	keys := strings.Split(key, ".")
	if len(keys) == 1 {

		// columns[0]
		if ok, key, idx := yao.isArrayKey(key); ok {
			return yao.deleteArrayValue(input, key, idx)
		}

		// label
		delete(input, key)
		return nil
	}

	// columns[0].label
	if ok, key, idx := yao.isArrayKey(keys[0]); ok {

		// columns
		arr := any.Of(input.Get(key)).CArray()
		if len(arr) <= idx {
			return fmt.Errorf("%s[%d] does not existed", key, idx)
		}

		// columns[0].label
		item := any.Of(arr[idx]).MapStr()
		err := yao.deleteValue(item, strings.Join(keys[1:], "."))
		if err != nil {
			return err
		}
		arr[idx] = item

		// set columns
		input.Set(key, arr)
		return nil
	}

	// table.name
	// table
	mapstr := any.Of(input.Get(keys[0])).MapStr()

	// table.name
	err := yao.deleteValue(mapstr, strings.Join(keys[1:], "."))
	if err != nil {
		return err
	}

	// set table
	input.Set(keys[0], mapstr)
	return nil
}

func (yao *YAO) deleteArrayValue(content maps.MapStr, key string, idx int) error {
	arr := content.Get(key)
	if !any.Of(arr).IsSlice() {
		return fmt.Errorf("%s is not array", key)
	}

	v := any.Of(arr).CArray()
	if len(v) <= idx {
		return fmt.Errorf("%s[%d] does not existed", key, idx)
	}

	v = append(v[:idx], v[idx+1:]...)
	content.Set(key, v)
	return nil
}

func (yao *YAO) getValue(new maps.MapStr, value interface{}) interface{} {
	v, ok := value.(string)
	if ok && strings.HasPrefix(v, "$new.") {
		key := strings.TrimPrefix(v, "$new.")
		return new.Get(key)
	}
	return value
}

func (yao *YAO) compileFromRemoteAlias() {}

func (yao *YAO) compileFromLocal() error { return nil }
