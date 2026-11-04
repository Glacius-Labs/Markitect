// Package claude starts a Run's background sessions with the unmodified Claude
// Code command line, on the user's own stored login (draft concept 4.0). It
// generates each session's settings, so that nothing of a Run reaches the
// project's settings or a modeling session.
package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

const (
	defaultModel  = "sonnet"
	defaultEffort = "high"
	// commandTimeout bounds one call of the command line.
	commandTimeout = 90 * time.Second
	// startWait bounds how long Start waits for the session to come up.
	startWait = 30 * time.Second
	// shellTimeoutMS gives a shell call up to two hours, so that a slow submit
	// answers instead of moving to the background.
	shellTimeoutMS = "7200000"
)

// CLI drives the claude command line.
type CLI struct {
	// Executable is the markitect binary the hooks call.
	Executable string
}

var _ run.Sessions = CLI{}

var backgrounded = regexp.MustCompile(`backgrounded\W+([0-9a-f]{6,})`)

// Start starts a background session in a new worktree and returns its ID once
// the provider lists it.
func (c CLI) Start(ctx context.Context, spec run.SessionSpec) (string, error) {
	if _, err := exec.LookPath("markitect"); err != nil {
		return "", errors.New("markitect is not on the path, and the session calls it by name; put its folder on the path")
	}
	settings, err := c.writeSettings(spec)
	if err != nil {
		return "", err
	}
	args := []string{"--bg", "--worktree", spec.Run}
	args = append(args, launchFlags(spec, settings)...)
	args = append(args, spec.Prompt)
	return c.dispatch(ctx, spec.Dir, args)
}

// Resume continues an ended session in the background. The settings are passed
// again, since the provider does not keep them across a resume.
func (c CLI) Resume(ctx context.Context, id string, spec run.SessionSpec) (string, error) {
	full, err := c.sessionID(ctx, id)
	if err != nil {
		return "", err
	}
	settings, err := c.writeSettings(spec)
	if err != nil {
		return "", err
	}
	args := []string{"--bg", "--resume", full}
	args = append(args, launchFlags(spec, settings)...)
	args = append(args, "You were interrupted. Carry on with the task: markitect checklist shows it, and markitect submit says what is next.")
	return c.dispatch(ctx, spec.Dir, args)
}

func (c CLI) dispatch(ctx context.Context, dir string, args []string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = dir
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errOut.String() + " " + out.String())
		var missing *exec.Error
		if errors.As(err, &missing) {
			return "", fmt.Errorf("the claude command line is not on the path: %w", err)
		}
		if strings.Contains(msg, "Workspace not trusted") {
			return "", fmt.Errorf("Claude Code does not trust this folder for background sessions yet, even if it opens here without asking. Run this once in a terminal in %s and accept its question: claude --bg \"Reply with one word.\"", dir)
		}
		return "", fmt.Errorf("claude --bg: %w: %s", err, msg)
	}
	m := backgrounded.FindStringSubmatch(out.String())
	if m == nil {
		return "", fmt.Errorf("claude --bg did not name its session: %s", strings.TrimSpace(out.String()))
	}
	id := m[1]
	until := time.Now().Add(startWait)
	for {
		// A session counts as up once the provider lists it, in any state: a quick
		// one may have ended already.
		if state, err := c.State(ctx, id); err == nil && state != run.SessionUnknown {
			return id, nil
		}
		if time.Now().After(until) {
			return "", fmt.Errorf("the session %s did not come up within %s", id, startWait)
		}
		time.Sleep(time.Second)
	}
}

func launchFlags(spec run.SessionSpec, settings string) []string {
	model, effort := spec.Model, spec.Effort
	if model == "" {
		model = defaultModel
	}
	if effort == "" {
		effort = defaultEffort
	}
	args := []string{"--model", model, "--effort", effort, "--permission-mode", "dontAsk", "--settings", settings, "--setting-sources", "project", "--strict-mcp-config", "--name", "markitect " + spec.Run}
	if spec.Agent != "" {
		args = append(args, "--agent", spec.Agent)
	}
	return args
}

// session is one row of claude agents --json.
type session struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionId"`
	State     string `json:"state"`
	Status    string `json:"status"`
}

