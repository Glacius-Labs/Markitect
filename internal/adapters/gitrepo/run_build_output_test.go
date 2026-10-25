package gitrepo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/runstore"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
	"github.com/Glacius-Labs/Markitect/internal/workflow"
)

// These tests hold the build-output rule of the workspace capture to concept
// §6.2: the file format never overrules a registration, nothing unaccounted
// enters a candidate, and no file leaves a candidate without a trace.

// elfHeader opens every ELF file.
const elfHeader = "\x7fELF\x02\x01\x01\x00"

// withArtifacts appends exact registrations of a/deep's item to shard.
func withArtifacts(shard string, paths ...string) string {
	for _, p := range paths {
		shard += "  - path: " + p + "\n    form: build\n    realizes: [a/deep/item]\n    why: A binary the model requires.\n"
	}
	return shard
}

// Registration, not the file format, decides what a snapshot leaves out. An
// untracked ELF file that a registry shard names exactly is captured, also
// when its registration is not staged yet, and merge preparation never removes
// it. One that only a directory entry covers, or that nothing registers, is a
// build output: it stays untracked in the copy, and Outputs reports it.
func TestRegistrationNotFormatDecidesWhatASnapshotLeavesOut(t *testing.T) {
	ctx := context.Background()
	for _, backend := range []string{"local", "linux"} {
		t.Run(backend, func(t *testing.T) {
			r := testkit.NewRepo(t)
			r.Write(".markitect/registry/registry.yaml", "apiVersion: markitect.registry/v1\nartifacts:\n  - path: firmware/\n    form: build\n    realizes: [a/Thing]\n    why: Firmware images.\n")
			r.Write(".markitect/registry/a/registry.yaml", withArtifacts("apiVersion: markitect.registry/v1\nartifacts:\n", "out/firmware"))
			r.Write("firmware/README.md", "Firmware images.\n")
			base := r.Commit("base")
			r.Git("switch", "--quiet", "-c", "incoming")
			r.Write("out/firmware", elfHeader+"incoming firmware")
			r.Write("out/tool", elfHeader+"incoming tool")
			incoming := r.Commit("incoming")
			r.Git("switch", "--quiet", "--detach", base)
			r.Write("out/firmware", elfHeader+"modeled firmware")
			r.Write("out/tool", elfHeader+"go build output")
			executables, command := localExecutables(r.Dir), gitCommand(localCommand)
			if backend == "linux" {
				if _, err := exec.LookPath("node"); err != nil {
					t.Skip("the Linux probe runs Node, which this host lacks")
				}
				c := &linuxCopy{service: &linuxService{exec: hostLinuxTools{}}, directory: r.Dir}
				executables, command = c.executables, c.service.command
			}
			if err := clearOutputs(ctx, r.Dir, incoming, command, executables); err != nil {
				t.Fatal(err)
			}
			if status := r.Git("status", "--porcelain", "--untracked-files=all"); status != "?? out/firmware" {
				t.Fatalf("merge preparation removed a registered file or kept a build output: %q", status)
			}
			// The Manager registers a second binary without staging anything,
			// builds its tool again and adds an image below the directory entry.
			r.Write(".markitect/registry/a/registry.yaml", withArtifacts("apiVersion: markitect.registry/v1\nartifacts:\n", "out/firmware", "out/fresh"))
			r.Write("out/fresh", elfHeader+"registered, never staged")
			r.Write("out/tool", elfHeader+"go build output")
			r.Write("firmware/image", elfHeader+"image below a directory entry")
			fixed, err := snapshotGit(ctx, r.Dir, command, nil, executables)
			if err != nil {
				t.Fatal(err)
			}
			want := []string{".markitect/registry/a/registry.yaml", ".markitect/registry/registry.yaml", "firmware/README.md", "out/firmware", "out/fresh"}
			if files := strings.Fields(r.Git("ls-tree", "-r", "--name-only", fixed.ID)); !reflect.DeepEqual(files, want) {
				t.Fatal("snapshot lost a registered binary or captured a build output", files)
			}
			for name, content := range map[string]string{"out/firmware": elfHeader + "modeled firmware", "out/fresh": elfHeader + "registered, never staged"} {
				if size := r.Git("cat-file", "-s", fixed.ID+":"+name); size != fmt.Sprint(len(content)) {
					t.Fatal("registered binary changed in the snapshot", name, size)
				}
			}
			outputs, err := untrackedOutputs(ctx, r.Dir, command, executables)
			if err != nil || !reflect.DeepEqual(outputs, []string{"firmware/image", "out/tool"}) {
				t.Fatal("build outputs not reported exactly", outputs, err)
			}
			if status := r.Git("status", "--porcelain", "--untracked-files=all"); status != "?? firmware/image\n?? out/tool" {
				t.Fatalf("build outputs did not stay untracked in the copy: %q", status)
			}
		})
	}
}

