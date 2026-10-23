package model_test

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

func TestNameCollisionKey(t *testing.T) {
	for _, tt := range []struct {
		name, input, want string
	}{
		{"full fold", "straße", "strasse"},
		{"case and full fold", "Straße", "strasse"},
		{"case", "Model", "model"},
		{"NFC", "É", "e\u0301"},
		{"NFD", "E\u0301", "e\u0301"},
		{"uncased", "文档", "文档"},
		{"empty is valid UTF-8", "", ""},
		{"no directory grammar check", "../Root Name", "../root name"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			original := tt.input
			key, ok := model.NameCollisionKey(tt.input)
			if !ok || key != tt.want {
				t.Fatalf("NameCollisionKey(%q) = (%q, %t), want (%q, true)", tt.input, key, ok, tt.want)
			}
			if tt.input != original {
				t.Fatal("original name changed")
			}
		})
	}
}

func TestNameCollisionKeyComparison(t *testing.T) {
	for _, tt := range []struct {
		name, a, b string
		collide    bool
	}{
		{"full fold", "straße", "strasse", true},
		{"case", "Model", "model", true},
		{"canonical equivalence", "é", "e\u0301", true},
		{"Hangul decomposition", "각", "\u1100\u1161\u11a8", true},
		{"distinct names", "model", "models", false},
		{"no transliteration", "ä", "ae", false},
		{"default dotted I", "İ", "i", false},
		{"default dotless I", "ı", "i", false},
		{"no compatibility normalization", "ａ", "a", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a, aOK := model.NameCollisionKey(tt.a)
			b, bOK := model.NameCollisionKey(tt.b)
			if !aOK || !bOK || (a == b) != tt.collide {
				t.Fatalf("keys for %q and %q = (%q, %t), (%q, %t); want collision %t", tt.a, tt.b, a, aOK, b, bOK, tt.collide)
			}
		})
	}
}

func TestNameCollisionKeyRejectsInvalidUTF8(t *testing.T) {
	for _, input := range []string{
		"\xff", "model\xff", "\xc0\xaf", "\xe2\x82", "\xed\xa0\x80", "\xf4\x90\x80\x80",
	} {
		if key, ok := model.NameCollisionKey(input); ok || key != "" {
			t.Errorf("NameCollisionKey(%q) = (%q, %t), want (empty, false)", input, key, ok)
		}
	}
	// An actual replacement character is valid UTF-8, unlike invalid bytes.
	if key, ok := model.NameCollisionKey("\ufffd"); !ok || key != "\ufffd" {
		t.Fatalf("NameCollisionKey(replacement character) = (%q, %t)", key, ok)
	}
}
