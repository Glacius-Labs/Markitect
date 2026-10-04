package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
)

type zipFixtureEntry struct {
	name string
	data []byte
	mode os.FileMode
	attr uint32
}

func makeZip(t *testing.T, entries []zipFixtureEntry) []byte {
	t.Helper()
	var output bytes.Buffer
	w := zip.NewWriter(&output)
	for _, item := range entries {
		header := &zip.FileHeader{Name: item.name, Method: zip.Store}
		header.SetMode(item.mode)
		header.ExternalAttrs |= item.attr
		writer, err := w.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(item.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func validZipEntries() []zipFixtureEntry {
	return []zipFixtureEntry{
		{name: "go.mod", data: []byte("module markitect\n\ngo 1.24.0\n"), mode: 0644},
		{name: "go.sum", data: []byte(""), mode: 0644},
		{name: "cmd/markitect/main.go", data: []byte("package main\n"), mode: 0644},
		{name: "internal/app/host.go", data: []byte("package app\n"), mode: 0644},
	}
}

func fixtureManifest(data []byte) manifest {
	digest := sha256.Sum256(data)
	return manifest{version: "0.1.0-dev", source: ".markitect/tool/source.zip", sha256: hex.EncodeToString(digest[:])}
}

func TestParseManifestStrictFieldsAndCRLF(t *testing.T) {
	good := []byte("version: \"1.2.3-rc.4+build.7\"\nsource: \".markitect/tool/source.zip\"\nsha256: \"" + strings.Repeat("a", 64) + "\"\n")
	parsed, err := parseManifest(bytes.ReplaceAll(good, []byte("\n"), []byte("\r\n")))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.version != "1.2.3-rc.4+build.7" || parsed.sha256 != strings.Repeat("a", 64) {
		t.Fatalf("unexpected parsed manifest: %#v", parsed)
	}
	bad := [][]byte{
		[]byte("version: \"1.0.0\"\nversion: \"1.0.0\"\nsha256: \"" + strings.Repeat("a", 64) + "\"\n"),
		[]byte("version: \"1.0.0\"\nsource: \"x.zip\"\nsha256: \"" + strings.Repeat("A", 64) + "\"\n"),
		[]byte("version: \"1.0.0\"\nsource: \"x.zip\"\nsha256: \"" + strings.Repeat("a", 64) + "\"\nextra: \"no\"\n"),
		[]byte("version: \"1.0.0\"\rsource: \"x.zip\"\nsha256: \"" + strings.Repeat("a", 64) + "\"\n"),
		[]byte("version: \"1.0.0\" trailing\nsource: \"x.zip\"\nsha256: \"" + strings.Repeat("a", 64) + "\"\n"),
		[]byte("version: \"latest\"\nsource: \"x.zip\"\nsha256: \"" + strings.Repeat("a", 64) + "\"\n"),
	}
	for _, sample := range bad {
		if _, err := parseManifest(sample); err == nil {
			t.Errorf("accepted invalid manifest %q", sample)
		}
	}
}

func TestReadManifestBoundsAndRejectsLink(t *testing.T) {
	root := t.TempDir()
	lock := []byte("version: \"1.2.3\"\nsource: \".markitect/tool/source.zip\"\nsha256: \"" + strings.Repeat("a", 64) + "\"\n")
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, lockName)), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, lockName), bytes.Repeat([]byte("x"), maxLockSize+1), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readManifest(root); err == nil {
		t.Fatal("oversized lock was accepted")
	}
	if err := os.WriteFile(filepath.Join(root, lockName), lock, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".markitect", "tool"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".markitect", "tool", "source.zip"), []byte("not a zip"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readManifest(root); err != nil {
		t.Fatalf("valid bounded manifest rejected: %v", err)
	}
}

func TestReadBoundedRegularFileEnforcesSizeAndType(t *testing.T) {
	root := t.TempDir()
	regular := filepath.Join(root, "regular")
	if err := os.WriteFile(regular, []byte("12345"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := readBoundedRegularFile(regular, 5, "fixture"); err != nil {
		t.Fatalf("exact size was rejected: %v", err)
	}
	if _, err := readBoundedRegularFile(regular, 4, "fixture"); err == nil {
		t.Fatal("oversized regular file was accepted")
	}
	if _, err := readBoundedRegularFile(root, 10, "fixture"); err == nil {
		t.Fatal("directory was accepted as a regular file")
	}
}

func TestSafeRepoPathRejectsTraversalAndWindowsNames(t *testing.T) {
	for _, name := range []string{"../escape.zip", "/absolute.zip", "C:/drive.zip", `a\\b.zip`, "a/../b.zip", "a/CON.txt", "a/name?.zip", "a/trailing."} {
		if err := safeRepoPath(name); err == nil {
			t.Errorf("accepted unsafe path %q", name)
		}
	}
	root := t.TempDir()
	if _, err := checkedRepoPath(root, "../escape.zip", false); err == nil {
		t.Fatal("checkedRepoPath accepted traversal")
	}
}

func TestReadArchiveVerifiesDigestLayoutAndRejectsUnsafeEntries(t *testing.T) {
	valid := makeZip(t, validZipEntries())
	manifest := fixtureManifest(valid)
	archivePath := filepath.Join(t.TempDir(), "source.zip")
	if err := os.WriteFile(archivePath, valid, 0644); err != nil {
		t.Fatal(err)
	}
	a, err := readArchive(archivePath, manifest.sha256, defaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.files) != 4 {
		t.Fatalf("archive has %d files, want 4", len(a.files))
	}
	if _, err := readArchive(archivePath, strings.Repeat("b", 64), defaultLimits); err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("wrong digest error = %v", err)
	}

	for _, test := range []struct {
		name    string
		entries []zipFixtureEntry
		limits  limits
	}{
		{name: "traversal", entries: append(validZipEntries(), zipFixtureEntry{name: "../escape", mode: 0644})},
		{name: "symlink", entries: append(validZipEntries(), zipFixtureEntry{name: "link", data: []byte("target"), mode: os.ModeSymlink | 0777})},
		{name: "reparse", entries: append(validZipEntries(), zipFixtureEntry{name: "reparse", data: []byte("x"), mode: 0644, attr: 0x400})},
		{name: "case-collision", entries: append(validZipEntries(), zipFixtureEntry{name: "CMD/other.go", data: []byte("x"), mode: 0644})},
		{name: "file-directory-collision", entries: append(validZipEntries(), zipFixtureEntry{name: "internal", data: []byte("x"), mode: 0644})},
		{name: "count-limit", entries: validZipEntries(), limits: limits{maxEntries: 2, maxFile: 100, maxTotal: 1000, maxArchive: 1000}},
		{name: "file-limit", entries: append(validZipEntries(), zipFixtureEntry{name: "internal/large.go", data: bytes.Repeat([]byte("a"), 8), mode: 0644}), limits: limits{maxEntries: 100, maxFile: 4, maxTotal: 100, maxArchive: 10000}},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := makeZip(t, test.entries)
			path := filepath.Join(t.TempDir(), "source.zip")
			if err := os.WriteFile(path, data, 0644); err != nil {
				t.Fatal(err)
			}
			lim := test.limits
			if lim.maxEntries == 0 {
				lim = defaultLimits
			}
			_, err := readArchive(path, hexDigest(data), lim)
			if err == nil {
				t.Fatal("unsafe or over-limit archive was accepted")
			}
		})
	}
}

