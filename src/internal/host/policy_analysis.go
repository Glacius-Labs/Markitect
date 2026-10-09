package host

import "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"

const PolicyFailureAnalysisMode = "policy-failure-analysis"

// AnalysisEvidence marks output produced for inspection of a candidate. It is
// not verification or acceptance evidence.
type AnalysisEvidence struct {
	Mode      string            `yaml:"mode"`
	Complete  bool              `yaml:"complete"`
	Base      *AnalysisSnapshot `yaml:"base,omitempty"`
	Candidate AnalysisSnapshot  `yaml:"candidate"`
	Notice    string            `yaml:"notice"`
}

// AnalysisSnapshot binds an analysis side to the same normalized model
// identity exposed by model and reports its two independent validation axes.
type AnalysisSnapshot struct {
	Snapshot            core.ModelSnapshot `yaml:"snapshot"`
	ConfigDigest        string             `yaml:"configDigest"`
	ModelDigest         string             `yaml:"modelDigest"`
	StructuralStatus    string             `yaml:"structuralStatus"`
	PolicyStatus        string             `yaml:"policyStatus"`
	ValidationStatus    string             `yaml:"validationStatus"`
	FailedPolicyResults int                `yaml:"failedPolicyResults"`
}

func analysisSnapshot(model core.SemanticModel) AnalysisSnapshot {
	failed := 0
	for _, result := range model.PolicyResults {
		if result.Status == core.PolicyFailed {
			failed++
		}
	}
	return AnalysisSnapshot{
		Snapshot: model.Snapshot, ConfigDigest: model.ConfigDigest, ModelDigest: model.ModelDigest,
		StructuralStatus: model.StructuralStatus, PolicyStatus: model.PolicyStatus,
		ValidationStatus: model.ValidationStatus, FailedPolicyResults: failed,
	}
}

func policyStatus(structuralStatus string, results []core.PolicyResult) string {
	if structuralStatus != "passed" {
		return "unknown"
	}
	waived := false
	for _, result := range results {
		switch result.Status {
		case core.PolicyFailed:
			return "failed"
		case core.PolicyWaived:
			waived = true
		}
	}
	if waived {
		return "waived"
	}
	return "passed"
}

func cloneDiagnostics(diagnostics []core.Diagnostic) []core.Diagnostic {
	cloned := append([]core.Diagnostic(nil), diagnostics...)
	for i := range cloned {
		if diagnostics[i].PolicyResult != nil {
			ref := *diagnostics[i].PolicyResult
			cloned[i].PolicyResult = &ref
		}
	}
	return cloned
}
