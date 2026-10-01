package publish

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testRunID = "7654321"
const testCommit = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const testTag = "v1.2.3"

func TestPublishReadOnlyPreflightAndOrderedImmutablePublish(t *testing.T) {
	assetsDir, assets := makeAssets(t, testTag, testRunID, 2, testCommit)
	fake := newFakeGH(assets)
	result, err := Execute(context.Background(), fake, Options{Tag: testTag, RunID: testRunID, AssetsDir: assetsDir})
	if err != nil || result.Status != "preflight-passed" || fake.published || fake.draftCreated {
		t.Fatalf("read-only preflight result = (%+v, %v), draft=%v", result, err, fake.draftCreated)
	}
	if !containsCall(fake.calls, "run download") {
		t.Fatal("preflight did not download the exact run-scoped artifact")
	}

	fake = newFakeGH(assets)
	result, err = Execute(context.Background(), fake, Options{Tag: testTag, RunID: testRunID, AssetsDir: assetsDir, Publish: true})
	if err != nil || result.Status != "published-verified" || !result.Immutable || !fake.published {
		t.Fatalf("publish result = (%+v, %v), published=%v", result, err, fake.published)
	}
	assertCallOrder(t, fake.calls, "run download", "-X POST", "release upload", "releases/42/assets", "draft=false", "release verify", "verify-asset")
	for _, call := range fake.calls {
		joined := strings.Join(call, " ")
		if strings.HasPrefix(joined, "api ") && !strings.Contains(joined, "--hostname github.com") {
			t.Errorf("GitHub API call did not pin host: %v", call)
		}
		if (strings.Contains(joined, "release upload") || strings.Contains(joined, "verify-asset") || strings.Contains(joined, "run download")) && !strings.Contains(joined, "github.com/Glacius-Labs/Markitect") {
			t.Errorf("remote gh command did not pin host/repository: %v", call)
		}
	}
}

func TestPrereleaseFlagFollowsSemVerTag(t *testing.T) {
	if !isPrereleaseTag("v1.2.3-rc.1") || isPrereleaseTag("v1.2.3+build.7") || isPrereleaseTag("v1.2.3") {
		t.Fatal("prerelease classification does not follow the SemVer prerelease component")
	}
}

func TestPublishRejectsBadRunOrOwnerBeforeDraft(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*fakeGH)
		want   string
	}{
		{"failed-run", func(f *fakeGH) { f.run.Conclusion = "failure" }, "successful completed"},
		{"wrong-workflow", func(f *fakeGH) { f.run.Path = ".github/workflows/ci.yaml@refs/tags/" + testTag }, "successful completed"},
		{"wrong-source-tag", func(f *fakeGH) { f.tagCommit = strings.Repeat("b", 40) }, "remote tag"},
		{"no-owner-admin", func(f *fakeGH) { f.admin = false }, "administrator"},
		{"immutable-disabled", func(f *fakeGH) { f.immutable = false }, "immutable releases"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, assets := makeAssets(t, testTag, testRunID, 2, testCommit)
			fake := newFakeGH(assets)
			tc.mutate(fake)
			_, err := Execute(context.Background(), fake, Options{Tag: testTag, RunID: testRunID, AssetsDir: dir, Publish: true})
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) || fake.draftCreated {
				t.Fatalf("Execute error=%v, draft=%v", err, fake.draftCreated)
			}
		})
	}
}

func TestPublishRejectsInvalidProvenanceAndTamperedRunArtifact(t *testing.T) {
	dir, assets := makeAssets(t, testTag, testRunID, 9, testCommit)
	fake := newFakeGH(assets)
	if _, err := Execute(context.Background(), fake, Options{Tag: testTag, RunID: testRunID, AssetsDir: dir}); err == nil || !strings.Contains(err.Error(), "provenance does not match") {
		t.Fatalf("mismatched provenance error = %v", err)
	}
	if fake.draftCreated {
		t.Fatal("invalid provenance created a draft")
	}

	dir, assets = makeAssets(t, testTag, testRunID, 2, testCommit)
	fake = newFakeGH(assets)
	fake.artifactFiles[assetNames(testTag)[1]] = append([]byte(nil), assets[assetNames(testTag)[1]]...)
	fake.artifactFiles[assetNames(testTag)[1]][0] ^= 1
	fake.artifactFiles[assetNames(testTag)[3]] = mustYAML(t, validProvenance(testTag, testRunID, 2, testCommit, fake.artifactFiles))
	if _, err := Execute(context.Background(), fake, Options{Tag: testTag, RunID: testRunID, AssetsDir: dir, Publish: true}); err == nil || !strings.Contains(err.Error(), "byte-for-byte") {
		t.Fatalf("tampered artifact error = %v", err)
	}
	if fake.draftCreated {
		t.Fatal("tampered artifact created a draft")
	}
}

