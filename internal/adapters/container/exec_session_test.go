package container

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestExecSessionRetainsOneHardenedContainerAcrossCommands(t *testing.T) {
	e, calls := fakeCommandEngine(t,
		engineStep{Out: commandID}, engineStep{Out: commandID}, engineStep{Out: commandState("running", true, 0)},
		engineStep{Out: "first"}, engineStep{Err: "Git refused", Code: 1}, engineStep{Out: "last"}, engineStep{}, engineStep{},
	)
	session, err := e.StartExec(context.Background(), commandSpec(), []string{"/bin/sleep", "infinity"})
	if err != nil || session.ID() != commandID {
		t.Fatalf("start %v %v", session, err)
	}
	var output bytes.Buffer
	if err := session.Exec(context.Background(), []string{"/usr/bin/git", "literal $text"}, nil, &output); err != nil || output.String() != "first" {
		t.Fatalf("first exec %q %v", output.String(), err)
	}
	if err := session.Exec(context.Background(), []string{"/usr/bin/git", "fail"}, nil, &output); err == nil || !strings.Contains(err.Error(), "Git refused") {
		t.Fatalf("lost command failure: %v", err)
	}
	if err := session.Exec(context.Background(), []string{"/usr/bin/git", "last"}, nil, &output); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := session.Close(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if err := session.Exec(context.Background(), []string{"/usr/bin/git", "late"}, nil, &output); err == nil {
		t.Fatal("exec after close")
	}
	got := calls()
	if len(got) != 8 || got[0][0] != "create" || got[1][0] != "start" || got[2][0] != "inspect" || got[6][0] != "rm" || got[7][0] != "ps" {
		t.Fatalf("wrong lifetime: %v", got)
	}
	create := strings.Join(got[0], " ")
	if strings.Contains(create, "--cap-add") {
		t.Fatalf("ordinary exec changed default capability hardening: %s", create)
	}
	for _, hardening := range []string{"--read-only", "--cap-drop ALL", "no-new-privileges", "--user 1000:1000", "--network none"} {
		if !strings.Contains(create, hardening) {
			t.Fatalf("lost hardening %s: %s", hardening, create)
		}
	}
	if !reflect.DeepEqual(got[3], []string{"exec", "--interactive", "--user", "1000:1000", commandID, "/usr/bin/git", "literal $text"}) {
		t.Fatalf("command changed: %v", got[3])
	}
	if session.GitExecCount() != 3 {
		t.Fatalf("lost started Git exec count after close: %d", session.GitExecCount())
	}
}

func TestExecSessionCancelledCloseStillObservesRemoval(t *testing.T) {
	e, calls := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{}, engineStep{Out: commandState("running", true, 0)}, engineStep{}, engineStep{})
	session, err := e.StartExec(context.Background(), commandSpec(), []string{"/bin/sleep", "infinity"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := session.Close(ctx); err != nil {
		t.Fatal(err)
	}
	got := calls()
	if got[len(got)-2][0] != "rm" || got[len(got)-1][0] != "ps" {
		t.Fatalf("cancelled close did not observe removal: %v", got)
	}
}

func TestExecSessionCloseFailureRemainsVisibleAndBlocksExec(t *testing.T) {
	e, calls := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{}, engineStep{Out: commandState("running", true, 0)}, engineStep{Err: "remove failed", Code: 1}, engineStep{Out: commandID})
	session, err := e.StartExec(context.Background(), commandSpec(), []string{"/bin/sleep", "infinity"})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := session.Close(context.Background()); err == nil || !strings.Contains(err.Error(), "remove failed") {
			t.Fatalf("cleanup failure lost: %v", err)
		}
	}
	if err := session.Exec(context.Background(), []string{"/usr/bin/git", "late"}, nil, &bytes.Buffer{}); err == nil || len(calls()) != 5 {
		t.Fatal("uncertain close allowed another command or hid cleanup")
	}
}

