package projectwork

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectcoverage"
)

func TestNativeSkillPathsHaveExactToolOwnership(t *testing.T) {
	for _, provider := range []string{"codex", "claude"} {
		paths := NativeSkillPaths(provider)
		if len(paths) != 1+2*len(NativeSkillNames()) {
			t.Fatalf("incomplete %s native paths", provider)
		}
		for _, path := range paths {
			if !IsToolPath(Config{}, path) {
				t.Fatalf("generated file lacks ownership: %s", path)
			}
		}
	}
	for _, path := range []string{".agents/skills/markitect-model-first/SKILL.md", ".claude/skills/markitect-model-first/SKILL.md"} {
		if IsToolPath(Config{}, path) {
			t.Fatalf("obsolete router path remains registered as an owned output: %s", path)
		}
	}
	for _, path := range []string{".agents/skills/custom/SKILL.md", ".agents/skills/markitect-implement/custom.md", ".claude/skills/markitect-check/private.md"} {
		if IsToolPath(Config{}, path) {
			t.Fatalf("unmanaged skill claimed by Markitect: %s", path)
		}
	}
}

func TestFullCoverageBindsUnknownOutsideLegacyRoots(t *testing.T) {
	root := testGitRoot(t)
	if _, err := Init(root, "Coverage fixture", true); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "notes/unknown.md", "unclassified repository file\n")
	project, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if project.Coverage == nil || project.Coverage.Accounted || project.Coverage.Conforming {
		t.Fatalf("full coverage did not expose unknown outside-root path: %+v", project.Coverage)
	}
	found := false
	for _, entry := range project.Coverage.Entries {
		if entry.Path == "notes/unknown.md" {
			found = true
		}
	}
	if !found {
		t.Fatal("full census omitted untracked path outside InventoryRoots")
	}
}

func TestTransitionalExclusionIsAccountedButBlocksConformance(t *testing.T) {
	root := testGitRoot(t)
	if _, err := Init(root, "Coverage fixture", true); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, filepath.FromSlash(ManifestPath))
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(manifest), "documentPath: "+DefaultDocumentPath+"\n", "documentPath: README.md\n", 1)
	updated = strings.Replace(updated, "transitionalExclusions: []\n", "transitionalExclusions:\n  - path: "+DefaultDocumentPath+"\n    reason: Prior generated view awaits migration\n  - path: legacy/\n    reason: Existing file awaits explicit modeling\n", 1)
	writeFile(t, root, ManifestPath, updated)
	writeFile(t, root, "legacy/unmodeled.txt", "tracked by the migration boundary\n")
	project, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if project.Coverage == nil || !project.Coverage.Accounted || project.Coverage.Conforming {
		t.Fatalf("transitional path must be accounted but nonconforming: %+v", project.Coverage)
	}
	for _, entry := range project.Coverage.Entries {
		if entry.Path == "legacy/unmodeled.txt" {
			if entry.Class != projectcoverage.ClassTransitional || entry.Reason != "Existing file awaits explicit modeling" {
				t.Fatalf("transitional path classification = %+v", entry)
			}
			return
		}
	}
	t.Fatal("coverage omitted transitional path")
}

