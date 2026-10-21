package cockpit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The projection decodes these data keys of the events the run engine records
// with map data. The test reads the engine's source and fails when one of them
// is no longer produced for its kind, so a renamed key cannot silently empty a
// view. Kinds the engine records with a non-literal kind share the key "*".
var consumedKeys = map[string][]string{
	"run.start":             {"run", "managers", "start", "agentTurns", "passes", "deadline", "binding"},
	"pass.start":            {"purpose", "review", "writes", "tree"},
	"manager.state_changed": {"from", "to", "stage", "why", "waitingFor"},
	"manager.release":       {"why", "parent"},
	"turn.start":            {"role", "kind", "profile", "readOnly", "resume", "mode", "context"},
	"turn.end":              {"role", "kind", "usage", "stopped", "reports", "reason", "compacted", "probes"},
	"probe.requested":       {"id", "obligation", "claim", "expected", "check", "standKind", "proposal", "stand", "path", "rerun"},
	"probe.disposed":        {"receipt", "disposition", "voided", "why"},
	"dependency.bound":      {"on", "stage", "serves", "proposal", "rebound"},
	"dependency.unbound":    {"on", "stage", "proposal", "cycle"},
	"proposal.submitted":    {"proposal", "kind", "purpose", "envelope", "findings", "closed", "disputed"},
	"proposal.checked":      {"proposal", "passed"},
	"proposal.invalidated":  {"proposal", "why"},
	"proposal.withdrawn":    {"proposal", "why"},
	"basis.invalidated":     {"proposal", "reason"},
	"result.reopened":       {"proposal", "by", "reason"},
	"*":                     {"proposal", "by", "stand", "reason"},
	"report.wait":           {"for", "reason", "stage"},
	"message":               {"from", "to", "text"},
	"candidate":             {"tree"},
	"gate":                  {"at", "result"},
	"pass.sealed":           {"purpose", "verdict", "tree", "managers"},
	"outcome":               {"status", "reason"},
	"helper.start":          {"helper", "role"},
	"helper.end":            {"helper", "role"},
}

// consumedFields are the JSON fields of the submission envelope and its plan
// payload the projection reads.
var consumedFields = map[string][]string{
	"envelope":         {"summary", "verdict", "children", "plan"},
	"plan":             {"dependencies"},
	"dependency":       {"owner", "need", "pass", "stage"},
	"coverage":         {"definition", "status", "note", "evidence"},
	"evidenceItem":     {"claim", "receipt", "source", "location", "reasoning"},
	"ProbeReceipt":     {"id", "pass", "requester", "turn", "obligation", "claim", "expected", "check", "standKind", "proposal", "stand", "path", "outcome", "exitCode", "timedOut", "uncertain", "reason", "rerun", "disposition"},
	"ProbeDisposition": {"as", "by", "pass", "proposal", "reason", "evidence", "finding", "rerun", "passing", "pending"},
	"Finding":          {"id", "owner", "raisedBy", "obligation", "observed", "blocking", "correction", "kind", "expected", "pass", "review", "candidate", "disputes", "closed", "closedPass", "closedReason", "closedAs", "closedEvidence", "subject", "definition", "receipts"},
	"TurnRecord":       {"id", "manager", "role", "pass", "kind", "mode", "context", "compacted", "usage", "stopped", "reports", "reason"},
}

func TestProjectionReadsOnlyKeysTheRunEngineRecords(t *testing.T) {
	produced, fields := producerContract(t)
	for kind, keys := range consumedKeys {
		got := produced[kind]
		if got == nil {
			t.Errorf("the run engine records no %s event with map data", kind)
			continue
		}
		for _, key := range keys {
			if !got[key] {
				t.Errorf("%s: the projection reads %q, which the run engine does not record (it records %v)", kind, key, sortedKeys(got))
			}
		}
	}
	for typ, names := range consumedFields {
		for _, name := range names {
			if !fields[typ][name] {
				t.Errorf("%s: the projection reads field %q, which the run engine's type does not have", typ, name)
			}
		}
	}
}

