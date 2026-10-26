package run

import "context"

// HostTime pauses the clock of the response ctx belongs to, as serving a tool
// call does, until the returned function ends the host's work. A Session
// adapter uses it while it waits outside the provider, such as before
// repeating a response after a transient provider error. The Run deadline
// still bounds that wait.
func HostTime(ctx context.Context) func() {
	if clock, ok := ctx.Value(clockKey{}).(interface{ serve() func() }); ok {
		return clock.serve()
	}
	return func() {}
}
