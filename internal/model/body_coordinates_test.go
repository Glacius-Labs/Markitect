package model

import (
	"strings"
	"testing"
)

func coordinateFixture(t *testing.T, source string) (Entry, Value) {
	t.Helper()
	entry := Entry{Path: "defs/item.yaml", Kind: Regular, Bytes: []byte(source)}
	root, problems, complete := parseYAML(entry.Bytes, 64)
	if !complete || len(problems) != 0 {
		t.Fatalf("fixture YAML: complete=%v problems=%+v", complete, problems)
	}
	body, ok := lookup(root, "body")
	if !ok || body.kind != StringValue {
		t.Fatalf("body value=%+v present=%v", body, ok)
	}
	return entry, body
}

func occurrenceFor(t *testing.T, body Value, needle string, from int) (bodyOccurrence, int) {
	t.Helper()
	start := strings.Index(body.text[from:], needle)
	if start < 0 {
		t.Fatalf("%q not found in body %q", needle, body.text)
	}
	start += from
	return bodyOccurrence{span: BodySpan{StartByte: start, EndByte: start + len(needle)}}, start + len(needle)
}

func expectedYamlPosition(t *testing.T, source string, byteOffset int) Position {
	t.Helper()
	position, ok := yamlBytePosition([]byte(source), byteOffset)
	if !ok {
		t.Fatalf("invalid source byte offset %d", byteOffset)
	}
	return position
}

func assertCoordinate(t *testing.T, span *SourceSpan, source string, start, end int) {
	t.Helper()
	if span == nil {
		t.Fatal("expected a proven source span")
	}
	want := SourceSpan{Start: expectedYamlPosition(t, source, start), End: expectedYamlPosition(t, source, end)}
	if *span != want {
		t.Fatalf("span=%+v want=%+v", *span, want)
	}
}

func TestBodySourceSpansMapQuotedEscapesAndRepeatedLinks(t *testing.T) {
	source := "title: \"café\"\nbody: \"[[api|A\\0B]] then [[api]]\"\n"
	entry, body := coordinateFixture(t, source)
	first, next := occurrenceFor(t, body, "[[api|A\x00B]]", 0)
	second, _ := occurrenceFor(t, body, "[[api]]", next)
	got := bodySourceSpans(entry, body, []bodyOccurrence{first, second})
	if len(got) != 2 {
		t.Fatalf("got %d spans", len(got))
	}
	firstStart := strings.Index(source, "[[api|A\\0B]]")
	firstEnd := firstStart + len("[[api|A\\0B]]")
	assertCoordinate(t, got[0], source, firstStart, firstEnd)
	secondStart := strings.LastIndex(source, "[[api]]")
	assertCoordinate(t, got[1], source, secondStart, secondStart+len("[[api]]"))
	if *got[0] == *got[1] {
		t.Fatal("repeated links mapped to the same source span")
	}
}

func TestBodySourceSpansMapEveryYAMLEscapedNULSpelling(t *testing.T) {
	for _, escape := range []string{"\\0", "\\x00", "\\u0000", "\\U00000000"} {
		t.Run(escape, func(t *testing.T) {
			source := "body: \"[[api|A" + escape + "B]]\"\n"
			entry, body := coordinateFixture(t, source)
			occurrence, _ := occurrenceFor(t, body, "[[api|A\x00B]]", 0)
			got := bodySourceSpans(entry, body, []bodyOccurrence{occurrence})
			start := strings.Index(source, "[[api|A"+escape+"B]]")
			assertCoordinate(t, got[0], source, start, start+len("[[api|A"+escape+"B]]"))
		})
	}
}

func TestBodySourceSpansMapSingleQuotedAndUnicodeEscapes(t *testing.T) {
	source := "body: '[[api|café]] and [[api|owner''s]]'\n"
	entry, body := coordinateFixture(t, source)
	first, next := occurrenceFor(t, body, "[[api|café]]", 0)
	second, _ := occurrenceFor(t, body, "[[api|owner's]]", next)
	got := bodySourceSpans(entry, body, []bodyOccurrence{first, second})
	firstStart := strings.Index(source, "[[api|café]]")
	assertCoordinate(t, got[0], source, firstStart, firstStart+len("[[api|café]]"))
	secondStart := strings.Index(source, "[[api|owner''s]]")
	assertCoordinate(t, got[1], source, secondStart, secondStart+len("[[api|owner''s]]"))
}

