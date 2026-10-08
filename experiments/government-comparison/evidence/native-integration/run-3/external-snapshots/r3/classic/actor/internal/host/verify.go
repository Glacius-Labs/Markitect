package host

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

const (
	verifyOutputLimit = 1 << 20
	verifyWaitDelay   = 2 * time.Second
	verifyDefaultTime = 10 * time.Minute
)

// GateResult records one explicitly declared repository check run against the
// fixed Markitect snapshot. Tool is the executable token declared in spec.checks.
type GateResult struct {
	Name         string `yaml:"name"`
	Tool         string `yaml:"tool"`
	ExitCode     int    `yaml:"exitCode"`
	Milliseconds int64  `yaml:"milliseconds"`
	Output       string `yaml:"output,omitempty"`
}

// VerifyError classifies why repository verification did not complete. Kind is
// one of incomplete-evidence, tool-missing, gate-failure, timeout, or output-limit.
type VerifyError struct {
	Kind string
	Gate string
	Err  error
}

func (e *VerifyError) Error() string {
	if e.Gate != "" {
		return fmt.Sprintf("repository verification %s at check %q: %v", e.Kind, e.Gate, e.Err)
	}
	return fmt.Sprintf("repository verification %s: %v", e.Kind, e.Err)
}

func (e *VerifyError) Unwrap() error { return e.Err }

type verifyCommand struct {
	name string
	tool string
	args []string
	env  []string
}

// VerifyRepository runs only checks explicitly declared on the Project from
// the same immutable snapshot used by the graph. Each check has a 10-minute
// execution limit. An absent checks list is incomplete repository evidence;
// callers may still use graph-only validation.
func VerifyRepository(p *Project) ([]GateResult, error) {
	return verifyRepositoryWithTimeout(p, verifyDefaultTime)
}

// VerifyRepresentationChecks isolates each check at the same fixed input bytes
// and rejects mutation of any original snapshot file. It is not an OS sandbox.
func VerifyRepresentationChecks(p *Project) ([]GateResult, error) {
	return verifyRepositoryScoped(p, verifyDefaultTime, true)
}
func verifyRepositoryWithTimeout(p *Project, timeout time.Duration) ([]GateResult, error) {
	return verifyRepositoryScoped(p, timeout, false)
}
func verifyRepositoryScoped(p *Project, timeout time.Duration, immutableInputs bool) ([]GateResult, error) {
	if p == nil || p.Snapshot == nil || p.Graph == nil || p.Graph.Project == nil {
		return nil, &VerifyError{Kind: "incomplete-evidence", Err: errors.New("a parsed project and source snapshot are required")}
	}
	return verifySnapshotScoped(p.Snapshot, p.Graph.Project.Spec.Checks, timeout, immutableInputs)
}

// VerifySnapshotChecks reuses the fixed-snapshot verifier for explicitly supplied
// checks. It does not authenticate check ownership or prove semantic sufficiency.
func VerifySnapshotChecks(captured *snapshot.Snapshot, checks []authoring.Check) ([]GateResult, error) {
	return verifySnapshotScoped(captured, checks, verifyDefaultTime, true)
}

func verifySnapshotScoped(captured *snapshot.Snapshot, checks []authoring.Check, timeout time.Duration, immutableInputs bool) ([]GateResult, error) {
	if captured == nil {
		return nil, &VerifyError{Kind: "incomplete-evidence", Err: errors.New("a source snapshot is required")}
	}
	if captured.Provisional {
		return nil, &VerifyError{Kind: "incomplete-evidence", Err: errors.New("verify requires --revision; working trees cannot provide immutable evidence")}
	}
	if timeout <= 0 {
		return nil, &VerifyError{Kind: "incomplete-evidence", Err: errors.New("check timeout must be positive")}
	}
	commands, err := planVerifyCommands(checks)
	if err != nil {
		return nil, err
	}
	temporary, err := os.MkdirTemp("", "markitect-verify-")
	if err != nil {
		return nil, &VerifyError{Kind: "incomplete-evidence", Err: err}
	}
	defer os.RemoveAll(temporary)
	temporary, err = filepath.EvalSymlinks(temporary)
	if err != nil {
		return nil, &VerifyError{Kind: "incomplete-evidence", Err: err}
	}
	if err = source.Materialize(captured, temporary); err != nil {
		return nil, &VerifyError{Kind: "incomplete-evidence", Err: err}
	}

	results := make([]GateResult, 0, len(commands))
	tools := make(map[string]string)
	for _, command := range commands {
		executable, ok := tools[command.tool]
		if !ok {
			executable, err = findVerifyTool(command.tool)
			if err != nil {
				return results, &VerifyError{Kind: "tool-missing", Gate: command.name, Err: err}
			}
			tools[command.tool] = executable
		}
		directory := temporary
		if immutableInputs {
			directory, err = os.MkdirTemp("", "markitect-projection-check-")
			if err != nil {
				return results, &VerifyError{Kind: "incomplete-evidence", Gate: command.name, Err: err}
			}
			if err = source.Materialize(captured, directory); err != nil {
				os.RemoveAll(directory)
				return results, &VerifyError{Kind: "incomplete-evidence", Gate: command.name, Err: err}
			}
		}
		result, runErr := runVerifyCommand(command, executable, directory, timeout)
		if immutableInputs {
			integrityErr := snapshotFilesUnchanged(captured, directory)
			os.RemoveAll(directory)
			if integrityErr != nil {
				result.ExitCode = -1
				runErr = &VerifyError{Kind: "input-mutation", Gate: command.name, Err: integrityErr}
			}
		}
		results = append(results, result)
		if runErr != nil {
			return results, runErr
		}
	}
	return results, nil
}

