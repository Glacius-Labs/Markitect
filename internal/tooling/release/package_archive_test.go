package release

import (
	"bytes"
	"strings"
	"testing"
)

func TestNormalizeTextSourceRejectsInvalidUTF8AndNUL(t *testing.T) {
	for _, input := range [][]byte{{0xff, 0xfe}, []byte("valid\x00text")} {
		if _, err := normalizeTextSource("docs/readme.md", input); err == nil {
			t.Errorf("accepted invalid text bytes %v", input)
		}
	}
	if got, err := normalizeTextSource("schema/manifest.yaml", []byte("a\r\nb\rc")); err != nil || string(got) != "a\nb\nc" {
		t.Fatalf("normalization = %q, err=%v", got, err)
	}
	if got, err := normalizeTextSource("schema/legacy.yml", []byte("a\r\nb\rc")); err != nil || string(got) != "a\nb\nc" {
		t.Fatalf(".yml normalization = %q, err=%v", got, err)
	}
	for _, input := range [][]byte{{0xff}, []byte("has\x00nul")} {
		if _, err := normalizeTextSource("schema/legacy.yml", input); err == nil {
			t.Errorf(".yml accepted invalid text bytes %v", input)
		}
	}
}

func TestNormalizeTextSourceDoesNotRewriteOtherFiles(t *testing.T) {
	for _, name := range []string{"schema/manifest.json", "assets/payload.bin"} {
		input := []byte{0xff, 0x00, '\r', '\n'}
		got, err := normalizeTextSource(name, input)
		if err != nil || !bytes.Equal(got, input) {
			t.Errorf("normalizeTextSource(%q) changed non-target bytes: %v, %v", name, got, err)
		}
	}
}

func TestMakeArchiveRejectsCaseInsensitivePathCollisions(t *testing.T) {
	for _, files := range [][]sourceFile{
		{{name: "cmd/Foo.go"}, {name: "cmd/foo.go"}},
		{{name: "Internal/a.go"}, {name: "internal/b.go"}},
		{{name: "cmd"}, {name: "CMD/a.go"}},
		{{name: "cmd/a.go"}, {name: "cmd/a.go"}},
	} {
		if _, err := makeArchive(files); err == nil || !strings.Contains(err.Error(), "archive path collision") && !strings.Contains(err.Error(), "duplicate archive entry") {
			t.Errorf("makeArchive(%v) error = %v, want path collision", files, err)
		}
	}
}

func TestSafeArchivePathRejectsTraversalAndWindowsSpecialPaths(t *testing.T) {
	for _, candidate := range []string{"../outside.go", "/root.go", "C:/root.go", `internal\\escape.go`, "internal/../escape.go", "internal/CON.go", "internal/name?.go", "internal/a/../b.go"} {
		if err := safeArchivePath(candidate); err == nil {
			t.Errorf("accepted unsafe path %q", candidate)
		}
	}
	for _, candidate := range []string{"go.mod", "src/cmd/markitect/main.go", "schema/manifest.yaml"} {
		if err := safeArchivePath(candidate); err != nil {
			t.Errorf("rejected valid path %q: %v", candidate, err)
		}
	}
}
