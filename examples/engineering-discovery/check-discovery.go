package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/app"
	"go.yaml.in/yaml/v3"
)

const (
	evidenceVersion = "markitect.example.org/engineering-discovery-evidence/v1alpha1"
	decisionVersion = "markitect.example.org/engineering-discovery-decision/v1alpha1"
	maxDossierBytes = 1 << 20
)

type evidenceManifest struct {
	Version        string           `yaml:"version"`
	Repository     string           `yaml:"repository"`
	Revision       string           `yaml:"revision"`
	SnapshotDigest string           `yaml:"snapshotDigest"`
	Entry          string           `yaml:"entry"`
	ToolVersion    string           `yaml:"toolVersion"`
	ToolDigest     string           `yaml:"toolDigest"`
	Run            runEvidence      `yaml:"run"`
	Sources        []evidenceSource `yaml:"sources"`
}

type runEvidence struct {
	Path          string `yaml:"path"`
	ManifestHash  string `yaml:"manifestHash"`
	SelectionHash string `yaml:"selectionHash"`
	ContextDigest string `yaml:"contextDigest"`
}

type evidenceSource struct {
	ID      string `yaml:"id"`
	Stance  string `yaml:"stance"`
	Path    string `yaml:"path"`
	Hash    string `yaml:"hash"`
	Excerpt string `yaml:"excerpt"`
	Note    string `yaml:"note"`
}

type decisionRecord struct {
	Version       string `yaml:"version"`
	Status        string `yaml:"status"`
	Reviewer      string `yaml:"reviewer"`
	DecidedAt     string `yaml:"decidedAt"`
	Rationale     string `yaml:"rationale"`
	CandidateHash string `yaml:"candidateHash"`
	EvidenceHash  string `yaml:"evidenceHash"`
}

var evidenceIDPattern = regexp.MustCompile(`^\s*-\s+(E-[A-Z0-9-]+)\b`)
var fullCommitPattern = regexp.MustCompile(`^(?:[0-9a-fA-F]{40}|[0-9a-fA-F]{64})$`)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "discovery check:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("check-discovery", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repo := flags.String("repo", "", "copied Project repository root")
	revision := flags.String("revision", "", "full immutable Git commit ID")
	runPath := flags.String("run", "", "ContextRun manifest path in the selected snapshot")
	evidencePath := flags.String("evidence", "", "external evidence ledger YAML")
	candidatePath := flags.String("candidate", "", "external candidate Markdown")
	decisionPath := flags.String("decision", "", "external human decision YAML")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *repo == "" || *revision == "" || *runPath == "" || *evidencePath == "" || *candidatePath == "" || *decisionPath == "" {
		return errors.New("require --repo, --revision, --run, --evidence, --candidate and --decision")
	}
	if !fullCommitPattern.MatchString(*revision) {
		return errors.New("--revision must be a full immutable Git commit ID")
	}

	projectRoot, err := filepath.Abs(*repo)
	if err != nil {
		return fmt.Errorf("resolve Project root: %w", err)
	}
	for _, external := range []string{*evidencePath, *candidatePath, *decisionPath} {
		if err := requireOutsideProject(projectRoot, external); err != nil {
			return err
		}
	}

	project, err := app.Load(projectRoot, *revision)
	if err != nil {
		return fmt.Errorf("load fixed Project snapshot: %w", err)
	}
	if len(project.Diagnostics) != 0 {
		return fmt.Errorf("fixed Project has %d graph diagnostics", len(project.Diagnostics))
	}
	evidenceBytes, err := readBounded(*evidencePath)
	if err != nil {
		return err
	}
	candidateBytes, err := readBounded(*candidatePath)
	if err != nil {
		return err
	}
	decisionBytes, err := readBounded(*decisionPath)
	if err != nil {
		return err
	}
	var evidence evidenceManifest
	if err := decodeStrict(evidenceBytes, &evidence); err != nil {
		return fmt.Errorf("parse evidence ledger: %w", err)
	}
	var decision decisionRecord
	if err := decodeStrict(decisionBytes, &decision); err != nil {
		return fmt.Errorf("parse decision record: %w", err)
	}
	manifestBytes, ok := project.Snapshot.Files[*runPath]
	if !ok {
		return fmt.Errorf("ContextRun manifest %q is absent from the selected snapshot", *runPath)
	}
	if strings.TrimSpace(evidence.ToolVersion) == "" || strings.TrimSpace(evidence.ToolDigest) == "" {
		return errors.New("evidence ledger must bind the ContextRun tool version and digest")
	}
	context, err := app.CompileRunContext(project, *runPath, manifestBytes, evidence.ToolVersion, evidence.ToolDigest)
	if err != nil {
		return fmt.Errorf("compile fixed ContextRun: %w", err)
	}
	if !context.Complete || context.Run == nil {
		return errors.New("ContextRun is incomplete")
	}

	if evidence.Version != evidenceVersion || strings.TrimSpace(evidence.Repository) == "" {
		return errors.New("evidence ledger version and stable repository ID are required")
	}
	if evidence.Revision != project.Snapshot.ID || evidence.SnapshotDigest != project.Snapshot.Digest() {
		return errors.New("evidence ledger does not match the selected immutable revision and snapshot digest")
	}
	if evidence.Entry != context.Entry || evidence.Run.Path != context.Run.ManifestPath ||
		evidence.Run.ManifestHash != context.Run.ManifestHash || evidence.Run.SelectionHash != context.Run.SelectionHash ||
		evidence.Run.ContextDigest != context.Digest {
		return errors.New("evidence ledger does not match the recomputed ContextRun identity")
	}
	if len(evidence.Sources) == 0 {
		return errors.New("evidence ledger must include selected sources")
	}
	known := make(map[string]evidenceSource, len(evidence.Sources))
	selected := make(map[string]app.ContextInput, len(context.Inputs))
	for _, input := range context.Inputs {
		if input.Role == "source" {
			selected[input.Path] = input
		}
	}
	seenPaths := map[string]bool{}
	for _, source := range evidence.Sources {
		if source.ID == "" || source.Note == "" || source.Excerpt == "" || seenPaths[source.Path] {
			return fmt.Errorf("evidence source %q needs a unique path, ID, exact excerpt and note", source.ID)
		}
		switch source.Stance {
		case "supports", "counterexample", "qualifies":
		default:
			return fmt.Errorf("evidence %s has unsupported stance %q", source.ID, source.Stance)
		}
		if _, exists := known[source.ID]; exists {
			return fmt.Errorf("duplicate evidence ID %s", source.ID)
		}
		input, exists := selected[source.Path]
		if !exists || input.Status != "included" || input.Hash != source.Hash || !strings.Contains(input.Text, source.Excerpt) {
			return fmt.Errorf("evidence %s is not the exact included ContextRun input at %q", source.ID, source.Path)
		}
		seenPaths[source.Path] = true
		known[source.ID] = source
	}
	if len(known) != len(selected) {
		return errors.New("evidence ledger must account for every selected ContextRun source")
	}

	refs, err := candidateEvidenceRefs(string(candidateBytes))
	if err != nil {
		return err
	}
	if len(refs) != len(known) {
		return errors.New("candidate must cite every evidence item exactly once")
	}
	support, counterexample := false, false
	for id, section := range refs {
		source, exists := known[id]
		if !exists {
			return fmt.Errorf("candidate cites unknown evidence ID %s", id)
		}
		want := map[string]string{"supports": "supporting evidence", "counterexample": "counterexamples", "qualifies": "qualifying evidence"}[source.Stance]
		if section != want {
			return fmt.Errorf("candidate cites %s under %q; evidence stance requires %q", id, section, want)
		}
		support = support || source.Stance == "supports"
		counterexample = counterexample || source.Stance == "counterexample"
	}
	if !support || !counterexample {
		return errors.New("candidate must show at least one supporting item and one counterexample")
	}
	if decision.Version != decisionVersion || strings.TrimSpace(decision.Reviewer) == "" ||
		strings.TrimSpace(decision.DecidedAt) == "" || strings.TrimSpace(decision.Rationale) == "" {
		return errors.New("decision version, reviewer, date and rationale are required")
	}
	switch decision.Status {
	case "accepted", "rejected", "deferred", "split", "revise":
	default:
		return fmt.Errorf("unsupported human decision status %q", decision.Status)
	}
	if decision.CandidateHash != app.Hash(candidateBytes) || decision.EvidenceHash != app.Hash(evidenceBytes) {
		return errors.New("decision is stale: candidate or evidence-ledger bytes changed after review")
	}

	fmt.Printf("Integrity checks passed for %d selected evidence items at %s (%s).\n", len(known), project.Snapshot.ID, project.Snapshot.Digest())
	fmt.Printf("Source identity label: %s (caller-supplied).\n", evidence.Repository)
	fmt.Printf("Recorded decision: %s by %s. Reviewer identity is not authenticated; no canonical resource or policy was adopted.\n", decision.Status, decision.Reviewer)
	return nil
}

