package v8

import (
	"fmt"
	"time"

	"github.com/yaoapp/kun/log"
)

// Dispatcher is a runner dispatcher
type Dispatcher struct {
	availables chan *Runner
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
		availables: make(chan *Runner, max),
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
	// for _, runner := range dispatcher.availables.data {
	// 	dispatcher.destroy(runner)
	// 	runner.signal <- RunnerCommandDestroy
	// }
}

func (dispatcher *Dispatcher) online(runner *Runner) {
	dispatcher.availables <- runner
	log.Trace("[dispatcher] [%s] runner online. availables:%d, total:%d, %s", runner.id, len(dispatcher.availables), dispatcher.total, dispatcher.health)
}

func (dispatcher *Dispatcher) create() {
	if dispatcher.total >= dispatcher.max {
		log.Error("[dispatcher] the runner is max. availables:%d, total:%d, %s", len(dispatcher.availables), dispatcher.total, dispatcher.health)
		return
	}
	runner := NewRunner(true)
	ready := make(chan bool)
	defer close(ready)

	go runner.Start(ready)
	<-ready
	dispatcher.total++
	log.Trace("[dispatcher] [%s] runner create. availables:%d, total:%d, %s", runner.id, len(dispatcher.availables), dispatcher.total, dispatcher.health)
}

// Select select a free v8 runner
func (dispatcher *Dispatcher) Select(timeout time.Duration) (*Runner, error) {

	go dispatcher.totalCount()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timecnt := 1 * time.Millisecond
	for {
		select {
		case <-ticker.C:
			timecnt = timecnt + 100*time.Millisecond
			if timecnt > timeout {
				return nil, fmt.Errorf("[dispatcher] select timeout %v", timeout)
			}

			if uint(len(dispatcher.availables)) < dispatcher.max {
				dispatcher.missingCount()
				go dispatcher.create()
			}
			break

		case runner := <-dispatcher.availables:
			log.Debug(fmt.Sprintf("--- [%s] -----------------", runner.id))
			log.Debug(fmt.Sprintf("1.  [%s] Select a free v8 runner. availables=%d", runner.id, len(dispatcher.availables)))
			return runner, nil

		}
	}

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
	return fmt.Sprintf("missing:%d, total:%d", health.missing, health.total)
}

// Reset reset the health
func (health *Health) Reset() {
	health.missing = 0
	health.total = 0
}
