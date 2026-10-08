package snapshot

import (
	"reflect"
	"testing"
)

func TestDigestMatchesLegacyFingerprintAndIgnoresIdentity(t *testing.T) {
	base := &Snapshot{ID: "external-a", Files: map[string][]byte{"a.txt": []byte("hello\n")}, Modes: map[string]string{"a.txt": RegularMode}}
	const legacyFingerprint = "67c1aeeead453d256f159e71556610720496981e665c5408ebdd6c5eeb879007"
	if got := base.Digest(); got != legacyFingerprint {
		t.Fatalf("Digest() = %q, want legacy fingerprint %q", got, legacyFingerprint)
	}
	otherIdentity := &Snapshot{ID: "external-b", Provisional: true, Files: base.Files, Modes: base.Modes}
	if got := otherIdentity.Digest(); got != legacyFingerprint {
		t.Fatalf("Digest() changed with external ID or provisional status: %q", got)
	}
}

func TestCompareClassifiesAndSortsChanges(t *testing.T) {
	before := &Snapshot{
		Files: map[string][]byte{"mode.txt": []byte("same"), "modify.txt": []byte("old"), "remove.txt": []byte("gone")},
		Modes: map[string]string{"mode.txt": RegularMode, "modify.txt": RegularMode, "remove.txt": RegularMode},
	}
	after := &Snapshot{
		Files: map[string][]byte{"add.txt": []byte("new"), "mode.txt": []byte("same"), "modify.txt": []byte("new")},
		Modes: map[string]string{"add.txt": RegularMode, "mode.txt": ExecutableMode, "modify.txt": RegularMode},
	}
	want := ChangeSet{Added: []string{"add.txt"}, Modified: []string{"mode.txt", "modify.txt"}, Removed: []string{"remove.txt"}}
	got := Compare(before, after)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Compare() = %#v, want %#v", got, want)
	}
	if paths, wantPaths := got.Paths(), []string{"add.txt", "mode.txt", "modify.txt", "remove.txt"}; !reflect.DeepEqual(paths, wantPaths) {
		t.Fatalf("Paths() = %v, want %v", paths, wantPaths)
	}
}

func TestCompareTreatsUnknownAndNewFilesConservatively(t *testing.T) {
	before := &Snapshot{Files: map[string][]byte{"mode-unknown.txt": []byte("same")}, Modes: map[string]string{}}
	after := &Snapshot{Files: map[string][]byte{"mode-unknown.txt": []byte("same"), "new.txt": []byte("new")}, Modes: map[string]string{"mode-unknown.txt": RegularMode, "new.txt": "100600"}}
	want := ChangeSet{Added: []string{"new.txt"}, Modified: []string{"mode-unknown.txt"}}
	if got := Compare(before, after); !reflect.DeepEqual(got, want) {
		t.Fatalf("Compare() = %#v, want %#v", got, want)
	}
}
