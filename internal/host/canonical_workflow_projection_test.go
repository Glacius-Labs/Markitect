package host

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/internal/modules/githooks"
)

func gitHooksRequest() canonical.ProjectionRequest {
	const api = "example.test/v1"
	definition := core.Definition{APIVersion: api, Kind: "Build", Metadata: core.Metadata{Namespace: "delivery", Name: "local"}, Purpose: "A locally verified build.", Spec: map[string]any{}}
	policy := core.Definition{
		APIVersion: "markitect.foundation/v1", Kind: "ProjectionPolicy", Metadata: core.Metadata{Namespace: "delivery", Name: "build-to-hooks"},
		Purpose: "Run required local checks.", Spec: map[string]any{
			"sourceKind": map[string]any{"apiVersion": api, "kind": "Build"}, "targetTechnology": "githooks", "guidance": "Keep the check active.",
		},
	}
	return canonical.ProjectionRequest{
		RequestDigest: "sha256:" + strings.Repeat("1", 64),
		Definitions:   []core.Definition{definition},
		Schemas:       []core.Schema{{APIVersion: api, Purpose: "Delivery contracts.", Kinds: map[string]core.Kind{"Build": {Purpose: "A build verification target."}}}},
		Policies:      []core.Definition{policy}, TargetPrefix: ".githooks",
		Projector: canonical.ProjectorRegistration{ID: githooks.ProjectorID, Version: "1.0.0", Target: githooks.TargetTechnology, AllowedRoots: []string{".githooks"}, RequiredChecks: []string{"canonical-workflow-check"}},
	}
}

func TestPrepareGitHooksProjectionInputUsesOnlyNamedProjectCheckAndScopedInventory(t *testing.T) {
	request := gitHooksRequest()
	checks := []authoring.Check{
		{Name: "canonical-workflow-check", Run: []string{"git", "diff", "--cached", "--check", "file with spaces.txt"}},
		{Name: "unselected", Run: []string{"sh", "-c", "exit 1"}},
	}
	inventory := []source.WorkingFileMetadata{
		{Path: ".githooks/other", Mode: snapshot.RegularMode},
		{Path: "docs/README.md", Mode: snapshot.RegularMode},
	}
	observed := &snapshot.Snapshot{Files: map[string][]byte{
		".githooks/other": []byte("observed"), "docs/README.md": []byte("outside target"),
	}}
	previous := &records.ProjectionRecord{RequestDigest: "sha256:" + strings.Repeat("2", 64), State: records.StateMaterializedUnverified, Artifacts: []records.Artifact{{Path: ".githooks/pre-commit", Digest: "sha256:" + strings.Repeat("3", 64), Mode: snapshot.ExecutableMode}}}
	input, escalations, err := prepareGitHooksProjectionInput(request, checks, inventory, observed, previous)
	if err != nil {
		t.Fatal(err)
	}
	if len(escalations) != 0 || !input.InventoryComplete || input.RequestDigest != request.RequestDigest {
		t.Fatalf("input scope/escalations = %#v / %#v", input, escalations)
	}
	if len(input.RequiredChecks) != 1 || input.RequiredChecks[0].Name != "canonical-workflow-check" || strings.Join(input.RequiredChecks[0].Argv, "\x00") != strings.Join(checks[0].Run, "\x00") {
		t.Fatalf("required checks = %#v", input.RequiredChecks)
	}
	if len(input.ObservedArtifacts) != 1 || input.ObservedArtifacts[0].Path != ".githooks/other" || string(input.ObservedArtifacts[0].Bytes) != "observed" {
		t.Fatalf("observed artifacts = %#v", input.ObservedArtifacts)
	}
	if input.Previous == nil || !input.Previous.Complete || len(input.Previous.Artifacts) != 1 || input.Previous.Artifacts[0].Mode != snapshot.ExecutableMode || input.Verification != nil {
		t.Fatalf("prior/verification binding = %#v", input)
	}
}

func TestProposeGitHooksProjectionCarriesExecutableOutputMode(t *testing.T) {
	checks := []authoring.Check{{Name: "canonical-workflow-check", Run: []string{"git", "diff", "--cached", "--check", "--", "file with spaces.txt"}}}
	proposal, err := proposeGitHooksProjection(gitHooksRequest(), checks, nil, &snapshot.Snapshot{Files: map[string][]byte{}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if proposal.Decision != "work" || len(proposal.Outputs) != 1 || len(proposal.OutputModes) != 1 {
		t.Fatalf("proposal = %#v", proposal)
	}
	const path = ".githooks/pre-commit"
	if proposal.OutputModes[path] != snapshot.ExecutableMode || !strings.Contains(string(proposal.Outputs[path]), "'file with spaces.txt'") {
		t.Fatalf("output mode or exact argv missing: modes=%#v output=%s", proposal.OutputModes, proposal.Outputs[path])
	}
}

func TestCanonicalWorkflowEntrypointDoesNotTreatUnknownTargetAsMarkdown(t *testing.T) {
	request := gitHooksRequest()
	request.Projector = canonical.ProjectorRegistration{ID: "other-projector", Version: "1.0.0", Target: "markdown"}
	if entrypoint, err := canonicalWorkflowEntrypoint(request); err == nil {
		t.Fatalf("unknown projector resolved to %q", entrypoint)
	}
	request.Projector = canonical.ProjectorRegistration{ID: "markdown-documentation", Version: "1.0.0", Target: "other"}
	if entrypoint, err := canonicalWorkflowEntrypoint(request); err == nil {
		t.Fatalf("unknown target resolved to %q", entrypoint)
	}
}
