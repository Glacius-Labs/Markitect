package cockpit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// The projection decodes these data keys of the events the run engine records
// with map data. The test reads the engine's source and fails when one of them
// is no longer produced for its kind, so a renamed key cannot silently empty a
// view. Kinds the engine records with a non-literal kind share the key "*".
var consumedKeys = map[string][]string{
	"run.start":             {"run", "managers", "start", "agentTurns", "passes", "deadline"},
	"pass.start":            {"purpose", "review", "writes", "tree"},
	"manager.state_changed": {"from", "to", "stage", "why", "waitingFor"},
	"manager.release":       {"why", "parent"},
	"turn.start":            {"role", "kind", "profile", "readOnly", "resume"},
	"turn.end":              {"role", "kind", "usage", "stopped", "reports", "reason"},
	"proposal.submitted":    {"proposal", "kind", "purpose", "envelope", "findings", "closed", "disputed"},
	"proposal.checked":      {"proposal", "passed"},
	"proposal.invalidated":  {"proposal", "why"},
	"proposal.withdrawn":    {"proposal", "why"},
	"basis.invalidated":     {"proposal", "reason"},
	"result.reopened":       {"proposal", "by", "reason"},
	"*":                     {"proposal", "by", "stand", "reason"},
	"report.wait":           {"for", "reason"},
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
	"envelope":   {"summary", "verdict", "children", "plan"},
	"plan":       {"dependencies"},
	"dependency": {"owner", "need", "pass", "stage"},
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
