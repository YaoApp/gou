package v8

import (
	"fmt"
	"sync"
	"time"

	"github.com/yaoapp/kun/log"
)

// RunnerMap is a runner map
type RunnerMap struct {
	data   map[uint]*Runner
	length uint
	mutex  *sync.RWMutex
}

// Dispatcher is a runner dispatcher
type Dispatcher struct {
	availables RunnerMap
	total      uint
	min        uint
	max        uint
}

// the global dispatcher instance
// initialize when the v8 start
var dispatcher *Dispatcher = nil

// NewDispatcher is a runner dispatcher
func NewDispatcher(min, max uint) *Dispatcher {

	// Test the min and max
	min = 200
	max = 200
	return &Dispatcher{
		availables: RunnerMap{data: make(map[uint]*Runner), mutex: &sync.RWMutex{}},
		total:      0,
		min:        min,
		max:        max,
	}
}

// Start start the v8 mannager
func (dispatcher *Dispatcher) Start() error {
	for i := uint(0); i < dispatcher.min; i++ {
		dispatcher.create()
	}
	log.Info("[dispatcher] the dispatcher is started. runners %d", dispatcher.total)
	return nil
}

// Stop stop the v8 mannager
func (dispatcher *Dispatcher) Stop() {
	for _, runner := range dispatcher.availables.data {
		dispatcher.destory(runner)
		runner.signal <- RunnerCommandDestroy
	}
}

func (dispatcher *Dispatcher) offline(runner *Runner) {
	dispatcher.availables.mutex.Lock()
	defer dispatcher.availables.mutex.Unlock()
	dispatcher._offline(runner)
}

func (dispatcher *Dispatcher) _offline(runner *Runner) {
	delete(dispatcher.availables.data, runner.id)
	log.Info("[dispatcher] runner %p offline (%d/%d) (%d/%d)", runner, len(dispatcher.availables.data), dispatcher.total, tempCount, cacheCount)
	// create a new runner if the total runners are less than max
	// if dispatcher.total < dispatcher.max {
	// 	go dispatcher.create()
	// }
}

func (dispatcher *Dispatcher) online(runner *Runner) {
	dispatcher.availables.mutex.Lock()
	defer dispatcher.availables.mutex.Unlock()
	dispatcher.availables.data[runner.id] = runner
	log.Info("[dispatcher] runner %p online (%d/%d) (%d/%d)", runner, len(dispatcher.availables.data), dispatcher.total, tempCount, cacheCount)
}

func (dispatcher *Dispatcher) create() {
	runner := NewRunner(true)
	go runner.Start()
	dispatcher.total++
	log.Info("[dispatcher] runner %p create (%d/%d) (%d/%d)", runner, len(dispatcher.availables.data), dispatcher.total, tempCount, cacheCount)
}

func (dispatcher *Dispatcher) destory(runner *Runner) {
	dispatcher.availables.mutex.Lock()
	defer dispatcher.availables.mutex.Unlock()
	delete(dispatcher.availables.data, runner.id)
	dispatcher.total--
	log.Info("[dispatcher] runner %p destory (%d,%d) (%d/%d)", runner, len(dispatcher.availables.data), dispatcher.total, tempCount, cacheCount)
}

// Select select a free v8 runner
func (dispatcher *Dispatcher) Select(timeout time.Duration) (*Runner, error) {
	dispatcher.availables.mutex.Lock()
	defer dispatcher.availables.mutex.Unlock()
	for _, runner := range dispatcher.availables.data {
		cacheCount++
		dispatcher._offline(runner)
		fmt.Println("--------------------", runner.id)
		fmt.Println("Select a free v8 runner id", runner.id, "count", len(dispatcher.availables.data))
		return runner, nil
	}

	tempCount++
	runner := NewRunner(false)
	go runner.Start()
	return runner, nil
}

// Scaling scale the v8 runners, check every 10 seconds
// Release the v8 runners if the free runners are more than max size
// Create a new v8 runner if the free runners are less than min size
func (dispatcher *Dispatcher) Scaling() {
}
