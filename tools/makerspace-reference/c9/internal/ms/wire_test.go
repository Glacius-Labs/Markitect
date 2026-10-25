package ms_test

import (
	"bytes"
	"strings"
	"testing"

	"makerspace/internal/ms"
)

func TestInvalidRequests(t *testing.T) {
	e := newEnv(t)
	good := `"args":{"actor":"admin","at":"2026-03-02T09:00Z"}`
	cases := []struct {
		name, line string
		id         any
	}{
		{"not json", `{"id":"a"`, nil},
		{"empty", ``, nil},
		{"array", `[1]`, nil},
		{"number", `5`, nil},
		{"null", `null`, nil},
		{"missing id", `{"op":"listMachines",` + good + `}`, nil},
		{"numeric id", `{"id":5,"op":"listMachines",` + good + `}`, nil},
		{"empty id", `{"id":"","op":"listMachines",` + good + `}`, nil},
		{"missing op", `{"id":"a",` + good + `}`, "a"},
		{"numeric op", `{"id":"a","op":3,` + good + `}`, "a"},
		{"unknown op", `{"id":"a","op":"nope",` + good + `}`, "a"},
		{"op case", `{"id":"a","op":"ListMachines",` + good + `}`, "a"},
		{"missing args", `{"id":"a","op":"listMachines"}`, "a"},
		{"args array", `{"id":"a","op":"listMachines","args":[]}`, "a"},
		{"args null", `{"id":"a","op":"listMachines","args":null}`, "a"},
		{"extra member", `{"id":"a","op":"listMachines",` + good + `,"extra":1}`, "a"},
		{"unknown argument", `{"id":"a","op":"listMachines","args":{"actor":"admin","at":"2026-03-02T09:00Z","x":1}}`, "a"},
		{"argument case", `{"id":"a","op":"getMember","args":{"actor":"admin","at":"2026-03-02T09:00Z","memberID":"ada"}}`, "a"},
		{"actor case", `{"id":"a","op":"listMachines","args":{"Actor":"admin","at":"2026-03-02T09:00Z"}}`, "a"},
	}
	for _, c := range cases {
		r := e.raw(c.line)
		if r["status"] != "invalidRequest" || r["id"] != c.id || r["reason"] == "" {
			t.Errorf("%s: %v", c.name, r)
		}
		if _, ok := r["code"]; ok || len(r) != 3 {
			t.Errorf("%s: unexpected members %v", c.name, r)
		}
	}
	// the same id twice is two requests
	for i := 0; i < 2; i++ {
		r := e.raw(`{"id":"same","op":"listMachines",` + good + `}`)
		if r["status"] != "ok" || r["id"] != "same" {
			t.Fatalf("%v", r)
		}
	}
}

func TestInvalidRequestChangesNothing(t *testing.T) {
	e := newEnv(t)
	e.raw(`{"id":"a","op":"registerMember","args":{"actor":"admin","at":"2026-03-02T09:00Z","memberId":"ada","name":"A","tier":"basic","x":1}}`)
	e.refused("admin", "2026-03-02T09:00Z", "getMember", m{"memberId": "ada"}, "not_found")
	if n := len(list(e.ok("admin", "2026-03-02T09:00Z", "listAuditEntries", m{})["entries"])); n != 0 {
		t.Fatalf("audit entries: %d", n)
	}
}

func TestResponseShapes(t *testing.T) {
	e := newEnv(t)
	r := e.send("admin", "2026-03-02T09:00Z", "listMachines", m{})
	assertKeys(t, r, "id", "status", "result")
	assertKeys(t, obj(r["result"]), "machines")
	if l := list(obj(r["result"])["machines"]); l == nil || len(l) != 0 {
		t.Fatalf("machines: %v", l)
	}
	r = e.send("admin", "2026-03-02T09:00Z", "getMember", m{"memberId": "x"})
	assertKeys(t, r, "id", "status", "code", "reason")
}

func TestOneResponsePerLine(t *testing.T) {
	sv := ms.NewServer()
	out := sv.Handle([]byte(`{"id":"1","op":"listMachines","args":{"actor":"admin","at":"2026-03-02T09:00Z"}}`))
	if bytes.ContainsAny(out, "\r\n") {
		t.Fatalf("response spans lines: %q", out)
	}
	// a last line without a line feed is still a request
	if !strings.Contains(string(out), `"status":"ok"`) {
		t.Fatalf("%s", out)
	}
	// a long line is accepted
	long := `{"id":"2","op":"registerUser","args":{"actor":"admin","at":"2026-03-02T09:00Z","userId":"u","name":"` +
		strings.Repeat("x", 60000) + `","role":"staff"}}`
	if out := sv.Handle([]byte(long)); !strings.Contains(string(out), `"status":"ok"`) {
		t.Fatalf("%.80s", out)
	}
}

