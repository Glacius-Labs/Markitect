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
	"github.com/Glacius-Labs/Markitect/internal/modules/azurepipelines"
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

func azurePipelinesRequest() canonical.ProjectionRequest {
	request := gitHooksRequest()
	request.TargetPrefix = "pipelines"
	request.Projector = canonical.ProjectorRegistration{
		ID: azurepipelines.ProjectorID, Version: azurepipelines.ModuleVersion, Target: azurepipelines.TargetTechnology,
		AllowedRoots: []string{"pipelines"}, RequiredChecks: []string{azurepipelines.RequiredCheck},
	}
	request.Policies[0].Spec["targetTechnology"] = azurepipelines.TargetTechnology
	return request
}

func TestPrepareAzurePipelinesProjectionInputUsesExactSelectedCheckAndScopedInventory(t *testing.T) {
	checks := []authoring.Check{
		{Name: azurepipelines.RequiredCheck, Run: []string{"go", "test", "./..."}},
		{Name: "unselected", Run: []string{"go", "run", "./cmd/ignored"}},
	}
	inventory := []source.WorkingFileMetadata{{Path: "pipelines/notes.txt", Mode: snapshot.RegularMode}, {Path: "docs/README.md", Mode: snapshot.RegularMode}}
	observed := &snapshot.Snapshot{Files: map[string][]byte{"pipelines/notes.txt": []byte("seen"), "docs/README.md": []byte("outside")}}
	input, escalations, err := prepareAzurePipelinesProjectionInput(azurePipelinesRequest(), checks, inventory, observed, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(escalations) != 0 || !input.InventoryComplete || !input.CanonicalAffected || len(input.Checks) != 1 {
		t.Fatalf("input scope/checks = %#v, escalations=%#v", input, escalations)
	}
	if input.Checks[0].Name != azurepipelines.RequiredCheck || strings.Join(input.Checks[0].Argv, "\x00") != strings.Join(checks[0].Run, "\x00") {
		t.Fatalf("selected Azure checks = %#v", input.Checks)
	}
	if len(input.ObservedArtifacts) != 1 || input.ObservedArtifacts[0].Path != "pipelines/notes.txt" || string(input.ObservedArtifacts[0].Bytes) != "seen" {
		t.Fatalf("observed target inventory = %#v", input.ObservedArtifacts)
	}
}

func TestProposeAzurePipelinesProjectionCarriesRegularOutputMode(t *testing.T) {
	checks := []authoring.Check{{Name: azurepipelines.RequiredCheck, Run: []string{"go", "test", "./..."}}}
	proposal, err := proposeAzurePipelinesProjection(azurePipelinesRequest(), checks, nil, &snapshot.Snapshot{Files: map[string][]byte{}}, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	const path = "pipelines/azure-pipelines.yml"
	if proposal.Decision != "work" || len(proposal.Outputs) != 1 || proposal.OutputModes[path] != snapshot.RegularMode {
		t.Fatalf("Azure proposal = %#v", proposal)
	}
	if !strings.Contains(string(proposal.Outputs[path]), "displayName: 'canonical-workflow-check'") || !strings.Contains(string(proposal.Outputs[path]), "script: 'go test ./...'") {
		t.Fatalf("pipeline omitted configured check: %s", proposal.Outputs[path])
	}
}

func TestPrepareAdaptersFailClosedWhenSelectedInventoryBytesAreUnavailable(t *testing.T) {
	_, _, err := prepareGitHooksProjectionInput(gitHooksRequest(), []authoring.Check{{Name: "canonical-workflow-check", Run: []string{"git", "diff", "--check"}}}, []source.WorkingFileMetadata{{Path: ".githooks/pre-commit", Mode: snapshot.ExecutableMode}}, &snapshot.Snapshot{Files: map[string][]byte{}}, nil)
	if err == nil || !strings.Contains(err.Error(), "has no observed bytes") {
		t.Fatalf("missing selected bytes error = %v", err)
	}
	_, _, err = prepareAzurePipelinesProjectionInput(azurePipelinesRequest(), []authoring.Check{{Name: azurepipelines.RequiredCheck, Run: []string{"go", "test", "./..."}}}, []source.WorkingFileMetadata{{Path: "pipelines/azure-pipelines.yml", Mode: snapshot.RegularMode}}, &snapshot.Snapshot{Files: map[string][]byte{}}, nil, false)
	if err == nil || !strings.Contains(err.Error(), "has no observed bytes") {
		t.Fatalf("missing selected bytes error = %v", err)
	}
}