func (c CLI) list(ctx context.Context) ([]session, error) {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "claude", "agents", "--json", "--all").Output()
	if err != nil {
		return nil, fmt.Errorf("claude agents: %w", err)
	}
	var rows []session
	if err := json.Unmarshal(out, &rows); err != nil {
		return nil, fmt.Errorf("claude agents: %w", err)
	}
	return rows, nil
}

// State says how a session stands. Only a session the provider reports as working
// counts: a finished one's process lives on for about an hour, and one that waits
// for a person will wait for ever, since nobody is watching a Run.
func (c CLI) State(ctx context.Context, id string) (run.SessionState, error) {
	rows, err := c.list(ctx)
	if err != nil {
		return run.SessionUnknown, err
	}
	for _, s := range rows {
		if s.ID == id || s.SessionID == id {
			if s.State == "working" {
				return run.SessionWorking, nil
			}
			return run.SessionEnded, nil
		}
	}
	return run.SessionUnknown, nil
}

func (c CLI) sessionID(ctx context.Context, id string) (string, error) {
	rows, err := c.list(ctx)
	if err != nil {
		return "", err
	}
	for _, s := range rows {
		if (s.ID == id || s.SessionID == id) && s.SessionID != "" {
			return s.SessionID, nil
		}
	}
	return id, nil
}

// Stop ends a session's process.
func (c CLI) Stop(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	return exec.CommandContext(ctx, "claude", "stop", id).Run()
}

// Remove forgets an ended session.
func (c CLI) Remove(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	return exec.CommandContext(ctx, "claude", "rm", id).Run()
}

// writeSettings writes the session's settings next to the Run's state.
func (c CLI) writeSettings(spec run.SessionSpec) (string, error) {
	data, err := json.MarshalIndent(c.settings(spec), "", "  ")
	if err != nil {
		return "", err
	}
	file := filepath.Join(spec.Store, "runs", spec.Run+".settings.json")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return "", err
	}
	return file, os.WriteFile(file, data, 0o644)
}

// settings are what a session runs under. They are not in the project, so a
// modeling session gets none of it.
func (c CLI) settings(spec run.SessionSpec) map[string]any {
	model := spec.Model
	if model == "" {
		model = defaultModel
	}
	hook := func(event string) []map[string]any {
		command := fmt.Sprintf("%q hook %s --store %q --run %s", filepath.ToSlash(c.Executable), event, filepath.ToSlash(spec.Store), spec.Run)
		return []map[string]any{{"hooks": []map[string]any{{"type": "command", "command": command}}}}
	}
	s := map[string]any{
		// The default branches from the remote's default branch, and would miss the
		// spec commit that was just made.
		"worktree": map[string]any{"baseRef": "head"},
		// Modeling notes do not reach a run, and a run's notes do not reach the next.
		"autoMemoryEnabled": false,
		"permissions": map[string]any{
			"allow": []string{"Read", "Edit", "Write", "NotebookEdit", "Glob", "Grep", "LSP", "Bash", "Agent", "TodoWrite", "WebFetch", "WebSearch"},
			// The owner pushes. Nothing goes to a remote from a run.
			"deny": []string{"Bash(git push *)", "Bash(git push)", "Bash(gh *)"},
		},
		"env": map[string]string{
			"CLAUDE_CODE_SUBAGENT_MODEL":       model,
			"CLAUDE_CODE_SUBAGENT_MODEL_FORCE": "1",
			"CLAUDE_CODE_DISABLE_AUTO_MEMORY":  "1",
			"BASH_DEFAULT_TIMEOUT_MS":          shellTimeoutMS,
			"BASH_MAX_TIMEOUT_MS":              shellTimeoutMS,
		},
		"hooks": map[string]any{
			run.HookStopName:        hook(run.HookStop),
			run.HookStopFailureName: hook(run.HookStopFailure),
			run.HookPreToolName:     []map[string]any{{"matcher": "*", "hooks": hook(run.HookPreTool)[0]["hooks"]}},
		},
	}
	if home, err := os.UserHomeDir(); err == nil {
		// Personal rules stay out of a run: the flags alone do not keep them out.
		dir := filepath.ToSlash(filepath.Join(home, ".claude"))
		s["claudeMdExcludes"] = []string{dir + "/CLAUDE.md", dir + "/rules/**"}
	}
	return s
}