func TestArgumentValues(t *testing.T) {
	e := shop(t)
	at := "2026-03-02T09:00Z"
	bad := func(op string, args m) { e.refused("admin", at, op, args, "invalid_argument") }
	reg := func(extra m) m {
		a := m{"memberId": "zed", "name": "Zed", "tier": "basic"}
		for k, v := range extra {
			a[k] = v
		}
		return a
	}
	for _, id := range []any{"", "-x", ".x", "_x", "a b", "a/b", "ä", strings.Repeat("a", 65), 5, nil, true, []any{}} {
		bad("registerMember", reg(m{"memberId": id}))
	}
	for _, id := range []string{"a", "A1", "9", "a.b_c-d", strings.Repeat("a", 64)} {
		e.ok("admin", at, "registerMember", reg(m{"memberId": id}))
	}
	for _, name := range []any{"", " ", "\t\n", " ", " ", 5, nil} {
		bad("registerMember", reg(m{"name": name}))
	}
	e.ok("admin", at, "registerMember", reg(m{"memberId": "padded", "name": " x "}))
	got := obj(e.ok("admin", at, "getMember", m{"memberId": "padded"})["member"])
	if got["name"] != " x " {
		t.Fatalf("name was trimmed: %v", got)
	}
	bad("registerMember", reg(m{"tier": "gold"}))
	bad("registerMember", reg(m{"tier": "Basic"}))
	bad("registerMember", map[string]any{"memberId": "q", "name": "q"})
	for _, n := range []string{"-0", "-1", "10.0", "1e3", "1E3", `"5"`, "null", "true", "100000000000000000000", "1000001"} {
		line := `{"id":"n","op":"addCourse","args":{"actor":"admin","at":"` + at +
			`","courseId":"nc","familyId":"laser","title":"T","feeCents":` + n + `,"capacity":1}}`
		if r := e.raw(line); r["code"] != "invalid_argument" {
			t.Errorf("feeCents %s: %v", n, r)
		}
	}
	for _, n := range []string{"0", "1000000"} {
		line := `{"id":"n","op":"addCourse","args":{"actor":"admin","at":"` + at +
			`","courseId":"nc` + n + `","familyId":"laser","title":"T","feeCents":` + n + `,"capacity":1}}`
		if r := e.raw(line); r["status"] != "ok" {
			t.Errorf("feeCents %s: %v", n, r)
		}
	}
	for _, ts := range []any{"2026-03-02T09:00", "2026-03-02 09:00Z", "2026-03-02T09:00:00Z", "2026-02-30T09:00Z",
		"2026-03-02T24:00Z", "2026-03-02T09:60Z", "1999-12-31T23:59Z", "2100-01-01T00:00Z", "2026-3-2T9:00Z", 5, nil} {
		bad("listMachines", m{"at": ts})
	}
	e2 := newEnv(t)
	e2.ok("admin", "2000-01-01T00:00Z", "listMachines", m{})
	e2.ok("admin", "2099-12-31T23:59Z", "listMachines", m{})
	e2.ok("admin", "2099-12-31T23:59Z", "revenueReport", m{"month": "2099-12"})
	for _, d := range []any{"2026-02-29", "2026-13-01", "2026-3-01", "1999-12-31", "2100-01-01", "2026-03-02T00:00Z"} {
		bad("getMachineSchedule", m{"machineId": "lc1", "date": d})
	}
	e.ok("admin", at, "getMachineSchedule", m{"machineId": "lc1", "date": "2028-02-29"})
	for _, mo := range []any{"2026-13", "2026-00", "1999-12", "2100-01", "2026-3", "2026-03-01"} {
		bad("revenueReport", m{"month": mo})
	}
}

func TestMissingAndNullArguments(t *testing.T) {
	e := shop(t)
	at := "2026-03-02T09:00Z"
	e.refused("admin", at, "getMember", m{}, "invalid_argument")
	e.refused("admin", at, "getMember", m{"memberId": nil}, "invalid_argument")
	line := `{"id":"1","op":"listMachines","args":{"at":"2026-03-02T09:00Z"}}`
	if r := e.raw(line); r["code"] != "invalid_argument" {
		t.Fatalf("%v", r)
	}
	line = `{"id":"1","op":"listMachines","args":{"actor":"admin"}}`
	if r := e.raw(line); r["code"] != "invalid_argument" {
		t.Fatalf("%v", r)
	}
}

func TestIdentifiersAreCaseSensitive(t *testing.T) {
	e := shop(t)
	e.ok("sam", "2026-03-02T09:00Z", "registerMember", m{"memberId": "Ada", "name": "Other", "tier": "basic"})
	e.refused("SAM", "2026-03-02T09:00Z", "listMachines", m{}, "unknown_actor")
}
