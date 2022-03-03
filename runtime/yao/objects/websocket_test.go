package objects

// func TestWebsocketSend(t *testing.T) {
// 	iso := v8go.NewIsolate()
// 	defer iso.Dispose()

// 	ws := NewWebSocket()
// 	global := v8go.NewObjectTemplate(iso)
// 	global.Set("WebSocket", ws.ExportFunction(iso))

// 	ctx := v8go.NewContext(iso, global)
// 	defer ctx.Close()

// 	v, err := ctx.RunScript(`new WebSocket("ws");`, "")
// 	assert.Nil(t, err)
// 	utils.Dump(v)

// 	v, err = ctx.RunScript(`new WebSocket("wss");`, "")
// 	assert.Nil(t, err)
// 	utils.Dump(v)

// 	v, err = ctx.RunScript(`var test = ()=>{var xxx = new WebSocket("xxx");  xxx.send();};test();`, "")
// 	assert.Nil(t, err)

// 	v, err = ctx.RunScript(`
// 	var test = ()=>{
// 		var res = 1
// 		try {
// 			res = new WebSocket();

// 		} catch(err) {
// 			res = err
// 		}
// 		return res
// 	}
// 	test();
// 	`, "")
// 	utils.Dump(err)
// 	utils.Dump(v)
// }
