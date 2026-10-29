package bootstrap

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/remotecontainer"
	"github.com/Glacius-Labs/Markitect/internal/adapters/remoteserver"
	"github.com/Glacius-Labs/Markitect/internal/adapters/rungit"
	runengine "github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// newAgents constructs the provider adapters that start makers and
// checkers in their containers.
var newAgents = providerAgents

// runs builds one Run engine per command: the host's Git over the target
// repository, the Run's remote in a container and the provider adapters.
type runs struct{}

func (runs) Run(ctx context.Context, o cli.RunOptions) (*runengine.Result, error) {
	agents, err := newAgents(ctx)
	if err != nil {
		return nil, err
	}
	target, branch, err := rungit.Target(ctx, o.Repo, o.Remote, o.Branch)
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "markitect-run-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	host, err := rungit.Open(ctx, filepath.Join(dir, "host.git"), target)
	if err != nil {
		return nil, err
	}
	remotes, err := newRemotes()
	if err != nil {
		return nil, err
	}
	o.Request.Target = branch
	e := &runengine.Engine{Git: host, Remotes: remotes, Agents: agents}
	return e.Run(ctx, o.Request)
}

// containerRemotes starts each Run's remote as a container with the Docker
// CLI. MARKITECT_ENGINE names the CLI (default docker), MARKITECT_REMOTE_BINARY
// the Linux markitect-remote program (default next to this executable) and
// MARKITECT_REMOTE_IMAGE the image when execution.yaml names no checks.
// remoteLimits bound the Run's remote container. It serves Git and runs the
// required checks one push at a time, a whole build and test run of the
// project, so it gets more than one Manager container.
var remoteLimits = remotecontainer.Limits{CPUs: "2", Memory: "4g", Pids: 1024}

type containerRemotes struct {
	engine *remotecontainer.Engine
	binary string
}

func newRemotes() (*containerRemotes, error) {
	cliPath := os.Getenv("MARKITECT_ENGINE")
	if cliPath == "" {
		p, err := exec.LookPath("docker")
		if err != nil {
			return nil, errors.New("no container engine: install Docker or set MARKITECT_ENGINE")
		}
		cliPath = p
	}
	engine, err := remotecontainer.NewEngine(cliPath)
	if err != nil {
		return nil, err
	}
	binary := os.Getenv("MARKITECT_REMOTE_BINARY")
	if binary == "" {
		self, err := os.Executable()
		if err != nil {
			return nil, err
		}
		binary = filepath.Join(filepath.Dir(self), "markitect-remote")
	}
	if _, err := os.Stat(binary); err != nil {
		return nil, fmt.Errorf("the Run remote's program %s: %w; set MARKITECT_REMOTE_BINARY", binary, err)
	}
	return &containerRemotes{engine: engine, binary: binary}, nil
}

func (c *containerRemotes) Start(ctx context.Context, plan *runremote.Plan, bundle io.Reader) (runengine.Remote, error) {
	image := plan.Image()
	if image == "" {
		image = os.Getenv("MARKITECT_REMOTE_IMAGE")
	}
	if image == "" {
		return nil, errors.New("execution.yaml names no checks and so no image for the Run's remote; set MARKITECT_REMOTE_IMAGE to an image with Git and a shell")
	}
	r, err := remotecontainer.Start(ctx, c.engine, remotecontainer.Spec{Image: image, Binary: c.binary, Plan: plan, Limits: remoteLimits}, bundle)
	if err != nil {
		return nil, err
	}
	return containerRemote{r}, nil
}

type containerRemote struct{ r *remotecontainer.Remote }

func (c containerRemote) URL() string     { return c.r.URL }
func (c containerRemote) Network() string { return c.r.Network }

func (c containerRemote) Export(ctx context.Context, w io.Writer) error { return c.r.Export(ctx, w) }

func (c containerRemote) Log(ctx context.Context) ([]runengine.Push, error) {
	data, err := c.r.Log(ctx)
	if err != nil {
		return nil, err
	}
	var pushes []runengine.Push
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64<<10), 64<<20)
	for sc.Scan() {
		var e remoteserver.Entry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			return nil, err
		}
		pushes = append(pushes, runengine.Push{User: e.User, Updates: e.Updates, Decision: e.Decision})
	}
	return pushes, sc.Err()
}

func (c containerRemote) Check(ctx context.Context, bundle io.Reader, commit string) ([]runremote.CheckResult, error) {
	return c.r.Check(ctx, bundle, commit)
}

func (c containerRemote) StartFrom(ctx context.Context, branch, commit string) error {
	return c.r.StartFrom(ctx, branch, commit)
}

func (c containerRemote) Close(ctx context.Context) error { return c.r.Close(ctx) }
