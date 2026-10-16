package model

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestCanonicalNormalizationAndCaselessNames(t *testing.T) {
	tests := []struct {
		name string
		a, b string
	}{
		{name: "canonical accent", a: "é", b: "e\u0301"},
		{name: "recursive canonical decomposition", a: "Ǻ", b: "A\u030a\u0301"},
		{name: "Hangul", a: "각", b: "\u1100\u1161\u11a8"},
		{name: "full fold sharp s", a: "straße", b: "STRASSE"},
		{name: "full fold capital sharp s", a: "Straße", b: "strasse"},
		{name: "fold canonical-equivalent accents", a: "É", b: "e\u0301"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if nameKey(tt.a) != nameKey(tt.b) {
				t.Fatalf("nameKey(%q) != nameKey(%q): %q != %q", tt.a, tt.b, nameKey(tt.a), nameKey(tt.b))
			}
		})
	}
	if nameKey("ä") == nameKey("ae") {
		t.Fatal("canonical caseless matching must not perform compatibility or transliteration folding")
	}
	if nameKey("İ") == nameKey("i") {
		t.Fatal("default folding must preserve the distinction between dotted capital I and i")
	}
	if nameKey("ı") == nameKey("i") {
		t.Fatal("default folding must preserve the distinction between dotless i and i")
	}
}