func requireOutsideProject(projectRoot, externalPath string) error {
	absolute, err := filepath.Abs(externalPath)
	if err != nil {
		return fmt.Errorf("resolve dossier path: %w", err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		return fmt.Errorf("resolve Project root aliases: %w", err)
	}
	resolvedExternal, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return fmt.Errorf("resolve dossier path aliases: %w", err)
	}
	if !strings.EqualFold(filepath.VolumeName(resolvedRoot), filepath.VolumeName(resolvedExternal)) {
		return nil
	}
	relative, err := filepath.Rel(resolvedRoot, resolvedExternal)
	if err != nil {
		return fmt.Errorf("compare dossier and Project paths: %w", err)
	}
	if relative == "." || (!filepath.IsAbs(relative) && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
		return fmt.Errorf("candidate, evidence ledger and decision must remain outside the Project snapshot and its Areas")
	}
	return nil
}

func readBounded(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("inspect %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular file", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("inspect opened %s: %w", path, err)
	}
	if !openedInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular file", path)
	}
	data, err := io.ReadAll(io.LimitReader(file, maxDossierBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) == 0 || len(data) > maxDossierBytes {
		return nil, fmt.Errorf("%s must be between 1 byte and 1 MiB", path)
	}
	if !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
		return nil, fmt.Errorf("%s must be valid UTF-8 text without NUL", path)
	}
	return data, nil
}

func decodeStrict(data []byte, target any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return errors.New("expected exactly one YAML document")
	} else if err != io.EOF {
		return err
	}
	return nil
}

func candidateEvidenceRefs(candidate string) (map[string]string, error) {
	sections := map[string]string{
		"## Supporting evidence": "supporting evidence",
		"## Counterexamples":     "counterexamples",
		"## Qualifying evidence": "qualifying evidence",
	}
	refs := map[string]string{}
	section := ""
	for _, line := range strings.Split(candidate, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			section = sections[trimmed]
			continue
		}
		match := evidenceIDPattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		if section == "" {
			return nil, fmt.Errorf("evidence ID %s appears outside a recognized evidence section", match[1])
		}
		if _, duplicate := refs[match[1]]; duplicate {
			return nil, fmt.Errorf("candidate repeats evidence ID %s", match[1])
		}
		refs[match[1]] = section
	}
	return refs, nil
}