func TestExtractedSourceIsVerifiedAndTamperingInvalidatesIt(t *testing.T) {
	data := makeZip(t, validZipEntries())
	archivePath := filepath.Join(t.TempDir(), "source.zip")
	if err := os.WriteFile(archivePath, data, 0644); err != nil {
		t.Fatal(err)
	}
	a, err := readArchive(archivePath, hexDigest(data), defaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	cache := t.TempDir()
	sourceDir, err := findOrExtractSource(cache, a)
	if err != nil {
		t.Fatal(err)
	}
	if valid, err := verifyExtracted(sourceDir, a); err != nil || !valid {
		t.Fatalf("valid extracted source = %v, err=%v", valid, err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "cmd", "markitect", "main.go"), []byte("tampered"), 0644); err != nil {
		t.Fatal(err)
	}
	if valid, err := verifyExtracted(sourceDir, a); err != nil || valid {
		t.Fatalf("tampered source verification = %v, err=%v", valid, err)
	}
}

func TestBuildCacheReusesOnlyVerifiedBinaryAndUsesLocalGoCaches(t *testing.T) {
	data := makeZip(t, validZipEntries())
	archivePath := filepath.Join(t.TempDir(), "source.zip")
	if err := os.WriteFile(archivePath, data, 0644); err != nil {
		t.Fatal(err)
	}
	archive, err := readArchive(archivePath, hexDigest(data), defaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	m := fixtureManifest(data)
	t.Setenv("GOCACHE", "")
	t.Setenv("GOTMPDIR", "")
	buildCount := 0
	var buildArgs []string
	var buildEnv []string
	toolchain := "go1.27.1"
	probe := func(_ string, _ string, env []string) (string, error) {
		if envValue(env, "GOTOOLCHAIN") != "auto" {
			t.Fatalf("toolchain probe was not allowed to select module toolchain")
		}
		return toolchain, nil
	}
	fakeBuild := func(_ string, args []string, dir string, env []string) error {
		buildCount++
		buildArgs = append([]string(nil), args...)
		buildEnv = append([]string(nil), env...)
		if dir == "" {
			t.Fatal("builder received empty source directory")
		}
		i := indexOf(args, "-o")
		if i < 0 || i+1 == len(args) {
			t.Fatalf("builder output argument missing: %v", args)
		}
		return os.WriteFile(args[i+1], []byte("mock executable"), 0755)
	}
	first, err := buildOrReuse(root, m, archive, "fake-go", os.Environ(), fakeBuild, probe)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildOrReuse(root, m, archive, "fake-go", os.Environ(), fakeBuild, probe)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || buildCount != 1 {
		t.Fatalf("cache reused %q -> %q with %d builds", first, second, buildCount)
	}
	noiseEnv := []string{"PATH=original", "GOFLAGS=-overlay=foreign", "GOAMD64=v3", "GOEXPERIMENT=loopvar", "GOTOOLCHAIN=local", "UNRELATED_SETTING=changed"}
	noiseReuse, err := buildOrReuse(root, m, archive, "fake-go", noiseEnv, fakeBuild, probe)
	if err != nil {
		t.Fatal(err)
	}
	if noiseReuse != first || buildCount != 1 {
		t.Fatalf("irrelevant caller environment invalidated cache: builds=%d", buildCount)
	}
	if indexOf(buildArgs, "-buildvcs=false") < 0 || indexOf(buildArgs, "-trimpath") < 0 || !contains(buildArgs, "-X main.version="+m.version+" -X github.com/Glacius-Labs/Markitect/internal/host/cli.version="+m.version) {
		t.Fatalf("build flags missing: %v", buildArgs)
	}
	base := filepath.Join(root, ".artifacts", "markitect")
	if envValue(buildEnv, "GOCACHE") != filepath.Join(base, "go-build") || envValue(buildEnv, "GOTMPDIR") != filepath.Join(base, "go-tmp") {
		t.Fatalf("unexpected local Go cache env keys: %v", envKeys(buildEnv))
	}
	if envValue(buildEnv, "GOFLAGS") != "" || envValue(buildEnv, "GOWORK") != "off" || envValue(buildEnv, "GOENV") != "off" || envValue(buildEnv, "GOOS") != runtime.GOOS || envValue(buildEnv, "GOARCH") != runtime.GOARCH || envValue(buildEnv, "CGO_ENABLED") != "0" || envValue(buildEnv, "GOTOOLCHAIN") != toolchain {
		t.Fatalf("Go build env is not isolated to native pinned source: %v", envKeys(buildEnv))
	}
	archFeature := map[string]string{"amd64": "GOAMD64", "arm64": "GOARM64", "386": "GO386", "arm": "GOARM"}[runtime.GOARCH]
	if archFeature != "" && envValue(buildEnv, archFeature) == "" {
		t.Fatalf("portable architecture default %s is missing", archFeature)
	}
	if _, err := os.Stat(filepath.Join(base, "go-build")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(base, "go-tmp")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(first, []byte("replaced executable"), 0755); err != nil {
		t.Fatal(err)
	}
	third, err := buildOrReuse(root, m, archive, "fake-go", os.Environ(), fakeBuild, probe)
	if err != nil {
		t.Fatal(err)
	}
	if buildCount != 2 || third == first {
		t.Fatalf("replaced executable was not rebuilt: builds=%d old=%s new=%s", buildCount, first, third)
	}
	toolchain = "go1.28.0"
	fourth, err := buildOrReuse(root, m, archive, "fake-go", os.Environ(), fakeBuild, probe)
	if err != nil {
		t.Fatal(err)
	}
	if buildCount != 3 || fourth == third {
		t.Fatalf("toolchain change did not invalidate cache: builds=%d old=%s new=%s", buildCount, third, fourth)
	}
}

func TestBuildEnvironmentPreservesExplicitOverrides(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0755); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(root, "shared-cache")
	if err := os.Mkdir(cache, 0755); err != nil {
		t.Fatal(err)
	}
	callerCache := filepath.Join(root, "caller-cache")
	callerTmp := filepath.Join(root, "caller-tmp")
	if err := os.Mkdir(callerCache, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(callerTmp, 0755); err != nil {
		t.Fatal(err)
	}
	env, err := environmentForBuild(source, cache, []string{"PATH=original", "GOCACHE=" + callerCache, "GOTMPDIR=" + callerTmp, "GOPROXY=https://user:secret@example.invalid", "GOSUMDB=off", "GOAUTH=netrc", "GOFLAGS=-overlay=evil.json", "GOWORK=../go.work", "GOENV=evil", "GOOS=wasm", "GOARCH=wasm", "GOAMD64=v3", "GOEXPERIMENT=evil", "GOFIPS140=latest", "GOCACHEPROG=external-cache", "GODEBUG=toolchaintrace=1", "GOTOOLCHAIN=local", "CGO_ENABLED=1", "CGO_CFLAGS=-evil", "GOOGLE_APPLICATION_CREDENTIALS=fake-test-credential-path"}, "go1.27.1")
	if err != nil {
		t.Fatal(err)
	}
	if envValue(env, "GOCACHE") != callerCache || envValue(env, "GOTMPDIR") != callerTmp || envValue(env, "PATH") != "original" {
		t.Fatalf("explicit build env changed; keys=%v", envKeys(env))
	}
	if envValue(env, "GOFLAGS") != "" || envValue(env, "GOWORK") != "off" || envValue(env, "GOENV") != "off" || envValue(env, "GOOS") != runtime.GOOS || envValue(env, "GOARCH") != runtime.GOARCH || envValue(env, "GOAMD64") == "v3" || envValue(env, "GOEXPERIMENT") != "" || envValue(env, "GOFIPS140") != "" || envValue(env, "GOCACHEPROG") != "" || envValue(env, "GODEBUG") != "" || envValue(env, "CGO_ENABLED") != "0" || envValue(env, "CGO_CFLAGS") != "" || envValue(env, "GOTOOLCHAIN") != "go1.27.1" {
		t.Fatalf("unsafe Go build variables were retained; keys=%v", envKeys(env))
	}
	if envValue(env, "GOPROXY") != "https://user:secret@example.invalid" || envValue(env, "GOSUMDB") != "off" || envValue(env, "GOAUTH") != "netrc" || envValue(env, "GOOGLE_APPLICATION_CREDENTIALS") != "fake-test-credential-path" {
		t.Fatalf("module download/checksum settings were not preserved; keys=%v", envKeys(env))
	}
}

func TestBuildMutationIsNotStampedAsPinnedSource(t *testing.T) {
	data := makeZip(t, validZipEntries())
	archivePath := filepath.Join(t.TempDir(), "source.zip")
	if err := os.WriteFile(archivePath, data, 0644); err != nil {
		t.Fatal(err)
	}
	archive, err := readArchive(archivePath, hexDigest(data), defaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	m := fixtureManifest(data)
	fakeBuild := func(_ string, args []string, dir string, _ []string) error {
		if err := os.WriteFile(args[indexOf(args, "-o")+1], []byte("mock executable"), 0755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, "cmd", "markitect", "main.go"), []byte("mutated during build"), 0644)
	}
	probe := func(_ string, _ string, _ []string) (string, error) { return "go1.27.1", nil }
	if _, err := buildOrReuse(root, m, archive, "fake-go", nil, fakeBuild, probe); err == nil || !strings.Contains(err.Error(), "changed during build") {
		t.Fatalf("source mutation result = %v", err)
	}
	platformDir := filepath.Join(root, ".artifacts", "markitect", m.sha256, runtime.GOOS+"-"+runtime.GOARCH)
	entries, err := os.ReadDir(platformDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "build-stamp-") {
			t.Fatalf("mutated source received build stamp %s", entry.Name())
		}
	}
}

func TestInjectRepoAndDiscoverRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, lockName)), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, lockName), []byte("lock"), 0644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "sub", "dir")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	gotRoot, err := discoverRoot(nested)
	if err != nil || gotRoot != filepath.Clean(root) {
		t.Fatalf("discoverRoot = %q, want=%q, err=%v", gotRoot, filepath.Clean(root), err)
	}
	if got := injectRepo([]string{"check", "--revision", "abc"}, root); !reflect.DeepEqual(got, []string{"check", "--repo", root, "--revision", "abc"}) {
		t.Fatalf("repo args = %v", got)
	}
	if got := injectRepo([]string{"check", "--repo=other"}, root); !reflect.DeepEqual(got, []string{"check", "--repo=other"}) {
		t.Fatalf("explicit repo changed: %v", got)
	}
	if got := injectRepo([]string{"version"}, root); !reflect.DeepEqual(got, []string{"version"}) {
		t.Fatalf("version args changed: %v", got)
	}
	if got := injectRepo([]string{"authoring"}, root); !reflect.DeepEqual(got, []string{"authoring"}) {
		t.Fatalf("authoring args changed: %v", got)
	}
	if got := injectRepo([]string{"authoring", "--format", "yaml"}, root); !reflect.DeepEqual(got, []string{"authoring", "--format", "yaml"}) {
		t.Fatalf("authoring flags changed: %v", got)
	}
	for _, args := range [][]string{{"licenses"}, {"licenses", "--write"}, {"help"}, {"help", "init"}, {"--help"}, {"-h"}} {
		if got := injectRepo(args, root); !reflect.DeepEqual(got, args) {
			t.Fatalf("repository-independent arguments changed: got=%v want=%v", got, args)
		}
	}
	if got := injectRepo([]string{"find", "--query", "authoring"}, root); !reflect.DeepEqual(got, []string{"find", "--repo", root, "--query", "authoring"}) {
		t.Fatalf("consumer query did not receive repo: %v", got)
	}
}