// An exact ignore entry matches a census path that the snapshot omits. The
// candidate must still see that path, and a candidate delta is classified on
// top of the census rather than instead of it.
func TestCandidateCoverageKeepsExactIgnoreSatisfiedByCensus(t *testing.T) {
	root := testGitRoot(t)
	if _, err := Init(root, "Coverage fixture", true); err != nil {
		t.Fatal(err)
	}
	ignore := "apiVersion: " + projectcoverage.IgnoreAPIVersion + "\nkind: RepositoryIgnore\nentries:\n  - path: README.md\n    reason: Fixture readme outside the model\n"
	writeFile(t, root, projectcoverage.IgnorePath, ignore)
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-m", "init with exact ignore")
	head := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))
	for _, revision := range []string{"", head} {
		base, err := Load(root, revision)
		if err != nil {
			t.Fatal(err)
		}
		if base.Coverage == nil || !base.Coverage.Conforming {
			t.Fatalf("precondition (revision %q): census not conforming: %+v", revision, base.Coverage)
		}
		candidate := classifiedCandidate(t, base, base.Snapshot)
		if candidate.Coverage == nil || candidate.Coverage.Conforming != base.Coverage.Conforming || len(candidate.Coverage.Findings) != 0 {
			t.Errorf("revision %q: unchanged candidate conforming=%v, census conforming=%v; findings=%+v",
				revision, candidate.Coverage != nil && candidate.Coverage.Conforming, base.Coverage.Conforming, candidate.Coverage.Findings)
		}
		added := &snapshot.Snapshot{ID: base.Snapshot.ID, Provisional: base.Snapshot.Provisional,
			Files: map[string][]byte{"notes/unknown.md": []byte("unclassified candidate file\n")}, Modes: map[string]string{"notes/unknown.md": snapshot.RegularMode}}
		for file, data := range base.Snapshot.Files {
			added.Files[file], added.Modes[file] = data, base.Snapshot.Modes[file]
		}
		changed := classifiedCandidate(t, base, added)
		if changed.Coverage.Conforming || len(changed.Coverage.Findings) != 1 || changed.Coverage.Findings[0].Code != "coverage.unclassified" || changed.Coverage.Findings[0].Path != "notes/unknown.md" {
			t.Errorf("revision %q: candidate delta was not classified on top of the census: %+v", revision, changed.Coverage.Findings)
		}
	}
}

// DEC-006: a transitional path is accounted for but not conforming. The
// snapshot omits it, so only the census lets the candidate keep that state.
func TestCandidateCoverageKeepsTransitionalPathsNonconforming(t *testing.T) {
	root := testGitRoot(t)
	if _, err := Init(root, "Coverage fixture", true); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, filepath.FromSlash(ManifestPath))
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(manifest), "transitionalExclusions: []\n", "transitionalExclusions:\n  - path: README.md\n    reason: Fixture readme awaits modeling\n  - path: legacy/\n    reason: Existing file awaits explicit modeling\n", 1)
	if updated == string(manifest) {
		t.Fatal("could not add transitional exclusions")
	}
	writeFile(t, root, ManifestPath, updated)
	writeFile(t, root, "legacy/old.txt", "legacy bytes\n")
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-m", "init with transitional exclusions")
	head := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))
	for _, revision := range []string{"", head} {
		base, err := Load(root, revision)
		if err != nil {
			t.Fatal(err)
		}
		if base.Coverage == nil || !base.Coverage.Accounted || base.Coverage.Conforming {
			t.Fatalf("precondition (revision %q): census must be accounted but nonconforming: %+v", revision, base.Coverage)
		}
		candidate := classifiedCandidate(t, base, base.Snapshot)
		if candidate.Coverage == nil || candidate.Coverage.Accounted != base.Coverage.Accounted || candidate.Coverage.Conforming != base.Coverage.Conforming {
			t.Errorf("revision %q: candidate accounted=%v conforming=%v, census accounted=%v conforming=%v",
				revision, candidate.Coverage != nil && candidate.Coverage.Accounted, candidate.Coverage != nil && candidate.Coverage.Conforming, base.Coverage.Accounted, base.Coverage.Conforming)
		}
	}
}

// A nonconforming verdict always shows its reason: a present transitional
// path is listed as a finding, while a declared but absent one is not.
func TestProjectDocumentListsTransitionalPathsThatBlockConformance(t *testing.T) {
	root := testGitRoot(t)
	if _, err := Init(root, "Coverage fixture", true); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(ManifestPath)))
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(manifest), "transitionalExclusions: []\n", "transitionalExclusions:\n  - path: legacy/\n    reason: Existing file awaits explicit modeling\n  - path: absent/\n    reason: Nothing here yet\n", 1)
	if updated == string(manifest) {
		t.Fatal("could not add transitional exclusions")
	}
	writeFile(t, root, ManifestPath, updated)
	writeFile(t, root, "legacy/old.txt", "legacy bytes\n")
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-m", "init with transitional exclusions")
	head := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))
	for _, revision := range []string{"", head} {
		project, err := Load(root, revision)
		if err != nil {
			t.Fatal(err)
		}
		got := documentText(project)
		if !strings.Contains(got, "- Conforming: no\n") {
			t.Fatalf("revision %q: want a nonconforming verdict:\n%s", revision, got)
		}
		if strings.Contains(got, "No path or definition has a finding.") {
			t.Fatalf("revision %q: nonconforming verdict without a listed reason:\n%s", revision, got)
		}
		if !strings.Contains(got, "- legacy/old.txt\n  - transitional: blocks conformance until the path is classified (Existing file awaits explicit modeling)\n") {
			t.Fatalf("revision %q: present transitional path is not listed:\n%s", revision, got)
		}
		if strings.Contains(got, "- absent/") && strings.Contains(got, "transitional: blocks conformance until the path is classified (Nothing here yet)") {
			t.Fatalf("revision %q: absent transitional selector listed as blocking:\n%s", revision, got)
		}
	}
}

