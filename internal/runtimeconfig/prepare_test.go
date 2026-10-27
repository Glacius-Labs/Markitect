package runtimeconfig

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type checkPeer func(context.Context, ProfileInput) (ProfileObservation, error)

func (f checkPeer) PrepareCheckProfile(ctx context.Context, p ProfileInput) (ProfileObservation, error) {
	return f(ctx, p)
}
func runtimeInputs(t *testing.T, document string) project.RawInputs {
	t.Helper()
	p, err := snapshot.NewPath("runtime.yaml")
	if err != nil {
		t.Fatal(err)
	}
	e, err := snapshot.NewEntry(p, snapshot.Regular, []byte(document))
	if err != nil {
		t.Fatal(err)
	}
	return project.RawInputs{RuntimeProfiles: []project.SourceFile{{Location: project.SourceLocation{Boundary: "explicit-runtime", Path: p}, File: e}}}
}

const checkEnvelope = "apiVersion: markitect.runtime/v1\nagentProfiles: {}\ncheckProfiles:\n  tests:\n    container: {profileVersion: 1}\n"

func observedCheck(p ProfileInput) ProfileObservation {
	path, _ := snapshot.NewPath("observed-file")
	e, _ := snapshot.NewEntry(path, snapshot.Regular, []byte("observed image executable"))
	return ProfileObservation{InputBinding: p.Binding(), Status: model.Passed, Contract: ProfileContract{Name: "test-image-check", Version: 1}, CheckImage: &CheckImageObservation{ImageID: "sha256:" + strings.Repeat("a", 64), Executable: AgentImageMetadataFile{Path: "/usr/bin/check", ContentID: e.ContentID(), Size: int64(len(e.Bytes()))}}}
}
func TestPrepareRejectsInvalidEnvelopesBeforeAdapters(t *testing.T) {
	for _, doc := range []string{"", strings.Replace(checkEnvelope, "markitect.runtime/v1", "old/v1", 1), checkEnvelope + "unknown: true\n", strings.Replace(checkEnvelope, "container:", "process:", 1), strings.Replace(checkEnvelope, "profileVersion: 1", "profileVersion: 1, profileVersion: 2", 1), strings.Replace(checkEnvelope, "profileVersion: 1", "profileVersion: &x 1, alias: *x", 1), checkEnvelope + "---\n{}\n"} {
		calls := 0
		r, err := Prepare(context.Background(), runtimeInputs(t, doc), nil, checkPeer(func(context.Context, ProfileInput) (ProfileObservation, error) {
			calls++
			return ProfileObservation{}, nil
		}))
		if err != nil || r.Status == model.Passed || calls != 0 || len(r.Profiles) != 0 {
			t.Fatalf("status=%s calls=%d error=%v", r.Status, calls, err)
		}
	}
}

type agentPeer func(context.Context, ProfileInput) (ProfileObservation, error)

func (f agentPeer) PrepareAgentProfile(ctx context.Context, p ProfileInput) (ProfileObservation, error) {
	return f(ctx, p)
}

// A profile selects exactly one Agent adapter: codexAppServer or claudeCode.
// Both sections in one profile are refused before any port runs.
func TestPrepareSelectsExactlyOneAgentSection(t *testing.T) {
	for _, tc := range []struct {
		name, entry, section string
	}{
		{"codex", "    codexAppServer: {profileVersion: 3}\n", "codexAppServer"},
		{"claude", "    claudeCode: {profileVersion: 1}\n", "claudeCode"},
		{"both", "    codexAppServer: {profileVersion: 3}\n    claudeCode: {profileVersion: 1}\n", ""},
		{"neither", "    other: {profileVersion: 1}\n", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := "apiVersion: markitect.runtime/v1\nagentProfiles:\n  markitect-agent:\n" + tc.entry + "checkProfiles: {}\n"
			var sections []string
			r, err := Prepare(context.Background(), runtimeInputs(t, doc), agentPeer(func(_ context.Context, p ProfileInput) (ProfileObservation, error) {
				sections = append(sections, p.Section())
				return ProfileObservation{InputBinding: p.Binding(), Status: model.NotCompleted}, nil
			}), nil)
			if err != nil {
				t.Fatal(err)
			}
			if tc.section != "" {
				if len(sections) != 1 || sections[0] != tc.section || len(r.Profiles) != 1 {
					t.Fatalf("section %s did not reach the Agent port alone: %v %+v", tc.section, sections, r.Diagnostics)
				}
				return
			}
			if r.Status != model.Failed || len(sections) != 0 || len(r.Profiles) != 0 {
				t.Fatalf("status=%s sections=%v profiles=%d", r.Status, sections, len(r.Profiles))
			}
			found := map[string]string{}
			for _, d := range r.Diagnostics {
				found[d.Code+" "+d.Source.Field] = d.Message
			}
			if got := found["runtime.section /agentProfiles/markitect-agent"]; !strings.HasSuffix(got, "adapter section: codexAppServer or claudeCode.") {
				t.Fatalf("section refusal differs: %v", r.Diagnostics)
			}
		})
	}
}