// producerContract parses the run engine's production source and returns, per
// recorded kind, the keys of its map data, and per struct type the JSON names
// of its fields.
func producerContract(t *testing.T) (map[string]map[string]bool, map[string]map[string]bool) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "run", "*.go"))
	if err != nil || len(files) == 0 {
		t.Fatalf("run engine source not found: %v", err)
	}
	produced := map[string]map[string]bool{}
	fields := map[string]map[string]bool{}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Body != nil {
					collectEvents(d.Body, produced)
				}
			case *ast.GenDecl:
				collectFields(d, fields)
			}
		}
	}
	return produced, fields
}

// collectEvents finds Event{Kind: ..., Data: ...} literals in one function.
// Map data is either a literal or a local variable built from a literal and
// later index assignments in the same function.
func collectEvents(body *ast.BlockStmt, produced map[string]map[string]bool) {
	vars := map[string]map[string]bool{}
	ast.Inspect(body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, lhs := range assign.Lhs {
			switch l := lhs.(type) {
			case *ast.Ident:
				if i < len(assign.Rhs) {
					if keys := mapKeys(assign.Rhs[i]); keys != nil {
						vars[l.Name] = keys
					}
				}
			case *ast.IndexExpr:
				if id, ok := l.X.(*ast.Ident); ok && vars[id.Name] != nil {
					if key := stringLit(l.Index); key != "" {
						vars[id.Name][key] = true
					}
				}
			}
		}
		return true
	})
	ast.Inspect(body, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		if id, ok := lit.Type.(*ast.Ident); !ok || id.Name != "Event" {
			return true
		}
		kind, keys := "", map[string]bool(nil)
		for _, elt := range lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			switch kv.Key.(*ast.Ident).Name {
			case "Kind":
				if kind = stringLit(kv.Value); kind == "" {
					kind = "*"
				}
			case "Data":
				if keys = mapKeys(kv.Value); keys == nil {
					if id, ok := kv.Value.(*ast.Ident); ok {
						keys = vars[id.Name]
					}
				}
			}
		}
		if kind != "" && keys != nil {
			if produced[kind] == nil {
				produced[kind] = map[string]bool{}
			}
			for k := range keys {
				produced[kind][k] = true
			}
		}
		return true
	})
}

func mapKeys(expr ast.Expr) map[string]bool {
	lit, ok := expr.(*ast.CompositeLit)
	if !ok {
		return nil
	}
	if _, ok := lit.Type.(*ast.MapType); !ok {
		return nil
	}
	keys := map[string]bool{}
	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if key := stringLit(kv.Key); key != "" {
				keys[key] = true
			}
		}
	}
	return keys
}

func stringLit(expr ast.Expr) string {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return ""
	}
	return s
}

