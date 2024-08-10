package v8

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/fatih/color"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/gou/runtime/v8/objects/console"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

// Scripts loaded scripts
var Scripts = map[string]*Script{}

// Modules the scripts for modules
var Modules = map[string]Module{}

// ImportMap the import maps
var ImportMap = map[string][]Import{}

// RootScripts the scripts for studio
var RootScripts = map[string]*Script{}

// var importRe = regexp.MustCompile(`import\s*\t*\n*[^;]*;`)
var importRe = regexp.MustCompile(`import\s+\t*\n*(\*\s+as\s+\w+|\{[^}]+\}|\w+)\s+from\s+["']([^"']+)["'];?`)
var exportRe = regexp.MustCompile(`export\s+(default|function|class|const|var|let)\s+`)

var internalKeepModuleSuffixes = []string{"/yao.ts", "/yao", "/gou", "/gou.ts"}
var internalKeepModules = []string{"@yao", "@yaoapps", "@yaoapp", "@gou"}

// the lock for the scripts
var syncLock = sync.Mutex{}

// GetModuleName get the module name
func GetModuleName(file string) string {
	replaces := []string{"@", "/", ".", "-", "[", "]", "(", ")", "{", "}", ":", ",", ";", " ", "\t", "\n", "\r"}
	for _, replace := range replaces {
		file = strings.ReplaceAll(file, replace, "_")
	}
	return file
}

// NewScript create a new script
func NewScript(file string, id string, timeout ...time.Duration) *Script {

	t := time.Duration(0)
	if len(timeout) > 0 {
		t = timeout[0]
	}

	return &Script{
		ID:      id,
		File:    file,
		Timeout: t,
	}
}

// Open open the script
func (script *Script) Open(source []byte) error {
	var err error = nil
	if strings.HasSuffix(script.File, ".ts") {
		source, err = TransformTS(script.File, source)
		if err != nil {
			return err
		}
	}
	script.Source = string(source)
	return nil
}

// MakeScript make a script from source
func MakeScript(source []byte, file string, timeout time.Duration, isroot ...bool) (*Script, error) {
	syncLock.Lock()
	defer syncLock.Unlock()
	script := NewScript(file, file, timeout)
	err := script.Open(source)
	if err != nil {
		return nil, err
	}
	script.Root = false
	if len(isroot) > 0 {
		script.Root = isroot[0]
	}
	return script, nil
}

// Load load the script
func Load(file string, id string) (*Script, error) {
	source, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}
	script, err := MakeScript(source, file, 5*time.Second, false)
	if err != nil {
		return nil, err
	}
	Scripts[id] = script
	return script, nil
}

// LoadRoot load the script with root privileges
func LoadRoot(file string, id string) (*Script, error) {
	source, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}
	script, err := MakeScript(source, file, 5*time.Second, true)
	if err != nil {
		return nil, err
	}
	RootScripts[id] = script
	return script, nil
}

// CLearModules clear the modules cache
func CLearModules() {
	Modules = map[string]Module{}
	ImportMap = map[string][]Import{}
	clearSourceMaps()
}

// TransformTS transform the typescript
func TransformTS(file string, source []byte) ([]byte, error) {

	tsCode, err := tsImports(file, removeCommentsAndKeepLines(source))
	if err != nil {
		return nil, err
	}

	result := api.Transform(tsCode, api.TransformOptions{
		Loader:     api.LoaderTS,
		Target:     api.ESNext,
		Sourcefile: file,
		Sourcemap:  api.SourceMapExternal,
	})

	if len(result.Errors) > 0 {
		errors := []string{}
		for _, err := range result.Errors {
			errors = append(errors, fmt.Sprintf("%s", err.Text))
		}
		return nil, fmt.Errorf("transform ts code error: %v", strings.Join(errors, "\n"))
	}

	SourceMaps[file] = result.Map
	SourceCodes[file] = result.Code

	// Add the module source
	jsCode := result.Code

	// Add the import module
	if runtimeOption.Import {
		importCodes := []string{}
		if imports, has := ImportMap[file]; has {
			for _, imp := range imports {
				module, has := Modules[imp.AbsPath]
				if has {
					importCodes = append(importCodes, fmt.Sprintf("%s;const %s = %s;", module.Source, imp.Name, module.GlobalName))
				}
			}
		}
		if len(importCodes) > 0 {
			jsCode = []byte(strings.Join(importCodes, ";") + string(result.Code))
		}
	}

	return []byte(
		exportRe.ReplaceAllStringFunc(string(jsCode), func(m string) string {
			return strings.ReplaceAll(m, "export ", "")
		})), nil
}

