// Command replay runs a bounded public Markitect-report replay. It selects one
// fixed revision and one exact path, then stores a temporary external handoff
// and validates a synthetic interpretation record over those captured bytes.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/examples/selective-adoption/pathspell"
	"go.yaml.in/yaml/v3"
)

const (
	publicMarkitectCommit = "93181bb9bc1af0e663b3daa1a4ff772822320307"
	publicReportPath      = "docs/validation/parallel-wave-konfyra.md"
)

type preview struct {
	Status  string `yaml:"status"`
	Handoff struct {
		ID           string `yaml:"id"`
		Digest       string `yaml:"digest"`
		Repositories []struct {
			ID             string `yaml:"id"`
			Commit         string `yaml:"commit"`
			SnapshotDigest string `yaml:"snapshotDigest"`
			Files          []struct {
				Path   string `yaml:"path"`
				Digest string `yaml:"digest"`
			} `yaml:"files"`
		} `yaml:"repositories"`
	} `yaml:"handoff"`
}

type storedHandoff struct {
	ID       string `yaml:"id"`
	Digest   string `yaml:"digest"`
	Coverage []any  `yaml:"coverage"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("selective-adoption replay", flag.ContinueOnError)
	markitect := flags.String("markitect", "", "path to a built Markitect executable")
	publicRepo := flags.String("public-repo", "", "local Markitect repository containing the fixed public report commit")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *markitect == "" || *publicRepo == "" {
		return errors.New("usage: replay --markitect ABS_BINARY --public-repo ABS_MARKITECT_ROOT")
	}
	repo, err := pathspell.Canonical(*publicRepo)
	if err != nil {
		return err
	}
	parent, err := os.MkdirTemp("", "markitect-selective-replay-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(parent)
	parent, err = pathspell.Canonical(parent)
	if err != nil {
		return fmt.Errorf("canonicalize temporary replay path: %w", err)
	}
	scope := map[string]any{
		"apiVersion": "markitect.example.org/adoption-scope/v1alpha1",
		"id":         "public-wave-report-replay",
		"purpose":    "Review one public validation report without treating it as repository-wide evidence.",
		"review":     "public-report-replay",
		"privacy":    map[string]any{"constraints": "Only the named public report may be captured; no excerpts are reproduced.", "allowExcerpts": false},
		"retention":  "Temporary local handoff, removed when this replay exits.",
		"repositories": []any{map[string]any{
			"id": "markitect", "root": repo, "commit": publicMarkitectCommit,
			"paths": []any{map[string]any{"path": publicReportPath, "reason": "The one public sanitized report selected for this bounded replay."}},
		}},
		"coverage": []any{map[string]any{"id": "report-scope", "repository": "markitect", "question": "What scope does one selected public report support?", "state": "examined", "reason": "Only the exact report path and pinned revision are in scope."}},
	}
	scopeBytes, err := yaml.Marshal(scope)
	if err != nil {
		return err
	}
	scopePath := filepath.Join(parent, "scope.yaml")
	if err := os.WriteFile(scopePath, scopeBytes, 0600); err != nil {
		return err
	}
	workspace := filepath.Join(parent, "handoff")
	previewOutput, err := runCLI(*markitect, "prepare", "--scope", scopePath, "--output", workspace)
	if err != nil {
		return fmt.Errorf("read-only prepare preview failed: %w\n%s", err, previewOutput)
	}
	var plan preview
	if err := yaml.Unmarshal(previewOutput, &plan); err != nil {
		return fmt.Errorf("decode prepare preview: %w", err)
	}
	if plan.Status != "planned" || !validDigest(plan.Handoff.Digest) || len(plan.Handoff.Repositories) != 1 || plan.Handoff.Repositories[0].ID != "markitect" || plan.Handoff.Repositories[0].Commit != publicMarkitectCommit || len(plan.Handoff.Repositories[0].Files) != 1 || plan.Handoff.Repositories[0].Files[0].Path != publicReportPath {
		return errors.New("preview did not preserve the exact one-file public selection")
	}
	if _, statErr := os.Lstat(workspace); !os.IsNotExist(statErr) {
		return fmt.Errorf("preview unexpectedly created external destination %s", workspace)
	}
	writeOutput, err := runCLI(*markitect, "prepare", "--scope", scopePath, "--output", workspace, "--write", "--expect", plan.Handoff.Digest)
	if err != nil {
		return fmt.Errorf("expected-digest handoff write failed: %w\n%s", err, writeOutput)
	}
	handoffBytes, err := os.ReadFile(filepath.Join(workspace, "handoff.yaml"))
	if err != nil {
		return err
	}
	var handoff storedHandoff
	if err := yaml.Unmarshal(handoffBytes, &handoff); err != nil {
		return fmt.Errorf("decode stored handoff: %w", err)
	}
	if handoff.ID != plan.Handoff.ID || handoff.Digest != plan.Handoff.Digest || len(handoff.Coverage) != 1 {
		return errors.New("stored handoff differs from the selected preview")
	}
	file := plan.Handoff.Repositories[0].Files[0]
	// This is an explicitly synthetic interpretation record. It makes no claim
	// about the report's substantive findings; it demonstrates bounded byte
	// binding, uncertainty retention, and non-adoption on this public input.
	candidate := map[string]any{
		"apiVersion": "markitect.example.org/copy-me-candidate/v1alpha1", "stableID": "single-report-scope",
		"proposedRule":   "A single selected report supports claims only within its declared review scope.",
		"scope":          "Transport and selection boundaries of this one report; substantive claims remain unreviewed.",
		"conditions":     []string{"The report remains pinned to the exact selected commit and path."},
		"classification": "unclear", "support": []string{"report-artifact"}, "counterexamples": []string{},
		"qualifies": []string{"scope-limitation"}, "confidence": "low",
		"confidenceBasis": "One public report was selected; no other repository content was examined.",
		"alternatives":    []string{"Treat the report as a project-specific historical record only."},
		"uncertainty":     []string{"The report's truth, completeness and representativeness are outside this transport replay."},
		"questions":       []string{"What independent evidence would be needed for a broader claim?"},
	}
	candidateBytes, err := yaml.Marshal(candidate)
	if err != nil {
		return err
	}
	queue := map[string]any{
		"apiVersion": "markitect.example.org/copy-me-queue/v1alpha1",
		"evidence": []any{
			map[string]any{
				"id": "report-artifact", "repository": "markitect", "path": publicReportPath,
				"sourceDigest": file.Digest, "stance": "supports",
				"observation": "The exact public report artifact is captured at the pinned revision; this does not assess its content claims.",
			},
			map[string]any{
				"id": "scope-limitation", "repository": "markitect", "path": publicReportPath,
				"sourceDigest": file.Digest, "stance": "qualifies",
				"observation": "This one selected artifact does not establish repository-wide coverage or truth.",
			},
		},
		"candidates": []any{map[string]any{"stableID": "single-report-scope", "path": "candidates/scope.yaml", "digest": rawDigest(candidateBytes)}},
		"coverage":   handoff.Coverage,
		"requests":   []any{},
	}
	queueBytes, err := yaml.Marshal(queue)
	if err != nil {
		return err
	}
	decision := map[string]any{
		"apiVersion": "markitect.example.org/copy-me-decision/v1alpha1", "id": "bounded-replay-review",
		"reviewer": "synthetic-example", "date": "2026-10-04", "rationale": "The record demonstrates byte-bound review input only; no adoption is authorized.",
		"scope":  "Transport and selection boundaries of this one report; substantive claims remain unreviewed.",
		"status": "defer", "candidateID": "single-report-scope", "candidateDigest": rawDigest(candidateBytes),
		"queueDigest": rawDigest(queueBytes), "handoffDigest": rawDigest(handoffBytes), "handoffIdentity": handoff.Digest,
	}
	decisionBytes, err := yaml.Marshal(decision)
	if err != nil {
		return err
	}
	queuePath := filepath.Join(parent, "queue.yaml")
	candidateDir := filepath.Join(parent, "candidates")
	if err := os.Mkdir(candidateDir, 0700); err != nil {
		return err
	}
	candidatePath := filepath.Join(candidateDir, "scope.yaml")
	decisionPath := filepath.Join(parent, "decision.yaml")
	for path, data := range map[string][]byte{queuePath: queueBytes, candidatePath: candidateBytes, decisionPath: decisionBytes} {
		if err := os.WriteFile(path, data, 0600); err != nil {
			return err
		}
	}
	copyOutput, err := runCLI(*markitect, "copy-me", "--workspace", workspace, "--queue", queuePath, "--decision", decisionPath)
	if err != nil {
		return fmt.Errorf("Copy Me byte-reference validation failed: %w\n%s", err, copyOutput)
	}
	var report struct {
		HandoffIdentity         string `yaml:"handoffIdentity"`
		Candidates              []any  `yaml:"candidates"`
		Decision                *any   `yaml:"decision"`
		UnauthenticatedReviewer bool   `yaml:"unauthenticatedReviewer"`
		Adopted                 bool   `yaml:"adopted"`
	}
	if err := yaml.Unmarshal(copyOutput, &report); err != nil {
		return fmt.Errorf("decode Copy Me report: %w", err)
	}
	if report.HandoffIdentity != handoff.Digest || len(report.Candidates) != 1 || report.Decision == nil || !report.UnauthenticatedReviewer || report.Adopted {
		return errors.New("Copy Me report did not preserve bounded, unauthenticated, non-adopting status")
	}
	fmt.Printf("handoff: %s\ncommit: %s\nselectedPath: %s\ncopyMe: validated\nreviewer: unauthenticated\nadopted: false\nuncertainty: report claims and representativeness not assessed\n", handoff.Digest, publicMarkitectCommit, publicReportPath)
	return nil
}

func runCLI(binary string, args ...string) ([]byte, error) {
	command := exec.Command(binary, args...)
	return command.CombinedOutput()
}

func rawDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func validDigest(value string) bool {
	if len(value) != 64 || strings.TrimSpace(value) != value {
		return false
	}
	for _, c := range value {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
