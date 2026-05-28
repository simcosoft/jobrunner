package jobrunner

import (
	"github.com/simcosoft/cron/v3"
)

const DEFAULT_JOB_POOL_SIZE = 10

var (
	// Singleton instance of the underlying job scheduler.
	MainCron *cron.Cron

	// This limits the number of jobs allowed to run concurrently.
	workPermits chan struct{}

	// Is a single job allowed to run concurrently with itself?
	selfConcurrent bool
)

var (
	functions = []any{makeWorkPermits, isSelfConcurrent}
)

func makeWorkPermits(bufferCapacity int) {
	if bufferCapacity <= 0 {
		workPermits = make(chan struct{}, DEFAULT_JOB_POOL_SIZE)
	} else {
		workPermits = make(chan struct{}, bufferCapacity)
	}
}

func isSelfConcurrent(concurrencyFlag int) {

	if concurrencyFlag <= 0 {
		selfConcurrent = false
	} else {
		selfConcurrent = true
	}
}


func Start(v ...int) {

	MainCron = cron.New()

	for i, option := range v {
		if i >= len(functions) {
			break
		}
		functions[i].(func(int))(option)
	}

	MainCron.Start()
}
