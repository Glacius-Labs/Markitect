package gitrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/tooling/guardedwrite"
)

type blockedGit struct {
	PID     int
	Address string
	Args    []string
}

// Re-execute this test binary as Git. Only an armed source/guard query blocks;
// ordinary acquisition still uses real Git. The listener is an OS-owned liveness
// witness: it must disappear when CommandContext kills and waits for this process.
func publicationGitProcess() {
	base := strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
	control := os.Getenv("MARKITECT_TEST_PUBLICATION_GIT")
	if base != "git" || control == "" {
		return
	}
	args := os.Args[1:]
	mode, _ := os.ReadFile(filepath.Join(control, "armed"))
	query := strings.Join(args, " ")
	selected := !slices.Contains(args, "--literal-pathspecs")
	combined := strings.Contains(query, "--show-toplevel --git-dir --git-common-dir --show-object-format")
	if selected && string(mode) == "fallback" && combined {
		fmt.Println("ambiguous metadata")
		os.Exit(0)
	}
	block := selected && (string(mode) == "identity" && combined || string(mode) == "fallback" && strings.HasSuffix(query, "rev-parse --show-toplevel") || string(mode) == "branch" && strings.HasSuffix(query, "branch --show-current") || string(mode) == "head" && strings.HasSuffix(query, "rev-parse --verify HEAD^{commit}"))
	if block {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			os.Exit(90)
		}
		data, _ := json.Marshal(blockedGit{os.Getpid(), listener.Addr().String(), args})
		if os.WriteFile(filepath.Join(control, "ready"), data, 0600) != nil {
			os.Exit(91)
		}
		for {
			c, err := listener.Accept()
			if err != nil {
				os.Exit(92)
			}
			c.Close()
		}
	}
	cmd := exec.Command(os.Getenv("MARKITECT_TEST_PUBLICATION_REAL_GIT"), args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			os.Exit(exit.ExitCode())
		}
		os.Exit(93)
	}
	os.Exit(0)
}

func publicationGitShim(t *testing.T) string {
	t.Helper()
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	name := "git"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MARKITECT_TEST_PUBLICATION_GIT", dir)
	t.Setenv("MARKITECT_TEST_PUBLICATION_REAL_GIT", realGit)
	// The copied binary may carry -race. Do not charge its default exit sleep
	// once per forwarded Git query against the original test execution window.
	t.Setenv("GORACE", strings.TrimSpace(os.Getenv("GORACE")+" atexit_sleep_ms=0"))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return dir
}

