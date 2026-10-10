package projectworkspace

import (
	"bytes"
	"strings"
	"testing"
)

func testRequest(t *testing.T) Request {
	t.Helper()
	return Request{
		RepositoryRoot:     t.TempDir(),
		RepositoryIdentity: "repo:sample",
		BaseSHA:            strings.Repeat("a", 40),
		OverlayDigest:      "sha256:" + strings.Repeat("b", 64),
		TaskID:             "manager:sample",
		AllowedPaths:       []string{"src/", "docs"},
		ExcludedPaths:      []string{"src/generated/"},
	}
}

func testHandle(r Request) Handle {
	return Handle{
		ID:                 "workspace-1",
		CWD:                r.RepositoryRoot + "/candidate",
		RepositoryRoot:     r.RepositoryRoot,
		RepositoryIdentity: r.RepositoryIdentity,
		BaseSHA:            r.BaseSHA,
		OverlayDigest:      r.OverlayDigest,
		TaskID:             r.TaskID,
		BaseDigest:         "sha256:" + strings.Repeat("c", 64),
	}
}

var testLimits = Limits{MaxFiles: 8, MaxFileBytes: 64, MaxTotalBytes: 128}

func TestNormalizeDeltaSupportsBinaryDeleteRenameAndDeterministicOrder(t *testing.T) {
	r := testRequest(t)
	h := testHandle(r)
	changes := []Change{
		{Kind: ChangeRename, OldPath: "src/old.bin", Path: "src/new.bin", Mode: "100755", Content: []byte{0, 0xff, 0x01}},
		{Kind: ChangeDelete, Path: "docs/retired.md"},
		{Kind: ChangeAdd, Path: "src/add.bin", Mode: "100644", Content: []byte{0x00, 0x80, 0xfe}},
	}
	delta, err := NormalizeDelta(r, h, changes, testLimits)
	if err != nil {
		t.Fatal(err)
	}
	if len(delta.Changes) != 3 || delta.Changes[0].Path != "docs/retired.md" || delta.Changes[1].Path != "src/add.bin" {
		t.Fatalf("changes not canonically sorted: %#v", delta.Changes)
	}
	if delta.Changes[2].Kind != ChangeRename || delta.Changes[2].OldPath != "src/old.bin" || !bytes.Equal(delta.Changes[2].Content, changes[0].Content) {
		t.Fatalf("rename/binary bytes were not preserved: %#v", delta.Changes[2])
	}
	if delta.Changes[0].Content != nil || delta.Changes[0].Mode != "" {
		t.Fatalf("delete unexpectedly carries content or mode: %#v", delta.Changes[0])
	}
	reversed := []Change{changes[2], changes[1], changes[0]}
	reordered, err := NormalizeDelta(r, h, reversed, testLimits)
	if err != nil {
		t.Fatal(err)
	}
	if delta.Digest != reordered.Digest {
		t.Fatalf("digest depends on caller order: %s != %s", delta.Digest, reordered.Digest)
	}
	changes[0].Content[0] = 0x7f
	if delta.Changes[2].Content[0] != 0 {
		t.Fatal("normalized delta retained caller-owned mutable content bytes")
	}
}

func TestDeltaDigestBindsBaseAndOverlayAndHandleMatchesRequest(t *testing.T) {
	r := testRequest(t)
	h := testHandle(r)
	change := []Change{{Kind: ChangeModify, Path: "src/main.go", Mode: "100644", Content: []byte("package main\n")}}
	one, err := NormalizeDelta(r, h, change, testLimits)
	if err != nil {
		t.Fatal(err)
	}
	other := r
	other.BaseSHA = strings.Repeat("d", 40)
	otherHandle := testHandle(other)
	two, err := NormalizeDelta(other, otherHandle, change, testLimits)
	if err != nil {
		t.Fatal(err)
	}
	if one.Digest == two.Digest || one.BaseSHA == two.BaseSHA {
		t.Fatal("delta digest did not bind the immutable base SHA")
	}
	other.OverlayDigest = "sha256:" + strings.Repeat("e", 64)
	otherHandle = testHandle(other)
	three, err := NormalizeDelta(other, otherHandle, change, testLimits)
	if err != nil {
		t.Fatal(err)
	}
	if two.Digest == three.Digest {
		t.Fatal("delta digest did not bind the WIP overlay digest")
	}
	h.TaskID = "manager:other"
	if _, err := NormalizeDelta(r, h, change, testLimits); err == nil {
		t.Fatal("mismatched handle task identity was accepted")
	}
}