type entry struct {
	absfile string
	source  string
	file    string
}

func removeCommentsAndKeepLines(code []byte) []byte {
	lines := strings.Split(string(code), "\n")
	for i, line := range lines {
		// Start with /*
		if strings.HasPrefix(strings.TrimSpace(line), "/*") {
			lines[i] = ""
			for {
				if strings.Contains(line, "*/") {
					break
				}
				i++
				line = lines[i]
				lines[i] = ""
			}
		}

		// Start with //
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			lines[i] = ""
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func getEntryPoints(file string, tsCode string, loaded map[string]bool) (string, []entry, error) {
	entryPoints := []entry{}
	root := application.App.Root()
	absFile := filepath.Join(root, file)

	tsCode, imports, err := replaceImportCode(file, []byte(tsCode))
	if err != nil {
		return "", nil, err
	}
	ImportMap[file] = imports
	entryPoints = append(entryPoints, entry{file: file, absfile: absFile, source: tsCode})

	for _, imp := range imports {
		if loaded[imp.Path] {
			continue
		}
		loaded[imp.Path] = true
		source, err := application.App.Read(imp.Path)
		if err != nil {
			return "", nil, err
		}

		_, subEntryPoints, err := getEntryPoints(imp.Path, string(source), loaded)
		if err != nil {
			return "", nil, err
		}

		entryPoints = append(entryPoints, subEntryPoints...)
	}
	return tsCode, entryPoints, nil

}

func loadModule(file string, tsCode string) error {

	errors := []string{}
	root := application.App.Root()
	absFile := filepath.Join(root, file)

	// Check if the module loaded
	if _, has := Modules[absFile]; has {
		return nil
	}

	globalName := GetModuleName(file)
	entryPoints := []entry{}
	loaded := map[string]bool{}
	tsCode, entryPoints, err := getEntryPoints(file, tsCode, loaded)
	if err != nil {
		return err
	}

	files := []string{}
	codes := map[string]string{}
	for _, entry := range entryPoints {
		files = append(files, entry.absfile)
		codes[entry.absfile] = entry.source
	}

	paths := strings.Split(file, string(os.PathSeparator))
	dir := filepath.Join(root, paths[0]) // <app_root>/scripts, <app_root>/services, etc..
	outdir := filepath.Join(string(os.PathSeparator), "outdir")

	result := api.Build(api.BuildOptions{
		EntryPoints: files,
		Bundle:      true,
		Write:       false,
		Target:      api.ESNext,
		GlobalName:  globalName,
		Loader: map[string]api.Loader{
			".ts": api.LoaderTS,
		},
		Sourcemap: api.SourceMapExternal,
		Outbase:   dir,
		Outdir:    outdir,
		Plugins: []api.Plugin{
			{
				Name: "custom-import-plugin",
				Setup: func(build api.PluginBuild) {
					build.OnLoad(api.OnLoadOptions{Filter: `.*\.ts$`}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
						contents := codes[args.Path]
						return api.OnLoadResult{
							Contents: &contents,
							Loader:   api.LoaderTS,
						}, nil
					})
				},
			},
		},
	})

	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			errors = append(errors, err.Text)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("transform module error: %v.\n%s", strings.Join(errors, "\n"), tsCode)
	}

	if len(result.OutputFiles) > 1 {
		for _, out := range result.OutputFiles {
			if strings.HasSuffix(out.Path, ".js.map") {
				key := strings.TrimPrefix(strings.ReplaceAll(out.Path, ".js.map", ".ts"), outdir)
				key = filepath.Join(dir, key)
				ModuleSourceMaps[key] = out.Contents

			} else if strings.HasSuffix(out.Path, ".js") {
				key := strings.TrimPrefix(strings.ReplaceAll(out.Path, ".js", ".ts"), outdir)
				key = filepath.Join(dir, key)
				Modules[key] = Module{
					File:       file,
					GlobalName: globalName,
					Source:     string(out.Contents),
				}
			}
		}
	}

	return nil
}

func tsImports(file string, source []byte) (string, error) {

	err := loadModule(file, string(source))
	if err != nil {
		return "", err
	}

	tsCode := importRe.ReplaceAllStringFunc(string(source), func(m string) string { // Remove the import as comments
		lines := strings.Split(m, "\n")
		for i, line := range lines {
			lines[i] = "// " + line
		}
		return strings.Join(lines, "\n")
	})

	return tsCode, nil
}

