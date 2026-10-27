package runner

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

const hostEngine = `C:\Program Files\Docker\Docker\resources\bin\docker.exe`

func exampleProfile(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "examples", "runtime-profile.example.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	quoted := "'" + hostEngine + "'"
	return []byte(strings.ReplaceAll(string(data), "ENGINE_EXECUTABLE", quoted))
}

func decodeYAML(t *testing.T, data []byte) any {
	t.Helper()
	var value any
	if err := yaml.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestTranslateRuntimeProfileNamesTheRunnerEngineOnly(t *testing.T) {
	original := exampleProfile(t)
	translated, err := TranslateRuntimeProfile(original, hostEngine)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(translated), "docker.exe") || strings.Count(string(translated), EngineBinary) != 2 {
		t.Fatalf("both engineExecutable values must name the runner engine:\n%s", translated)
	}
	// Apart from the engine, the decoded profile is unchanged.
	want := decodeYAML(t, []byte(strings.ReplaceAll(string(original), "'"+hostEngine+"'", EngineBinary)))
	if got := decodeYAML(t, translated); !reflect.DeepEqual(got, want) {
		t.Fatalf("profile changed beyond the engine:\n got %#v\nwant %#v", got, want)
	}
}

func TestTranslateRuntimeProfileAcceptsRepositoryInstructions(t *testing.T) {
	profile := strings.Replace(string(exampleProfile(t)), "instructions: []", "instructions:\n        - boundary: .\n          path: AGENTS.md\n        - boundary: docs/agents\n          path: rules.md", 1)
	translated, err := TranslateRuntimeProfile([]byte(profile), hostEngine)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(translated), "docs/agents") {
		t.Fatalf("instructions lost:\n%s", translated)
	}
}

func TestTranslateRuntimeProfileRefusesWhatTheRunnerCannotCarry(t *testing.T) {
	profile := string(exampleProfile(t))
	for name, data := range map[string]string{
		"other engine":         strings.Replace(profile, hostEngine, `C:\docker\docker.exe`, 1),
		"external instruction": strings.Replace(profile, "instructions: []", "instructions:\n        - boundary: 'C:\\Users\\me\\rules'\n          path: rules.md", 1),
		"escaping instruction": strings.Replace(profile, "instructions: []", "instructions:\n        - boundary: ../rules\n          path: rules.md", 1),
		"alias":                strings.Replace(profile, "markitect-agent:", "markitect-agent: &agent", 1),
		"two documents":        profile + "\n---\napiVersion: markitect.runtime/v1\n",
		"no engine":            "apiVersion: markitect.runtime/v1\n",
		"empty":                "",
		"invalid":              "a: [",
	} {
		if _, err := TranslateRuntimeProfile([]byte(data), hostEngine); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
