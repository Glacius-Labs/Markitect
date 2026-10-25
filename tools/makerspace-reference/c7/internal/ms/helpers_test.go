package ms_test

import (
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"makerspace/internal/ms"
)

type env struct {
	t  *testing.T
	sv *ms.Server
	n  int
}

type resp = map[string]any

func newEnv(t *testing.T) *env { return &env{t: t, sv: ms.NewServer()} }

// raw sends a line and returns the parsed response.
func (e *env) raw(line string) resp {
	e.t.Helper()
	out := e.sv.Handle([]byte(line))
	var r resp
	if err := json.Unmarshal(out, &r); err != nil {
		e.t.Fatalf("response is not JSON: %s", out)
	}
	return r
}

// send builds a request from the arguments and returns the parsed response.
func (e *env) send(actor, at, op string, args map[string]any) resp {
	e.t.Helper()
	e.n++
	a := map[string]any{"actor": actor, "at": at}
	for k, v := range args {
		a[k] = v
	}
	line, _ := json.Marshal(map[string]any{"id": "q" + strconv.Itoa(e.n), "op": op, "args": a})
	return e.raw(string(line))
}

func (e *env) ok(actor, at, op string, args map[string]any) map[string]any {
	e.t.Helper()
	r := e.send(actor, at, op, args)
	if r["status"] != "ok" {
		e.t.Fatalf("%s: want ok, got %v", op, r)
	}
	return r["result"].(map[string]any)
}

func (e *env) refused(actor, at, op string, args map[string]any, code string) {
	e.t.Helper()
	r := e.send(actor, at, op, args)
	if r["status"] != "refused" || r["code"] != code {
		e.t.Fatalf("%s: want refused %s, got %v", op, code, r)
	}
	if reason, _ := r["reason"].(string); reason == "" {
		e.t.Fatalf("%s: refusal without reason: %v", op, r)
	}
}

type m = map[string]any

func obj(v any) map[string]any { return v.(map[string]any) }
func list(v any) []any         { return v.([]any) }

func keys(v map[string]any) []string {
	out := make([]string, 0, len(v))
	for k := range v {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func assertKeys(t *testing.T, v map[string]any, want ...string) {
	t.Helper()
	sort.Strings(want)
	if !reflect.DeepEqual(keys(v), want) {
		t.Fatalf("members = %v, want %v", keys(v), want)
	}
}

func num(v any) int { return int(v.(float64)) }

// shop sets up a small makerspace: staff sam, member ada (plus), family laser,
// machine lc1 (rate 45) and a certification for ada, all at the given time.
func shop(t *testing.T) *env {
	e := newEnv(t)
	const at = "2026-03-02T09:00Z"
	e.ok("admin", at, "registerUser", m{"userId": "sam", "name": "Sam", "role": "staff"})
	e.ok("admin", at, "registerUser", m{"userId": "stu", "name": "Stu", "role": "steward"})
	e.ok("sam", at, "registerMember", m{"memberId": "ada", "name": "Ada Byron", "tier": "plus"})
	e.ok("admin", at, "addFamily", m{"familyId": "laser", "name": "Laser cutters"})
	e.ok("admin", at, "addMachine", m{"machineId": "lc1", "name": "Laser 1", "familyId": "laser", "rateCentsPerMinute": 45})
	e.ok("sam", at, "addCourse", m{"courseId": "laser-101", "familyId": "laser", "title": "Laser basics", "feeCents": 3000, "capacity": 2})
	e.ok("sam", at, "enrollInCourse", m{"courseId": "laser-101", "memberId": "ada"})
	e.ok("sam", at, "completeCourse", m{"courseId": "laser-101", "memberId": "ada"})
	return e
}
