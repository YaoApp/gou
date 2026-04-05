package api

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/yaoapp/kun/log"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xun"
)

// APIs is the loaded API list
var APIs = map[string]*API{}
var apisMu sync.RWMutex

// Load load the api and register it to the global APIs map
func Load(file, id string, guard ...string) (*API, error) {

	data, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}

	api, err := LoadSource(file, data, id, guard...)
	if err != nil {
		return nil, err
	}

	apisMu.Lock()
	APIs[id] = api
	apisMu.Unlock()

	return api, nil
}

// LoadSource parses API source data and returns an *API without modifying global state
func LoadSource(file string, data []byte, id string, guard ...string) (*API, error) {

	http := HTTP{}
	err := application.Parse(file, data, &http)
	if err != nil {
		log.Error("[API] Load %s Error: %s", id, err.Error())
		return nil, err
	}

	// Filesystem Router
	if http.Group == "" {
		http.Group = strings.ReplaceAll(strings.ToLower(id), ".", "/")
	}

	// Validate API
	uniquePathCheck := map[string]bool{}
	for _, path := range http.Paths {
		unique := fmt.Sprintf("%s.%s", path.Method, path.Path)
		if _, has := uniquePathCheck[unique]; has {
			log.Error("[API] Load %s is already registered", id)
			return nil, fmt.Errorf("[API] Load %s Error: is already registered", id)
		}
		uniquePathCheck[unique] = true
	}

	// Default Guard
	if http.Guard == "" && len(guard) > 0 {
		http.Guard = guard[0]
	}

	return &API{
		ID:   id,
		File: file,
		HTTP: http,
		Type: "http",
	}, nil
}

// Select select api
func Select(id string) *API {
	apisMu.RLock()
	api, has := APIs[id]
	apisMu.RUnlock()
	if !has {
		exception.New("[API] %s not loaded", 500, id).Throw()
	}
	return api
}

// SetRoutes set the api routes
func SetRoutes(router *gin.Engine, path string, allows ...string) {

	// Error handler
	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {

		var code = http.StatusInternalServerError

		if err, ok := recovered.(string); ok {
			c.JSON(code, xun.R{
				"code":    code,
				"message": fmt.Sprintf("%s", err),
			})
		} else if err, ok := recovered.(exception.Exception); ok {
			code = err.Code
			c.JSON(code, xun.R{
				"code":    code,
				"message": err.Message,
			})
		} else if err, ok := recovered.(*exception.Exception); ok {
			code = err.Code
			c.JSON(code, xun.R{
				"code":    code,
				"message": err.Message,
			})
		} else {
			c.JSON(code, xun.R{
				"code":    code,
				"message": fmt.Sprintf("%v", recovered),
			})
		}

		c.AbortWithStatus(code)
	}))

	// Load apis
	apisMu.RLock()
	snapshot := make([]*API, 0, len(APIs))
	for _, api := range APIs {
		snapshot = append(snapshot, api)
	}
	apisMu.RUnlock()

	for _, api := range snapshot {
		api.HTTP.Routes(router, path, allows...)
	}
}

// SetGuards set guards
func SetGuards(guards map[string]gin.HandlerFunc) {
	HTTPGuards = guards
}

// AddGuard add guard
func AddGuard(name string, guard gin.HandlerFunc) {
	HTTPGuards[name] = guard
}

// Reload reloads a single API definition from its file
func (api *API) Reload() (*API, error) {
	return Load(api.File, api.ID)
}

// FindHandler finds a handler by method and path from the route table
// Returns the API, Path, Handler, extracted parameters, and error
func FindHandler(method, path string) (*API, *Path, gin.HandlerFunc, map[string]string, error) {
	routeTable.mu.RLock()
	defer routeTable.mu.RUnlock()

	entry, params := routeTable.find(method, path)
	if entry == nil {
		return nil, nil, nil, nil, fmt.Errorf("route not found: %s %s", method, path)
	}

	handler := BuildHandler(entry.API.HTTP, *entry.Path)
	return entry.API, entry.Path, handler, params, nil
}

// ReloadAPIs reloads all API definitions from the specified directory
// This function is thread-safe and performs atomic replacement of both APIs and the route table
func ReloadAPIs(root string) error {
	exts := []string{"*.http.yao", "*.http.json", "*.http.jsonc"}

	exists, err := application.App.Exists(root)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	// Build into a local map — no lock needed, single-goroutine Walk callback
	newAPIs := make(map[string]*API)
	var loadErr error

	err = application.App.Walk(root, func(rootPath, file string, isdir bool) error {
		if isdir {
			return nil
		}

		id := strings.TrimPrefix(file, "/")
		id = strings.TrimPrefix(id, root+"/")
		id = strings.TrimPrefix(id, root)
		id = strings.TrimSuffix(id, filepath.Ext(id))
		id = strings.TrimSuffix(id, filepath.Ext(id))
		id = strings.ReplaceAll(id, "/", ".")

		data, err := application.App.Read(file)
		if err != nil {
			loadErr = err
			return nil
		}

		api, err := LoadSource(file, data, id)
		if err != nil {
			loadErr = err
			return nil
		}
		newAPIs[id] = api
		return nil
	}, exts...)

	if err != nil {
		return err
	}

	if loadErr != nil {
		log.Warn("[API] ReloadAPIs partial error: %s", loadErr.Error())
	}

	// Atomically replace global APIs and rebuild route table
	apisMu.Lock()
	APIs = newAPIs
	apisMu.Unlock()

	routeTable.mu.Lock()
	defer routeTable.mu.Unlock()

	routeTable.clear()
	for _, api := range newAPIs {
		for i := range api.HTTP.Paths {
			path := &api.HTTP.Paths[i]
			fullPath := buildFullPath(api.HTTP.Group, path.Path)

			entry := &RouteEntry{
				Pattern: fullPath,
				API:     api,
				Path:    path,
			}

			if hasPathParams(fullPath) {
				entry.Regex, entry.Params = compilePattern(fullPath)
			}

			routeTable.addEntry(path.Method, entry)
		}
	}

	return nil
}

// BuildRouteTable builds the route table from loaded APIs
// Should be called after loading APIs and before using FindHandler
func BuildRouteTable() {
	apisMu.RLock()
	snapshot := make(map[string]*API, len(APIs))
	for k, v := range APIs {
		snapshot[k] = v
	}
	apisMu.RUnlock()

	routeTable.mu.Lock()
	defer routeTable.mu.Unlock()

	routeTable.clear()
	for _, api := range snapshot {
		for i := range api.HTTP.Paths {
			path := &api.HTTP.Paths[i]
			fullPath := buildFullPath(api.HTTP.Group, path.Path)

			entry := &RouteEntry{
				Pattern: fullPath,
				API:     api,
				Path:    path,
			}

			if hasPathParams(fullPath) {
				entry.Regex, entry.Params = compilePattern(fullPath)
			}

			routeTable.addEntry(path.Method, entry)
		}
	}
}
