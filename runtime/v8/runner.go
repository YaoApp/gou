package v8

import (
	"fmt"

	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

// Runner is the v8 runner
type Runner struct {
	iso    *v8go.Isolate
	ctx    *v8go.Context
	status uint8
	signal chan uint8
	kind   string
}

var tempCount = 0
var cacheCount = 0

const (
	// RunnerStatusInit is the runner status init
	RunnerStatusInit uint8 = iota

	// RunnerStatusRunning is the runner status running
	RunnerStatusRunning

	// RunnerStatusCleaning is the runner status cleaning
	RunnerStatusCleaning

	// RunnerStatusFree is the runner status free
	RunnerStatusFree

	// RunnerStatusDestroy is the runner status destroy
	RunnerStatusDestroy

	// RunnerCommandDestroy is the runner command destroy
	RunnerCommandDestroy

	// RunnerCommandClean is the runner command clean
	RunnerCommandClean

	// RunnerCommandReset is the runner command reset
	RunnerCommandReset

	// RunnerCommandStatus is the runner command status
	RunnerCommandStatus
)

// NewRunner create a new v8 runner
func NewRunner() *Runner {
	iso := v8go.YaoNewIsolate()
	tmpl := MakeTemplate(iso)
	ctx := v8go.NewContext(iso, tmpl)
	return &Runner{iso: iso, ctx: ctx, signal: make(chan uint8, 1), status: RunnerStatusInit}
}

func (runner *Runner) key() string {
	return fmt.Sprintf("%p", runner)
}

// Start start the v8 runner
func (runner *Runner) Start(ready func(error)) error {

	// Set the status to free
	if runner.status != RunnerStatusInit {
		if ready != nil {
			ready(fmt.Errorf("[runner] you can't start a runner with status: [%d]", runner.status))
		}
		return fmt.Errorf("[runner] you can't start a runner with status: [%d]", runner.status)
	}

	runner.status = RunnerStatusFree
	dispatcher.Register(runner)

	if ready != nil {
		ready(nil)
	}

	// Command loop
	for {
		select {
		case signal := <-runner.signal:
			switch signal {

			case RunnerCommandClean:
				runner.clean()
				break

			case RunnerCommandReset:
				runner.reset()
				break

			case RunnerCommandDestroy:
				runner.destory()
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
func (runner *Runner) Reset(cb func()) {
	runner.signal <- RunnerCommandReset
}

// Exec send a script to the v8 runner to execute
func (runner *Runner) Exec(script *Script, method string, args ...interface{}) interface{} {
	runner.status = RunnerStatusRunning
	dispatcher.UpdateStatus(runner, RunnerStatusRunning)

	if runner.kind == "temp" {
		close(runner.signal)
		runner.ctx.Close()
		runner.iso.Dispose()
		runner.iso = nil
		runner.ctx = nil
		runner = nil
		return fmt.Sprintf("runner exec temp script: %s.%s ", script.ID, method)
	}

	go func() { runner.signal <- RunnerCommandReset }()
	return fmt.Sprintf("runner exec script: %s.%s ", script.ID, method)
}

// Context get the context
func (runner *Runner) Context() (*v8go.Context, error) {
	return runner.ctx, nil
}

// destory the runner
func (runner *Runner) destory() {
	runner.status = RunnerStatusDestroy
	dispatcher.Unregister(runner)
	close(runner.signal)

	runner.ctx.Close()
	runner.iso.Dispose()
	runner.iso = nil
	runner.ctx = nil
	runner = nil
}

// reset the runner
func (runner *Runner) reset() {

	runner.status = RunnerStatusDestroy
	dispatcher.UpdateStatus(runner, RunnerStatusDestroy)

	runner.ctx.Close()
	runner.iso.Dispose()
	runner.iso = v8go.YaoNewIsolate()
	runner.ctx = v8go.NewContext(runner.iso, MakeTemplate(runner.iso))
	// log.Info("[runner] reset the runner: [%p]", runner)

	// Set the status to free
	runner.status = RunnerStatusFree
	dispatcher.UpdateStatus(runner, RunnerStatusFree)

}

func (runner *Runner) clean() {
	runner.status = RunnerStatusCleaning
	runner.ctx.Close()
	runner.ctx = v8go.NewContext(runner.iso, MakeTemplate(runner.iso))
	// log.Info("[runner] clean the runner: [%p]", runner)

	// Set the status to free
	runner.status = RunnerStatusFree
	dispatcher.UpdateStatus(runner, RunnerStatusFree)

}
