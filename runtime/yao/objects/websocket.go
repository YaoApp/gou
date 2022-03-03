package objects

// on()
// push()
// send()
// close()
// WebSocket(url, protocols)

// // WebSocket the websocket object
// type WebSocket struct{}

// // NewWebSocket create a new websocket object
// func NewWebSocket() *WebSocket {
// 	return &WebSocket{}
// }

// // AddEventListener  js call AddEventListener method
// func (ws *WebSocket) AddEventListener() {}

// // Send send message to a websocket connection
// func (ws *WebSocket) Send() {}

// // Close close a websocket connection
// func (ws *WebSocket) Close() {}

// // ExportObject Export as a javascript Object
// func (ws *WebSocket) ExportObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
// 	tmpl := v8go.NewObjectTemplate(iso)
// 	tmpl.Set("send", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
// 		protocol, err := info.This().Get("protocol")
// 		fmt.Println("Send is called", protocol, err)
// 		return protocol
// 	}))
// 	return tmpl
// }

// // ExportFunction Export as a javascript function
// func (ws *WebSocket) ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {

// 	object := ws.ExportObject(iso)

// 	inst := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
// 		args := info.Args()
// 		this, err := object.NewInstance(info.Context())
// 		if err != nil {
// 			return iso.ThrowException(JSError(info.Context(), err.Error(), 500))
// 		}

// 		if len(args) < 1 {
// 			return iso.ThrowException(JSError(info.Context(), "Missing parameters", 400))
// 		}

// 		err = this.Set("protocol", args[0])
// 		if err != nil {
// 			fmt.Println("ERROR:", err)
// 		}

// 		err = this.Set("create", time.Now().String())
// 		if err != nil {
// 			fmt.Println("ERROR:", err)
// 		}

// 		return this.Value
// 	})

// 	return inst
// }

// // JSError Return js error object
// func JSError(ctx *v8go.Context, message string, code int) *v8go.Value {

// 	global := ctx.Global()
// 	errorObj, _ := global.Get("Error")
// 	if errorObj.IsFunction() {
// 		fn, _ := errorObj.AsFunction()
// 		c, _ := v8go.NewValue(ctx.Isolate(), uint32(code))
// 		m, _ := v8go.NewValue(ctx.Isolate(), message)
// 		v, _ := fn.Call(c, m)
// 		obj, _ := v.AsObject()
// 		obj.Set("code", uint32(code))
// 		return v
// 	}

// 	tmpl := v8go.NewObjectTemplate(ctx.Isolate())
// 	inst, _ := tmpl.NewInstance(ctx)
// 	inst.Set("message", message)
// 	inst.Set("code", uint32(code))
// 	return inst.Value
// }
