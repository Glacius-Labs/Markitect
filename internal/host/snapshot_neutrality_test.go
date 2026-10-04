package host

import (
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
)

const (
	neutralityTextPath     = "docs/general/text/behavior.yaml"
	neutralityWorkflowPath = "docs/general/workflows/review.yaml"
	neutralitySourcePath   = "docs/general/src/worker.go"
	neutralityEntry        = "sample/Workflow/review"
)

// This exercises the application boundary with caller-owned snapshots. It
// deliberately does not load a working tree or ask Git to identify a source.
func TestApplicationAcceptsOpaqueSnapshotsWithoutSourceCoupling(t *testing.T) {
	baseSnapshot := neutralityFixture(t, "fixture:base", "attemptLimit = 3\n")
	candidateSnapshot := neutralityFixture(t, "fixture:candidate", "attemptLimit = 5\n")

	base, err := Parse(baseSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := Parse(candidateSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	for name, project := range map[string]*Project{"base": base, "candidate": candidate} {
		if len(project.Diagnostics) != 0 {
			t.Errorf("%s snapshot has diagnostics: %#v", name, project.Diagnostics)
		}
	}

	ctx, err := CompileContext(candidate, neutralityEntry, "test-tool")
	if err != nil {
		t.Fatal(err)
	}
	var sourceInput *ContextInput
	for i := range ctx.Inputs {
		if ctx.Inputs[i].Key == "file:"+neutralitySourcePath {
			sourceInput = &ctx.Inputs[i]
			break
		}
	}
	if sourceInput == nil {
		t.Fatalf("context omitted declared source file %s: %#v", neutralitySourcePath, ctx.Inputs)
	}
	if sourceInput.Text != "attemptLimit = 5\n" || sourceInput.Hash != Hash(candidateSnapshot.Files[neutralitySourcePath]) {
		t.Fatalf("context source input = %#v, want candidate snapshot bytes and hash", sourceInput)
	}
	for _, key := range []string{"sample/Text/behavior", "sample/Rule/policy", neutralityEntry} {
		if !contextHasInput(ctx, key) {
			t.Errorf("context omitted dependency %s", key)
		}
	}
	if contextHasInput(ctx, "/Project/sample-project") == false {
		t.Error("context omitted Project policy")
	}

	impact := Changes(base, candidate)
	if !reflect.DeepEqual(impact.Changed, []string{neutralitySourcePath}) {
		t.Fatalf("changed paths = %v, want only declared source %s", impact.Changed, neutralitySourcePath)
	}
	wantAffected := []string{"sample/Text/behavior", neutralityEntry}
	if !reflect.DeepEqual(impact.Affected, wantAffected) {
		t.Fatalf("affected = %v, want dependency closure %v", impact.Affected, wantAffected)
	}

	unknownSnapshot := neutralityFixture(t, "fixture:unknown", "attemptLimit = 5\n")
	unknownSnapshot.Files["docs/general/notes.md"] = []byte("unmodelled input\n")
	unknownSnapshot.Modes["docs/general/notes.md"] = unknownSnapshot.Modes[neutralitySourcePath]
	unknown, err := Parse(unknownSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	broad := Changes(candidate, unknown)
	wantBroad := []string{"/Project/sample-project", "sample/Rule/policy", "sample/Text/behavior", neutralityEntry}
	if !reflect.DeepEqual(broad.Affected, wantBroad) {
		t.Fatalf("unknown file change affected %v, want conservative project-wide invalidation %v", broad.Affected, wantBroad)
	}

	sameContentDifferentIdentity := neutralityFixture(t, "fixture:other-identity", "attemptLimit = 5\n")
	identityOnly, err := Parse(sameContentDifferentIdentity)
	if err != nil {
		t.Fatal(err)
	}
	identityImpact := Changes(candidate, identityOnly)
	if len(identityImpact.Changed) != 0 || len(identityImpact.Affected) != 0 {
		t.Fatalf("identity-only change produced impact: %#v", identityImpact)
	}
	identityContext, err := CompileContext(identityOnly, neutralityEntry, "test-tool")
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Digest != identityContext.Digest || ctx.SnapshotDigest != identityContext.SnapshotDigest {
		t.Fatalf("source identity changed content digests: context (%s, %s), identity-only (%s, %s)", ctx.Digest, ctx.SnapshotDigest, identityContext.Digest, identityContext.SnapshotDigest)
	}
}

func neutralityFixture(t *testing.T, id, sourceText string) *snapshot.Snapshot {
	t.Helper()
	snap := fixtureFiles(t, id, "Keep the owner source.", projectNS)
	delete(snap.Files, skillPath)
	delete(snap.Modes, skillPath)

	text := authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion,
		Kind:     "Text",
		Metadata: core.Metadata{Name: "behavior", Namespace: projectNS}}, Spec: authoring.Spec{Text: "The worker has an attempt limit.", Files: []string{neutralitySourcePath}},
	}
	workflow := authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion,
		Kind:     "Workflow",
		Metadata: core.Metadata{Name: "review", Namespace: projectNS}}, Spec: authoring.Spec{
		Text:  "Review the implementation against its documented behavior.",
		Uses:  []core.Ref{{Kind: "Text", Name: "behavior"}},
		Rules: []core.Ref{{Kind: "Rule", Name: "policy"}},
	},
	}
	snap.Files[neutralityTextPath] = encodeResource(t, text)
	snap.Files[neutralityWorkflowPath] = encodeResource(t, workflow)
	snap.Files[neutralitySourcePath] = []byte(sourceText)
	snap.Modes[neutralityTextPath] = snap.Modes[rulePath]
	snap.Modes[neutralityWorkflowPath] = snap.Modes[rulePath]
	snap.Modes[neutralitySourcePath] = snap.Modes[rulePath]
	return snap
}

func contextHasInput(context *Context, key string) bool {
	for _, input := range context.Inputs {
		if input.Key == key {
			return true
		}
	}
	return false
}
