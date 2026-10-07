package government

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPromoteCompareAndSwapWritesBoundIntent(t *testing.T) {
	fixture := newPromotionFixture(t)
	request := fixture.request()
	result, err := Promote(context.Background(), request)
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}
	if result.Status != "promoted" || result.ActualActive != request.NewCommit || result.ExpectedOld != request.ExpectedOld || result.NewCommit != request.NewCommit {
		t.Fatalf("unexpected promotion result: %+v", result)
	}
	if len(result.FencingToken) != 64 || result.IntentPath == "" {
		t.Fatalf("promotion result lacks fencing token or intent path: %+v", result)
	}
	intentBytes, err := os.ReadFile(result.IntentPath)
	if err != nil {
		t.Fatalf("read durable intent: %v", err)
	}
	var intent promotionIntent
	if err := json.Unmarshal(intentBytes, &intent); err != nil {
		t.Fatalf("decode intent: %v", err)
	}
	if intent.ExpectedOld != request.ExpectedOld || intent.NewCommit != request.NewCommit || intent.MaterialCandidateID != request.MaterialCandidateID || intent.EvidenceID != request.EvidenceID || intent.DecisionID != request.DecisionID || intent.IdempotencyKey != request.IdempotencyKey || intent.FencingToken != result.FencingToken {
		t.Fatalf("intent does not retain promotion bindings: %+v", intent)
	}
	active := fixture.git("rev-parse", request.ActiveRef)
	if active != request.NewCommit {
		t.Fatalf("active ref = %s, want %s", active, request.NewCommit)
	}
}

func TestPromoteRejectsStaleBase(t *testing.T) {
	fixture := newPromotionFixture(t)
	request := fixture.request()
	fixture.git("update-ref", request.ActiveRef, request.NewCommit)
	result, err := Promote(context.Background(), request)
	if err == nil || result.Status != "stale-base" || result.ActualActive != request.NewCommit {
		t.Fatalf("Promote() = (%+v, %v), want stale-base", result, err)
	}
	if result.IntentPath != "" {
		t.Fatalf("stale base should be rejected before creating intent: %+v", result)
	}
}

func TestPromoteRejectsTreeAndParentMismatch(t *testing.T) {
	t.Run("tree mismatch", func(t *testing.T) {
		fixture := newPromotionFixture(t)
		request := fixture.request()
		request.ExpectedTreeID = fixture.oldTree
		if result, err := Promote(context.Background(), request); err == nil || !strings.Contains(err.Error(), "tree differs") || result.Status != "rejected" {
			t.Fatalf("Promote() = (%+v, %v), want tree mismatch rejection", result, err)
		}
	})
	t.Run("parent mismatch", func(t *testing.T) {
		fixture := newPromotionFixture(t)
		request := fixture.request()
		other := fixture.git("commit-tree", request.ExpectedTreeID, "-m", "unrelated root")
		request.NewCommit = other
		request.ExpectedTreeID = fixture.git("rev-parse", other+"^{tree}")
		if result, err := Promote(context.Background(), request); err == nil || !strings.Contains(err.Error(), "expected active commit as its parent") || result.Status != "rejected" {
			t.Fatalf("Promote() = (%+v, %v), want parent mismatch rejection", result, err)
		}
	})
}

func TestPromoteRejectsUserBranchAndCheckedOutManagedRef(t *testing.T) {
	t.Run("user branch", func(t *testing.T) {
		fixture := newPromotionFixture(t)
		request := fixture.request()
		request.ActiveRef = "refs/heads/main"
		if result, err := Promote(context.Background(), request); err == nil || result.Status != "rejected" {
			t.Fatalf("Promote() = (%+v, %v), want user branch rejection", result, err)
		}
	})
	t.Run("checked out active ref", func(t *testing.T) {
		fixture := newPromotionFixture(t)
		request := fixture.request()
		fixture.git("symbolic-ref", "HEAD", request.ActiveRef)
		if result, err := Promote(context.Background(), request); err == nil || !strings.Contains(err.Error(), "checked out") || result.Status != "rejected" {
			t.Fatalf("Promote() = (%+v, %v), want checked-out ref rejection", result, err)
		}
	})
}

