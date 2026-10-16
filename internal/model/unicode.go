package model

import (
	"sort"
	"unicode/utf8"
)

type ucdRange struct{ lo, hi rune }

// nameKey returns the Unicode 15.1 canonical caseless key (D145). Invalid
// UTF-8 has no name key; callers must reject it before comparing identities.
func nameKey(s string) string {
	if !utf8.ValidString(s) {
		return ""
	}
	first := normalizeNFD(s)
	var folded []rune
	for _, r := range first {
		if replacement, ok := ucdCaseFold[r]; ok {
			folded = append(folded, []rune(replacement)...)
		} else {
			folded = append(folded, r)
		}
	}
	return normalizeNFD(string(folded))
}

// isNFC reports whether s is valid UTF-8 already in Unicode 15.1 NFC.
func isNFC(s string) bool {
	return utf8.ValidString(s) && normalizeNFC(s) == s
}

func normalizeNFD(s string) string {
	decomposed := make([]rune, 0, len(s))
	for _, r := range s {
		appendCanonicalDecomposition(&decomposed, r)
	}
	canonicalOrder(decomposed)
	return string(decomposed)
}

func appendCanonicalDecomposition(dst *[]rune, r rune) {
	// Hangul syllables have algorithmic canonical decomposition and are not
	// listed in UnicodeData.txt's decomposition field.
	const (
		sBase  = 0xAC00
		lBase  = 0x1100
		vBase  = 0x1161
		tBase  = 0x11A7
		lCount = 19
		vCount = 21
		tCount = 28
		nCount = vCount * tCount
		sCount = lCount * nCount
	)
	if r >= sBase && r < sBase+sCount {
		index := int(r - sBase)
		*dst = append(*dst, rune(lBase+index/nCount), rune(vBase+(index%nCount)/tCount))
		if trailing := index % tCount; trailing != 0 {
			*dst = append(*dst, rune(tBase+trailing))
		}
		return
	}
	if decomposition, ok := ucdDecomposition[r]; ok {
		for _, part := range decomposition {
			appendCanonicalDecomposition(dst, part)
		}
		return
	}
	*dst = append(*dst, r)
}

func canonicalOrder(runes []rune) {
	// Sort each contiguous non-starter run. Stable sorting preserves the order
	// of equal combining classes and avoids quadratic insertion on long runs.
	for start := 0; start < len(runes); {
		marks := start
		if ucdCombiningClass[runes[start]] == 0 {
			marks++
		}
		end := marks
		for end < len(runes) && ucdCombiningClass[runes[end]] != 0 {
			end++
		}
		if end-marks > 1 {
			sort.SliceStable(runes[marks:end], func(i, j int) bool {
				return ucdCombiningClass[runes[marks+i]] < ucdCombiningClass[runes[marks+j]]
			})
		}
		if end == start { // A starter with no following marks.
			start++
		} else {
			start = end
		}
	}
}

func normalizeNFC(s string) string {
	decomposed := []rune(normalizeNFD(s))
	if len(decomposed) < 2 {
		return string(decomposed)
	}
	composed := make([]rune, 0, len(decomposed))
	composed = append(composed, decomposed[0])
	starterPos := 0
	starter := decomposed[0]
	lastClass := ucdCombiningClass[starter]
	if lastClass != 0 {
		starterPos = -1
	}
	for _, r := range decomposed[1:] {
		class := ucdCombiningClass[r]
		if starterPos >= 0 && (lastClass == 0 || lastClass < class) {
			if composite, ok := composeCanonical(starter, r); ok {
				composed[starterPos] = composite
				starter = composite
				continue
			}
		}
		if class == 0 {
			starterPos = len(composed)
			starter = r
		}
		composed = append(composed, r)
		lastClass = class
	}
	return string(composed)
}

func composeCanonical(first, second rune) (rune, bool) {
	// Hangul L+V and LV+T composition is algorithmic.
	const (
		sBase  = 0xAC00
		lBase  = 0x1100
		vBase  = 0x1161
		tBase  = 0x11A7
		lCount = 19
		vCount = 21
		tCount = 28
		nCount = vCount * tCount
		sCount = lCount * nCount
	)
	if first >= lBase && first < lBase+lCount && second >= vBase && second < vBase+vCount {
		lIndex, vIndex := first-lBase, second-vBase
		return rune(sBase + (lIndex*vCount+vIndex)*tCount), true
	}
	if first >= sBase && first < sBase+sCount && (first-sBase)%tCount == 0 && second > tBase && second < tBase+tCount {
		return first + (second - tBase), true
	}
	composite, ok := ucdComposition[(uint64(first)<<21)|uint64(second)]
	return composite, ok
}

