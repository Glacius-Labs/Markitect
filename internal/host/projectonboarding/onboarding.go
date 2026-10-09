// Package projectonboarding prepares project-local instructions for contributors
// using Codex and/or Claude. It never changes global agent configuration.
package projectonboarding

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	hostwrite "github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
)

const APIVersion = "markitect.example.org/project-onboarding/v1alpha1"

const workflowPath = ".markitect/workflows/model-first.md"

type Provider string

const (
	Codex  Provider = "codex"
	Claude Provider = "claude"
)

// Options selects native contributor surfaces and the generated project
// documentation destination. Providers must be codex, claude, or both.
type Options struct {
	Providers         []Provider `json:"providers"`
	DocumentationPath string     `json:"documentationPath"`
}

type FileChange struct {
	Path    string `json:"path"`
	Action  string `json:"action"` // create, update, delete, or unchanged
	Content string `json:"content"`
}

type TargetBasis struct {
	Path        string `json:"path"`
	Exists      bool   `json:"exists"`
	ContentHash string `json:"contentHash,omitempty"`
	Mode        string `json:"mode,omitempty"`
}

// Plan is a digest-bound preview. Apply recomputes it from the current
// project, Git state, and target files before using the shared Host writer.
type Plan struct {
	APIVersion     string        `json:"apiVersion"`
	RepositoryRoot string        `json:"repositoryRoot"`
	ModelDigest    string        `json:"modelDigest"`
	ProjectDigest  string        `json:"projectDigest"`
	Branch         string        `json:"branch"`
	Head           string        `json:"head"`
	Options        Options       `json:"options"`
	Targets        []TargetBasis `json:"targets"`
	Files          []FileChange  `json:"files"`
	Digest         string        `json:"digest"`
	Written        []string      `json:"written,omitempty"`
}

// Preview returns the exact project-local onboarding files without writing.
// The project digest binds selected model and configuration bytes.
func Preview(root, modelDigest string, options Options) (Plan, error) {
	var plan Plan
	providers, err := normalizeProviders(options.Providers)
	if err != nil {
		return plan, err
	}
	options.Providers = providers
	project, err := projectwork.Load(root, "")
	if err != nil {
		return plan, fmt.Errorf("load current project: %w", err)
	}
	canonicalDocumentationPath := projectwork.DocumentPath(project.Config)
	if options.DocumentationPath == "" {
		options.DocumentationPath = canonicalDocumentationPath
	}
	if options.DocumentationPath != canonicalDocumentationPath {
		return plan, fmt.Errorf("documentation destination %q differs from the canonical project destination %q; update the project configuration first", options.DocumentationPath, canonicalDocumentationPath)
	}
	if err := validateDocumentationPath(options.DocumentationPath); err != nil {
		return plan, err
	}
	if modelDigest == "" || project.Report.ModelDigest != modelDigest {
		return plan, errors.New("onboarding model digest does not match the current project model")
	}
	files, err := renderFiles(options)
	if err != nil {
		return plan, err
	}
	for _, file := range files {
		if strings.EqualFold(file.Path, options.DocumentationPath) {
			return plan, fmt.Errorf("documentation destination %q conflicts with a native onboarding output", options.DocumentationPath)
		}
	}
	legacyPaths := staleSkillPaths(providers)
	for _, legacyPath := range legacyPaths {
		if strings.EqualFold(legacyPath, options.DocumentationPath) {
			return plan, fmt.Errorf("documentation destination %q conflicts with a legacy skill migration target", options.DocumentationPath)
		}
	}
	paths := make([]string, 0, len(files)+len(providers))
	for i := range files {
		paths = append(paths, files[i].Path)
	}
	paths = append(paths, legacyPaths...)
	capture, err := hostwrite.CaptureGuardedWrite(root, paths)
	if err != nil {
		return plan, fmt.Errorf("capture onboarding targets: %w", err)
	}
	plan = Plan{
		APIVersion: APIVersion, RepositoryRoot: capture.Identity.Root, ModelDigest: project.Report.ModelDigest,
		ProjectDigest: project.Digest, Branch: capture.Branch,
		Head: capture.Head, Options: options, Targets: make([]TargetBasis, 0, len(paths)),
		Files: files,
	}
	for _, file := range files {
		observed := capture.Files[file.Path]
		basis := TargetBasis{Path: file.Path, Exists: observed.Exists}
		if observed.Exists {
			basis.ContentHash = contentHash(observed.Bytes)
			basis.Mode = fmt.Sprintf("%04o", observed.Mode.Perm())
			merged, mergeErr := mergeFile(file.Path, string(observed.Bytes), file.Content)
			if mergeErr != nil {
				return Plan{}, fmt.Errorf("prepare %s: %w", file.Path, mergeErr)
			}
			file.Content = merged
		} else {
			file.Content += "\n"
		}
		plan.Files[indexFile(plan.Files, file.Path)].Content = file.Content
		plan.Targets = append(plan.Targets, basis)
		if observed.Exists && string(observed.Bytes) == file.Content {
			plan.Files[indexFile(plan.Files, file.Path)].Action = "unchanged"
		} else if observed.Exists {
			plan.Files[indexFile(plan.Files, file.Path)].Action = "update"
		} else {
			plan.Files[indexFile(plan.Files, file.Path)].Action = "create"
		}
	}
	for _, legacyPath := range legacyPaths {
		observed := capture.Files[legacyPath]
		basis := TargetBasis{Path: legacyPath, Exists: observed.Exists}
		if observed.Exists {
			basis.ContentHash = contentHash(observed.Bytes)
			basis.Mode = fmt.Sprintf("%04o", observed.Mode.Perm())
			if !isPreviouslyGeneratedRouter(observed.Bytes) {
				return Plan{}, fmt.Errorf("legacy skill path %s conflicts with custom or mixed content; preserving it", legacyPath)
			}
			plan.Files = append(plan.Files, FileChange{Path: legacyPath, Action: "delete"})
		}
		plan.Targets = append(plan.Targets, basis)
	}
	sort.Slice(plan.Files, func(i, j int) bool { return plan.Files[i].Path < plan.Files[j].Path })
	sort.Slice(plan.Targets, func(i, j int) bool { return plan.Targets[i].Path < plan.Targets[j].Path })
	plan.Digest, err = digestPlan(plan)
	if err != nil {
		return Plan{}, err
	}
	return plan, nil
}

