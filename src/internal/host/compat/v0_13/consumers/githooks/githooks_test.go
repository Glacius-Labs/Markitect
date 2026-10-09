package githooks

import (
	"reflect"
	"strings"
	"testing"

	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
)

func fixtureModel() core.SemanticModel {
	return core.SemanticModel{
		APIVersion:       core.SemanticModelVersion,
		StructuralStatus: "passed",
		Snapshot:         core.ModelSnapshot{ID: "commit-1", Digest: strings.Repeat("a", 64)},
		ModelDigest:      strings.Repeat("b", 64),
		Resources:        []core.ModelResource{{Identity: core.ModelIdentity{Key: "development/Rule/hooks-owner"}}},
	}
}

func TestEmptyScopeIsNotConfigured(t *testing.T) {
	report, err := Check(Input{Model: fixtureModel(), Config: Config{APIVersion: ConfigVersion}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != StatusNotConfigured || len(report.Findings) != 1 || report.Findings[0].Code != "hooks.not-configured" {
		t.Fatalf("empty scope must not imply hook verification: %#v", report)
	}
}

func TestCheckUsesConfiguredExactBytesAndOwnerFacts(t *testing.T) {
	content := []byte("#!/bin/sh\ngo test ./...\n")
	config := Config{APIVersion: ConfigVersion, Hooks: []Hook{{
		Name: "pre-commit-tests", Stage: "pre-commit", Path: ".githooks/pre-commit",
		Digest: digest(content), Owner: "development/Rule/hooks-owner",
	}}}
	report, err := Check(Input{
		Model: fixtureModel(), Config: config,
		Artifacts: map[string][]byte{
			".githooks/pre-commit": content,
			"unconfigured/other":   []byte("ignored"),
		},
		Ownership: map[string][]string{".githooks/pre-commit": {"development/Rule/hooks-owner"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != StatusPassed || len(report.Findings) != 0 {
		t.Fatalf("configured exact hook should pass: %#v", report)
	}
}

func TestCheckReportsMissingStaleAndUnlinkedHook(t *testing.T) {
	config := Config{APIVersion: ConfigVersion, Hooks: []Hook{{
		Name: "commit-message", Stage: "commit-msg", Path: ".githooks/commit-msg",
		Digest: strings.Repeat("c", 64), Owner: "development/Rule/hooks-owner",
	}}}
	report, err := Check(Input{
		Model: fixtureModel(), Config: config,
		Artifacts: map[string][]byte{".githooks/commit-msg": []byte("changed")},
		Ownership: map[string][]string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != StatusFindings || !hasCode(report.Findings, "hooks.digest.mismatch") || !hasCode(report.Findings, "hooks.owner.link-missing") {
		t.Fatalf("stale and unlinked path must be visible: %#v", report)
	}

	config.Hooks[0].Owner = "development/Rule/unknown"
	report, err = Check(Input{Model: fixtureModel(), Config: config})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCode(report.Findings, "hooks.artifact.missing") || !hasCode(report.Findings, "hooks.owner.resource-missing") {
		t.Fatalf("missing bytes and unknown managed owner must be visible: %#v", report)
	}
}

func TestDecodeConfigRejectsUnknownFieldsAndNonExactPaths(t *testing.T) {
	for _, input := range []string{
		"apiVersion: " + ConfigVersion + "\nhooks: []\nextra: true\n",
		"apiVersion: " + ConfigVersion + "\nhooks:\n- name: x\n  stage: pre-commit\n  path: .githooks/*\n  owner: owner\n",
	} {
		if _, err := DecodeConfig([]byte(input)); err == nil {
			t.Fatalf("invalid closed config accepted: %s", input)
		}
	}
}

func hasCode(findings []Finding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func TestSupportedStagesAndCustomPath(t *testing.T) {
	stages := []string{"pre-commit", "pre-push", "commit-msg", "custom"}
	config := Config{APIVersion: ConfigVersion}
	artifacts := map[string][]byte{}
	ownership := map[string][]string{}
	for i, stage := range stages {
		path := ".githooks/entry-" + string(rune('a'+i))
		content := []byte("literal entrypoint")
		config.Hooks = append(config.Hooks, Hook{Name: stage, Stage: stage, Path: path, Digest: digest(content), Owner: "development/Rule/hooks-owner"})
		artifacts[path] = content
		ownership[path] = []string{"development/Rule/hooks-owner"}
	}
	report, err := Check(Input{Model: fixtureModel(), Config: config, Artifacts: artifacts, Ownership: ownership})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != StatusPassed || len(report.Findings) != 0 {
		t.Fatalf("supported stages should pass: %#v", report)
	}
}

func TestFindingOrderIsStableAcrossHookConfigOrder(t *testing.T) {
	hooks := []Hook{
		{Name: "second", Stage: "pre-push", Path: ".githooks/second", Digest: strings.Repeat("c", 64), Owner: "development/Rule/hooks-owner"},
		{Name: "first", Stage: "pre-commit", Path: ".githooks/first", Digest: strings.Repeat("c", 64), Owner: "development/Rule/hooks-owner"},
	}
	one, err := Check(Input{Model: fixtureModel(), Config: Config{APIVersion: ConfigVersion, Hooks: hooks}})
	if err != nil {
		t.Fatal(err)
	}
	two, err := Check(Input{Model: fixtureModel(), Config: Config{APIVersion: ConfigVersion, Hooks: []Hook{hooks[1], hooks[0]}}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(one, two) {
		t.Fatalf("report depends on input order:\n%#v\n%#v", one, two)
	}
}

func TestReportBindsNormalizedConfigWithoutChangingModel(t *testing.T) {
	model := fixtureModel()
	first := Config{APIVersion: ConfigVersion, Hooks: []Hook{{
		Name: "pre-commit-tests", Stage: "pre-commit", Path: ".githooks/pre-commit",
		Digest: strings.Repeat("c", 64), Owner: "development/Rule/hooks-owner",
	}}}
	second := first
	second.Hooks = append([]Hook(nil), first.Hooks...)
	second.Hooks[0].Name = "renamed-pre-commit-tests"
	one, err := Check(Input{Model: model, Config: first})
	if err != nil {
		t.Fatal(err)
	}
	two, err := Check(Input{Model: model, Config: second})
	if err != nil {
		t.Fatal(err)
	}
	if one.ModelDigest != two.ModelDigest || one.SnapshotDigest != two.SnapshotDigest {
		t.Fatal("test must hold the model snapshot fixed")
	}
	if one.ConfigDigest == "" || one.ConfigDigest == two.ConfigDigest {
		t.Fatalf("report must bind the changed module config: first=%q second=%q", one.ConfigDigest, two.ConfigDigest)
	}
}

func TestConfigRejectsPortableCaseFoldPathDuplicates(t *testing.T) {
	config := Config{APIVersion: ConfigVersion, Hooks: []Hook{
		{Name: "upper", Stage: "pre-commit", Path: ".githooks/Pre-Commit", Digest: strings.Repeat("c", 64), Owner: "owner"},
		{Name: "lower", Stage: "pre-commit", Path: ".githooks/pre-commit", Digest: strings.Repeat("c", 64), Owner: "owner"},
	}}
	_, err := Check(Input{Model: fixtureModel(), Config: config})
	if err == nil || !strings.Contains(err.Error(), "portable case folding") {
		t.Fatalf("case-fold colliding hook paths must be rejected: %v", err)
	}
}
