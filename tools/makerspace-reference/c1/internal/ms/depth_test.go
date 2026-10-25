package ms_test

import (
	"strings"
	"testing"
)

// nested builds an array nested to the given level, counting levels from the
// level of the value itself.
func nested(levels int) string {
	return strings.Repeat("[", levels) + strings.Repeat("]", levels)
}

func TestNestingUpToLevel100(t *testing.T) {
	e := newEnv(t)
	at := `"actor":"admin","at":"2026-03-02T09:00Z"`
	// the request object is level 1, args level 2, an argument value level 3: depth 98 inside it reaches level 100
	r := e.raw(`{"id":"d","op":"registerUser","args":{` + at + `,"userId":"u","role":"staff","name":` + nested(98) + `}}`)
	if r["code"] != "invalid_argument" || r["id"] != "d" {
		t.Fatalf("%v", r)
	}
	// an unknown argument with a deeply nested value is still an invalid request
	r = e.raw(`{"id":"d","op":"listMachines","args":{` + at + `,"x":` + nested(98) + `}}`)
	if r["status"] != "invalidRequest" || r["id"] != "d" {
		t.Fatalf("%v", r)
	}
	// a valid request is processed in full while its extra top-level value is not part of the rules
	r = e.raw(`{"id":"d","op":"listMachines","args":{` + at + `}}`)
	if r["status"] != "ok" {
		t.Fatalf("%v", r)
	}
	// invalid JSON is answered by section 2.4 whatever its depth
	r = e.raw(`{"id":"d","op":"listMachines","args":{` + at + `,"x":` + strings.Repeat("[", 150) + `}}`)
	if r["status"] != "invalidRequest" || r["id"] != nil {
		t.Fatalf("%v", r)
	}
}

func TestActorThatIsNotAnIdentifier(t *testing.T) {
	e := newEnv(t)
	for _, actor := range []any{"bad id!", strings.Repeat("a", 65), "", "-x", 5, nil} {
		e.refused("admin", "2026-03-02T09:00Z", "listMachines", m{"actor": actor}, "invalid_argument")
	}
	e.refused("nobody", "2026-03-02T09:00Z", "listMachines", m{}, "unknown_actor")
	e.refused("admin", "2026-03-02T09:00Z", "listMachines", m{"actor": strings.Repeat("a", 64)}, "unknown_actor")
	if r := e.raw(`{"id":"1","op":"listMachines","args":{"at":"2026-03-02T09:00Z"}}`); r["code"] != "invalid_argument" {
		t.Fatalf("%v", r)
	}
}
