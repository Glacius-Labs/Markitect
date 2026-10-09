package projectbriefing

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
)

func TestBriefingEventRejectsMalformedIdentitiesUsingCompilerGrammar(t *testing.T) {
	root, base, revision := identityProjectFixture(t, "example", "modified")
	bundle, err := Generate(root, base, revision, testProvenance())
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		mutate func(*core.DefinitionIdentity)
	}{
		{"missing API version", func(i *core.DefinitionIdentity) { i.APIVersion = "" }},
		{"invalid API version", func(i *core.DefinitionIdentity) { i.APIVersion = "invalid" }},
		{"missing kind", func(i *core.DefinitionIdentity) { i.Kind = "" }},
		{"invalid kind", func(i *core.DefinitionIdentity) { i.Kind = "bad kind" }},
		{"missing name", func(i *core.DefinitionIdentity) { i.Name = "" }},
		{"invalid name", func(i *core.DefinitionIdentity) { i.Name = "bad/name" }},
		{"invalid namespace", func(i *core.DefinitionIdentity) { i.Namespace = "bad/namespace" }},
		{"whitespace namespace", func(i *core.DefinitionIdentity) { i.Namespace = " " }},
	} {
		t.Run(test.name, func(t *testing.T) {
			data, err := json.Marshal(bundle.Events[0])
			if err != nil {
				t.Fatal(err)
			}
			var event Event
			if err := json.Unmarshal(data, &event); err != nil {
				t.Fatal(err)
			}
			test.mutate(&event.DefinitionID)
			for _, value := range []*core.Definition{event.Before, event.After} {
				value.APIVersion, value.Kind = event.DefinitionID.APIVersion, event.DefinitionID.Kind
				value.Metadata = core.Metadata{Namespace: event.DefinitionID.Namespace, Name: event.DefinitionID.Name}
			}
			// Re-seal the event so identity rejection cannot be masked by a stale
			// digest or mismatched before/after payload.
			payload := struct {
				Since, Revision, Key, Change string
				Before, After                *core.Definition
			}{bundle.SinceRevision, bundle.Revision, event.DefinitionID.Key(), event.Change, event.Before, event.After}
			event.Digest = hash(payload)
			event.ID = "model-change-" + event.Digest[:24]
			if err := validateEvent(bundle, event); !errors.Is(err, ErrInvalidBundle) || !strings.Contains(err.Error(), "invalid definition identity") {
				t.Fatalf("malformed identity passed or failed an unrelated binding: %v", err)
			}
		})
	}
}

func TestGlobalBriefingRetainsEventAndReferenceRejections(t *testing.T) {
	root, base, revision := identityProjectFixture(t, "", "modified")
	bundle, err := Generate(root, base, revision, testProvenance())
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		mutate func(*Bundle)
	}{
		{"event ID", func(b *Bundle) { b.Events[0].ID = "invented-event" }},
		{"event digest", func(b *Bundle) { b.Events[0].Digest = strings.Repeat("f", 64) }},
		{"provenance", func(b *Bundle) { b.Events[0].Provenance.Actor = "different actor" }},
		{"missing decision", func(b *Bundle) { b.Provenance.DecisionReference = "" }},
		{"unknown change", func(b *Bundle) { b.Events[0].Change = "invented" }},
		{"payload identity", func(b *Bundle) { b.Events[0].After.Metadata.Name = "unrelated" }},
		{"duplicate event", func(b *Bundle) { b.Events = append(b.Events, b.Events[0]) }},
		{"global event ref", func(b *Bundle) { b.Global.EventIDs = []string{"unknown-event"} }},
		{"manager event ref", func(b *Bundle) { b.Managers[0].EventIDs = []string{"unknown-event"} }},
		{"duplicate affected ref", func(b *Bundle) {
			b.Events[0].AffectedManagers = append(b.Events[0].AffectedManagers, b.Events[0].AffectedManagers[0])
		}},
		{"missing model digest", func(b *Bundle) { b.ModelDigest = "" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			data, err := json.Marshal(bundle)
			if err != nil {
				t.Fatal(err)
			}
			var invalid Bundle
			if err := json.Unmarshal(data, &invalid); err != nil {
				t.Fatal(err)
			}
			test.mutate(&invalid)
			invalid.Digest = FullBundleDigest(invalid)
			if err := validateBundle(invalid); !errors.Is(err, ErrInvalidBundle) {
				t.Fatalf("invalid bundle accepted: %v", err)
			}
			_, initial, err := Read(root)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Write(root, invalid, initial); !errors.Is(err, ErrInvalidBundle) {
				t.Fatalf("invalid bundle persisted: %v", err)
			}
			state, after, err := Read(root)
			if err != nil || initial != after || len(state.Briefings) != 0 {
				t.Fatalf("rejected record mutated store: state=%+v err=%v", state, err)
			}
		})
	}
}