// A path the candidate deletes leaves the census. Renaming or deleting a
// modelled file together with its Artifact path must not leave the old path
// behind as unclassified.
func TestCandidateCoverageDropsRenamedAndDeletedPaths(t *testing.T) {
	root, head := modelledCoverageFixture(t, nil, map[string]string{"src/a.txt": "a\n", "src/b.txt": "b\n"}, "src/a.txt", "src/b.txt")
	for _, revision := range []string{"", head} {
		base, err := Load(root, revision)
		if err != nil {
			t.Fatal(err)
		}
		if base.Coverage == nil || !base.Coverage.Conforming {
			t.Fatalf("precondition (revision %q): census not conforming: %+v", revision, base.Coverage)
		}
		cases := map[string]*snapshot.Snapshot{
			"rename": changedSnapshot(base.Snapshot, map[string]string{"src/c.txt": "b\n", coverageArtifactPath: coverageArtifact("src/a.txt", "src/c.txt")}, "src/b.txt"),
			"delete": changedSnapshot(base.Snapshot, map[string]string{coverageArtifactPath: coverageArtifact("src/a.txt")}, "src/b.txt"),
		}
		for name, s := range cases {
			candidate := classifiedCandidate(t, base, s)
			if !candidate.Coverage.Conforming {
				t.Errorf("revision %q %s: candidate not conforming: %+v", revision, name, candidate.Coverage.Findings)
			}
			for _, entry := range candidate.Coverage.Entries {
				if entry.Path == "src/b.txt" {
					t.Errorf("revision %q %s: deleted path is still classified: %+v", revision, name, entry)
				}
			}
		}
	}
}

