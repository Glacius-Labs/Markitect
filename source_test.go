package markitect

import (
	"io/fs"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// The recipe and what it copies must stay inside what the app carries, or the
// app's own build of the image fails on a user's machine and nowhere else.
func TestImageSourceHoldsWhatTheRecipeCopies(t *testing.T) {
	recipe, err := fs.ReadFile(ImageSource, "packaging/runtime/Dockerfile")
	if err != nil {
		t.Fatal(err)
	}
	copied := 0
	for _, line := range strings.Split(string(recipe), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] != "COPY" || strings.HasPrefix(fields[1], "--") {
			continue
		}
		for _, source := range fields[1 : len(fields)-1] {
			copied++
			if _, err := fs.Stat(ImageSource, source); err != nil {
				t.Errorf("the recipe copies %s, which the app does not carry", source)
			}
		}
	}
	if copied == 0 {
		t.Error("the recipe copies nothing from its context")
	}
}

// What the app carries must compile by itself: a file left out, such as one
// whose name starts with an underscore, would break only the app's own build.
func TestImageSourceCompilesForTheImage(t *testing.T) {
	dir := t.TempDir()
	if err := os.CopyFS(dir, ImageSource); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", "-trimpath", "-o", os.DevNull, "./cmd/markitect")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0", "GOFLAGS=-buildvcs=false")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("markitect: %v\n%s", err, out)
	}
	cmd = exec.Command("go", "build", "-trimpath", "-o", os.DevNull, "./cmd/markitect-agentd")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0", "GOFLAGS=-buildvcs=false")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("markitect-agentd: %v\n%s", err, out)
	}
}

// The ignore file bounds the context of a build from a checkout; the app must
// carry at least as much.
func TestImageSourceHoldsWhatTheIgnoreFileAdmits(t *testing.T) {
	ignore, err := os.ReadFile("packaging/runtime/Dockerfile.dockerignore")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(ignore), "\n") {
		admitted, ok := strings.CutPrefix(strings.TrimSpace(line), "!")
		if !ok {
			continue
		}
		admitted = strings.TrimSuffix(admitted, "/")
		if _, err := fs.Stat(ImageSource, admitted); err != nil {
			t.Errorf("the ignore file admits %s, which the app does not carry", admitted)
		}
	}
}