func TestPromoteRequiresExclusiveCommonDirectoryLock(t *testing.T) {
	fixture := newPromotionFixture(t)
	request := fixture.request()
	lockPath := filepath.Join(fixture.repo, ".git", promotionLockName(request.ActiveRef))
	if err := os.WriteFile(lockPath, []byte("held by another writer\n"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := Promote(context.Background(), request)
	if err == nil || !strings.Contains(err.Error(), "exclusive repository promotion lock") || result.Status != "rejected" {
		t.Fatalf("Promote() = (%+v, %v), want lock rejection", result, err)
	}
	if active := fixture.git("rev-parse", request.ActiveRef); active != request.ExpectedOld {
		t.Fatalf("active ref changed despite held lock: %s", active)
	}
}

func TestPromoteSuppressesConfiguredReferenceTransactionHook(t *testing.T) {
	fixture := newPromotionFixture(t)
	request := fixture.request()
	hooks := filepath.Join(fixture.repo, "government-test-hooks")
	if err := os.Mkdir(hooks, 0700); err != nil {
		t.Fatal(err)
	}
	hook := "#!/bin/sh\n" +
		"if [ \"$1\" = prepared ] && grep -q 'refs/markitect/government/active/project' ; then\n" +
		"  git -c core.hooksPath= update-ref refs/heads/main \"$PROMOTION_TEST_TARGET\"\n" +
		"fi\n"
	if err := os.WriteFile(filepath.Join(hooks, "reference-transaction"), []byte(hook), 0600); err != nil {
		t.Fatal(err)
	}
	fixture.git("config", "core.hooksPath", filepath.ToSlash(hooks))
	userHeadBefore := fixture.git("rev-parse", "refs/heads/main")
	if userHeadBefore != request.NewCommit || userHeadBefore == request.ExpectedOld {
		t.Fatalf("fixture user branch precondition: main=%s new=%s old=%s", userHeadBefore, request.NewCommit, request.ExpectedOld)
	}
	t.Setenv("PROMOTION_TEST_TARGET", request.ExpectedOld)
	fixture.git("update-ref", request.ActiveRef, request.NewCommit, request.ExpectedOld)
	if changed := fixture.git("rev-parse", "refs/heads/main"); changed != request.ExpectedOld {
		t.Fatalf("hook probe did not move the user branch: main=%s", changed)
	}
	fixture.git("update-ref", request.ActiveRef, request.ExpectedOld, request.NewCommit)
	fixture.git("update-ref", "refs/heads/main", userHeadBefore, request.ExpectedOld)
	if restored := fixture.git("rev-parse", "refs/heads/main"); restored != userHeadBefore {
		t.Fatalf("hook probe did not restore the user branch: main=%s", restored)
	}
	result, err := Promote(context.Background(), request)
	if err != nil {
		t.Fatalf("Promote() with configured reference-transaction hook: %v", err)
	}
	if result.Status != "promoted" {
		t.Fatalf("promotion status = %q, want promoted", result.Status)
	}
	if actual := fixture.git("rev-parse", "refs/heads/main"); actual != userHeadBefore {
		t.Fatalf("configured hook moved user branch from %s to %s", userHeadBefore, actual)
	}
	if actual := fixture.git("rev-parse", request.ActiveRef); actual != request.NewCommit {
		t.Fatalf("managed ref = %s, want %s", actual, request.NewCommit)
	}
}

func TestRecoverPromotionUsesExactRefHistoryAfterActiveAdvances(t *testing.T) {
	fixture := newPromotionFixture(t)
	request := fixture.request()
	result, err := Promote(context.Background(), request)
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}
	if err := os.Remove(result.CompletionPath); err != nil {
		t.Fatalf("remove completion to simulate crash window: %v", err)
	}
	advanced := fixture.git("commit-tree", request.ExpectedTreeID, "-p", request.NewCommit, "-m", "later active revision")
	fixture.git("update-ref", "--create-reflog", request.ActiveRef, advanced, request.NewCommit)
	recovered, err := RecoverPromotion(context.Background(), request)
	if err != nil {
		t.Fatalf("RecoverPromotion() error = %v", err)
	}
	if recovered.Status != "promoted" || recovered.ActualActive != advanced || recovered.FencingToken != result.FencingToken {
		t.Fatalf("recovery did not recognize exact prior edge and later active ref: %+v", recovered)
	}
	if active := fixture.git("rev-parse", request.ActiveRef); active != advanced {
		t.Fatalf("recovery moved active ref to %s, want %s", active, advanced)
	}
	again, err := RecoverPromotion(context.Background(), request)
	if err != nil || again.Status != "promoted" || again.ActualActive != advanced {
		t.Fatalf("repeat recovery = (%+v, %v)", again, err)
	}
	if active := fixture.git("rev-parse", request.ActiveRef); active != advanced {
		t.Fatalf("repeat recovery changed active ref to %s", active)
	}
}

func TestRecoverPromotionLeavesPreCASIntentIncomplete(t *testing.T) {
	fixture := newPromotionFixture(t)
	request := fixture.request()
	intent := promotionIntent{Version: 2, ExpectedOld: request.ExpectedOld, NewCommit: request.NewCommit, ExpectedTreeID: request.ExpectedTreeID,
		MaterialCandidateID: request.MaterialCandidateID, EvidenceID: request.EvidenceID, DecisionID: request.DecisionID,
		IdempotencyKey: request.IdempotencyKey, ActiveRef: request.ActiveRef, FencingToken: strings.Repeat("a", 64), CreatedAt: time.Now().UTC()}
	path := filepath.Join(fixture.state, promotionIntentName(request.IdempotencyKey))
	if err := writePromotionIntent(path, intent); err != nil {
		t.Fatal(err)
	}
	recovered, err := RecoverPromotion(context.Background(), request)
	if err == nil || recovered.Status != "incomplete" || !strings.Contains(err.Error(), "no exact tokenized promotion edge") {
		t.Fatalf("RecoverPromotion() = (%+v, %v), want incomplete pre-CAS effect", recovered, err)
	}
	if active := fixture.git("rev-parse", request.ActiveRef); active != request.ExpectedOld {
		t.Fatalf("recovery replayed CAS: active=%s", active)
	}
	if _, err := os.Stat(recovered.CompletionPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("pre-CAS recovery wrote completion: %v", err)
	}
	wrongTree := request
	wrongTree.ExpectedTreeID = fixture.oldTree
	if result, err := RecoverPromotion(context.Background(), wrongTree); err == nil || result.Status != "rejected" {
		t.Fatalf("recovery accepted a changed candidate tree binding: (%+v, %v)", result, err)
	}
}

func TestAcquireLeaseExcludesLiveRunnerAndRecognizesOnlySameHostRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "writer.lease")
	first, err := AcquireLease(path)
	if err != nil {
		t.Fatal(err)
	}
	if second, err := AcquireLease(path); err == nil {
		_ = second.Close()
		t.Fatal("second live runner acquired the lease")
	}
	firstToken := first.Token()
	if err := first.Verify(); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	second, err := AcquireLease(path)
	if err != nil {
		t.Fatalf("same-host orphan takeover was not proven by OS exclusion: %v", err)
	}
	if second.Token() == firstToken {
		t.Fatal("new lease reused the previous fencing token")
	}
	_ = second.Close()
	if err := os.WriteFile(path, []byte("legacy lock format\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if lease, err := AcquireLease(path); err == nil {
		_ = lease.Close()
		t.Fatal("unrecognized legacy lock was taken over")
	}
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if lease, err := AcquireLease(path); err == nil {
		_ = lease.Close()
		t.Fatal("empty preexisting lock file was treated as a fresh lease")
	}
}

