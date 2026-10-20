package model

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

// yamlProblem is a source-local failure discovered while reading one YAML
// value tree. Key normal form and collisions use the pinned Unicode profile;
// key lexical roles are checked separately in the manifest/Definition context.
type yamlProblem struct {
	code, message string
	position      Position
}

var (
	integerPattern = regexp.MustCompile(`^-?(0|[1-9][0-9]*)$`)
	numberPattern  = regexp.MustCompile(`^(-?(0|[1-9][0-9]*))(?:\.([0-9]+))?(?:[eE]([+-]?[0-9]+))?$`)
)

// parseYAML reads the strict YAML value profile into the Model check's typed IR.
// completed is false only when the supplied recursion budget is exceeded.
func parseYAML(data []byte, depthLimit int) (Value, []yamlProblem, bool) {
	if !utf8.Valid(data) {
		return Value{}, []yamlProblem{{code: "yaml.utf8", message: "source is not valid UTF-8"}}, true
	}
	if depthLimit < 1 {
		return Value{}, []yamlProblem{{code: "limit.yaml_depth", message: "YAML nesting depth limit exceeded"}}, false
	}

	prepared, directiveProblems := prepareYAMLDirectives(data)
	if len(directiveProblems) > 0 {
		return Value{}, directiveProblems, true
	}
	decoder := yaml.NewDecoder(bytes.NewReader(prepared))
	var doc yaml.Node
	if err := decoder.Decode(&doc); err != nil {
		if isYAMLParserDepthLimit(err) {
			return Value{}, []yamlProblem{{code: "limit.yaml_depth", message: err.Error()}}, false
		}
		return Value{}, []yamlProblem{{code: "yaml.syntax", message: err.Error(), position: yamlErrorPosition(err)}}, true
	}
	if doc.Kind == 0 || len(doc.Content) == 0 {
		return Value{}, []yamlProblem{{code: "yaml.root", message: "exactly one non-empty mapping document is required"}}, true
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err == nil && extra.Kind != 0 {
		return Value{}, []yamlProblem{{code: "yaml.documents", message: "exactly one YAML document is allowed", position: nodePosition(&extra)}}, true
	} else if err != nil && err != io.EOF {
		if isYAMLParserDepthLimit(err) {
			return Value{}, []yamlProblem{{code: "limit.yaml_depth", message: err.Error()}}, false
		}
		return Value{}, []yamlProblem{{code: "yaml.syntax", message: err.Error(), position: yamlErrorPosition(err)}}, true
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return Value{}, []yamlProblem{{code: "yaml.root", message: "the YAML document root must be a mapping", position: nodePosition(root)}}, true
	}

	problems := []yamlProblem{}
	value, complete := decodeYAMLNode(root, 1, depthLimit, &problems)
	sort.SliceStable(problems, func(i, j int) bool {
		a, b := problems[i].position, problems[j].position
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Column != b.Column {
			return a.Column < b.Column
		}
		if problems[i].code != problems[j].code {
			return problems[i].code < problems[j].code
		}
		return problems[i].message < problems[j].message
	})
	return value, problems, complete
}

// yaml.v3 accepts YAML 1.1's directive spelling but rejects 1.2 in some
// releases. Validate directives only before the first document content, then
// make the same-length version substitution solely for parser compatibility.
func prepareYAMLDirectives(data []byte) ([]byte, []yamlProblem) {
	copyData := append([]byte(nil), data...)
	problems := []yamlProblem{}
	seenYAML := false
	firstLine := true
	for offset := 0; offset < len(data); {
		end := bytes.IndexByte(data[offset:], '\n')
		if end < 0 {
			end = len(data)
		} else {
			end += offset + 1
		}
		line := bytes.TrimSuffix(data[offset:end], []byte("\n"))
		line = bytes.TrimSuffix(line, []byte("\r"))
		view := line
		if firstLine && bytes.HasPrefix(view, []byte{0xef, 0xbb, 0xbf}) {
			view = view[3:]
		}
		firstLine = false
		trimmed := strings.TrimSpace(string(view))
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			offset = end
			continue
		}
		if len(view) == 0 || view[0] != '%' {
			break
		}
		if len(view) > 0 && view[0] == '%' {
			text := string(view)
			bodyText := text
			if comment := strings.IndexByte(text, '#'); comment >= 0 {
				if comment == 0 || (text[comment-1] != ' ' && text[comment-1] != '\t') {
					bodyText = text + " invalid-comment-spacing"
				} else {
					bodyText = text[:comment]
				}
			}
			body := strings.TrimSpace(bodyText)
			parts := strings.Fields(body)
			pos := Position{Line: 1, Column: 1}
			for i := 0; i < offset; i++ {
				if data[i] == '\n' {
					pos.Line++
				}
			}
			if len(parts) == 2 && parts[0] == "%YAML" && parts[1] == "1.2" && !seenYAML {
				seenYAML = true
				version := bytes.Index(line, []byte("1.2"))
				if version >= 0 {
					copyData[offset+version+2] = '1'
				}
			} else {
				problems = append(problems, yamlProblem{code: "yaml.directive", message: "only one %YAML 1.2 directive is allowed; other directives and versions are forbidden", position: pos})
			}
		}
		offset = end
	}
	return copyData, problems
}