// runCollaboration runs c as collaboration.execute does, with bounds that
// leave real checks on slow hosts enough time, and reopens its candidate.
func runCollaboration(t *testing.T, c *collaboration, id string) run.Result {
	t.Helper()
	_, prepared, git, input := runFixture(t)
	records, err := runstore.New(git.StateRoot)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	request := run.Request{ID: id, Project: "fixture", Prepared: prepared, Input: input, Mandate: run.Implement, MaxManagers: 5, Allowance: run.NewAllowance(time.Now().Add(5*time.Minute), 100), MessageTimeout: 3 * time.Minute, CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second}
	result, err := (&run.Executor{Sessions: c, Workspaces: git, Records: records}).Execute(ctx, request)
	if err != nil || result.Technical != run.Completed || result.Candidate == nil {
		t.Fatalf("Run did not complete: %s %v %v", result.Technical, result.Errors, err)
	}
	c.assertNoHostNarration(t)
	c.candidate = t.TempDir()
	if err = git.Reopen(ctx, *result.Candidate, c.candidate); err != nil {
		t.Fatal(err)
	}
	return result
}

// Scenarios 1 and 2: the model requires a binary artifact, and a/deep builds
// and registers it. Whether or not a/deep also stages it, its check passes, its
// handoff carries the binary to its parent, and the candidate holds its exact
// bytes, without any correction.
func TestRegisteredBinaryArtifactsReachTheCandidateStagedOrNot(t *testing.T) {
	binaries := map[string]string{"out/firmware": elfHeader + "registered, not staged", "out/staged": elfHeader + "registered and staged"}
	var check run.Check
	c := &collaboration{}
	c.script = func(ctx context.Context, m *collaborator, message run.Message) (bool, error) {
		if m.spec.Manager.Package != "a/deep" || len(message.Events) == 0 || message.Events[0].Assignment == nil {
			return false, nil
		}
		if err := m.handle(ctx, message.Events[0]); err != nil {
			return true, err
		}
		for name, content := range binaries {
			if err := m.write(name, content); err != nil {
				return true, err
			}
		}
		if err := m.write(".markitect/registry/a/deep/registry.yaml", withArtifacts(deepRegistry, "out/firmware", "out/staged")); err != nil {
			return true, err
		}
		if _, err := localCommand(ctx, m.spec.Workspace.Location().Directory, nil, "add", "--", "out/staged"); err != nil {
			return true, err
		}
		var err error
		if check, err = m.spec.Tools.Check(ctx); err != nil {
			return true, err
		}
		return true, m.judge(ctx, run.Submitted, nil)
	}
	result := runCollaboration(t, c, "registered-binaries")
	if !check.Valid || check.BuildOutputs != nil {
		t.Fatal("a registered binary failed its check or counted as a build output", check)
	}
	deep, a := c.manager("a/deep"), c.manager("a")
	if result.Decision != run.Submitted || len(deep.tasks) != 1 || deep.judgments != 1 || len(a.handoffs) != 1 {
		t.Fatal("a registered binary needed a correction", result.Decision, len(deep.tasks), deep.judgments, len(a.handoffs))
	}
	if h := a.handoffs[0]; h.Check == nil || !h.Check.Valid || h.Check.BuildOutputs != nil || !strings.Contains(h.Diff, "out/firmware") || !strings.Contains(h.Diff, "out/staged") {
		t.Fatal("the parent did not receive both binaries", h.Check, h.Diff)
	}
	for name, content := range binaries {
		if got := c.read(t, name); got != content {
			t.Fatal("candidate lost a registered binary or changed it", name)
		}
	}
}

