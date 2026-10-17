package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

type publicationControl struct {
	calls   int
	failAt  int
	onCheck func(int)
}

func (c *publicationControl) Check(ctx context.Context) error {
	c.calls++
	if c.onCheck != nil {
		c.onCheck(c.calls)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.failAt > 0 && c.calls >= c.failAt {
		return errors.New("controller lost")
	}
	return nil
}

func publicationEntry(t *testing.T, p, text string) snapshot.Entry {
	t.Helper()
	e, err := snapshot.NewEntry(fixturePath(t, p), snapshot.Regular, []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func publicationFixture(t *testing.T) (*Repository, *testkit.Repo, apply.PublicationRequest) {
	t.Helper()
	repo := testkit.NewRepo(t)
	repo.Git("config", "core.filemode", "false")
	repo.Write("model.yaml", "formatVersion: 1\nid: publication-fixture\nroot: model\ndependencies: {}\n")
	repo.Write("model/root.yaml", "id: root\nkind: Package\nbody: root\nspec: {}\n")
	repo.Write(".gitignore", "ignored/\n.markitect/state/\n")
	repo.Write("host/instruction.txt", "instruction\n")
	selection := apply.InputSelection{Repository: repo.Dir, Local: apply.ModelSelection{Boundary: repo.Dir, ManifestPath: fixturePath(t, "model.yaml")}, Instructions: []apply.SourceLocation{{Boundary: repo.Dir, Path: fixturePath(t, "host/instruction.txt")}}}
	repo.Write("real/change.txt", "old\n")
	repo.Write("real/delete.txt", "delete\n")
	repo.Commit("Publication files")
	r := adapter(t)
	captured, err := r.Capture(context.Background(), apply.CaptureRequest{Selection: selection})
	if err != nil {
		t.Fatal(err)
	}
	target := captured.Raw.Census.Entries()
	target = slices.DeleteFunc(target, func(e snapshot.Entry) bool {
		return e.Path().String() == "real/delete.txt" || e.Path().String() == "real/change.txt"
	})
	target = append(target, publicationEntry(t, "real/change.txt", "new\n"), publicationEntry(t, "new/deep/add.txt", "added\n"))
	tree, err := snapshot.NewTree(target)
	if err != nil {
		t.Fatal(err)
	}
	req := apply.PublicationRequest{ID: "publication", Run: "run", Repository: repo.Dir, Guard: captured.Guard, Changes: snapshot.Compare(captured.Raw.Census, tree)}
	return r, repo, req
}

func TestPublisherExactChangesAndReadOnlyGitAdministration(t *testing.T) {
	r, repo, req := publicationFixture(t)
	beforeIndex, err := os.ReadFile(filepath.Join(repo.Dir, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	head := repo.Git("rev-parse", "HEAD")
	control := &publicationControl{}
	permit, err := r.Preflight(context.Background(), req, control)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := r.Publish(context.Background(), permit, control)
	if err != nil || receipt.Outcome != apply.PublicationApplied {
		t.Fatalf("%+v / %v", receipt, err)
	}
	if len(receipt.Paths) != 3 || slices.ContainsFunc(receipt.Paths, func(p apply.PublishedPath) bool { return p.State != apply.PathAfter }) {
		t.Fatal(receipt)
	}
	index, err := os.ReadFile(filepath.Join(repo.Dir, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	if string(index) != string(beforeIndex) || repo.Git("rev-parse", "HEAD") != head {
		t.Fatal("publication changed HEAD/index")
	}
	if control.calls < 4 {
		t.Fatal("missing live controller fences")
	}
	// Recovery only observes; it cannot invent the earlier final fence.
	observed, err := adapter(t).Inspect(context.Background(), req)
	if err != nil || observed.Outcome != apply.PublicationUnknown || !reflect.DeepEqual(observed.Paths, receipt.Paths) {
		t.Fatalf("%+v / %v", observed, err)
	}
}

func TestPublisherPermitCopiesAndBindsWholeRequest(t *testing.T) {
	for _, field := range []string{"id", "run", "repository", "guard", "changes", "caller-alias"} {
		t.Run(field, func(t *testing.T) {
			r, repo, req := publicationFixture(t)
			control := &publicationControl{}
			permit, err := r.Preflight(context.Background(), req, control)
			if err != nil {
				t.Fatal(err)
			}
			switch field {
			case "id":
				permit.Request.ID = "different"
			case "run":
				permit.Request.Run = "different"
			case "repository":
				permit.Request.Repository += "/sub"
			case "guard":
				permit.Request.Guard = "fake"
			case "changes":
				permit.Request.Changes = permit.Request.Changes[:1]
			case "caller-alias":
				req.Changes[0] = snapshot.Change{}
			}
			receipt, err := r.Publish(context.Background(), permit, control)
			if field == "caller-alias" {
				if err != nil || receipt.Outcome != apply.PublicationApplied {
					t.Fatalf("caller alias affected permit: %+v %v", receipt, err)
				}
			} else {
				if err == nil || receipt.Outcome != apply.PublicationFailed {
					t.Fatalf("tampered permit: %+v %v", receipt, err)
				}
				data, _ := os.ReadFile(filepath.Join(repo.Dir, "real", "change.txt"))
				if string(data) != "old\n" {
					t.Fatal("tampered permit wrote")
				}
			}
		})
	}
}

func TestPublisherStaleGuardControllerAndFinalOutsideDrift(t *testing.T) {
	for _, when := range []string{"before", "controller-before", "outside-after", "controller-after", "external-before", "index-after"} {
		t.Run(when, func(t *testing.T) {
			r, repo, req := publicationFixture(t)
			control := &publicationControl{}
			permit, err := r.Preflight(context.Background(), req, control)
			if err != nil {
				t.Fatal(err)
			}
			switch when {
			case "before":
				repo.Write("ignored/drift.txt", "drift")
			case "controller-before":
				control.failAt = control.calls + 1
			case "outside-after":
				r.observe = func(event, p string) {
					if event == "publication-after-write" {
						repo.Write("ignored/drift.txt", "drift")
					}
				}
			case "controller-after":
				r.observe = func(event, p string) {
					if event == "publication-after-write" {
						control.failAt = control.calls + 1
					}
				}
			case "external-before":
				repo.Write("host/instruction.txt", "changed")
			case "index-after":
				r.observe = func(event, p string) {
					if event == "publication-after-write" {
						repo.Git("update-index", "--assume-unchanged", "model/root.yaml")
					}
				}
			}
			receipt, err := r.Publish(context.Background(), permit, control)
			want := apply.PublicationStale
			if strings.HasSuffix(when, "after") {
				want = apply.PublicationUnknown
			}
			if err == nil || receipt.Outcome != want || len(receipt.Paths) != len(req.Changes) {
				t.Fatalf("%+v / %v", receipt, err)
			}
			changed := !strings.HasPrefix(when, "controller-")
			if errors.Is(err, apply.ErrCaptureChanged) != changed {
				t.Fatalf("capture classification changed=%v: %v", changed, err)
			}
			if changed && !errors.Is(err, ErrChanged) && !errors.Is(err, ErrDirty) {
				t.Fatalf("lost adapter drift cause: %v", err)
			}
		})
	}
}

func TestPublisherPreflightClassifiesObservedCaptureChanges(t *testing.T) {
	for _, kind := range []string{"head", "tracked-file", "external-input", "root-identity", "during-acquisition"} {
		t.Run(kind, func(t *testing.T) {
			r, repo, req := publicationFixture(t)
			want := ErrChanged
			switch kind {
			case "head":
				repo.Commit("New HEAD with unchanged files")
			case "tracked-file":
				repo.Write("real/change.txt", "other writer\n")
				want = ErrDirty
			case "external-input":
				dir := t.TempDir()
				p := filepath.Join(dir, "instruction.txt")
				if err := os.WriteFile(p, []byte("original\n"), 0600); err != nil {
					t.Fatal(err)
				}
				selection := r.captures[req.Guard].selection
				selection.Instructions = []apply.SourceLocation{{Boundary: dir, Path: fixturePath(t, "instruction.txt")}}
				captured, err := r.Capture(context.Background(), apply.CaptureRequest{Selection: selection})
				if err != nil {
					t.Fatal(err)
				}
				req.Guard = captured.Guard
				if err := os.WriteFile(p, []byte("changed\n"), 0600); err != nil {
					t.Fatal(err)
				}
			case "root-identity":
				original := filepath.Join(t.TempDir(), "original")
				if err := os.Rename(repo.Dir, original); err != nil {
					t.Fatal(err)
				}
				copyBoundary(t, original, repo.Dir)
			case "during-acquisition":
				mutated := false
				r.observe = func(event, p string) {
					if event == "before-confirm" && !mutated {
						mutated = true
						repo.Write("ignored/new.txt", "drift\n")
					}
				}
			}
			permit, err := r.Preflight(context.Background(), req, &publicationControl{})
			if !errors.Is(err, apply.ErrCaptureChanged) || !errors.Is(err, want) || permit.Token != "" {
				t.Fatalf("capture drift did not retain consumer and adapter errors: %+v %v", permit, err)
			}
		})
	}
}

func TestPublisherPreflightKeepsUnknownFailuresDistinctFromCaptureChange(t *testing.T) {
	for _, kind := range []string{"budget", "canceled", "git-error"} {
		t.Run(kind, func(t *testing.T) {
			r, repo, req := publicationFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "budget":
				r.limits.Entries = 1
			case "canceled":
				cancel()
			case "git-error":
				repo.Write(".git/config", "[invalid\n")
			}
			permit, err := r.Preflight(ctx, req, &publicationControl{})
			if err == nil || errors.Is(err, apply.ErrCaptureChanged) || permit.Token != "" {
				t.Fatalf("unknown failure misclassified as drift: %+v %v", permit, err)
			}
			if kind == "budget" && !errors.Is(err, ErrLimit) || kind == "canceled" && !errors.Is(err, context.Canceled) {
				t.Fatalf("lost original failure: %v", err)
			}
		})
	}
	for _, cause := range []error{ErrUnsafe, ErrLimit, context.Canceled, os.ErrPermission, errors.New("unknown I/O")} {
		// Acquisition can join a drift observation with another failure. The
		// incomplete observation must not become a confirmed capture change.
		err := fmt.Errorf("confirmation: %w", errors.Join(ErrChanged, cause))
		got := publicationCaptureError(err)
		if got != err || errors.Is(got, apply.ErrCaptureChanged) || !errors.Is(got, cause) {
			t.Fatalf("mixed acquisition failure misclassified: %v", got)
		}
	}
}

func TestPublisherBeforeWriteDriftRetainsFailedReceipt(t *testing.T) {
	for _, p := range []string{"new/deep/add.txt", "real/change.txt"} {
		t.Run(p, func(t *testing.T) {
			r, repo, req := publicationFixture(t)
			req.Changes = slices.DeleteFunc(req.Changes, func(c snapshot.Change) bool { return c.Path().String() != p })
			permit, err := r.Preflight(context.Background(), req, &publicationControl{})
			if err != nil {
				t.Fatal(err)
			}
			r.observe = func(event, path string) {
				if event == "publication-before-write" {
					repo.Write(path, "other writer\n")
				}
			}
			receipt, err := r.Publish(context.Background(), permit, &publicationControl{})
			if !errors.Is(err, apply.ErrCaptureChanged) || !errors.Is(err, ErrChanged) || receipt.Outcome != apply.PublicationFailed || len(receipt.Paths) != 1 || receipt.Paths[0].State != apply.PathOther {
				t.Fatalf("before-write capture drift lost classification or receipt: %+v %v", receipt, err)
			}
			data, readErr := os.ReadFile(filepath.Join(repo.Dir, filepath.FromSlash(p)))
			if readErr != nil || string(data) != "other writer\n" {
				t.Fatalf("refused publication changed other writer's file: %q %v", data, readErr)
			}
		})
	}
}

func TestPublisherDoesNotUseCallerChangeSlicesAfterBinding(t *testing.T) {
	r, _, req := publicationFixture(t)
	control := &publicationControl{onCheck: func(n int) {
		if n == 1 {
			req.Changes[0] = snapshot.Change{}
		}
	}}
	permit, err := r.Preflight(context.Background(), req, control)
	if err != nil {
		t.Fatal(err)
	}
	r.observe = func(event, p string) {
		if event == "publication-before-write" {
			permit.Request.Changes[0] = snapshot.Change{}
		}
	}
	receipt, err := r.Publish(context.Background(), permit, &publicationControl{})
	if err != nil || receipt.Outcome != apply.PublicationApplied {
		t.Fatalf("mutable caller slice changed binding: %+v %v", receipt, err)
	}
}

func TestPublisherRefusesRedirectedParentAtPreflightAndWrite(t *testing.T) {
	for _, when := range []string{"preflight", "write"} {
		t.Run(when, func(t *testing.T) {
			r, repo, req := publicationFixture(t)
			outside := t.TempDir()
			redirectedParent := filepath.Join(repo.Dir, "new")
			link := func() {
				if err := os.Symlink(outside, redirectedParent); err != nil {
					if runtime.GOOS != "windows" {
						t.Fatal(err)
					}
					if out, err := exec.Command("cmd", "/c", "mklink", "/J", redirectedParent, outside).CombinedOutput(); err != nil {
						t.Fatalf("create redirect fixture: %v %s", err, out)
					}
				}
			}
			if when == "preflight" {
				link()
				if _, err := r.Preflight(context.Background(), req, &publicationControl{}); err == nil || errors.Is(err, apply.ErrCaptureChanged) {
					t.Fatalf("redirected parent accepted or classified as capture drift: %v", err)
				}
			} else {
				permit, err := r.Preflight(context.Background(), req, &publicationControl{})
				if err != nil {
					t.Fatal(err)
				}
				r.observe = func(event, p string) {
					if event == "publication-before-write" && p == "new/deep/add.txt" {
						link()
					}
				}
				receipt, err := r.Publish(context.Background(), permit, &publicationControl{})
				if err == nil || receipt.Outcome != apply.PublicationUnknown || receipt.Paths[0].State != apply.PathOther {
					t.Fatalf("redirected write: %+v %v", receipt, err)
				}
			}
			names, err := os.ReadDir(outside)
			if err != nil || len(names) != 0 {
				t.Fatalf("redirect target changed: %v %v", names, err)
			}
		})
	}
}

func TestPublisherPartialWritesRetainedAndClassified(t *testing.T) {
	r, repo, req := publicationFixture(t)
	control := &publicationControl{}
	permit, err := r.Preflight(context.Background(), req, control)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	r.observe = func(event, p string) {
		if event == "publication-before-write" {
			count++
			if count == 2 {
				repo.Write(p, "other writer")
			}
		}
	}
	receipt, err := r.Publish(context.Background(), permit, control)
	if err == nil || receipt.Outcome != apply.PublicationUnknown || receipt.Paths[0].State != apply.PathAfter || receipt.Paths[1].State != apply.PathOther || receipt.Paths[2].State != apply.PathBefore {
		t.Fatalf("%+v / %v", receipt, err)
	}
	data, _ := os.ReadFile(filepath.Join(repo.Dir, "new", "deep", "add.txt"))
	if string(data) != "added\n" {
		t.Fatal("partial write removed")
	}
}

func TestPublisherGitCheckoutAndCleanLineEndings(t *testing.T) {
	r, repo, req := publicationFixture(t)
	// Explicit text conversion also applies to new paths without an index entry.
	repo.Write(".gitattributes", "real/change.txt text eol=crlf\nnew/deep/add.txt text eol=crlf\n")
	repo.Commit("Checkout policy")
	selection := r.captures[req.Guard].selection
	captured, err := r.Capture(context.Background(), apply.CaptureRequest{Selection: selection})
	if err != nil {
		t.Fatal(err)
	}
	req.Guard = captured.Guard
	permit, err := r.Preflight(context.Background(), req, &publicationControl{})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := r.Publish(context.Background(), permit, &publicationControl{})
	if err != nil || receipt.Outcome != apply.PublicationApplied {
		t.Fatalf("%+v / %v", receipt, err)
	}
	data, _ := os.ReadFile(filepath.Join(repo.Dir, "real", "change.txt"))
	if string(data) != "new\r\n" {
		t.Fatalf("not Git checkout form: %q", data)
	}
	g, err := identifyGit(context.Background(), repo.Dir)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := g.run("hash-object", "--path=real/change.txt", "real/change.txt")
	if err != nil {
		t.Fatal(err)
	}
	c, err := newPublicationConverter(context.Background(), r.limits)
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	expected, err := c.object([]byte("new\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(actual)) != expected {
		t.Fatal("owner Git clean readback differs")
	}
}

func TestPublisherEmptyChangeSetAndUnissuedGuard(t *testing.T) {
	r, _, req := publicationFixture(t)
	req.Changes = nil
	permit, err := r.Preflight(context.Background(), req, &publicationControl{})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := r.Publish(context.Background(), permit, &publicationControl{})
	if err != nil || receipt.Outcome != apply.PublicationNoChange || len(receipt.Paths) != 0 {
		t.Fatalf("%+v / %v", receipt, err)
	}
	req.Guard = "gitrepo-capture/1:invented"
	if _, err := r.Preflight(context.Background(), req, &publicationControl{}); !errors.Is(err, ErrUnsafe) || errors.Is(err, apply.ErrCaptureChanged) {
		t.Fatalf("invented guard accepted or classified as capture drift: %v", err)
	}
}

func TestPublisherRefusesFiltersModesAndAliasesBeforeWriting(t *testing.T) {
	for _, kind := range []string{"filter", "encoding", "ident", "mode", "alias", "empty-dir-alias", "zero", "duplicate", "before", "attributes", "attribute-alias", "split-index"} {
		t.Run(kind, func(t *testing.T) {
			r, repo, req := publicationFixture(t)
			switch kind {
			case "split-index":
				repo.Git("update-index", "--split-index")
			case "filter", "encoding", "ident":
				attr := map[string]string{"filter": "filter=unsafe", "encoding": "working-tree-encoding=UTF-16", "ident": "ident"}[kind]
				repo.Write(".gitattributes", "*.txt "+attr+"\n")
				repo.Git("add", "--", ".gitattributes")
				repo.Git("commit", "--quiet", "--no-verify", "--message", "Unsafe checkout policy")
				selection := r.captures[req.Guard].selection
				captured, err := r.Capture(context.Background(), apply.CaptureRequest{Selection: selection})
				if err != nil {
					t.Fatal(err)
				}
				req.Guard = captured.Guard
			case "mode":
				before, _ := req.Changes[1].Before()
				after, err := snapshot.NewEntry(before.Path(), snapshot.Executable, []byte("new\n"))
				if err != nil {
					t.Fatal(err)
				}
				b, _ := snapshot.NewTree([]snapshot.Entry{before})
				a, _ := snapshot.NewTree([]snapshot.Entry{after})
				req.Changes = snapshot.Compare(b, a)
			case "alias":
				a, _ := snapshot.NewTree([]snapshot.Entry{publicationEntry(t, "REAL/change.txt", "new\n")})
				req.Changes = snapshot.Compare(snapshot.Tree{}, a)
			case "empty-dir-alias":
				if err := os.Mkdir(filepath.Join(repo.Dir, "NEW"), 0755); err != nil {
					t.Fatal(err)
				}
				// Empty directories are not in the capture census but must still be safe.
				selection := r.captures[req.Guard].selection
				captured, err := r.Capture(context.Background(), apply.CaptureRequest{Selection: selection})
				if err != nil {
					t.Fatal(err)
				}
				req.Guard = captured.Guard
			case "zero":
				req.Changes = []snapshot.Change{{}}
			case "duplicate":
				req.Changes = append(req.Changes, req.Changes[0])
			case "before":
				b, _ := snapshot.NewTree([]snapshot.Entry{publicationEntry(t, "real/change.txt", "wrong before")})
				a, _ := snapshot.NewTree([]snapshot.Entry{publicationEntry(t, "real/change.txt", "new\n")})
				req.Changes = snapshot.Compare(b, a)
			case "attributes", "attribute-alias":
				name := ".gitattributes"
				if kind == "attribute-alias" {
					name = ".GITATTRIBUTES"
				}
				a, _ := snapshot.NewTree([]snapshot.Entry{publicationEntry(t, name, "*.txt text\n")})
				req.Changes = snapshot.Compare(snapshot.Tree{}, a)
			}
			if _, err := r.Preflight(context.Background(), req, &publicationControl{}); err == nil || errors.Is(err, apply.ErrCaptureChanged) {
				t.Fatalf("invalid request accepted or classified as capture drift: %v", err)
			}
			data, _ := os.ReadFile(filepath.Join(repo.Dir, "real", "change.txt"))
			if string(data) != "old\n" {
				t.Fatal("preflight wrote owner file")
			}
		})
	}
}

func TestPublisherDoesNotTruncateOtherHardLinks(t *testing.T) {
	r, repo, req := publicationFixture(t)
	outside := filepath.Join(t.TempDir(), "linked.txt")
	if err := os.Link(filepath.Join(repo.Dir, "real", "change.txt"), outside); err != nil {
		t.Fatal(err)
	}
	permit, err := r.Preflight(context.Background(), req, &publicationControl{})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := r.Publish(context.Background(), permit, &publicationControl{})
	if err != nil || receipt.Outcome != apply.PublicationApplied {
		t.Fatalf("%+v / %v", receipt, err)
	}
	data, err := os.ReadFile(outside)
	if err != nil || string(data) != "old\n" {
		t.Fatalf("outside hard link changed: %q %v", data, err)
	}
}

func TestPublisherWriteErrorWithAllPathsAfterIsUnknown(t *testing.T) {
	r, repo, req := publicationFixture(t)
	req.Changes = slices.DeleteFunc(req.Changes, func(c snapshot.Change) bool { return c.Path().String() != "real/change.txt" })
	permit, err := r.Preflight(context.Background(), req, &publicationControl{})
	if err != nil {
		t.Fatal(err)
	}
	// Replace only the directory identity after the file mutation. Every file
	// keeps its expected content, but the write's original directory fence fails.
	moved := filepath.Join(t.TempDir(), "original-real")
	r.observe = func(event, p string) {
		if event != "publication-file-written" {
			return
		}
		if err := os.Rename(filepath.Join(repo.Dir, "real"), moved); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"change.txt", "delete.txt"} {
			data, err := os.ReadFile(filepath.Join(moved, name))
			if err != nil {
				t.Fatal(err)
			}
			repo.Write("real/"+name, string(data))
		}
	}
	receipt, err := r.Publish(context.Background(), permit, &publicationControl{})
	if err == nil || receipt.Outcome != apply.PublicationUnknown || len(receipt.Paths) != 1 || receipt.Paths[0].State != apply.PathAfter {
		t.Fatalf("write error completion was misclassified: %+v %v", receipt, err)
	}
}