// A working-tree check reads the working tree. Deleting or renaming a model
// file there, staged or not, leaves the old path only in HEAD or the index, so
// coverage must not report it unclassified before the change is committed. A
// fixed revision that still holds the unlisted file keeps reporting it.
func TestWorkingTreeCoverageFollowsModelFileMoves(t *testing.T) {
	const notePath, renamedPath = ModelRoot + "/goal.yaml", ModelRoot + "/renamed.yaml"
	manifest := func(mode string, modelFiles ...string) string {
		text := "apiVersion: " + APIVersion + "\nname: Coverage fixture\ncoverageMode: " + mode + "\ndocumentPath: " + DefaultDocumentPath + "\nmodelFiles:\n  - " + initManagerPath + "\n"
		for _, file := range modelFiles {
			text += "  - " + file + "\n"
		}
		return text + "inventoryRoots: []\nexclusions: []\n"
	}
	moves := map[string]func(t *testing.T, root string) []string{
		"delete": func(t *testing.T, root string) []string {
			if err := os.Remove(filepath.Join(root, filepath.FromSlash(notePath))); err != nil {
				t.Fatal(err)
			}
			return nil
		},
		"git rm": func(t *testing.T, root string) []string {
			gitTest(t, root, "rm", "-q", notePath)
			return nil
		},
		"rename": func(t *testing.T, root string) []string {
			if err := os.Rename(filepath.Join(root, filepath.FromSlash(notePath)), filepath.Join(root, filepath.FromSlash(renamedPath))); err != nil {
				t.Fatal(err)
			}
			return []string{renamedPath}
		},
		"git mv": func(t *testing.T, root string) []string {
			gitTest(t, root, "mv", notePath, renamedPath)
			return []string{renamedPath}
		},
	}
	for _, mode := range []string{"full", "selected"} {
		for name, move := range moves {
			t.Run(mode+"/"+name, func(t *testing.T) {
				root := testGitRoot(t)
				if _, err := Init(root, "Coverage fixture", true); err != nil {
					t.Fatal(err)
				}
				writeFile(t, root, ManifestPath, manifest(mode, notePath))
				writeFile(t, root, projectcoverage.IgnorePath, "apiVersion: "+projectcoverage.IgnoreAPIVersion+"\nkind: RepositoryIgnore\nentries:\n  - path: README.md\n    reason: Fixture readme outside the model\n")
				writeFile(t, root, notePath, statementDefinition("A standalone goal."))
				gitTest(t, root, "add", ".")
				gitTest(t, root, "commit", "-m", "model with a second file")
				moved := move(t, root)
				writeFile(t, root, ManifestPath, manifest(mode, moved...))
				report, err := Coverage(root, "")
				if err != nil {
					t.Fatal(err)
				}
				if !report.Accounted || !report.Conforming || len(report.Findings) != 0 {
					t.Fatalf("working tree after the move: accounted=%v conforming=%v findings=%+v", report.Accounted, report.Conforming, report.Findings)
				}
				if len(moved) != 0 {
					return
				}
				gitTest(t, root, "commit", "-m", "unlist the model file only", "--", ManifestPath)
				fixed, err := Coverage(root, strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD")))
				if err != nil {
					t.Fatal(err)
				}
				if fixed.Accounted || len(fixed.Findings) != 1 || fixed.Findings[0].Code != "coverage.unclassified" || fixed.Findings[0].Path != notePath {
					t.Fatalf("fixed revision that still holds %s: accounted=%v findings=%+v", notePath, fixed.Accounted, fixed.Findings)
				}
			})
		}
	}
}

// An exact ignore entry is used only by a file the checked tree has. Deleting
// its file in the working tree reports the entry unused before the commit; a
// fixed revision that still holds the file keeps the entry used.
func TestWorkingTreeCoverageReportsIgnoreEntryOfDeletedFile(t *testing.T) {
	for _, mode := range []string{"full", "selected"} {
		t.Run(mode, func(t *testing.T) {
			root := testGitRoot(t)
			if _, err := Init(root, "Coverage fixture", true); err != nil {
				t.Fatal(err)
			}
			manifest, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(ManifestPath)))
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, root, ManifestPath, strings.Replace(string(manifest), "coverageMode: full\n", "coverageMode: "+mode+"\n", 1))
			writeFile(t, root, projectcoverage.IgnorePath, "apiVersion: "+projectcoverage.IgnoreAPIVersion+"\nkind: RepositoryIgnore\nentries:\n  - path: README.md\n    reason: Fixture readme outside the model\n")
			gitTest(t, root, "add", ".")
			gitTest(t, root, "commit", "-m", "ignore the readme")
			head := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))
			if err := os.Remove(filepath.Join(root, "README.md")); err != nil {
				t.Fatal(err)
			}
			report, err := Coverage(root, "")
			if err != nil {
				t.Fatal(err)
			}
			if report.Conforming || len(report.Findings) != 1 || report.Findings[0].Code != "coverage.ignore-unused" || report.Findings[0].Path != "README.md" {
				t.Fatalf("working tree without the ignored file: conforming=%v findings=%+v", report.Conforming, report.Findings)
			}
			fixed, err := Coverage(root, head)
			if err != nil {
				t.Fatal(err)
			}
			if !fixed.Conforming || len(fixed.Findings) != 0 {
				t.Fatalf("fixed revision that still holds the ignored file: conforming=%v findings=%+v", fixed.Conforming, fixed.Findings)
			}
		})
	}
}

