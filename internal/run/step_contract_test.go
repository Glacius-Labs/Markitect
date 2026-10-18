package run_test

import (
	"encoding/json"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

var phases = []run.Phase{run.PhaseExplore, run.PhaseWork, run.PhaseIntegrate, run.PhaseReply, run.PhaseReviewerExpectations, run.PhaseReviewerCandidate}

func TestStepContractRoundTripsAndRefusesUnknownPhases(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	run.PhaseExplore:              {"4da24a336165e89d8073fc7c1c53606646abd459b479659c8f8f5940f246fafb", "1f0b174a025fb959c71ff1d14a6271f349dfc68647c9c673f1ea2367b2325377"},
	run.PhaseWork:                 {"e219a08a176d67e7afdfa58c1c3cc8cc4ca14b974fcb5d259293bfcd28d07afe", "b0de462d61e80cc2600af31c9e20b4f25407747261d22b7bc96c23c3fea831a4"},
	run.PhaseIntegrate:            {"f5c4e1ef3fbd27e54ecca2c1d703f0e2c8260629ba97f7f97f87bd30d10f1efc", "b0de462d61e80cc2600af31c9e20b4f25407747261d22b7bc96c23c3fea831a4"},
	run.PhaseReply:                {"97bc6117362b920e19b10bba44f4c7d11f95ba1930e03a8d9cdbc15b3b77b3f6", "3b201b87d143889b9e6824c0b34a1eb101417d65d4d063a76334842d9fad41e7"},
	run.PhaseReviewerExpectations: {"260fcac91c1b7d66fe258ce24fa1216efd4f708a688bcb654eec88973404f109", "3e4a547df0f19182b20de4b11608d8576fb8dbede14aacb9fd10ef3c6c7fdc17"},
	run.PhaseReviewerCandidate:    {"731fe1b7da9210298c7b774e0eddc50fb08b979f922d4bd31c4eb44dc2457de4", "8b271539b08f54a65e6e190c48ab7c3364711bc8dcea435bbca00cbbda88f4a8"},
}

func TestContractAndCatalogBytesArePinned(t *testing.T) {
	t.Parallel()
	for _, phase := range phases {
		c, _ := run.NewStepContract(phase)
		k, _ := run.ToolCatalog(phase)
		if c.Ref().ContentID.String() != golden[phase][0] || k.Ref().ContentID.String() != golden[phase][1] {
			t.Fatal(phase, "recorded contract or catalogue bytes changed")
		}
	}
}