func TestDeltaEnforcesEmptyOwnershipScopeAndPreciseBoundaries(t *testing.T) {
	r := testRequest(t)
	h := testHandle(r)
	for _, path := range []string{"src2/out.go", "src/generated/file.go", "outside.md", ".git/config", ".markitect/project.yaml"} {
		t.Run(path, func(t *testing.T) {
			_, err := NormalizeDelta(r, h, []Change{{Kind: ChangeAdd, Path: path, Mode: "100644", Content: []byte("x")}}, testLimits)
			if err == nil {
				t.Fatalf("unauthorized path %q was accepted", path)
			}
		})
	}
	empty := r
	empty.AllowedPaths = nil
	emptyHandle := testHandle(empty)
	if _, err := NormalizeDelta(empty, emptyHandle, []Change{{Kind: ChangeDelete, Path: "src/file.go"}}, testLimits); err == nil {
		t.Fatal("an empty ownership scope was treated as unrestricted")
	}
	if _, err := NormalizeDelta(r, h, []Change{{Kind: ChangeRename, OldPath: "src/generated/old.go", Path: "src/new.go", Mode: "100644", Content: []byte("x")}}, testLimits); err == nil {
		t.Fatal("rename from excluded scope was accepted")
	}
}

func TestDeltaRejectsUnsafePathsAliasesModesAndAmbiguousOperations(t *testing.T) {
	r := testRequest(t)
	h := testHandle(r)
	cases := []struct {
		name    string
		changes []Change
	}{
		{"traversal", []Change{{Kind: ChangeAdd, Path: "src/../escape", Mode: "100644", Content: []byte("x")}}},
		{"backslash", []Change{{Kind: ChangeAdd, Path: `src\escape`, Mode: "100644", Content: []byte("x")}}},
		{"windows invalid character", []Change{{Kind: ChangeAdd, Path: "src/name:stream", Mode: "100644", Content: []byte("x")}}},
		{"windows device name", []Change{{Kind: ChangeAdd, Path: "src/CON.txt", Mode: "100644", Content: []byte("x")}}},
		{"invalid utf8", []Change{{Kind: ChangeAdd, Path: string([]byte{'s', 'r', 'c', '/', 0xff}), Mode: "100644", Content: []byte("x")}}},
		{"nested git metadata", []Change{{Kind: ChangeAdd, Path: "src/.GIT/config", Mode: "100644", Content: []byte("x")}}},
		{"git metadata short name", []Change{{Kind: ChangeAdd, Path: "src/GIT~1/config", Mode: "100644", Content: []byte("x")}}},
		{"unsupported mode", []Change{{Kind: ChangeAdd, Path: "src/file.go", Mode: "1007555", Content: []byte("x")}}},
		{"delete carries content", []Change{{Kind: ChangeDelete, Path: "src/file.go", Content: []byte{}}}},
		{"rename lacks old path", []Change{{Kind: ChangeRename, Path: "src/file.go", Mode: "100644", Content: []byte("x")}}},
		{"same path repeated", []Change{{Kind: ChangeAdd, Path: "src/file.go", Mode: "100644", Content: []byte("x")}, {Kind: ChangeDelete, Path: "src/file.go"}}},
		{"case alias", []Change{{Kind: ChangeAdd, Path: "src/File.go", Mode: "100644", Content: []byte("x")}, {Kind: ChangeAdd, Path: "src/file.go", Mode: "100644", Content: []byte("y")}}},
		{"file directory collision", []Change{{Kind: ChangeAdd, Path: "src/node", Mode: "100644", Content: []byte("x")}, {Kind: ChangeAdd, Path: "src/node/child", Mode: "100644", Content: []byte("y")}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NormalizeDelta(r, h, tc.changes, testLimits); err == nil {
				t.Fatal("invalid delta was accepted")
			}
		})
	}
	if _, err := NormalizeDelta(r, h, []Change{{Kind: ChangeAdd, Path: "src/notes~1/file.go", Mode: "100644", Content: []byte("x")}}, testLimits); err != nil {
		t.Fatalf("an ordinary name with ~1 was refused: %v", err)
	}
}