// A transitional exclusion blocks conformance only while the checked tree
// still has a file under it. Deleting its only file in the working tree lets
// the working tree conform; the fixed revision that still holds it does not.
func TestWorkingTreeCoverageConformsAfterDeletingTransitionalFile(t *testing.T) {
	for _, mode := range []string{"full", "selected"} {
		t.Run(mode, func(t *testing.T) {
			root, _ := modelledCoverageFixture(t, []string{"src/legacy.txt"}, map[string]string{"src/a.txt": "a\n", "src/legacy.txt": "legacy\n"}, "src/a.txt")
			manifest := coverageManifest("src/legacy.txt")
			if mode == "selected" {
				// Selected mode sees the required Artifact only through its inventory roots.
				manifest = strings.Replace(strings.Replace(manifest, "coverageMode: full\n", "coverageMode: selected\n", 1), "inventoryRoots: []\n", "inventoryRoots:\n  - src\n", 1)
			}
			writeFile(t, root, ManifestPath, manifest)
			gitTest(t, root, "commit", "-a", "--allow-empty", "-m", "coverage mode "+mode)
			head := strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))
			gitTest(t, root, "rm", "-q", "src/legacy.txt")
			report, err := Coverage(root, "")
			if err != nil {
				t.Fatal(err)
			}
			if !report.Accounted || !report.Conforming || len(report.Findings) != 0 {
				t.Fatalf("working tree without the transitional file: accounted=%v conforming=%v findings=%+v", report.Accounted, report.Conforming, report.Findings)
			}
			fixed, err := Coverage(root, head)
			if err != nil {
				t.Fatal(err)
			}
			if !fixed.Accounted || fixed.Conforming {
				t.Fatalf("fixed revision that still holds the transitional file: accounted=%v conforming=%v", fixed.Accounted, fixed.Conforming)
			}
		})
	}
}

// Modelling a transitional file in place drops its exclusion and adds it to
// an Artifact without rewriting it. The census never read its bytes, so they
// are read on demand from the base source instead of reported unavailable.
func TestCandidateCoverageReadsTransitionalFileModelledInPlace(t *testing.T) {
	root, head := modelledCoverageFixture(t, []string{"src/legacy.txt"}, map[string]string{"src/a.txt": "a\n", "src/legacy.txt": "legacy\n"}, "src/a.txt")
	for _, revision := range []string{"", head} {
		base, err := Load(root, revision)
		if err != nil {
			t.Fatal(err)
		}
		if base.Coverage == nil || !base.Coverage.Accounted || base.Coverage.Conforming {
			t.Fatalf("precondition (revision %q): census must be accounted but nonconforming: %+v", revision, base.Coverage)
		}
		s := changedSnapshot(base.Snapshot, map[string]string{ManifestPath: coverageManifest(), coverageArtifactPath: coverageArtifact("src/a.txt", "src/legacy.txt")})
		candidate := classifiedCandidate(t, base, s)
		if !candidate.Coverage.Conforming {
			t.Errorf("revision %q: in-place modelled transitional file is not conforming: status=%s findings=%+v", revision, candidate.Report.Status, candidate.Coverage.Findings)
		}
		if string(candidate.Snapshot.Files["src/legacy.txt"]) != "legacy\n" {
			t.Errorf("revision %q: candidate did not bind the modelled file's base bytes", revision)
		}
	}
}

// Ignored and transitional files are in the census but in no snapshot, so
// review, checks and Apply never see them. A candidate that deletes one is
// refused rather than classified as if Apply would remove it.
func TestCandidateCoverageRefusesDeletesOfCensusOnlyFiles(t *testing.T) {
	root, head := modelledCoverageFixture(t, []string{"src/legacy.txt"}, map[string]string{"src/a.txt": "a\n", "src/legacy.txt": "legacy\n"}, "src/a.txt")
	base, err := Load(root, head)
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := FromSnapshot(base.Root, base.Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"src/legacy.txt", "README.md"} {
		if _, err := ClassifyCandidate(base, compiled, file); err == nil || !strings.Contains(err.Error(), "outside the reviewed snapshot") {
			t.Errorf("deleting census-only file %s was not refused: %v", file, err)
		}
	}
	if _, err := ClassifyCandidate(base, compiled, "src/never.txt"); err != nil {
		t.Errorf("deleting an absent path was refused: %v", err)
	}
}

