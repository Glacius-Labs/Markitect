package spec

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// The form of a field's value (DEC-117). A Kind declares it on a field:
// type enum with its values, type measure, items for each entry of a list,
// and unique. The value stays prose in the text; these read it.

// enumValues reads the values of a field of type enum: a list of words. A
// value holds none of the marks that end or wrap the word in the text, since
// EnumValue could never read it.
func enumValues(n *yaml.Node, bad func(*yaml.Node, string, ...any)) []string {
	if n.Kind != yaml.SequenceNode || len(n.Content) == 0 {
		bad(n, "values is a list of words, such as [sync, async]")
		return nil
	}
	var out []string
	for _, item := range n.Content {
		v := strings.TrimSpace(item.Value)
		switch {
		case item.Kind != yaml.ScalarNode || v == "" || strings.ContainsAny(v, " \t\n") || contains(out, v):
			bad(item, "each value is one word, named once, such as sync")
			continue
		case strings.ContainsAny(v, enumMarks):
			bad(item, "the value %q holds a mark that ends or wraps a word in the text, so no text could give it; a value holds none of %s", v, enumMarksShown)
			continue
		}
		out = append(out, v)
	}
	return out
}

// itemsDecl reads the form of a list field's entries: links, the Kinds one of
// which each entry links, and code, true when each entry holds a code in
// backticks.
func itemsDecl(n *yaml.Node, bad func(*yaml.Node, string, ...any), kinds func(*yaml.Node, string) []string) *ItemsDecl {
	if n.Kind != yaml.MappingNode || len(n.Content) == 0 {
		bad(n, "items says what each entry holds, such as {links: std/Condition, code: true}")
		return nil
	}
	d := &ItemsDecl{}
	for j := 0; j+1 < len(n.Content); j += 2 {
		k, v := n.Content[j], n.Content[j+1]
		switch k.Value {
		case "links":
			if d.Limited {
				bad(k, "items says links twice")
				continue
			}
			d.Limited = true
			if v.Tag == "!!null" {
				bad(v, "items names the Kinds each entry links, such as links: std/Condition")
				continue
			}
			d.Links = kinds(v, "links")
		case "code":
			code, ok := yamlBool(v)
			if !ok {
				bad(v, "code is true or false: true when each entry holds a code in backticks")
				continue
			}
			d.Code = code
		default:
			bad(k, "items declares only links and code, not %q", k.Value)
		}
	}
	return d
}

// Measure is a value of type measure: a number with its unit and, where the
// text gives them, a comparison and what it is of, as in `p95 < 300 ms`.
type Measure struct {
	Number float64 `json:"number"`
	Unit   string  `json:"unit"`
	Op     string  `json:"op,omitempty"`
	Of     string  `json:"of,omitempty"`
}

var (
	// A number that stands alone, not inside a word, a time or a version, with
	// the word or the percent sign after it.
	measureNumber = regexp.MustCompile(`(^|[^\w.:/,-])(\d+(?:,\d{3})*(?:\.\d+)?)\s*(%|[\p{L}µ°]+)`)
	// The word before a comparison in the compact form, such as p95.
	measureOf = regexp.MustCompile(`([\p{L}][\w.]*)\s*$`)
	// A text that says no measure is promised.
	measureNone = regexp.MustCompile(`(?i)\b(none|not promised|nothing is promised|no measure|not measurable|cannot be measured|not provable)\b`)
	codeSpan    = regexp.MustCompile("`([^`\n]+)`")
	// A link as written, whose label may hold a code in backticks or a number
	// that belongs to no value.
	linkSpan = regexp.MustCompile(`\[\[[^\]\n]*\]\]`)
)

// withoutLinks blanks out the links of a text, so that a code or a number in
// a link's label is not read as the value.
func withoutLinks(text string) string { return linkSpan.ReplaceAllString(text, " ") }

// measureOps are the comparisons a measure may state, written as a sign or in
// words, longest first, with the sign each stands for.
var measureOps = []struct{ text, op string }{
	{"no more than", "<="}, {"no less than", ">="}, {"fewer than", "<"}, {"less than", "<"}, {"more than", ">"},
	{"at most", "<="}, {"at least", ">="}, {"up to", "<="}, {"within", "<="}, {"under", "<"}, {"below", "<"},
	{"above", ">"}, {"over", ">"}, {"<=", "<="}, {">=", ">="}, {"≤", "<="}, {"≥", ">="}, {"<", "<"}, {">", ">"}, {"=", "="},
}

// unitStop are words after a number that are no unit: the number counts what
// follows them, is one of several, or is a name, as in "ISO 25010 only".
var unitStop = map[string]bool{"of": true, "and": true, "or": true, "to": true, "in": true, "at": true, "on": true, "for": true, "by": true, "from": true, "the": true, "a": true, "an": true, "x": true, "only": true}

