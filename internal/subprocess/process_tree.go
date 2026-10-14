package subprocess

import "time"

var processTerminationGracePeriod = 5 * time.Second

// processTree owns the CLI process and its descendants on each platform.
type processTree interface {
	gracefulStop() error
	forceStop() error
	wait(time.Duration) bool
	close() error
}