// The view lists no repository inventory, so committing a conforming file
// leaves it byte for byte unchanged. A file with a finding appears in it and
// leaves it again when the file is removed.
func TestProjectDocumentChangesOnlyWithModelOrFindings(t *testing.T) {
	root, head := modelledCoverageFixture(t, nil, map[string]string{"src/a.txt": "a\n"}, "src/")
	render := func(revision string) string {
		t.Helper()
		project, err := Load(root, revision)
		if err != nil {
			t.Fatal(err)
		}
		document, err := Document(project, false)
		if err != nil {
			t.Fatal(err)
		}
		return document
	}
	base := render(head)
	for _, want := range []string{"- Ignored: README.md — Fixture readme outside the model\n", "- Accounted: yes\n- Conforming: yes\n\nNo path or definition has a finding.\n"} {
		if !strings.Contains(base, want) {
			t.Fatalf("conforming view lacks %q:\n%s", want, base)
		}
	}
	for _, absent := range []string{head, "src/a.txt"} {
		if strings.Contains(base, absent) {
			t.Fatalf("view names %q, which is neither model nor finding:\n%s", absent, base)
		}
	}
	writeFile(t, root, "src/b.txt", "b\n")
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-m", "add a modelled file")
	if got := render(strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))); got != base {
		t.Fatalf("committing a conforming file changed the view:\n%s\nwant\n%s", got, base)
	}
	writeFile(t, root, "notes/loose.md", "unmodelled\n")
	if got := render(""); !strings.Contains(got, "- Conforming: no\n") || !strings.Contains(got, "- notes/loose.md\n  - error (coverage.unclassified): ") {
		t.Fatalf("view does not list the unclassified file as a finding:\n%s", got)
	}
	if err := os.Remove(filepath.Join(root, "notes", "loose.md")); err != nil {
		t.Fatal(err)
	}
	if got := render(""); got != base {
		t.Fatalf("removing the file with a finding did not restore the view:\n%s\nwant\n%s", got, base)
	}
}

const coverageArtifactPath = ModelRoot + "/code.yaml"

// modelledCoverageFixture commits a full-coverage project whose required
// Artifact realizes artifactPaths, with the readme ignored and the given
// transitional exclusions and files.
func modelledCoverageFixture(t *testing.T, transitional []string, files map[string]string, artifactPaths ...string) (string, string) {
	t.Helper()
	root := testGitRoot(t)
	if _, err := Init(root, "Coverage fixture", true); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, ManifestPath, coverageManifest(transitional...))
	writeFile(t, root, projectcoverage.IgnorePath, "apiVersion: "+projectcoverage.IgnoreAPIVersion+"\nkind: RepositoryIgnore\nentries:\n  - path: README.md\n    reason: Fixture readme outside the model\n")
	writeFile(t, root, ModelRoot+"/goal.yaml", statementDefinition("Realize the fixture."))
	writeFile(t, root, coverageArtifactPath, coverageArtifact(artifactPaths...))
	for name, content := range files {
		writeFile(t, root, name, content)
	}
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-m", "modelled coverage fixture")
	return root, strings.TrimSpace(gitTest(t, root, "rev-parse", "HEAD"))
}

func coverageManifest(transitional ...string) string {
	text := "apiVersion: " + APIVersion + "\nname: Coverage fixture\ncoverageMode: full\ndocumentPath: " + DefaultDocumentPath +
		"\nmodelFiles:\n  - " + initManagerPath + "\n  - " + ModelRoot + "/goal.yaml\n  - " + coverageArtifactPath + "\ninventoryRoots: []\nexclusions: []\n"
	if len(transitional) > 0 {
		text += "transitionalExclusions:\n"
		for _, path := range transitional {
			text += "  - path: " + path + "\n    reason: Existing file awaits explicit modeling\n"
		}
	}
	return text
}

func coverageArtifact(paths ...string) string {
	return "apiVersion: " + APIVersion + "\nkind: Artifact\nmetadata:\n  name: project-code\n  namespace: \"\"\npurpose: Realizes the project goal.\nspec:\n  role: implementation\n  realizes:\n    - apiVersion: " + APIVersion +
		"\n      kind: Statement\n      namespace: \"\"\n      name: project-goal\n  paths: [" + strings.Join(paths, ", ") + "]\n  required: true\n"
}

func changedSnapshot(base *snapshot.Snapshot, writes map[string]string, deletes ...string) *snapshot.Snapshot {
	result := &snapshot.Snapshot{ID: base.ID, Provisional: base.Provisional, Files: map[string][]byte{}, Modes: map[string]string{}}
	for file, data := range base.Files {
		result.Files[file], result.Modes[file] = append([]byte(nil), data...), base.Modes[file]
	}
	for file, content := range writes {
		result.Files[file], result.Modes[file] = []byte(content), snapshot.RegularMode
	}
	for _, file := range deletes {
		delete(result.Files, file)
		delete(result.Modes, file)
	}
	return result
}

