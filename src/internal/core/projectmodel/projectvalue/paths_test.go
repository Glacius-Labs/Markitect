package projectvalue

import "testing"

func TestNativeSkillPathsHaveExactToolOwnership(t *testing.T) {
	for _, provider := range []string{"codex", "claude"} {
		paths := NativeSkillPaths(provider)
		if len(paths) != 1+2*len(NativeSkillNames()) {
			t.Fatalf("incomplete %s native paths", provider)
		}
		for _, path := range paths {
			if !IsToolPath(Config{}, path) {
				t.Fatalf("generated file lacks ownership: %s", path)
			}
		}
	}
	for _, path := range []string{".agents/skills/markitect-model-first/SKILL.md", ".claude/skills/markitect-model-first/SKILL.md"} {
		if IsToolPath(Config{}, path) {
			t.Fatalf("obsolete router path remains registered as an owned output: %s", path)
		}
	}
	for _, path := range []string{".agents/skills/custom/SKILL.md", ".agents/skills/markitect-implement/custom.md", ".claude/skills/markitect-check/private.md"} {
		if IsToolPath(Config{}, path) {
			t.Fatalf("unmanaged skill claimed by Markitect: %s", path)
		}
	}
}

func TestRecordSelectorsRetainSemanticOperationalOwners(t *testing.T) {
	wanted := map[string]string{".markitect/records/run/": "projectrun", ".markitect/records/exploration/": "projectexplore", ".markitect/records/briefing/": "projecthistory", ".markitect/records/brownfield/": "projectadoption"}
	for _, tool := range ToolPaths(Config{}) {
		if owner, ok := wanted[tool.Selector]; ok {
			if tool.Owner != owner || !tool.Operational {
				t.Fatalf("record selector semantics: %+v", tool)
			}
			delete(wanted, tool.Selector)
		}
	}
	if len(wanted) != 0 {
		t.Fatalf("missing record selectors: %v", wanted)
	}
	if IsToolPath(Config{}, ".markitect/records/unknown/record.json") {
		t.Fatal("arbitrary record kind claimed")
	}
}
