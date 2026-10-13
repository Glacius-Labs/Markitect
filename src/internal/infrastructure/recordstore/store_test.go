package recordstore

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestImmutableCollisionAndExactPreimage(t *testing.T) {
	s, _ := openFixture(t)
	n := fixtureName()
	if err := s.Create(n, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := s.Create(n, []byte("second")); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("collision: %v", err)
	}
	before, err := s.Read(n, 10)
	if err != nil || string(before.Bytes) != "first" {
		t.Fatalf("read %q: %v", before.Bytes, err)
	}
	if err = s.Replace(n, before, []byte("next"), nil); err != nil {
		t.Fatal(err)
	}
	if err = s.Replace(n, before, []byte("stale"), nil); err == nil {
		t.Fatal("stale byte CAS accepted")
	}
	after, _ := s.Read(n, 10)
	if string(after.Bytes) != "next" {
		t.Fatalf("unexpected bytes %q", after.Bytes)
	}
}
func TestReadListAbsentNeverCreate(t *testing.T) {
	root := t.TempDir()
	s, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	f, err := s.Read(fixtureName(), 10)
	if err != nil || f.Exists {
		t.Fatal(f, err)
	}
	entries, err := s.List(Run, "", "")
	if err != nil || len(entries) != 0 {
		t.Fatal(entries, err)
	}
	files, err := os.ReadDir(root)
	if err != nil || len(files) != 0 {
		t.Fatalf("reader created state: %v %v", files, err)
	}
}
func TestCutoverTransportOnlyAndMixedAuthority(t *testing.T) {
	s, root := openFixture(t)
	id := fixtureName().ID
	packet := "fedcba9876543210fedcba9876543210"
	input := filepath.ToSlash(filepath.Join(".markitect", "runs", id, "external-response-"+packet+".json"))
	wire := []byte("transport only")
	writeFixture(t, root, input, wire)
	if err := s.CheckCutover(Run); err != nil {
		t.Fatal(err)
	}
	if err := s.Create(fixtureName(), []byte("canonical")); err != nil {
		t.Fatal(err)
	}
	actual, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(input)))
	if !bytes.Equal(actual, wire) {
		t.Fatal("input changed")
	}
	writeFixture(t, root, filepath.ToSlash(filepath.Join(".markitect", "runs", id, "plan.json")), []byte("old authority"))
	if err := s.CheckCutover(Run); err == nil {
		t.Fatal("mixed old authority was hidden by transport")
	}
	if err := s.Create(Name{Kind: Run, ID: packet, Role: "plan.json"}, nil); err == nil {
		t.Fatal("old authority did not block new write")
	}
}
func TestCutoverRejectsEmptyOldIDAndEveryOtherKind(t *testing.T) {
	for _, kind := range []Kind{Run, Exploration, Briefing, Brownfield} {
		t.Run(string(kind), func(t *testing.T) {
			s, root := openFixture(t)
			var old string
			switch kind {
			case Run:
				old = ".markitect/runs/" + fixtureName().ID
			case Exploration:
				old = ".markitect/state/explorations/old.json"
			case Briefing:
				old = ".markitect/state/briefings/history.json"
			case Brownfield:
				old = ".markitect/drafts/brownfield/old/session.json"
			}
			if kind == Run {
				if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(old)), 0700); err != nil {
					t.Fatal(err)
				}
			} else {
				writeFixture(t, root, old, []byte("preserved"))
			}
			if err := s.CheckCutover(kind); err == nil {
				t.Fatal("legacy authority accepted")
			}
		})
	}
}
func TestClosedNames(t *testing.T) {
	for _, bad := range []Name{{Kind: "other", ID: "x", Role: "record.json"}, {Kind: Run, ID: fixtureName().ID, Role: "../escape.json"}, {Kind: Exploration, ID: "CON", Role: "record.json"}, {Kind: Run, ID: fixtureName().ID, Role: "private/log.json"}, {Kind: Brownfield, ID: "session", Role: "unknown.json"}} {
		if _, err := bad.Path(); err == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
}