// Apply writes the exact preview only when expectDigest matches. It reloads
// and re-renders under the Host writer lock, binding model/config, branch,
// HEAD, and selected target bytes to the preview.
func Apply(root string, plan Plan, expectDigest string) (Plan, error) {
	if expectDigest == "" || plan.Digest == "" || expectDigest != plan.Digest {
		return plan, errors.New("onboarding write requires the exact preview digest")
	}
	current, err := Preview(root, plan.ModelDigest, plan.Options)
	if err != nil {
		return plan, err
	}
	if current.Digest != expectDigest {
		return plan, errors.New("onboarding preview is stale; create a fresh preview")
	}
	paths := make([]string, len(current.Targets))
	for i, target := range current.Targets {
		paths[i] = target.Path
	}
	capture, err := hostwrite.CaptureGuardedWrite(root, paths)
	if err != nil {
		return plan, fmt.Errorf("recapture onboarding targets: %w", err)
	}
	if capture.Identity.Root != current.RepositoryRoot || capture.Branch != current.Branch || capture.Head != current.Head {
		return plan, errors.New("repository, branch, or HEAD changed after onboarding preview")
	}
	for _, expected := range current.Targets {
		actual, ok := capture.Files[expected.Path]
		if !ok || actual.Exists != expected.Exists || actual.Exists && (contentHash(actual.Bytes) != expected.ContentHash || fmt.Sprintf("%04o", actual.Mode.Perm()) != expected.Mode) {
			return plan, fmt.Errorf("onboarding target %s changed after preview", expected.Path)
		}
	}
	changes := make([]hostwrite.GuardedWriteChange, 0, len(current.Files))
	for _, file := range current.Files {
		if file.Action == "unchanged" {
			continue
		}
		if file.Action == "delete" {
			changes = append(changes, hostwrite.GuardedWriteChange{Path: file.Path, Delete: true})
			continue
		}
		mode := fs.FileMode(0644)
		if observed := capture.Files[file.Path]; observed.Exists {
			mode = observed.Mode.Perm()
		}
		changes = append(changes, hostwrite.GuardedWriteChange{Path: file.Path, Bytes: []byte(file.Content), Mode: mode})
	}
	validate := func() error {
		fresh, loadErr := projectwork.Load(root, "")
		if loadErr != nil {
			return loadErr
		}
		if fresh.Digest != current.ProjectDigest || fresh.Report.ModelDigest != current.ModelDigest {
			return errors.New("project model or configuration changed after onboarding preview")
		}
		return nil
	}
	result, err := hostwrite.ApplyGuardedWriteChecked(capture.Root, capture, changes, validate)
	current.Written = append([]string(nil), result.CompletedPaths...)
	if err != nil {
		return current, fmt.Errorf("onboarding write was partial after %s: %w", strings.Join(current.Written, ", "), err)
	}
	return current, nil
}

