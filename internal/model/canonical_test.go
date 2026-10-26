package model

import (
	"bytes"
	"strings"
	"testing"
)

func privateCanonical(t *testing.T, input Input) canonicalCandidate {
	t.Helper()
	scopes, graph, bindings := privateBindings(input)
	contracts := checkContracts(scopes, graph, bindings)
	values := checkValues(scopes, graph, contracts)
	bodies := checkBodyLinks(scopes, graph, values, effectiveLimits(input.Limits))
	if bodies.status[0] != Passed {
		t.Fatalf("canonical fixture content incomplete: contracts=%+v values=%+v bodies=%+v", contracts.findings, values.findings, bodies.findings)
	}
	candidate, ok := projectCanonical(scopes[0], contracts, values, bodies)
	if !ok {
		t.Fatal("checked content did not produce a private candidate")
	}
	return candidate
}

func TestCanonicalCoreMatchesNormativeBytesAndSHA(t *testing.T) {
	core := completeCoreSource()
	core.Entries[0].Bytes = bytes.Replace(core.Entries[0].Bytes, []byte("version: '1.0'"), []byte("version: '1.2.0'"), 1)
	candidate := privateCanonical(t, Input{Root: core})
	if len(candidate.bytes) != 4589 || string(candidate.bytes) != canonicalCoreVector || candidate.digest != "sha256:b338a1d686be91495e893f085d85013cc91f8f740bef8725210adc2e276f000a" {
		t.Fatalf("normative Core vector differs (%d bytes, %s): %s", len(candidate.bytes), candidate.digest, candidate.bytes)
	}
}

func TestCanonicalJSONHasExactUTF8AndEscapes(t *testing.T) {
	text := "/<> &\u2028\u2029\x00\x01\b\t\n\f\r\"\\📄"
	got := encodeCanonical(canonicalObject(canonicalMember{"ä", canonicalString(text)}, canonicalMember{"z", canonicalNode{kind: 'b', boolean: true}}, canonicalMember{"\x01", canonicalString("value")}))
	wanted := "{\"\\u0001\":\"value\",\"z\":true,\"ä\":\"/<> &\u2028\u2029\\u0000\\u0001\\b\\t\\n\\f\\r\\\"\\\\📄\"}"
	if string(got) != wanted {
		t.Fatalf("canonical encoding differs: got %q wanted %q", got, wanted)
	}
	var controls strings.Builder
	for char := range 32 {
		controls.WriteByte(byte(char))
	}
	encoded := encodeCanonical(canonicalString(controls.String()))
	if bytes.Contains(encoded, []byte{'\n'}) || !bytes.Contains(encoded, []byte(`\u001f`)) {
		t.Fatalf("control escape range incomplete: %q", encoded)
	}
}

func TestCanonicalExactNumbersAndDeclarationByteOrder(t *testing.T) {
	var previous []byte
	for _, literal := range []string{"42", "42.0", "4.2e1"} {
		candidate := privateCanonical(t, singleValueInput("number", "1", "1", "", "{field: "+literal+"}"))
		if previous != nil && !bytes.Equal(previous, candidate.bytes) {
			t.Fatal("equivalent exact decimals differ")
		}
		previous = candidate.bytes
		if !bytes.Contains(candidate.bytes, []byte(`["number","42","0"]`)) {
			t.Fatal("wrong normalized decimal")
		}
	}
	for _, tt := range []struct{ valueType, literal, encoded string }{
		{"integer", "1" + strings.Repeat("0", 4999) + "1", `["integer","1` + strings.Repeat("0", 4999) + `1"]`},
		{"number", "1e1000000000", `["number","1","1000000000"]`},
		{"number", "-0.000e-999999999999999999999999999999", `["number","0","0"]`},
		{"integer", "-0", `["integer","0"]`},
	} {
		candidate := privateCanonical(t, singleValueInput(tt.valueType, "1", "1", "", "{field: "+tt.literal+"}"))
		if !bytes.Contains(candidate.bytes, []byte(tt.encoded)) {
			t.Fatalf("exact number changed: %s", tt.literal)
		}
	}
	input := singleValueInput("enum", "1", "1", ", values: [\"\\t\", A]", "{field: A}")
	candidate := privateCanonical(t, input)
	if !bytes.Contains(candidate.bytes, []byte(`"values":["list",[["string","A"],["string","\t"]]]`)) {
		t.Fatal("declaration list sorted by raw strings instead of encoded JSON element bytes")
	}
}

func TestCanonicalDeepTypedProjectionUsesExplicitStacks(t *testing.T) {
	value := Value{kind: StringValue, text: "leaf"}
	for range 5000 {
		value = Value{kind: ObjectValue, fields: []Field{{Name: "child", Value: value}}}
	}
	node, ok := canonicalTypedValue(value)
	if !ok {
		t.Fatal("deep finite typed value refused")
	}
	encoded := encodeCanonical(node)
	if bytes.Count(encoded, []byte(`["object",`)) != 5000 || !bytes.Contains(encoded, []byte(`["string","leaf"]`)) {
		t.Fatal("deep finite object projection was truncated")
	}
}
