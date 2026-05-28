package jobrunner

import (
	"context"

	"github.com/robfig/cron/v3"
)

// Stop stops the scheduler and returns a context that is done
// when all running jobs have completed. Call <-Stop().Done() to wait
// for a graceful shutdown.
func Stop() context.Context {
	return MainCron.Stop()
}

// Remove a specific job from running
// Get EntryID from the list job entries jobrunner.Entries()
// If job is in the middle of running, once the process is finished it will be removed
func Remove(id cron.EntryID) {
	MainCron.Remove(id)
}
