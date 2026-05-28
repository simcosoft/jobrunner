package jobrunner

import (
	"github.com/simcosoft/cron/v3"
)

// Return detailed list of currently running recurring jobs
// to remove an entry, first retrieve the ID of entry
func Entries() []cron.Entry {
	return MainCron.Entries()
}