// ParseMeasure reads a value of type measure: the first number in the text
// that has a unit after it and a comparison before it, or else the first that
// has a unit after it, with, in the compact form `p95 < 300 ms`, what it is
// of. So in "at `100` users, `p95 < 300 ms`" the measure is 300 ms and the
// load stays text. Backticks and links are ignored. It is false when the text
// holds no number with a unit.
func ParseMeasure(text string) (Measure, bool) {
	plain := strings.ReplaceAll(withoutLinks(text), "`", "")
	var first *Measure
	for _, m := range measureNumber.FindAllStringSubmatchIndex(plain, -1) {
		unit := plain[m[6]:m[7]]
		lower := strings.ToLower(unit)
		if unitStop[lower] {
			continue
		}
		if lower == "per" {
			// A rate: the unit is what it is per, such as per minute.
			rest := strings.Fields(plain[m[7]:])
			if len(rest) == 0 {
				continue
			}
			unit += " " + strings.Trim(rest[0], ".,;:")
		}
		number, err := strconv.ParseFloat(strings.ReplaceAll(plain[m[4]:m[5]], ",", ""), 64)
		if err != nil {
			continue
		}
		v := Measure{Number: number, Unit: unit}
		before := strings.TrimRight(plain[:m[4]], " ")
		for _, o := range measureOps {
			if len(before) < len(o.text) || !strings.EqualFold(before[len(before)-len(o.text):], o.text) {
				continue
			}
			head := before[:len(before)-len(o.text)]
			if r := o.text[0]; r >= 'a' && r <= 'z' && head != "" && isWordByte(head[len(head)-1]) {
				continue // a word that only ends like one, such as "hover"
			}
			v.Op = o.op
			if !(o.text[0] >= 'a' && o.text[0] <= 'z') {
				if of := measureOf.FindStringSubmatch(head); of != nil {
					v.Of = of[1]
				}
			}
			break
		}
		if v.Op != "" {
			return v, true
		}
		if first == nil {
			first = &v
		}
	}
	if first != nil {
		return *first, true
	}
	return Measure{}, false
}

func isWordByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_'
}

// SaysNoMeasure says whether a text states that no measure is promised, which
// a field of type measure may say instead of a number.
func SaysNoMeasure(text string) bool { return measureNone.MatchString(text) }

// EnumValue reads a value of type enum: the first word of the text, in
// backticks, emphasis, quotes or plain, when it is one of the values, without
// regard to case.
func EnumValue(text string, values []string) (string, bool) {
	word := strings.TrimLeft(strings.ReplaceAll(text, "`", ""), " \t\n")
	if i := strings.IndexFunc(word, func(r rune) bool { return strings.ContainsRune(enumStop, r) }); i >= 0 {
		word = word[:i]
	}
	word = strings.Trim(word, enumWrap)
	for _, v := range values {
		if strings.EqualFold(word, v) {
			return v, true
		}
	}
	return "", false
}

// enumStop ends the first word of an enum's value, at a space or a mark, the
// dashes among them, and enumWrap is what may stand around it; neither is part
// of a value. enumMarks are those marks and the backtick, which a declared
// value may not hold.
const (
	enumPunct      = ".,;:!?—–"
	enumStop       = " \t\n" + enumPunct
	enumWrap       = "*_\"'()"
	enumMarks      = enumPunct + enumWrap + "`"
	enumMarksShown = ". , ; : ! ? — – * _ \" ' ( ) or a backtick"
)

// Code returns the first code in backticks in a text outside its links, or "".
func Code(text string) string {
	if m := codeSpan.FindStringSubmatch(withoutLinks(text)); m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// EntryCode returns the last code in backticks in an entry of a list outside
// its links, or "": the code that answers the entry, as in "[[c/free]], at
// most `3` loans: `limit_reached`".
func EntryCode(text string) string {
	codes := Codes(text)
	if len(codes) == 0 {
		return ""
	}
	return codes[len(codes)-1]
}

// Codes returns every code in backticks in a text outside its links.
func Codes(text string) []string {
	var out []string
	for _, m := range codeSpan.FindAllStringSubmatch(withoutLinks(text), -1) {
		out = append(out, strings.TrimSpace(m[1]))
	}
	return out
}

// Entries returns the entries of a list field: its list entries, or, for a
// field written on one line, its text as one entry.
func (f Field) Entries() []FieldItem {
	if len(f.Items) > 0 || strings.TrimSpace(f.Text) == "" {
		return f.Items
	}
	return []FieldItem{{Text: f.Text, Line: f.Line, Start: f.Start, End: f.End}}
}

// uniqueValue is what a field says for unique: its first code in backticks,
// or its text without the closing full stop.
func uniqueValue(f Field) string {
	if c := Code(f.Text); c != "" {
		return c
	}
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(f.Text), "."))
}

// FieldValue returns the typed value of a field whose Kind declares a type:
// the value of an enum as a string, or a Measure; nil for another field, or a
// value out of form.
func FieldValue(decl *FieldDecl, f Field) any {
	if decl == nil {
		return nil
	}
	switch decl.Type {
	case TypeEnum:
		if v, ok := EnumValue(f.Text, decl.Values); ok {
			return v
		}
	case TypeMeasure:
		if v, ok := fieldMeasure(f.Text); ok {
			return v
		}
	}
	return nil
}