func TestPathCollisionDiagnosticIsDeterministic(t *testing.T) {
	r := testRequest(t)
	h := testHandle(r)
	changes := []Change{
		{Kind: ChangeAdd, Path: "src/a", Mode: "100644", Content: []byte("a")},
		{Kind: ChangeAdd, Path: "src/a/x/y", Mode: "100644", Content: []byte("b")},
		{Kind: ChangeAdd, Path: "src/a/x", Mode: "100644", Content: []byte("c")},
	}
	var first string
	for i := 0; i < 32; i++ {
		_, err := NormalizeDelta(r, h, changes, testLimits)
		if err == nil {
			t.Fatal("file/directory collision was accepted")
		}
		if i == 0 {
			first = err.Error()
		} else if err.Error() != first {
			t.Fatalf("collision diagnostic varied: %q != %q", err, first)
		}
	}
}

func TestDeltaAllowsExplicitFileDirectoryReplacements(t *testing.T) {
	r := testRequest(t)
	h := testHandle(r)
	cases := []struct {
		name    string
		changes []Change
	}{
		{
			name: "delete file then add descendant",
			changes: []Change{
				{Kind: ChangeDelete, Path: "src/node"},
				{Kind: ChangeAdd, Path: "src/node/child.go", Mode: "100644", Content: []byte("child")},
			},
		},
		{
			name: "delete descendants then add file",
			changes: []Change{
				{Kind: ChangeDelete, Path: "src/node/child.go"},
				{Kind: ChangeAdd, Path: "src/node", Mode: "100644", Content: []byte("file")},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NormalizeDelta(r, h, tc.changes, testLimits); err != nil {
				t.Fatalf("explicit file/directory replacement was rejected: %v", err)
			}
		})
	}
}

func TestRequestRejectsCaseAliasedScopesAndRequiresFullBindings(t *testing.T) {
	r := testRequest(t)
	r.AllowedPaths = []string{"src/", "SRC"}
	if err := r.Validate(); err == nil {
		t.Fatal("case-aliased ownership scopes were accepted")
	}
	r = testRequest(t)
	r.BaseSHA = "abc123"
	if err := r.Validate(); err == nil {
		t.Fatal("short base revision was accepted as immutable snapshot identity")
	}
}

func TestRequestAndHandleRejectInvalidUTF8Bindings(t *testing.T) {
	r := testRequest(t)
	bad := string([]byte{0xff})
	for name, mutate := range map[string]func(*Request){
		"repository root":     func(r *Request) { r.RepositoryRoot += bad },
		"repository identity": func(r *Request) { r.RepositoryIdentity += bad },
		"task identity":       func(r *Request) { r.TaskID += bad },
	} {
		t.Run(name, func(t *testing.T) {
			value := r
			mutate(&value)
			if err := value.Validate(); err == nil {
				t.Fatal("invalid UTF-8 request binding was accepted")
			}
		})
	}

	h := testHandle(r)
	h.ID += bad
	if err := h.ValidateFor(r); err == nil {
		t.Fatal("invalid UTF-8 handle ID was accepted")
	}
	h = testHandle(r)
	h.CWD += bad
	if err := h.ValidateFor(r); err == nil {
		t.Fatal("invalid UTF-8 handle working directory was accepted")
	}
}

func TestDeltaEnforcesPositiveFileAndByteBounds(t *testing.T) {
	r := testRequest(t)
	h := testHandle(r)
	limits := Limits{MaxFiles: 1, MaxFileBytes: 2, MaxTotalBytes: 3}
	if _, err := NormalizeDelta(r, h, []Change{{Kind: ChangeAdd, Path: "src/large", Mode: "100644", Content: []byte("123")}}, limits); err == nil {
		t.Fatal("oversized individual file was accepted")
	}
	if _, err := NormalizeDelta(r, h, []Change{
		{Kind: ChangeAdd, Path: "src/a", Mode: "100644", Content: []byte("12")},
		{Kind: ChangeAdd, Path: "src/b", Mode: "100644", Content: []byte("3")},
	}, Limits{MaxFiles: 2, MaxFileBytes: 2, MaxTotalBytes: 2}); err == nil {
		t.Fatal("oversized total content was accepted")
	}
	if _, err := NormalizeDelta(r, h, []Change{{Kind: ChangeDelete, Path: "src/a"}, {Kind: ChangeDelete, Path: "src/b"}}, limits); err == nil {
		t.Fatal("too many operations were accepted")
	}
	if _, err := NormalizeDelta(r, h, nil, Limits{}); err == nil {
		t.Fatal("zero limits were treated as unlimited")
	}
}
