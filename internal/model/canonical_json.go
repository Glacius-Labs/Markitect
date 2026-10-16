package model

import "sort"

// The model-digest/1 tree is deliberately smaller than general JSON. It has no
// null or floating point values, and only the manifest's formatVersion is an
// untagged number. Encoding/json also escapes permitted HTML and line-separator
// characters differently from this contract.
type canonicalNode struct {
	kind    byte
	text    string
	boolean bool
	fields  []canonicalMember
	items   []canonicalNode
}

type canonicalMember struct {
	name  string
	value canonicalNode
}

func canonicalString(text string) canonicalNode { return canonicalNode{kind: 's', text: text} }
func canonicalArray(items ...canonicalNode) canonicalNode {
	return canonicalNode{kind: 'a', items: items}
}
func canonicalObject(fields ...canonicalMember) canonicalNode {
	return canonicalNode{kind: 'o', fields: fields}
}
func canonicalReference(id Identity) canonicalNode {
	return canonicalArray(canonicalString("reference"), canonicalString(id.Package), canonicalString(id.ID))
}

// Encoding and value projection use explicit stacks: a caller-configured YAML
// budget does not become an implicit Go call-stack budget in the final phase.
func encodeCanonical(root canonicalNode) []byte {
	type job struct {
		node      canonicalNode
		literal   string
		separator byte
	}
	stack := []job{{node: root}}
	var result []byte
	for len(stack) > 0 {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]
		if current.separator != 0 {
			result = append(result, current.separator)
			continue
		}
		if current.literal != "" {
			result = append(result, current.literal...)
			continue
		}
		node := current.node
		switch node.kind {
		case 's':
			result = appendCanonicalString(result, node.text)
		case 'n':
			result = append(result, node.text...)
		case 'b':
			if node.boolean {
				result = append(result, "true"...)
			} else {
				result = append(result, "false"...)
			}
		case 'a':
			result = append(result, '[')
			stack = append(stack, job{separator: ']'})
			for index := len(node.items) - 1; index >= 0; index-- {
				stack = append(stack, job{node: node.items[index]})
				if index > 0 {
					stack = append(stack, job{separator: ','})
				}
			}
		case 'o':
			fields := append([]canonicalMember(nil), node.fields...)
			sort.Slice(fields, func(i, j int) bool { return fields[i].name < fields[j].name })
			result = append(result, '{')
			stack = append(stack, job{separator: '}'})
			for index := len(fields) - 1; index >= 0; index-- {
				stack = append(stack, job{node: fields[index].value}, job{separator: ':'}, job{node: canonicalString(fields[index].name)})
				if index > 0 {
					stack = append(stack, job{separator: ','})
				}
			}
		default:
			panic("model: unsupported internal canonical JSON node")
		}
	}
	return result
}

func appendCanonicalString(result []byte, text string) []byte {
	const hex = "0123456789abcdef"
	result = append(result, '"')
	for index := 0; index < len(text); index++ {
		char := text[index]
		switch char {
		case '"', '\\':
			result = append(result, '\\', char)
		case '\b':
			result = append(result, '\\', 'b')
		case '\t':
			result = append(result, '\\', 't')
		case '\n':
			result = append(result, '\\', 'n')
		case '\f':
			result = append(result, '\\', 'f')
		case '\r':
			result = append(result, '\\', 'r')
		default:
			if char < 0x20 {
				result = append(result, '\\', 'u', '0', '0', hex[char>>4], hex[char&15])
			} else {
				result = append(result, char)
			}
		}
	}
	return append(result, '"')
}

func canonicalTypedValue(value Value) (canonicalNode, bool) {
	type job struct {
		value  Value
		output *canonicalNode
	}
	var root canonicalNode
	stack := []job{{value: value, output: &root}}
	for len(stack) > 0 {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]
		v := current.value
		tag := canonicalString(string(v.kind))
		switch v.kind {
		case StringValue, EnumValue, IntegerValue:
			*current.output = canonicalArray(tag, canonicalString(v.text))
		case BooleanValue:
			*current.output = canonicalArray(tag, canonicalNode{kind: 'b', boolean: v.boolean})
		case NumberValue:
			*current.output = canonicalArray(tag, canonicalString(v.decimal.Coefficient), canonicalString(v.decimal.Exponent))
		case ReferenceValue:
			*current.output = canonicalReference(v.reference)
		case ListValue:
			*current.output = canonicalArray(tag, canonicalNode{kind: 'a', items: make([]canonicalNode, len(v.items))})
			for index := range v.items {
				stack = append(stack, job{value: v.items[index], output: &current.output.items[1].items[index]})
			}
		case ObjectValue:
			*current.output = canonicalArray(tag, canonicalNode{kind: 'o', fields: make([]canonicalMember, len(v.fields))})
			for index, field := range v.fields {
				member := &current.output.items[1].fields[index]
				member.name = field.Name
				stack = append(stack, job{value: field.Value, output: &member.value})
			}
		default:
			return canonicalNode{}, false
		}
	}
	return root, true
}
