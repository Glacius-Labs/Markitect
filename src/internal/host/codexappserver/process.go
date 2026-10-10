package codexappserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// SupportedProviderVersion pins the generated v1 initialize / v2 thread-turn
// shapes used here. Advancing this pin requires protocol fixtures and review.
const SupportedProviderVersion = "codex-cli 0.162.0"
const protocolIdentity = "codex-app-server/v2/0.162.0/markitect-workspace-output-v2/stable-0bf5254bede109d4ae03ce2e81372e4c93a30b359c0749ec7dce7a9382a7f857/experimental-5c0ee37a723e4672108aa5c68b0c63cd540e17d408ca2f8ac1e7ebf76e737ac2/review-assessment-helper-scope-v2/request-evidence-v1/semantic-response-v1/native-workspace-edit-guidance-v1"

type processConnection struct {
	in     io.WriteCloser
	out    io.ReadCloser
	cmd    *exec.Cmd
	guard  processTreeGuard
	once   sync.Once
	done   chan error
	stderr *boundedDigest
	err    error
}
type boundedDigest struct {
	mu       sync.Mutex
	data     []byte
	limit    int
	overflow bool
}

func (b *boundedDigest) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(p)
	if n > b.limit-len(b.data) {
		b.overflow = true
		p = p[:b.limit-len(b.data)]
	}
	b.data = append(b.data, p...)
	return n, nil
}
func (b *boundedDigest) digest() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	sum := sha256.Sum256(b.data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func selectedEnvironment(allowlist *[]string) []string {
	if allowlist == nil {
		return os.Environ()
	}
	out := []string{}
	for _, name := range *allowlist {
		for _, entry := range os.Environ() {
			key, _, _ := strings.Cut(entry, "=")
			if strings.EqualFold(key, name) {
				out = append(out, entry)
				break
			}
		}
	}
	return out
}
func verifyVersion(ctx context.Context, cfg Config, env []string) error {
	if err := validateSandboxPlatform(cfg); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, cfg.Command, "--version")
	cmd.Env = env
	b := &boundedDigest{limit: 1024}
	cmd.Stdout = b
	cmd.Stderr = b
	if err := cmd.Run(); err != nil {
		return errors.New("App Server executable version probe failed")
	}
	if b.overflow || strings.TrimSpace(string(b.data)) != cfg.ProviderVersion {
		return errors.New("App Server executable version differs from pinned provider version")
	}
	return nil
}
func startProcess(cfg Config, cwd string, env []string, maxStderr int) (*processConnection, error) {
	if err := validateSandboxPlatform(cfg); err != nil {
		return nil, err
	}
	guard, err := newProcessTreeGuard()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(cfg.Command, appServerArgs(cfg)...)
	cmd.Dir = cwd
	cmd.Env = env
	if err = guard.prepare(cmd); err != nil {
		_ = guard.close()
		return nil, err
	}
	in, err := cmd.StdinPipe()
	if err != nil {
		_ = guard.close()
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		_ = in.Close()
		_ = guard.close()
		return nil, err
	}
	b := &boundedDigest{limit: maxStderr}
	cmd.Stderr = b
	if err = cmd.Start(); err != nil {
		_ = in.Close()
		_ = out.Close()
		_ = guard.close()
		return nil, err
	}
	if err = guard.attach(cmd); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		_ = in.Close()
		_ = out.Close()
		_ = guard.close()
		return nil, err
	}
	p := &processConnection{in: in, out: out, cmd: cmd, guard: guard, done: make(chan error, 1), stderr: b}
	go func() { p.done <- cmd.Wait() }()
	return p, nil
}

func appServerArgs(cfg Config) []string {
	args := make([]string, 0, 5)
	if cfg.PermissionProfile == ":workspace" {
		// This owned-workspace default authorizes ordinary task work without
		// asking again per file. The workspace sandbox and Host delta guards
		// remain in force; no user/global configuration is changed.
		args = append(args, "-c", "approval_policy=never", "-c", "sandbox_workspace_write={writable_roots=[],network_access=false,exclude_tmpdir_env_var=false,exclude_slash_tmp=false}")
		args = append(args, "-c", "features.memories=false", "-c", "allow_login_shell=false")
	}
	if cfg.WindowsSandboxBackend == WindowsSandboxBackendMXC {
		// Codex documents -c as a global CLI option, so it must precede the
		// app-server subcommand. This changes only this child process.
		args = append(args, "-c", "windows.sandbox=mxc")
	}
	return append(args, "app-server", "--listen", "stdio://")
}

func validateSandboxPlatform(cfg Config) error {
	if cfg.WindowsSandboxBackend != "" && runtime.GOOS != "windows" {
		return errors.New("Windows sandbox backend mxc is only supported on Windows")
	}
	return nil
}
func (p *processConnection) Read(b []byte) (int, error)  { return p.out.Read(b) }
func (p *processConnection) Write(b []byte) (int, error) { return p.in.Write(b) }
func (p *processConnection) Close() error {
	p.once.Do(func() {
		_ = p.in.Close()
		p.err = p.guard.close()
		_ = p.out.Close()
		select {
		case <-p.done:
		case <-time.After(30 * time.Second):
			p.err = errors.Join(p.err, errors.New("App Server process cleanup uncertain"))
		}
	})
	return p.err
}
