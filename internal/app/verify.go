package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/source"
)

const (
	verifyOutputLimit = 1 << 20
	verifyWaitDelay   = 2 * time.Second
	verifyDefaultTime = 10 * time.Minute
)

type GateResult struct {
	Profile      string `yaml:"profile"`
	Name         string `yaml:"name"`
	Tool         string `yaml:"tool"`
	ExitCode     int    `yaml:"exitCode"`
	Milliseconds int64  `yaml:"milliseconds"`
	Output       string `yaml:"output,omitempty"`
}

// VerifyError classifies why repository verification did not complete. Kind is
// one of incomplete-evidence, tool-missing, gate-failure, timeout, or output-limit.
type VerifyError struct {
	Kind    string
	Profile string
	Gate    string
	Err     error
}

func (e *VerifyError) Error() string {
	if e.Gate != "" {
		return fmt.Sprintf("repository verification %s for profile %s at %s: %v", e.Kind, e.Profile, e.Gate, e.Err)
	}
	return fmt.Sprintf("repository verification %s for profile %s: %v", e.Kind, e.Profile, e.Err)
}

func (e *VerifyError) Unwrap() error { return e.Err }

type verifyCommand struct {
	profile string
	name    string
	tool    string
	args    []string
	env     []string
}

// VerifyRepository runs the profile's fixed allowlisted commands from the
// same immutable snapshot used by the graph. Each gate has a 10-minute limit.
func VerifyRepository(p *Project) ([]GateResult, error) {
	return verifyRepositoryWithTimeout(p, verifyDefaultTime)
}

