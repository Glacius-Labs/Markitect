package apply_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
)

// Unlike the convenience controlled ports, this port intentionally preserves
// the response's supplied binding, to test missing and cached request echoes.
type rawHostAgentPort func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error)

func TestHostLegacyProfileIdentityRemainsLiteral(t *testing.T) {
	// Measured on assigned base 0392cd2f before AgentImage existed. These literal
	// identities are retained by historical v1/v2 RUN bindings, not rehashed.
	want := map[string]string{
		"agentProfiles": "sha256:a06f3be1ced8e898cdd5198f6da2dee9dbae7652b76f0a08314144fd890180df",
		"checkProfiles": "sha256:887d4ab208cd0396378075cbfb70d4a12bcdca8e755ef9de3fefe191951d0f34",
	}
	selection, raw := hostSettingsInputs(t)
	m := registryModel(t, selection, raw)
	agent, check := hostPorts(t)
	result, err := m.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), agent, check)
	if err != nil || result.Status() != model.Passed {
		t.Fatalf("legacy profile fixture: %v", err)
	}
	if len(result.ProfileObservations()) != len(want) {
		t.Fatal("legacy fixture profile set changed")
	}
	for _, p := range result.ProfileObservations() {
		identity, ok := p.Identity()
		if !ok || identity != want[p.Input().Namespace()] {
			t.Fatalf("legacy literal profile identity changed: %s %s", p.Input().Namespace(), identity)
		}
	}
}

func (p rawHostAgentPort) PrepareAgentProfile(c context.Context, i apply.ProfileInput) (apply.ProfileObservation, error) {
	return p(c, i)
}

func TestHostMetadataRejectsMissingOrOtherRequestsBinding(t *testing.T) {
	selection, raw := hostSettingsInputs(t)
	m := registryModel(t, selection, raw)
	agent, check := hostPorts(t)
	first, err := m.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), agent, check)
	if err != nil || first.Status() != model.Passed {
		t.Fatal("binding baseline failed")
	}
	cached := first.ProfileObservations()[0].Observation()
	for _, missing := range []bool{false, true} {
		otherRaw := m.Inputs().Raw
		otherSelection := m.Request().Selection
		addHostDocument(t, &otherSelection, &otherRaw, "# Different exact request\n"+testHostDocument)
		other := registryModel(t, otherSelection, otherRaw)
		response := cached
		if missing {
			response.InputBinding = apply.ProfileBinding{}
		}
		port := rawHostAgentPort(func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error) { return response, nil })
		result, err := other.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), other.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), port, check)
		if err != nil || result.Status() != model.Failed {
			t.Fatalf("cached/missing echo accepted: %v %+v", err, result.Diagnostics())
		}
		if _, ok := result.Settings(); ok {
			t.Fatal("cached response fabricated new settings binding")
		}
		if _, ok := result.ProfileObservations()[0].Identity(); ok {
			t.Fatal("cached response acquired new comparison identity")
		}
	}
}

func TestHostMetadataInventoryOrderIsCanonicalAndDeletionChangesIdentity(t *testing.T) {
	selection, raw := hostSettingsInputs(t)
	m := registryModel(t, selection, raw)
	_, check := hostPorts(t)
	prepare := func(reverse, remove bool) string {
		port := hostAgentPort(func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error) {
			o := testObservation(t, true)
			o.Inventories[0].Files = append(o.Inventories[0].Files, testMetadata(t, "actual-schemas", "extra.json", "extra schema"))
			o.Inventories = append(o.Inventories, apply.ProfileMetadataInventory{Boundary: "other-schemas", Files: []apply.ProfileMetadataFile{testMetadata(t, "other-schemas", "schema.json", "other schema")}})
			if reverse {
				slices.Reverse(o.Inventories[0].Files)
				slices.Reverse(o.Inventories)
			}
			if remove {
				o.Inventories[0].Files = o.Inventories[0].Files[:1]
			}
			return o, nil
		})
		result, err := m.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), port, check)
		if err != nil || result.Status() != model.Passed {
			t.Fatalf("inventory fixture failed: %v %+v", err, result.Diagnostics())
		}
		identity, _ := result.ProfileObservations()[0].Identity()
		return identity
	}
	baseline := prepare(false, false)
	if baseline != prepare(true, false) {
		t.Fatal("inventory enumeration order changed comparison identity")
	}
	if baseline == prepare(false, true) {
		t.Fatal("deleting schema inventory member did not change identity")
	}
}

func TestHostMetadataRequiresExplicitContractNameAndPositiveVersion(t *testing.T) {
	selection, raw := hostSettingsInputs(t)
	m := registryModel(t, selection, raw)
	for _, agent := range []bool{true, false} {
		for _, contract := range []apply.ProfileContract{
			{Name: "codex-app-server"}, // A nonempty name alone was formerly accepted.
			{Version: 1},
			{Name: "  ", Version: 1},
			{Name: "literal-adapter-contract", Version: 42},
		} {
			a, c := hostPorts(t)
			if agent {
				a = func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error) {
					o := testObservation(t, true)
					o.Contract = contract
					return o, nil
				}
			} else {
				c = func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error) {
					o := testObservation(t, false)
					o.Contract = contract
					return o, nil
				}
			}
			result, err := m.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), a, c)
			valid := contract.Version == 42
			want := model.Failed
			if valid {
				want = model.Passed
			}
			if err != nil || result.Status() != want {
				t.Fatalf("contract %+v agent=%v: %v %s %+v", contract, agent, err, result.Status(), result.Diagnostics())
			}
			for _, profile := range result.ProfileObservations() {
				if (profile.Input().Namespace() == "agentProfiles") != agent {
					continue
				}
				if _, ok := profile.Identity(); ok != valid {
					t.Fatal("invalid contract acquired an identity, or Apply imposed an adapter version registry")
				}
				if profile.Observation().Contract != contract {
					t.Fatal("literal contract values were normalized or lost")
				}
			}
			if !valid {
				if agent {
					if _, ok := result.Settings(); ok {
						t.Fatal("invalid Agent contract exposed settings")
					}
				} else {
					if _, ok := result.Checks(); ok {
						t.Fatal("invalid Check contract exposed selections")
					}
				}
			}
		}
	}
}
