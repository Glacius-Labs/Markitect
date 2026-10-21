package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
)

// CockpitRequest selects what the cockpit observes: state directories named
// directly or through a run configuration, the repository with its model and
// dependency models as check selects them, and its loopback listen address.
// A run configuration together with a Runtime profile enables model
// sessions and the start of runs. Build is the implementation identity of this
// product build, which a run the cockpit starts binds as markitect apply does.
type CockpitRequest struct {
	States         []string
	RunConfig      string
	RuntimeProfile string
	Selection      apply.InputSelection
	Listen         string
	Build          string
}

// CockpitFunc serves the cockpit until ctx ends. ready receives the address
// the browser opens once the cockpit listens.
type CockpitFunc func(ctx context.Context, request CockpitRequest, ready func(url string)) error

// WindowOpener opens the cockpit's address in a window and returns without
// waiting for it. It is best effort: an error is reported, never fatal.
type WindowOpener func(url string) error

// WithWindowOpener installs the opener `markitect cockpit` calls once the
// cockpit listens. Without it no window is opened.
func (c *Command) WithWindowOpener(open WindowOpener) *Command {
	c.window = open
	return c
}

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
	runtimeProfile := flags.String("runtime-profile", "", "Runtime profile that enables model sessions with --run-config")
	noOpen := flags.Bool("no-open", false, "do not open the cockpit in a window")
	listen := flags.String("listen", "", "loopback address to listen on")
	repo := flags.String("repo", ".", "repository to observe and propose settings for")
	manifest := flags.String("model", ".markitect/model.yaml", "explicit model.yaml")
	var dependencies manifestList
	flags.Var(&dependencies, "dependency", "explicit dependency model.yaml; repeat")
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
	if *runtimeProfile != "" && (*config == "" || !filepath.IsAbs(*runtimeProfile) || filepath.Clean(*runtimeProfile) != *runtimeProfile) {
		fmt.Fprintln(stderr, "--runtime-profile requires --run-config and an explicit absolute native path")
		return 2
	}
	root, err := filepath.Abs(*repo)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	selection := apply.InputSelection{Repository: root}
	selection.Local, err = selectManifest(root, *manifest)
	for _, dependency := range dependencies {
		if err != nil {
			break
		}
		selected, e := selectManifest(root, dependency)
		err = e
		selection.Dependencies = append(selection.Dependencies, selected)
	}
	if err != nil {
		fmt.Fprintln(stderr, "invalid explicit model selection")
		return 2
	}
	request := CockpitRequest{States: states, RunConfig: *config, RuntimeProfile: *runtimeProfile, Selection: selection, Listen: *listen, Build: buildinfo.Build()}
	err = c.cockpit(ctx, request, func(url string) {
		emit(map[string]any{"operation": "cockpit", "status": "serving", "url": url})
		if *noOpen || c.window == nil {
			return
		}
		if err := c.window(url); err != nil {
			fmt.Fprintf(stderr, "could not open the cockpit window (open %s yourself): %v\n", url, err)
		}
	})
	if err != nil {
		emit(map[string]string{"operation": "cockpit", "status": "not_completed", "error": err.Error()})
		return 3
	}
	return 0
}
