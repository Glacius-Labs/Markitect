package publish

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/release"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"go.yaml.in/yaml/v3"
)

func makeAssets(t *testing.T, tag, runID string, attempt int, commit string) (string, namedBytes) {
	t.Helper()
	files := map[string][]byte{
		"go.mod":                            []byte("module github.com/Glacius-Labs/Markitect\n\ngo 1.27.1\n"),
		"go.sum":                            []byte("example.com/dependency v1.0.0 h1:checksum\n"),
		"LICENSE":                           []byte("Apache License\nVersion 2.0\n"),
		"cmd/markitect/main.go":             []byte("package main\nvar version = \"1.2.3\"\nfunc main() {}\n"),
		"internal/example/example.go":       []byte("package example\nfunc Ready() bool { return true }\n"),
		"integration/run-markitect.go":      []byte("package main\nfunc main() {}\n"),
		"integration/run-markitect_test.go": []byte("package main\nimport \"testing\"\nfunc TestReady(t *testing.T) {}\n"),
	}
	snapshot := &snapshot.Snapshot{ID: commit, Files: files, Modes: map[string]string{}}
	for name := range files {
		snapshot.Modes[name] = "100644"
	}
	bundle, err := release.BuildBundle(snapshot, strings.TrimPrefix(tag, "v"))
	if err != nil {
		t.Fatalf("BuildBundle fixture: %v", err)
	}
	assets := namedBytes{
		assetNames(tag)[0]: bundle,
		assetNames(tag)[1]: []byte("\x7fELF-fake-linux-binary"),
		assetNames(tag)[2]: []byte("MZ-fake-windows-binary"),
	}
	p := validProvenance(tag, runID, attempt, commit, assets)
	assets[assetNames(tag)[3]] = mustYAML(t, p)
	dir := t.TempDir()
	writeAssets(t, dir, assets)
	return dir, assets
}

func validProvenance(tag, runID string, attempt int, commit string, assets namedBytes) Provenance {
	names := assetNames(tag)
	return Provenance{SchemaVersion: "1", Tag: tag, SourceCommit: commit, WorkflowRunID: runID, WorkflowRunAttempt: fmt.Sprint(attempt), WorkflowRun: fmt.Sprintf("https://github.com/%s/actions/runs/%s/attempts/%d", repository, runID, attempt), Toolchain: "go version go1.27.1 windows/amd64", Assets: []Asset{{names[0], digest(assets[names[0]])}, {names[1], digest(assets[names[1]])}, {names[2], digest(assets[names[2]])}}}
}

