package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// gitCall is one Git command under the repository adapter's invocation policy:
// no GIT_* environment, no system or global configuration, no hooks, literal
// pathspecs and the file transport only (clone and bundles need it).
type gitCall struct {
	// dir is the repository (git -C); empty runs outside any repository.
	dir string
	// safe lists further repository paths Git may open although another user owns them.
	safe  []string
	stdin []byte
}

const (
	maxGitOutput = 64 << 20
	maxGitError  = 16 << 10
)

func (c gitCall) run(ctx context.Context, args ...string) ([]byte, error) {
	prefix := []string{"--no-replace-objects", "--literal-pathspecs",
		"-c", "core.hooksPath=" + os.DevNull, "-c", "core.fsmonitor=false", "-c", "core.autocrlf=false",
		"-c", "protocol.allow=never", "-c", "protocol.file.allow=always", "-c", "commit.gpgsign=false"}
	for _, p := range append(append([]string(nil), c.safe...), c.dir) {
		if p != "" {
			prefix = append(prefix, "-c", "safe.directory="+filepath.ToSlash(p))
		}
	}
	if c.dir != "" {
		prefix = append(prefix, "-C", c.dir)
	}
	cmd := exec.CommandContext(ctx, "git", append(prefix, args...)...)
	cmd.Env = gitEnvironment()
	cmd.WaitDelay = time.Second
	if c.stdin != nil {
		cmd.Stdin = bytes.NewReader(c.stdin)
	}
	out, stderr := &boundedBuffer{limit: maxGitOutput}, &boundedBuffer{limit: maxGitError}
	cmd.Stdout, cmd.Stderr = out, stderr
	err := cmd.Run()
	if out.exceeded {
		return nil, fmt.Errorf("Git %s output exceeds %d bytes", args[0], maxGitOutput)
	}
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("Git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return out.Bytes(), nil
}

// text runs the command and returns its trimmed output.
func (c gitCall) text(ctx context.Context, args ...string) (string, error) {
	out, err := c.run(ctx, args...)
	return strings.TrimSpace(string(out)), err
}

func gitEnvironment() []string {
	var env []string
	for _, value := range os.Environ() {
		name, _, ok := strings.Cut(value, "=")
		if ok && !strings.HasPrefix(strings.ToUpper(name), "GIT_") {
			env = append(env, value)
		}
	}
	return append(env, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_TERMINAL_PROMPT=0", "GIT_NO_LAZY_FETCH=1")
}

// nulPaths splits NUL-terminated Git path output.
func nulPaths(out []byte) []string {
	var paths []string
	for _, p := range strings.Split(string(out), "\x00") {
		if p != "" {
			paths = append(paths, p)
		}
	}
	return paths
}

// samePaths reports whether a and b hold the same set of distinct paths.
func samePaths(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x, y := append([]string(nil), a...), append([]string(nil), b...)
	sort.Strings(x)
	sort.Strings(y)
	for i := range x {
		if x[i] != y[i] || i > 0 && x[i] == x[i-1] {
			return false
		}
	}
	return true
}

type boundedBuffer struct {
	bytes.Buffer
	limit    int64
	exceeded bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if int64(len(p)) > b.limit-int64(b.Len()) {
		b.exceeded = true
		return 0, errors.New("output limit exceeded")
	}
	return b.Buffer.Write(p)
}

// decodeStrict decodes exactly one JSON value with known fields only.
func decodeStrict(data []byte, target any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	if _, err := d.Token(); !errors.Is(err, io.EOF) {
		return errors.New("trailing data")
	}
	return nil
}