// Each case creates an independent generic project, without provider execution
// or study inputs. Commits exercise the declared committed-model policy only;
// they do not authenticate a human decision.
func TestGlobalAndNamedDefinitionChangesRoundTrip(t *testing.T) {
	for _, namespace := range []string{"", "example"} {
		for _, change := range []string{"added", "modified", "removed"} {
			t.Run(namespace+"/"+change, func(t *testing.T) {
				root, base, revision := identityProjectFixture(t, namespace, change)
				before, err := projectwork.Load(root, base)
				if err != nil {
					t.Fatal(err)
				}
				after, err := projectwork.Load(root, revision)
				if err != nil {
					t.Fatal(err)
				}
				built, err := Build(before, after, testProvenance())
				if err != nil {
					t.Fatal(err)
				}
				generated, err := Generate(root, base, revision, testProvenance())
				if err != nil || built.Digest != generated.Digest || len(generated.Events) != 1 {
					t.Fatalf("Build/Generate mismatch: bundle=%+v err=%v", generated, err)
				}
				event := generated.Events[0]
				if event.Change != change || event.DefinitionID.Namespace != namespace {
					t.Fatalf("changed definition identity or operation lost: %+v", event)
				}
				_, initial, err := Read(root)
				if err != nil {
					t.Fatal(err)
				}
				written, err := Write(root, generated, initial)
				if err != nil {
					t.Fatalf("persist legal %q namespace: %v", namespace, err)
				}
				stored, loaded, err := Read(root)
				if err != nil || written != loaded || len(stored.Briefings) != 1 || stored.Briefings[0].Digest != generated.Digest {
					t.Fatalf("reload differs: state=%+v digest=%s err=%v", stored, loaded, err)
				}
				manager := generated.Managers[0].ManagerID
				briefings, events, binding, err := LoadForManager(root, after.Model.Digest, manager, revision)
				if err != nil || len(briefings) != 1 || len(events) != 1 || events[0].Digest != event.Digest || binding == "" {
					t.Fatalf("manager reload: briefings=%d events=%+v binding=%s err=%v", len(briefings), events, binding, err)
				}
				again, err := Write(root, generated, written)
				if err != nil || again != written {
					t.Fatalf("idempotent write: digest=%s err=%v", again, err)
				}
			})
		}
	}
}

func TestGlobalAndNamedAcceptedHistoryReloadAndResume(t *testing.T) {
	for _, namespace := range []string{"", "example"} {
		t.Run(namespace, func(t *testing.T) {
			root, base, revision := identityProjectFixture(t, namespace, "modified")
			first, err := EnsureAcceptedHistory(root, revision)
			if err != nil || first.BaselineRevision != base || len(first.Bundles) != 1 {
				t.Fatalf("accepted canonical change: receipt=%+v err=%v", first, err)
			}
			state, binding, err := Read(root)
			if err != nil || state.History == nil || state.History.Revision != revision || binding != first.StoreDigest {
				t.Fatalf("accepted history reload: state=%+v digest=%s err=%v", state, binding, err)
			}
			repeated, err := EnsureAcceptedHistory(root, revision)
			if err != nil || repeated.StoreDigest != binding || len(repeated.Bundles) != 0 {
				t.Fatalf("accepted history resume: receipt=%+v err=%v", repeated, err)
			}
			if err := os.WriteFile(filepath.Join(root, ".markitect", "notes.txt"), []byte("Non-canonical commit.\n"), 0644); err != nil {
				t.Fatal(err)
			}
			gitTest(t, root, "add", ".markitect/notes.txt")
			gitCommitTest(t, root, "advance without canonical change")
			next := gitOutputTest(t, root, "rev-parse", "HEAD")
			resumed, err := EnsureAcceptedHistory(root, next)
			if err != nil || resumed.Revision != next || len(resumed.Bundles) != 0 {
				t.Fatalf("non-canonical resume: receipt=%+v err=%v", resumed, err)
			}
			after, err := projectwork.Load(root, next)
			if err != nil {
				t.Fatal(err)
			}
			manager := first.Bundles[0].Managers[0].ManagerID
			_, events, _, err := LoadForManager(root, after.Model.Digest, manager, next)
			if err != nil || len(events) != 1 || events[0].DefinitionID.Namespace != namespace {
				t.Fatalf("resumed manager events=%+v err=%v", events, err)
			}
		})
	}
}