func normalizeProviders(providers []Provider) ([]Provider, error) {
	if len(providers) == 0 {
		return nil, errors.New("select codex, claude, or both")
	}
	seen := map[Provider]bool{}
	for _, provider := range providers {
		if provider != Codex && provider != Claude {
			return nil, fmt.Errorf("unsupported onboarding provider %q; choose codex or claude", provider)
		}
		if seen[provider] {
			return nil, fmt.Errorf("onboarding provider %q is repeated", provider)
		}
		seen[provider] = true
	}
	result := append([]Provider(nil), providers...)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

func validateDocumentationPath(value string) error {
	if value == "" || strings.ContainsAny(value, "\\\x00") || path.IsAbs(value) || path.Clean(value) != value || value == "." || strings.HasPrefix(value, "../") || !strings.HasSuffix(strings.ToLower(value), ".md") {
		return fmt.Errorf("documentation destination %q must be a normalized repository-relative Markdown path", value)
	}
	legacyView := value == projectwork.ViewPath || strings.HasPrefix(strings.ToLower(value), ".markitect/views/")
	if strings.EqualFold(value, ".markitect") || strings.HasPrefix(strings.ToLower(value), ".markitect/") && !legacyView {
		return errors.New("documentation destination must be outside .markitect/")
	}
	return nil
}

func contentHash(data []byte) string {
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func digestPlan(plan Plan) (string, error) {
	plan.Digest = ""
	plan.Written = nil
	data, err := json.Marshal(plan)
	if err != nil {
		return "", fmt.Errorf("encode onboarding plan: %w", err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func indexFile(files []FileChange, target string) int {
	for i := range files {
		if files[i].Path == target {
			return i
		}
	}
	panic("onboarding target missing from rendered files")
}

// staleSkillPaths selects only the two exact files emitted by the former
// compatibility router. They remain guarded migration targets, not generated
// outputs or registered native skill paths.
func staleSkillPaths(providers []Provider) []string {
	paths := make([]string, 0, len(providers))
	for _, provider := range providers {
		switch provider {
		case Codex:
			paths = append(paths, ".agents/skills/markitect-model-first/SKILL.md")
		case Claude:
			paths = append(paths, ".claude/skills/markitect-model-first/SKILL.md")
		}
	}
	return paths
}

// isPreviouslyGeneratedRouter permits deletion only when the entire file is
// the exact old Markitect output. Normalizing CRLF allows Windows checkout
// translation while still rejecting any extra user bytes or metadata.
func isPreviouslyGeneratedRouter(data []byte) bool {
	normalized := strings.ReplaceAll(string(data), "\r\n", "\n")
	return normalized == previouslyGeneratedRouter()
}

func previouslyGeneratedRouter() string {
	const description = "Compatibility router for Markitect project work; select the matching init, extract, design, suggest, configure, implement, cleanup, verify, apply, or check skill without loading the full workflow."
	const body = "Choose the operation skill that matches the work. Ordinary requests to implement a Work Item start with `markitect-implement`, which chains required operations autonomously. Use `markitect-init` for setup, `markitect-extract` for Brownfield adoption, `markitect-design` for intent changes, `markitect-suggest` for proposal-only recommendations, `markitect-configure` for existing project/runtime settings, `markitect-cleanup` for behavior-preserving refactoring, `markitect-verify` for realization assessment, `markitect-apply` for an already verified candidate, and `markitect-check` for structural diagnostics."
	return fmt.Sprintf("---\nname: markitect-model-first\ndescription: %q\n---\n\n%s\n", description, managedBlock(body))
}
