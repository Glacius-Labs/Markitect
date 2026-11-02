package container

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// buildTailLines bounds the lines of a failed build that its error repeats.
const buildTailLines = 12

// ImageBuild is one image build from a tar build context.
type ImageBuild struct {
	Tag string
	// Dockerfile is the recipe's path inside Context.
	Dockerfile string
	Context    io.Reader
	// Progress, when set, receives each line the build prints.
	Progress func(line string)
}

// BuildImage builds an image and tags it. Ending ctx ends the build; the engine
// keeps its finished steps for the next one, and the tag does not move.
func (e *Engine) BuildImage(ctx context.Context, b ImageBuild) error {
	if err := plainReference(b.Tag); err != nil {
		return err
	}
	if b.Dockerfile == "" || strings.HasPrefix(b.Dockerfile, "-") || b.Context == nil {
		return fmt.Errorf("image build of %s needs a recipe and a build context", b.Tag)
	}
	lines := &buildLines{progress: b.Progress}
	cmd := exec.CommandContext(ctx, e.path, "build", "--progress=plain", "--tag", b.Tag, "--file", b.Dockerfile, "-")
	cmd.Stdin = b.Context
	cmd.Stdout, cmd.Stderr = lines, lines
	// The engine's build plugin holds these pipes too and ends just after it.
	cmd.WaitDelay = 10 * time.Second
	err := cmd.Run()
	lines.flush()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		return fmt.Errorf("container engine build: %w: %s", err, strings.Join(lines.tail, "\n"))
	}
	return nil
}

// Images lists the tagged references of the images that match a pattern such
// as name:prefix-*. An engine that does not answer is an error, not an empty
// list.
func (e *Engine) Images(ctx context.Context, pattern string) ([]string, error) {
	if err := plainReference(pattern); err != nil {
		return nil, err
	}
	out, err := e.output(ctx, "image", "ls", "--filter", "reference="+pattern, "--format", "{{.Repository}}:{{.Tag}}")
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(out)), nil
}

// RemoveImage removes one tag. An image a container still runs from stays, and
// that is an error.
func (e *Engine) RemoveImage(ctx context.Context, ref string) error {
	if err := plainReference(ref); err != nil {
		return err
	}
	_, err := e.output(ctx, "image", "rm", ref)
	return err
}

func plainReference(ref string) error {
	if ref == "" || strings.HasPrefix(ref, "-") || strings.ContainsAny(ref, " \n\r\x00") {
		return fmt.Errorf("image reference %q is not plain", ref)
	}
	return nil
}

// buildLines splits a build's output into lines and keeps the last few. The
// command writes to it from one goroutine at a time.
type buildLines struct {
	progress func(string)
	rest     []byte
	tail     []string
}

func (l *buildLines) Write(p []byte) (int, error) {
	l.rest = append(l.rest, p...)
	for {
		i := bytes.IndexByte(l.rest, '\n')
		if i < 0 {
			return len(p), nil
		}
		l.line(string(l.rest[:i]))
		l.rest = l.rest[i+1:]
	}
}

func (l *buildLines) flush() {
	if len(l.rest) > 0 {
		l.line(string(l.rest))
		l.rest = nil
	}
}

func (l *buildLines) line(s string) {
	s = strings.TrimRight(s, "\r")
	if l.progress != nil {
		l.progress(s)
	}
	l.tail = append(l.tail, s)
	if len(l.tail) > buildTailLines {
		l.tail = l.tail[1:]
	}
}
