package run_test

import (
	"encoding/json"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

var phases = []run.Phase{run.PhaseExplore, run.PhaseWork, run.PhaseIntegrate, run.PhaseReply, run.PhaseReviewerExpectations, run.PhaseReviewerCandidate}

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
	if run.PhaseTool(run.PhaseExplore, run.ToolRegister) || run.PhaseTool(run.PhaseReviewerCandidate, run.ToolSubmit) {
		t.Fatal("a phase offers a tool outside its role")
	}
	first, _ := run.ToolCatalog(run.PhaseWork)
	second, _ := run.ToolCatalog(run.PhaseWork)
	if first.Ref() != second.Ref() {
		t.Fatal("catalog bytes are not stable")
	}
}

// golden pins the recorded bytes of every contract and catalogue. A change must
// be deliberate: recorded steps keep their original identities.
var golden = map[run.Phase][2]string{
	run.PhaseExplore:              {"4da24a336165e89d8073fc7c1c53606646abd459b479659c8f8f5940f246fafb", "602757ca7943449f8099da19fee2dacfcc3ac499fc7792477593d9c1d8fce5b4"},
	run.PhaseWork:                 {"e219a08a176d67e7afdfa58c1c3cc8cc4ca14b974fcb5d259293bfcd28d07afe", "c52dea07bad38aa3508ef53c0aebf80db2b696e69dcd0cd9299af8fbf50524af"},
	run.PhaseIntegrate:            {"f5c4e1ef3fbd27e54ecca2c1d703f0e2c8260629ba97f7f97f87bd30d10f1efc", "c52dea07bad38aa3508ef53c0aebf80db2b696e69dcd0cd9299af8fbf50524af"},
	run.PhaseReply:                {"97bc6117362b920e19b10bba44f4c7d11f95ba1930e03a8d9cdbc15b3b77b3f6", "68e941ae6c8b01e13a562f3c5ba1e7d93236f7512c9581c1807ee218fb772b2b"},
	run.PhaseReviewerExpectations: {"260fcac91c1b7d66fe258ce24fa1216efd4f708a688bcb654eec88973404f109", "b40682f0daa353aee0f581964c4ba9d685ad8041719ca42006950883a97c79b8"},
	run.PhaseReviewerCandidate:    {"731fe1b7da9210298c7b774e0eddc50fb08b979f922d4bd31c4eb44dc2457de4", "668c8347c1fd7aac52eb524dbf783cdc9f5fe79489312a79e5088213e40185ce"},
}

func TestContractAndCatalogBytesArePinned(t *testing.T) {
	for _, phase := range phases {
		c, _ := run.NewStepContract(phase)
		k, _ := run.ToolCatalog(phase)
		if c.Ref().ContentID.String() != golden[phase][0] || k.Ref().ContentID.String() != golden[phase][1] {
			t.Fatal(phase, "recorded contract or catalogue bytes changed")
		}
	}
}
