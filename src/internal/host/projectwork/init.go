package projectwork

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/host/guardedwrite"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
	"go.yaml.in/yaml/v3"
)

const initManagerPath = ModelRoot + "/manager.yaml"

// Init creates a read-only exact plan by default. A write creates only the
// five named Markitect-owned files through Host's guarded writer.
func Init(root, name string, write bool) (InitPlan, error) {
	if strings.TrimSpace(name) == "" || name != strings.TrimSpace(name) {
		return InitPlan{}, fmt.Errorf("project name must be nonempty and have no surrounding whitespace")
	}
	config := map[string]any{
		"apiVersion":             APIVersion,
		"name":                   name,
		"coverageMode":           "full",
		"workflowMode":           WorkflowModeGuided,
		"acceptancePolicy":       AcceptancePolicyCommittedModel,
		"documentPath":           DefaultDocumentPath,
		"modelFiles":             []string{initManagerPath},
		"inventoryRoots":         []string{},
		"exclusions":             []any{},
		"transitionalExclusions": []any{},
	}
	configBytes, err := yaml.Marshal(config)
	if err != nil {
		return InitPlan{}, fmt.Errorf("encode project config: %w", err)
	}
	manager := map[string]any{
		"apiVersion": projectmodel.APIVersion,
		"kind":       "Manager",
		"metadata":   map[string]any{"name": "project-owner", "namespace": ""},
		"purpose":    "Owns the repository-wide engineering mandate and delegates bounded responsibility.",
		"spec":       map[string]any{"owns": []string{"."}},
	}
	managerBytes, err := yaml.Marshal(manager)
	if err != nil {
		return InitPlan{}, fmt.Errorf("encode root Manager: %w", err)
	}
	files := []FileChange{
		{Path: ManifestPath, Content: string(configBytes)},
		{Path: RuntimePath, Content: "{}\n"},
		{Path: initManagerPath, Content: string(managerBytes)},
		{Path: ".markitect/.gitignore", Content: "/cache/\n/runs/\n/state/\n/views/\n"},
	}
	previewSnapshot := &snapshot.Snapshot{Provisional: true, Files: map[string][]byte{}, Modes: map[string]string{}}
	for _, file := range files {
		previewSnapshot.Files[file.Path] = []byte(file.Content)
		previewSnapshot.Modes[file.Path] = snapshot.RegularMode
	}
	project, err := FromSnapshot(root, previewSnapshot)
	if err != nil {
		return InitPlan{}, fmt.Errorf("validate initial project model: %w", err)
	}
	files = append(files, FileChange{Path: DocumentPath(project.Config), Content: documentText(project)})
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	plan := InitPlan{APIVersion: APIVersion, Name: name, Files: files}
	plan.Digest, err = initPlanDigest(plan)
	if err != nil {
		return InitPlan{}, err
	}
	if !write {
		return plan, nil
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return plan, fmt.Errorf("resolve project root: %w", err)
	}
	paths := make([]string, len(files))
	changes := make([]guardedwrite.Change, len(files))
	for i, file := range files {
		paths[i] = file.Path
		changes[i] = guardedwrite.Change{Path: file.Path, Bytes: []byte(file.Content), Mode: 0644}
	}
	capture, err := guardedwrite.CaptureFiles(rootAbs, paths)
	if err != nil {
		return plan, fmt.Errorf("capture project initialization targets: %w", err)
	}
	for _, file := range paths {
		if target, exists := capture.Files[file]; exists && target.Exists {
			return plan, fmt.Errorf("project initialization target already exists: %s", file)
		}
	}
	result, err := guardedwrite.Apply(capture.Root, capture, changes)
	plan.Written = append([]string(nil), result.CompletedPaths...)
	if err != nil {
		return plan, fmt.Errorf("project initialization was partial after %s: %w", strings.Join(plan.Written, ", "), err)
	}
	return plan, nil
}

func initPlanDigest(plan InitPlan) (string, error) {
	value := map[string]any{"apiVersion": plan.APIVersion, "name": plan.Name, "files": plan.Files}
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode project initialization plan: %w", err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}
