package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type opsOracleCase struct {
	Name     string   `json:"name"`
	Root     string   `json:"root,omitempty"`
	Text     string   `json:"text,omitempty"`
	Hook     string   `json:"hook,omitempty"`
	CI       string   `json:"ci,omitempty"`
	Commands []string `json:"commands,omitempty"`
	Declared bool     `json:"declared,omitempty"`
	Pass     bool     `json:"pass"`
}

type opsOracleCases struct {
	A        []opsOracleCase `json:"a"`
	B        []opsOracleCase `json:"b"`
	Runbooks []opsOracleCase `json:"runbooks"`
	Gates    []opsOracleCase `json:"gates"`
	Scopes   []opsScopeCase  `json:"scopes"`
}

type opsScopePathCase struct {
	Path    string `json:"path"`
	InScope bool   `json:"in_scope"`
}

type opsScopeCase struct {
	Name       string             `json:"name"`
	Task       string             `json:"task"`
	Prior      string             `json:"prior"`
	Arm        string             `json:"arm"`
	TaskScopes []string           `json:"task_scopes"`
	P01Scopes  []string           `json:"p01_scopes"`
	P02Scopes  []string           `json:"p02_scopes"`
	Want       []string           `json:"want"`
	Paths      []opsScopePathCase `json:"paths"`
	Reject     bool               `json:"reject,omitempty"`
}