func replaceImportCode(file string, source []byte) (string, []Import, error) {
	var err error = nil
	errors := []string{}
	imports := []Import{}
	tsCode := importRe.ReplaceAllStringFunc(string(source), func(m string) string {
		matches := importRe.FindStringSubmatch(m)
		if len(matches) == 3 {
			importClause, importPath := matches[1], matches[2]

			// Filter the internal keep modules
			for _, keep := range internalKeepModuleSuffixes {
				if strings.HasSuffix(importPath, keep) {
					lines := strings.Split(m, "\n")
					for i, line := range lines {
						lines[i] = "// " + line

					}
					return strings.Join(lines, "\n")
				}
			}
			for _, keep := range internalKeepModules {
				if strings.HasPrefix(importPath, keep) {
					lines := strings.Split(m, "\n")
					for i, line := range lines {
						lines[i] = "// " + line
					}
					return strings.Join(lines, "\n")
				}
			}

			relImportPath, err := getImportPath(file, importPath)
			if err != nil {
				errors = append(errors, err.Error())
				return m
			}

			absImportPath := filepath.Join(application.App.Root(), relImportPath)

			name := strings.TrimSpace(importClause)
			if strings.Index(importClause, "*") >= 0 {
				arr := strings.Split(importClause, " as ")
				if len(arr) == 2 {
					name = strings.TrimSpace(arr[1])
				}
			} else if strings.Index(importClause, " as ") >= 0 {
				name = strings.ReplaceAll(importClause, " as ", ":")
			}

			imports = append(imports, Import{
				Name:    name,
				Path:    relImportPath,
				AbsPath: absImportPath,
				Clause:  importClause,
			})
			return fmt.Sprintf(`import %s from "%s";`, importClause, absImportPath)
		}
		return m
	})

	if len(errors) > 0 {
		err = fmt.Errorf("transform ts code error: %v", strings.Join(errors, "\n"))
	}

	return tsCode, imports, err
}

func getImportPath(file string, path string) (string, error) {

	var tsfile string
	var fromTsConfig bool = false
	if runtimeOption.TSConfig != nil {
		var err error
		tsfile, fromTsConfig, err = runtimeOption.TSConfig.GetFileName(path)
		if err != nil {
			return "", err
		}
		if fromTsConfig {
			file = tsfile
		}
	}

	if !fromTsConfig {
		relpath := filepath.Dir(file)
		file = filepath.Join(relpath, path)
	}

	if !strings.HasSuffix(path, ".ts") {
		if exist, _ := application.App.Exists(file + ".ts"); exist {
			file = file + ".ts"
			return file, nil

		} else if exist, _ := application.App.Exists(filepath.Join(path, "index.ts")); exist {
			file = file + "index.ts"
			return file, nil
		}
	}

	if exist, _ := application.App.Exists(file); !exist {
		return "", fmt.Errorf("file %s not exists", file)
	}

	return file, nil
}

// Transform the javascript
func Transform(source string, globalName string) string {
	result := api.Transform(source, api.TransformOptions{
		Loader:     api.LoaderJS,
		Format:     api.FormatIIFE,
		GlobalName: globalName,
	})
	return string(result.Code)
}

// Select a script
func Select(id string) (*Script, error) {
	script, has := Scripts[id]
	if !has {
		return nil, fmt.Errorf("script %s not exists", id)
	}
	return script, nil
}

// SelectRoot a script with root privileges
func SelectRoot(id string) (*Script, error) {

	script, has := RootScripts[id]
	if has {
		return script, nil
	}

	script, has = Scripts[id]
	if !has {
		return nil, fmt.Errorf("script(root) %s not exists", id)
	}

	return script, nil
}

// NewContext create a new context
func (script *Script) NewContext(sid string, global map[string]interface{}) (*Context, error) {

	timeout := script.Timeout
	if timeout == 0 {
		timeout = time.Duration(runtimeOption.ContextTimeout) * time.Millisecond
	}

	// The performance mode
	if runtimeOption.Mode == "performance" {

		runner, err := dispatcher.Select(time.Duration(runtimeOption.DefaultTimeout) * time.Millisecond)
		if err != nil {
			return nil, err
		}

		runner.global = global
		runner.sid = sid
		ctx, err := runner.Context()
		if err != nil {
			return nil, err
		}

		return &Context{
			ID:          script.ID,
			Sid:         sid,
			Data:        global,
			Root:        script.Root,
			Timeout:     timeout,
			Runner:      runner,
			Context:     ctx,
			SourceRoots: script.SourceRoots,
		}, nil

	}

	iso, err := SelectIsoStandard(time.Duration(runtimeOption.DefaultTimeout) * time.Millisecond)
	if err != nil {
		return nil, err
	}

	ctx := v8go.NewContext(iso, iso.Template)

	// Create instance of the script
	instance, err := iso.CompileUnboundScript(script.Source, script.File, v8go.CompileOptions{})
	if err != nil {
		return nil, fmt.Errorf("scripts.%s %s", script.ID, err.Error())
	}
	v, err := instance.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("scripts.%s %s", script.ID, err.Error())
	}
	defer v.Release()

	// console.log("foo", "bar", 1, 2, 3, 4)
	err = console.New().Set("console", ctx)
	if err != nil {
		return nil, fmt.Errorf("scripts.%s %s", script.ID, err.Error())
	}

	return &Context{
		ID:            script.ID,
		Sid:           sid,
		Data:          global,
		Root:          script.Root,
		Timeout:       timeout,
		Isolate:       iso,
		Context:       ctx,
		UnboundScript: instance,
		SourceRoots:   script.SourceRoots,
	}, nil
}