func collectFields(decl *ast.GenDecl, fields map[string]map[string]bool) {
	for _, spec := range decl.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			continue
		}
		names := map[string]bool{}
		for _, f := range st.Fields.List {
			if f.Tag == nil {
				continue
			}
			tag, err := strconv.Unquote(f.Tag.Value)
			if err != nil {
				continue
			}
			if name, _, _ := strings.Cut(reflect.StructTag(tag).Get("json"), ","); name != "" {
				names[name] = true
			}
		}
		fields[ts.Name.Name] = names
	}
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// recordedKinds parses the run engine's and Apply's production source and
// returns every event kind they can record. A kind written as a literal is
// found directly. A kind held in a local variable is read from the kind-like
// literals of its function; one passed as a parameter, from the literal
// arguments of the calls to that function.
func recordedKinds(t *testing.T) map[string]bool {
	t.Helper()
	kinds := map[string]bool{}
	kindLike := regexp.MustCompile(`^[a-z]+(\.[a-z_]+)+$`)
	type dynamic struct {
		fn      string
		literal bool // held in a local variable rather than a parameter
		body    *ast.BlockStmt
	}
	var dyn []dynamic
	var files []*ast.File
	fset := token.NewFileSet()
	for _, dir := range []string{"run", "apply"} {
		names, err := filepath.Glob(filepath.Join("..", "..", dir, "*.go"))
		if err != nil || len(names) == 0 {
			t.Fatalf("%s source not found: %v", dir, err)
		}
		for _, name := range names {
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			file, err := parser.ParseFile(fset, name, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			files = append(files, file)
		}
	}
	isEvent := func(typ ast.Expr) bool {
		switch x := typ.(type) {
		case *ast.Ident:
			return x.Name == "Event"
		case *ast.SelectorExpr:
			return x.Sel.Name == "Event"
		}
		return false
	}
	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			params := map[string]bool{}
			for _, f := range fn.Type.Params.List {
				for _, n := range f.Names {
					params[n.Name] = true
				}
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				lit, ok := n.(*ast.CompositeLit)
				if !ok || !isEvent(lit.Type) {
					return true
				}
				for _, elt := range lit.Elts {
					kv, ok := elt.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					if key, ok := kv.Key.(*ast.Ident); !ok || key.Name != "Kind" {
						continue
					}
					if kind := stringLit(kv.Value); kind != "" {
						kinds[kind] = true
					} else if id, ok := kv.Value.(*ast.Ident); ok {
						dyn = append(dyn, dynamic{fn: fn.Name.Name, literal: !params[id.Name], body: fn.Body})
					}
				}
				return true
			})
		}
	}
	if len(kinds) == 0 {
		t.Fatal("no recorded event kind found in the run engine's source")
	}
	families := map[string]bool{}
	for kind := range kinds {
		family, _, _ := strings.Cut(kind, ".")
		families[family] = true
	}
	for _, d := range dyn {
		if d.literal {
			ast.Inspect(d.body, func(n ast.Node) bool {
				if kind := stringLit(asExpr(n)); kindLike.MatchString(kind) {
					if family, _, _ := strings.Cut(kind, "."); families[family] {
						kinds[kind] = true
					}
				}
				return true
			})
			continue
		}
		for _, file := range files {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				name := ""
				switch f := call.Fun.(type) {
				case *ast.Ident:
					name = f.Name
				case *ast.SelectorExpr:
					name = f.Sel.Name
				}
				if name != d.fn {
					return true
				}
				for _, arg := range call.Args {
					if kind := stringLit(arg); kindLike.MatchString(kind) {
						kinds[kind] = true
					}
				}
				return true
			})
		}
	}
	return kinds
}

func asExpr(n ast.Node) ast.Expr {
	e, _ := n.(ast.Expr)
	return e
}

// projected reports whether the projection interprets a record of this kind:
// it is neither unknown nor merely listed.
func projected(kind string) bool {
	p := newProjector("r")
	p.apply(Record{Seq: 1, Kind: kind, Data: []byte(`{}`)})
	return len(p.unknown) == 0 && !listedOnly[kind]
}

func TestEveryRecordedKindIsProjectedOrListedOnly(t *testing.T) {
	kinds := recordedKinds(t)
	for _, want := range []string{"run.start", "lookup.definition", "lookup.brief", "probe.requested", "probe.receipt", "probe.disposed", "basis.returned", "dependency.bound", "workspace.close"} {
		if !kinds[want] {
			t.Errorf("the source scan does not find %s: it would also miss other kinds", want)
		}
	}
	for kind := range kinds {
		switch {
		case projected(kind):
		case listedOnly[kind]:
		default:
			t.Errorf("the run engine records %s, which the projection neither interprets nor lists as bookkeeping; add it to apply or to listedOnly", kind)
		}
		if e := entry(Record{Seq: 1, Kind: kind, Data: []byte(`{}`)}); e.Class == "other" || e.Text == "" {
			t.Errorf("%s has no entry text of its own: %+v", kind, e)
		}
	}
}

