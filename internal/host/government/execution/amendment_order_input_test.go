package execution

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/government"
)

func TestAmendmentOrderPathRealizationPersistsOwnerEscalationWithoutActors(t *testing.T) {
	opts := amendmentFixtureOptions(t, "amend-model", "invoice", "delivery-rule")
	configPath := filepath.Join(opts.Repo, opts.ConfigPath)
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var source government.Source
	if err := government.Decode(configBytes, &source); err != nil {
		t.Fatal(err)
	}
	artifact := core.DefinitionIdentity{APIVersion: government.APIVersion, Kind: "Artifact", Namespace: "invoice", Name: "issued-order"}
	root := core.DefinitionIdentity{APIVersion: government.APIVersion, Kind: "Area", Namespace: "invoice", Name: "root"}
	subject := core.DefinitionIdentity{APIVersion: "markitect.government-example/v1alpha1", Kind: "Requirement", Namespace: "invoice", Name: "delivery-rule"}
	source.Definitions = append(source.Definitions,
		core.Definition{APIVersion: government.APIVersion, Kind: "Artifact", Metadata: core.Metadata{Namespace: artifact.Namespace, Name: artifact.Name}, Purpose: "Issued order is a canonical immutable input to the delivery rule.", Spec: map[string]any{"path": opts.OrderPath, "class": "canonical", "writer": root}},
		core.Definition{APIVersion: government.APIVersion, Kind: "Realization", Metadata: core.Metadata{Namespace: "invoice", Name: "delivery-order-record"}, Purpose: "The issued order records the delivery rule request.", Spec: map[string]any{"subject": subject, "artifact": artifact, "role": "Records the issued delivery rule request."}},
	)
	updatedConfig, err := json.MarshalIndent(source, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	var normalizedSource government.Source
	if err := government.Decode(updatedConfig, &normalizedSource); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, updatedConfig, 0644); err != nil {
		t.Fatal(err)
	}
	model := government.Compile(normalizedSource)
	if len(model.Findings) != 0 {
		t.Fatalf("test prior model is invalid: %+v", model.Findings)
	}
	orderPath := filepath.Join(opts.Repo, opts.OrderPath)
	orderBytes, err := os.ReadFile(orderPath)
	if err != nil {
		t.Fatal(err)
	}
	var order government.Order
	if err := government.Decode(orderBytes, &order); err != nil {
		t.Fatal(err)
	}
	order.ActiveConstitution = model.Digest
	updatedOrder, err := json.Marshal(order)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(orderPath, updatedOrder, 0644); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) string { return fixtureGit(t, opts.Repo, args...) }
	git("add", opts.ConfigPath, opts.OrderPath)
	git("-c", "commit.gpgsign=false", "commit", "-m", "bind canonical order realization to prior model")
	base := git("rev-parse", "HEAD")
	git("update-ref", opts.Runtime.ActiveRef, base)
	opts.Runtime.ExpectedBase = base

	report, runErr := Run(context.Background(), opts)
	if runErr == nil {
		t.Fatalf("amendment unexpectedly selected and proceeded with its immutable OrderPath: %+v", report)
	}
	if report.Status != "blocked" || report.Stage != "amendment-authority" || len(report.Actors) != 0 || report.Candidate != nil || report.Decision != nil || report.Promotion != nil {
		t.Fatalf("forbidden OrderPath selection should block before actor invocation: err=%v report=%+v", runErr, report)
	}
	if report.AmendmentAssessment == nil || report.AmendmentAssessment.Status != "blocked-escalation-required" || len(report.AmendmentAssessment.Findings) != 1 || report.AmendmentAssessment.Findings[0].Code != "amendment.order-input" || !report.AmendmentAssessment.Findings[0].EscalateToOwner {
		t.Fatalf("blocked OrderPath lacks concrete Owner-required assessment: %+v", report.AmendmentAssessment)
	}
	if len(report.Escalations) != 1 || !report.Escalations[0].OwnerRequired || report.Escalations[0].Round != 0 || report.Escalations[0].PromotionAttempted || report.Escalations[0].OrderDigest != government.Digest(order) || report.Escalations[0].BaseRevision != base {
		t.Fatalf("OrderPath block lacks an exact durable Owner escalation: %+v", report.Escalations)
	}
	var persisted EscalationRecord
	data, err := os.ReadFile(filepath.Join(filepath.Dir(report.ReportPath), "escalation-00.json"))
	if err != nil {
		t.Fatalf("durable Owner escalation record was not written: %v", err)
	}
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.ID != report.Escalations[0].ID || persisted.Findings[0].Code != "amendment.order-input" || !persisted.OwnerRequired {
		t.Fatalf("persisted escalation differs from report or lost required decision: %+v", persisted)
	}
	assertUnchangedActive(t, opts, base)
}