func decodeYAMLNode(node *yaml.Node, depth, limit int, problems *[]yamlProblem) (Value, bool) {
	if depth > limit {
		*problems = append(*problems, yamlProblem{code: "limit.yaml_depth", message: "YAML nesting depth limit exceeded", position: nodePosition(node)})
		return Value{}, false
	}
	if node.Anchor != "" || node.Kind == yaml.AliasNode {
		*problems = append(*problems, yamlProblem{code: "yaml.anchor_alias", message: "YAML anchors and aliases are forbidden", position: nodePosition(node)})
		return Value{}, true
	}
	if node.Style&yaml.TaggedStyle != 0 {
		*problems = append(*problems, yamlProblem{code: "yaml.explicit_tag", message: "explicit YAML tags are forbidden", position: nodePosition(node)})
		return Value{}, true
	}
	switch node.Kind {
	case yaml.ScalarNode:
		return decodeScalar(node, problems), true
	case yaml.SequenceNode:
		items := make([]Value, 0, len(node.Content))
		complete := true
		for _, child := range node.Content {
			item, ok := decodeYAMLNode(child, depth+1, limit, problems)
			items = append(items, item)
			complete = complete && ok
			if !ok {
				break
			}
		}
		return Value{kind: ListValue, items: items, position: nodePosition(node)}, complete
	case yaml.MappingNode:
		fields := make([]Field, 0, len(node.Content)/2)
		seen := map[string]Position{}
		seenNames := map[string]string{}
		complete := true
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyNode, valueNode := node.Content[i], node.Content[i+1]
			if keyNode.Kind != yaml.ScalarNode || keyNode.Anchor != "" || keyNode.Style&yaml.TaggedStyle != 0 {
				*problems = append(*problems, yamlProblem{code: "yaml.key_type", message: "mapping keys must be untagged strings", position: nodePosition(keyNode)})
				_, ok := decodeYAMLNode(valueNode, depth+1, limit, problems)
				complete = complete && ok
				if !ok {
					break
				}
				continue
			}
			if keyNode.Value == "<<" && keyNode.Tag == "!!merge" {
				*problems = append(*problems, yamlProblem{code: "yaml.merge_key", message: "YAML merge keys are forbidden", position: nodePosition(keyNode)})
				_, ok := decodeYAMLNode(valueNode, depth+1, limit, problems)
				complete = complete && ok
				if !ok {
					break
				}
				continue
			}
			key := decodeScalar(keyNode, problems)
			if key.kind != StringValue {
				*problems = append(*problems, yamlProblem{code: "yaml.key_type", message: "mapping keys must be strings", position: nodePosition(keyNode)})
				_, ok := decodeYAMLNode(valueNode, depth+1, limit, problems)
				complete = complete && ok
				if !ok {
					break
				}
				continue
			}
			if first, ok := seen[key.text]; ok {
				*problems = append(*problems, yamlProblem{code: "yaml.duplicate_key", message: fmt.Sprintf("duplicate mapping key %q (first at %d:%d)", key.text, first.Line, first.Column), position: nodePosition(keyNode)})
			} else {
				seen[key.text] = nodePosition(keyNode)
			}
			if !isNFC(key.text) {
				*problems = append(*problems, yamlProblem{code: "name.key_nfc", message: "mapping keys must already be NFC", position: nodePosition(keyNode)})
			}
			keyName := nameKey(key.text)
			if first, exists := seenNames[keyName]; exists && first != key.text {
				*problems = append(*problems, yamlProblem{code: "name.key_collision", message: fmt.Sprintf("mapping key %q collides with %q", key.text, first), position: nodePosition(keyNode)})
			} else {
				seenNames[keyName] = key.text
			}
			value, ok := decodeYAMLNode(valueNode, depth+1, limit, problems)
			fields = append(fields, Field{Name: key.text, Value: value})
			complete = complete && ok
			if !ok {
				break
			}
		}
		return Value{kind: ObjectValue, fields: fields, position: nodePosition(node)}, complete
	default:
		*problems = append(*problems, yamlProblem{code: "yaml.node_type", message: "unsupported YAML node", position: nodePosition(node)})
		return Value{}, true
	}
}