func TestArchiveStampParser(t *testing.T) {
	digest := strings.Repeat("a", 64)
	data := stampYAML("1.2.3", digest, "go1.27.1", buildPolicy, strings.Repeat("b", 64))
	stamp, err := parseStamp(data)
	if err != nil || stamp["source_sha256"] != digest {
		t.Fatalf("parse stamp = %v, %v", stamp, err)
	}
	if _, err := parseStamp([]byte("version: \"1.2.3\"\nversion: \"1.2.3\"\nexecutable_sha256: \"" + strings.Repeat("b", 64) + "\"\n")); err == nil {
		t.Fatal("duplicate stamp fields accepted")
	}
}

func TestCacheRejectsBuildPolicyStampMismatch(t *testing.T) {
	platformDir := t.TempDir()
	digest := strings.Repeat("a", 64)
	m := manifest{version: "1.2.3", sha256: digest}
	nonce := "0011223344556677"
	executable := filepath.Join(platformDir, "markitect-"+nonce)
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}
	contents := []byte("mock executable")
	if err := os.WriteFile(executable, contents, 0755); err != nil {
		t.Fatal(err)
	}
	stamp := stampYAML(m.version, m.sha256, "go1.27.1", "older-policy", hexDigest(contents))
	if err := os.WriteFile(filepath.Join(platformDir, "build-stamp-"+nonce+".yaml"), stamp, 0644); err != nil {
		t.Fatal(err)
	}
	if got, err := findCachedExecutable(platformDir, m, "go1.27.1"); err != nil || got != "" {
		t.Fatalf("stale build policy cache result = %q, %v", got, err)
	}
}

func hexDigest(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func indexOf(values []string, value string) int {
	for i, item := range values {
		if item == value {
			return i
		}
	}
	return -1
}

func contains(values []string, value string) bool { return indexOf(values, value) >= 0 }

func envValue(values []string, key string) string {
	for _, value := range values {
		name, content, ok := strings.Cut(value, "=")
		if ok && name == key {
			return content
		}
	}
	return ""
}

func envKeys(values []string) []string {
	keys := make([]string, 0, len(values))
	for _, value := range values {
		key, _, ok := strings.Cut(value, "=")
		if ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}