func verifyRepositoryWithTimeout(p *Project, timeout time.Duration) ([]GateResult, error) {
	if p == nil || p.Snapshot == nil || p.Graph == nil || p.Graph.Project == nil {
		return nil, &VerifyError{Kind: "incomplete-evidence", Err: errors.New("a parsed project and source snapshot are required")}
	}
	if p.Snapshot.Provisional {
		return nil, &VerifyError{Kind: "incomplete-evidence", Err: errors.New("verify requires --revision; working trees cannot provide immutable evidence")}
	}
	if timeout <= 0 {
		return nil, &VerifyError{Kind: "incomplete-evidence", Err: errors.New("gate timeout must be positive")}
	}
	profile := p.Graph.Project.Spec.Profile
	commands, err := planVerifyCommands(profile, p.Snapshot.Files)
	if err != nil {
		return nil, err
	}
	temporary, err := os.MkdirTemp("", "markitect-verify-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(temporary)
	temporary, err = filepath.EvalSymlinks(temporary)
	if err != nil {
		return nil, err
	}
	if err = source.Materialize(p.Snapshot, temporary); err != nil {
		return nil, err
	}

	results := make([]GateResult, 0, len(commands))
	tools := make(map[string]string)
	for _, command := range commands {
		executable, ok := tools[command.tool]
		if !ok {
			executable, err = findVerifyTool(command.tool)
			if err != nil {
				return results, &VerifyError{Kind: "tool-missing", Profile: profile, Gate: command.name, Err: err}
			}
			tools[command.tool] = executable
		}
		result, runErr := runVerifyCommand(command, executable, temporary, timeout)
		results = append(results, result)
		if runErr != nil {
			return results, runErr
		}
	}
	return results, nil
}

func planVerifyCommands(profile string, files map[string][]byte) ([]verifyCommand, error) {
	var commands []verifyCommand
	switch profile {
	case "konfyra":
		commands = []verifyCommand{
			{profile: profile, name: "scripts/render-governance-adapters.py --check", tool: "python", args: []string{"-B", "scripts/render-governance-adapters.py", "--check"}},
			{profile: profile, name: "python unittest discover scripts/tests", tool: "python", args: []string{"-B", "-m", "unittest", "discover", "-s", "scripts/tests", "-v"}},
		}
		_, hasRunner := files["scripts/run-markitect.go"]
		_, hasTest := files["scripts/markitect-bootstrap_test.go"]
		if hasRunner != hasTest {
			return nil, &VerifyError{Kind: "incomplete-evidence", Profile: profile, Err: errors.New("Go Markitect bootstrap evidence must include both scripts/run-markitect.go and scripts/markitect-bootstrap_test.go")}
		}
		if hasRunner {
			commands = append(commands, verifyCommand{
				profile: profile, name: "go test -v scripts/run-markitect.go scripts/markitect-bootstrap_test.go", tool: "go",
				args: []string{"test", "-v", "scripts/run-markitect.go", "scripts/markitect-bootstrap_test.go"},
			})
		}
	case "cockpit":
		commands = []verifyCommand{
			{profile: profile, name: "scripts/check_docs.py", tool: "python", args: []string{"-B", "scripts/check_docs.py"}},
			{profile: profile, name: "scripts/render_adapters.py --check", tool: "python", args: []string{"-B", "scripts/render_adapters.py", "--check"}},
		}
	default:
		return nil, &VerifyError{Kind: "incomplete-evidence", Profile: profile, Err: errors.New("profile has no repository gate adapter; check covers the Markitect graph only")}
	}
	return commands, nil
}

func findVerifyTool(tool string) (string, error) {
	if tool == "python" {
		if python, err := exec.LookPath("python"); err == nil {
			return python, nil
		}
		if python, err := exec.LookPath("python3"); err == nil {
			return python, nil
		}
		return "", errors.New("Python is required by this repository profile")
	}
	toolPath, err := exec.LookPath(tool)
	if err != nil {
		return "", fmt.Errorf("%s is required by this repository profile: %w", tool, err)
	}
	return toolPath, nil
}

func runVerifyCommand(command verifyCommand, executable, directory string, timeout time.Duration) (GateResult, error) {
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	output := &boundedVerifyOutput{limit: verifyOutputLimit, cancel: cancel}
	cmd := exec.CommandContext(ctx, executable, command.args...)
	cmd.Dir = directory
	cmd.Env = source.CleanGitEnv()
	if len(command.env) > 0 {
		cmd.Env = append(cmd.Env, command.env...)
	}
	cmd.Stdout, cmd.Stderr = output, output
	cmd.WaitDelay = verifyWaitDelay
	runErr := cmd.Run()
	result := GateResult{
		Profile: command.profile, Name: command.name, Tool: command.tool,
		ExitCode: 0, Milliseconds: time.Since(started).Milliseconds(), Output: output.String(),
	}
	if output.exceeded() {
		result.ExitCode = -1
		return result, &VerifyError{Kind: "output-limit", Profile: command.profile, Gate: command.name, Err: fmt.Errorf("output exceeded %d bytes; gate was cancelled", verifyOutputLimit)}
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		result.ExitCode = -1
		return result, &VerifyError{Kind: "timeout", Profile: command.profile, Gate: command.name, Err: fmt.Errorf("exceeded %s execution limit", timeout)}
	}
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			return result, &VerifyError{Kind: "gate-failure", Profile: command.profile, Gate: command.name, Err: fmt.Errorf("process exited with code %d", result.ExitCode)}
		}
		result.ExitCode = -1
		return result, &VerifyError{Kind: "gate-failure", Profile: command.profile, Gate: command.name, Err: runErr}
	}
	return result, nil
}

type boundedVerifyOutput struct {
	mu       sync.Mutex
	buffer   bytes.Buffer
	limit    int
	cancel   context.CancelFunc
	tooLarge bool
}

func (w *boundedVerifyOutput) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	remaining := w.limit - w.buffer.Len()
	if remaining > 0 {
		if len(data) > remaining {
			_, _ = w.buffer.Write(data[:remaining])
		} else {
			_, _ = w.buffer.Write(data)
		}
	}
	if len(data) > remaining && !w.tooLarge {
		w.tooLarge = true
		w.cancel()
	}
	return len(data), nil
}

func (w *boundedVerifyOutput) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.String()
}

func (w *boundedVerifyOutput) exceeded() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.tooLarge
}
