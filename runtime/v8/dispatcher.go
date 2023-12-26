package v8

import (
	"fmt"
	"sync"
	"time"

	"github.com/yaoapp/kun/log"
)

// RunnerMap is a runner map
type RunnerMap struct {
	data  map[*Runner]uint8
	mutex *sync.RWMutex
}

// Dispatcher is a runner dispatcher
type Dispatcher struct {
	availables RunnerMap
	runners    RunnerMap
	count      int
	min        int
	max        int
	stoped     bool
}

// the global dispatcher instance
// initialize when the v8 start
var dispatcher *Dispatcher = nil

// NewDispatcher is a runner dispatcher
func NewDispatcher(min, max int) *Dispatcher {

	// Test the min and max
	min = 50
	max = 150
	return &Dispatcher{
		availables: RunnerMap{data: make(map[*Runner]uint8), mutex: &sync.RWMutex{}},
		runners:    RunnerMap{data: make(map[*Runner]uint8), mutex: &sync.RWMutex{}},
		min:        min,
		max:        max,
		stoped:     false,
	}
}

// Start start the v8 mannager
func (dispatcher *Dispatcher) Start() error {
	for i := 0; i < dispatcher.min; i++ {
		runner := NewRunner()
		go runner.Start(nil)
	}
	log.Info("[dispatcher] the dispatcher is started. runners %d", dispatcher.count)
	return nil
}

// Stop stop the v8 mannager
func (dispatcher *Dispatcher) Stop() {
	dispatcher.stoped = true
	defer func() { dispatcher.stoped = false }()
	for runner := range dispatcher.runners.data {
		runner.signal <- RunnerCommandDestroy
	}
}

// Register register a new v8 runner
func (dispatcher *Dispatcher) Register(runner *Runner) {
	dispatcher.runners.mutex.Lock()
	defer dispatcher.runners.mutex.Unlock()
	dispatcher.runners.data[runner] = runner.status
	dispatcher.count = len(dispatcher.runners.data)
	if runner.status == RunnerStatusFree {
		dispatcher.online(runner)
	}
}

// Unregister unregister a v8 runner
func (dispatcher *Dispatcher) Unregister(runner *Runner) {
	if runner.status == RunnerStatusRunning {
		//  TODO: send a command to the runner
		log.Error("[dispatcher] you can't unregister a running runner")
		return
	}

	dispatcher.runners.mutex.Lock()
	defer dispatcher.runners.mutex.Unlock()
	delete(dispatcher.runners.data, runner)
	dispatcher.count = len(dispatcher.runners.data)
}

// UpdateStatus update the v8 runner status
func (dispatcher *Dispatcher) UpdateStatus(runner *Runner, status uint8) {
	dispatcher.runners.mutex.Lock()
	defer dispatcher.runners.mutex.Unlock()
	dispatcher.runners.data[runner] = status

	// log.Info("[dispatcher] update runner %p status %d", runner, status)

	if status == RunnerStatusFree {
		dispatcher.online(runner)
		return
	}
	dispatcher.offline(runner)
}

func (dispatcher *Dispatcher) offline(runner *Runner) {
	dispatcher.availables.mutex.Lock()
	defer dispatcher.availables.mutex.Unlock()
	delete(dispatcher.availables.data, runner)
	log.Info("[dispatcher] runner %p offline %d (%d/%d)", runner, len(dispatcher.availables.data), tempCount, cacheCount)

	// Create a new runner if the free runners are less than max size
	if dispatcher.count < dispatcher.max {
		go NewRunner().Start(nil)
	}
}

func (dispatcher *Dispatcher) online(runner *Runner) {
	dispatcher.availables.mutex.Lock()
	defer dispatcher.availables.mutex.Unlock()
	dispatcher.availables.data[runner] = runner.status
	log.Info("[dispatcher] runner %p online %d (%d/%d)", runner, len(dispatcher.availables.data), tempCount, cacheCount)
}

// Select select a free v8 runner
func (dispatcher *Dispatcher) Select(timeout time.Duration) (*Runner, error) {
	if dispatcher.stoped {
		return nil, fmt.Errorf("[dispatcher] the dispatcher is stoped")
	}

	dispatcher.availables.mutex.RLock()
	defer dispatcher.availables.mutex.RUnlock()
	for runner := range dispatcher.availables.data {
		// log.Info("[dispatcher] select a free runner %p", runner)
		cacheCount++
		return runner, nil
	}

	tempCount++
	runner := NewRunner()
	runner.status = RunnerStatusFree
	runner.kind = "temp"
	return runner, nil
}

// Scaling scale the v8 runners, check every 10 seconds
// Release the v8 runners if the free runners are more than max size
// Create a new v8 runner if the free runners are less than min size
func (dispatcher *Dispatcher) Scaling() {
}
