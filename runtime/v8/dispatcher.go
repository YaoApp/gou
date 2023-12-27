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
	health     *Health
	total      uint
	min        uint
	max        uint
}

// Health is the health check
type Health struct {
	missing uint // the missing runners
	total   uint // the total runners
}

// the global dispatcher instance
// initialize when the v8 start
var dispatcher *Dispatcher = nil

// NewDispatcher is a runner dispatcher
func NewDispatcher(min, max uint) *Dispatcher {

	// Test the min and max
	// min = 10
	// max = 200
	return &Dispatcher{
		availables: RunnerMap{data: make(map[uint]*Runner), mutex: &sync.RWMutex{}},
		health:     &Health{missing: 0, total: 0},
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
	log.Trace("[dispatcher] the dispatcher is started. runners %d", dispatcher.total)
	go dispatcher.Scaling()
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
	log.Trace("[dispatcher] runner %d offline (%d/%d) %s", runner.id, len(dispatcher.availables.data), dispatcher.total, dispatcher.health)
	// create a new runner if the total runners are less than max
	// if dispatcher.total < dispatcher.max {
	// 	go dispatcher.create()
	// }
}

func (dispatcher *Dispatcher) online(runner *Runner) {
	dispatcher.availables.mutex.Lock()
	defer dispatcher.availables.mutex.Unlock()
	dispatcher.availables.data[runner.id] = runner
	log.Trace("[dispatcher] runner %d online (%d/%d) %s", runner.id, len(dispatcher.availables.data), dispatcher.total, dispatcher.health)
}

func (dispatcher *Dispatcher) create() {
	runner := NewRunner(true)
	ready := make(chan bool)
	go runner.Start(ready)
	<-ready
	dispatcher.total++
	log.Trace("[dispatcher] runner %d create (%d/%d) %s", runner.id, len(dispatcher.availables.data), dispatcher.total, dispatcher.health)
}

func (dispatcher *Dispatcher) destory(runner *Runner) {
	dispatcher.availables.mutex.Lock()
	defer dispatcher.availables.mutex.Unlock()
	delete(dispatcher.availables.data, runner.id)
	dispatcher.total--
	log.Trace("[dispatcher] runner %d destory (%d,%d) %s", runner.id, len(dispatcher.availables.data), dispatcher.total, dispatcher.health)
}

// Select select a free v8 runner
func (dispatcher *Dispatcher) Select(timeout time.Duration) (*Runner, error) {
	dispatcher.availables.mutex.Lock()
	defer dispatcher.availables.mutex.Unlock()
	go dispatcher.totalCount()

	for _, runner := range dispatcher.availables.data {
		dispatcher._offline(runner)
		log.Debug(fmt.Sprintf("--- %d -----------------\n", runner.id))
		log.Debug(fmt.Sprintln("1.  Select a free v8 runner id", runner.id, "count", len(dispatcher.availables.data)))
		return runner, nil
	}

	go dispatcher.missingCount()
	runner := NewRunner(false)
	ready := make(chan bool)
	go runner.Start(ready)
	<-ready
	return runner, nil
}

// Scaling scale the v8 runners, check every 10 seconds
// Release the v8 runners if the free runners are more than max size
// Create a new v8 runner if the free runners are less than min size
func (dispatcher *Dispatcher) Scaling() {

	log.Info("[dispatcher] the dispatcher is scaling. min %d, max %d", dispatcher.min, dispatcher.max)

	// check the free runners every 30 seconds
	// @todo: release the free runners if the free runners are more than min size
	ticker := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-ticker.C:
			percent := float64(dispatcher.health.missing) / float64(dispatcher.health.total)
			log.Trace("[dispatcher] the health percent is %f", percent)
			dispatcher.health.Reset()
			if percent > 0.2 {
				for i := uint(0); i < dispatcher.max-dispatcher.total; i++ {
					dispatcher.create()
				}
				continue
			}
		}
	}
}

func (dispatcher *Dispatcher) missingCount() {
	dispatcher.health.missing = dispatcher.health.missing + 1
}

func (dispatcher *Dispatcher) totalCount() {
	dispatcher.health.total = dispatcher.health.total + 1
}

func (health *Health) String() string {
	return fmt.Sprintf("missing: %d, total: %d", health.missing, health.total)
}

// Reset reset the health
func (health *Health) Reset() {
	health.missing = 0
	health.total = 0
}
