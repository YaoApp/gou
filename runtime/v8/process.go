package v8

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func init() {
	process.Register("scripts", processScripts)
	process.Register("studio", processStudio)
}

// processScripts
func processScripts(process *process.Process) interface{} {

	script, err := Select(process.ID)
	if err != nil {
		exception.New("scripts.%s not loaded", 404, process.ID).Throw()
		return nil
	}

	return script.Exec(process)

	// script, err := Select(process.ID)
	// if err != nil {
	// 	exception.New("scripts.%s not loaded", 404, process.ID).Throw()
	// 	return nil
	// }

	// if runtimeOption.Mode == "normal" {
	// 	return runNormalMode(script, process.Sid, process.Global, process.Method, process.Args...)
	// }

	// ctx, err := script.NewContext(process.Sid, process.Global)
	// if err != nil {
	// 	message := fmt.Sprintf("scripts.%s failed to create context. %+v", process.ID, err)
	// 	log.Error("[V8] process error. %s", message)
	// 	exception.New(message, 500).Throw()
	// 	return nil
	// }
	// defer ctx.Close()

	// res, err := ctx.Call(process.Method, process.Args...)
	// if err != nil {
	// 	exception.New(err.Error(), 500).Throw()
	// }

	// return res
}

// wrk -t12 -c400 -d30s 'http://maxdev.yao.run/api/register/wechat/check/status?state=881119&sn=136-552-234'
// func runNormalMode(script *Script, sid string, data map[string]interface{}, method string, args ...interface{}) interface{} {

// 	// defer runtime.GC()
// 	// iso, err := SelectIso(2000 * time.Millisecond)
// 	// if err != nil {
// 	// 	return err
// 	// }
// 	// defer iso.Unlock()

// 	iso := v8go.NewIsolate()
// 	defer iso.Dispose()

// 	tmpl := MakeTemplate(iso)
// 	ctx := v8go.NewContext(iso, tmpl)
// 	defer ctx.Close()

// 	// v, err := context.RunScript(script.Source, script.File)

// 	instance, err := iso.CompileUnboundScript(script.Source, script.File, v8go.CompileOptions{})
// 	if err != nil {
// 		return err
// 	}

// 	v, err := instance.Run(ctx)
// 	if err != nil {
// 		return err
// 	}
// 	defer v.Release()

// 	global := ctx.Global()
// 	jsArgs, err := bridge.JsValues(ctx, args)
// 	if err != nil {
// 		return fmt.Errorf("%s.%s %s", script.ID, method, err.Error())
// 	}

// 	defer bridge.FreeJsValues(jsArgs)

// 	goData := map[string]interface{}{
// 		"SID":  sid,
// 		"ROOT": script.Root,
// 		"DATA": data,
// 	}

// 	jsData, err := bridge.JsValue(ctx, goData)
// 	if err != nil {
// 		return err
// 	}

// 	err = global.Set("__yao_data", jsData)
// 	if err != nil {
// 		return err
// 	}
// 	defer func() {
// 		if !jsData.IsNull() && !jsData.IsUndefined() {
// 			jsData.Release()
// 		}
// 	}()

// 	jsRes, err := global.MethodCall(method, bridge.Valuers(jsArgs)...)
// 	if err != nil {
// 		return fmt.Errorf("%s.%s %+v", script.ID, method, err)
// 	}

// 	goRes, err := bridge.GoValue(jsRes, ctx)
// 	if err != nil {
// 		return fmt.Errorf("%s.%s %s", script.ID, method, err.Error())
// 	}

// 	return goRes

// }

// processScripts scripts.ID.Method
func processStudio(process *process.Process) interface{} {

	script, err := SelectRoot(process.ID)
	if err != nil {
		exception.New("studio.%s not loaded", 404, process.ID).Throw()
		return nil
	}
	return script.Exec(process)

}
