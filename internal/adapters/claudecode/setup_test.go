package claudecode

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/runtimeconfig"
	"go.yaml.in/yaml/v3"
)

func setupRuntimeProfile(engine string) RuntimeProfile {
	return RuntimeProfile{
		EngineExecutable: engine, ImageID: fixtureImage, AgentdVersion: "development", CredentialRef: "setup-token", MaxWireBytes: 1 << 20,
		Session: SessionProfile{
			Version: 1, Model: "claude-haiku-5-5", Effort: "high", SystemPrompt: "A generic: product prompt with \"quotes\" and ${NO_EXPANSION}.",
			BuiltinTools: []string{"Read", "Write", "Edit", "Glob", "Grep", "Bash"}, Store: "setup-store", CPUs: 2, MemoryBytes: 4 << 30, Pids: 1024, TmpBytes: 1 << 30, Network: "",
		},
	}
}

func TestRuntimeSectionRoundTripsThroughTheMetadataDecoderAndProjection(t *testing.T) {
	want := setupRuntimeProfile(filepath.Join(t.TempDir(), "engine"))
	section, err := want.Section()
	if err != nil {
		t.Fatal(err)
	}
	text, err := yaml.Marshal(section)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := yaml.Marshal(section)
	if string(text) != string(again) || !strings.Contains(string(text), "instructions: []\n") {
		t.Fatalf("encoding is not deterministic or lost the empty instruction list:\n%s", text)
	}
	var document yaml.Node
	if err := yaml.Unmarshal(text, &document); err != nil {
		t.Fatal(err)
	}
	p := preparation{out: runtimeconfig.ProfileObservation{Status: model.Passed}}
	config, ok := p.decode(document.Content[0])
	if !ok || p.out.Status != model.Passed {
		t.Fatalf("the decoder refused the encoding: %+v\n%s", p.out.Diagnostics, text)
	}
	if config.engine != want.EngineExecutable || config.imageID != want.ImageID || config.agentd != want.AgentdVersion || config.credentialRef != want.CredentialRef || config.maxWireBytes != want.MaxWireBytes || len(config.instructions) != 0 || !reflect.DeepEqual(config.session, want.Session) {
		t.Fatalf("decoded %+v\nwant %+v", config, want)
	}
	// The same bytes pass the complete preparation and project to a Profile.
	fake := &fakeImages{data: []byte(fixtureMetadata)}
	prepared := preparedExecution(t, string(text), imagesPreparer(fake))
	profile, err := ProfileFromPrepared(prepared)
	if err != nil || profile.Session.Model != "claude-haiku-5-5" || profile.CredentialRef != "setup-token" || !reflect.DeepEqual(profile.Session, want.Session) {
		t.Fatalf("%v %+v", err, profile)
	}
}

func TestRuntimeSectionRefusesInvalidProfiles(t *testing.T) {
	engine := filepath.Join(t.TempDir(), "engine")
	for name, change := range map[string]func(*RuntimeProfile){
		"effort":     func(p *RuntimeProfile) { p.Session.Effort = "max" },
		"image":      func(p *RuntimeProfile) { p.ImageID = "latest" },
		"model":      func(p *RuntimeProfile) { p.Session.Model = " " },
		"store":      func(p *RuntimeProfile) { p.Session.Store = "not a volume" },
		"tool":       func(p *RuntimeProfile) { p.Session.BuiltinTools = []string{"Task"} },
		"credential": func(p *RuntimeProfile) { p.CredentialRef = "" },
		"agentd":     func(p *RuntimeProfile) { p.AgentdVersion = "" },
		"wire":       func(p *RuntimeProfile) { p.MaxWireBytes = 0 },
	} {
		p := setupRuntimeProfile(engine)
		change(&p)
		if _, err := p.Section(); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestObserveRuntimeBindsImageAndRefusesUnsupportedMetadata(t *testing.T) {
	got, err := ObserveRuntime(context.Background(), &fakeImages{data: []byte(fixtureMetadata)}, "markitect-runtime:tag")
	if err != nil || got != (RuntimeObservation{ImageID: fixtureImage, AgentdVersion: "development", CLIVersion: supportedClaudeVersion}) {
		t.Fatalf("%v %+v", err, got)
	}
	for name, images := range map[string]*fakeImages{
		"old CLI":       {data: []byte(strings.Replace(fixtureMetadata, "2.1.301", "2.1.300", 1))},
		"malformed":     {data: []byte("claude=x\n")},
		"inspect error": {data: []byte(fixtureMetadata), inspectErr: errors.New("no engine")},
		"read error":    {data: []byte(fixtureMetadata), readErr: errors.New("no file")},
		"changed":       {data: []byte(fixtureMetadata), ids: []string{fixtureImage, "sha256:" + strings.Repeat("c", 64)}},
		"not an id":     {data: []byte(fixtureMetadata), ids: []string{"latest"}},
	} {
		if _, err := ObserveRuntime(context.Background(), images, "ref"); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	if _, err := ObserveRuntime(context.Background(), nil, "ref"); err == nil {
		t.Error("nil reader accepted")
	}
}