func mustYAML(t *testing.T, v any) []byte {
	t.Helper()
	b, err := yaml.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func writeAssets(t *testing.T, dir string, files namedBytes) {
	t.Helper()
	for n, b := range files {
		if err := os.WriteFile(filepath.Join(dir, n), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
func cloneAssets(in namedBytes) namedBytes {
	out := namedBytes{}
	for n, b := range in {
		out[n] = append([]byte(nil), b...)
	}
	return out
}
func containsCall(calls [][]string, fragment string) bool {
	for _, c := range calls {
		if strings.Contains(strings.Join(c, " "), fragment) {
			return true
		}
	}
	return false
}
func assertCallOrder(t *testing.T, calls [][]string, fragments ...string) {
	t.Helper()
	at := 0
	for _, fragment := range fragments {
		found := -1
		for i := at; i < len(calls); i++ {
			if strings.Contains(strings.Join(calls[i], " "), fragment) {
				found = i
				break
			}
		}
		if found < 0 {
			t.Fatalf("call %q missing or out of order: %v", fragment, calls)
		}
		at = found + 1
	}
}

type fakeGH struct {
	calls                                                     [][]string
	assets, artifactFiles                                     namedBytes
	run                                                       runInfo
	attemptRun                                                runInfo
	tagCommit                                                 string
	admin, immutable                                          bool
	releaseExists, draftCreated, published, badUploadedDigest bool
	releaseDraft                                              bool
	releaseImmutable                                          bool
	releasePublishedAt                                        string
	draftHasAssets                                            bool
	patchTransitionUnknown                                    bool
	releaseVerifyFailures                                     int
	assetVerifyFailures                                       map[string]int
	uploaded                                                  namedBytes
	usedArtifact                                              string
}

func newFakeGH(assets namedBytes) *fakeGH {
	run := runInfo{ID: 7654321, Name: "Release", Path: workflowPath + "@refs/tags/" + testTag, Event: "push", Status: "completed", Conclusion: "success", HeadBranch: testTag, HeadSHA: testCommit, RunAttempt: 2}
	return &fakeGH{assets: cloneAssets(assets), artifactFiles: cloneAssets(assets), run: run, attemptRun: run, tagCommit: testCommit, admin: true, immutable: true, releaseImmutable: true, releasePublishedAt: "2026-10-01T00:00:00Z", uploaded: namedBytes{}}
}

func (f *fakeGH) Run(_ context.Context, _ string, args ...string) ([]byte, error) {
	args = append([]string(nil), args...)
	f.calls = append(f.calls, args)
	if len(args) >= 2 && args[0] == "auth" && args[1] == "status" {
		return []byte("authenticated"), nil
	}
	if len(args) >= 2 && args[0] == "run" && args[1] == "download" {
		f.usedArtifact = valueAfter(args, "--name")
		dir := valueAfter(args, "--dir")
		for n, b := range f.artifactFiles {
			if err := os.WriteFile(filepath.Join(dir, n), b, 0600); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
	if len(args) >= 2 && args[0] == "release" && args[1] == "download" {
		dir := valueAfter(args, "--dir")
		for n, b := range f.assets {
			if err := os.WriteFile(filepath.Join(dir, n), b, 0600); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
	if len(args) >= 2 && args[0] == "release" && args[1] == "upload" {
		for _, p := range args[3:] {
			if strings.HasPrefix(p, "--") {
				break
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return nil, err
			}
			f.uploaded[filepath.Base(p)] = b
		}
		return nil, nil
	}
	if len(args) >= 2 && args[0] == "release" && args[1] == "verify" {
		if f.releaseVerifyFailures > 0 {
			f.releaseVerifyFailures--
			return nil, errors.New("release attestation is not yet available")
		}
		return []byte("verified"), nil
	}
	if len(args) >= 2 && args[0] == "release" && args[1] == "verify-asset" {
		if _, err := os.Stat(args[3]); err != nil {
			return nil, err
		}
		name := filepath.Base(args[3])
		if f.assetVerifyFailures[name] > 0 {
			f.assetVerifyFailures[name]--
			return nil, errors.New("asset attestation is not yet available")
		}
		return []byte("verified"), nil
	}
	if len(args) > 0 && args[0] == "api" {
		return f.api(args)
	}
	return nil, fmt.Errorf("unexpected fake gh command: %v", args)
}

func (f *fakeGH) api(args []string) ([]byte, error) {
	endpoint := ""
	for _, a := range args {
		if strings.HasPrefix(a, "repos/") {
			endpoint = a
		}
	}
	if strings.Contains(strings.Join(args, " "), "-X POST") {
		f.draftCreated = true
		draft := releaseInfo{ID: 42, TagName: testTag, Draft: true}
		if f.draftHasAssets {
			draft.Assets = []remoteAsset{{Name: "unexpected", Digest: "sha256:" + strings.Repeat("0", 64)}}
		}
		return json.Marshal(draft)
	}
	if strings.Contains(strings.Join(args, " "), "-X PATCH") {
		f.published = true
		if f.patchTransitionUnknown {
			return nil, errors.New("connection lost after patch request")
		}
		return json.Marshal(releaseInfo{ID: 42, TagName: testTag, Draft: false})
	}
	var body any
	status := 200
	switch {
	case endpoint == "repos/"+repository:
		body = map[string]any{"full_name": repository, "permissions": map[string]any{"admin": f.admin}}
	case endpoint == "repos/"+repository+"/immutable-releases":
		body = map[string]any{"enabled": f.immutable}
	case endpoint == fmt.Sprintf("repos/%s/actions/runs/%d", repository, 7654321):
		body = f.run
	case endpoint == fmt.Sprintf("repos/%s/actions/runs/%d/attempts/2", repository, 7654321):
		body = f.attemptRun
	case endpoint == fmt.Sprintf("repos/%s/actions/runs/%d/artifacts", repository, 7654321):
		body = map[string]any{"artifacts": []any{map[string]any{"name": "markitect-release-" + testCommit, "expired": false, "workflow_run": map[string]any{"id": 7654321}}}}
	case strings.Contains(endpoint, "/git/ref/tags/"):
		body = map[string]any{"object": map[string]any{"type": "commit", "sha": f.tagCommit}}
	case endpoint == "repos/"+repository+"/releases/tags/"+testTag:
		if !f.releaseExists {
			status = 404
			body = map[string]any{"message": "Not Found"}
		} else {
			assetRows := make([]remoteAsset, 0, len(f.assets))
			for name, data := range f.assets {
				assetRows = append(assetRows, remoteAsset{Name: name, Digest: "sha256:" + digest(data)})
			}
			body = releaseInfo{ID: 42, TagName: testTag, Draft: f.releaseDraft, Immutable: f.releaseImmutable, PublishedAt: f.releasePublishedAt, Assets: assetRows}
		}
	case endpoint == "repos/"+repository+"/releases/42/assets":
		list := make([]remoteAsset, 0, len(f.uploaded))
		for name, data := range f.uploaded {
			d := "sha256:" + digest(data)
			if f.badUploadedDigest {
				d = "sha256:" + strings.Repeat("0", 64)
			}
			list = append(list, remoteAsset{Name: name, Digest: d})
		}
		body = list
	case endpoint == "repos/"+repository+"/releases/42":
		list := make([]remoteAsset, 0, len(f.uploaded))
		for name, data := range f.uploaded {
			d := "sha256:" + digest(data)
			if f.badUploadedDigest {
				d = "sha256:" + strings.Repeat("0", 64)
			}
			list = append(list, remoteAsset{Name: name, Digest: d})
		}
		body = releaseInfo{ID: 42, TagName: testTag, Draft: !f.published, Immutable: f.published, Assets: list}
	default:
		return nil, fmt.Errorf("unexpected fake API endpoint %q; args=%v", endpoint, args)
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	out := append([]byte(fmt.Sprintf("HTTP/2.0 %d %s\r\nContent-Type: application/json\r\n\r\n", status, httpStatusText(status))), data...)
	if status == 404 {
		return out, errors.New("gh api returned HTTP 404")
	}
	return out, nil
}

func valueAfter(args []string, key string) string {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == key {
			return args[i+1]
		}
	}
	return ""
}
func httpStatusText(code int) string {
	if code == 200 {
		return "OK"
	}
	if code == 404 {
		return "Not Found"
	}
	return "Unknown"
}
