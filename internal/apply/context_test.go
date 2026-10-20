package apply_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
)

func selectedContext(t *testing.T, selections []organization.ContextSelection, pkg string) organization.ContextSelection {
	t.Helper()
	for _, selection := range selections {
		if selection.Package() == pkg {
			return selection
		}
	}
	t.Fatalf("missing Manager selection %q", pkg)
	return organization.ContextSelection{}
}

func contextID(pkg, id string) model.Identity { return model.Identity{Package: pkg, ID: id} }

func hasContextReason(entry organization.ContextDefinition, kind organization.ContextReasonKind, from model.Identity, at string) bool {
	for _, reason := range entry.Reasons {
		if reason.Kind == kind && reason.From == from && reason.To == entry.Definition.Identity && reason.At == at {
			return true
		}
	}
	return false
}

func TestM1StaticContextSelectionsRetainActualModelAndExecution(t *testing.T) {
	selection, raw := executionInputs(t, documentedExecution(t))
	parent := registryModel(t, selection, raw).DecodeExecution()
	result := parent.SelectContexts()
	contexts, ok := result.Selections()
	if !ok || result.Status() != model.Passed || len(result.Diagnostics()) != 0 {
		t.Fatalf("M1 selection failed: %+v", result.Diagnostics())
	}
	var packages []string
	for _, selected := range contexts {
		packages = append(packages, selected.Package())
	}
	if !reflect.DeepEqual(packages, []string{".", "catalog", "lending", "lending/reminders", "library", "members"}) {
		t.Fatalf("wrong ordered local Managers: %q", packages)
	}
	root := selectedContext(t, contexts, ".")
	if len(root.Definitions()) != 0 {
		t.Fatal("ordinary root acquired descendant Definitions without modeled scope")
	}
	lending := selectedContext(t, contexts, "lending")
	borrowID := contextID("lending", "borrow")
	borrow, found := lending.Definition(borrowID)
	if !found || !hasContextReason(borrow, organization.OwnDefinition, borrowID, "") || borrow.Definition.Source.Path != "model/lending/borrow.yaml" || borrow.Definition.Source.Digest == "" {
		t.Fatal("own Definition/source/reason lost")
	}
	member, found := lending.Definition(contextID("members", "member"))
	if !found || !hasContextReason(member, organization.OutgoingReference, borrowID, "/bodyLinks/0") {
		t.Fatalf("direct outgoing reason lost: %+v", member.Reasons)
	}
	useCase, found := lending.Definition(contextID("software", "UseCase"))
	if !found || !hasContextReason(useCase, organization.KindContract, borrowID, "/kind") {
		t.Fatal("bound imported Kind contract lost")
	}
	checked := parent.Model().Checked()
	for _, selected := range contexts {
		if selected.ModelID() != checked.Models[0].ID || selected.ModelDigest() != checked.Models[0].Digest {
			t.Fatal("selection/model binding changed")
		}
	}
	if !reflect.DeepEqual(result.Execution().Model().Inputs(), parent.Model().Inputs()) || result.Execution().Model().Inputs().Guard == "" || !reflect.DeepEqual(result.Execution().Partial(), parent.Partial()) || result.Execution().Partial().Scope != nil {
		t.Fatal("capture/source/absence binding changed")
	}
	for _, revision := range []string{"", fixtureHead} {
		t.Run("preview-"+revision, func(t *testing.T) {
			source := controlledSource{preview: func(context.Context, apply.PreviewRequest) (apply.PreviewedInputs, error) {
				return apply.PreviewedInputs{Raw: raw, Head: fixtureHead, Revision: revision}, nil
			}}
			preview, err := apply.NewModelService(nil, source).PreviewModel(context.Background(), apply.PreviewRequest{Selection: selection, Revision: revision}, model.Limits{})
			if err != nil {
				t.Fatal(err)
			}
			decoded := preview.DecodeExecution()
			prepared := decoded.SelectContexts()
			got, usable := prepared.Selections()
			if !usable || prepared.Status() != model.Passed || !reflect.DeepEqual(got, contexts) || !reflect.DeepEqual(prepared.Execution().Model().Inputs(), preview.Inputs()) || prepared.Execution().Model().Inputs().Revision != revision {
				t.Fatalf("preview binding changed: %+v", prepared.Diagnostics())
			}
		})
	}
}
