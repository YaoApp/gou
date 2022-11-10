package lang

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/yaoapp/kun/log"
	"gopkg.in/yaml.v3"
)

var regVar, _ = regexp.Compile(`([\\]*)\$L\(([^\)]+)\)`)

// Dicts the dictionaries loaded
var Dicts = map[string]*Dict{}

// Widgets build-in widgets path mapping
var Widgets = map[string]string{
	"models": "model",
	"flows":  "flow",
	"apis":   "api",
}

// RegisterWidget Register the path of widget
func RegisterWidget(path, name string) {
	Widgets[path] = name
}

// Default the default language
var Default *Dict = &Dict{Global: Words{}, Widgets: map[string]Words{}}

// Pick get the dictionary by the ISO 639-1 standard language code
func Pick(name string) *Dict {
	dict, has := Dicts[name]
	if !has {
		return &Dict{
			Name:    name,
			Global:  Words{},
			Widgets: map[string]Words{},
		}
	}
	return dict
}

// Load the language dictionaries from the path
func Load(root string) error {
	root = path.Join(root, string(os.PathSeparator))
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}

		if strings.Count(strings.TrimPrefix(path, root), string(os.PathSeparator)) != 1 {
			return nil
		}

		dict, err := Open(path)
		if err != nil {
			return err
		}

		langName := filepath.Base(path)
		if _, has := Dicts[langName]; has {
			Dicts[langName].Merge(dict)
			return nil
		}

		Dicts[langName] = dict
		return nil
	})
}

// Open the dictionary from the language dictionary root
func Open(langRoot string) (*Dict, error) {
	langRoot = path.Join(langRoot, "/")
	langName := strings.ToLower(path.Base(langRoot))
	dict := &Dict{
		Name:    langName,
		Global:  Words{},
		Widgets: map[string]Words{},
	}

	err := filepath.Walk(langRoot, func(filename string, info os.FileInfo, err error) error {
		if err != nil {
			log.With(log.F{"root": langRoot, "filename": filename}).Error(err.Error())
			return err
		}

		if !strings.HasSuffix(filename, ".yml") {
			return nil
		}

		if filepath.Base(filename) == "global.yml" {
			words, err := OpenYaml(filename)
			if err != nil {
				return err
			}
			dict.Global = words
			return nil
		}

		name, inst := getWidgetName(langRoot, filename)
		words, err := OpenYaml(filename)
		if err != nil {
			return err
		}
		if _, has := dict.Widgets[name]; !has {
			dict.Widgets[name] = Words{}
		}

		if strings.HasSuffix(filename, ".global.yml") {
			dict.Widgets[name] = words
			return nil
		}

		fullname := fmt.Sprintf("%s.%s", name, inst)
		dict.Widgets[fullname] = words
		return nil
	})

	return dict, err
}

// getWidgetName   root: "/tests"  file: "/tests/apis/foo/bar.http.json"
func getWidgetName(root string, file string) (string, string) {
	sep := string(os.PathSeparator)
	filename := strings.TrimPrefix(file, root+sep) // "apis/foo/bar.http.json"

	parts := strings.SplitN(filename, sep, 2)

	widgetPath := parts[0]
	widget := strings.ToLower(widgetPath)
	if w, has := Widgets[widget]; has {
		widget = w
	}

	filename = strings.TrimPrefix(filename, widgetPath+sep) // foo/bar.http.json"
	namer := strings.Split(filename, ".")                   // ["foo/bar", "http", "json"]
	nametypes := strings.Split(namer[0], sep)               // ["foo", "bar"]
	name := strings.Join(nametypes, ".")                    // "foo.bar"
	return widget, name
}

// OpenYaml dict file
func OpenYaml(file string) (Words, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	words := Words{}
	err = yaml.Unmarshal(data, &words)
	if err != nil {
		return nil, err
	}
	return words, nil
}

// Replace replace the value in the global dictionary
// if was replaced return true else return false
func Replace(value *string) bool {
	if Default == nil {
		return false
	}

	if value == nil {
		return false
	}

	if v, has := Default.Global[*value]; has {
		*value = v
		return true
	}

	return false
}

// AsDefault set current dict as default
func (dict *Dict) AsDefault() *Dict {
	Default = dict
	return dict
}

