package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/codexappserver"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestMain(m *testing.M) { testkit.Main(m) }

type bootstrapImageReader struct{ calls []string }

const bootstrapImageID = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
const bootstrapSchema = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func (r *bootstrapImageReader) ImageID(_ context.Context, id string) (string, error) {
	r.calls = append(r.calls, "inspect:"+id)
	return bootstrapImageID, nil
}
func (r *bootstrapImageReader) ReadImageFile(_ context.Context, id, path string) ([]byte, error) {
	r.calls = append(r.calls, "read:"+id+":"+path)
	return []byte("codex=codex-cli 0.162.0\ncodexSchema=" + bootstrapSchema + "\nagentd=development\n"), nil
}

func TestBootstrapImageFactoryUsesOnlySelectedSourceBoundConfiguration(t *testing.T) {
	repo := productFixture(t)
	execution, err := os.ReadFile(filepath.Join(repo.Dir, ".markitect", "execution.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ReplaceAll(string(execution), "\r\n", "\n")
	start := strings.Index(text, "\nchecks:\n")
	if start < 0 {
		t.Fatal("fixture execution checks missing")
	}
	repo.Write(".markitect/execution.yaml", text[:start]+"\nchecks: []\n")
	engine := filepath.Join(t.TempDir(), "explicit-engine-not-executed")
	runtimeProfile := filepath.Join(t.TempDir(), "runtime.yaml")
	profile := fmt.Sprintf("apiVersion: markitect.runtime/v1\nagentProfiles:\n  markitect-agent:\n    codexAppServer:\n      profileVersion: 2\n      engineExecutable: %q\n      imageID: %s\n      agentdVersion: development\n      recordedCodexSchemaDigest: %s\n      instructions: []\ncheckProfiles: {}\n", engine, bootstrapImageID, bootstrapSchema)
	if err := os.WriteFile(runtimeProfile, []byte(profile), 0600); err != nil {
		t.Fatal(err)
	}
	reader := &bootstrapImageReader{}
	var received []codexappserver.EngineConfig
	factory := func(config codexappserver.EngineConfig) (codexappserver.ImageReader, error) {
		received = append(received, config)
		return reader, nil
	}
	var out, errout bytes.Buffer
	code := run(context.Background(), []string{"check", "--repo", repo.Dir}, &out, &errout, factory)
	if code != 0 || len(received) != 0 {
		t.Fatalf("structural preparation used engine: exit=%d factories=%v stderr=%s", code, received, errout.String())
	}
	out.Reset()
	errout.Reset()
	code = run(context.Background(), []string{"check", "--repo", repo.Dir, "--metadata", "--runtime-profile", runtimeProfile}, &out, &errout, factory)
	if code != 0 {
		t.Fatalf("metadata exit=%d stderr=%s output=%s", code, errout.String(), out.String())
	}
	// A complete source reacquisition may prepare the same selected profile
	// again. Every actual attempt must use only the supplied configuration.
	if len(received) == 0 || len(reader.calls) != 3*len(received) {
		t.Fatalf("actual selected factory/config or bounded read lost: %v %v", received, reader.calls)
	}
	for i, config := range received {
		if config.Executable != engine || reader.calls[3*i] != "inspect:"+bootstrapImageID || reader.calls[3*i+1] != "read:"+bootstrapImageID+":/opt/markitect/runtime.env" || reader.calls[3*i+2] != "inspect:"+bootstrapImageID {
			t.Fatalf("unbound attempt: %v %v", config, reader.calls)
		}
	}
	if !strings.Contains(out.String(), `"AgentImage"`) || !strings.Contains(out.String(), `"ImageID": "`+bootstrapImageID+`"`) || strings.Contains(out.String(), "offline-metadata") {
		t.Fatal("image evidence missing from actual metadata report")
	}
	// Production's concrete factory rejects this missing explicitly selected
	// engine path. There is no ambient executable or old metadata fallback.
	out.Reset()
	errout.Reset()
	code = Run(context.Background(), []string{"check", "--repo", repo.Dir, "--metadata", "--runtime-profile", runtimeProfile}, &out, &errout)
	if code != 3 || !strings.Contains(out.String(), "codex_image.acquisition") {
		t.Fatalf("production factory fallback: exit=%d stderr=%s", code, errout.String())
	}
}

func productFixture(t *testing.T) *testkit.Repo {
	t.Helper()
	repo := testkit.NewRepo(t)
	repo.Git("checkout", "-b", "codex/read-only-consumer-fixture")
	err := filepath.WalkDir("../../.markitect/model", func(name string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() {
			return nil
		}
		relative, err := filepath.Rel("../../.markitect/model", name)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		repo.Write(".markitect/model/"+filepath.ToSlash(relative), string(data))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"model.yaml", "model-lock.yaml", "execution.yaml"} {
		data, err := os.ReadFile("../../.markitect/" + name)
		if err != nil {
			t.Fatal(err)
		}
		repo.Write(".markitect/"+name, string(data))
	}
	repo.Write("docs/markitect/project.md", "reviewed prior generated target\n")
	repo.Write(".markitect/registry/documentation/registry.yaml", "apiVersion: markitect.registry/v1\nartifacts:\n  - path: docs/markitect/project.md\n    form: doc\n    realizes: [documentation/current-guidance]\n    why: The exact generated view target.\n")
	repo.Commit("canonical v1 consumer fixture")
	return repo
}

func invoke(t *testing.T, repo string, want int, args ...string) string {
	t.Helper()
	var out, errout bytes.Buffer
	args = append(args, "--repo", repo)
	code := Run(context.Background(), args, &out, &errout)
	if code != want {
		t.Fatalf("Run(%v)=%d want=%d\nstdout=%s\nstderr=%s", args, code, want, out.String(), errout.String())
	}
	return out.String()
}

type docsReport struct {
	View struct {
		Status, Path, ContentDigest, PreviewDigest, CanonicalModelDigest, Markdown string
		Written                                                                    bool
		CompletedPaths                                                             []string
	}
}

func docsPreview(t *testing.T, repo string, extra ...string) docsReport {
	t.Helper()
	var report docsReport
	out := invoke(t, repo, 0, append([]string{"docs"}, extra...)...)
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatal(err)
	}
	if report.View.Status != "passed" || report.View.PreviewDigest == "" {
		t.Fatal(out)
	}
	return report
}

func TestActualStructuralPreparationAndFixedContext(t *testing.T) {
	repo := productFixture(t)
	out := invoke(t, repo.Dir, 0, "check")
	var r struct {
		Status, RequestedStage string
		Stages                 struct{ Metadata, Settings struct{ Requested bool } }
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatal(err)
	}
	if r.Status != "passed" || r.RequestedStage != "structural" || r.Stages.Metadata.Requested || r.Stages.Settings.Requested {
		t.Fatal(out)
	}
	if !strings.Contains(out, `"owner": "documentation"`) || !strings.Contains(out, `"path": "docs/markitect/project.md"`) || !strings.Contains(out, `"realizes"`) {
		t.Fatal("actual artifact owner/basis was lost behind private producer fields: " + out)
	}
	out = invoke(t, repo.Dir, 0, "context", "product/model", "--revision", "HEAD")
	if !strings.Contains(out, "language-v1") || !strings.Contains(out, `"reasons"`) || !strings.Contains(out, repo.Git("rev-parse", "HEAD")) {
		t.Fatal(out)
	}
	invoke(t, repo.Dir, 1, "context", "absent-package")
	invoke(t, repo.Dir, 3, "check", "--metadata")
	repo.Write("unaccounted.txt", "a real census entry\n")
	out = invoke(t, repo.Dir, 1, "check")
	if !strings.Contains(out, "unaccounted.txt") {
		t.Fatal(out)
	}
	invoke(t, repo.Dir, 0, "check", "--revision", "HEAD")
}

// staleRegistryLink leaves the product fixture an artifact of a Definition the
// owner deleted from the model: its registry link is stale.
func staleRegistryLink(repo *testkit.Repo) {
	repo.Write("docs/markitect/removed.md", "realizes a deleted Definition\n")
	repo.Write(".markitect/registry/documentation/registry.yaml", "apiVersion: markitect.registry/v1\nartifacts:\n  - path: docs/markitect/project.md\n    form: doc\n    realizes: [documentation/current-guidance]\n    why: The exact generated view target.\n  - path: docs/markitect/removed.md\n    form: doc\n    realizes: [documentation/removed]\n    why: Realized a Definition the owner deleted.\n")
	repo.Commit("Delete a realized Definition")
}

// check passes a stale registry link, because removing it is realization for
// Apply, and lists it apart from failures, so a model error stays distinct.
func TestCheckReportsAStaleRegistryLinkAsRealizationStillToDo(t *testing.T) {
	repo := productFixture(t)
	staleRegistryLink(repo)
	out := invoke(t, repo.Dir, 0, "check")
	var r struct {
		Status string
		Stages struct {
			Structural struct {
				Registry struct{ Status string }
				Files    struct {
					Status     string
					Accounting struct {
						Findings []any
						Stale    []struct{ Code, Path, Owner, Message string }
					}
				}
			}
		}
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatal(err)
	}
	files := r.Stages.Structural.Files
	if r.Status != "passed" || r.Stages.Structural.Registry.Status != "passed" || files.Status != "passed" || len(files.Accounting.Findings) != 0 {
		t.Fatal("a stale registry link failed the input", out)
	}
	stale := files.Accounting.Stale
	if len(stale) != 1 || stale[0].Code != "registry.stale_link" || stale[0].Path != "docs/markitect/removed.md" || stale[0].Owner != "documentation" || !strings.Contains(stale[0].Message, "documentation/removed") {
		t.Fatal("check does not list the stale link as realization still to do", stale)
	}
}

// A shard of a Package the model no longer has passes check too: its
// registrations and the shard are listed as realization still to do.
func TestCheckReportsAnOrphanedShardAsRealizationStillToDo(t *testing.T) {
	repo := productFixture(t)
	repo.Write("docs/markitect/removed.md", "realized by a removed Package\n")
	repo.Write(".markitect/registry/removed/registry.yaml", "apiVersion: markitect.registry/v1\nartifacts:\n  - path: docs/markitect/removed.md\n    form: doc\n    realizes: [removed/guide]\n    why: Realized before the Package was removed.\n")
	repo.Commit("Remove a Package that has a registry shard")
	out := invoke(t, repo.Dir, 0, "check")
	var r struct {
		Status string
		Stages struct {
			Structural struct {
				Registry struct {
					Status   string
					Orphaned []string
				}
				Files struct {
					Accounting struct {
						Stale []struct{ Code, Path, Owner string }
					}
				}
			}
		}
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatal(err)
	}
	registry, stale := r.Stages.Structural.Registry, r.Stages.Structural.Files.Accounting.Stale
	if r.Status != "passed" || registry.Status != "passed" || len(registry.Orphaned) != 1 || registry.Orphaned[0] != ".markitect/registry/removed/registry.yaml" {
		t.Fatal("an orphaned shard failed the input or was not listed", out)
	}
	if len(stale) != 1 || stale[0].Code != "registry.orphaned" || stale[0].Path != "docs/markitect/removed.md" || stale[0].Owner != "." {
		t.Fatal("the orphaned registration is not work for its area owner", stale)
	}
}

func TestDocumentationBindsTargetAndModelAndReadsBackActualWrite(t *testing.T) {
	repo := productFixture(t)
	first := docsPreview(t, repo.Dir)
	if first.View.ContentDigest == first.View.PreviewDigest || first.View.ContentDigest == first.View.CanonicalModelDigest {
		t.Fatal("different identities conflated")
	}
	repo.Write(first.View.Path, "competing target edit\n")
	out := invoke(t, repo.Dir, 1, "docs", "--write", "--expect", first.View.PreviewDigest)
	if !strings.Contains(out, "previewDigest changed") {
		t.Fatal(out)
	}
	next := docsPreview(t, repo.Dir)
	if next.View.ContentDigest != first.View.ContentDigest || next.View.CanonicalModelDigest != first.View.CanonicalModelDigest || next.View.PreviewDigest == first.View.PreviewDigest {
		t.Fatal("target edit did not change only the reviewed write-plan binding")
	}
	const rule = ".markitect/model/product/model/language-v1.yaml"
	data, err := os.ReadFile(filepath.Join(repo.Dir, filepath.FromSlash(rule)))
	if err != nil {
		t.Fatal(err)
	}
	repo.Write(rule, strings.Replace(string(data), "Check the complete", "Check the complete and precise", 1))
	invoke(t, repo.Dir, 1, "docs", "--write", "--expect", next.View.PreviewDigest)
	current := docsPreview(t, repo.Dir)
	if current.View.CanonicalModelDigest == first.View.CanonicalModelDigest || current.View.ContentDigest == first.View.ContentDigest {
		t.Fatal("model change lost from view identity")
	}
	out = invoke(t, repo.Dir, 0, "docs", "--write", "--expect", current.View.PreviewDigest)
	var written docsReport
	if err := json.Unmarshal([]byte(out), &written); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(filepath.Join(repo.Dir, filepath.FromSlash(current.View.Path)))
	if err != nil {
		t.Fatal(err)
	}
	if !written.View.Written || len(written.View.CompletedPaths) != 1 || string(actual) != current.View.Markdown || fmt.Sprintf("sha256:%x", sha256.Sum256(actual)) != current.View.ContentDigest {
		t.Fatal("actual effect/readback differs from preview or report")
	}
	fixed := docsPreview(t, repo.Dir, "--revision", "HEAD")
	invoke(t, repo.Dir, 1, "docs", "--revision", "HEAD", "--write", "--expect", fixed.View.PreviewDigest)
	unchanged, err := os.ReadFile(filepath.Join(repo.Dir, filepath.FromSlash(current.View.Path)))
	if err != nil || !bytes.Equal(actual, unchanged) {
		t.Fatal("historical write affected current target")
	}
}

func TestDocumentationCannotOverwriteAuthoredInstructionEvenWhenMissing(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(fmt.Sprint(missing), func(t *testing.T) {
			repo := productFixture(t)
			path := filepath.Join(repo.Dir, ".markitect", "execution.yaml")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			repo.Write(".markitect/execution.yaml", strings.Replace(string(data), "checks:", "instructions:\n  .: [docs/markitect/project.md]\nchecks:", 1))
			if missing {
				if err := os.Remove(filepath.Join(repo.Dir, "docs", "markitect", "project.md")); err != nil {
					t.Fatal(err)
				}
			}
			out := invoke(t, repo.Dir, 1, "docs")
			if !strings.Contains(out, "authored instruction input") {
				t.Fatal(out)
			}
			if missing {
				if _, err := os.Stat(filepath.Join(repo.Dir, "docs", "markitect", "project.md")); !os.IsNotExist(err) {
					t.Fatal("missing protected input was created")
				}
			}
		})
	}
}

func TestMalformedManifestIsKnownFailureButMissingBytesAreIncomplete(t *testing.T) {
	repo := productFixture(t)
	repo.Write(".markitect/model.yaml", "formatVersion: bad\nid: fixture\ndependencies: {}\n")
	invoke(t, repo.Dir, 1, "check")
	invoke(t, repo.Dir, 1, "docs")
	if err := os.Remove(filepath.Join(repo.Dir, ".markitect", "model.yaml")); err != nil {
		t.Fatal(err)
	}
	invoke(t, repo.Dir, 3, "docs")
}

func moveFixtureCore(t *testing.T, repo *testkit.Repo, destination, rootDeclaration string) string {
	t.Helper()
	old := filepath.Join(repo.Dir, ".markitect", "model", "markitect", "core")
	entries, err := os.ReadDir(old)
	if err != nil {
		t.Fatal(err)
	}
	modelRoot := destination
	if rootDeclaration != "." {
		modelRoot = filepath.Join(destination, rootDeclaration)
	}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(old, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(modelRoot, "markitect", "core", entry.Name())
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.RemoveAll(filepath.Join(repo.Dir, ".markitect", "model", "markitect")); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile("../../testdata/core/model.yaml")
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(destination, "model.yaml")
	if err := os.WriteFile(manifestPath, []byte(strings.Replace(string(manifest), "root: model", "root: "+rootDeclaration, 1)), 0644); err != nil {
		t.Fatal(err)
	}
	local, err := os.ReadFile(filepath.Join(repo.Dir, ".markitect", "model.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	repo.Write(".markitect/model.yaml", strings.Replace(string(local), "dependencies: {}", "dependencies: {markitect-core: '1.1.0'}", 1))
	lock, err := os.ReadFile("../../testdata/library/m1/model-lock.yaml")
	if err != nil {
		t.Fatal(err)
	}
	coreLock, _, found := strings.Cut(string(lock), "\n  software-basics:")
	if !found {
		t.Fatal("normative core lock entry missing")
	}
	repo.Write(".markitect/model-lock.yaml", coreLock+"\n")
	return manifestPath
}

func TestExternalModelAndRuntimeSourcesRemainPreviewOnly(t *testing.T) {
	for _, kind := range []string{"dependency", "runtime"} {
		t.Run(kind, func(t *testing.T) {
			repo := productFixture(t)
			var options []string
			if kind == "dependency" {
				manifest := moveFixtureCore(t, repo, t.TempDir(), "model")
				options = []string{"--dependency", manifest}
			} else {
				runtimeProfile := filepath.Join(t.TempDir(), "runtime.yaml")
				if err := os.WriteFile(runtimeProfile, []byte("apiVersion: markitect.runtime/v1\nagentProfiles: {}\ncheckProfiles: {}\n"), 0644); err != nil {
					t.Fatal(err)
				}
				options = []string{"--runtime-profile", runtimeProfile}
			}
			preview := docsPreview(t, repo.Dir, options...)
			args := append([]string{"docs", "--write", "--expect", preview.View.PreviewDigest}, options...)
			out := invoke(t, repo.Dir, 3, args...)
			if !strings.Contains(out, "requires repository-bound sources") {
				t.Fatal(out)
			}
			actual, err := os.ReadFile(filepath.Join(repo.Dir, "docs", "markitect", "project.md"))
			if err != nil || string(actual) != "reviewed prior generated target\n" {
				t.Fatalf("target changed: %q %v", actual, err)
			}
		})
	}
}

func TestRepositoryBoundDependencyKeepsDocumentationWritable(t *testing.T) {
	repo := productFixture(t)
	manifest := moveFixtureCore(t, repo, filepath.Join(repo.Dir, "inputs", "core"), "model")
	preview := docsPreview(t, repo.Dir, "--dependency", manifest)
	out := invoke(t, repo.Dir, 0, "docs", "--dependency", manifest, "--write", "--expect", preview.View.PreviewDigest)
	var written docsReport
	if err := json.Unmarshal([]byte(out), &written); err != nil {
		t.Fatal(err)
	}
	if !written.View.Written || len(written.View.CompletedPaths) != 1 {
		t.Fatal(out)
	}
	actual, err := os.ReadFile(filepath.Join(repo.Dir, "docs", "markitect", "project.md"))
	if err != nil || string(actual) != preview.View.Markdown {
		t.Fatalf("dependency-bound write/readback mismatch: %v", err)
	}
}