func TestEngineeringOpsOracleV2Contracts(t *testing.T) {
	pwsh, err := exec.LookPath("pwsh")
	if err != nil {
		if os.Getenv("GITHUB_ACTIONS") == "true" {
			t.Fatal("CI must execute the Operations oracle v2 contract; pwsh is unavailable")
		}
		t.Skip("pwsh is not installed")
	}
	if _, err := exec.LookPath("python"); err != nil {
		if os.Getenv("GITHUB_ACTIONS") == "true" {
			t.Fatal("CI must execute the bounded Python AST contract; python is unavailable")
		}
		t.Skip("python is not installed")
	}

	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate Operations oracle")
	}
	oraclePath, err := filepath.Abs(filepath.Join(filepath.Dir(sourceFile), "..", "projects", "engineering-ops", "oracle", "evaluate.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	cases := opsOracleCases{}
	root := t.TempDir()
	projectRoot := filepath.Dir(filepath.Dir(oraclePath))
	taskSetPath := filepath.Join(projectRoot, "tasks", "task-set.yaml")
	parallelTaskSetPath := filepath.Join(projectRoot, "parallel-task-set.yaml")
	task08A := readOpsTaskScopes(t, taskSetPath, "08", "a")
	task08B := readOpsTaskScopes(t, taskSetPath, "08", "b")
	p01A := readOpsTaskScopes(t, parallelTaskSetPath, "P01", "a")
	p01B := readOpsTaskScopes(t, parallelTaskSetPath, "P01", "b")
	p02A := readOpsTaskScopes(t, parallelTaskSetPath, "P02", "a")
	p02B := readOpsTaskScopes(t, parallelTaskSetPath, "P02", "b")
	cases.Scopes = append(cases.Scopes,
		opsScopeCase{Name: "ordinary task 08 A keeps its own scope", Task: "08", Arm: "a", TaskScopes: task08A, P01Scopes: p01A, P02Scopes: p02A, Want: task08A, Paths: []opsScopePathCase{{Path: "docs/operations/runbooks/lease-renewal.md", InScope: true}, {Path: "scripts/validate.py", InScope: false}}},
		opsScopeCase{Name: "integration task 08 A unions only P01 and P02 scopes", Task: "08", Prior: "06", Arm: "a", TaskScopes: task08A, P01Scopes: p01A, P02Scopes: p02A, Want: opsScopeUnion(p01A, p02A), Paths: []opsScopePathCase{{Path: "scripts/validate.py", InScope: true}, {Path: ".githooks/pre-commit", InScope: true}, {Path: "docs/operations/runbooks/lease-renewal.md", InScope: true}, {Path: ".markitect/areas/operations/verification.rule.yaml", InScope: false}}},
		opsScopeCase{Name: "ordinary task 08 B keeps its own scope", Task: "08", Arm: "b", TaskScopes: task08B, P01Scopes: p01B, P02Scopes: p02B, Want: task08B, Paths: []opsScopePathCase{{Path: ".markitect/areas/operations/verification.rule.yaml", InScope: true}, {Path: ".githooks/pre-commit", InScope: false}}},
		opsScopeCase{Name: "integration task 08 B unions only P01 and P02 scopes", Task: "08", Prior: "06", Arm: "b", TaskScopes: task08B, P01Scopes: p01B, P02Scopes: p02B, Want: opsScopeUnion(p01B, p02B), Paths: []opsScopePathCase{{Path: "scripts/validate.py", InScope: true}, {Path: ".agents/skills/engineering-operations/SKILL.md", InScope: true}, {Path: ".claude/CLAUDE.md", InScope: false}}},
		opsScopeCase{Name: "P01 A retains its own scope", Task: "P01", Prior: "06", Arm: "a", TaskScopes: p01A, P01Scopes: p01A, P02Scopes: p02A, Want: p01A, Paths: []opsScopePathCase{{Path: ".github/workflows/ci.yaml", InScope: true}, {Path: "docs/operations/runbooks/lease-renewal.md", InScope: false}}},
		opsScopeCase{Name: "P01 B retains its own scope", Task: "P01", Prior: "06", Arm: "b", TaskScopes: p01B, P01Scopes: p01B, P02Scopes: p02B, Want: p01B, Paths: []opsScopePathCase{{Path: ".githooks/pre-commit", InScope: true}, {Path: ".agents/skills/engineering-operations/SKILL.md", InScope: false}}},
		opsScopeCase{Name: "P02 A retains its own scope", Task: "P02", Prior: "06", Arm: "a", TaskScopes: p02A, P01Scopes: p01A, P02Scopes: p02A, Want: p02A, Paths: []opsScopePathCase{{Path: "docs/engineering/managed-files.yaml", InScope: true}, {Path: "scripts/validate.py", InScope: false}}},
		opsScopeCase{Name: "P02 B retains its own scope", Task: "P02", Prior: "06", Arm: "b", TaskScopes: p02B, P01Scopes: p01B, P02Scopes: p02B, Want: p02B, Paths: []opsScopePathCase{{Path: ".agents/skills/engineering-operations/SKILL.md", InScope: true}, {Path: ".githooks/pre-commit", InScope: false}}},
		opsScopeCase{Name: "reject unsupported task 08 prior marker", Task: "08", Prior: "05", Arm: "a", TaskScopes: task08A, P01Scopes: p01A, P02Scopes: p02A, Reject: true},
		opsScopeCase{Name: "reject unsupported ordinary task prior marker", Task: "07", Prior: "06", Arm: "a", TaskScopes: task08A, P01Scopes: p01A, P02Scopes: p02A, Reject: true},
		opsScopeCase{Name: "reject unsupported P01 prior marker", Task: "P01", Prior: "05", Arm: "a", TaskScopes: p01A, P01Scopes: p01A, P02Scopes: p02A, Reject: true},
	)

	for _, label := range []string{"Root module vet", "Go vet", "Static analysis"} {
		caseRoot := filepath.Join(root, "a-positive-"+strings.ReplaceAll(label, " ", "-"))
		writeOpsAFixture(t, caseRoot, label, "go vet ./...", []string{
			"Operations policy", "Operations checker tests", "Root module tests", "Nested module tests", label,
		})
		cases.A = append(cases.A, opsOracleCase{Name: "A accepts mapped vet label " + label, Root: caseRoot, Pass: true})
	}
	missingVet := filepath.Join(root, "a-missing-vet-assertion")
	writeOpsAFixture(t, missingVet, "Go vet", "go vet ./...", []string{
		"Operations policy", "Operations checker tests", "Root module tests", "Nested module tests",
	})
	cases.A = append(cases.A, opsOracleCase{Name: "A rejects vet key omitted from literal expectation", Root: missingVet})
	wrongMap := filepath.Join(root, "a-misbound-vet")
	writeOpsAFixture(t, wrongMap, "Go vet", "go test ./...", []string{
		"Operations policy", "Operations checker tests", "Root module tests", "Nested module tests", "Go vet",
	})
	cases.A = append(cases.A, opsOracleCase{Name: "A rejects vet key mapped to another command", Root: wrongMap})
	missingOriginal := filepath.Join(root, "a-missing-original")
	writeOpsAFixture(t, missingOriginal, "Go vet", "go vet ./...", []string{
		"Operations policy", "Operations checker tests", "Nested module tests", "Go vet",
	})
	cases.A = append(cases.A, opsOracleCase{Name: "A rejects removal of an original check name", Root: missingOriginal})
	misboundOriginal := filepath.Join(root, "a-misbound-original")
	writeOpsAFixture(t, misboundOriginal, "Go vet", "go vet ./...", []string{
		"Operations policy", "Operations checker tests", "Root module tests", "Nested module tests", "Go vet",
	})
	mutateOpsFile(t, misboundOriginal, "scripts/check_operations.py", func(s string) string {
		return strings.Replace(s, "python scripts/check_operations.py", "python scripts/other.py", 1)
	})
	cases.A = append(cases.A, opsOracleCase{Name: "A rejects drift in an original name-to-command mapping", Root: misboundOriginal})
	decoy := filepath.Join(root, "a-comment-decoy")
	writeOpsAFixture(t, decoy, "Go vet", "go vet ./...", []string{
		"Operations policy", "Operations checker tests", "Root module tests", "Nested module tests", "Other vet",
	})
	testFile := filepath.Join(decoy, "scripts", "test_check_operations.py")
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	data = append([]byte("# Go vet and go vet ./... are only decoys\n"), data...)
	if err := os.WriteFile(testFile, data, 0o600); err != nil {
		t.Fatal(err)
	}
	cases.A = append(cases.A, opsOracleCase{Name: "A rejects comments as assertion evidence", Root: decoy})
	deadCode := filepath.Join(root, "a-dead-code-assertion")
	writeOpsAFixture(t, deadCode, "Go vet", "go vet ./...", []string{
		"Operations policy", "Operations checker tests", "Root module tests", "Nested module tests", "Go vet",
	})
	mutateOpsFile(t, deadCode, "scripts/test_check_operations.py", func(s string) string {
		return strings.Replace(s, "        self.assertEqual(set(EXPECTED_RUNS), {", "        if False:\n            self.assertEqual(set(EXPECTED_RUNS), {", 1)
	})
	cases.A = append(cases.A, opsOracleCase{Name: "A rejects assertion buried in a dead branch", Root: deadCode})
	unreachable := filepath.Join(root, "a-unreachable-assertion")
	writeOpsAFixture(t, unreachable, "Go vet", "go vet ./...", []string{
		"Operations policy", "Operations checker tests", "Root module tests", "Nested module tests", "Go vet",
	})
	mutateOpsFile(t, unreachable, "scripts/test_check_operations.py", func(s string) string {
		return strings.Replace(s, "        self.assertEqual(set(EXPECTED_RUNS), {", "        return\n        self.assertEqual(set(EXPECTED_RUNS), {", 1)
	})
	cases.A = append(cases.A, opsOracleCase{Name: "A rejects assertion after direct return", Root: unreachable})
	for _, terminator := range []string{"return", "raise AssertionError()"} {
		guarded := filepath.Join(root, "a-literal-true-"+strings.ReplaceAll(terminator, " ", "-"))
		writeOpsAFixture(t, guarded, "Go vet", "go vet ./...", []string{
			"Operations policy", "Operations checker tests", "Root module tests", "Nested module tests", "Go vet",
		})
		mutateOpsFile(t, guarded, "scripts/test_check_operations.py", func(s string) string {
			return strings.Replace(s, "        self.assertEqual(set(EXPECTED_RUNS), {", "        if True:\n            "+terminator+"\n        self.assertEqual(set(EXPECTED_RUNS), {", 1)
		})
		cases.A = append(cases.A, opsOracleCase{Name: "A rejects assertion after literal if True terminator " + terminator, Root: guarded})
	}

	validB := filepath.Join(root, "b-valid")
	writeOpsBFixture(t, validB)
	cases.B = append(cases.B, opsOracleCase{Name: "B accepts prose guidance routed to explicit check owners", Root: validB, Pass: true})
	semanticGuidance := filepath.Join(root, "b-literal-guidance")
	writeOpsBFixture(t, semanticGuidance)
	mutateOpsFile(t, semanticGuidance, ".markitect/areas/operations/verification.rule.yaml", func(s string) string {
		return strings.Replace(s, "CI and pre-commit both run go vet, root Go tests, and nested-module Go tests. Run the configured Markitect artifact and hook/pipeline checks.", "CI and pre-commit both run go vet ./..., go test ./..., and go -C tools/process-sentinel test ./.... Markitect verification is configured in the project.", 1)
	})
	cases.B = append(cases.B, opsOracleCase{Name: "B accepts literal command guidance without mandated semantic wording", Root: semanticGuidance, Pass: true})
	novelVetLabel := filepath.Join(root, "b-novel-vet-label")
	writeOpsBFixture(t, novelVetLabel)
	mutateOpsFile(t, novelVetLabel, "markitect.yaml", func(s string) string {
		return strings.Replace(s, "root-vet", "static-analysis", 1)
	})
	mutateOpsFile(t, novelVetLabel, ".markitect/modules/pipelines.config", func(s string) string {
		return strings.Replace(s, "root-vet", "static-analysis", 1)
	})
	cases.B = append(cases.B, opsOracleCase{Name: "B binds a novel vet check label to its explicit pipeline link", Root: novelVetLabel, Pass: true})
	missingInput := filepath.Join(root, "b-rule-missing-pipeline-input")
	writeOpsBFixture(t, missingInput)
	mutateOpsFile(t, missingInput, ".markitect/areas/operations/verification.rule.yaml", func(s string) string {
		return strings.ReplaceAll(s, "        - .markitect/modules/pipelines.config\n", "")
	})
	cases.B = append(cases.B, opsOracleCase{Name: "B rejects absent declared pipeline input", Root: missingInput})
	wrongOwner := filepath.Join(root, "b-wrong-module-owner")
	writeOpsBFixture(t, wrongOwner)
	mutateOpsFile(t, wrongOwner, ".markitect/modules/pipelines.config", func(s string) string {
		return strings.ReplaceAll(s, "owner: operations/Rule/verification", "owner: operations/Rule/process-lifecycle")
	})
	cases.B = append(cases.B, opsOracleCase{Name: "B rejects wrong canonical module owner", Root: wrongOwner})
	wrongArgv := filepath.Join(root, "b-wrong-project-argv")
	writeOpsBFixture(t, wrongArgv)
	mutateOpsFile(t, wrongArgv, "markitect.yaml", func(s string) string {
		return strings.Replace(s, "        - vet\n", "        - test\n", 1)
	})
	cases.B = append(cases.B, opsOracleCase{Name: "B rejects misconfigured Project vet check", Root: wrongArgv})
	wrongVetLink := filepath.Join(root, "b-wrong-vet-link")
	writeOpsBFixture(t, wrongVetLink)
	mutateOpsFile(t, wrongVetLink, ".markitect/modules/pipelines.config", func(s string) string {
		return strings.Replace(s, "name: root-vet", "name: stale-vet-name", 1)
	})
	cases.B = append(cases.B, opsOracleCase{Name: "B rejects a vet check without a matching explicit pipeline key", Root: wrongVetLink})
	wrongPointer := filepath.Join(root, "b-wrong-ci-run-pointer")
	writeOpsBFixture(t, wrongPointer)
	mutateOpsFile(t, wrongPointer, ".markitect/modules/pipelines.config", func(s string) string {
		return strings.Replace(s, "/jobs/quality/steps/3/run", "/jobs/quality/steps/0/run", 1)
	})
	cases.B = append(cases.B, opsOracleCase{Name: "B rejects vet pointer resolving to uses-only checkout step", Root: wrongPointer})
	duplicateRun := filepath.Join(root, "b-duplicate-vet-run")
	writeOpsBFixture(t, duplicateRun)
	mutateOpsFile(t, duplicateRun, ".github/workflows/ci.yaml", func(s string) string {
		return strings.Replace(s, "        run: go vet ./...\n", "        run: go vet ./...\n        run: go test ./...\n", 1)
	})
	cases.B = append(cases.B, opsOracleCase{Name: "B rejects duplicate run fields at configured pointer", Root: duplicateRun})
	falseVetStep := filepath.Join(root, "b-vet-step-if-false")
	writeOpsBFixture(t, falseVetStep)
	mutateOpsFile(t, falseVetStep, ".github/workflows/ci.yaml", func(s string) string {
		return strings.Replace(s, "        run: go vet ./...\n", "        run: go vet ./...\n        if: false\n", 1)
	})
	cases.B = append(cases.B, opsOracleCase{Name: "B rejects configured vet pointer in an if-false CI step", Root: falseVetStep})
	ignoredVetStep := filepath.Join(root, "b-vet-step-continue-on-error")
	writeOpsBFixture(t, ignoredVetStep)
	mutateOpsFile(t, ignoredVetStep, ".github/workflows/ci.yaml", func(s string) string {
		return strings.Replace(s, "        run: go vet ./...\n", "        run: go vet ./...\n        continue-on-error: true\n", 1)
	})
	cases.B = append(cases.B, opsOracleCase{Name: "B rejects configured vet pointer in a continue-on-error CI step", Root: ignoredVetStep})
	commentedCIVet := filepath.Join(root, "b-comment-only-ci-vet")
	writeOpsBFixture(t, commentedCIVet)
	mutateOpsFile(t, commentedCIVet, ".github/workflows/ci.yaml", func(s string) string {
		return strings.Replace(s, "        run: go vet ./...", "        # run: go vet ./...", 1)
	})
	cases.B = append(cases.B, opsOracleCase{Name: "B rejects a comment-only CI vet command", Root: commentedCIVet})
	commentedPointer := filepath.Join(root, "b-comment-only-pointer")
	writeOpsBFixture(t, commentedPointer)
	mutateOpsFile(t, commentedPointer, ".markitect/modules/pipelines.config", func(s string) string {
		return strings.Replace(s, "        yamlPath: /jobs/quality/steps/3/run", "        # yamlPath: /jobs/quality/steps/3/run", 1)
	})
	cases.B = append(cases.B, opsOracleCase{Name: "B rejects comment-only expectedChecks pointer", Root: commentedPointer})
	missingGate := filepath.Join(root, "b-hook-missing-vet")
	writeOpsBFixture(t, missingGate)
	mutateOpsFile(t, missingGate, ".githooks/pre-commit", func(s string) string { return strings.ReplaceAll(s, "go vet ./...\n", "") })
	cases.B = append(cases.B, opsOracleCase{Name: "B rejects vet missing from a gate", Root: missingGate})
	exitedHook := filepath.Join(root, "b-hook-exits-before-gates")
	writeOpsBFixture(t, exitedHook)
	mutateOpsFile(t, exitedHook, ".githooks/pre-commit", func(s string) string { return "#!/bin/sh\nexit 1\n" + s })
	cases.B = append(cases.B, opsOracleCase{Name: "B rejects hook gates placed after unconditional exit", Root: exitedHook})
	functionHook := filepath.Join(root, "b-hook-gate-only-in-function")
	writeOpsBFixture(t, functionHook)
	mutateOpsFile(t, functionHook, ".githooks/pre-commit", func(s string) string {
		return strings.Replace(s, "go vet ./...", "check_later() {\n  go vet ./...\n}", 1)
	})
	cases.B = append(cases.B, opsOracleCase{Name: "B rejects vet command contained only in hook function", Root: functionHook})
	missingCIVet := filepath.Join(root, "b-ci-missing-vet")
	writeOpsBFixture(t, missingCIVet)
	mutateOpsFile(t, missingCIVet, ".github/workflows/ci.yaml", func(s string) string {
		return strings.ReplaceAll(s, "      - name: Root module vet\n        run: go vet ./...\n", "")
	})
	cases.B = append(cases.B, opsOracleCase{Name: "B rejects vet missing from CI independently of hook", Root: missingCIVet})
	commentedHookVet := filepath.Join(root, "b-comment-only-hook-vet")
	writeOpsBFixture(t, commentedHookVet)
	mutateOpsFile(t, commentedHookVet, ".githooks/pre-commit", func(s string) string {
		return strings.Replace(s, "go vet ./...\n", "# go vet ./...\n", 1)
	})
	cases.B = append(cases.B, opsOracleCase{Name: "B rejects a comment-only hook vet command", Root: commentedHookVet})
	missingProjection := filepath.Join(root, "b-ci-missing-projection")
	writeOpsBFixture(t, missingProjection)
	mutateOpsFile(t, missingProjection, ".github/workflows/ci.yaml", func(s string) string {
		return strings.Replace(s, "        run: markitect verify --repo . --revision $GITHUB_SHA", "        # run: markitect verify --repo . --revision $GITHUB_SHA", 1)
	})
	cases.B = append(cases.B, opsOracleCase{Name: "B rejects comment-only CI projection command", Root: missingProjection})
	missingArtifactHook := filepath.Join(root, "b-hook-missing-artifact")
	writeOpsBFixture(t, missingArtifactHook)
	mutateOpsFile(t, missingArtifactHook, ".githooks/pre-commit", func(s string) string {
		return strings.Replace(s, "markitect-check-artifacts --repo . --config markitect-artifacts.yaml", "# markitect-check-artifacts --repo . --config markitect-artifacts.yaml", 1)
	})
	cases.B = append(cases.B, opsOracleCase{Name: "B rejects comment-only artifact command", Root: missingArtifactHook})
	missingRoute := filepath.Join(root, "b-provider-route-missing")
	writeOpsBFixture(t, missingRoute)
	mutateOpsFile(t, missingRoute, ".agents/skills/engineering-operations/SKILL.md", func(s string) string {
		return strings.ReplaceAll(s, "verification.rule.yaml", "other.rule.yaml")
	})
	cases.B = append(cases.B, opsOracleCase{Name: "B rejects missing provider route to canonical Rule", Root: missingRoute})

	validRunbook := `# Lease renewal runbook

Use the same request ID for every retry of one renewal. The service uses that ID to recognize a repeated operation; a retry must not create a second renewal. Keep the request ID with the operation record for the full retry and reconciliation period, and retain it until the final outcome is known.

If a renewal times out, report the operation status to the service owner. Check the operation using its original request ID before retrying. If the outcome still cannot be confirmed, retry with that same ID and record the result.

Logs may include the request ID and operation status needed for reconciliation. Never log secrets, credentials, or request or response payloads.`
	cases.Runbooks = append(cases.Runbooks,
		opsOracleCase{Name: "runbook accepts idempotent behavior without keyword", Text: validRunbook, Declared: true, Pass: true},
		opsOracleCase{Name: "runbook accepts antecedent request ID reused for the same renewal", Text: strings.Replace(validRunbook, "Use the same request ID for every retry of one renewal.", "Assign a stable request ID to the renewal. Retries for the same renewal must reuse it.", 1), Declared: true, Pass: true},
		opsOracleCase{Name: "runbook accepts reviewed logical-renewal ID antecedent and retry reuse", Text: strings.Replace(validRunbook, "Use the same request ID for every retry of one renewal. The service uses that ID to recognize a repeated operation; a retry must not create a second renewal.", "Give each logical renewal a stable request ID before sending it. Treat that ID as the idempotency key: retries for the same renewal must reuse it so a retry cannot create a second renewal.", 1), Declared: true, Pass: true},
		opsOracleCase{Name: "runbook accepts that-ID reuse without every/each/all adjacency", Text: strings.Replace(validRunbook, "Use the same request ID for every retry of one renewal.", "Assign a request ID before sending the renewal. A retry of that same renewal must reuse that ID.", 1), Declared: true, Pass: true},
		opsOracleCase{Name: "runbook rejects reviewed antecedent without an ID reuse rule", Text: strings.Replace(validRunbook, "Use the same request ID for every retry of one renewal. The service uses that ID to recognize a repeated operation; a retry must not create a second renewal.", "Give each logical renewal a stable request ID before sending it. Treat that ID as the idempotency key: retries for the same renewal cannot create a second renewal.", 1), Declared: true},
		opsOracleCase{Name: "runbook rejects a different ID on same-renewal retry", Text: strings.Replace(validRunbook, "Use the same request ID for every retry of one renewal. The service uses that ID to recognize a repeated operation; a retry must not create a second renewal.", "Give each logical renewal a stable request ID before sending it. Treat that ID as the idempotency key: retries for the same renewal must use a different request ID.", 1), Declared: true},
		opsOracleCase{Name: "runbook accepts explanatory fresh-ID duplicate risk followed by correction", Text: validRunbook + " A retry with a fresh ID may create a duplicate; therefore reuse the original ID.", Declared: true, Pass: true},
		opsOracleCase{Name: "runbook rejects fresh IDs on each retry", Text: strings.Replace(validRunbook, "same request ID for every retry", "a fresh request ID for every retry", 1), Declared: true},
		opsOracleCase{Name: "runbook rejects duplicate renewal semantics", Text: strings.Replace(validRunbook, "must not create a second renewal", "may create a second renewal", 1), Declared: true},
		opsOracleCase{Name: "runbook rejects unsafe normative fresh-ID duplicate advice", Text: validRunbook + " Retry with a fresh ID; this may create a duplicate renewal.", Declared: true},
		opsOracleCase{Name: "runbook rejects normative fresh-ID advice even when later contradicted", Text: validRunbook + " Retry with a fresh ID; this may create a duplicate renewal, therefore reuse the original ID.", Declared: true},
		opsOracleCase{Name: "runbook rejects fresh ID for every retry despite other reuse text", Text: validRunbook + " Every retry for the same renewal must use a fresh request ID.", Declared: true},
		opsOracleCase{Name: "runbook rejects same-renewal retry without an ID reuse rule", Text: strings.Replace(validRunbook, "Use the same request ID for every retry of one renewal.", "Assign a stable request ID to the renewal. Retries for that same renewal may use any request ID.", 1), Declared: true},
		opsOracleCase{Name: "runbook rejects secret and payload logging", Text: strings.Replace(validRunbook, "Never log secrets, credentials, or request or response payloads.", "Logs may include secrets, credentials, and request or response payloads.", 1), Declared: true},
		opsOracleCase{Name: "runbook accepts secrets and payloads staying out of logs", Text: strings.Replace(validRunbook, "Never log secrets, credentials, or request or response payloads.", "Secrets and payloads stay out of logs.", 1), Declared: true, Pass: true},
		opsOracleCase{Name: "runbook accepts keeping secrets and payloads out of logs", Text: strings.Replace(validRunbook, "Never log secrets, credentials, or request or response payloads.", "Keep secrets and payloads out of logs.", 1), Declared: true, Pass: true},
		opsOracleCase{Name: "runbook accepts curly-apostrophe don't log secrets or payloads", Text: strings.Replace(validRunbook, "Never log secrets, credentials, or request or response payloads.", "Don’t log secrets or payloads.", 1), Declared: true, Pass: true},
		opsOracleCase{Name: "runbook rejects missing canonical input declaration", Text: validRunbook, Declared: false},
		opsOracleCase{Name: "runbook accepts request ID retained for explicit duration", Text: strings.Replace(validRunbook, "Keep the request ID with the operation record for the full retry and reconciliation period, and retain it until the final outcome is known.", "Keep the request ID for 30 days.", 1), Declared: true, Pass: true},
		opsOracleCase{Name: "runbook accepts request ID scoped to operation record", Text: strings.Replace(validRunbook, "Keep the request ID with the operation record for the full retry and reconciliation period, and retain it until the final outcome is known.", "Keep the request ID in the operation record.", 1), Declared: true, Pass: true},
		opsOracleCase{Name: "runbook accepts retention purpose tied to retry", Text: strings.Replace(validRunbook, "Keep the request ID with the operation record for the full retry and reconciliation period, and retain it until the final outcome is known.", "Store the request ID so retries can reuse the original operation.", 1), Declared: true, Pass: true},
		opsOracleCase{Name: "runbook accepts timeout reporting without outcome, elapsed, or caller wording", Text: validRunbook, Declared: true, Pass: true},
		opsOracleCase{Name: "runbook rejects timeout without a reporting action", Text: strings.Replace(validRunbook, "If a renewal times out, report the operation status to the service owner.", "If a renewal times out, check the operation.", 1), Declared: true},
		opsOracleCase{Name: "runbook rejects timeout reported as success", Text: validRunbook + " If a renewal times out, report it as successful.", Declared: true},
		opsOracleCase{Name: "runbook accepts explicit not-success timeout status", Text: validRunbook + " On timeout, report outcome unknown, not success.", Declared: true, Pass: true},
		opsOracleCase{Name: "runbook accepts straight-apostrophe don't success negation", Text: validRunbook + " On timeout, don't report success until the outcome is confirmed.", Declared: true, Pass: true},
		opsOracleCase{Name: "runbook accepts curly-apostrophe don’t success negation", Text: validRunbook + " On timeout, don’t report success until the outcome is confirmed.", Declared: true, Pass: true},
		opsOracleCase{Name: "runbook accepts timed-out wording", Text: strings.Replace(validRunbook, "If a renewal times out, report the operation status to the service owner.", "If a renewal timed out, notify the operator and check the request ID.", 1), Declared: true, Pass: true},
		opsOracleCase{Name: "runbook accepts success only after confirmed outcome", Text: validRunbook + " On timeout, report the timeout and record success only after the final outcome is confirmed.", Declared: true, Pass: true},
		opsOracleCase{Name: "runbook rejects absent request-ID retention statement", Text: strings.Replace(validRunbook, "Keep the request ID with the operation record for the full retry and reconciliation period, and retain it until the final outcome is known.", "", 1), Declared: true},
	)
	cases.Gates = append(cases.Gates,
		opsOracleCase{Name: "active hook/CI gate commands", Hook: "#!/bin/sh\nset -eu\ngo vet ./...\ngo test ./...\n", CI: "jobs:\n  quality:\n    steps:\n      - run: go vet ./...\n      - run: go test ./...\n", Commands: []string{"go vet ./...", "go test ./..."}, Pass: true},
		opsOracleCase{Name: "comment decoys do not satisfy active gates", Hook: "#!/bin/sh\n# go vet ./...\n", CI: "jobs:\n  quality:\n    steps:\n      # - run: go vet ./...\n", Commands: []string{"go vet ./..."}},
		opsOracleCase{Name: "simple active shell chaining is recognized", Hook: "#!/bin/sh\nset -eu && go vet ./...; go test ./...\n", CI: "jobs:\n  quality:\n    steps:\n      - run: go vet ./...\n      - run: go test ./...\n", Commands: []string{"go vet ./...", "go test ./..."}, Pass: true},
		opsOracleCase{Name: "hook command after unconditional exit is not evidence", Hook: "#!/bin/sh\nexit 1\ngo vet ./...\n", CI: "jobs:\n  quality:\n    steps:\n      - run: go vet ./...\n", Commands: []string{"go vet ./..."}},
		opsOracleCase{Name: "command inside uncalled hook function is not evidence", Hook: "#!/bin/sh\ncheck_later() {\n  go vet ./...\n}\n", CI: "jobs:\n  quality:\n    steps:\n      - run: go vet ./...\n", Commands: []string{"go vet ./..."}},
		opsOracleCase{Name: "command inside same-line hook function is not evidence", Hook: "#!/bin/sh\ncheck_later() { go vet ./...; }\n", CI: "jobs:\n  quality:\n    steps:\n      - run: go vet ./...\n", Commands: []string{"go vet ./..."}},
		opsOracleCase{Name: "CI command in if-false step is not evidence", Hook: "#!/bin/sh\ngo vet ./...\n", CI: "jobs:\n  quality:\n    steps:\n      - run: go vet ./...\n        if: false\n", Commands: []string{"go vet ./..."}},
		opsOracleCase{Name: "CI command in continue-on-error step is not evidence", Hook: "#!/bin/sh\ngo vet ./...\n", CI: "jobs:\n  quality:\n    steps:\n      - run: go vet ./...\n        continue-on-error: true\n", Commands: []string{"go vet ./..."}},
	)

	manifest, err := json.Marshal(cases)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "cases.json")
	if err := os.WriteFile(manifestPath, manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	script := `
param([string]$EvaluatorPath, [string]$CasesPath)
$tokens = $null
$parseErrors = $null
$ast = [System.Management.Automation.Language.Parser]::ParseFile($EvaluatorPath, [ref]$tokens, [ref]$parseErrors)
if ($parseErrors.Count -gt 0) { throw "Evaluator parse failed: $($parseErrors[0].Message)" }
$wanted = @('Test-A7FastCheckAssertionContract', 'Get-ProjectChecks', 'Test-ArgvEquals', 'Test-ProjectCheckArgv', 'Test-ActiveHookCommand', 'Get-CIQualitySteps', 'Get-CINormalizedRun', 'Test-ActiveCIRunCommand', 'Test-ActiveCheckGates', 'Get-PipelineCheckPointer', 'Get-CIQualityRunAtPointer', 'Test-B7VerificationAndGateContract', 'Test-LeaseRenewalRunbookContract', 'Select-TaskScopes', 'Test-InScope')
foreach ($name in $wanted) {
    $function = $ast.Find({ param($node) $node -is [System.Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -eq $name }, $true)
    if (-not $function) { throw "Required predicate $name was not found" }
    Invoke-Expression $function.Extent.Text
}
$cases = Get-Content -LiteralPath $CasesPath -Raw | ConvertFrom-Json
foreach ($case in $cases.a) {
    $actual = [bool](Test-A7FastCheckAssertionContract $case.root)
    if ($actual -ne [bool]$case.pass) { throw "A: $($case.name) returned $actual, expected $($case.pass)" }
}
foreach ($case in $cases.b) {
    $actual = [bool](Test-B7VerificationAndGateContract $case.root)
    if ($actual -ne [bool]$case.pass) { throw "B: $($case.name) returned $actual, expected $($case.pass)" }
}
foreach ($case in $cases.runbooks) {
    $actual = [bool](Test-LeaseRenewalRunbookContract $case.text ([bool]$case.declared))
    if ($actual -ne [bool]$case.pass) { throw "Runbook: $($case.name) returned $actual, expected $($case.pass)" }
}
foreach ($case in $cases.scopes) {
    $selectorError = $null
    try {
        $actualScopes = @(Select-TaskScopes ([string]$case.task) ([string]$case.prior) ([string[]]$case.task_scopes) ([string[]]$case.p01_scopes) ([string[]]$case.p02_scopes))
    } catch { $selectorError = $_.Exception.Message; $actualScopes = @() }
    if ($case.reject) {
        if (-not $selectorError) { throw "Scope: $($case.name) was accepted, expected rejection" }
        continue
    }
    if ($selectorError) { throw "Scope: $($case.name) was rejected: $selectorError" }
    $actualSorted = @($actualScopes | Sort-Object -Unique)
    $expectedSorted = @($case.want | Sort-Object -Unique)
    if (($actualSorted -join '|') -cne ($expectedSorted -join '|')) { throw "Scope: $($case.name) selected [$($actualSorted -join ', ')], expected [$($expectedSorted -join ', ')]" }
    foreach ($pathCase in $case.paths) {
        $inScope = [bool](Test-InScope ([string]$pathCase.path) ([string[]]$actualScopes))
        if ($inScope -ne [bool]$pathCase.in_scope) { throw "Scope: $($case.name) path $($pathCase.path) returned $inScope, expected $($pathCase.in_scope)" }
    }
}
foreach ($case in $cases.gates) {
    $actual = [bool](Test-ActiveCheckGates $case.hook $case.ci ([string[]]$case.commands))
    if ($actual -ne [bool]$case.pass) { throw "Gates: $($case.name) returned $actual, expected $($case.pass)" }
}
Write-Output "Passed $($cases.a.Count + $cases.b.Count + $cases.runbooks.Count + $cases.gates.Count + $cases.scopes.Count) Operations oracle v2 contract cases."
`
	scriptPath := filepath.Join(root, "ops-oracle-v2-contract.ps1")
	if err := os.WriteFile(scriptPath, []byte(strings.TrimSpace(script)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(pwsh, "-NoLogo", "-NoProfile", "-NonInteractive", "-File", scriptPath, oraclePath, manifestPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Operations oracle v2 contract failed: %v\n%s", err, output)
	}
	t.Log(strings.TrimSpace(string(output)))
}

func readOpsTaskScopes(t *testing.T, path, taskID, arm string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(data), "\n")
	idPattern := regexp.MustCompile(`^\s*- id:\s*["']?` + regexp.QuoteMeta(taskID) + `["']?\s*$`)
	cardStart := -1
	for i, line := range lines {
		if idPattern.MatchString(strings.TrimSuffix(line, "\r")) {
			cardStart = i
			break
		}
	}
	if cardStart < 0 {
		t.Fatalf("task %s not found in %s", taskID, path)
	}
	cardEnd := len(lines)
	nextCardPattern := regexp.MustCompile(`^\s*- id:`)
	for i := cardStart + 1; i < len(lines); i++ {
		if nextCardPattern.MatchString(strings.TrimSuffix(lines[i], "\r")) {
			cardEnd = i
			break
		}
	}
	body := strings.Join(lines[cardStart+1:cardEnd], "\n")
	armPattern := regexp.MustCompile(`(?m)^\s{6}` + regexp.QuoteMeta(arm) + `:\s*\[([^\]]*)\]`)
	match := armPattern.FindStringSubmatch(body)
	if len(match) != 2 {
		t.Fatalf("task %s arm %s scope not found in %s", taskID, arm, path)
	}
	items := strings.Split(match[1], ",")
	scopes := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		value = strings.Trim(value, `"'`)
		if value != "" {
			scopes = append(scopes, value)
		}
	}
	return scopes
}

func opsScopeUnion(left, right []string) []string {
	set := make(map[string]struct{}, len(left)+len(right))
	for _, scope := range append(append([]string(nil), left...), right...) {
		set[scope] = struct{}{}
	}
	union := make([]string, 0, len(set))
	for scope := range set {
		union = append(union, scope)
	}
	sort.Strings(union)
	return union
}

func writeOpsAFixture(t *testing.T, root, vetLabel, vetCommand string, expectedNames []string) {
	t.Helper()
	checkerDir := filepath.Join(root, "scripts")
	if err := os.MkdirAll(checkerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	original := []struct{ name, command string }{
		{"Operations policy", "python scripts/check_operations.py"},
		{"Operations checker tests", "python -m unittest discover -s scripts"},
		{"Root module tests", "go test ./..."},
		{"Nested module tests", "go -C tools/process-sentinel test ./..."},
	}
	var checker strings.Builder
	checker.WriteString("EXPECTED_RUNS = {\n")
	for _, item := range original {
		fmt.Fprintf(&checker, "    %s: %s,\n", strconv.Quote(item.name), strconv.Quote(item.command))
	}
	fmt.Fprintf(&checker, "    %s: %s,\n}\n", strconv.Quote(vetLabel), strconv.Quote(vetCommand))
	if err := os.WriteFile(filepath.Join(checkerDir, "check_operations.py"), []byte(checker.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	var test strings.Builder
	test.WriteString("import unittest\nfrom check_operations import EXPECTED_RUNS\n\nclass Ops(unittest.TestCase):\n    def test_hook_and_pipeline_cover_each_declared_fast_check(self):\n        self.assertEqual(set(EXPECTED_RUNS), {\n")
	for _, name := range expectedNames {
		fmt.Fprintf(&test, "            %s,\n", strconv.Quote(name))
	}
	test.WriteString("        })\n")
	if err := os.WriteFile(filepath.Join(checkerDir, "test_check_operations.py"), []byte(test.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeOpsBFixture(t *testing.T, root string) {
	t.Helper()
	files := map[string]string{
		".markitect/areas/operations/verification.rule.yaml": `apiVersion: markitect.example.org/v1alpha1
kind: Rule
metadata:
    name: verification
    namespace: operations
spec:
    text: |
        CI and pre-commit both run go vet, root Go tests, and nested-module Go tests. Run the configured Markitect artifact and hook/pipeline checks.
    files:
        - .githooks/pre-commit
        - .github/workflows/ci.yaml
        - .markitect/modules/githooks.config
        - .markitect/modules/pipelines.config
`,
		".markitect/areas/operations/engineering-operations.skill.yaml": `apiVersion: markitect.example.org/v1alpha1
kind: Skill
spec:
    rules:
        - name: verification
`,
		".agents/skills/engineering-operations/SKILL.md": "Read [Rule](../../../.markitect/areas/operations/verification.rule.yaml).\n",
		".claude/skills/engineering-operations/SKILL.md": "Read [Rule](../../../.markitect/areas/operations/verification.rule.yaml).\n",
		"markitect.yaml": `checks:
        - name: managed-artifacts
          run:
              - markitect-check-artifacts
              - --repo
              - .
              - --config
              - markitect-artifacts.yaml
        - name: hook-and-pipeline-contracts
          run:
              - markitect-check-modules
              - --repo
              - .
              - --hooks
              - .markitect/modules/githooks.config
              - --pipelines
              - .markitect/modules/pipelines.config
        - name: root-tests
          run:
              - go
              - test
              - ./...
        - name: root-vet
          run:
              - go
              - vet
              - ./...
        - name: nested-module-tests
          run:
              - go
              - -C
              - tools/process-sentinel
              - test
              - ./...
`,
		".markitect/modules/pipelines.config": `pipelines:
  - name: github-ci
    path: .github/workflows/ci.yaml
    owner: operations/Rule/verification
    expectedChecks:
      - name: root-vet
        yamlPath: /jobs/quality/steps/3/run
      - name: root-tests
        yamlPath: /jobs/quality/steps/4/run
      - name: nested-module-tests
        yamlPath: /jobs/quality/steps/5/run
`,
		".markitect/modules/githooks.config": `hooks:
  - name: local-verification
    path: .githooks/pre-commit
    owner: operations/Rule/verification
`,
		".githooks/pre-commit":      "#!/bin/sh\nset -eu\nmarkitect check --repo .\nmarkitect-check-modules --repo . --hooks .markitect/modules/githooks.config --pipelines .markitect/modules/pipelines.config\nmarkitect-check-artifacts --repo . --config markitect-artifacts.yaml\ngo vet ./...\ngo test ./...\ngo -C tools/process-sentinel test ./...\n",
		".github/workflows/ci.yaml": "name: CI\njobs:\n  quality:\n    steps:\n      - uses: actions/checkout@v4\n      - uses: actions/setup-go@v5\n        with:\n          go-version: '1.23'\n      - name: Verify Markitect and declared checks\n        run: markitect verify --repo . --revision $GITHUB_SHA\n      - name: Root module vet\n        run: go vet ./...\n      - name: Root module tests\n        run: go test ./...\n      - name: Nested module tests\n        run: go -C tools/process-sentinel test ./...\n",
	}
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func mutateOpsFile(t *testing.T, root, relativePath string, mutate func(string) string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(mutate(string(data))), 0o600); err != nil {
		t.Fatal(err)
	}
}
