package v8

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-sourcemap/sourcemap"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	"rogchap.com/v8go"
)

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
}

// StackLogEntry stack log entry
type StackLogEntry struct {
	Function string `json:"function,omitempty"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
}

// NewSourceMap create a new source map
func NewSourceMap(content []byte) (*SourceMap, error) {
	var sm SourceMap = SourceMap{}
	err := jsoniter.Unmarshal(content, &sm)
	if err != nil {
		return nil, err
	}
	return &sm, nil
}

func (sm *SourceMap) String() string {
	content, _ := jsoniter.MarshalToString(sm)
	return content
}

// Bytes s
func (sm *SourceMap) Bytes() []byte {
	bytes, _ := jsoniter.Marshal(sm)
	return bytes
}

// Merge merge source map
func (sm *SourceMap) Merge(sm2 *SourceMap) {
	sm.Sources = append(sm.Sources, sm2.Sources...)
	sm.Names = append(sm.Names, sm2.Names...)
	sm.Mappings += sm2.Mappings
	sm.SourcesContent = append(sm.SourcesContent, sm2.SourcesContent...)
}

// StackTrace get the stack trace
func StackTrace(jserr *v8go.JSError) string {

	entries := parseStackTrace(jserr.StackTrace)
	if entries == nil || len(entries) == 0 {
		return jserr.StackTrace
	}

	return jserr.StackTrace

	// fmt.Println("source map")
	// for k := range SourceMaps {
	// 	fmt.Println(k)
	// }

	// first := entries[0]
	// sm := SourceMaps[first.File].Bytes()
	// fmt.Println("source map", string(sm))

	// for _, entry := range entries {
	// 	err := entry.originalPositionFor(sm)
	// 	if err != nil {
	// 		fmt.Println(err)
	// 	}
	// }

	// utils.Dump(entries)
	// return jserr.StackTrace
}

func (entry *StackLogEntry) originalPositionFor(sourceMap []byte) error {

	smap, err := sourcemap.Parse(entry.File, sourceMap)
	if err != nil {
		return err
	}

	file, fn, line, col, ok := smap.Source(entry.Line, entry.Column)
	if !ok {
		return fmt.Errorf("no original position for %s:%d:%d", entry.File, entry.Line, entry.Column)
	}

	if !strings.HasPrefix(file, string(os.PathSeparator)) {
		file = filepath.Join(application.App.Root(), file)
	}

	file, _ = filepath.Abs(file)
	entry.File = file
	entry.Function = fn
	entry.Line = line
	entry.Column = col
	return nil
}

func parseStackTrace(trace string) []*StackLogEntry {
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