func classifiedCandidate(t *testing.T, base *Project, s *snapshot.Snapshot) *Project {
	t.Helper()
	compiled, err := FromSnapshot(base.Root, s)
	if err != nil {
		t.Fatal(err)
	}
	classified, err := ClassifyCandidate(base, compiled)
	if err != nil {
		t.Fatal(err)
	}
	return classified
}

func TestExplorationOperationalRegistrationIsExact(t *testing.T) {
	root := testGitRoot(t)
	if _, err := Init(root, "Coverage fixture", true); err != nil {
		t.Fatal(err)
	}
	before, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, ".markitect/state/explorations/readiness.json", "operational exploration state\n")
	withExploration, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if before.Digest != withExploration.Digest {
		t.Fatalf("registered operational exploration bytes changed the project binding: %s != %s", before.Digest, withExploration.Digest)
	}
	var explorationRegistered bool
	for _, entry := range withExploration.Coverage.Entries {
		if entry.Path == ".markitect/state/explorations/readiness.json" {
			explorationRegistered = entry.Class == projectcoverage.ClassToolOwned && entry.Operational
		}
	}
	if !explorationRegistered {
		t.Fatal("exploration state was not classified as operational tool-owned state")
	}
	writeFile(t, root, ".markitect/state/unregistered.json", "must remain visible\n")
	withUnknown, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if withUnknown.Coverage.Accounted || withUnknown.Coverage.Conforming {
		t.Fatal("registering exploration state made arbitrary Markitect state conforming")
	}
	for _, entry := range withUnknown.Coverage.Entries {
		if entry.Path == ".markitect/state/unregistered.json" && entry.Class != "" {
			t.Fatalf("unregistered state was silently assigned class %q", entry.Class)
		}
	}
}

func TestOperationalRunBytesDoNotChangeStableProjectBinding(t *testing.T) {
	root := testGitRoot(t)
	if _, err := Init(root, "Coverage fixture", true); err != nil {
		t.Fatal(err)
	}
	before, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, ".markitect/runs/run-1/state.json", "first\n")
	after, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, ".markitect/runs/run-1/state.json", "changed\n")
	changed, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if before.Digest != after.Digest || before.Digest != changed.Digest {
		t.Fatalf("operational run state changed stable binding: %s, %s, %s", before.Digest, after.Digest, changed.Digest)
	}
}

func TestDraftProposalBytesDoNotInvalidateTheirProjectBasis(t *testing.T) {
	root := testGitRoot(t)
	if _, err := Init(root, "Coverage fixture", true); err != nil {
		t.Fatal(err)
	}
	before, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, ".markitect/drafts/proposal.json", "first reviewed proposal\n")
	after, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, ".markitect/drafts/proposal.json", "updated reviewed proposal\n")
	changed, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if before.Digest != after.Digest || before.Digest != changed.Digest {
		t.Fatalf("draft proposal bytes invalidated the stable project basis: %s, %s, %s", before.Digest, after.Digest, changed.Digest)
	}
	var classified bool
	for _, entry := range changed.Coverage.Entries {
		if entry.Path == ".markitect/drafts/proposal.json" {
			classified = entry.Class == projectcoverage.ClassToolOwned && entry.Operational
		}
	}
	if !classified {
		t.Fatal("draft proposal was not retained as explicitly classified operational tool state")
	}
}

func TestIgnoredBytesAreUnboundButMembershipIsBound(t *testing.T) {
	root := testGitRoot(t)
	if _, err := Init(root, "Coverage fixture", true); err != nil {
		t.Fatal(err)
	}
	ignore := "apiVersion: " + projectcoverage.IgnoreAPIVersion + "\nkind: RepositoryIgnore\nentries:\n  - path: scratch/\n    reason: Temporary local outputs\n"
	writeFile(t, root, projectcoverage.IgnorePath, ignore)
	writeFile(t, root, "scratch/output.bin", "first bytes\n")
	first, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "scratch/output.bin", "different bytes\n")
	second, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest != second.Digest {
		t.Fatalf("ignored bytes changed project digest: %s != %s", first.Digest, second.Digest)
	}
	if err := os.Remove(filepath.Join(root, "scratch", "output.bin")); err != nil {
		t.Fatal(err)
	}
	third, err := Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if third.Digest == second.Digest {
		t.Fatal("ignored path membership change did not stale project digest")
	}
}
