package websocket

import (
	"fmt"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/gou/websocket"
	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

// WebSocket Javascript WebSocket
type WebSocket struct{}

// New create a new WebSocket object
func New() *WebSocket {
	return &WebSocket{}
}

// ExportObject Export as a WebSocket Object
func (ws *WebSocket) ExportObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
	tmpl := v8go.NewObjectTemplate(iso)
	tmpl.Set("push", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {

		jsURL, err := info.This().Get("url")
		if err != nil {
			msg := fmt.Sprintf("WebSocket url: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		if !jsURL.IsString() {
			msg := fmt.Sprintf("WebSocket url: %s", "is not a string")
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		if len(info.Args()) < 1 {
			msg := fmt.Sprintf("WebSocket args: %s", "Missing parameters")
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		jsMessage := info.Args()[0]
		if err != nil {
			msg := fmt.Sprintf("WebSocket message: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		if !jsMessage.IsString() {
			msg := fmt.Sprintf("WebSocket message: %s", "is not a string")
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		protocols := []string{}
		if info.This().Has("protocols") {
			jsProtocols, err := info.This().Get("protocols")
			if err != nil {
				msg := fmt.Sprintf("WebSocket protocols: %s", err.Error())
				log.Error("%s", msg)
				return bridge.JsException(info.Context(), msg)
			}

			if obj, err := jsProtocols.AsObject(); err == nil {
				count := obj.InternalFieldCount()
				for i := uint32(0); i < count; i++ {
					protocols = append(protocols, obj.GetInternalField(i).String())
				}
			}
			log.Warn("WebSocket protocols: is not an object")
		}

		conn, err := websocket.NewWebSocket(jsURL.String(), protocols)
		if err != nil {
			msg := fmt.Sprintf("WebSocket connection: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		err = websocket.Push(conn, jsMessage.String())
		if err != nil {
			msg := fmt.Sprintf("WebSocket push: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		return v8go.Undefined(iso)
	}))
	return tmpl
}

// ExportFunction Export as a javascript WebSocket function
func (ws *WebSocket) ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {
	object := ws.ExportObject(iso)
	tmpl := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			msg := fmt.Sprintf("WebSocket args: %s", "Missing parameters")
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		this, err := object.NewInstance(info.Context())
		if err != nil {
			msg := fmt.Sprintf("WebSocket: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		this.Set("url", args[0].String())
		var jsProtocols *v8go.Object
		tmpl := v8go.NewObjectTemplate(info.Context().Isolate())
		if len(args) > 1 {
			tmpl.SetInternalFieldCount(uint32(len(args) - 1))
			jsProtocols, _ = tmpl.NewInstance(info.Context())
			for i, v := range args[1:] {
				jsProtocols.SetInternalField(uint32(i), v.String())
			}
		}
		this.Set("protocols", jsProtocols)
		return this.Value
	})

	return tmpl

}

// on()
// push()
// send()
// close()
// push()
// WebSocket(url, protocols)
