package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
)

// CockpitRequest selects what the cockpit observes: state directories named
// directly or through a run configuration, and its loopback listen address.
type CockpitRequest struct {
	States    []string
	RunConfig string
	Listen    string
}

// CockpitFunc serves the cockpit until ctx ends. ready receives the address
// the browser opens once the cockpit listens.
type CockpitFunc func(ctx context.Context, request CockpitRequest, ready func(url string)) error

// WithCockpit installs the cockpit. Without it the command reports that the
// cockpit is unavailable.
func (c *Command) WithCockpit(cockpit CockpitFunc) *Command {
	c.cockpit = cockpit
	return c
}

func (c *Command) cockpitVerb(ctx context.Context, args []string, stdout, stderr io.Writer, emit func(any) int) int {
	if c.cockpit == nil {
		emit(map[string]string{"operation": "cockpit", "status": "not_completed", "reason": "The cockpit is not configured."})
		return 3
	}
	flags := flag.NewFlagSet("cockpit", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var states manifestList
	flags.Var(&states, "state", "state directory to observe; repeat")
	config := flags.String("run-config", "", "run configuration whose state directory to observe")
	listen := flags.String("listen", "", "loopback address to listen on")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			io.WriteString(stdout, help)
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || len(states) == 0 && *config == "" {
		fmt.Fprintln(stderr, "cockpit requires --state DIR or --run-config PATH; see help")
		return 2
	}
	if *config != "" && (!filepath.IsAbs(*config) || filepath.Clean(*config) != *config) {
		fmt.Fprintln(stderr, "--run-config requires an explicit absolute native path")
		return 2
	}
	request := CockpitRequest{States: states, RunConfig: *config, Listen: *listen}
	err := c.cockpit(ctx, request, func(url string) {
		emit(map[string]any{"operation": "cockpit", "status": "serving", "url": url})
	})
	if err != nil {
		emit(map[string]string{"operation": "cockpit", "status": "not_completed", "error": err.Error()})
		return 3
	}
	return 0
}