func TestValidReflogTimestampRejectsMalformedValues(t *testing.T) {
	for _, value := range []string{"", "+", "-", "999999999999999999999999999999999", "1x"} {
		if validReflogTimestamp(value) {
			t.Errorf("validReflogTimestamp(%q) = true, want false", value)
		}
	}
	for _, value := range []string{"0", "1791401677", "-1"} {
		if !validReflogTimestamp(value) {
			t.Errorf("validReflogTimestamp(%q) = false, want true", value)
		}
	}
}

func TestOperationalDirectoryAndRecordAreDurablyPublished(t *testing.T) {
	parent := t.TempDir()
	directory, err := CreateOperationalDirectory(parent, "government-run-")
	if err != nil {
		t.Fatalf("CreateOperationalDirectory() error = %v", err)
	}
	resolved, err := ResolveOperationalDirectory(directory)
	if err != nil || resolved != directory {
		t.Fatalf("ResolveOperationalDirectory() = (%q, %v), want %q", resolved, err, directory)
	}
	path := filepath.Join(directory, "report.json")
	want := []byte("{\"status\":\"incomplete\"}\n")
	if err := WriteOperationalRecord(path, want); err != nil {
		t.Fatalf("WriteOperationalRecord() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(want) {
		t.Fatalf("operational record = %q, %v; want %q", got, err, want)
	}
	if err := WriteOperationalRecord(path, want); err == nil {
		t.Fatal("WriteOperationalRecord() replaced an existing record")
	}
}

func TestResolveOperationalDirectoryNormalizesExtendedPaths(t *testing.T) {
	directory := t.TempDir()
	want, err := ResolveOperationalDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ResolveOperationalDirectory(promotionExtendedPathForTest(directory))
	if err != nil || !samePromotionPathSpelling(got, want) {
		t.Fatalf("extended DOS path resolved to (%q, %v), want canonical %q", got, err, want)
	}
	unc := promotionExtendedPathForTest(`\\server\share\directory`)
	if strings.HasPrefix(strings.ToLower(unc), `\\?\`) {
		got, err := normalizePromotionPath(unc)
		if err != nil || got != `\\server\share\directory` {
			t.Fatalf("extended UNC path normalized to (%q, %v)", got, err)
		}
	}
}

func TestPromoteRejectsStateInsideRepositoryThrough8Dot3Alias(t *testing.T) {
	fixture := newPromotionFixture(t)
	insideState := filepath.Join(fixture.repo, "operational state")
	if err := os.Mkdir(insideState, 0700); err != nil {
		t.Fatal(err)
	}
	shortState, available := promotionShortPathForTest(insideState)
	if !available {
		t.Skip("filesystem does not provide a distinct 8.3 path")
	}
	request := fixture.request()
	request.StateDirectory = shortState
	result, err := Promote(context.Background(), request)
	if err == nil || result.Status != "rejected" || result.IntentPath != "" {
		t.Fatalf("Promote() with aliased in-repository state = (%+v, %v), want rejection before intent", result, err)
	}
}

type promotionFixture struct {
	t       *testing.T
	repo    string
	state   string
	old     string
	oldTree string
	new     string
	newTree string
}

func newPromotionFixture(t *testing.T) *promotionFixture {
	t.Helper()
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	state := filepath.Join(root, "state")
	if err := os.Mkdir(repo, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(state, 0700); err != nil {
		t.Fatal(err)
	}
	fixture := &promotionFixture{t: t, repo: repo, state: state}
	fixture.git("init", "-b", "main")
	fixture.git("config", "user.name", "Promotion Test")
	fixture.git("config", "user.email", "promotion@example.invalid")
	if err := os.WriteFile(filepath.Join(repo, "file.txt"), []byte("base\n"), 0600); err != nil {
		t.Fatal(err)
	}
	fixture.git("add", "file.txt")
	fixture.git("commit", "-m", "base")
	fixture.old = fixture.git("rev-parse", "HEAD")
	fixture.oldTree = fixture.git("rev-parse", "HEAD^{tree}")
	if err := os.WriteFile(filepath.Join(repo, "file.txt"), []byte("candidate\n"), 0600); err != nil {
		t.Fatal(err)
	}
	fixture.git("commit", "-am", "candidate")
	fixture.new = fixture.git("rev-parse", "HEAD")
	fixture.newTree = fixture.git("rev-parse", "HEAD^{tree}")
	fixture.git("update-ref", "refs/markitect/government/active/project", fixture.old)
	return fixture
}

func (f *promotionFixture) request() PromotionRequest {
	f.t.Helper()
	return PromotionRequest{
		Repo: f.repo, ActiveRef: "refs/markitect/government/active/project", ExpectedOld: f.old,
		NewCommit: f.new, ExpectedTreeID: f.newTree,
		MaterialCandidateID: testPromotionDigest("candidate"), EvidenceID: testPromotionDigest("evidence"),
		DecisionID: testPromotionDigest("decision"), StateDirectory: f.state,
		IdempotencyKey: "g2-test-" + time.Now().UTC().Format("20060102T150405.000000000"),
	}
}

func (f *promotionFixture) git(args ...string) string {
	f.t.Helper()
	gitArgs := append([]string{"-C", f.repo}, args...)
	command := exec.Command("git", gitArgs...)
	command.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=NUL")
	output, err := command.CombinedOutput()
	if err != nil {
		f.t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func testPromotionDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func TestRecoverPromotionRejectsMalformedHistoryIdentity(t *testing.T) {
	fixture := newPromotionFixture(t)
	request := fixture.request()
	result, err := Promote(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(fixture.repo, ".git", "logs", filepath.FromSlash(request.ActiveRef))
	line := request.ExpectedOld + " " + request.NewCommit + " arbitrary 0 +0000\tmarkitect government promotion " + result.FencingToken + "\n"
	if err := os.WriteFile(path, []byte(line), 0600); err != nil {
		t.Fatal(err)
	}
	recovered, err := RecoverPromotion(context.Background(), request)
	if err == nil || recovered.Status == "promoted" {
		t.Fatalf("malformed history counted as promotion: %+v: %v", recovered, err)
	}
}
