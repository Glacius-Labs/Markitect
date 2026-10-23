package gitrepo

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestForbiddenGitProgramHelper(t *testing.T) {
	for i, arg := range os.Args {
		if arg == "--gitrepo-forbidden-program" && i+1 < len(os.Args) {
			_ = os.WriteFile(os.Args[i+1], []byte("unexpected program execution"), 0o644)
			_, _ = io.Copy(os.Stdout, os.Stdin)
			os.Exit(0)
		}
	}
}

func TestBoundedOutputProcess(t *testing.T) {
	for i, arg := range os.Args {
		if arg != "--gitrepo-bounded-output" || i+1 >= len(os.Args) {
			continue
		}
		var output *os.File
		switch os.Args[i+1] {
		case "stdout":
			output = os.Stdout
		case "stderr":
			output = os.Stderr
		default:
			os.Exit(2)
		}
		chunk := bytes.Repeat([]byte("x"), 1<<20)
		_, _ = output.Write(chunk)
		os.Exit(0)
	}
}

func TestBoundedOutputEnforcesExecPipeLimit(t *testing.T) {
	for _, stream := range []string{"stdout", "stderr"} {
		t.Run(stream, func(t *testing.T) {
			const limit = 1024
			output := &boundedOutput{limit: limit}
			cmd := exec.Command(os.Args[0], "-test.run=^TestBoundedOutputProcess$", "--", "--gitrepo-bounded-output", stream)
			cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
			if stream == "stdout" {
				cmd.Stdout = output
			} else {
				cmd.Stderr = output
			}
			if err := boundedCommandError(cmd.Run(), output); !errors.Is(err, ErrLimit) || !output.limitExceeded {
				t.Fatalf("over-limit %s error = %v, want ErrLimit", stream, err)
			}
			if got := int64(len(output.Bytes())); got > limit {
				t.Fatalf("retained %s bytes = %d, limit = %d", stream, got, limit)
			}
		})
	}
}

func TestBoundedOutputWriteKeepsLimitAndAccessors(t *testing.T) {
	output := &boundedOutput{limit: 3}
	if n, err := output.Write([]byte("abc")); n != 3 || err != nil {
		t.Fatalf("write at limit = (%d, %v)", n, err)
	}
	if n, err := output.Write([]byte("d")); n != 0 || !errors.Is(err, ErrLimit) {
		t.Fatalf("write beyond limit = (%d, %v), want ErrLimit", n, err)
	}
	if got := string(output.Bytes()); got != "abc" || output.String() != got {
		t.Fatalf("retained bytes=%q string=%q, want abc", got, output.String())
	}
}

func forbiddenProgram(t *testing.T, marker string) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%q -test.run=TestForbiddenGitProgramHelper -- --gitrepo-forbidden-program %q", filepath.ToSlash(exe), filepath.ToSlash(marker))
}

func TestCleanCaptureNeverRunsConfiguredFiltersOrFsmonitor(t *testing.T) {
	repo, selection, _ := libraryFixture(t)
	repo.Write(".gitattributes", "real/library.go filter=forbidden\n")
	repo.Commit("Attributes before external filter configuration")
	marker := filepath.Join(testkit.TempDir(t), "executed")
	command := forbiddenProgram(t, marker)
	repo.Git("config", "filter.forbidden.clean", command)
	repo.Git("config", "core.fsmonitor", command)
	r := adapter(t)
	if _, err := r.Capture(context.Background(), project.CaptureRequest{Selection: selection}); err != nil {
		t.Fatal(err)
	}
	repo.Write("real/library.go", "dirty raw content requiring a real comparison")
	result, err := r.Capture(context.Background(), project.CaptureRequest{Selection: selection})
	if !errors.Is(err, ErrDirty) || result.Guard != "" {
		t.Fatalf("dirty file escaped raw comparison: %v", err)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("source acquisition executed configured user program")
	}
}

func TestAmbientGitOverridesAndReplaceRefsCannotRedirectAcquisition(t *testing.T) {
	repo, selection, head := libraryFixture(t)
	foreign := testkit.NewRepo(t)
	foreign.Write("foreign", "wrong repository")
	foreign.Commit("Foreign")
	for key, value := range map[string]string{"GIT_DIR": filepath.Join(foreign.Dir, ".git"), "GIT_WORK_TREE": foreign.Dir, "GIT_INDEX_FILE": filepath.Join(foreign.Dir, "foreign-index"), "GIT_OBJECT_DIRECTORY": filepath.Join(foreign.Dir, ".git", "objects"), "GIT_CONFIG_COUNT": "1", "GIT_CONFIG_KEY_0": "core.bare", "GIT_CONFIG_VALUE_0": "true"} {
		t.Setenv(key, value)
	}
	r := adapter(t)
	first, err := r.Capture(context.Background(), project.CaptureRequest{Selection: selection})
	if err != nil || first.Head != head {
		t.Fatalf("ambient Git redirect: %v head=%s", err, first.Head)
	}
	// A replacement commit must not replace the explicitly selected old tree.
	repo.Write("inputs/local/model.yaml", "invalid replacement bytes")
	replacement := repo.Commit("Replacement tree")
	repo.Git("replace", head, replacement)
	fixed, err := r.CapturePreview(context.Background(), project.PreviewRequest{Selection: selection, Revision: head})
	if err != nil {
		t.Fatal(err)
	}
	manifest, _ := fixed.Raw.Local.Files.Lookup(fixturePath(t, "model.yaml"))
	if !strings.Contains(string(manifest.Bytes()), "id: small-library") {
		t.Fatal("replace refs changed fixed source")
	}
}

