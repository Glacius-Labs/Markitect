package upgrade

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// release serves the releases page of a project: /latest redirects to the tag,
// and /download/TAG/NAME serves what files holds.
func release(t *testing.T, tag string, files map[string]string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/latest", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/tag/"+tag, http.StatusFound)
	})
	mux.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/download/"+tag+"/")
		body, ok := files[name]
		if !ok || !strings.HasPrefix(r.URL.Path, "/download/"+tag+"/") {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, body)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func sum(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// published is the files of a release for the platform goos/goarch.
func published(version, goos, goarch, binary string) map[string]string {
	asset := AssetName(version, goos, goarch)
	return map[string]string{
		asset:                  binary,
		ChecksumsName(version): sum("other") + "  some-other-file\n" + sum(binary) + "  " + asset + "\n",
	}
}

func TestLatestReadsTheTagFromTheRedirect(t *testing.T) {
	srv := release(t, "v0.15.0", nil)
	got, err := Client{Base: srv.URL}.Latest(context.Background())
	if err != nil || got != "0.15.0" {
		t.Fatalf("Latest() = %q, %v; want 0.15.0", got, err)
	}
}

func TestLatestWithoutARelease(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"not found":         http.NotFound,
		"the releases page": func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/releases", http.StatusFound) },
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(handler)
			defer srv.Close()
			if _, err := (Client{Base: srv.URL}).Latest(context.Background()); err != ErrNoRelease {
				t.Fatalf("Latest() error = %v, want ErrNoRelease", err)
			}
		})
	}
}

func TestLatestRefusesATagThatIsNoVersion(t *testing.T) {
	for _, tag := range []string{"nightly", "v1.2", "v1.2.3/../x", "v1.2.3;rm"} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "/tag/"+tag)
			w.WriteHeader(http.StatusFound)
		}))
		_, err := Client{Base: srv.URL}.Latest(context.Background())
		srv.Close()
		if err == nil || err == ErrNoRelease {
			t.Errorf("tag %q: Latest() error = %v, want a refusal", tag, err)
		}
	}
}

func TestLatestFailsOnAServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", http.StatusBadGateway) }))
	defer srv.Close()
	if _, err := (Client{Base: srv.URL}).Latest(context.Background()); err == nil {
		t.Fatal("Latest() accepted a server error")
	}
}

func TestInstallReplacesTheExecutable(t *testing.T) {
	for _, goos := range []string{"linux", "darwin", "windows"} {
		t.Run(goos, func(t *testing.T) {
			dir := t.TempDir()
			exe := filepath.Join(dir, "markitect")
			if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
				t.Fatal(err)
			}
			srv := release(t, "v0.15.0", published("0.15.0", goos, "arm64", "new binary"))
			c := Client{Base: srv.URL, GOOS: goos, GOARCH: "arm64"}
			if err := c.Install(context.Background(), "0.15.0", exe); err != nil {
				t.Fatal(err)
			}
			if got, _ := os.ReadFile(exe); string(got) != "new binary" {
				t.Fatalf("the executable holds %q", got)
			}
			if runtime.GOOS != "windows" {
				if st, _ := os.Stat(exe); st.Mode().Perm()&0o111 == 0 {
					t.Fatalf("the executable lost its execute bit: %v", st.Mode())
				}
			}
			entries, _ := os.ReadDir(dir)
			for _, e := range entries {
				if e.Name() != "markitect" && e.Name() != "markitect.old" {
					t.Errorf("Install left %s behind", e.Name())
				}
			}
		})
	}
}

func TestInstallRefusesABinaryThatDoesNotMatchItsChecksum(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "markitect")
	os.WriteFile(exe, []byte("old"), 0o755)
	files := published("0.15.0", "linux", "amd64", "new binary")
	files[AssetName("0.15.0", "linux", "amd64")] = "tampered"
	srv := release(t, "v0.15.0", files)
	err := Client{Base: srv.URL, GOOS: "linux", GOARCH: "amd64"}.Install(context.Background(), "0.15.0", exe)
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("Install() error = %v, want a checksum refusal", err)
	}
	assertUntouched(t, dir, exe)
}

