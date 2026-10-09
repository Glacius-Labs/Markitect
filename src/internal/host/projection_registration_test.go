package host

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/authoring"
)

func projectionRegistration(version string, config map[string]any) *Project {
	return &Project{Resources: []*authoring.Resource{{
		Core: authoring.Core{Kind: "Project"},
		Spec: authoring.Spec{Adapters: []authoring.AdapterConfig{{Name: "projection", Type: "local-projection", Version: version, Config: config}}},
	}}}
}

func TestRegisteredProjectionReadsOneVersionedRegistration(t *testing.T) {
	p := projectionRegistration("v1alpha1", map[string]any{"contracts": "arch/contracts.config", "coverage": "arch/coverage.yml"})
	config, coverage, active, err := RegisteredProjection(p)
	if err != nil {
		t.Fatal(err)
	}
	if !active || config != "arch/contracts.config" || coverage != "arch/coverage.yml" {
		t.Fatalf("registration = (%q, %q, %v), want exact configured paths and active", config, coverage, active)
	}

	p.Resources[0].Spec.Adapters = nil
	config, coverage, active, err = RegisteredProjection(p)
	if err != nil || active || config != "" || coverage != "" {
		t.Fatalf("unregistered Project should be inactive, got (%q, %q, %v, %v)", config, coverage, active, err)
	}
}

func TestRegisteredProjectionRejectsInvalidRegistrationDeterministically(t *testing.T) {
	tests := []struct {
		name, version, want string
		config              map[string]any
		count               int
	}{
		{name: "unsupported version", version: "v2", want: "version must be v1alpha1", config: map[string]any{"contracts": "a.config", "coverage": "b.yml"}},
		{name: "unknown key", version: "v1alpha1", want: "exactly contracts and coverage", config: map[string]any{"contracts": "a.config", "coverage": "b.yml", "other": "x"}},
		{name: "missing coverage", version: "v1alpha1", want: "exactly contracts and coverage", config: map[string]any{"contracts": "a.config"}},
		{name: "unsafe path", version: "v1alpha1", want: "unsafe path component", config: map[string]any{"contracts": "../a.config", "coverage": "b.yml"}},
		{name: "parent alias", version: "v1alpha1", want: "non-overlapping", config: map[string]any{"contracts": "arch", "coverage": "arch/coverage.yml"}},
		{name: "multiple registrations", version: "v1alpha1", want: "multiple local-projection", config: map[string]any{"contracts": "a.config", "coverage": "b.yml"}, count: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := projectionRegistration(test.version, test.config)
			if test.count == 2 {
				p.Resources[0].Spec.Adapters = append(p.Resources[0].Spec.Adapters, p.Resources[0].Spec.Adapters[0])
			}
			for i := 0; i < 3; i++ {
				_, _, active, err := RegisteredProjection(p)
				if err == nil || !strings.Contains(err.Error(), test.want) || active {
					t.Fatalf("call %d expected inactive error containing %q, got active=%v err=%v", i, test.want, active, err)
				}
			}
		})
	}
}

func TestRegisteredProjectionRequiresExactlyOneProjectResource(t *testing.T) {
	if _, _, _, err := RegisteredProjection(nil); err == nil {
		t.Fatal("nil Project should be rejected")
	}
	if _, _, _, err := RegisteredProjection(&Project{}); err == nil {
		t.Fatal("missing Project resource should be rejected")
	}
	p := projectionRegistration("v1alpha1", map[string]any{"contracts": "a.config", "coverage": "b.yml"})
	p.Resources = append(p.Resources, p.Resources[0])
	if _, _, _, err := RegisteredProjection(p); err == nil {
		t.Fatal("multiple Project resources should be rejected")
	}
}