func planVerifyCommands(checks []authoring.Check) ([]verifyCommand, error) {
	if len(checks) == 0 {
		return nil, &VerifyError{Kind: "incomplete-evidence", Err: errors.New("the Project declares no repository checks; graph-only validation is available, but it does not verify repository checks")}
	}
	commands := make([]verifyCommand, 0, len(checks))
	names := make(map[string]bool, len(checks))
	for _, check := range checks {
		if err := authoring.ValidateCheck(check); err != nil {
			return nil, &VerifyError{Kind: "incomplete-evidence", Gate: check.Name, Err: fmt.Errorf("invalid declared repository check: %w", err)}
		}
		name := check.Name
		if names[name] {
			return nil, &VerifyError{Kind: "incomplete-evidence", Gate: name, Err: errors.New("repository check names must be unique")}
		}
		names[name] = true
		tool := check.Run[0]
		args := append([]string(nil), check.Run[1:]...)
		commands = append(commands, verifyCommand{name: name, tool: tool, args: args})
	}
	return commands, nil
}

func findVerifyTool(tool string) (string, error) {
	if strings.TrimSpace(tool) == "" || strings.ContainsAny(tool, `/\\`) || filepath.IsAbs(tool) {
		return "", errors.New("check executable must be a bare command name resolved from PATH")
	}
	toolPath, err := exec.LookPath(tool)
	if err != nil {
		return "", fmt.Errorf("check executable %q is unavailable on PATH: %w", tool, err)
	}
	if !filepath.IsAbs(toolPath) {
		return "", fmt.Errorf("PATH resolved check executable %q to non-absolute path %q", tool, toolPath)
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
	cmd.Env = verifyEnvironment(command.env)
	cmd.Stdout, cmd.Stderr = output, output
	cmd.WaitDelay = verifyWaitDelay
	runErr := cmd.Run()
	result := GateResult{
		Name: command.name, Tool: command.tool,
		ExitCode: 0, Milliseconds: time.Since(started).Milliseconds(), Output: output.String(),
	}
	if output.exceeded() {
		result.ExitCode = -1
		return result, &VerifyError{Kind: "output-limit", Gate: command.name, Err: fmt.Errorf("output exceeded %d bytes; check was cancelled", verifyOutputLimit)}
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		result.ExitCode = -1
		return result, &VerifyError{Kind: "timeout", Gate: command.name, Err: fmt.Errorf("exceeded %s execution limit", timeout)}
	}
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			return result, &VerifyError{Kind: "gate-failure", Gate: command.name, Err: fmt.Errorf("process exited with code %d", result.ExitCode)}
		}
		result.ExitCode = -1
		return result, &VerifyError{Kind: "incomplete-evidence", Gate: command.name, Err: runErr}
	}
	return result, nil
}

// verifyEnvironment removes Go's ambient configuration and workspace redirects
// for every check, and clears inherited Git repository/object/config redirection.
func verifyEnvironment(extra []string) []string {
	filtered := make([]string, 0, len(os.Environ())+len(extra)+3)
	for _, entry := range append(source.CleanGitEnv(), extra...) {
		key, _, _ := strings.Cut(entry, "=")
		if strings.EqualFold(key, "GOFLAGS") || strings.EqualFold(key, "GOENV") || strings.EqualFold(key, "GOWORK") {
			continue
		}
		filtered = append(filtered, entry)
	}
	return append(filtered, "GOFLAGS=", "GOENV=off", "GOWORK=off")
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

func verifySnapshotFilesUnchanged(p *Project, directory string) error {
	return snapshotFilesUnchanged(p.Snapshot, directory)
}

func snapshotFilesUnchanged(captured *snapshot.Snapshot, directory string) error {
	names := make([]string, 0, len(captured.Files))
	for name := range captured.Files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		dest, err := safeDestination(directory, name)
		if err != nil {
			return fmt.Errorf("check replaced snapshot path %s: %w", name, err)
		}
		info, err := os.Lstat(dest)
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("check removed or replaced snapshot file %s", name)
		}
		if runtime.GOOS != "windows" && (info.Mode().Perm()&0111 != 0) != (captured.Modes[name] == "100755") {
			return fmt.Errorf("check changed snapshot executable mode %s", name)
		}
		data, err := os.ReadFile(dest)
		if err != nil || !bytes.Equal(data, captured.Files[name]) {
			return fmt.Errorf("check changed snapshot bytes %s", name)
		}
	}
	return nil
}