// Apply Replace the words in the dictionary
// if was replaced return true else return false
func (dict *Dict) Apply(lang Lang) {
	lang.Lang(func(widgetName string, instance string, value *string) bool {
		res := dict.Replace([]string{fmt.Sprintf("%s.%s", widgetName, instance)}, value)
		if res {
			return res
		}
		return dict.ReplaceMatch([]string{fmt.Sprintf("%s.%s", widgetName, instance)}, value)
	})
}

// Merge the new dict
func (dict *Dict) Merge(new *Dict) {

	// Merge global
	if new.Global != nil {
		if dict.Global == nil {
			dict.Global = Words{}
		}
		for k, v := range new.Global {
			dict.Global[k] = v
		}
	}

	// Merge Widgets
	if new.Widgets != nil {
		if dict.Widgets == nil {
			dict.Widgets = map[string]Words{}
		}
		for name, words := range new.Widgets {
			if _, has := dict.Widgets[name]; !has {
				dict.Widgets[name] = Words{}
			}
			for key, val := range words {
				dict.Widgets[name][key] = val
			}
		}
	}

}

// Replace replace the value in the dictionary
// if was replaced return true else return false
func (dict *Dict) Replace(names []string, value *string) bool {
	if value == nil {
		return false
	}

	if strings.HasPrefix(*value, "\\:\\:") {
		val := strings.Replace(*value, "\\:\\:", "::", 1)
		*value = val
		return false
	}

	if !strings.HasPrefix(*value, "::") {
		return false
	}

	val := strings.TrimLeft(*value, "::")
	for _, name := range names {
		if words, has := dict.Widgets[name]; has {
			if v, has := words[val]; has {
				*value = v
				return true
			}
		}
	}

	if v, has := dict.Global[val]; has {
		*value = v
		return true
	}

	*value = val
	return true
}

// ReplaceMatch replace the value in the dictionary
func (dict *Dict) ReplaceMatch(names []string, value *string) bool {
	if value == nil {
		return false
	}

	matches := regVar.FindAllStringSubmatch(*value, -1)
	res := false
	for _, match := range matches {
		old := strings.TrimSpace(match[0])
		key := match[2]
		if match[1] == "\\" {
			*value = strings.ReplaceAll(*value, old, fmt.Sprintf("$L(%s)", key))
			continue
		}

		next := false
		for _, name := range names {
			if words, has := dict.Widgets[name]; has {
				if v, has := words[key]; has {
					*value = strings.ReplaceAll(*value, old, v)
					res = true
					next = true
					continue
				}
			}
		}
		if next {
			continue
		}

		if v, has := dict.Global[key]; has {
			*value = strings.ReplaceAll(*value, old, v)
			res = true
			continue
		}

		res = true
		*value = strings.ReplaceAll(*value, old, key)
	}

	return res
}

// ReplaceClone replace the value in dictionary
func (dict *Dict) ReplaceClone(widgets []string, input interface{}) (interface{}, error) {
	ref := reflect.ValueOf(input)
	kind := ref.Kind()

	switch kind {

	case reflect.Interface:

		return dict.ReplaceClone(widgets, ref.Elem())

	case reflect.Pointer:

		newPtr := reflect.New(ref.Type())

		ref = reflect.Indirect(ref)
		if !ref.IsValid() {
			return nil, nil
		}

		new, err := dict.ReplaceClone(widgets, ref.Interface())
		if err != nil {
			return nil, err
		}

		newSt := reflect.New(ref.Type())
		newSt.Elem().Set(reflect.ValueOf(new))
		newPtr.Elem().Set(newSt)
		return newPtr.Elem().Interface(), nil

	case reflect.String:
		new := ref.String()
		dict.replaceString(widgets, &new)
		return new, nil

	case reflect.Map:
		new := reflect.MakeMap(ref.Type())
		keys := ref.MapKeys()
		for _, key := range keys {
			val, err := dict.ReplaceClone(widgets, ref.MapIndex(key).Interface())
			if err == nil {
				if key.Kind() == reflect.String {
					newKey, err := dict.ReplaceClone(widgets, key.String())
					if err == nil {
						key = reflect.ValueOf(newKey)
					}
				}
				new.SetMapIndex(key, reflect.ValueOf(val))
			}
		}
		return new.Interface(), nil

	case reflect.Slice:
		values := reflect.MakeSlice(ref.Type(), 0, 0)
		for i := 0; i < ref.Len(); i++ {
			val, err := dict.ReplaceClone(widgets, ref.Index(i).Interface())
			if val == nil {
				values = reflect.Append(values, reflect.ValueOf(nil))
				continue
			}
			if err == nil {
				values = reflect.Append(values, reflect.ValueOf(val))
			}
		}
		return values.Interface(), nil

	case reflect.Struct:
		value := copyStruct(ref)
		for i := 0; i < ref.NumField(); i++ {
			if value.Field(i).CanSet() {
				if value.Field(i).Interface() != nil {
					val, err := dict.ReplaceClone(widgets, ref.Field(i).Interface())
					if err == nil && val != nil {
						value.Field(i).Set(reflect.ValueOf(val).Convert(ref.Field(i).Type()))
					}
				}
			}
		}
		return value.Interface(), nil
	}

	// fmt.Printf("%#v %#v\n", ref.IsValid(), input)
	if !ref.IsValid() {
		return nil, nil
	}
	return ref.Interface(), nil
}

