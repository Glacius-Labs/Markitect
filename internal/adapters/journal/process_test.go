package journal

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestProcessControllerChild(t *testing.T) {
	data := os.Getenv("MARKITECT_JOURNAL_TEST_REQUEST")
	if data == "" {
		return
	}
	var r apply.ControlRequest
	if err := json.Unmarshal([]byte(data), &r); err != nil {
		t.Fatal(err)
	}
	s := acquire(t, r)
	if _, err := s.Append(context.Background(), batch(t, "before-kill", 0, "recorded-event")); err != nil {
		t.Fatal(err)
	}
	fmt.Println("JOURNAL_CONTROLLER_READY")
	// The parent owns this process and kills it; no Close or simulated fsync.
	if _, err := bufio.NewReader(os.Stdin).ReadString('\n'); err != nil {
		t.Fatal(err)
	}
}

func TestProcessLifetimeLockAndCommittedReopenAfterKill(t *testing.T) {
	r := request(t)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestProcessControllerChild$", "-test.timeout=15s")
	data, _ := json.Marshal(r)
	cmd.Env = append(os.Environ(), "MARKITECT_JOURNAL_TEST_REQUEST="+string(data))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	var diagnostics strings.Builder
	cmd.Stderr = &diagnostics
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	t.Cleanup(func() {
		stdin.Close()
		if !waited {
			cmd.Process.Kill()
			cmd.Wait()
		}
	})
	ready := false
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		if scanner.Text() == "JOURNAL_CONTROLLER_READY" {
			ready = true
			break
		}
	}
	if !ready {
		t.Fatalf("child did not acquire: %v %s", scanner.Err(), diagnostics.String())
	}
	other := r
	other.StateDirectory = filepath.Join(testkit.TempDir(t), "alternate")
	start := time.Now()
	if got, err := New().Acquire(context.Background(), other); err == nil {
		got.Close()
		t.Fatal("other process bypassed repository lock")
	}
	if time.Since(start) > time.Second {
		t.Fatal("controller lock contention waited")
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("expected killed process")
	}
	waited = true
	s := acquire(t, r)
	if s.Binding().Epoch != 2 {
		t.Fatal("lost monotone epoch", s.Binding())
	}
	page, err := s.Read(context.Background(), run.EventQuery{Limit: 1})
	if err != nil || page.Revision != 1 || len(page.Batches) != 1 || page.Batches[0].ID() != "before-kill" {
		t.Fatal(page, err)
	}
	if _, err := New().Acquire(context.Background(), other); !errors.Is(err, apply.ErrControlUnavailable) {
		t.Fatal(err)
	}
	// Process-kill/reopen establishes this acknowledged commit survived this
	// process loss. It is not evidence about power loss or previous child jobs.
}
