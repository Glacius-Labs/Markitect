package projectrun

import (
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
)

func TestCostModeValidationAndUnmeteredCostStaysUnknown(t *testing.T) {
	agent := Agent{Command: "executor", Model: "m", ProviderVersion: "v", Timeout: Duration(time.Minute), MaxStdoutBytes: 1, MaxStderrBytes: 1}
	for _, test := range []struct {
		name     string
		mode     string
		pricing  Pricing
		rejected string
	}{
		{"metered without rates", "", Pricing{}, "at least one positive rate"},
		{"explicit metered", CostModeMetered, Pricing{InputMicrosPerMillion: 1}, ""},
		{"unmetered without rates", CostModeUnmetered, Pricing{}, ""},
		{"unmetered with rates", CostModeUnmetered, Pricing{OutputMicrosPerMillion: 1}, "must not declare pricing"},
		{"unknown mode", "free", Pricing{InputMicrosPerMillion: 1}, "unsupported costMode"},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := agent
			candidate.CostMode, candidate.Pricing = test.mode, test.pricing
			_, err := candidate.AgentConfig()
			if test.rejected == "" && err != nil || test.rejected != "" && (err == nil || !strings.Contains(err.Error(), test.rejected)) {
				t.Fatalf("AgentConfig error = %v, want %q", err, test.rejected)
			}
		})
	}
	input, output := int64(1_000_000), int64(1_000_000)
	usage := &agentexec.Usage{Source: "provider-reported", InputTokens: &input, OutputTokens: &output}
	unmetered := agent
	unmetered.CostMode = CostModeUnmetered
	if cost, known, overflow := estimateAgentCost(usage, unmetered); cost != 0 || known || overflow {
		t.Fatalf("unmetered reported usage was priced: cost=%d known=%v overflow=%v", cost, known, overflow)
	}
	if unmetered.requiresReportedUsage() {
		t.Fatal("unmetered process executor must not require reported usage")
	}
	metered := agent
	metered.Pricing = Pricing{InputMicrosPerMillion: 2, OutputMicrosPerMillion: 3}
	if cost, known, _ := estimateAgentCost(usage, metered); cost != 5 || !known || !metered.requiresReportedUsage() {
		t.Fatalf("metered process executor: cost=%d known=%v requiresUsage=%v", cost, known, metered.requiresReportedUsage())
	}
	native := metered
	native.Transport = TransportCodexAppServer
	if native.requiresReportedUsage() {
		t.Fatal("native App Server missing usage must stay unknown cost, not a stop")
	}
}