func TestInstallRefusesAReleaseWithoutAChecksumForThisPlatform(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "markitect")
	os.WriteFile(exe, []byte("old"), 0o755)
	files := published("0.15.0", "linux", "amd64", "new binary")
	files[ChecksumsName("0.15.0")] = sum("x") + "  something-else\n"
	srv := release(t, "v0.15.0", files)
	err := Client{Base: srv.URL, GOOS: "linux", GOARCH: "amd64"}.Install(context.Background(), "0.15.0", exe)
	if err == nil || !strings.Contains(err.Error(), "do not list") {
		t.Fatalf("Install() error = %v, want a refusal", err)
	}
	assertUntouched(t, dir, exe)
}

func TestInstallKeepsTheOldBinaryWhenThereIsNoDownload(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "markitect")
	os.WriteFile(exe, []byte("old"), 0o755)
	files := published("0.15.0", "linux", "amd64", "new binary")
	delete(files, AssetName("0.15.0", "linux", "amd64"))
	srv := release(t, "v0.15.0", files)
	err := Client{Base: srv.URL, GOOS: "linux", GOARCH: "amd64"}.Install(context.Background(), "0.15.0", exe)
	if err == nil || !strings.Contains(err.Error(), "has no") {
		t.Fatalf("Install() error = %v, want the missing binary named", err)
	}
	assertUntouched(t, dir, exe)
}

func TestInstallRefusesAPlatformWithoutBinaries(t *testing.T) {
	err := Client{Base: "http://unused.invalid", GOOS: "plan9", GOARCH: "386"}.Install(context.Background(), "0.15.0", filepath.Join(t.TempDir(), "markitect"))
	if err == nil || !strings.Contains(err.Error(), "plan9/386") {
		t.Fatalf("Install() error = %v, want the platform named", err)
	}
}

// assertUntouched checks that a failed Install left the executable alone and
// no temporary file next to it.
func assertUntouched(t *testing.T, dir, exe string) {
	t.Helper()
	if got, _ := os.ReadFile(exe); string(got) != "old" {
		t.Errorf("the executable holds %q after a failed install", got)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("a failed install left %v", names)
	}
}

func TestAssetNames(t *testing.T) {
	if got := AssetName("0.15.0", "windows", "amd64"); got != "markitect-v0.15.0-windows-amd64.exe" {
		t.Errorf("windows: %s", got)
	}
	if got := AssetName("0.15.0", "linux", "arm64"); got != "markitect-v0.15.0-linux-arm64" {
		t.Errorf("linux: %s", got)
	}
	if got := ChecksumsName("0.15.0"); got != "markitect-v0.15.0-checksums.txt" {
		t.Errorf("checksums: %s", got)
	}
}

func TestNewReadsTheMirrorFromTheEnvironment(t *testing.T) {
	t.Setenv(BaseEnv, "http://mirror.example/releases/")
	if got := New().base(); got != "http://mirror.example/releases" {
		t.Errorf("base = %q", got)
	}
	t.Setenv(BaseEnv, "")
	if got := New().base(); got != DefaultBase {
		t.Errorf("base = %q, want the default", got)
	}
}

func TestRunReturnsTheExitCode(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MARKITECT_UPGRADE_HELPER", "3")
	var out, errout bytes.Buffer
	code, err := Client{}.Run(context.Background(), exe, []string{"-test.run=TestHelperProcess"}, &out, &errout)
	if err != nil || code != 3 {
		t.Fatalf("Run() = %d, %v; want 3", code, err)
	}
	if !strings.Contains(out.String(), "helper says hi") {
		t.Errorf("stdout = %q", out.String())
	}
}

func TestRunReportsAProgramThatCannotStart(t *testing.T) {
	if _, err := (Client{}).Run(context.Background(), filepath.Join(t.TempDir(), "absent"), nil, nil, nil); err == nil {
		t.Fatal("Run() started a program that does not exist")
	}
}

// TestHelperProcess is what TestRunReturnsTheExitCode starts: this test binary,
// which prints a line and exits with the code in the environment.
func TestHelperProcess(t *testing.T) {
	code := os.Getenv("MARKITECT_UPGRADE_HELPER")
	if code == "" {
		t.Skip("only run by TestRunReturnsTheExitCode")
	}
	fmt.Println("helper says hi")
	n := 0
	fmt.Sscan(code, &n)
	os.Exit(n)
}