func TestBodySourceSpansMapLiteralAndFoldedScalars(t *testing.T) {
	cases := []struct {
		name, source, needle string
	}{
		{"literal", "body: |\n  before\n  [[api]]\n  after\nother: x\n", "[[api]]"},
		{"folded", "body: >-\n  before\n  [[api]]\n  after\nother: x\n", "[[api]]"},
		{"literal-comment-digit", "body: | # indentation 9\n  [[api]]\nother: x\n", "[[api]]"},
		{"folded-comment-digit", "body: >- # indentation 9\r\n  before\r\n  [[api]]\r\nother: x\r\n", "[[api]]"},
		{"explicit-indent-before-chomp", "body: |2- # indentation 9\n  [[api]]\nother: x\n", "[[api]]"},
		{"explicit-indent-after-chomp", "body: >+2 # indentation 9\n  [[api]]\nother: x\n", "[[api]]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry, body := coordinateFixture(t, tc.source)
			occurrence, _ := occurrenceFor(t, body, tc.needle, 0)
			got := bodySourceSpans(entry, body, []bodyOccurrence{occurrence})
			start := strings.Index(tc.source, tc.needle)
			assertCoordinate(t, got[0], tc.source, start, start+len(tc.needle))
		})
	}
}

func TestBodySourceSpansUseDecodedOffsetsWithContainersAndCRLF(t *testing.T) {
	source := "kind: Item\r\nbody: \"prefix\r\n  > [[api|A\\u03A9]]\"\r\ntitle: done\r\n"
	entry, body := coordinateFixture(t, source)
	needle := "[[api|AΩ]]"
	occurrence, _ := occurrenceFor(t, body, needle, 0)
	got := bodySourceSpans(entry, body, []bodyOccurrence{occurrence})
	start := strings.Index(source, "[[api|A\\u03A9]]")
	assertCoordinate(t, got[0], source, start, start+len("[[api|A\\u03A9]]"))
}

func TestBodySourceSpansHandleYAMLColumnsAfterUTF8(t *testing.T) {
	source := "{ title: café, body: \"[[api]]\" }\n"
	entry := Entry{Path: "defs/item.yaml", Kind: Regular, Bytes: []byte(source)}
	root, problems, complete := parseYAML(entry.Bytes, 64)
	if !complete || len(problems) != 0 {
		t.Fatalf("fixture YAML: complete=%v problems=%+v", complete, problems)
	}
	body, _ := lookup(root, "body")
	occurrence, _ := occurrenceFor(t, body, "[[api]]", 0)
	got := bodySourceSpans(entry, body, []bodyOccurrence{occurrence})
	start := strings.Index(source, "[[api]]")
	assertCoordinate(t, got[0], source, start, start+len("[[api]]"))
}

func TestBodySourceSpansRefuseStaleOrUnprovableMetadata(t *testing.T) {
	source := "body: \"[[api]]\"\n"
	entry, body := coordinateFixture(t, source)
	occurrence, _ := occurrenceFor(t, body, "[[api]]", 0)
	stale := body
	stale.text = "[[other]]"
	if got := bodySourceSpans(entry, stale, []bodyOccurrence{occurrence}); len(got) != 1 || got[0] != nil {
		t.Fatalf("stale body mapping=%+v", got)
	}
	withSibling, siblingBody := coordinateFixture(t, "body: \"[[api]]\"\nnote: \"[[api]]\"\n")
	note := lookupValueForTest(t, withSibling, "note")
	stalePosition := body
	stalePosition.position = note.position
	siblingOccurrence, _ := occurrenceFor(t, siblingBody, "[[api]]", 0)
	if got := bodySourceSpans(withSibling, stalePosition, []bodyOccurrence{siblingOccurrence}); len(got) != 1 || got[0] != nil {
		t.Fatalf("same-text sibling metadata was accepted: %+v", got)
	}
	if got := bodySourceSpans(entry, body, []bodyOccurrence{{span: BodySpan{StartByte: -1, EndByte: 2}}}); len(got) != 1 || got[0] != nil {
		t.Fatalf("invalid body byte range mapping=%+v", got)
	}
	flowEntry := Entry{Path: "defs/item.yaml", Kind: Regular, Bytes: []byte("body: { value: text }\n")}
	if got := bodySourceSpans(flowEntry, Value{kind: StringValue, text: "[[api]]", position: Position{Line: 1, Column: 7}}, []bodyOccurrence{occurrence}); len(got) != 1 || got[0] != nil {
		t.Fatalf("non-scalar YAML mapping=%+v", got)
	}
}

func lookupValueForTest(t *testing.T, entry Entry, name string) Value {
	t.Helper()
	root, problems, complete := parseYAML(entry.Bytes, 64)
	if !complete || len(problems) != 0 {
		t.Fatalf("fixture YAML: complete=%v problems=%+v", complete, problems)
	}
	value, ok := lookup(root, name)
	if !ok {
		t.Fatalf("missing fixture key %q", name)
	}
	return value
}