// Each Agent section binds its own image provider; Claude Code evidence
// carries no Codex schema digest.
func TestPrepareBindsAgentImageProviderToSection(t *testing.T) {
	path, _ := snapshot.NewPath("runtime.env")
	metadata, _ := snapshot.NewEntry(path, snapshot.Regular, []byte("claude=x\nagentd=y\n"))
	image := func(provider AgentImageProvider, schema string) *AgentImageObservation {
		return &AgentImageObservation{Provider: provider, ImageID: "sha256:" + strings.Repeat("b", 64), Metadata: AgentImageMetadataFile{Path: "/opt/markitect/runtime.env", ContentID: metadata.ContentID(), Size: 18}, CLIVersion: "2.1.308 (Claude Code)", AgentdVersion: "y", RecordedCodexSchemaDigest: schema}
	}
	for _, tc := range []struct {
		name  string
		image *AgentImageObservation
		pass  bool
	}{
		{"claude evidence", image(AgentImageClaudeCode, ""), true},
		{"codex provider in a claude section", image(AgentImageCodexAppServer, "sha256:"+strings.Repeat("c", 64)), false},
		{"claude evidence with a codex schema", image(AgentImageClaudeCode, "sha256:"+strings.Repeat("c", 64)), false},
		{"unknown provider", image("other", ""), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := "apiVersion: markitect.runtime/v1\nagentProfiles:\n  markitect-agent:\n    claudeCode: {profileVersion: 1}\ncheckProfiles: {}\n"
			r, err := Prepare(context.Background(), runtimeInputs(t, doc), agentPeer(func(_ context.Context, p ProfileInput) (ProfileObservation, error) {
				return ProfileObservation{InputBinding: p.Binding(), Status: model.Passed, Contract: ProfileContract{Name: "markitect.claude-code.image-metadata", Version: 1}, AgentImage: tc.image}, nil
			}), nil)
			if err != nil || len(r.Profiles) != 1 {
				t.Fatal(err, r.Diagnostics)
			}
			if _, ok := r.Profiles[0].Identity(); ok != tc.pass {
				t.Fatalf("identity=%v, want %v: %+v", ok, tc.pass, r.Profiles[0].Observation().Diagnostics)
			}
		})
	}
}

func TestPrepareBindsExactInputsAndOwnsObservations(t *testing.T) {
	var portObservation ProfileObservation
	r, err := Prepare(context.Background(), runtimeInputs(t, checkEnvelope), nil, checkPeer(func(_ context.Context, p ProfileInput) (ProfileObservation, error) {
		portObservation = observedCheck(p)
		return portObservation, nil
	}))
	if err != nil || r.Status != model.Passed || len(r.Profiles) != 1 {
		t.Fatal(r, err)
	}
	p := r.Profiles[0]
	id, ok := p.Identity()
	if !ok {
		t.Fatal("complete static identity absent")
	}
	portObservation.CheckImage.ImageID = "changed"
	o := p.Observation()
	o.CheckImage.Executable.Path = "changed"
	node := p.Input().Inline()
	node.Content[0].Value = "changed"
	if p.Observation().CheckImage.ImageID == "changed" || p.Observation().CheckImage.Executable.Path == "changed" || p.Input().Inline().Content[0].Value == "changed" {
		t.Fatal("retained static inputs alias mutable port data")
	}
	r2, _ := Prepare(context.Background(), runtimeInputs(t, strings.Replace(checkEnvelope, "profileVersion: 1", "profileVersion: 2", 1)), nil, checkPeer(func(_ context.Context, p ProfileInput) (ProfileObservation, error) { return observedCheck(p), nil }))
	id2, ok := r2.Profiles[0].Identity()
	if !ok || id == id2 {
		t.Fatal("changed captured configuration reused static identity")
	}
}
func TestPrepareRefusesMissingMixedOrMismatchedEvidenceAndRetainsFailure(t *testing.T) {
	failure := errors.New("controlled image read failed")
	for _, kind := range []string{"missing binding", "different binding", "missing image", "mixed image", "missing contract", "uncaptured instruction", "required instruction", "unknown status", "partial error"} {
		t.Run(kind, func(t *testing.T) {
			r, err := Prepare(context.Background(), runtimeInputs(t, checkEnvelope), nil, checkPeer(func(_ context.Context, p ProfileInput) (ProfileObservation, error) {
				o := observedCheck(p)
				switch kind {
				case "missing binding":
					o.InputBinding = ProfileBinding{}
				case "different binding":
					q := p
					q.name = "different"
					o.InputBinding = q.Binding()
				case "missing image":
					o.CheckImage = nil
				case "mixed image":
					o.AgentImage = &AgentImageObservation{}
				case "missing contract":
					o.Contract = ProfileContract{}
				case "uncaptured instruction", "required instruction":
					path, _ := snapshot.NewPath("not-acquired.md")
					ref := project.SourceLocation{Boundary: "outside", Path: path}
					if kind == "required instruction" {
						o.RequiredInstructions = []project.SourceLocation{ref}
					} else {
						o.Instructions = []project.SourceLocation{ref}
					}
				case "unknown status":
					o.Status = "invented"
				case "partial error":
					return o, failure
				}
				return o, nil
			}))
			if r.Status == model.Passed || len(r.Profiles) != 1 {
				t.Fatal(r, err)
			}
			if _, ok := r.Profiles[0].Identity(); ok {
				t.Fatal("incomplete or invalid observation gained identity")
			}
			if kind == "partial error" && !errors.Is(err, failure) {
				t.Fatal("actual failure lost", err)
			}
		})
	}
}
