package gitrepo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

const gitSettingsRuntimeDocument = `apiVersion: markitect.runtime/v1
agentProfiles:
  library-agent:
    codexAppServer: {profileVersion: 1}
checkProfiles:
  library-tests:
    process: {profileVersion: 1}
`
const gitSettingsRuntimeInstruction = "\xef\xbb\xbfExplicit external Runtime guidance.\r\n"

func gitSettingsFixture(t *testing.T) (*testkit.Repo, apply.InputSelection, string, apply.SourceLocation) {
	t.Helper()
	repo, selection, head := fileCheckSeamFixture(t)
	external := testkit.TempDir(t)
	for name, content := range map[string]string{"runtime.yaml": gitSettingsRuntimeDocument, "ops/manager.md": gitSettingsRuntimeInstruction} {
		file := filepath.Join(external, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	selection.RuntimeProfiles = []apply.SourceLocation{{Boundary: external, Path: fixturePath(t, "runtime.yaml")}}
	runtimeLocation := apply.SourceLocation{Boundary: external, Path: fixturePath(t, "ops/manager.md")}
	return repo, selection, head, runtimeLocation
}

// Only these metadata observations are controlled. Source inputs and ProfileInput
// are real acquired bytes. These ports start no executable/provider/check and
// make no claim about the actual drivers' validation contracts or readiness.
type gitSettingsMetadataPorts struct {
	t              *testing.T
	policy         []apply.SourceLocation
	agents, checks []apply.ProfileInput
}

func (p *gitSettingsMetadataPorts) observation(input apply.ProfileInput, agent bool) apply.ProfileObservation {
	p.t.Helper()
	file := func(boundary, name, data string) apply.ProfileMetadataFile {
		return apply.ProfileMetadataFile{Location: apply.SourceLocation{Boundary: boundary, Path: fixturePath(p.t, name)}, ContentID: snapshot.ContentID(sha256.Sum256([]byte(data))), Size: int64(len(data)), Mode: snapshot.Regular}
	}
	o := apply.ProfileObservation{InputBinding: input.Binding(), Status: model.Passed, Contract: apply.ProfileContract{Name: "controlled-git-settings-seam", Version: 1}, Files: []apply.ProfileMetadataFile{file("controlled-program", "program", "controlled static program metadata")}}
	if agent {
		o.Inventories = []apply.ProfileMetadataInventory{{Boundary: "controlled-schemas", Files: []apply.ProfileMetadataFile{file("controlled-schemas", "schema.json", "controlled static schema")}}}
		o.Instructions = append([]apply.SourceLocation(nil), p.policy...)
	}
	return o
}

func (p *gitSettingsMetadataPorts) PrepareAgentProfile(_ context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
	p.agents = append(p.agents, input)
	return p.observation(input, true), nil
}
func (p *gitSettingsMetadataPorts) PrepareCheckProfile(_ context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
	p.checks = append(p.checks, input)
	return p.observation(input, false), nil
}

func TestM1GitSettingsPreparationSharesRepoAndRuntimeClosure(t *testing.T) {
	_, full, head, runtimeLocation := gitSettingsFixture(t)
	project := apply.SourceLocation{Boundary: full.Repository, Path: fixturePath(t, "ops/manager.md")}
	policy := []apply.SourceLocation{project, runtimeLocation, runtimeLocation, project}
	for _, complete := range []bool{false, true} {
		name := "derived"
		if complete {
			name = "complete"
		}
		for _, mode := range []string{"capture", "working"} {
			t.Run(name+"/"+mode, func(t *testing.T) {
				selection := full
				wantCalls := 2
				if complete {
					selection.Instructions = append(append([]apply.SourceLocation(nil), full.Instructions...), runtimeLocation)
					wantCalls = 1
				} else {
					selection.Execution, selection.Registry, selection.Ignore, selection.Instructions = nil, nil, nil, nil
				}
				port := &recordingPreparationGit{repository: adapter(t)}
				metadata := &gitSettingsMetadataPorts{t: t, policy: policy}
				service := apply.NewPreparationService(apply.NewModelService(port, port))
				var first, last apply.RawInputs
				var settings apply.CapturedSettingsPreparation
				var observation []apply.PreparedProfile
				var firstNeeds []apply.SourceLocation
				var accounting realization.Accounting
				if mode == "capture" {
					request := apply.CaptureRequest{Selection: selection}
					result, err := service.CaptureSettings(context.Background(), request, model.Limits{}, metadata, metadata)
					prepared, ok := result.Prepared()
					if err != nil || !ok || result.Status() != model.Passed || len(port.captures) != wantCalls || len(result.Attempts()) != wantCalls || !reflect.DeepEqual(result.Request(), request) {
						t.Fatalf("actual Settings capture failed: %s %v", result.Status(), err)
					}
					if complete && !reflect.DeepEqual(port.captureRequests[0], request) {
						t.Fatal("complete explicit request changed")
					}
					for i, attempt := range result.Attempts() {
						assertGitCapturedSettingsAttempt(t, attempt, port.captureRequests[i], port.captures[i])
					}
					if !reflect.DeepEqual(prepared.Structural().Model().Inputs(), port.captures[wantCalls-1]) || port.captures[wantCalls-1].Guard == "" || port.captures[wantCalls-1].Head != head {
						t.Fatal("Prepared did not keep the last actual full capture/opaque guard")
					}
					first, last = port.captures[0].Raw, port.captures[wantCalls-1].Raw
					settings = prepared.Settings()
					observation = settings.ProfileObservations()
					firstNeeds = result.Attempts()[0].Settings().RequiredInstructions()
					accounting, _ = prepared.Structural().Files().Accounting()
					resolved, ok := settings.Settings()
					root, found := resolved.Manager(".")
					checks, checked := settings.Checks()
					if !ok || !found || !checked || len(checks) != 1 || root.Execute.Profile.Value.Identity == "" {
						t.Fatal("actual static settings/check selections missing")
					}
					assertGitSettingsRoles(t, resolved)
					authored, _ := prepared.Structural().Execution().Document()
					if !reflect.DeepEqual(checks[0].Declaration(), authored.Checks.Value[0]) {
						t.Fatal("captured literal check declaration changed")
					}
				} else {
					request := apply.PreviewRequest{Selection: selection}
					result, err := service.PreviewSettings(context.Background(), request, model.Limits{}, metadata, metadata)
					prepared, ok := result.Prepared()
					if err != nil || !ok || result.Status() != model.Passed || len(port.previews) != wantCalls || len(result.Attempts()) != wantCalls || !reflect.DeepEqual(result.Request(), request) {
						t.Fatalf("actual external Runtime working preview failed: %s %v", result.Status(), err)
					}
					if complete && !reflect.DeepEqual(port.previewRequests[0], request) {
						t.Fatal("complete explicit preview changed")
					}
					for i, attempt := range result.Attempts() {
						if !reflect.DeepEqual(attempt.Structural().Model().Inputs(), port.previews[i]) || !reflect.DeepEqual(attempt.Structural().Model().Request(), port.previewRequests[i]) || !reflect.DeepEqual(attempt.Settings().FileCheck(), attempt.Structural().Files()) || !reflect.DeepEqual(attempt.Settings().Defaults(), attempt.Defaults()) || !reflect.DeepEqual(attempt.Defaults().ProductInput(), apply.ProductStrategyDefaults()) || port.previews[i].Revision != "" || port.previews[i].Head != head || attempt.Error() != nil {
							t.Fatal("working preview changed an actual acquisition/default/settings binding")
						}
					}
					if !reflect.DeepEqual(prepared.Structural().Model().Inputs(), port.previews[wantCalls-1]) {
						t.Fatal("preview Prepared retained an earlier acquisition")
					}
					first, last = port.previews[0].Raw, port.previews[wantCalls-1].Raw
					observation = prepared.Settings().ProfileObservations()
					firstNeeds = result.Attempts()[0].Settings().RequiredInstructions()
					accounting, _ = prepared.Structural().Files().Accounting()
					resolved, available := prepared.Settings().Settings()
					if !available {
						t.Fatal("working settings missing")
					}
					assertGitSettingsRoles(t, resolved)
					if checks, ok := prepared.Settings().Checks(); !ok || len(checks) != 1 {
						t.Fatal("working checks missing")
					} else {
						authored, _ := prepared.Structural().Execution().Document()
						if !reflect.DeepEqual(checks[0].Declaration(), authored.Checks.Value[0]) {
							t.Fatal("working literal check declaration changed")
						}
					}
				}
				if len(metadata.agents) != wantCalls || len(metadata.checks) != wantCalls || len(observation) != 2 || accounting.Status() != realization.Accounted || accounting.TreeID() != last.Census.Digest() || len(accounting.Files()) != len(last.Census.Entries()) || len(accounting.Findings()) != 0 {
					t.Fatal("metadata ran outside the shared acquisition bound or actual Census accounting failed")
				}
				if !complete {
					assertOriginalPreparationGap(t, first, last)
					if len(firstNeeds) != 2 || len(metadata.agents[0].Instructions()) != 0 {
						t.Fatal("first metadata observation did not expose both project and external Runtime needs")
					}
				} else if len(firstNeeds) != 0 {
					t.Fatal("complete first input invented missing Runtime instructions")
				}
				assertGitSettingsInputSeparation(t, last, runtimeLocation, observation[0])
				if mode == "capture" {
					resolved, _ := settings.Settings()
					root, _ := resolved.Manager(".")
					if len(root.Execute.Instructions) != 3 || !bytes.Equal(root.Execute.Instructions[0].Value.Bytes, []byte(fileSeamInstruction("ops/manager.md"))) || !reflect.DeepEqual(root.Execute.Instructions[0], root.Execute.Instructions[2]) {
						t.Fatal("Runtime instructions replaced project facts or authored repeats")
					}
				}
			})
		}
	}
}

func assertGitCapturedSettingsAttempt(t *testing.T, attempt apply.CapturedSettingsAttempt, request apply.CaptureRequest, inputs apply.CapturedInputs) {
	t.Helper()
	structural := attempt.Structural()
	if attempt.Error() != nil || !reflect.DeepEqual(structural.Model().Request(), request) || !reflect.DeepEqual(structural.Model().Inputs(), inputs) || !reflect.DeepEqual(attempt.Settings().FileCheck(), structural.Files()) || !reflect.DeepEqual(attempt.Settings().Defaults(), attempt.Defaults()) || !reflect.DeepEqual(attempt.Defaults().Execution().Model().Inputs(), inputs) || !reflect.DeepEqual(attempt.Defaults().ProductInput(), apply.ProductStrategyDefaults()) || structural.Model().Checked().Status != model.Passed || len(structural.Model().Checked().Models) != 3 {
		t.Fatal("attempt changed original Source/Raw/guard/defaults/settings reports")
	}
}

func assertGitSettingsInputSeparation(t *testing.T, raw apply.RawInputs, runtimeLocation apply.SourceLocation, profile apply.PreparedProfile) {
	t.Helper()
	if len(raw.Instructions) != 4 || len(raw.RuntimeProfiles) != 1 || raw.RuntimeProfiles[0].RepositoryPath.String() != "" || !bytes.Equal(raw.RuntimeProfiles[0].File.Bytes(), []byte(gitSettingsRuntimeDocument)) {
		t.Fatal("explicit external Runtime profile document or distinct instruction acquisition missing")
	}
	if profile.Input().Document() != raw.RuntimeProfiles[0] || !reflect.DeepEqual(profile.Input().Instructions(), raw.Instructions) || profile.Observation().InputBinding != profile.Input().Binding() || profile.Input().Namespace() != "agentProfiles" || profile.Input().Name() != "library-agent" || profile.Input().Field() != "/agentProfiles/library-agent/codexAppServer" || profile.Input().Inline() == nil {
		t.Fatal("actual ProfileInput was not bound to acquired Runtime/Instruction bytes")
	}
	projectRaw := raw
	projectRaw.Instructions = nil
	seen := make(map[apply.SourceLocation]bool)
	for _, file := range raw.Instructions {
		if seen[file.Location] {
			t.Fatal("derived instruction location captured twice")
		}
		seen[file.Location] = true
		if file.Location == runtimeLocation {
			if file.RepositoryPath.String() != "" || !bytes.Equal(file.File.Bytes(), []byte(gitSettingsRuntimeInstruction)) {
				t.Fatal("external Runtime gained repo coordinates or project bytes")
			}
		} else {
			projectRaw.Instructions = append(projectRaw.Instructions, file)
		}
	}
	assertPreparationFileClosure(t, projectRaw)
	projectFile, found := raw.Census.Lookup(fixturePath(t, "ops/manager.md"))
	if !found || !bytes.Equal(projectFile.Bytes(), []byte(fileSeamInstruction("ops/manager.md"))) || bytes.Equal(projectFile.Bytes(), []byte(gitSettingsRuntimeInstruction)) {
		t.Fatal("Runtime bytes entered the repository Census")
	}
	// Compare the literal ordered policy against the actual acquired locations;
	// the repository and Runtime location share Path but have different boundaries.
	var repoLocation apply.SourceLocation
	for _, file := range projectRaw.Instructions {
		if file.RepositoryPath.String() == "ops/manager.md" {
			repoLocation = file.Location
		}
	}
	if !reflect.DeepEqual(profile.Observation().Instructions, []apply.SourceLocation{repoLocation, runtimeLocation, runtimeLocation, repoLocation}) {
		t.Fatal("Runtime policy order/repeats/coordinates changed")
	}
}

func assertGitSettingsRoles(t *testing.T, settings organization.Settings) {
	t.Helper()
	for _, expected := range []struct {
		pkg               string
		manager, reviewer []string
	}{
		{".", []string{"ops/manager.md", "ops/reviewer.md", "ops/manager.md"}, []string{"ops/reviewer.md"}},
		{"lending", []string{"ops/manager.md", "ops/reviewer.md", "ops/manager.md", "ops/lending.md"}, []string{"ops/reviewer.md", "ops/manager.md", "ops/reviewer.md"}},
	} {
		resolved, found := settings.Manager(expected.pkg)
		if !found {
			t.Fatalf("missing actual Manager settings: %s", expected.pkg)
		}
		for _, role := range []struct {
			actual []organization.Setting[organization.InstructionFact]
			want   []string
		}{{resolved.Execute.Instructions, expected.manager}, {resolved.Verify.Instructions, expected.reviewer}} {
			var ids []string
			for _, instruction := range role.actual {
				ids = append(ids, instruction.Value.ID)
				if !bytes.Equal(instruction.Value.Bytes, []byte(fileSeamInstruction(instruction.Value.ID))) {
					t.Fatal("external Runtime bytes became project instruction facts")
				}
			}
			if !reflect.DeepEqual(ids, role.want) {
				t.Fatalf("authored role instruction order/repeats changed: %s %q", expected.pkg, ids)
			}
		}
	}
}