func TestAmendmentBlockedRecursiveDelegationPersistsOwnerEscalation(t *testing.T) {
	opts := recursiveFixtureOptions(t)
	configPath := filepath.Join(opts.Repo, opts.ConfigPath)
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var source government.Source
	if err := government.Decode(configBytes, &source); err != nil {
		t.Fatal(err)
	}
	for index := range source.Definitions {
		definition := &source.Definitions[index]
		if definition.Kind == "Mandate" && (definition.Metadata.Name == "quantity-delegation" || definition.Metadata.Name == "price-delegation") {
			definition.Spec["actions"] = []string{"implement", "review", "amend-model"}
		}
	}
	root := core.DefinitionIdentity{APIVersion: government.APIVersion, Kind: "Area", Namespace: "invoice", Name: "root"}
	subject := core.DefinitionIdentity{APIVersion: "markitect.government-example/v1alpha1", Kind: "Requirement", Namespace: "invoice", Name: "composed-total"}
	artifact := core.DefinitionIdentity{APIVersion: government.APIVersion, Kind: "Artifact", Namespace: "invoice", Name: "government-source"}
	source.Definitions = append(source.Definitions,
		core.Definition{APIVersion: government.APIVersion, Kind: "Artifact", Metadata: core.Metadata{Namespace: artifact.Namespace, Name: artifact.Name}, Purpose: "Canonical Government source used to bind an authorized amendment.", Spec: map[string]any{"path": opts.ConfigPath, "class": "canonical", "writer": root}},
		core.Definition{APIVersion: government.APIVersion, Kind: "Realization", Metadata: core.Metadata{Namespace: "invoice", Name: "total-source"}, Purpose: "The composed total is declared in the canonical Government source.", Spec: map[string]any{"subject": subject, "artifact": artifact, "role": "Declares the composed invoice total requirement."}},
	)
	updatedConfig, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	var normalizedSource government.Source
	if err := government.Decode(updatedConfig, &normalizedSource); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, updatedConfig, 0644); err != nil {
		t.Fatal(err)
	}
	model := government.Compile(normalizedSource)
	if len(model.Findings) != 0 {
		t.Fatalf("recursive amendment prior model is invalid: %+v", model.Findings)
	}
	orderPath := filepath.Join(opts.Repo, opts.OrderPath)
	orderBytes, err := os.ReadFile(orderPath)
	if err != nil {
		t.Fatal(err)
	}
	var order government.Order
	if err := government.Decode(orderBytes, &order); err != nil {
		t.Fatal(err)
	}
	order.ActiveConstitution, order.Action = model.Digest, "amend-model"
	order.Purpose = "Exercise a blocked recursive amendment authority preflight."
	updatedOrder, err := json.Marshal(order)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(orderPath, updatedOrder, 0644); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) string { return fixtureGit(t, opts.Repo, args...) }
	git("add", opts.ConfigPath, opts.OrderPath)
	git("-c", "commit.gpgsign=false", "commit", "-m", "bind canonical source and amendment order")
	base := git("rev-parse", "HEAD")
	git("update-ref", opts.Runtime.ActiveRef, base)
	opts.Runtime.ExpectedBase = base
	opts.Runtime.Amendment = &AmendmentRuntime{MaxRepairs: 0}
	opts.Runtime.Recursion.Limits.MaxCalls = 1

	report, runErr := Run(context.Background(), opts)
	if runErr == nil {
		t.Fatalf("amendment unexpectedly invoked actors despite an insufficient frozen delegation budget: %+v", report)
	}
	if report.Status != "blocked" || report.Stage != "amendment-delegation" || report.Delegation == nil || report.Delegation.Status != "blocked" || len(report.Actors) != 0 || report.Candidate != nil || report.Decision != nil || report.Promotion != nil {
		t.Fatalf("blocked recursive authority should escalate before actors: err=%v report=%+v", runErr, report)
	}
	if report.AmendmentAssessment == nil || report.AmendmentAssessment.Status != "blocked-escalation-required" || len(report.AmendmentAssessment.Findings) == 0 || report.AmendmentAssessment.Findings[0].Code != "delegation.calls" {
		t.Fatalf("delegation conflict lacks concrete blocked assessment findings: %+v", report.AmendmentAssessment)
	}
	if len(report.Escalations) != 1 || !report.Escalations[0].OwnerRequired || report.Escalations[0].DelegationDigest != report.Delegation.Digest || len(report.Escalations[0].DelegationFindings) == 0 || report.Escalations[0].PromotionAttempted {
		t.Fatalf("delegation conflict lacks a concrete Owner escalation: %+v", report.Escalations)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(report.ReportPath), "delegation.json")); err != nil {
		t.Fatalf("blocked delegation record was not persisted: %v", err)
	}
	assertUnchangedActive(t, opts, base)
}