// Kinds on the listed-only allowlist need not be producible by the current
// run engine: a record of an earlier build stays readable, and removing the
// producer (the per-Turn baseline drops the working environments of a Turn)
// must not fail this test. environment.release, the turn.start environment key
// and the turn.end instance key are such historical entries.
func TestListedOnlyKindsAndHistoricalKeysStayReadable(t *testing.T) {
	s := newScenario(t)
	s.add("run.start", "", "", "", map[string]any{"run": "r", "managers": []string{"."}})
	s.add("pass.start", "p1", "", "", map[string]any{"purpose": "implement", "review": false, "writes": true, "tree": "t"})
	s.add("turn.start", "p1", ".", "r-p1-root-m1", map[string]any{"role": "manager", "kind": "execute", "readOnly": false, "environment": "env-1", "mode": "work", "context": "brief"})
	s.add("turn.end", "p1", ".", "r-p1-root-m1", map[string]any{"role": "manager", "stopped": true, "reports": []string{}, "instance": map[string]any{"id": "x", "reused": true}})
	s.add("environment.release", "", "", "", map[string]any{"environment": "env-1", "instance": "x", "removed": true})
	records := s.records()
	v := Project("r", records, len(records), Source{}, nil)
	for _, g := range v.Gaps {
		if g.Code == "record.kind" || g.Code == "record.data" {
			t.Fatalf("a historical key or kind is no gap: %+v", g)
		}
	}
	if len(v.Turns) != 1 || v.Turns[0].Mode != "work" {
		t.Fatalf("turns = %+v", v.Turns)
	}
	for _, kind := range slices.Sorted(maps.Keys(listedOnly)) {
		if projected(kind) {
			t.Errorf("%s is listed only and also projected", kind)
		}
	}
	if !listedOnly["environment.release"] {
		t.Fatal("environment.release of earlier builds is listed only")
	}
}

// The projection reads report.json by the names the producer writes. The test
// compares the JSON names of the fields the cockpit decodes with the producer's
// report types, read from its source.
func TestReportDecodesOnlyFieldsTheProducerWrites(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "cli", "*.go"))
	if err != nil || len(files) == 0 {
		t.Fatalf("producer source not found: %v", err)
	}
	fields := map[string]map[string]bool{}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			if d, ok := decl.(*ast.GenDecl); ok {
				collectFields(d, fields)
			}
		}
	}
	jsonNames := func(typ reflect.Type) []string {
		var out []string
		for i := 0; i < typ.NumField(); i++ {
			if name, _, _ := strings.Cut(typ.Field(i).Tag.Get("json"), ","); name != "" {
				out = append(out, name)
			}
		}
		return out
	}
	for _, c := range []struct {
		decoded  reflect.Type
		producer string
	}{{reflect.TypeOf(rawReport{}), "ApplyReport"}, {reflect.TypeOf(rawRun{}), "RunReport"}} {
		if len(fields[c.producer]) == 0 {
			t.Fatalf("the producer's %s is not found", c.producer)
		}
		for _, name := range jsonNames(c.decoded) {
			if !fields[c.producer][name] {
				t.Errorf("the cockpit decodes %q, which the producer's %s does not write (it writes %v)", name, c.producer, sortedKeys(fields[c.producer]))
			}
		}
	}
	// A report of the producer's shape decodes into every part the view shows.
	r, err := decodeReport([]byte(`{"formatVersion":1,"operation":"apply","status":"passed","preparation":{"attempts":[{"head":"h"}]},"binding":{"strategy":"staged"},
		"run":{"id":"r","status":"verified","passes":[{"id":"p1","purpose":"assess","review":true,"sealed":true,"verdict":"satisfied","managers":[]}],"findings":[],"questions":[],"problems":[],"turns":[{"id":"t"}],"probes":[],"usage":{"costUSD":1}},
		"publication":{"ID":"r","Outcome":"no_change","Paths":null,"Reason":""},"retainedWork":"w","evidenceDirectory":"d"}`))
	if err != nil || r.Head != "h" || r.Run == nil || r.Run.Turns != 1 || len(r.Run.Passes) != 1 || r.Publication == nil || r.Retained != "w" || r.Run.Usage.CostUSD != 1 {
		t.Fatalf("report = %+v, %v", r, err)
	}
}
