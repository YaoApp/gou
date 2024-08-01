package v8

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/gou/runtime/v8/objects/console"
	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

// ID is the runner id
type ID string

func (id ID) String() string {
	return strings.Split(string(id), "-")[0]
}

// Runner is the v8 runner
type Runner struct {
	id        ID
	iso       *v8go.Isolate
	ctx       *v8go.Context
	tmpl      *v8go.ObjectTemplate
	status    uint8
	signal    chan uint8
	chResp    chan interface{}
	keepalive bool
	script    *Script
	method    string
	sid       string
	args      []interface{}
	global    map[string]interface{}
	caches    map[string]*v8go.Object
}

const (
	// RunnerStatusInit is the runner status init
	RunnerStatusInit uint8 = iota

	// RunnerStatusRunning is the runner status running
	RunnerStatusRunning

	// RunnerStatusCleaning is the runner status cleaning
	RunnerStatusCleaning

	// RunnerStatusReady is the runner status ready
	RunnerStatusReady

	// RunnerStatusDestroy is the runner status destroy
	RunnerStatusDestroy

	// RunnerCommandDestroy is the runner command destroy
	RunnerCommandDestroy

	// RunnerCommandReset is the runner command reset
	RunnerCommandReset

	// RunnerCommandExec is the runner command exec
	RunnerCommandExec

	// RunnerCommandStatus is the runner command status
	RunnerCommandStatus
)

// NewRunner create a new v8 runner
func NewRunner(keepalive bool) *Runner {
	return &Runner{
		id:        ID(uuid.New().String()),
		iso:       nil,
		ctx:       nil,
		signal:    make(chan uint8, 2),
		keepalive: keepalive,
		status:    RunnerStatusInit,
	}
}

// Start start the v8 runner
func (runner *Runner) Start(ready chan bool) error {

	// Set the status to free
	if runner.status != RunnerStatusInit {
		err := fmt.Errorf("[runner] you can't start a runner with status: [%d]", runner.status)
		log.Error(err.Error())
		return err
	}

	iso := v8go.YaoNewIsolate()
	tmpl := MakeTemplate(iso)
	ctx := v8go.NewContext(iso, tmpl)
	runner.iso = iso
	runner.ctx = ctx
	runner.tmpl = tmpl
	runner.status = RunnerStatusReady
	if runner.keepalive {
		dispatcher.online(runner)
	}

	ticker := time.NewTicker(time.Millisecond * 50)

	ready <- true

	// Command loop
	for {
		select {
		case <-ticker.C:
			break

		case signal := <-runner.signal:
			switch signal {

			case RunnerCommandReset:
				runner.reset()
				break

			case RunnerCommandExec:
				runner.exec()
				break

			case RunnerCommandDestroy:
				runner.destroy()
				return nil

			default:
				log.Warn("runner unknown signal: %d", signal)
			}

		}
	}
}

// Destroy send a destroy signal to the v8 runner
func (runner *Runner) Destroy(cb func()) {
	runner.signal <- RunnerCommandDestroy
}

// Reset send a reset signal to the v8 runner
func (runner *Runner) Reset() {
	runner.signal <- RunnerCommandReset
}

// Exec send a script to the v8 runner to execute
func (runner *Runner) Exec(script *Script) interface{} {

	runner.status = RunnerStatusRunning
	runner.script = script
	runner.chResp = make(chan interface{})
	log.Debug(fmt.Sprintf("2.  [%s] Exec script %s.%s. status:%d, keepalive:%v, signal:%d", runner.id, script.ID, runner.method, runner.status, runner.keepalive, len(runner.signal)))

	runner.signal <- RunnerCommandExec
	select {
	case res := <-runner.chResp:
		return res
	}
}

// Context get the context
func (runner *Runner) Context() (*v8go.Context, error) {
	return runner.ctx, nil
}

