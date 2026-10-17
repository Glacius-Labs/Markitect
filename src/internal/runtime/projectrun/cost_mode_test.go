package projectrun

import (
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
)

func TestCostModeValidationAndUnmeteredCostStaysUnknown(t *testing.T) {
	agent := projectconfig.Agent{Command: "executor", Model: "m", ProviderVersion: "v", Timeout: projectconfig.Duration(time.Minute), MaxStdoutBytes: 1, MaxStderrBytes: 1}
	for _, test := range []struct {
		name     string
		mode     string
		pricing  projectconfig.Pricing
		rejected string
	}{
		{"metered without rates", "", projectconfig.Pricing{}, "at least one positive rate"},
		{"explicit metered", projectconfig.CostModeMetered, projectconfig.Pricing{InputMicrosPerMillion: 1}, ""},
		{"unmetered without rates", projectconfig.CostModeUnmetered, projectconfig.Pricing{}, ""},
		{"unmetered with rates", projectconfig.CostModeUnmetered, projectconfig.Pricing{OutputMicrosPerMillion: 1}, "must not declare pricing"},
		{"unknown mode", "free", projectconfig.Pricing{InputMicrosPerMillion: 1}, "unsupported costMode"},
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
	unmetered.CostMode = projectconfig.CostModeUnmetered
	if cost, known, overflow := estimateAgentCost(usage, unmetered); cost != 0 || known || overflow {
		t.Fatalf("unmetered reported usage was priced: cost=%d known=%v overflow=%v", cost, known, overflow)
	}
	if requiresReportedUsage(unmetered) {
		t.Fatal("unmetered process executor must not require reported usage")
	}
	metered := agent
	metered.Pricing = projectconfig.Pricing{InputMicrosPerMillion: 2, OutputMicrosPerMillion: 3}
	if cost, known, _ := estimateAgentCost(usage, metered); cost != 5 || !known || !requiresReportedUsage(metered) {
		t.Fatalf("metered process executor: cost=%d known=%v requiresUsage=%v", cost, known, requiresReportedUsage(metered))
	}
	native := metered
	native.Transport = projectconfig.TransportCodexAppServer
	if requiresReportedUsage(native) {
		t.Fatal("native App Server missing usage must stay unknown cost, not a stop")
	}
	native.CostMode, native.Pricing = projectconfig.CostModeUnmetered, projectconfig.Pricing{}
	native.Command = filepath.Join(testkit.TempDir(t), "executor")
	native.AppServer = &projectconfig.AppServerSettings{ReasoningEffort: "medium", MaxEventBytes: 1 << 20}
	config := validRuntime()
	config.Agents = map[string]projectconfig.Agent{"manager": native}
	if err := projectconfig.ValidateRuntime(config); err == nil || !strings.Contains(err.Error(), "must be metered") {
		t.Fatalf("unmetered App Server agent error = %v", err)
	}
}