func TestOwnedExecInitializationFailureClosesObservedContainer(t *testing.T) {
	e, calls := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{}, engineStep{Out: commandState("running", true, 0)}, engineStep{Code: 1, Err: "root was not empty"}, engineStep{}, engineStep{})
	spec := commandSpec()
	spec.Mounts = []Mount{{Volume: "owned", Target: "/assignments/one", NoCopy: true}}
	session, err := e.StartOwnedExec(context.Background(), spec, []string{"/bin/sleep", "infinity"})
	if err == nil || !strings.Contains(err.Error(), "root was not empty") || session == nil || session.ID() != commandID {
		t.Fatalf("failed initialization lost its observed handle: %v %v", session, err)
	}
	if err := session.Close(context.Background()); err != nil {
		t.Fatalf("successful cleanup became uncertain: %v", err)
	}
	if err := session.Exec(context.Background(), []string{"/usr/bin/git", "late"}, nil, &bytes.Buffer{}); err == nil {
		t.Fatal("failed initialization allowed a command")
	}
	got := calls()
	if got[4][0] != "rm" || got[4][2] != commandID || got[5][0] != "ps" {
		t.Fatalf("failed init lost owned cleanup: %v", got)
	}
}

func TestOwnedExecInitializationFailureRetainsUncertainClosure(t *testing.T) {
	e, calls := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{}, engineStep{Out: commandState("running", true, 0)}, engineStep{Code: 1, Err: "root was not empty"}, engineStep{Code: 1, Err: "remove failed"}, engineStep{Out: commandID})
	spec := commandSpec()
	spec.Mounts = []Mount{{Volume: "owned", Target: "/assignments/one", NoCopy: true}}
	session, err := e.StartOwnedExec(context.Background(), spec, []string{"/bin/sleep", "infinity"})
	if session == nil || session.ID() != commandID || err == nil {
		t.Fatalf("failed initialization lost its observed handle: %v %v", session, err)
	}
	for _, message := range []string{"root was not empty", "remove failed", "removal was not observed"} {
		if !strings.Contains(err.Error(), message) {
			t.Fatalf("lost startup or cleanup failure %q: %v", message, err)
		}
	}
	if err := session.Close(context.Background()); err == nil || !strings.Contains(err.Error(), "remove failed") || !strings.Contains(err.Error(), "removal was not observed") {
		t.Fatalf("closure uncertainty lost: %v", err)
	}
	if err := session.Exec(context.Background(), []string{"/usr/bin/git", "late"}, nil, &bytes.Buffer{}); err == nil || len(calls()) != 6 {
		t.Fatal("failed initialization allowed another command or retried cleanup")
	}
}

func TestOwnedExecInitializationIsOneRootExecAndOrdinaryCommandsStayNonRoot(t *testing.T) {
	e, calls := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{}, engineStep{Out: commandState("running", true, 0)}, engineStep{}, engineStep{}, engineStep{}, engineStep{})
	spec := commandSpec()
	spec.Mounts = []Mount{{Volume: "owned", Target: "/assignments/one", NoCopy: true}, {Volume: "other", Target: "/assignments/two", NoCopy: true}}
	session, err := e.StartOwnedExec(context.Background(), spec, []string{"/bin/sleep", "infinity"})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Exec(context.Background(), []string{"/usr/bin/git", "status"}, nil, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if err := session.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := calls()
	if !strings.Contains(strings.Join(got[0], " "), "--cap-add CHOWN") || got[3][2] != "0:0" || got[4][3] != "1000:1000" || got[3][len(got[3])-2] != "/assignments/one" || got[3][len(got[3])-1] != "/assignments/two" {
		t.Fatalf("ownership initialization escaped its boundary: %v", got)
	}
	if session.GitExecCount() != 1 {
		t.Fatal("root initializer counted as a Git exec")
	}
}

