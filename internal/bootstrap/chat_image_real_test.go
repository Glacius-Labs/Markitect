package bootstrap

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	markitect "github.com/Glacius-Labs/Markitect"
	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
)

// TestRealEngineBuildsTheChatImage builds the image the native app builds, from
// the source it carries, on a real engine. It runs with MARKITECT_TEST_ENGINE,
// as CI's native job does on Linux, and needs the network and a few minutes. It
// builds under a name of its own, so the images of a developer's app stay.
func TestRealEngineBuildsTheChatImage(t *testing.T) {
	path := os.Getenv("MARKITECT_TEST_ENGINE")
	if path == "" {
		t.Skip("set MARKITECT_TEST_ENGINE to an absolute container engine path")
	}
	engine, err := container.NewEngine(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	image := newChatImage(markitect.ImageSource)
	image.name = "markitect-chat-image-test"
	tag, err := image.tag()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = engine.RemoveImage(context.Background(), tag) })

	// Stop during the build: it ends soon, and no tag appears afterwards.
	stopped, stop := context.WithCancel(ctx)
	defer stop()
	var at time.Time
	_, err = image.ensure(stopped, engine, func(s string) {
		if strings.HasPrefix(s, chatImagePreparing+": ") && at.IsZero() {
			at = time.Now()
			stop()
		}
	})
	if !errors.Is(err, context.Canceled) || at.IsZero() {
		t.Fatalf("a stopped build: %v, stopped at %v", err, at)
	}
	t.Logf("the stopped build ended %s after the stop", time.Since(at).Round(100*time.Millisecond))
	if took := time.Since(at); took > 30*time.Second {
		t.Errorf("a stopped build took %s to end", took)
	}
	time.Sleep(5 * time.Second)
	if found, err := engine.Images(ctx, tag); err != nil || len(found) != 0 {
		t.Fatalf("a stopped build left %q %v", found, err)
	}
	// The tag alone would not show a build that goes on, since a whole build
	// takes longer than this wait. Windows ends the command differently, which
	// this does not show.
	if runtime.GOOS == "linux" {
		if out, err := exec.Command("pgrep", "-af", "--tag "+tag).CombinedOutput(); err == nil {
			t.Errorf("a stopped build still runs: %s", out)
		}
	}

	var said []string
	began := time.Now()
	id, err := image.ensure(ctx, engine, func(s string) { said = append(said, s) })
	if err != nil {
		t.Fatalf("the build: %v\nsaid: %q", err, said)
	}
	t.Logf("built %s in %s, %d lines said", tag, time.Since(began).Round(time.Second), len(said))
	if !strings.HasPrefix(id, "sha256:") {
		t.Errorf("image ID %q", id)
	}
	stepped := false
	for _, s := range said {
		stepped = stepped || strings.HasPrefix(s, chatImagePreparing+": ")
	}
	if !stepped || said[0] != chatImagePreparing || said[len(said)-1] != "Starting the chat's agent" {
		t.Errorf("the chat said %q", said)
	}

	// The image holds what an agent needs.
	out, err := exec.CommandContext(ctx, path, "run", "--rm", "--entrypoint", "/bin/sh", id, "-c",
		"set -e; test -x /usr/local/bin/markitect-agentd; markitect | grep -q 'markitect impact'; export HOME=/tmp; claude --version; codex --version").CombinedOutput()
	if err != nil || !strings.Contains(string(out), "Claude Code") || !strings.Contains(string(out), "codex-cli") {
		t.Errorf("the image: %v\n%s", err, out)
	}

	said = nil
	if _, err := image.ensure(ctx, engine, func(s string) { said = append(said, s) }); err != nil || len(said) != 0 {
		t.Errorf("a later chat: %v, said %q", err, said)
	}
}
