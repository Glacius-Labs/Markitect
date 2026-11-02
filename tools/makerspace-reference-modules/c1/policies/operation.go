package policies

// Rec is a JSON object with exactly the members of one record.
type Rec = map[string]any

// ValueKind is the field type of an argument.
type ValueKind int

const (
	KindIdentifier ValueKind = iota
	KindText
	KindInteger
	KindTimestamp
	KindDate
	KindMonth
	KindEnum
)

// ArgSpec describes one argument of an operation. Exists, when set, tells
// whether the record the argument names exists (check 5 of the order of
// checks). Owner, when set, gives the member the named record belongs to
// (check 6).
type ArgSpec struct {
	Name     string
	Kind     ValueKind
	Min, Max int
	Enum     []string
	Exists   func(key string) bool
	Owner    func(key string) string
}

// Value is a validated argument: a string for every kind except integers.
type Value struct {
	S string
	N int
}

// Call is one request that passed the checks before the operation rules.
type Call struct {
	ActorID   string
	ActorRole string
	At        int
	Vals      map[string]Value
}

// Str is the string value of an argument.
func (c *Call) Str(name string) string { return c.Vals[name].S }

// Num is the integer value of an argument.
func (c *Call) Num(name string) int { return c.Vals[name].N }

// Operation is one row of the operation index: how a caller reaches the
// operation and what it does. Perms has one letter per role, a(ll), o(wn) or
// n(one), in the order of RoleIndex. Run gives the result, the audit target
// and the refusal, if any.
type Operation struct {
	Name     string
	Mutation bool
	Perms    string
	Args     []ArgSpec
	Run      func(c *Call) (Rec, string, *Refusal)
}

// ID is an identifier argument that names nothing yet.
func ID(name string) ArgSpec { return ArgSpec{Name: name, Kind: KindIdentifier} }

// IDRef is an identifier argument that must name an existing record.
func IDRef(name string, exists func(key string) bool) ArgSpec {
	return ArgSpec{Name: name, Kind: KindIdentifier, Exists: exists}
}

// OwnedIDRef is an identifier argument that must name an existing record that
// belongs to the member owner gives.
func OwnedIDRef(name string, exists func(key string) bool, owner func(key string) string) ArgSpec {
	return ArgSpec{Name: name, Kind: KindIdentifier, Exists: exists, Owner: owner}
}

// MemberRef is the memberId argument: it must name an existing member, and the
// record concerned belongs to that member.
func MemberRef(exists func(key string) bool) ArgSpec {
	return ArgSpec{Name: "memberId", Kind: KindIdentifier, Exists: exists, Owner: func(key string) string { return key }}
}

// Text is a text argument.
func Text(name string) ArgSpec { return ArgSpec{Name: name, Kind: KindText} }

// Integer is an integer argument within [min, max].
func Integer(name string, min, max int) ArgSpec {
	return ArgSpec{Name: name, Kind: KindInteger, Min: min, Max: max}
}

// Stamp is a timestamp argument.
func Stamp(name string) ArgSpec { return ArgSpec{Name: name, Kind: KindTimestamp} }

// Date is a date argument.
func Date(name string) ArgSpec { return ArgSpec{Name: name, Kind: KindDate} }

// Month is a month argument.
func Month(name string) ArgSpec { return ArgSpec{Name: name, Kind: KindMonth} }

// Enum is an argument equal to one of the listed strings.
func Enum(name string, values ...string) ArgSpec {
	return ArgSpec{Name: name, Kind: KindEnum, Enum: values}
}