func TestCorruptBlobOIDAndMissingPromisorObjectRefuseWithoutEffects(t *testing.T) {
	for _, kind := range []string{"hash-mismatch", "missing-promisor"} {
		t.Run(kind, func(t *testing.T) {
			repo, selection, head := libraryFixture(t)
			oid := repo.Git("rev-parse", head+":real/library.go")
			object := filepath.Join(repo.Dir, ".git", "objects", oid[:2], oid[2:])
			if err := os.Chmod(object, 0o644); err != nil {
				t.Fatal(err)
			}
			marker := filepath.Join(testkit.TempDir(t), "remote-executed")
			if kind == "hash-mismatch" {
				data := []byte("\xef\xbb\xbfpackage library\r\n")
				data[0] ^= 1
				var compressed bytes.Buffer
				writer := zlib.NewWriter(&compressed)
				_, _ = fmt.Fprintf(writer, "blob %d\x00", len(data))
				_, _ = writer.Write(data)
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(object, compressed.Bytes(), 0o644); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.Remove(object); err != nil {
					t.Fatal(err)
				}
				repo.Git("config", "core.repositoryformatversion", "1")
				repo.Git("config", "extensions.partialclone", "origin")
				repo.Git("config", "remote.origin.promisor", "true")
				repo.Git("config", "remote.origin.url", "ext::"+forbiddenProgram(t, marker))
				repo.Git("config", "protocol.ext.allow", "always")
			}
			before := directoryDigest(t, filepath.Join(repo.Dir, ".git"))
			result, err := adapter(t).CapturePreview(context.Background(), project.PreviewRequest{Selection: selection, Revision: head})
			if err == nil || result.Head != head || result.Revision != head {
				t.Fatalf("unsafe object claimed successful source: %v", err)
			}
			if kind == "hash-mismatch" && !errors.Is(err, ErrChanged) {
				t.Fatalf("blob was not content verified: %v", err)
			}
			if directoryDigest(t, filepath.Join(repo.Dir, ".git")) != before {
				t.Fatal("read-only acquisition changed Git administration")
			}
			if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("source acquisition started promisor remote")
			}
		})
	}
}

func directoryDigest(t *testing.T, dir string) [32]byte {
	t.Helper()
	hash := sha256.New()
	err := fs.WalkDir(os.DirFS(dir), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(p)))
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(hash, "%d:%s:%d:", len(p), p, len(data))
		_, _ = hash.Write(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var result [32]byte
	copy(result[:], hash.Sum(nil))
	return result
}

func TestBoundaryCollisionsAreRejectedAndRechecked(t *testing.T) {
	for _, when := range []string{"present", "after-roots", "before-confirm"} {
		t.Run(when, func(t *testing.T) {
			repo, selection, _ := libraryFixture(t)
			named := filepath.Join(repo.Dir, "inputs", "straße")
			if err := os.Rename(selection.Local.Boundary, named); err != nil {
				t.Fatal(err)
			}
			selection.Local.Boundary = named
			alias := filepath.Join(repo.Dir, "inputs", "strasse")
			makeAlias := func() {
				if err := os.Mkdir(alias, 0o755); err != nil {
					t.Skipf("filesystem cannot represent D145 siblings: %v", err)
				}
			}
			r := adapter(t)
			var reads []string
			made := false
			if when == "present" {
				makeAlias()
				made = true
			}
			r.observe = func(event, p string) {
				if event == "before-file-open" && !strings.HasSuffix(p, "model.yaml") {
					reads = append(reads, p)
				}
				if !made && ((when == "after-roots" && event == "roots-validated") || (when == "before-confirm" && event == "before-confirm")) {
					made = true
					makeAlias()
				}
			}
			_, err := r.CapturePreview(context.Background(), project.PreviewRequest{Selection: selection})
			if err == nil || !made {
				t.Fatal("boundary collision accepted")
			}
			if when != "before-confirm" && len(reads) != 0 {
				t.Fatalf("invalid boundary allowed definition reads: %v", reads)
			}
			if when == "before-confirm" && !errors.Is(err, ErrChanged) {
				t.Fatalf("alias race was not retained: %v", err)
			}
		})
	}
}
