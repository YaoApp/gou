package store

import "rogchap.com/v8go"

var caches = New()

// NewContext create a new context
func NewContext(isolate, script string, ctx *v8go.Context) *Context {
	return &Context{
		isolate: isolate,
		script:  script,
		Context: ctx,
	}
}

// Release release the context
func (ctx *Context) Release() error {
	// Remove the context from cache and release the context
	RemoveContextCache(ctx.isolate, ctx.script)
	return nil
}

// GetContextFromCache get the context from cache
func GetContextFromCache(isolate, script string) (*Context, bool) {

	cache, has := caches.Get(isolate)
	if !has {
		return nil, false
	}

	ctx, has := cache.(*Cache).contexts[script]
	if !has {
		return nil, false
	}

	return ctx, true
}

// SetContextCache set the context to cache
func SetContextCache(isolate, script string, ctx *Context) {

	cache, has := caches.Get(isolate)
	if !has {
		cache = &Cache{
			key:      isolate,
			contexts: map[string]*Context{},
		}
	}
	cache.(*Cache).contexts[script] = ctx
	caches.Add(cache)
}

// RemoveContextCache remove the context cache
func RemoveContextCache(isolate, script string) {
	cache, has := caches.Get(isolate)
	if !has {
		return
	}

	ctx, has := cache.(*Cache).contexts[script]
	if !has {
		return
	}

	ctx.Context.Close()
	ctx.Context = nil
	ctx = nil
	delete(cache.(*Cache).contexts, script)
	return
}

// MakeIsolateCache make the isolate cache
func MakeIsolateCache(isolate string) {
	caches.Add(&Cache{
		key:      isolate,
		contexts: map[string]*Context{},
	})
}

// CleanIsolateCache clean the isolate cache
func CleanIsolateCache(isolate string) {
	cache, has := caches.Get(isolate)
	if !has {
		return
	}
	cache.Dispose()
	caches.Remove(isolate)
}
