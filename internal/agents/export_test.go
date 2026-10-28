package agents

import "time"

// SetSettle shortens the census wait for tests and returns the restore.
func SetSettle(d time.Duration) func() {
	old := settle
	settle = d
	return func() { settle = old }
}
