package run_test

import (
	"encoding/json"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

var phases = []string{run.PhaseExplore, run.PhaseWork, run.PhaseIntegration, run.PhaseReply, run.PhaseReviewExpectations, run.PhaseReviewAssessment}

func TestStepContractRoundTripsAndRefusesUnknownPhases(t *testing.T) {
	for _, phase := range phases {
		p, err := run.NewStepContract(phase)
		if err != nil {
			t.Fatal(phase, err)
		}
		c, err := run.DecodeStepContract(p)
		if err != nil || c.Phase != phase || c.Format != run.StepOutcomeFormat || len(c.Outcomes) == 0 {
			t.Fatal(phase, c, err)
		}
		for _, tool := range c.Outcomes {
			if !run.OutcomeTool(phase, tool) || !run.PhaseTool(phase, tool) {
				t.Fatal(phase, "outcome tool not offered", tool)
			}
		}
	}
	if _, err := run.NewStepContract("plan"); err == nil {
		t.Fatal("unknown phase accepted")
	}
	tampered, _ := run.NewPayload(run.StepContractMedia, "utf-8", []byte(`{"Format":"markitect.step-outcome/v1","Phase":"work","Outcomes":["submit"]}`))
	if _, err := run.DecodeStepContract(tampered); err == nil {
		t.Fatal("changed outcomes accepted")
	}
}

func TestToolCatalogOffersValidSchemasPerPhase(t *testing.T) {
	for _, phase := range phases {
		p, err := run.ToolCatalog(phase)
		if err != nil || p.Ref().MediaType != run.ToolCatalogMedia {
			t.Fatal(phase, err)
		}
		var catalog struct {
			Version int            `json:"version"`
			Tools   []run.ToolSpec `json:"tools"`
		}
		if err := json.Unmarshal(p.Bytes(), &catalog); err != nil || catalog.Version != 1 || len(catalog.Tools) == 0 {
			t.Fatal(phase, err)
		}
		for _, tool := range catalog.Tools {
			var schema map[string]any
			if err := json.Unmarshal(tool.InputSchema, &schema); err != nil || schema["type"] != "object" || tool.Description == "" || !run.PhaseTool(phase, tool.Name) {
				t.Fatal(phase, tool.Name, err)
			}
		}
	}
	if run.PhaseTool(run.PhaseExplore, run.ToolRegister) || run.PhaseTool(run.PhaseReviewAssessment, run.ToolSubmit) {
		t.Fatal("a phase offers a tool outside its role")
	}
	first, _ := run.ToolCatalog(run.PhaseWork)
	second, _ := run.ToolCatalog(run.PhaseWork)
	if first.Ref() != second.Ref() {
		t.Fatal("catalog bytes are not stable")
	}
}