func identityProjectFixture(t *testing.T, namespace, change string) (root, base, revision string) {
	t.Helper()
	root = t.TempDir()
	prefix := ".markitect/model/"
	if namespace != "" {
		prefix += namespace + "/"
	}
	managerPath, statementPath := prefix+"manager.yaml", prefix+"notification.yaml"
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, filepath.FromSlash(managerPath))), 0755); err != nil {
		t.Fatal(err)
	}
	manager := fmt.Sprintf("apiVersion: project.markitect.example.org/v1alpha1\nkind: Manager\nmetadata:\n  name: project\n  namespace: %q\npurpose: Own the generic example.\nspec:\n  owns: [.]\n", namespace)
	const rootManagerPath = ".markitect/model/manager.yaml"
	if namespace != "" {
		globalManager := "apiVersion: project.markitect.example.org/v1alpha1\nkind: Manager\nmetadata:\n  name: project\n  namespace: \"\"\npurpose: Own the generic example.\nspec:\n  owns: [.]\n"
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rootManagerPath)), []byte(globalManager), 0644); err != nil {
			t.Fatal(err)
		}
		manager = fmt.Sprintf("apiVersion: project.markitect.example.org/v1alpha1\nkind: Manager\nmetadata:\n  name: notifications\n  namespace: %q\npurpose: Own the notification slice.\nspec:\n  parent:\n    namespace: \"\"\n    name: project\n  owns: [notifications/]\n", namespace)
	}
	statement := func(description string) string {
		return fmt.Sprintf("apiVersion: project.markitect.example.org/v1alpha1\nkind: Statement\nmetadata:\n  name: notification\n  namespace: %q\npurpose: Specify notification wording.\nspec:\n  category: rule\n  description: %s\n  public: true\n", namespace, description)
	}
	initial, changed := statement("Initial notification."), statement("Updated notification.")
	if change == "added" {
		initial = ""
	} else if change == "removed" {
		changed = ""
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(managerPath)), []byte(manager), 0644); err != nil {
		t.Fatal(err)
	}
	writeModel := func(value string) {
		t.Helper()
		manifest := "apiVersion: project.markitect.example.org/v1alpha1\nname: definition-identity-example\nworkflowMode: guided\nacceptancePolicy: committed-model\nmodelFiles:\n  - " + managerPath + "\n"
		if namespace != "" {
			manifest += "  - " + rootManagerPath + "\n"
		}
		path := filepath.Join(root, filepath.FromSlash(statementPath))
		if value != "" {
			manifest += "  - " + statementPath + "\n"
			if err := os.WriteFile(path, []byte(value), 0644); err != nil {
				t.Fatal(err)
			}
		} else if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		manifest += "inventoryRoots: []\nexclusions: []\n"
		if err := os.WriteFile(filepath.Join(root, ".markitect", "project.yaml"), []byte(manifest), 0644); err != nil {
			t.Fatal(err)
		}
	}
	writeModel(initial)
	gitTest(t, root, "init", "--initial-branch=feature-identity")
	gitTest(t, root, "add", ".")
	gitCommitTest(t, root, "generic identity baseline")
	base = gitOutputTest(t, root, "rev-parse", "HEAD")
	writeModel(changed)
	gitTest(t, root, "add", ".markitect")
	gitCommitTest(t, root, "accepted canonical notification change")
	revision = gitOutputTest(t, root, "rev-parse", "HEAD")
	return root, base, revision
}
