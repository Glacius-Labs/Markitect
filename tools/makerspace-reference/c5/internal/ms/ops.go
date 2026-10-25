package ms

import (
	"encoding/json"
	"fmt"
)

type refKind int

const (
	refNone refKind = iota
	refMember
	refFamily
	refCourse
	refMachine
	refWindow
	refReservation
	refInvoice
	refMaterial
	refIncident
)

type refusal struct{ code, reason string }

func refuse(code, format string, a ...any) *refusal {
	return &refusal{code: code, reason: fmt.Sprintf(format, a...)}
}

// call is one request that passed the checks before the operation rules.
type call struct {
	s     *state
	actor *user
	at    int
	vals  map[string]value
}

func (c *call) str(name string) string { return c.vals[name].s }
func (c *call) num(name string) int    { return c.vals[name].n }

// member returns the member named by an argument that was resolved as a member.
func (c *call) member(name string) *member { return c.s.members[c.vals[name].s] }

type handler func(c *call) (result rec, target string, ref *refusal)

// opDef is one row of the operation index (section 14).
type opDef struct {
	name     string
	mutation bool
	perms    string // permissions of member, staff, admin, steward: a(ll), o(wn), n(one)
	args     []argSpec
	run      handler
}

var roleIndex = map[string]int{"member": 0, "staff": 1, "admin": 2, "steward": 3}

func id(name string) argSpec { return argSpec{name: name, kind: kindIdentifier} }
func idRef(name string, r refKind) argSpec {
	return argSpec{name: name, kind: kindIdentifier, ref: r}
}
func text(name string) argSpec { return argSpec{name: name, kind: kindText} }
func integer(name string, min, max int) argSpec {
	return argSpec{name: name, kind: kindInteger, min: min, max: max}
}
func stamp(name string) argSpec { return argSpec{name: name, kind: kindTimestamp} }
func date(name string) argSpec  { return argSpec{name: name, kind: kindDate} }
func month(name string) argSpec { return argSpec{name: name, kind: kindMonth} }
func enum(name string, values ...string) argSpec {
	return argSpec{name: name, kind: kindEnum, enum: values}
}

var tiers = []string{"basic", "plus", "pro"}

var operations map[string]*opDef

func lookupOp(name string) (*opDef, bool) {
	if operations == nil {
		operations = map[string]*opDef{}
		for _, list := range [][]*opDef{memberOps(), trainingOps(), equipmentOps(), bookingOps(),
			materialOps(), billingOps(), safetyOps(), reportOps(), auditOps()} {
			for _, op := range list {
				operations[op.name] = op
			}
		}
	}
	op, ok := operations[name]
	return op, ok
}

var commonArgs = []argSpec{id("actor"), stamp("at")}

func (s *state) exists(kind refKind, key string) bool {
	switch kind {
	case refMember:
		_, ok := s.members[key]
		return ok
	case refFamily:
		_, ok := s.families[key]
		return ok
	case refCourse:
		_, ok := s.courses[key]
		return ok
	case refMachine:
		_, ok := s.machines[key]
		return ok
	case refWindow:
		_, ok := s.windows[key]
		return ok
	case refReservation:
		_, ok := s.reservations[key]
		return ok
	case refInvoice:
		_, ok := s.invoices[key]
		return ok
	case refMaterial:
		_, ok := s.materials[key]
		return ok
	case refIncident:
		_, ok := s.incidents[key]
		return ok
	}
	return true
}

// owns reports whether the records named in the call belong to the actor
// (section 4.1).
func (c *call) owns() bool {
	if v, ok := c.vals["memberId"]; ok && v.s != c.actor.id {
		return false
	}
	if v, ok := c.vals["reservationId"]; ok && c.s.reservations[v.s].memberID != c.actor.id {
		return false
	}
	if v, ok := c.vals["invoiceId"]; ok && c.s.invoices[v.s].memberID != c.actor.id {
		return false
	}
	return true
}

// execute runs the checks of section 3.1 in order.
func (s *state) execute(op *opDef, raw map[string]json.RawMessage) response {
	refused := func(r *refusal) response {
		return response{status: "refused", code: r.code, reason: r.reason}
	}
	c := &call{s: s, vals: map[string]value{}}
	// 1 arguments
	for _, spec := range append(append([]argSpec{}, commonArgs...), op.args...) {
		rawVal, present := raw[spec.name]
		v, ok := validate(spec, rawVal, present)
		if !ok {
			return refused(refuse("invalid_argument", "Argument %s is missing or invalid.", spec.name))
		}
		c.vals[spec.name] = v
	}
	c.at = c.vals["at"].n
	// 2 actor
	c.actor = s.users[c.vals["actor"].s]
	if c.actor == nil {
		return refused(refuse("unknown_actor", "The actor %s is not a user.", c.vals["actor"].s))
	}
	// 3 request time
	if c.at < s.lastOK {
		return refused(refuse("time_regression", "The request time is earlier than the latest accepted request."))
	}
	// 4 role permission
	perm := op.perms[roleIndex[c.actor.role]]
	if perm == 'n' {
		s.auditForbidden(c, op)
		return refused(refuse("forbidden", "The role %s may not use %s.", c.actor.role, op.name))
	}
	// 5 referenced records
	for _, spec := range op.args {
		if spec.ref == refNone {
			continue
		}
		key := c.vals[spec.name].s
		if !s.exists(spec.ref, key) {
			return refused(refuse("not_found", "%s %s does not exist.", spec.name, key))
		}
	}
	// 6 ownership
	if perm == 'o' && !c.owns() {
		s.auditForbidden(c, op)
		return refused(refuse("forbidden", "The record belongs to another member."))
	}
	// 7 operation rules and effects
	result, target, ref := op.run(c)
	if ref != nil {
		return refused(ref)
	}
	s.lastOK = c.at
	if op.mutation {
		s.audit = append(s.audit, &auditEntry{seq: len(s.audit) + 1, at: c.at, actor: c.actor.id,
			actorRole: c.actor.role, op: op.name, target: target, outcome: "accepted"})
	}
	return response{status: "ok", result: result}
}

// auditForbidden records a refusal for missing permission (section 4.2). It
// does not move the request time.
func (s *state) auditForbidden(c *call, op *opDef) {
	s.audit = append(s.audit, &auditEntry{seq: len(s.audit) + 1, at: c.at, actor: c.actor.id,
		actorRole: c.actor.role, op: op.name, outcome: "refused"})
}
