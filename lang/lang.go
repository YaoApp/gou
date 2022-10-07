package lang

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
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
var Default *Dict = &Dict{Global: Words{}, Widgets: map[string]Widget{}}

// Pick get the dictionary by the ISO 639-1 standard language code
func Pick(name string) *Dict {
	dict, has := Dicts[name]
	if !has {
		return &Dict{
			Name:    name,
			Global:  Words{},
			Widgets: map[string]Widget{},
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
		Widgets: map[string]Widget{},
	}

	err := filepath.Walk(langRoot, func(filename string, info os.FileInfo, err error) error {
		if err != nil {
			log.With(log.F{"root": langRoot, "filename": filename}).Error(err.Error())
			return err
		}

		if strings.HasSuffix(filename, "global.yml") {
			words, err := OpenYaml(filename)
			if err != nil {
				return err
			}
			dict.Global = words
			return nil
		}

		if strings.HasSuffix(filename, ".yml") {
			widget, inst := getWidgetName(langRoot, filename)
			words, err := OpenYaml(filename)
			if err != nil {
				return err
			}
			if _, has := dict.Widgets[widget]; !has {
				dict.Widgets[widget] = map[string]Words{}
			}
			dict.Widgets[widget][inst] = words
		}

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

// Apply Replace the words in the dictionary
// if was replaced return true else return false
func (dict *Dict) Apply(lang Lang) {
	lang.Lang(func(widgetName string, inst string, value *string) bool {
		res := dict.Replace(widgetName, inst, value)
		if res {
			return res
		}
		return dict.ReplaceMatch(widgetName, inst, value)
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
			dict.Widgets = map[string]Widget{}
		}

		for name, widget := range new.Widgets {
			if dict.Widgets[name] == nil {
				dict.Widgets[name] = Widget{}
			}

			for inst, words := range widget {
				if dict.Widgets[name][inst] == nil {
					dict.Widgets[name][inst] = Words{}
				}
				for k, v := range words {
					dict.Widgets[name][inst][k] = v
				}
			}
		}
	}

}

// Replace replace the value in the dictionary
// if was replaced return true else return false
func (dict *Dict) Replace(widgetName string, inst string, value *string) bool {
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
	if widget, has := dict.Widgets[widgetName]; has {
		if words, has := widget[inst]; has {
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
	return false
}

// ReplaceMatch replace the value in the dictionary
func (dict *Dict) ReplaceMatch(widgetName string, inst string, value *string) bool {
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

		if widget, has := dict.Widgets[widgetName]; has {
			if words, has := widget[inst]; has {
				if v, has := words[key]; has {
					*value = strings.ReplaceAll(*value, old, v)
					res = true
					continue
				}
			}
		}

		if v, has := dict.Global[key]; has {
			*value = strings.ReplaceAll(*value, old, v)
			res = true
			continue
		}

		*value = strings.ReplaceAll(*value, old, key)
	}

	return res
}

// AsDefault set current dict as default
func (dict *Dict) AsDefault() *Dict {
	Default = dict
	return dict
}