// Scenario 3 with staging: a/deep stages its go build output. Accounting
// refuses the unregistered file, as before. a/deep corrects that within the
// normal flow by unstaging it; the binary stays in its copy as a build output,
// which the check and the handoff then list, and stays out of the candidate.
func TestStagedBuildOutputFailsAccountingUntilUnstaged(t *testing.T) {
	var checks []run.Check
	var handoffs []run.Handoff
	c := &collaboration{}
	c.script = func(ctx context.Context, m *collaborator, message run.Message) (bool, error) {
		if len(message.Events) == 0 || message.Events[0].Assignment == nil {
			return false, nil
		}
		switch m.spec.Manager.Package {
		case "a":
			if err := m.handle(ctx, message.Events[0]); err != nil {
				return true, err
			}
			m.started = true
			if err := m.spec.Tools.Delegate(ctx, "a/deep", "Realize your modeled responsibility.", nil); err != nil {
				return true, err
			}
			first, err := m.handoff(ctx)
			if err != nil {
				return true, err
			}
			if first.Check == nil || first.Check.Valid {
				return true, fmt.Errorf("a staged build output passed its check: %+v", first.Check)
			}
			findings := []run.Finding{{Text: first.Check.Diagnostic}}
			if err = m.spec.Tools.Review(ctx, "a/deep", first.Result.ID, run.Reject, findings); err != nil {
				return true, err
			}
			if err = m.spec.Tools.Delegate(ctx, "a/deep", "Correct your handoff.", findings); err != nil {
				return true, err
			}
			second, err := m.handoff(ctx)
			if err != nil {
				return true, err
			}
			handoffs = append(handoffs, first, second)
			if err = m.spec.Tools.Review(ctx, "a/deep", second.Result.ID, run.Approve, nil); err != nil {
				return true, err
			}
			return true, m.judge(ctx, run.Submitted, nil)
		case "a/deep":
			if err := m.handle(ctx, message.Events[0]); err != nil {
				return true, err
			}
			dir := m.spec.Workspace.Location().Directory
			if len(m.tasks) == 1 {
				if err := m.write("out/tool", elfHeader+"go build output"); err != nil {
					return true, err
				}
				if _, err := localCommand(ctx, dir, nil, "add", "--", "out/tool"); err != nil {
					return true, err
				}
			} else if _, err := localCommand(ctx, dir, nil, "rm", "--cached", "--quiet", "--", "out/tool"); err != nil {
				return true, err
			}
			check, err := m.spec.Tools.Check(ctx)
			if err != nil {
				return true, err
			}
			checks = append(checks, check)
			return true, m.judge(ctx, run.Submitted, nil)
		}
		return false, nil
	}
	result := runCollaboration(t, c, "staged-build-output")
	if result.Decision != run.Submitted || len(checks) != 2 || len(handoffs) != 2 || len(c.manager("a/deep").tasks) != 2 {
		t.Fatal("the correction did not complete within the normal flow", result.Decision, len(checks), len(handoffs))
	}
	staged, unstaged := checks[0], checks[1]
	if staged.Valid || !strings.Contains(staged.Diagnostic, "file.unexplained") || !strings.Contains(staged.Diagnostic, "out/tool") || staged.BuildOutputs != nil {
		t.Fatal("a staged build output was not refused as unaccounted", staged)
	}
	if !unstaged.Valid || !reflect.DeepEqual(unstaged.BuildOutputs, []string{"out/tool"}) {
		t.Fatal("the unstaged build output was not left out visibly", unstaged)
	}
	if h := handoffs[1]; h.Check == nil || !h.Check.Valid || !reflect.DeepEqual(h.Check.BuildOutputs, []string{"out/tool"}) {
		t.Fatal("the parent did not see what its child left out", h.Check)
	}
	if _, err := os.Stat(filepath.Join(c.candidate, "out/tool")); !os.IsNotExist(err) {
		t.Fatal("build output reached the candidate", err)
	}
}

// collaborationRuns executes every Run of a workflow with fresh controlled
// collaborators over one fixture's actual Git; a script takes over per mandate.
type collaborationRuns struct {
	git     *LocalGit
	records run.Records
	scripts map[run.Mandate]func(context.Context, *collaborator, run.Message) (bool, error)
	runs    map[run.Mandate]*collaboration
}

func (e *collaborationRuns) Execute(ctx context.Context, request run.Request) (run.Result, error) {
	c := &collaboration{script: e.scripts[request.Mandate]}
	e.runs[request.Mandate] = c
	return (&run.Executor{Sessions: c, Workspaces: e.git, Records: e.records}).Execute(ctx, request)
}