func isXIDStart(r rune) bool     { return inUCDRanges(ucdXIDStart, r) }
func isXIDContinue(r rune) bool  { return inUCDRanges(ucdXIDContinue, r) }
func isConnector(r rune) bool    { return inUCDRanges(ucdConnectors, r) }
func isMark(r rune) bool         { return inUCDRanges(ucdMarks, r) }
func isDecimalDigit(r rune) bool { return inUCDRanges(ucdDecimalDigits, r) }
func isLowercase(r rune) bool    { return inUCDRanges(ucdLowercase, r) }
func isUppercase(r rune) bool    { return inUCDRanges(ucdUppercase, r) }
func isTitlecase(r rune) bool    { return inUCDRanges(ucdTitlecase, r) }

func inUCDRanges(ranges []ucdRange, r rune) bool {
	i := sort.Search(len(ranges), func(i int) bool { return ranges[i].hi >= r })
	return i < len(ranges) && ranges[i].lo <= r
}

// nameWord validates a single name word using only the pinned Unicode tables.
// style is lower, pascal, or camel. digitStart applies only to the first rune.
func nameWord(s, style string, digitStart bool) bool {
	if s == "" || !isNFC(s) {
		return false
	}
	runes := []rune(s)
	if len(runes) == 0 || !isNameStart(runes[0], digitStart) {
		return false
	}
	for _, r := range runes[1:] {
		if !isXIDContinue(r) || isConnector(r) {
			return false
		}
	}
	if style != "lower" && style != "pascal" && style != "camel" {
		return false
	}
	var firstCased rune
	previousUpper := false
	for _, r := range runes {
		if isMark(r) {
			continue
		}
		lower := isLowercase(r)
		upper := isUppercase(r) || isTitlecase(r)
		if style == "lower" && upper {
			return false
		}
		if lower || upper {
			if firstCased == 0 {
				firstCased = r
			}
			if style != "lower" && upper && previousUpper {
				return false
			}
			previousUpper = upper
			continue
		}
		previousUpper = false
	}
	if firstCased == 0 {
		return true
	}
	if style == "lower" {
		return true
	}
	firstLower := isLowercase(firstCased)
	firstUpper := isUppercase(firstCased) || isTitlecase(firstCased)
	if style == "pascal" {
		return firstUpper
	}
	return firstLower
}

func isNameStart(r rune, digitStart bool) bool {
	if isXIDStart(r) && !isConnector(r) {
		return true
	}
	return digitStart && isDecimalDigit(r)
}

func validPackageName(s string) bool {
	if s == "" || !isNFC(s) {
		return false
	}
	for _, component := range splitOn(s, '/') {
		if !nameWord(component, "lower", false) {
			return false
		}
	}
	return true
}

func validFilename(s string) bool {
	const suffix = ".yaml"
	if !isNFC(s) || len(s) <= len(suffix) || s[len(s)-len(suffix):] != suffix {
		return false
	}
	stem := s[:len(s)-len(suffix)]
	parts := splitOn(stem, '-')
	for i, part := range parts {
		if !nameWord(part, "lower", i > 0) {
			return false
		}
	}
	return true
}

func validLocalName(s string) bool {
	if nameWord(s, "pascal", false) || nameWord(s, "camel", false) {
		return true
	}
	parts := splitOn(s, '-')
	for i, part := range parts {
		if !nameWord(part, "lower", i > 0) {
			return false
		}
	}
	return s != "" && isNFC(s)
}

func validAlias(s string) bool     { return nameWord(s, "pascal", false) }
func validFieldName(s string) bool { return nameWord(s, "camel", false) }

func validModelLabel(s string, version bool) bool {
	if s == "" || !isNFC(s) || (version && s == "latest") {
		return false
	}
	// §8.1 uses the broader wording “characters with case,” so unlike §7.2
	// name-word initial rules this includes Uppercase/Titlecase Nl characters.
	for _, r := range s {
		if isUppercase(r) || isTitlecase(r) {
			return false
		}
	}
	start := 0
	for i, r := range s {
		if r == '.' || r == '-' {
			if !nameWord(s[start:i], "lower", true) {
				return false
			}
			start = i + 1
		}
	}
	return nameWord(s[start:], "lower", true)
}

func splitOn(s string, separator rune) []string {
	parts := make([]string, 0, 1)
	start := 0
	for i, r := range s {
		if r == separator {
			parts = append(parts, s[start:i])
			start = i + len(string(r))
		}
	}
	return append(parts, s[start:])
}
