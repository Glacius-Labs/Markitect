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