func observeBlockedGit(t *testing.T, control string, cancel context.CancelFunc, done <-chan error, want error) {
	t.Helper()
	var process blockedGit
	until := time.Now().Add(20 * time.Second)
	for {
		data, err := os.ReadFile(filepath.Join(control, "ready"))
		if err == nil && json.Unmarshal(data, &process) == nil {
			break
		}
		select {
		case err := <-done:
			t.Fatalf("Git query was not reached: %v", err)
		default:
		}
		if time.Now().After(until) {
			t.Fatal("blocking Git query not observed")
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Cleanup(func() {
		c, err := net.DialTimeout("tcp", process.Address, time.Second)
		if err != nil {
			return
		}
		c.Close()
		if p, err := os.FindProcess(process.PID); err == nil {
			_ = p.Kill()
		}
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("blocked test Git did not drain after recovery kill")
		}
	})
	c, err := net.DialTimeout("tcp", process.Address, time.Second)
	if err != nil {
		t.Fatal("blocked process was not alive", err)
	}
	c.Close()
	if cancel != nil {
		cancel()
	}
	select {
	case err := <-done:
		if !errors.Is(err, want) {
			t.Fatalf("Git cancellation=%v, want %v; args=%v", err, want, process.Args)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Git query ignored cancellation")
	}
	c, err = net.DialTimeout("tcp", process.Address, time.Second)
	if err == nil {
		c.Close()
		t.Fatal("Git process survived cancellation and Wait")
	}
}

func TestPublicationGitQueriesCancelBeforeMutationAndAfterPartialTransfer(t *testing.T) {
	r, prepared, git, input := runFixture(t)
	ctx := context.Background()
	service, err := git.Open(ctx, run.Request{ID: "publication-cancel", Project: "fixture", Prepared: prepared, Input: input})
	if err != nil {
		t.Fatal(err)
	}
	copy, err := service.Copy(ctx, ".", run.Commit{ID: input.Commit, Tree: input.Tree})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"out/a.txt", "out/deep.txt"} {
		if err := os.WriteFile(filepath.Join(copy.Location().Directory, filepath.FromSlash(name)), []byte("candidate\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	fixed, err := copy.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := service.Retain(ctx, copy, fixed)
	if err != nil {
		t.Fatal(err)
	}
	if err := copy.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if err := service.Close(ctx); err != nil {
		t.Fatal(err)
	}
	// This Publisher seam fixture supplies a bound closed approval. It is not
	// native review or orchestration evidence; those have separate tests.
	approval := run.Result{ID: "approved", Project: "fixture", Binding: prepared.Binding(), Mandate: run.Verify, Technical: run.Completed, Decision: run.Approve, Input: ref, Candidate: &ref}
	for _, m := range prepared.Topology().Managers() {
		approval.Handoffs = append(approval.Handoffs, run.Handoff{Manager: m.Package})
		approval.Cleanup = append(approval.Cleanup, run.Closure{Resource: "session:" + m.Package, Closed: true}, run.Closure{Resource: "copy:" + m.Package, Closed: true})
	}
	approval.Cleanup = append(approval.Cleanup, run.Closure{Resource: "manager-workers", Closed: true}, run.Closure{Resource: "git-service", Closed: true})
	for _, test := range []struct{ stage, query string }{
		{"preflight", "identity"}, {"preflight", "fallback"}, {"preflight", "branch"}, {"preflight", "head"},
		{"publish", "identity"}, {"partial", "identity"}, {"partial", "branch"}, {"partial", "head"},
	} {
		t.Run(test.stage+"-"+test.query, func(t *testing.T) {
			control := publicationGitShim(t)
			arm := func() {
				if err := os.WriteFile(filepath.Join(control, "armed"), []byte(test.query), 0600); err != nil {
					t.Error(err)
				}
			}
			repository := adapter(t)
			if test.stage == "partial" {
				repository.observe = func(event, path string) {
					if event == "publication-before-target" && path == "out/deep.txt" {
						arm()
					}
				}
			} else {
				arm()
			}
			window, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			done := make(chan error, 1)
			var published apply.Publication
			publisher := &Publisher{Repository: repository, Store: git}
			go func() {
				if test.stage == "preflight" {
					done <- publisher.Preflight(window, prepared)
					return
				}
				var err error
				published, err = publisher.Publish(window, apply.PublicationRequest{Prepared: prepared, Approval: approval})
				done <- err
			}()
			observeBlockedGit(t, control, cancel, done, context.Canceled)
			if published.Readback {
				t.Fatal("cancellation reported publication success")
			}
			want := []string{}
			if test.stage == "partial" {
				want = []string{"out/a.txt"}
			}
			if len(published.CompletedPaths) != len(want) || len(want) > 0 && !reflect.DeepEqual(published.CompletedPaths, want) {
				t.Fatal("dishonest partial paths", published.CompletedPaths, want)
			}
			for _, name := range []string{"out/a.txt", "out/deep.txt"} {
				before, _ := prepared.Attempt().Model().Inputs().Raw.Census.Lookup(fixturePath(t, name))
				wantBytes := before.Bytes()
				if test.stage == "partial" && name == "out/a.txt" {
					wantBytes = []byte("candidate\n")
				}
				data, err := os.ReadFile(filepath.Join(r.Dir, filepath.FromSlash(name)))
				if err != nil || string(data) != string(wantBytes) {
					t.Fatal("unexpected target mutation", name, err)
				}
				r.Write(name, string(before.Bytes()))
			}
			if _, err := os.Stat(filepath.Join(r.Dir, ".markitect/state/write.lock")); !os.IsNotExist(err) {
				t.Fatal("writer lock survived cancellation", err)
			}
		})
	}
	t.Run("original-deadline", func(t *testing.T) {
		control := publicationGitShim(t)
		if err := os.WriteFile(filepath.Join(control, "armed"), []byte("identity"), 0600); err != nil {
			t.Fatal(err)
		}
		window, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		done := make(chan error, 1)
		go func() { _, err := guardedwrite.CaptureV1Files(window, r.Dir, []string{"out/a.txt"}); done <- err }()
		observeBlockedGit(t, control, nil, done, context.DeadlineExceeded)
	})
}