// paragraphBreak is a blank line, which ends a field's first paragraph.
var paragraphBreak = regexp.MustCompile(`\n[ \t]*\n`)

// firstParagraph is a field's text up to its first blank line: the value. What
// follows it says more and holds no value.
func firstParagraph(text string) string {
	text = strings.TrimSpace(text)
	if loc := paragraphBreak.FindStringIndex(text); loc != nil {
		return text[:loc[0]]
	}
	return text
}

// fieldMeasure reads the measure of a field of type measure from its first
// paragraph. A text that says no measure is promised before its first number
// holds none, as in "none is promised, since the 3 owners decide".
func fieldMeasure(text string) (Measure, bool) {
	p := firstParagraph(text)
	if none := measureNone.FindStringIndex(p); none != nil {
		if digit := strings.IndexAny(p, "0123456789"); digit < 0 || none[0] < digit {
			return Measure{}, false
		}
	}
	return ParseMeasure(p)
}

// ValueProblem is a field value out of the form its Kind declares. Vet says
// that the Kind leaves it to vet, so that it is a note and not an error.
type ValueProblem struct {
	Code, File, Message string
	Line                int
	Vet                 bool
}

// ValueProblems holds every field of the spec's own Definitions to the form
// its Kind declares (DEC-117): a value of type enum is one of its values, a
// value of type measure a number with a unit or says that none is promised,
// each entry of a list with items holds what they say, and a unique field
// differs from the same field of every other Definition of the Kind in the
// Package.
func (s *Spec) ValueProblems() []ValueProblem {
	var out []ValueProblem
	type key struct{ kind, field, pkg, value string }
	first := map[key]string{}
	for _, id := range s.OwnDefinitions() {
		d := s.Definitions[id]
		schema := s.Declared(d)
		if schema == nil {
			continue
		}
		seen := map[string]bool{}
		for _, f := range d.Fields {
			decl := schema.Field(f.Label)
			if decl == nil || !decl.Formed() || seen[f.Label] {
				continue
			}
			seen[f.Label] = true
			add := func(line int, format string, args ...any) {
				out = append(out, ValueProblem{Code: "field.value", File: d.File, Line: line, Message: fmt.Sprintf(format, args...), Vet: decl.Vet})
			}
			switch decl.Type {
			case TypeEnum:
				if _, ok := EnumValue(f.Text, decl.Values); !ok {
					add(f.Line, "%q of a %s is %s, in backticks or plain, as its first word", f.Label, d.Kind, enumChoice(decl.Values))
				}
			case TypeMeasure:
				if p := firstParagraph(f.Text); !SaysNoMeasure(p) {
					if _, ok := fieldMeasure(p); !ok {
						add(f.Line, "%q of a %s is a number with its unit, such as `30 days` or `p95 < 300 ms`, or says that none is promised", f.Label, d.Kind)
					}
				}
			}
			if it := decl.Items; it != nil {
				for i, entry := range f.Entries() {
					var lacks []string
					if it.Limited && !s.entryLinks(d, f, entry, it.Links) {
						lacks = append(lacks, "a link to "+oneOf(it.Links))
					}
					if it.Code && EntryCode(entry.Text) == "" {
						lacks = append(lacks, "a code in backticks")
					}
					if len(lacks) > 0 {
						add(entry.Line, "entry %d of %q lacks %s; each entry of it holds %s", i+1, f.Label, strings.Join(lacks, " and "), itemsForm(it))
					}
				}
			}
			if decl.Unique {
				v := uniqueValue(f)
				if v == "" {
					continue
				}
				k := key{d.Kind, f.Label, d.Package, v}
				if other, ok := first[k]; ok {
					out = append(out, ValueProblem{Code: "field.unique", File: d.File, Line: f.Line, Vet: decl.Vet,
						Message: fmt.Sprintf("%s gives the %q %s, as %s does; two Definitions of %s in %s have the same value, and each needs a different one", id, f.Label, v, other, d.Kind, PackageName(d.Package))})
					continue
				}
				first[k] = id
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out
}

// entryLinks says whether an entry of a field links a Definition of one of
// the Kinds.
func (s *Spec) entryLinks(d *Definition, f Field, entry FieldItem, kinds []string) bool {
	for _, l := range d.Links {
		if l.Kind || l.Target == "" || l.Field != f.Label || l.Start < entry.Start || l.Start >= entry.End {
			continue
		}
		kind := CorePackage
		if !l.Package {
			kind = s.Definitions[l.Target].Kind
		}
		if contains(kinds, kind) {
			return true
		}
	}
	return false
}

func enumChoice(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = "`" + v + "`"
	}
	if len(quoted) == 1 {
		return quoted[0]
	}
	return "one of " + strings.Join(quoted, ", ")
}

func itemsForm(it *ItemsDecl) string {
	var parts []string
	if it.Limited {
		parts = append(parts, "a link to "+oneOf(it.Links))
	}
	if it.Code {
		parts = append(parts, "a code in backticks")
	}
	return strings.Join(parts, " and ")
}