func TestNFCRequiresAlreadyNormalizedValidUTF8(t *testing.T) {
	for _, s := range []string{"", "é", "각", "A\u0338"} {
		if !isNFC(s) {
			t.Errorf("isNFC(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"e\u0301", "\u1100\u1161\u11a8", string([]byte{0xff})} {
		if isNFC(s) {
			t.Errorf("isNFC(%q) = true, want false", s)
		}
	}
}

func TestNameWordUnicodeStyles(t *testing.T) {
	cases := []struct {
		name       string
		value      string
		style      string
		digitStart bool
		want       bool
	}{
		{"Pascal acronym shape", "MqttService", "pascal", false, true},
		{"Pascal rejects acronym run", "MQTTService", "pascal", false, false},
		{"camel", "gültigBis", "camel", false, true},
		{"camel rejects uppercase initial", "GültigBis", "camel", false, false},
		{"Pascal titlecase initial", "ǅungla", "pascal", false, true},
		{"Pascal uppercase number initial", "Ⅰ", "pascal", false, true},
		{"camel rejects uppercase number initial", "Ⅰ", "camel", false, false},
		{"camel lowercase number initial", "ⅰ", "camel", false, true},
		{"Pascal rejects lowercase number initial", "ⅰ", "pascal", false, false},
		{"lower rejects uppercase number", "Ⅰ", "lower", false, false},
		{"lower accepts lowercase number", "ⅰ", "lower", false, true},
		{"lower uncased script", "文档", "lower", false, true},
		{"Pascal uncased script", "文档", "pascal", false, true},
		{"camel uncased script", "文档", "camel", false, true},
		{"uncased initial then lowercase", "文档a", "lower", false, true},
		{"NFC combining mark after initial", "q\u0307ui", "lower", false, true},
		{"combining mark does not separate capitals", "A\u0301B", "pascal", false, false},
		{"underscore excluded", "foo_bar", "lower", false, false},
		{"digit start rejected", "2mod", "lower", false, false},
		{"digit start enabled", "٢mod", "lower", true, true},
		{"non-NFC rejected", "u\u0308ber", "lower", false, false},
		{"unknown style rejected", "name", "snake", false, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := nameWord(tt.value, tt.style, tt.digitStart); got != tt.want {
				t.Errorf("nameWord(%q, %q, %t) = %t, want %t", tt.value, tt.style, tt.digitStart, got, tt.want)
			}
		})
	}
}

func TestNameForms(t *testing.T) {
	for _, s := range []string{"markitect/core", "straße", "ⅰ", "文档/überweisungen"} {
		if !validPackageName(s) {
			t.Errorf("validPackageName(%q) = false", s)
		}
	}
	for _, s := range []string{"", "/core", "markitect/", "markitect//core", "markitect/Core", "markitect/2core", "Ⅰ", "foo.bar", "foo\\bar", "foo/../bar", "e\u0301"} {
		if validPackageName(s) {
			t.Errorf("validPackageName(%q) = true", s)
		}
	}

	for _, s := range []string{"exchange.yaml", "max-attempts.yaml", "größe-2.yaml", "文档.yaml"} {
		if !validFilename(s) {
			t.Errorf("validFilename(%q) = false", s)
		}
	}
	for _, s := range []string{".yaml", "2exchange.yaml", "exchange--part.yaml", "Exchange.yaml", "exchange.yml", "exchange.part.yaml", "e\u0301.yaml"} {
		if validFilename(s) {
			t.Errorf("validFilename(%q) = true", s)
		}
	}
	if !validFilename("exchange-2part.yaml") {
		t.Fatal("filename words after a hyphen may begin with a Unicode decimal digit")
	}

	for _, s := range []string{"Kind", "requester", "api", "list-documents", "größe-2", "文档"} {
		if !validLocalName(s) {
			t.Errorf("validLocalName(%q) = false", s)
		}
	}
	for _, s := range []string{"", "foo.bar", "_hidden", "2api", "foo--bar", "AcronymID"} {
		if validLocalName(s) {
			t.Errorf("validLocalName(%q) = true", s)
		}
	}
	if !validAlias("Connector") || validAlias("connector") || !validAlias("文档") ||
		!validFieldName("messageId") || validFieldName("MessageId") || !validFieldName("文档") {
		t.Fatal("alias or effective field name accepted the wrong spelling")
	}
}

func TestModelLabelGrammar(t *testing.T) {
	for _, s := range []string{"markitect-core", "markitect.core", "2026-modell", "٢٠٢٦.١", "文档", "ⅰ", "x", "1.x", "release.x"} {
		if !validModelLabel(s, false) {
			t.Errorf("model ID %q rejected", s)
		}
	}
	for _, s := range []string{"1.0", "1.0.0-rc.1", "2026.10", "ⅰ", "x", "1.x", "release.x"} {
		if !validModelLabel(s, true) {
			t.Errorf("version %q rejected", s)
		}
	}
	for _, s := range []string{"", "Markitect-Core", "Ⅰ", "über_weisungen", "modell/name", "1..0", "1.0.0-", "e\u0301"} {
		if validModelLabel(s, false) {
			t.Errorf("model ID %q accepted", s)
		}
	}
	if !validModelLabel("latest", false) {
		t.Fatal("latest is an ordinary model-ID label")
	}
	for _, s := range []string{"", "Markitect-Core", "Ⅰ", "über_weisungen", "modell/name", "1..0", "1.0.0-", "e\u0301", "latest"} {
		if validModelLabel(s, true) {
			t.Errorf("version %q accepted", s)
		}
	}
}

func TestUnicode151PinnedProperties(t *testing.T) {
	// CJK Unified Ideographs Extension I (U+2EBF0) was added in Unicode 15.1.
	if !isXIDStart('\U0002EBF0') || !isXIDContinue('\U0002EBF0') {
		t.Fatal("Unicode 15.1 CJK Extension I character is missing from identifier properties")
	}
	if isXIDStart('\U00002FFC') || isXIDContinue('\U00002FFC') {
		t.Fatal("Unicode 15.1 ideographic description character was accepted as an identifier")
	}
}

func TestUnicodeCorpusSHA256(t *testing.T) {
	corpora := []struct {
		name     string
		encoded  string
		expected string
	}{
		{"NormalizationTest.txt", unicodeNormalizationTestGzipBase64, "871238e37e3be0696ec2bd0891119a041b052da1a84485eda05a5438724b223e"},
		{"CaseFolding.txt", unicodeCaseFoldingTestGzipBase64, "4e55acfdc32825a22e87670e9056a3bf94ad7c5400065778e9e10f8314372bcf"},
		{"DerivedCoreProperties.txt", unicodeDerivedCorePropertiesTestGzipBase64, "f55d0db69123431a7317868725b1fcbf1eab6b265d756d1bd7f0f6d9f9ee108b"},
		{"UnicodeData.txt", unicodeUnicodeDataTestGzipBase64, "2fc713e6a31a87c4850a37fe2caffa4218180fadb5de86b43a143ddb4581fb86"},
	}
	for _, corpus := range corpora {
		t.Run(corpus.name, func(t *testing.T) {
			digest := sha256.Sum256(decodeUnicodeTestCorpus(corpus.encoded))
			if got := hex.EncodeToString(digest[:]); got != corpus.expected {
				t.Fatalf("embedded corpus SHA-256 = %s, want %s", got, corpus.expected)
			}
		})
	}
}

func TestUnicodeNormalizationTestConformance(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader(string(decodeUnicodeTestCorpus(unicodeNormalizationTestGzipBase64))))
	part := ""
	expectedPartRows := map[string]int{"@Part0": 25, "@Part1": 17029, "@Part2": 1844, "@Part3": 176}
	partRows := map[string]int{}
	part1Scalars := make(map[rune]struct{}, expectedPartRows["@Part1"])
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "@Part") {
			part = strings.Fields(line)[0]
			if _, known := expectedPartRows[part]; !known {
				t.Fatalf("line %d: unexpected NormalizationTest section %s", lineNumber, part)
			}
			continue
		}
		if _, known := expectedPartRows[part]; !known {
			continue
		}
		columns := strings.Split(line, ";")
		if len(columns) < 5 {
			t.Fatalf("line %d: malformed NormalizationTest row", lineNumber)
		}
		var c [5]string
		for i := range c {
			var err error
			c[i], err = parseUnicodeCodePointSequence(columns[i])
			if err != nil {
				t.Fatalf("line %d column %d: %v", lineNumber, i+1, err)
			}
		}
		partRows[part]++
		if part == "@Part1" {
			scalars := []rune(c[0])
			if len(scalars) != 1 {
				t.Fatalf("line %d: @Part1 source is not one scalar", lineNumber)
			}
			part1Scalars[scalars[0]] = struct{}{}
		}
		// NormalizationTest.txt conformance invariants for c1 through c5.
		for _, input := range []string{c[0], c[1], c[2]} {
			if got := normalizeNFC(input); got != c[1] {
				t.Fatalf("line %d %s: NFC(%X) = %X, want %X", lineNumber, part, codePoints(input), codePoints(got), codePoints(c[1]))
			}
			if got := normalizeNFD(input); got != c[2] {
				t.Fatalf("line %d %s: NFD(%X) = %X, want %X", lineNumber, part, codePoints(input), codePoints(got), codePoints(c[2]))
			}
		}
		for _, input := range []string{c[3], c[4]} {
			if got := normalizeNFC(input); got != c[3] {
				t.Fatalf("line %d %s: NFC(%X) = %X, want %X", lineNumber, part, codePoints(input), codePoints(got), codePoints(c[3]))
			}
			if got := normalizeNFD(input); got != c[4] {
				t.Fatalf("line %d %s: NFD(%X) = %X, want %X", lineNumber, part, codePoints(input), codePoints(got), codePoints(c[4]))
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	rows := 0
	for part, expected := range expectedPartRows {
		if got := partRows[part]; got != expected {
			t.Fatalf("normalization corpus section %s has %d rows, want %d", part, got, expected)
		}
		rows += partRows[part]
	}
	if rows != 19074 || len(part1Scalars) != expectedPartRows["@Part1"] {
		t.Fatalf("normalization corpus has %d rows and %d unique @Part1 scalars", rows, len(part1Scalars))
	}
	checkedStable := 0
	for scalar := rune(0); scalar <= 0x10FFFF; scalar++ {
		if scalar >= 0xD800 && scalar <= 0xDFFF {
			continue
		}
		if _, covered := part1Scalars[scalar]; covered {
			continue
		}
		input := string(scalar)
		if got := normalizeNFC(input); got != input {
			t.Fatalf("unlisted scalar U+%04X is not NFC-stable: %X", scalar, codePoints(got))
		}
		if got := normalizeNFD(input); got != input {
			t.Fatalf("unlisted scalar U+%04X is not NFD-stable: %X", scalar, codePoints(got))
		}
		checkedStable++
	}
	if checkedStable != 1095035 {
		t.Fatalf("checked %d scalar stability cases, want 1095035", checkedStable)
	}
	t.Logf("verified all %d normalization rows (Part0=%d, Part1=%d, Part2=%d, Part3=%d), plus %d stable scalars outside Part1", rows, partRows["@Part0"], partRows["@Part1"], partRows["@Part2"], partRows["@Part3"], checkedStable)
}

func TestUnicodeDefaultCaseFoldingCorpus(t *testing.T) {
	want := parseDefaultCaseFolding(t, decodeUnicodeTestCorpus(unicodeCaseFoldingTestGzipBase64))
	if len(want) == 0 {
		t.Fatal("no default case-fold mappings parsed")
	}
	t.Logf("verified %d default C/F case-fold mappings", len(want))
	if len(want) != len(ucdCaseFold) {
		t.Fatalf("case-fold table has %d mappings, corpus has %d", len(ucdCaseFold), len(want))
	}
	for scalar, expected := range want {
		if got := ucdCaseFold[scalar]; got != expected {
			t.Errorf("case fold U+%04X = %X, want %X", scalar, codePoints(got), codePoints(expected))
		}
	}
	for scalar, got := range ucdCaseFold {
		if want[scalar] != got {
			t.Errorf("unexpected case fold U+%04X = %X", scalar, codePoints(got))
		}
	}

	// Check the D145 sequence for every scalar with a default C/F mapping.
	// Each mapping above is independently bound to the official corpus, while
	// NormalizationTest verifies the normalizer used at both ends of the key.
	for scalar := range want {
		input := string(scalar)
		firstNFD := normalizeNFD(input)
		var folded strings.Builder
		for _, r := range firstNFD {
			if replacement, ok := want[r]; ok {
				folded.WriteString(replacement)
			} else {
				folded.WriteRune(r)
			}
		}
		expectedKey := normalizeNFD(folded.String())
		if got := nameKey(input); got != expectedKey {
			t.Errorf("D145 key for U+%04X = %X, want %X", scalar, codePoints(got), codePoints(expectedKey))
		}
	}
}

func TestUnicodeDerivedCorePropertiesConformance(t *testing.T) {
	want := parsePropertyRanges(t, decodeUnicodeTestCorpus(unicodeDerivedCorePropertiesTestGzipBase64), map[string]bool{
		"XID_Start": true, "XID_Continue": true, "Lowercase": true, "Uppercase": true,
	})
	testUCDRangeProperty(t, "XID_Start", ucdXIDStart, want["XID_Start"])
	testUCDRangeProperty(t, "XID_Continue", ucdXIDContinue, want["XID_Continue"])
	testUCDRangeProperty(t, "Lowercase", ucdLowercase, want["Lowercase"])
	testUCDRangeProperty(t, "Uppercase", ucdUppercase, want["Uppercase"])
}

func TestUnicodeDataCategoryConformance(t *testing.T) {
	want := parseUnicodeDataCategories(t, decodeUnicodeTestCorpus(unicodeUnicodeDataTestGzipBase64))
	testUCDRangeProperty(t, "Marks", ucdMarks, want["Marks"])
	testUCDRangeProperty(t, "DecimalDigits", ucdDecimalDigits, want["DecimalDigits"])
	testUCDRangeProperty(t, "Connectors", ucdConnectors, want["Connectors"])
	testUCDRangeProperty(t, "Letters", ucdLetters, want["Letters"])
	testUCDRangeProperty(t, "Titlecase", ucdTitlecase, want["Titlecase"])
}

func parseDefaultCaseFolding(t *testing.T, data []byte) map[rune]string {
	t.Helper()
	wanted := make(map[rune]string)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		if line == "" {
			continue
		}
		fields := strings.Split(line, ";")
		if len(fields) < 3 {
			t.Fatalf("CaseFolding line %d malformed", lineNumber)
		}
		status := strings.TrimSpace(fields[1])
		if status != "C" && status != "F" { // S and T must not override default folding.
			continue
		}
		input, err := strconv.ParseInt(strings.TrimSpace(fields[0]), 16, 32)
		if err != nil {
			t.Fatalf("CaseFolding line %d code point: %v", lineNumber, err)
		}
		output, err := parseUnicodeCodePointSequence(fields[2])
		if err != nil {
			t.Fatalf("CaseFolding line %d mapping: %v", lineNumber, err)
		}
		if status == "F" || wanted[rune(input)] == "" {
			wanted[rune(input)] = output
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return wanted
}

func parsePropertyRanges(t *testing.T, data []byte, selected map[string]bool) map[string][]ucdRange {
	t.Helper()
	wanted := make(map[string][]ucdRange, len(selected))
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		if line == "" {
			continue
		}
		fields := strings.Split(line, ";")
		if len(fields) < 2 {
			t.Fatalf("DerivedCoreProperties line %d malformed", lineNumber)
		}
		property := strings.TrimSpace(fields[1])
		if !selected[property] {
			continue
		}
		lo, hi, err := parseUnicodeRange(fields[0])
		if err != nil {
			t.Fatalf("DerivedCoreProperties line %d range: %v", lineNumber, err)
		}
		wanted[property] = append(wanted[property], ucdRange{lo: lo, hi: hi})
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return wanted
}

func parseUnicodeDataCategories(t *testing.T, data []byte) map[string][]ucdRange {
	t.Helper()
	wanted := map[string][]ucdRange{"Marks": {}, "DecimalDigits": {}, "Connectors": {}, "Letters": {}, "Titlecase": {}}
	type pendingRange struct {
		start rune
		cat   string
	}
	var pending *pendingRange
	add := func(category string, lo, hi rune) {
		if strings.HasPrefix(category, "M") {
			wanted["Marks"] = append(wanted["Marks"], ucdRange{lo, hi})
		}
		if category == "Nd" {
			wanted["DecimalDigits"] = append(wanted["DecimalDigits"], ucdRange{lo, hi})
		}
		if category == "Pc" {
			wanted["Connectors"] = append(wanted["Connectors"], ucdRange{lo, hi})
		}
		if strings.HasPrefix(category, "L") {
			wanted["Letters"] = append(wanted["Letters"], ucdRange{lo, hi})
		}
		if category == "Lt" {
			wanted["Titlecase"] = append(wanted["Titlecase"], ucdRange{lo, hi})
		}
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		fields := strings.Split(scanner.Text(), ";")
		if len(fields) < 3 {
			t.Fatalf("UnicodeData line %d malformed", lineNumber)
		}
		raw, err := strconv.ParseInt(fields[0], 16, 32)
		if err != nil {
			t.Fatalf("UnicodeData line %d code point: %v", lineNumber, err)
		}
		name, category := fields[1], fields[2]
		if strings.HasSuffix(name, ", First>") {
			if pending != nil {
				t.Fatalf("UnicodeData line %d begins a range before closing prior range", lineNumber)
			}
			pending = &pendingRange{start: rune(raw), cat: category}
			continue
		}
		if strings.HasSuffix(name, ", Last>") {
			if pending == nil || pending.cat != category {
				t.Fatalf("UnicodeData line %d has unmatched range end", lineNumber)
			}
			add(category, pending.start, rune(raw))
			pending = nil
			continue
		}
		if pending != nil {
			t.Fatalf("UnicodeData line %d interrupts a First/Last range", lineNumber)
		}
		add(category, rune(raw), rune(raw))
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if pending != nil {
		t.Fatal("UnicodeData corpus ended with an unclosed First range")
	}
	return wanted
}

func testUCDRangeProperty(t *testing.T, name string, got, want []ucdRange) {
	t.Helper()
	count := 0
	for _, r := range want {
		count += int(r.hi-r.lo) + 1
	}
	for scalar := rune(0); scalar <= 0x10FFFF; scalar++ {
		if scalar >= 0xD800 && scalar <= 0xDFFF {
			continue
		}
		wantHas := rangeContains(want, scalar)
		if gotHas := inUCDRanges(got, scalar); gotHas != wantHas {
			t.Fatalf("%s U+%04X = %t, corpus says %t", name, scalar, gotHas, wantHas)
		}
	}
	t.Logf("verified %s over %d ranges and %d scalar values", name, len(want), count)
}

func rangeContains(ranges []ucdRange, scalar rune) bool {
	index := sort.Search(len(ranges), func(i int) bool { return ranges[i].hi >= scalar })
	return index < len(ranges) && ranges[index].lo <= scalar
}

func parseUnicodeRange(raw string) (rune, rune, error) {
	parts := strings.Split(strings.TrimSpace(raw), "..")
	lo, err := strconv.ParseInt(parts[0], 16, 32)
	if err != nil {
		return 0, 0, err
	}
	hi := lo
	if len(parts) == 2 {
		hi, err = strconv.ParseInt(parts[1], 16, 32)
		if err != nil {
			return 0, 0, err
		}
	} else if len(parts) != 1 {
		return 0, 0, fmt.Errorf("invalid range %q", raw)
	}
	return rune(lo), rune(hi), nil
}

func parseUnicodeCodePointSequence(raw string) (string, error) {
	fields := strings.Fields(raw)
	runes := make([]rune, 0, len(fields))
	for _, field := range fields {
		value, err := strconv.ParseInt(field, 16, 32)
		if err != nil || value < 0 || value > 0x10FFFF || (value >= 0xD800 && value <= 0xDFFF) {
			return "", fmt.Errorf("invalid scalar %q", field)
		}
		runes = append(runes, rune(value))
	}
	return string(runes), nil
}

func codePoints(s string) []string {
	result := make([]string, 0, len([]rune(s)))
	for _, r := range s {
		result = append(result, fmt.Sprintf("%04X", r))
	}
	return result
}
