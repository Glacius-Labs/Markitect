// Command markitect-release prepares or publishes a run-verified private
// Markitect GitHub release. A normal invocation is read-only; --publish is the
// explicit owner-authenticated state transition.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/tooling/publish"
	"go.yaml.in/yaml/v3"
)

type ghRunner struct {
	executable string
	prefixArgs []string
}

func (r ghRunner) Run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	executable := r.executable
	if executable == "" {
		executable = "gh"
	}
	return runGH(ctx, executable, r.prefixArgs, dir, args...)
}

func runGH(ctx context.Context, executable string, prefixArgs []string, dir string, args ...string) ([]byte, error) {
	allArgs := append(append([]string(nil), prefixArgs...), args...)
	cmd := exec.CommandContext(ctx, executable, allArgs...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		// Preserve stdout because `gh api --include` carries the HTTP status and
		// JSON response there, even on HTTP errors. Never merge or expose stderr:
		// diagnostics can contain sensitive authentication context.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return stdout.Bytes(), fmt.Errorf("gh exited with status %d", exitErr.ExitCode())
		}
		return stdout.Bytes(), fmt.Errorf("could not run gh: %w", err)
	}
	return stdout.Bytes(), nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func run(args []string) error {
	if len(args) > 0 && args[0] == "distribution" {
		return runDistribution(args[1:])
	}
	flags := flag.NewFlagSet("markitect-release", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	tag := flags.String("tag", "", "release tag, for example v1.2.3")
	runID := flags.String("run", "", "successful GitHub Actions Release workflow run ID")
	assets := flags.String("assets", "", "directory containing the four locally downloaded release assets")
	publishNow := flags.Bool("publish", false, "create, upload, publish, and verify the immutable GitHub release")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	result, err := publish.Execute(ctx, ghRunner{}, publish.Options{Tag: *tag, RunID: *runID, AssetsDir: *assets, Publish: *publishNow})
	if result != nil {
		data, marshalErr := yaml.Marshal(result)
		if marshalErr != nil {
			return marshalErr
		}
		if n, writeErr := os.Stdout.Write(data); writeErr != nil {
			return writeErr
		} else if n != len(data) {
			return errors.New("short write while printing YAML result")
		}
	}
	return err
}

func runDistribution(args []string) error {
	flags := flag.NewFlagSet("markitect-release distribution", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	tag := flags.String("tag", "", "published immutable stable release tag, for example v1.2.3")
	root := flags.String("repo", ".", "Markitect repository root for check or write")
	write := flags.Bool("write", false, "write the generated README sections and canonical WinGet manifests")
	export := flags.String("export-winget", "", "export manifests into a winget-pkgs checkout")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	if *tag == "" {
		return errors.New("--tag is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	result, err := publish.SyncDistribution(ctx, ghRunner{}, publish.DistributionOptions{Tag: *tag, RepositoryRoot: *root, Write: *write, ExportWinget: *export})
	if result != nil {
		data, marshalErr := yaml.Marshal(result)
		if marshalErr != nil {
			return marshalErr
		}
		if n, writeErr := os.Stdout.Write(data); writeErr != nil {
			return writeErr
		} else if n != len(data) {
			return errors.New("short write while printing YAML result")
		}
	}
	return err
}
