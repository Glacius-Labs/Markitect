package publish

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadPublishedDistributionRequiresImmutableAttestedRelease(t *testing.T) {
	dir, assets := makeAssets(t, testTag, testRunID, 2, testCommit)
	_ = dir
	fake := newFakeGH(assets)
	fake.releaseExists = true
	loaded, err := LoadPublishedDistribution(context.Background(), fake, testTag)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Tag != testTag || loaded.Version != "1.2.3" || loaded.Commit != testCommit || loaded.Digests[assetNames(testTag)[2]] != digest(assets[assetNames(testTag)[2]]) {
		t.Fatalf("loaded distribution = %#v", loaded)
	}
	assertCallOrder(t, fake.calls, "releases/tags/", "release verify", "release download", "git/ref/tags/", "actions/runs/7654321/attempts/2", "verify-asset")
	for _, call := range fake.calls {
		for _, arg := range call {
			if arg == "repos/Glacius-Labs/Markitect/actions/runs/7654321" {
				t.Fatal("loader used the latest workflow run endpoint instead of the provenance attempt")
			}
		}
	}
	for _, tc := range []struct {
		name   string
		mutate func(*fakeGH)
	}{
		{"draft", func(f *fakeGH) { f.releaseDraft = true }},
		{"mutable", func(f *fakeGH) { f.releaseImmutable = false }},
		{"unpublished", func(f *fakeGH) { f.releasePublishedAt = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := newFakeGH(assets)
			bad.releaseExists = true
			tc.mutate(bad)
			if _, err := LoadPublishedDistribution(context.Background(), bad, testTag); err == nil {
				t.Fatal("invalid release was accepted")
			}
		})
	}
	wrongAttempt := newFakeGH(assets)
	wrongAttempt.releaseExists = true
	wrongAttempt.attemptRun.RunAttempt = 1
	if _, err := LoadPublishedDistribution(context.Background(), wrongAttempt, testTag); err == nil || !strings.Contains(err.Error(), "successful run") {
		t.Fatalf("release with mismatched provenance attempt was accepted: %v", err)
	}
}

func TestSyncDistributionWritesChecksAndExportsDeterministicFiles(t *testing.T) {
	dir, assets := makeAssets(t, testTag, testRunID, 2, testCommit)
	_ = dir
	fake := newFakeGH(assets)
	fake.releaseExists = true
	root := t.TempDir()
	readme := "Intro stays here\n\n" + installStart + "\nold install\n" + installEnd + "\n\n" + tryStart + "\nold try\n" + tryEnd + "\n\nFooter stays here\n"
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(readme), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := SyncDistribution(context.Background(), fake, DistributionOptions{Tag: testTag, RepositoryRoot: root, Write: true})
	if err != nil || result.Status != "written" || len(result.Files) != 4 {
		t.Fatalf("write result = %#v, %v", result, err)
	}
	wantPaths := []string{filepath.Join(root, "README.md"), filepath.Join(root, "integration", "winget", "1.2.3", "GlaciusLabs.Markitect.installer.yaml"), filepath.Join(root, "integration", "winget", "1.2.3", "GlaciusLabs.Markitect.locale.en-US.yaml"), filepath.Join(root, "integration", "winget", "1.2.3", "GlaciusLabs.Markitect.yaml")}
	for i := range wantPaths {
		if result.Files[i] != wantPaths[i] {
			t.Fatalf("result paths = %#v, want sorted %#v", result.Files, wantPaths)
		}
	}
	gotReadme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Intro stays here", "Footer stays here", "v1.2.3", digest(assets[assetNames(testTag)[2]]), digest(assets[assetNames(testTag)[1]]), "markitect-sample-v1.2.3"} {
		if !strings.Contains(string(gotReadme), want) {
			t.Errorf("generated README omitted %q", want)
		}
	}
	result, err = SyncDistribution(context.Background(), fake, DistributionOptions{Tag: testTag, RepositoryRoot: root})
	if err != nil || result.Status != "checked" {
		t.Fatalf("check after write = %#v, %v", result, err)
	}
	files, err := renderDistribution(&PublishedDistribution{Tag: testTag, Version: "1.2.3", Digests: map[string]string{assetNames(testTag)[2]: digest(assets[assetNames(testTag)[2]]), assetNames(testTag)[1]: digest(assets[assetNames(testTag)[1]])}})
	if err != nil {
		t.Fatal(err)
	}
	installer := string(files["integration/winget/1.2.3/GlaciusLabs.Markitect.installer.yaml"])
	if !strings.Contains(installer, "InstallerUrl: https://github.com/Glacius-Labs/Markitect/releases/download/v1.2.3/markitect-v1.2.3-windows-amd64.exe") || !strings.Contains(installer, "InstallerSha256: "+strings.ToUpper(digest(assets[assetNames(testTag)[2]]))) {
		t.Fatalf("WinGet installer did not bind the verified Windows release asset:\n%s", installer)
	}
	exportDir := filepath.Join(t.TempDir(), "winget-pkgs")
	result, err = SyncDistribution(context.Background(), fake, DistributionOptions{Tag: testTag, ExportWinget: exportDir})
	if err != nil || result.Status != "exported" || len(result.Files) != 3 {
		t.Fatalf("export result = %#v, %v", result, err)
	}
	wantExportPaths := []string{
		filepath.Join(exportDir, "manifests", "g", "GlaciusLabs", "Markitect", "1.2.3", "GlaciusLabs.Markitect.installer.yaml"),
		filepath.Join(exportDir, "manifests", "g", "GlaciusLabs", "Markitect", "1.2.3", "GlaciusLabs.Markitect.locale.en-US.yaml"),
		filepath.Join(exportDir, "manifests", "g", "GlaciusLabs", "Markitect", "1.2.3", "GlaciusLabs.Markitect.yaml"),
	}
	for _, target := range result.Files {
		data, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		relative := filepath.ToSlash(filepath.Join("integration", "winget", "1.2.3", filepath.Base(target)))
		if len(data) == 0 || string(data) != string(files[relative]) {
			t.Fatalf("export %s was empty or differs from canonical manifest", target)
		}
	}
	for i := range wantExportPaths {
		if result.Files[i] != wantExportPaths[i] {
			t.Fatalf("export paths = %#v, want sorted %#v", result.Files, wantExportPaths)
		}
	}
}