func decodeScalar(node *yaml.Node, problems *[]yamlProblem) Value {
	pos := nodePosition(node)
	if node.Style == 0 && (node.Value == "" || isNullToken(node.Value)) {
		*problems = append(*problems, yamlProblem{code: "yaml.null", message: "null values and empty YAML values are forbidden", position: pos})
		return Value{position: pos}
	}
	if node.Style != 0 {
		return Value{kind: StringValue, text: node.Value, position: pos}
	}
	if node.Value == "true" || node.Value == "false" {
		return Value{kind: BooleanValue, boolean: node.Value == "true", position: pos}
	}
	if integerPattern.MatchString(node.Value) {
		integer, ok := new(big.Int).SetString(node.Value, 10)
		if ok {
			return Value{kind: IntegerValue, text: integer.String(), position: pos}
		}
	}
	if matchesNumber(node.Value) {
		return Value{kind: NumberValue, decimal: canonicalDecimal(node.Value), position: pos}
	}
	if isNonFinite(node.Value) {
		*problems = append(*problems, yamlProblem{code: "yaml.non_finite", message: "non-finite numbers are forbidden", position: pos})
		return Value{position: pos}
	}
	return Value{kind: StringValue, text: node.Value, position: pos}
}

func matchesNumber(s string) bool {
	return numberPattern.MatchString(s) && (strings.ContainsAny(s, ".eE"))
}
func isNullToken(s string) bool { return s == "null" || s == "Null" || s == "NULL" || s == "~" }
func isNonFinite(s string) bool {
	lower := strings.ToLower(s)
	if lower == ".nan" || lower == ".inf" || lower == "+.inf" || lower == "-.inf" || lower == "nan" {
		return true
	}
	unsigned := lower
	if strings.HasPrefix(unsigned, "+") || strings.HasPrefix(unsigned, "-") {
		unsigned = unsigned[1:]
	}
	return unsigned == "inf" || unsigned == "infinity"
}

func canonicalDecimal(s string) Decimal {
	parts := numberPattern.FindStringSubmatch(s)
	whole, fraction, exponent := parts[1], parts[3], parts[4]
	coefficient := whole + fraction
	negative := strings.HasPrefix(whole, "-")
	coefficient = strings.TrimPrefix(coefficient, "-")
	exp := new(big.Int)
	if exponent != "" {
		exp.SetString(exponent, 10)
	}
	exp.Sub(exp, big.NewInt(int64(len(fraction))))
	coefficient = strings.TrimLeft(coefficient, "0")
	if coefficient == "" {
		return Decimal{Coefficient: "0", Exponent: "0"}
	}
	trimmed := strings.TrimRight(coefficient, "0")
	exp.Add(exp, big.NewInt(int64(len(coefficient)-len(trimmed))))
	if negative {
		trimmed = "-" + trimmed
	}
	return Decimal{Coefficient: trimmed, Exponent: exp.String()}
}

func yamlErrorPosition(_ error) Position { return Position{} }

func isYAMLParserDepthLimit(err error) bool {
	return strings.Contains(err.Error(), "exceeded max depth")
}

func nodePosition(node *yaml.Node) Position {
	return Position{Line: max(1, node.Line), Column: max(1, node.Column)}
}
