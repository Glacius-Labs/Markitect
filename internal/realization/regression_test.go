package realization

import (
	"testing"
)

func TestPlainFileCellIsReadAsWritten(t *testing.T) {
	reg, ds := account(t, []string{"pkg/__init__.py"}, nil,
		page("", table+"| pkg/__init__.py | [[project]] | Package marker |\n"))
	if len(ds) != 0 || reg.Entries["pkg/__init__.py"] == nil {
		t.Fatalf("entries = %v, diagnostics = %v", reg.Entries, ds)
	}
}

func TestPlainFileCellAtTheRootKeepsItsMarkup(t *testing.T) {
	reg, ds := account(t, []string{"__init__.py"}, nil,
		page("", table+"| __init__.py | [[project]] | Package marker |\n"))
	if len(ds) != 0 || reg.Entries["__init__.py"] == nil {
		t.Fatalf("entries = %v, diagnostics = %v", reg.Entries, ds)
	}
}

func TestUnescapedLabelBarInRegistryRow(t *testing.T) {
	_, ds := account(t, []string{"a.go"}, nil,
		page("", table+"| a.go | [[members/member|member]] | lost cell |\n"))
	expect(t, ds, "registry.table", `.markitect/registry.md:3: registry.table: a link label inside a table needs its bar escaped`)

	reg, ds := account(t, []string{"a.go"}, nil,
		page("", table+"| a.go | [[members/member\\|member]] | escaped |\n"))
	if len(ds) != 0 || len(reg.Entries["a.go"].Realizes) != 1 {
		t.Fatalf("escaped bar: entries = %+v, diagnostics = %v", reg.Entries["a.go"], ds)
	}
}

func TestPatternErrorsNameTheirLine(t *testing.T) {
	_, ds := account(t, nil, nil, page("members", "---\npaths:\n  - src/members/**\n  - /abs\n---\n"))
	expect(t, ds, "registry.pattern", ".markitect/registry/members.md:4:")
}

func TestStaleFileOfARemovedPackageGoesToRoot(t *testing.T) {
	reg, _ := account(t, nil, nil,
		page("gone", "---\npaths: [src/gone/**]\n---\n"+table+"| src/gone/missing.go | | the Package and the file are gone |\n"))
	for _, s := range reg.Stale {
		if s.Code == "stale.file" && s.Package != "" {
			t.Fatalf("stale file assigned to the removed Package %q", s.Package)
		}
	}
}
