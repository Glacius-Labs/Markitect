package model

// ValueKind is a closed set of language values. The zero Value is not valid.
type ValueKind string

const (
	StringValue    ValueKind = "string"
	EnumValue      ValueKind = "enum"
	BooleanValue   ValueKind = "boolean"
	IntegerValue   ValueKind = "integer"
	NumberValue    ValueKind = "number"
	ReferenceValue ValueKind = "reference"
	ObjectValue    ValueKind = "object"
	ListValue      ValueKind = "list"
)

// Decimal represents coefficient * 10^exponent exactly. Both members are
// arbitrary-precision signed decimal integer strings; no binary floats occur.
// Compiler output strips coefficient trailing zeros, with zero always (0, 0).
type Decimal struct {
	Coefficient string
	Exponent    string
}

// Value is a tagged language value. Accessors report a kind mismatch rather
// than coercing. Internal construction prevents contradictory union members.
type Value struct {
	kind      ValueKind
	text      string
	boolean   bool
	decimal   Decimal
	reference Identity
	fields    []Field
	items     []Value
	position  Position
}

func (v Value) Kind() ValueKind { return v.kind }

func (v Value) Text() (string, bool) {
	return v.text, v.kind == StringValue || v.kind == EnumValue
}

func (v Value) Boolean() (bool, bool) { return v.boolean, v.kind == BooleanValue }

// Integer returns an arbitrary-precision integer's canonical decimal spelling.
func (v Value) Integer() (string, bool) { return v.text, v.kind == IntegerValue }

func (v Value) Number() (Decimal, bool) { return v.decimal, v.kind == NumberValue }

func (v Value) Reference() (Identity, bool) { return v.reference, v.kind == ReferenceValue }

// Fields and Items return copies so their callers cannot mutate a Value.
func (v Value) Fields() ([]Field, bool) {
	return append([]Field(nil), v.fields...), v.kind == ObjectValue
}

func (v Value) Items() ([]Value, bool) {
	return append([]Value(nil), v.items...), v.kind == ListValue
}
