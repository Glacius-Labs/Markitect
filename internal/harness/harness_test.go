package harness

import (
	"strings"
	"testing"
)

func TestMandateDutiesSeparateModelReadinessFromCandidateConformity(t *testing.T) {
	for _, test := range []struct {
		mandate   string
		required  []string
		forbidden []string
	}{
		{
			mandate:   "assessment",
			required:  []string{"checked model's consistency, sufficiency and realizability", "without implementing", "Missing or outdated artifacts are realization needs", "not grounds to reject model readiness", "use approve or reject for model readiness", "Approve does not certify artifact conformity", "always proceeds through Implement, even when no artifact changes are needed", "Material unresolved model decisions require model_blocker", "Do not use submitted"},
			forbidden: []string{"for the examined realization", "fixed candidate's conformity"},
		},
		{
			mandate:   "verify",
			required:  []string{"independently judges the fixed candidate's conformity", "full Manager hierarchy", "Do not repair the candidate during Verify", "host for a fresh Implement Run followed by a fresh Verify", "use approve or reject for the examined realization", "model_blocker", "Do not use submitted"},
			forbidden: []string{"for model readiness", "always proceeds through Implement"},
		},
		{
			mandate:   "implement",
			required:  []string{"every Manager including Root uses submitted", "Your own judge decision cannot be approve", "parent review of a fixed child commit may approve it"},
			forbidden: []string{"Do not use submitted", "use approve or reject for model readiness"},
		},
	} {
		t.Run(test.mandate, func(t *testing.T) {
			text := mandateDuties(test.mandate)
			for _, required := range test.required {
				if !strings.Contains(text, required) {
					t.Errorf("missing mandate boundary %q", required)
				}
			}
			for _, forbidden := range test.forbidden {
				if strings.Contains(text, forbidden) {
					t.Errorf("wrong mandate boundary %q", forbidden)
				}
			}
		})
	}
}
