package execution

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/government"
)

func TestAmendmentEscalationOwnerFindingDominatesAreaRoute(t *testing.T) {
	area := core.DefinitionIdentity{APIVersion: government.APIVersion, Kind: "Area", Name: "delivery"}
	subject := core.DefinitionIdentity{APIVersion: "orders.example/v1", Kind: "Rule", Namespace: "shop", Name: "rate"}
	mandate := core.Definition{
		APIVersion: government.APIVersion,
		Kind:       "Mandate",
		Metadata:   core.Metadata{Name: "delivery-amend", Namespace: "shop"},
		Purpose:    "Amend the scoped delivery rate rule",
		Spec: map[string]any{
			"area":    area,
			"scope":   []core.DefinitionIdentity{subject},
			"actions": []string{"amend-model"},
		},
	}
	model := government.Model{Canonical: core.Model{Definitions: []core.Definition{
		{APIVersion: government.APIVersion, Kind: "Area", Metadata: core.Metadata{Name: "delivery", Namespace: "shop"}, Purpose: "Delivery area", Spec: map[string]any{}},
		{APIVersion: subject.APIVersion, Kind: subject.Kind, Metadata: core.Metadata{Name: subject.Name, Namespace: subject.Namespace}, Purpose: "Delivery rate rule", Spec: map[string]any{}},
		mandate,
	}}}
	areaFinding := government.AmendmentFinding{Code: "amendment.scope", Subject: subject.Key(), EscalateTo: area}
	ownerFinding := government.AmendmentFinding{Code: "amendment.protected-subject", Subject: "root/goal", EscalateToOwner: true}

	higher, ownerRequired := amendmentEscalationRoute(model, []government.AmendmentFinding{areaFinding})
	if ownerRequired || higher.Key() != area.Key() {
		t.Fatalf("valid prior amend-model mandate should route Area-only finding to its Area: higher=%+v ownerRequired=%t", higher, ownerRequired)
	}
	for _, findings := range [][]government.AmendmentFinding{
		{areaFinding, ownerFinding},
		{ownerFinding, areaFinding},
	} {
		higher, ownerRequired = amendmentEscalationRoute(model, findings)
		if !ownerRequired {
			t.Fatal("mixed Owner and Area findings must require the trusted Owner decision")
		}
		if higher.Name != "" {
			t.Fatalf("Owner-required escalation must not expose a lower Area route: %+v", higher)
		}
	}
}