// Exec execute the script
// the default mode is "standard" and the other value is "performance".
// the "standard" mode save memory but will run slower. can be used in most cases, especially in arm64 device.
// the "performance" mode need more memory but will run faster. can be used in high concurrency and large script.
func (script *Script) Exec(process *process.Process) interface{} {
	if runtimeOption.Mode == "performance" {
		return script.execPerformance(process)
	}
	return script.execStandard(process)
}

// execPerformance execute the script in performance mode
func (script *Script) execPerformance(process *process.Process) interface{} {

	runner, err := dispatcher.Select(time.Duration(runtimeOption.DefaultTimeout) * time.Millisecond)
	if err != nil {
		exception.New("scripts.%s.%s %s", 500, script.ID, process.Method, err.Error()).Throw()
		return nil
	}

	runner.method = process.Method
	runner.args = process.Args
	runner.global = process.Global
	runner.sid = process.Sid
	return runner.Exec(script)
}

// execStandard execute the script in standard mode
func (script *Script) execStandard(process *process.Process) interface{} {

	iso, err := SelectIsoStandard(time.Duration(runtimeOption.DefaultTimeout) * time.Millisecond)
	if err != nil {
		exception.New("scripts.%s.%s %s", 500, script.ID, process.Method, err.Error()).Throw()
		return nil
	}
	defer iso.Dispose()

	ctx := v8go.NewContext(iso, iso.Template)
	defer ctx.Close()

	// Next Version will support this, snapshot will be used in the next version
	// ctx, err := iso.Context()
	// if err != nil {
	// 	exception.New("scripts.%s.%s %s", 500, script.ID, process.Method, err.Error()).Throw()
	// 	return nil
	// }

	// Create instance of the script
	instance, err := iso.CompileUnboundScript(script.Source, script.File, v8go.CompileOptions{})
	if err != nil {
		exception.New("scripts.%s.%s %s", 500, script.ID, process.Method, err.Error()).Throw()
		return nil
	}
	v, err := instance.Run(ctx)
	if err != nil {
		return err
	}
	defer v.Release()

	// Set the global data
	global := ctx.Global()
	err = bridge.SetShareData(ctx, global, &bridge.Share{
		Sid:    process.Sid,
		Root:   script.Root,
		Global: process.Global,
	})
	if err != nil {
		log.Error("scripts.%s.%s %s", script.ID, process.Method, err.Error())
		exception.New("scripts.%s.%s %s", 500, script.ID, process.Method, err.Error()).Throw()
		return nil
	}

	// console.log("foo", "bar", 1, 2, 3, 4)
	err = console.New().Set("console", ctx)
	if err != nil {
		exception.New("scripts.%s.%s %s", 500, script.ID, process.Method, err.Error()).Throw()
		return nil
	}

	// Run the method
	jsArgs, err := bridge.JsValues(ctx, process.Args)
	if err != nil {
		log.Error("scripts.%s.%s %s", script.ID, process.Method, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil

	}
	defer bridge.FreeJsValues(jsArgs)

	jsRes, err := global.MethodCall(process.Method, bridge.Valuers(jsArgs)...)
	if err != nil {

		// Debug output the error stack
		if e, ok := err.(*v8go.JSError); ok {
			color.Red("%s\n\n", StackTrace(e, script.SourceRoots))
		}

		log.Error("scripts.%s.%s %s", script.ID, process.Method, err.Error())
		exception.Err(err, 500).Throw()
		return nil
	}

	goRes, err := bridge.GoValue(jsRes, ctx)
	if err != nil {
		log.Error("scripts.%s.%s %s", script.ID, process.Method, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}

	return goRes
}

// ContextTimeout get the context timeout
func (script *Script) ContextTimeout() time.Duration {
	if script.Timeout > 0 {
		return script.Timeout
	}
	return time.Duration(runtimeOption.ContextTimeout) * time.Millisecond
}