// Scenarios 3 and 5 and a directory registration, up to Verify. a/deep runs go
// build, which leaves out/tool; builds out/required, a binary the model
// requires that a/deep does not register; and builds firmware/image below its
// directory entry firmware/. The host cannot tell a forgotten artifact from a
// build output and does not try: none of the three enters the candidate, and
// each is listed wherever the candidate is judged. a/deep's check and its
// handoff to its parent list them, the stored Implement record keeps them, and
// Verify's Root receives them with its assignment and its check of the fixed
// input, so the independent review can find a missing required binary.
func TestLeftOutBuildOutputsAreVisibleUpToVerify(t *testing.T) {
	_, prepared, git, input := runFixture(t)
	records, err := runstore.New(git.StateRoot)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"firmware/image", "out/required", "out/tool"}
	var implementCheck, verifyCheck run.Check
	var verifyTask run.Assignment
	runs := &collaborationRuns{git: git, records: records, runs: map[run.Mandate]*collaboration{}, scripts: map[run.Mandate]func(context.Context, *collaborator, run.Message) (bool, error){
		run.Implement: func(ctx context.Context, m *collaborator, message run.Message) (bool, error) {
			if m.spec.Manager.Package != "a/deep" || len(message.Events) == 0 || message.Events[0].Assignment == nil {
				return false, nil
			}
			if err := m.handle(ctx, message.Events[0]); err != nil {
				return true, err
			}
			if err := os.MkdirAll(filepath.Join(m.spec.Workspace.Location().Directory, "firmware"), 0700); err != nil {
				return true, err
			}
			shard := deepRegistry + "  - path: firmware/\n    form: build\n    realizes: [a/deep/item]\n    why: The firmware images.\n"
			for _, file := range [][2]string{{"out/tool", elfHeader + "go build output"}, {"out/required", elfHeader + "required, not registered"}, {"firmware/README.md", "Firmware images.\n"}, {"firmware/image", elfHeader + "below a directory entry"}, {".markitect/registry/a/deep/registry.yaml", shard}} {
				if err := m.write(file[0], file[1]); err != nil {
					return true, err
				}
			}
			var err error
			if implementCheck, err = m.spec.Tools.Check(ctx); err != nil {
				return true, err
			}
			return true, m.judge(ctx, run.Submitted, nil)
		},
		run.Verify: func(ctx context.Context, m *collaborator, message run.Message) (bool, error) {
			if m.spec.Manager.Package != "." || len(message.Events) == 0 || message.Events[0].Assignment == nil {
				return false, nil
			}
			verifyTask = *message.Events[0].Assignment
			var err error
			verifyCheck, err = m.spec.Tools.Check(ctx)
			// Root then reviews as usual.
			return false, err
		},
	}}
	base := run.Request{ID: "left-out", Project: "fixture", Prepared: prepared, Input: input, MaxManagers: 5, Allowance: run.NewAllowance(time.Now().Add(10*time.Minute), 300), MessageTimeout: 3 * time.Minute, CleanupTimeout: 3 * time.Second, PersistenceTimeout: 3 * time.Second}
	ctx := context.Background()
	out, err := workflow.Apply(ctx, runs, base)
	if err != nil || len(out.Runs) != 3 || out.Final.Mandate != run.Verify || out.Final.Decision != run.Approve || out.Final.Candidate == nil {
		t.Fatalf("workflow did not reach Verify approval: %d runs, %s %s %v", len(out.Runs), out.Final.Mandate, out.Final.Decision, err)
	}
	for _, c := range runs.runs {
		c.assertNoHostNarration(t)
	}
	implement := out.Runs[1]
	if !implementCheck.Valid || !reflect.DeepEqual(implementCheck.BuildOutputs, want) {
		t.Fatal("a/deep's check did not list what it leaves out", implementCheck)
	}
	if a := runs.runs[run.Implement].manager("a"); len(a.handoffs) != 1 || a.handoffs[0].Check == nil || !reflect.DeepEqual(a.handoffs[0].Check.BuildOutputs, want) {
		t.Fatal("the parent's handoff did not list what its child left out", a.handoffs)
	}
	stored, err := records.Read(ctx, "fixture", implement.ID)
	if err != nil || handoffOf(stored, "a/deep").Check == nil || !reflect.DeepEqual(handoffOf(stored, "a/deep").Check.BuildOutputs, want) || !reflect.DeepEqual(implement.BuildOutputs(), want) {
		t.Fatal("the stored Implement record lost what the Run left out", err)
	}
	if !reflect.DeepEqual(verifyTask.BuildOutputs, want) || !verifyCheck.Valid || !reflect.DeepEqual(verifyCheck.BuildOutputs, want) {
		t.Fatal("Verify did not see what Implement left out of its fixed input", verifyTask, verifyCheck)
	}
	candidate := t.TempDir()
	if err = git.Reopen(ctx, *out.Final.Candidate, candidate); err != nil {
		t.Fatal(err)
	}
	for _, name := range want {
		if _, err := os.Stat(filepath.Join(candidate, filepath.FromSlash(name))); !os.IsNotExist(err) {
			t.Fatal("a left-out binary reached the candidate", name, err)
		}
	}
	if data, err := os.ReadFile(filepath.Join(candidate, "firmware/README.md")); err != nil || string(data) != "Firmware images.\n" {
		t.Fatal("the registered directory's text artifact did not reach the candidate", err)
	}
}