func TestDistributionWritePreflightsEveryTargetBeforeMutation(t *testing.T) {
	_, assets := makeAssets(t, testTag, testRunID, 2, testCommit)
	fake := newFakeGH(assets)
	fake.releaseExists = true
	root := t.TempDir()
	readme := []byte("intro\n" + installStart + "\nold install\n" + installEnd + "\n" + tryStart + "\nold try\n" + tryEnd + "\nfooter\n")
	readmePath := filepath.Join(root, "README.md")
	if err := os.WriteFile(readmePath, readme, 0600); err != nil {
		t.Fatal(err)
	}
	manifestDir := filepath.Join(root, "integration", "winget", "1.2.3")
	if err := os.MkdirAll(manifestDir, 0700); err != nil {
		t.Fatal(err)
	}
	versionFile := filepath.Join(manifestDir, "GlaciusLabs.Markitect.yaml")
	if err := os.WriteFile(versionFile, []byte("preserve existing manifest"), 0600); err != nil {
		t.Fatal(err)
	}
	blockedInstaller := filepath.Join(manifestDir, "GlaciusLabs.Markitect.installer.yaml")
	if err := os.Mkdir(blockedInstaller, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := SyncDistribution(context.Background(), fake, DistributionOptions{Tag: testTag, RepositoryRoot: root, Write: true}); err == nil {
		t.Fatal("invalid manifest target was accepted")
	}
	after, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(readme) {
		t.Fatal("README changed before all output targets passed preflight")
	}
	afterVersion, err := os.ReadFile(versionFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(afterVersion) != "preserve existing manifest" {
		t.Fatal("existing manifest changed before all output targets passed preflight")
	}
}

func TestReplaceMarkedSectionsRejectsMissingAndDuplicateMarkers(t *testing.T) {
	generated := []byte(installStart + "\nnew install\n" + installEnd + "\n" + tryStart + "\nnew try\n" + tryEnd)
	if _, err := replaceMarkedSections([]byte("no markers"), generated); err == nil {
		t.Fatal("missing README markers were accepted")
	}
	duplicate := []byte(installStart + "\na\n" + installEnd + "\n" + installStart + "\nb\n" + installEnd + "\n" + tryStart + "\nc\n" + tryEnd)
	if _, err := replaceMarkedSections(duplicate, generated); err == nil {
		t.Fatal("duplicate README markers were accepted")
	}
}
