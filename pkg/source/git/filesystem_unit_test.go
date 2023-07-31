package git

import (
	"context"
	"testing"
	"time"
)

// WaitForUpdate is used by locally packaged tests to block until either the provided
// deadline is exceeded or the filesystem gets an update from a fetch call
func WaitForUpdate(t *testing.T, f *Source, d time.Duration) error {
	select {
	case <-time.After(d):
		return context.DeadlineExceeded
	case <-f.notify:
		return nil
	}
}