func (runner *Runner) exec() {

	defer func() {
		go func() {
			if !runner.keepalive {
				log.Debug(fmt.Sprintf("3.1 [%s] Send a destroy signal to the v8 runner. status:%d, keepalive:%v", runner.id, runner.status, runner.keepalive))
				runner.signal <- RunnerCommandDestroy
				log.Debug(fmt.Sprintf("3.2 [%s] Send a destroy signal to the v8 runner. sstatus:%d, keepalive:%v (done)", runner.id, runner.status, runner.keepalive))
				return
			}

			log.Debug(fmt.Sprintf("3.1 [%s] Send a reset signal to the v8 runner. status:%d, keepalive:%v", runner.id, runner.status, runner.keepalive))
			runner.signal <- RunnerCommandReset
			log.Debug(fmt.Sprintf("3.2 [%s] Send a reset signal to the v8 runner. status:%d, keepalive:%v (done)", runner.id, runner.status, runner.keepalive))
		}()
	}()

	// runner.chResp <- "OK"
	runner._exec()
}

func (runner *Runner) _exec() {

	// Create instance of the script
	instance, err := runner.iso.CompileUnboundScript(runner.script.Source, runner.script.File, v8go.CompileOptions{})
	if err != nil {
		runner.chResp <- err
		return
	}
	v, err := instance.Run(runner.ctx)
	if err != nil {
		runner.chResp <- err
		return
	}
	defer v.Release()

	// console.log("foo", "bar", 1, 2, 3, 4)
	err = console.New().Set("console", runner.ctx)
	if err != nil {
		runner.chResp <- err
		return
	}

	// Set the global data
	global := runner.ctx.Global()
	err = bridge.SetShareData(runner.ctx, global, &bridge.Share{
		Sid:    runner.sid,
		Root:   runner.script.Root,
		Global: runner.global,
	})
	if err != nil {
		runner.chResp <- err
		return
	}

	// Run the method
	jsArgs, err := bridge.JsValues(runner.ctx, runner.args)
	if err != nil {
		runner.chResp <- err
		return

	}
	defer bridge.FreeJsValues(jsArgs)

	jsRes, err := global.MethodCall(runner.method, bridge.Valuers(jsArgs)...)
	if err != nil {
		if e, ok := err.(*v8go.JSError); ok {
			color.Red("%s\n\n", StackTrace(e, runner.script.SourceRoots))
		}
		runner.chResp <- err
		return
	}

	goRes, err := bridge.GoValue(jsRes, runner.ctx)
	if err != nil {
		runner.chResp <- err
		return
	}

	runner.chResp <- goRes
}

// destroy the runner
func (runner *Runner) destroy() {

	log.Debug(fmt.Sprintf("4.  [%s] destroy the runner. status:%d, keepalive:%v ", runner.id, runner.status, runner.keepalive))
	log.Debug(fmt.Sprintf("--- [%s] end -----------------", runner.id))

	dispatcher.total--
	runner.status = RunnerStatusDestroy
	if runner.signal != nil {
		close(runner.signal)
	}

	runner.ctx.Close()
	runner.iso.Dispose()
	runner.iso = nil
	runner.ctx = nil
	runner.caches = nil
	runner.tmpl = nil
	runner = nil
}

// reset the runner
func (runner *Runner) reset() {

	log.Debug(fmt.Sprintf("4.  [%s] reset the runner. status:%d, keepalive:%v ", runner.id, runner.status, runner.keepalive))
	log.Debug(fmt.Sprintf("--- [%s] end -----------------", runner.id))
	runner.status = RunnerStatusCleaning

	runner.ctx.Close()
	runner.ctx = v8go.NewContext(runner.iso, runner.tmpl)

	if !runner.health() {
		runner.signal <- RunnerCommandDestroy
		dispatcher.create()
		return
	}

	// Set the status to free
	runner.status = RunnerStatusReady
	dispatcher.online(runner)

}

func (runner *Runner) health() bool {
	if runner.status == RunnerStatusDestroy {
		return true
	}
	runner.status = RunnerStatusCleaning
	stat := runner.iso.GetHeapStatistics()
	// utils.Dump(stat)

	log.Trace("[runner] [%s] health check. HeapStatistics:%d, HeapSizeRelease:%d", runner.id, stat.TotalHeapSize-stat.UsedHeapSize, runtimeOption.HeapSizeRelease)
	if stat.TotalHeapSize-stat.UsedHeapSize < runtimeOption.HeapSizeRelease || stat.NumberOfNativeContexts > 200 {
		log.Trace("[runner] [%s] health check. HeapStatistics: %d < %d Restart || %d", runner.id, stat.TotalHeapSize-stat.UsedHeapSize, runtimeOption.HeapSizeRelease, stat.NumberOfNativeContexts)
		return false
	}
	runner.status = RunnerStatusReady
	return true
}
