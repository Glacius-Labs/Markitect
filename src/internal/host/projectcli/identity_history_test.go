package projectcli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectapp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectbriefing"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexplore"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
)

func TestGlobalAndNamedCanonicalEditSupportsBriefingsAndReadiness(t *testing.T) {
	for _, namespace := range []string{"", "example"} {
		t.Run(namespace, func(t *testing.T) {
			root := t.TempDir()
			runGit(t, root, "init", "--initial-branch=feature-identity")
			cli := func(args ...string) []byte {
				t.Helper()
				return mustCLI(t, append([]string{args[0], "--repo", root}, args[1:]...)...)
			}
			writeDocs := func() {
				t.Helper()
				preview := decodeOutput[projectapp.DocumentResult](t, cli("docs"))
				cli("docs", "--expect", preview.Digest, "--write")
			}
			initPreview := decodeOutput[projectwork.InitPlan](t, cli("init", "--name", "generic-identity"))
			cli("init", "--name", "generic-identity", "--expect", initPreview.Digest, "--write")
			modelPath := ".markitect/model/manager.yaml"
			path := filepath.Join(root, filepath.FromSlash(modelPath))
			if namespace != "" {
				// Keep the global project owner and introduce a normal named child
				// before the baseline. This matches the responsibility hierarchy.
				updated := fmt.Sprintf("apiVersion: project.markitect.example.org/v1alpha1\nkind: Manager\nmetadata:\n  name: notifications\n  namespace: %q\npurpose: Owns the repository-wide engineering mandate for notifications.\nspec:\n  parent:\n    namespace: \"\"\n    name: project-owner\n  owns: [notifications/]\n", namespace)
				newPath := filepath.Join(root, ".markitect", "model", namespace, "manager.yaml")
				if err := os.MkdirAll(filepath.Dir(newPath), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(newPath, []byte(updated), 0644); err != nil {
					t.Fatal(err)
				}
				manifestPath := filepath.Join(root, ".markitect", "project.yaml")
				manifest, err := os.ReadFile(manifestPath)
				if err != nil {
					t.Fatal(err)
				}
				selectedPath := ".markitect/model/" + namespace + "/manager.yaml"
				if err := os.WriteFile(manifestPath, []byte(strings.Replace(string(manifest), modelPath+"\n", modelPath+"\n    - "+selectedPath+"\n", 1)), 0644); err != nil {
					t.Fatal(err)
				}
				modelPath, path = selectedPath, newPath
				writeDocs()
			}
			runGit(t, root, "add", ".")
			commit := func(message string) {
				t.Helper()
				runGitWithEnv(t, root, testCommitEnv, "commit", "-m", message)
			}
			commit("generic initial model")
			project, err := projectwork.Load(root, "")
			if err != nil {
				t.Fatal(err)
			}
			// This config binds the test executable for static runtime validation.
			// Readiness and briefings never invoke it or start a provider.
			writeExploreTestRuntime(t, root, project.Report.Managers)
			project, err = projectwork.Load(root, "")
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			mutation := projectwork.Mutation{APIVersion: projectwork.APIVersion, BaseDigest: project.Digest,
				Actor: projectwork.HumanActor, Goal: "Clarify the generic Manager mandate.",
				Files: []projectwork.FileChange{{Path: modelPath, Content: strings.Replace(string(data), "repository-wide engineering mandate", "repository-wide notification mandate", 1)}}}
			proposal, err := projectwork.EncodeMutation(mutation)
			if err != nil {
				t.Fatal(err)
			}
			editInput := writeDraft(t, root, ".markitect/drafts/identity-edit.json", proposal)
			preview := decodeOutput[projectwork.EditPlan](t, cli("edit", "--input", editInput))
			cli("edit", "--input", editInput, "--expect", preview.Digest, "--write")
			writeDocs()
			runGit(t, root, "add", modelPath, "docs/markitect/project.md")
			commit("accepted canonical mandate change")
			revision := gitOutput(t, root, "rev-parse", "HEAD")
			manager := ""
			for _, candidate := range project.Report.Managers {
				if candidate.Namespace == namespace {
					manager = candidate.ID
					break
				}
			}
			if manager == "" {
				t.Fatal("fixture has no Manager for the selected namespace")
			}
			record := projectexplore.Record{APIVersion: projectexplore.APIVersion, ID: "identity-work", Status: projectexplore.StatusActive,
				Request:   "Implement the declared notification mandate.",
				Scopes:    []projectexplore.Scope{{ID: "notification", Name: "Notification", Goal: "Implement the declared notification mandate.", Operation: "apply", ManagerIDs: []string{manager}}},
				Decisions: []projectexplore.Decision{}, Drafts: []projectexplore.DraftProposal{},
				Acknowledgements: []projectexplore.StructureAcknowledgement{}, Completions: []projectexplore.ApplyReceipt{}}
			exploreInput := writeDraft(t, root, ".markitect/drafts/identity-explore.json", record)
			exploration := decodeOutput[projectapp.ExploreResult](t, cli("explore", "--input", exploreInput))
			if exploration.Plan == nil {
				t.Fatalf("explore preview has no write plan: %#v", exploration)
			}
			cli("explore", "--input", exploreInput, "--expect", exploration.Plan.Digest, "--write")
			readyArgs := []string{"ready", "--exploration", record.ID, "--scope", "notification"}
			ready := decodeOutput[projectapp.ReadinessResult](t, cli(readyArgs...))
			if !ready.Binding.ModelAccepted || ready.Binding.ModelRevision != revision || ready.Readiness.Ready || len(ready.Readiness.Blockers) == 0 {
				t.Fatalf("accepted model should bind without inventing structure acknowledgement: %+v", ready)
			}
			list := decodeOutput[briefListOutput](t, cli("brief", "list"))
			if len(list.Notifications) != 1 || list.Notifications[0].DefinitionID.Namespace != namespace || list.Notifications[0].ResolutionStatus != "unresolved" {
				t.Fatalf("legal namespace briefing lost or falsely resolved: %+v", list)
			}
			state, digest, err := projectbriefing.Read(root)
			if err != nil || state.History == nil || state.History.Revision != revision || len(state.Briefings) != 1 {
				t.Fatalf("CLI history state=%+v err=%v", state, err)
			}
			summary := decodeOutput[overview](t, cli("status"))
			if len(summary.Briefings) == 0 {
				t.Fatalf("status overview has no pending briefings: %+v", summary)
			}
			resumed := decodeOutput[projectapp.ReadinessResult](t, cli(readyArgs...))
			_, afterDigest, err := projectbriefing.Read(root)
			if err != nil || afterDigest != digest || resumed.Readiness.Digest != ready.Readiness.Digest {
				t.Fatalf("readiness resume changed the immutable history/binding: err=%v", err)
			}
		})
	}
}
