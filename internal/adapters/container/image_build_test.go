package container

import (
	"archive/tar"
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func tarOf(t *testing.T, files map[string]string) *bytes.Reader {
	t.Helper()
	var out bytes.Buffer
	w := tar.NewWriter(&out)
	for name, text := range files {
		if err := w.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(text)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(text)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(out.Bytes())
}

func TestBuildLinesSplitAndKeepTheLastFew(t *testing.T) {
	var seen []string
	l := &buildLines{progress: func(s string) { seen = append(seen, s) }}
	_, _ = l.Write([]byte("#1 [build 1/2] FROM go\r\n#2 [bu"))
	_, _ = l.Write([]byte("ild 2/2] RUN make\n"))
	_, _ = l.Write([]byte("no newline"))
	l.flush()
	if strings.Join(seen, "|") != "#1 [build 1/2] FROM go|#2 [build 2/2] RUN make|no newline" {
		t.Fatalf("lines %q", seen)
	}
	for i := 0; i < 2*buildTailLines; i++ {
		l.line("x")
	}
	if len(l.tail) != buildTailLines {
		t.Fatalf("the tail keeps %d lines", len(l.tail))
	}
}

func TestImageCommandsRefuseWhatIsNotPlain(t *testing.T) {
	e := &Engine{path: "unused"}
	ctx := context.Background()
	if err := e.BuildImage(ctx, ImageBuild{Tag: "-x", Dockerfile: "D", Context: strings.NewReader("")}); err == nil {
		t.Error("a tag like a flag was built")
	}
	if err := e.BuildImage(ctx, ImageBuild{Tag: "a:b", Dockerfile: "-f", Context: strings.NewReader("")}); err == nil {
		t.Error("a recipe like a flag was built")
	}
	if err := e.BuildImage(ctx, ImageBuild{Tag: "a:b", Dockerfile: "D"}); err == nil {
		t.Error("a build without a context ran")
	}
	if _, err := e.Images(ctx, "a b"); err == nil {
		t.Error("a pattern with a space was listed")
	}
	if err := e.RemoveImage(ctx, ""); err == nil {
		t.Error("an empty reference was removed")
	}
}

// TestRealEngineBuildsAnImageFromATarContext runs with MARKITECT_TEST_ENGINE. Its
// image comes from scratch, so it pulls nothing.
func TestRealEngineBuildsAnImageFromATarContext(t *testing.T) {
	path := os.Getenv("MARKITECT_TEST_ENGINE")
	if path == "" {
		t.Skip("set MARKITECT_TEST_ENGINE to an absolute container engine path")
	}
	e, err := NewEngine(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	const tag = "markitect-build-test:app-1"
	t.Cleanup(func() { _ = e.RemoveImage(context.Background(), tag) })

	var lines []string
	err = e.BuildImage(ctx, ImageBuild{Tag: tag, Dockerfile: "dir/Recipe", Context: tarOf(t, map[string]string{
		"dir/Recipe": "FROM scratch\nCOPY a.txt /a.txt\n",
		"a.txt":      "hello\n",
	}), Progress: func(s string) { lines = append(lines, s) }})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(lines, "\n"), "COPY a.txt /a.txt") {
		t.Errorf("the build told %q", lines)
	}
	found, err := e.Images(ctx, "markitect-build-test:app-*")
	if err != nil || len(found) != 1 || found[0] != tag {
		t.Fatalf("images %q %v", found, err)
	}
	if _, err := e.ImageID(ctx, tag); err != nil {
		t.Fatal(err)
	}

	err = e.BuildImage(ctx, ImageBuild{Tag: tag + "-broken", Dockerfile: "Recipe", Context: tarOf(t, map[string]string{
		"Recipe": "FROM scratch\nCOPY missing.txt /missing.txt\n",
	})})
	if err == nil || !strings.Contains(err.Error(), "missing.txt") {
		t.Fatalf("a broken build said %v", err)
	}
	if found, _ := e.Images(ctx, tag+"-broken"); len(found) != 0 {
		t.Errorf("a broken build left the tag: %q", found)
	}

	if err := e.RemoveImage(ctx, tag); err != nil {
		t.Fatal(err)
	}
	if found, err := e.Images(ctx, tag); err != nil || len(found) != 0 {
		t.Fatalf("after removal %q %v", found, err)
	}
}