func TestPublishRejectsWrongSourceAndNonemptyDraft(t *testing.T) {
	dir, assets := makeAssets(t, testTag, testRunID, 2, strings.Repeat("b", 40))
	assets[assetNames(testTag)[3]] = mustYAML(t, validProvenance(testTag, testRunID, 2, testCommit, assets))
	writeAssets(t, dir, assets)
	fake := newFakeGH(assets)
	if _, err := Execute(context.Background(), fake, Options{Tag: testTag, RunID: testRunID, AssetsDir: dir, Publish: true}); err == nil || !strings.Contains(err.Error(), "bundle manifest") || fake.draftCreated {
		t.Fatalf("wrong-source bundle error=%v draft=%v", err, fake.draftCreated)
	}

	dir, assets = makeAssets(t, testTag, testRunID, 2, testCommit)
	fake = newFakeGH(assets)
	fake.draftHasAssets = true
	if _, err := Execute(context.Background(), fake, Options{Tag: testTag, RunID: testRunID, AssetsDir: dir, Publish: true}); err == nil || !strings.Contains(err.Error(), "empty draft") || fake.published || containsCall(fake.calls, "release upload") {
		t.Fatalf("nonempty draft error=%v published=%v calls=%v", err, fake.published, fake.calls)
	}
}

func TestPublishReportsUnknownTransitionOutcomeWithoutClaimingDraftRemains(t *testing.T) {
	dir, assets := makeAssets(t, testTag, testRunID, 2, testCommit)
	fake := newFakeGH(assets)
	fake.patchTransitionUnknown = true
	result, err := Execute(context.Background(), fake, Options{Tag: testTag, RunID: testRunID, AssetsDir: dir, Publish: true})
	if err == nil || result.Status != "publication-unknown" || !strings.Contains(err.Error(), "outcome") || strings.Contains(err.Error(), "remains for manual review") {
		t.Fatalf("transition failure result=(%+v,%v)", result, err)
	}
}

func TestPublishRejectsExistingReleaseAndUploadedDigestMismatch(t *testing.T) {
	dir, assets := makeAssets(t, testTag, testRunID, 2, testCommit)
	fake := newFakeGH(assets)
	fake.releaseExists = true
	if _, err := Execute(context.Background(), fake, Options{Tag: testTag, RunID: testRunID, AssetsDir: dir, Publish: true}); err == nil || !strings.Contains(err.Error(), "already exists") || fake.draftCreated {
		t.Fatalf("existing release error=%v draft=%v", err, fake.draftCreated)
	}

	fake = newFakeGH(assets)
	fake.badUploadedDigest = true
	result, err := Execute(context.Background(), fake, Options{Tag: testTag, RunID: testRunID, AssetsDir: dir, Publish: true})
	if err == nil || !strings.Contains(err.Error(), "uploaded asset") || result.Status != "draft-created" || fake.published {
		t.Fatalf("digest mismatch result=(%+v,%v) published=%v", result, err, fake.published)
	}
}

func TestProvenanceStrictFieldsAndHashes(t *testing.T) {
	if _, err := parseProvenance([]byte("schemaVersion: \"1\"\nschemaVersion: \"1\"\n")); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate key error = %v", err)
	}
	if _, err := parseProvenance([]byte("schemaVersion: \"1\"\nunknown: value\n")); err == nil || !strings.Contains(err.Error(), "field unknown not found") {
		t.Fatalf("unknown field error = %v", err)
	}
	dir, assets := makeAssets(t, testTag, testRunID, 2, testCommit)
	name := assetNames(testTag)[0]
	if err := os.WriteFile(filepath.Join(dir, name), []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readAssets(dir, testTag); err == nil || !strings.Contains(err.Error(), "digest") {
		t.Fatalf("hash mismatch error = %v", err)
	}
	_ = assets
}