func TestExecSessionStartFailuresOnlyRemoveObservedOwnID(t *testing.T) {
	for name, steps := range map[string][]engineStep{
		"name conflict": {{Err: "name in use", Code: 1}},
		"lost response": {{Out: commandID, Code: 1}, {}, {}},
		"start failed":  {{Out: commandID}, {Code: 1}, {}, {}},
		"wrong image":   {{Out: commandID}, {}, {Out: strings.ReplaceAll(commandState("running", true, 0), commandImage, "sha256:"+strings.Repeat("d", 64))}, {}, {}},
	} {
		t.Run(name, func(t *testing.T) {
			e, calls := fakeCommandEngine(t, steps...)
			session, err := e.StartExec(context.Background(), commandSpec(), []string{"/bin/sleep", "infinity"})
			if err == nil {
				t.Fatalf("accepted failed start: %v", session)
			}
			if name == "name conflict" {
				if session != nil {
					t.Fatalf("returned an unobserved handle: %v", session)
				}
			} else {
				if session == nil || session.ID() != commandID {
					t.Fatalf("lost observed handle: %v", session)
				}
				if err := session.Close(context.Background()); err != nil {
					t.Fatalf("successful cleanup became uncertain: %v", err)
				}
				if err := session.Exec(context.Background(), []string{"/usr/bin/git", "late"}, nil, &bytes.Buffer{}); err == nil {
					t.Fatal("failed start allowed a command")
				}
			}
			for _, call := range calls() {
				if call[0] == "rm" && call[len(call)-1] != commandID {
					t.Fatalf("removed unobserved container: %v", call)
				}
			}
			if name == "name conflict" && len(calls()) != 1 {
				t.Fatalf("name conflict removed another container: %v", calls())
			}
		})
	}
}

func TestExecSessionCancellationRemovesRemoteProcessContainer(t *testing.T) {
	e, calls := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{}, engineStep{Out: commandState("running", true, 0)}, engineStep{SleepMillis: 5000}, engineStep{}, engineStep{})
	session, err := e.StartExec(context.Background(), commandSpec(), []string{"/bin/sleep", "infinity"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- session.Exec(ctx, []string{"/usr/bin/git", "slow"}, nil, &bytes.Buffer{}) }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(os.Getenv("MARKITECT_FAKE_COMMAND_ENGINE"), "exec-ready")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("exec did not start")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation lost: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cancelled exec did not close")
	}
	if err := session.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := calls()
	if got[len(got)-2][0] != "rm" || got[len(got)-1][0] != "ps" {
		t.Fatalf("remote process cleanup not observed: %v", got)
	}
}

func TestRunningImageRequiresActualRunningContainerIdentity(t *testing.T) {
	valid := `[{"Id":"` + commandID + `","Name":"/owned-native","Image":"` + commandImage + `","State":{"Running":true}}]`
	for _, kind := range []string{"valid", "wrong name", "missing identity", "stopped", "missing image", "incomplete state", "engine failure"} {
		t.Run(kind, func(t *testing.T) {
			out := valid
			code := 0
			switch kind {
			case "wrong name":
				out = strings.ReplaceAll(out, "/owned-native", "/foreign")
			case "missing identity":
				out = strings.ReplaceAll(out, commandID, "")
			case "stopped":
				out = strings.ReplaceAll(out, "true", "false")
			case "missing image":
				out = strings.ReplaceAll(out, commandImage, "")
			case "incomplete state":
				out = strings.ReplaceAll(out, `"Running":true`, ``)
			case "engine failure":
				code = 1
			}
			e, calls := fakeCommandEngine(t, engineStep{Out: out, Code: code})
			image, err := e.RunningImage(context.Background(), "owned-native")
			if kind == "valid" {
				if err != nil || image != commandImage {
					t.Fatal(image, err)
				}
			} else if err == nil || image != "" {
				t.Fatal("unconfirmed image accepted", image, err)
			}
			if len(calls()) != 1 {
				t.Fatal(calls())
			}
		})
	}
}
