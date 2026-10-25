package ms_test

import "testing"

func auditLog(e *env, at string) []map[string]any {
	var out []map[string]any
	for _, x := range list(e.ok("admin", at, "listAuditEntries", m{})["entries"]) {
		out = append(out, obj(x))
	}
	return out
}

func TestForbiddenRefusalsAreAudited(t *testing.T) {
	e := shop(t)
	base := len(auditLog(e, "2026-03-05T10:00Z"))
	// a role without permission, for a read and for a mutation
	e.refused("ada", "2026-03-05T10:00Z", "listAuditEntries", m{}, "forbidden")
	e.refused("ada", "2026-03-05T10:01Z", "addFamily", m{"familyId": "x", "name": "x"}, "forbidden")
	// an ownership violation
	e.ok("sam", "2026-03-05T10:02Z", "registerMember", m{"memberId": "bob", "name": "Bob", "tier": "basic"})
	e.refused("bob", "2026-03-05T10:03Z", "getMember", m{"memberId": "ada"}, "forbidden")
	log := auditLog(e, "2026-03-05T10:04Z")
	got := log[base:]
	if len(got) != 4 {
		t.Fatalf("entries after the setup: %v", got)
	}
	want := []struct{ actor, role, op, outcome string }{
		{"ada", "member", "listAuditEntries", "refused"},
		{"ada", "member", "addFamily", "refused"},
		{"sam", "staff", "registerMember", "accepted"},
		{"bob", "member", "getMember", "refused"},
	}
	for i, w := range want {
		g := got[i]
		assertKeys(t, g, "seq", "at", "actor", "actorRole", "op", "target", "outcome")
		if g["actor"] != w.actor || g["actorRole"] != w.role || g["op"] != w.op || g["outcome"] != w.outcome || num(g["seq"]) != base+i+1 {
			t.Errorf("entry %d: %v", i, g)
		}
		if w.outcome == "refused" && g["target"] != nil {
			t.Errorf("refusal entry has target: %v", g)
		}
	}
	if got[0]["at"] != "2026-03-05T10:00Z" {
		t.Errorf("%v", got[0])
	}
	// the refusal itself is unchanged and the read that lists the log is not an entry
	if len(log) != base+4 {
		t.Errorf("listing the log added entries: %d", len(log))
	}
}

func TestOnlyForbiddenRefusalsAreAudited(t *testing.T) {
	e := shop(t)
	base := len(auditLog(e, "2026-03-05T10:00Z"))
	e.refused("ghost", "2026-03-05T10:00Z", "listAuditEntries", m{}, "unknown_actor")
	e.refused("ada", "2026-03-05T10:00Z", "getMember", m{"memberId": 3}, "invalid_argument")
	e.refused("ada", "2026-03-05T10:00Z", "getMember", m{"memberId": "nobody"}, "not_found")
	e.raw(`{"id":"x","op":"listAuditEntries","args":{"actor":"ada","at":"2026-03-05T10:00Z","extra":1}}`)
	// time_regression is checked before permission and is not recorded
	e.refused("ada", "2026-03-01T10:00Z", "listAuditEntries", m{}, "time_regression")
	if n := len(auditLog(e, "2026-03-05T10:00Z")) - base; n != 0 {
		t.Fatalf("other refusals added %d entries", n)
	}
}

func TestRefusalEntryDoesNotMoveRequestTime(t *testing.T) {
	e := shop(t)
	e.ok("sam", "2026-03-05T10:00Z", "listMachines", m{})
	e.refused("ada", "2026-03-09T10:00Z", "listAuditEntries", m{}, "forbidden")
	// the refusal at 03-09 did not move the request time, so 03-06 is fine
	e.ok("sam", "2026-03-06T10:00Z", "registerMember", m{"memberId": "bob", "name": "B", "tier": "basic"})
	log := auditLog(e, "2026-03-06T10:00Z")
	last := log[len(log)-1]
	prev := log[len(log)-2]
	if prev["at"] != "2026-03-09T10:00Z" || prev["outcome"] != "refused" || last["at"] != "2026-03-06T10:00Z" ||
		num(last["seq"]) != num(prev["seq"])+1 {
		t.Fatalf("%v %v", prev, last)
	}
}

func TestForbiddenRefusalChangesNothingElse(t *testing.T) {
	e := shop(t)
	e.refused("ada", "2026-03-02T09:00Z", "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-05T10:00Z", "end": "2026-03-05T11:00Z", "reason": "x"}, "forbidden")
	// the identifier counter was not consumed
	w := obj(e.ok("sam", "2026-03-02T09:00Z", "scheduleMaintenance", m{"machineId": "lc1", "start": "2026-03-05T10:00Z", "end": "2026-03-05T11:00Z", "reason": "x"})["window"])
	if w["windowId"] != "mnt-1" {
		t.Fatalf("%v", w)
	}
}