// ReplaceAll replace the value in dictionary
func (dict *Dict) ReplaceAll(widgets []string, ptr interface{}) error {

	ptrRef := reflect.ValueOf(ptr)
	if ptrRef.Kind() != reflect.Pointer {
		return fmt.Errorf("the value is %s, should be a pointer", ptrRef.Kind().String())
	}

	ref := reflect.Indirect(ptrRef)
	kind := ref.Kind()
	if kind == reflect.Interface {
		ref = ref.Elem()
		kind = ref.Kind()
	}

	switch kind {

	case reflect.Pointer:
		new := ref.Interface()
		if err := dict.ReplaceAll(widgets, new); err == nil {
			ptrRef.Elem().Set(reflect.ValueOf(new))
		}
		break

	case reflect.String:
		new := ref.String()
		if dict.replaceString(widgets, &new) {
			ptrRef.Elem().Set(reflect.ValueOf(new))
		}
		break

	case reflect.Map:
		keys := ref.MapKeys()
		for _, key := range keys {
			val := ref.MapIndex(key).Interface()
			if err := dict.ReplaceAll(widgets, &val); err == nil {
				if key.Kind() == reflect.String {
					newKey, err := dict.ReplaceClone(widgets, key.String())
					if err == nil {
						key = reflect.ValueOf(newKey)
					}
				}
				ref.SetMapIndex(key, reflect.ValueOf(val))
			}
		}
		ptrRef.Elem().Set(ref)
		break

	case reflect.Slice:
		values := reflect.MakeSlice(ref.Type(), 0, 0)
		for i := 0; i < ref.Len(); i++ {
			itemVal := ref.Index(i).Interface()
			if itemVal == nil {
				values = reflect.Append(values, reflect.ValueOf(nil))
				continue
			}

			if err := dict.ReplaceAll(widgets, &itemVal); err == nil {
				values = reflect.Append(values, reflect.ValueOf(itemVal))
			}
		}
		ptrRef.Elem().Set(values)
		break

	case reflect.Struct:
		value := copyStruct(ref)
		for i := 0; i < ref.NumField(); i++ {
			if value.Field(i).CanSet() {
				val := ref.Field(i).Interface()
				if val != nil {
					if err := dict.ReplaceAll(widgets, &val); err == nil {
						value.Field(i).Set(reflect.ValueOf(val).Convert(ref.Field(i).Type()))
					}
				}
			}
		}
		ptrRef.Elem().Set(value)
		break
	}

	return nil
}

func (dict *Dict) replaceString(widgets []string, ptr *string) bool {
	if dict.Replace(widgets, ptr) {
		return true
	}
	return dict.ReplaceMatch(widgets, ptr)
}

func copyStruct(ref reflect.Value) reflect.Value {
	value := reflect.New(ref.Type()).Elem()
	makeStruct(ref.Type(), value)
	return value
}

func makeStruct(t reflect.Type, v reflect.Value) {
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		ft := t.Field(i)
		switch ft.Type.Kind() {
		case reflect.Map:
			f.Set(reflect.MakeMap(ft.Type))
		case reflect.Slice:
			f.Set(reflect.MakeSlice(ft.Type, 0, 0))
		case reflect.Chan:
			f.Set(reflect.MakeChan(ft.Type, 0))
		case reflect.Struct:
			makeStruct(ft.Type, f)
		case reflect.Ptr:
			fv := reflect.New(ft.Type.Elem())
			makeStruct(ft.Type.Elem(), fv.Elem())
			if f.CanSet() {
				f.Set(fv)
			}
		default:
		}
	}
}
