package v8

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-sourcemap/sourcemap"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	"rogchap.com/v8go"
)

// SourceMaps the source maps
var SourceMaps = map[string][]byte{}

// SourceCodes the source codes
var SourceCodes = map[string][]byte{}

// ModuleSourceMaps the source maps for modules
var ModuleSourceMaps = map[string][]byte{}

// reStackEntry the stack entry regex
var reStackEntry = regexp.MustCompile(`at[ ]+(?P<Function>[^(]+)[ ]+\((?P<File>[^:]+):(?P<Line>\d+):(?P<Column>\d+)\)`)

// SourceMap source map
type SourceMap struct {
	Version        int      `json:"version"`
	File           string   `json:"file"`
	SourceRoot     string   `json:"sourceRoot,omitempty"`
	Sources        []string `json:"sources"`
	Names          []string `json:"names"`
	Mappings       string   `json:"mappings"`
	SourcesContent []string `json:"sourcesContent,omitempty"`
	bytes          []byte
	path           string
	offset         int
	count          int
}

// StackLogEntry stack log entry
type StackLogEntry struct {
	Function string `json:"function,omitempty"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Message  string `json:"message,omitempty"`
}

// StackLogEntryList stack log entry list
type StackLogEntryList []*StackLogEntry

// sourceMapIndex the source map index
type sourceMapIndex struct {
	indexes []int
	maps    []*SourceMap
	file    string
}

func clearSourceMaps() {
	ModuleSourceMaps = map[string][]byte{}
	SourceMaps = map[string][]byte{}
	SourceCodes = map[string][]byte{}
}

// StackTrace get the stack trace
func StackTrace(jserr *v8go.JSError, rootMapping interface{}) string {

	// Production mode will not show the stack trace
	if runtimeOption.Debug == false {
		return jserr.Message
	}

	// Development mode will show the stack trace
	entries := parseStackTrace(jserr.StackTrace)
	if entries == nil || len(entries) == 0 {
		return jserr.StackTrace
	}

	output, err := entries.String(rootMapping)
	if err != nil {
		return err.Error() + "\n" + jserr.StackTrace
	}

	return fmt.Sprintf("%s\n%s", jserr.Message, output)
}

func (entry *StackLogEntry) String() string {
	return fmt.Sprintf("    at %s (%s:%d:%d)", entry.Function, entry.File, entry.Line, entry.Column)
}

// String the stack log entry list to string
func (list StackLogEntryList) String(rootMapping interface{}) (string, error) {
	if len(list) == 0 {
		return "", fmt.Errorf("StackLogEntryList.String(), empty list")
	}

	index, err := parseSourceMaps(list[0].File)
	if err != nil || index == nil {
		return "", fmt.Errorf("StackLogEntryList.String(), parse source maps error %s", err)
	}

	for _, entry := range list {
		line := entry.Line
		sm := index.getSourceMap(line)
		line -= sm.offset
		smap, err := sourcemap.Parse(sm.path, sm.bytes)
		if err != nil {
			return "", fmt.Errorf("StackLogEntryList.String(), parse source maps error. %s", err)
		}

		file, fn, line, col, ok := smap.Source(line, entry.Column)
		if ok {
			entry.File = fmtFilePath(file, rootMapping)
			entry.Line = line
			entry.Column = col
			if fn != "" {
				entry.Function = fn
			}
		}
	}

	output := []string{}
	for _, entry := range list {
		output = append(output, entry.String())
	}
	return strings.Join(output, "\n"), nil
}

func (index *sourceMapIndex) getSourceMap(line int) *SourceMap {
	for i, offset := range index.indexes {
		if line < offset {
			return index.maps[i]
		}
	}
	return index.maps[0]
}

func parseSourceMaps(file string) (*sourceMapIndex, error) {

	data, has := SourceMaps[file]
	if !has {
		return nil, nil
	}

	source, has := SourceCodes[file]
	if !has {
		return nil, nil
	}

	sm, err := NewSourceMap(data)
	if err != nil {
		return nil, err
	}

	sm.count = cntSource(string(source))

	index := &sourceMapIndex{
		indexes: []int{},
		maps:    []*SourceMap{sm},
		file:    file,
	}
	var offset int = 0
	if runtimeOption.Import {
		if imports, has := ImportMap[file]; has {
			for _, imp := range imports {
				data := ModuleSourceMaps[imp.AbsPath]
				if !has {
					continue
				}

				module, has := Modules[imp.AbsPath]
				if !has {
					continue
				}

				ism, err := NewSourceMap(data)
				if err != nil {
					return nil, err
				}
				ism.count = cntSource(module.Source)
				index.maps = append(index.maps, ism)
				index.indexes = append(index.indexes, offset)
				ism.offset = offset
				ism.path = imp.Path
				offset += ism.count
			}
		}
	}

	sm.offset = offset
	sm.path = file
	index.indexes = append(index.indexes, offset)
	return index, nil
}

// NewSourceMap create a new source map
func NewSourceMap(data []byte) (*SourceMap, error) {
	var sourceMap SourceMap
	err := jsoniter.Unmarshal(data, &sourceMap)
	if err != nil {
		return nil, err
	}

	sourceMap.bytes = data
	sourceMap.offset = 0
	return &sourceMap, nil
}

func parseStackTrace(trace string) StackLogEntryList {
	res := []*StackLogEntry{}
	lines := strings.Split(trace, "\n")
	for _, line := range lines {
		match := reStackEntry.FindStringSubmatch(line)
		if match != nil {
			line, _ := strconv.Atoi(match[3])
			column, _ := strconv.Atoi(match[4])
			entry := &StackLogEntry{
				Function: match[1],
				File:     match[2],
				Line:     line,
				Column:   column,
			}
			res = append(res, entry)
		}
	}
	return res
}

func fmtFilePath(file string, rootMapping interface{}) string {
	file = strings.ReplaceAll(file, ".."+string(os.PathSeparator), "")
	if !strings.HasPrefix(file, string(os.PathSeparator)) {
		file = string(os.PathSeparator) + file
	}

	file = strings.TrimPrefix(file, application.App.Root())
	if rootMapping != nil {
		switch mapping := rootMapping.(type) {
		case map[string]string:
			for name, mappping := range mapping {
				if strings.HasPrefix(file, name) {
					file = mappping + strings.TrimPrefix(file, name)
					break
				}
			}
			break

		case func(string) string:
			file = mapping(file)
			break
		}
	}
	return file
}

func cntSource(source string) int {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	return strings.Count(source, "\n")
}
