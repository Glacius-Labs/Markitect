package organization_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
)

func requireRenderRefusal(t *testing.T, selection organization.ContextSelection, settings organization.Settings, input organization.RenderInput, code string) *organization.RenderRefusal {
	t.Helper()
	result, err := organization.RenderContext(selection, settings, input)
	var refusal *organization.RenderRefusal
	if !errors.As(err, &refusal) || refusal.Code != code {
		t.Fatalf("wanted %s refusal, got %v", code, err)
	}
	assertZeroRendered(t, result)
	return refusal
}

func TestRenderPreservesCompilerPassedNULBodyLinkAndString(t *testing.T) {
	input := richContextInput(t)
	for i := range input.Root.Entries {
		entry := &input.Root.Entries[i]
		if entry.Path == "model/a/own.yaml" {
			text := strings.Replace(string(entry.Bytes), "body: 'Own [[b/near|Neighbor]] and [[b/near]].'", `body: "[[b/near|A\0B]]"`, 1)
			entry.Bytes = []byte(strings.Replace(text, "note: text", `note: "S\0T"`, 1))
		}
	}
	compiled := compileFixture(t, input)
	selection := requireContext(t, compiled, "a")
	settings := requireSettings(t, requireTopology(t, compiled), settingsFixture(t))
	_, _, renderInput := renderFixture(t, "a", organization.ManagerView)
	result := requireRendered(t, selection, settings, renderInput)
	if bodies := sectionTexts(t, result, "original-body"); !bytes.Equal(bodies[0], []byte("[[b/near|A\x00B]]")) {
		t.Fatalf("canonical NUL body changed: %q", bodies[0])
	}
	var links []model.BodyLink
	if err := json.Unmarshal(sectionTexts(t, result, "original-body-link-occurrences")[0], &links); err != nil || len(links) != 1 || links[0].Display == nil || *links[0].Display != "A\x00B" {
		t.Fatalf("decoded display was repaired: %+v %v", links, err)
	}
	var spec map[string]any
	if err := json.Unmarshal(sectionTexts(t, result, "lossless-typed-spec-not-original-yaml")[0], &spec); err != nil {
		t.Fatal(err)
	}
	payload := spec["payload"].([]any)[1].(map[string]any)
	if payload["note"].([]any)[1] != "S\x00T" {
		t.Fatal("modeled string NUL changed")
	}
}

func TestRenderRejectsContradictoryConfigurationBasis(t *testing.T) {
	selection, settings, input := renderFixture(t, "a", organization.ManagerView)
	t.Run("cap-file-content", func(t *testing.T) {
		changed := input
		changed.MaxContextBytes.Source.ContentID[0] ^= 1
		requireRenderRefusal(t, selection, settings, changed, "invalid-input")
	})
	t.Run("cap-field-value", func(t *testing.T) {
		changed := input
		changed.MaxContextBytes.Source = settingSource("execution.yaml", "/defaults/rejectionLimit")
		requireRenderRefusal(t, selection, settings, changed, "invalid-input")
	})
	compiled := compileFixture(t, richContextInput(t))
	settings = requireSettings(t, requireTopology(t, compiled), settingsFixture(t))
	for _, pkg := range []string{"a", "empty"} {
		t.Run("scope-"+pkg, func(t *testing.T) {
			_, _, scopedInput := renderFixture(t, pkg, organization.ManagerView)
			policy := scopeSelector()
			consistent := requireContext(t, compiled, pkg, policy)
			requireRendered(t, consistent, settings, scopedInput)
			policy.Source.ContentID[0] ^= 1
			conflicting := requireContext(t, compiled, pkg, policy)
			requireRenderRefusal(t, conflicting, settings, scopedInput, "invalid-input")
			policy = scopeSelector()
			policy.Source.Field = "/defaults/rejectionLimit"
			conflicting = requireContext(t, compiled, pkg, policy)
			requireRenderRefusal(t, conflicting, settings, scopedInput, "invalid-input")
		})
	}
	// Even unused catalog facts remain part of the supplied fixed settings basis.
	facts := settingsFixture(t)
	facts.Profiles[2].Source = settingSource("unused-profile.yaml", "/alternate")
	settings = requireSettings(t, requireTopology(t, compiled), facts)
	input.MaxContextBytes.Source = facts.Profiles[2].Source
	selection = requireContext(t, compiled, "a")
	requireRenderRefusal(t, selection, settings, input, "invalid-input")
}

func TestRenderExactInventoryBeyondModelDigest(t *testing.T) {
	input := contextInput(t)
	compiled := compileFixture(t, input)
	selection := requireContext(t, compiled, "a")
	input.Root.Entries = append(input.Root.Entries, model.Entry{Path: "model/newempty", Kind: model.Directory})
	other := compileFixture(t, input)
	if compiled.Models[0].ID != other.Models[0].ID || compiled.Models[0].Digest != other.Models[0].Digest {
		t.Fatal("empty directory must discriminate inventory independently of canonical digest")
	}
	_, _, renderInput := renderFixture(t, "a", organization.ManagerView)
	oldSettings := requireSettings(t, requireTopology(t, compiled), settingsFixture(t))
	newSettings := requireSettings(t, requireTopology(t, other), settingsFixture(t))
	requireRenderRefusal(t, selection, newSettings, renderInput, "invalid-input")
	requireRenderRefusal(t, requireContext(t, other, "a"), oldSettings, renderInput, "invalid-input")
	requireRendered(t, selection, oldSettings, renderInput)
}
