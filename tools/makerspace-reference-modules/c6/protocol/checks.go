package protocol

import (
	"encoding/json"

	"makerspace/policies"
)

var commonArgs = []policies.ArgSpec{policies.ID("actor"), policies.Stamp("at")}

// owns reports whether the records named in the call belong to the actor
// (section 4.1).
func owns(op *policies.Operation, c *policies.Call) bool {
	for _, spec := range op.Args {
		if spec.Owner != nil && spec.Owner(c.Vals[spec.Name].S) != c.ActorID {
			return false
		}
	}
	return true
}

// execute runs the checks of section 3.1 in order.
func (sv *Server) execute(op *policies.Operation, raw map[string]json.RawMessage) response {
	refused := func(r *policies.Refusal) response {
		return response{status: "refused", code: r.Code, reason: r.Reason}
	}
	c := &policies.Call{Vals: map[string]policies.Value{}}
	// 1 arguments
	for _, spec := range append(append([]policies.ArgSpec{}, commonArgs...), op.Args...) {
		rawVal, present := raw[spec.Name]
		v, ok := validate(spec, rawVal, present)
		if !ok {
			return refused(policies.Refuse("invalid_argument", "Argument %s is missing or invalid.", spec.Name))
		}
		c.Vals[spec.Name] = v
	}
	c.At = c.Vals["at"].N
	// 2 actor
	actor := sv.members.User(c.Vals["actor"].S)
	if actor == nil {
		return refused(policies.Refuse("unknown_actor", "The actor %s is not a user.", c.Vals["actor"].S))
	}
	c.ActorID, c.ActorRole = actor.ID, actor.Role
	// 3 request time
	if c.At < sv.lastOK {
		return refused(policies.Refuse("time_regression", "The request time is earlier than the latest accepted request."))
	}
	// 4 role permission
	perm := op.Perms[policies.RoleIndex(c.ActorRole)]
	if perm == 'n' {
		sv.auditForbidden(c, op)
		return refused(policies.Refuse("forbidden", "The role %s may not use %s.", c.ActorRole, op.Name))
	}
	// 5 referenced records
	for _, spec := range op.Args {
		if spec.Exists == nil {
			continue
		}
		key := c.Vals[spec.Name].S
		if !spec.Exists(key) {
			return refused(policies.Refuse("not_found", "%s %s does not exist.", spec.Name, key))
		}
	}
	// 6 ownership
	if perm == 'o' && !owns(op, c) {
		sv.auditForbidden(c, op)
		return refused(policies.Refuse("forbidden", "The record belongs to another member."))
	}
	// 7 operation rules and effects
	result, target, ref := op.Run(c)
	if ref != nil {
		return refused(ref)
	}
	sv.lastOK = c.At
	if op.Mutation {
		sv.audit = append(sv.audit, &auditEntry{seq: len(sv.audit) + 1, at: c.At, actor: c.ActorID,
			actorRole: c.ActorRole, op: op.Name, target: target, outcome: "accepted"})
	}
	return response{status: "ok", result: result}
}
