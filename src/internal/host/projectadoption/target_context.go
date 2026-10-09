package projectadoption

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
)

// DistillationTargetManager is accepted target-model context. It informs
// proposal placement but is not source evidence and cannot ground claims.
type DistillationTargetManager struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Purpose   string   `json:"purpose"`
	Parent    string   `json:"parent"`
	Owns      []string `json:"owns"`
}

// DistillationTargetContract contains only public selected model Statements.
type DistillationTargetContract struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Namespace   string   `json:"namespace"`
	Owner       string   `json:"owner"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Uses        []string `json:"uses"`
	Requires    []string `json:"requires"`
}

// DistillationTargetContext gives a generator the target's accepted model
// hierarchy, public contracts, and selected model paths. It deliberately
// excludes target inventory bytes and does not turn target facts into source
// evidence.
type DistillationTargetContext struct {
	ProjectDigest   string                       `json:"projectDigest"`
	Revision        string                       `json:"revision"`
	ModelDigest     string                       `json:"modelDigest"`
	RootManagerID   string                       `json:"rootManagerId"`
	Managers        []DistillationTargetManager  `json:"managers"`
	PublicContracts []DistillationTargetContract `json:"publicContracts"`
	ModelFiles      []string                     `json:"modelFiles"`
	Digest          string                       `json:"digest"`
}

// TargetContextForProject derives a bounded target context from a Project
// already loaded and accepted by the caller. It performs no repository reads.
func TargetContextForProject(project *projectwork.Project) (DistillationTargetContext, error) {
	if project == nil || project.Snapshot == nil || project.Provisional || project.Snapshot.Provisional || !validFullCommit(project.Revision) || project.Snapshot.ID != project.Revision {
		return DistillationTargetContext{}, errors.New("target context requires a fixed committed Project revision and snapshot")
	}
	if !validDigest(project.Digest) || !validDigest(digestWithoutPrefix(project.Report.ModelDigest)) || project.Report.ModelDigest != project.Model.Digest || project.Report.Status != "succeeded" || !validDigest(digestWithoutPrefix(project.Report.Digest)) {
		return DistillationTargetContext{}, errors.New("target context requires a successful project-model report and bound model digest")
	}
	target := DistillationTargetContext{
		ProjectDigest:   project.Digest,
		Revision:        project.Revision,
		ModelDigest:     strings.TrimPrefix(project.Report.ModelDigest, "sha256:"),
		Managers:        []DistillationTargetManager{},
		PublicContracts: []DistillationTargetContract{},
		ModelFiles:      append([]string{}, project.Config.ModelFiles...),
	}
	for _, manager := range project.Report.Managers {
		if manager.Parent == "" && manager.Namespace == "" {
			if target.RootManagerID != "" {
				return DistillationTargetContext{}, errors.New("target Project has multiple root Managers")
			}
			target.RootManagerID = manager.ID
		}
		target.Managers = append(target.Managers, DistillationTargetManager{
			ID: manager.ID, Name: manager.Name, Namespace: manager.Namespace,
			Purpose: manager.Purpose, Parent: manager.Parent,
			Owns: append([]string{}, manager.Owns...),
		})
	}
	for _, statement := range project.Report.Statements {
		if !statement.Public {
			continue
		}
		target.PublicContracts = append(target.PublicContracts, DistillationTargetContract{
			ID: statement.ID, Name: statement.Name, Namespace: statement.Namespace,
			Owner: statement.Owner, Category: statement.Category,
			Description: statement.Description,
			Uses:        append([]string{}, statement.Uses...), Requires: append([]string{}, statement.Requires...),
		})
	}
	sort.Slice(target.Managers, func(i, j int) bool {
		if target.Managers[i].Namespace != target.Managers[j].Namespace {
			return target.Managers[i].Namespace < target.Managers[j].Namespace
		}
		return target.Managers[i].ID < target.Managers[j].ID
	})
	for i := range target.Managers {
		sort.Strings(target.Managers[i].Owns)
	}
	sort.Slice(target.PublicContracts, func(i, j int) bool { return target.PublicContracts[i].ID < target.PublicContracts[j].ID })
	for i := range target.PublicContracts {
		sort.Strings(target.PublicContracts[i].Uses)
		sort.Strings(target.PublicContracts[i].Requires)
	}
	sort.Strings(target.ModelFiles)
	SealTargetContext(&target)
	if err := ValidateTargetContext(target); err != nil {
		return DistillationTargetContext{}, err
	}
	return target, nil
}

func SealTargetContext(target *DistillationTargetContext) {
	if target == nil {
		return
	}
	copy := *target
	copy.Digest = ""
	target.Digest = digestValue(copy)
}

func ValidateTargetContext(target DistillationTargetContext) error {
	if !validDigest(target.ProjectDigest) || !validFullCommit(target.Revision) || !validDigest(target.ModelDigest) || target.RootManagerID == "" || target.Managers == nil || target.PublicContracts == nil || target.ModelFiles == nil {
		return errors.New("target context requires fixed project/model digests, revision, root Manager, and explicit Manager, contract, and model-file lists")
	}
	managerIDs := map[string]DistillationTargetManager{}
	namespaces := map[string]string{}
	rootCount := 0
	for _, manager := range target.Managers {
		if strings.TrimSpace(manager.ID) == "" || strings.TrimSpace(manager.Name) == "" || manager.Owns == nil {
			return errors.New("target Manager context requires an ID, name, and explicit ownership list")
		}
		if _, exists := managerIDs[manager.ID]; exists {
			return fmt.Errorf("target context repeats Manager %q", manager.ID)
		}
		if prior, exists := namespaces[manager.Namespace]; exists {
			return fmt.Errorf("target context maps namespace %q to Managers %q and %q", manager.Namespace, prior, manager.ID)
		}
		managerIDs[manager.ID] = manager
		namespaces[manager.Namespace] = manager.ID
		if manager.Parent == "" && manager.Namespace == "" {
			rootCount++
			if manager.ID != target.RootManagerID {
				return errors.New("target rootManagerId does not identify the active root Manager")
			}
		} else if manager.Parent == "" {
			return fmt.Errorf("target Manager %q has no parent", manager.ID)
		}
	}
	if rootCount != 1 || managerIDs[target.RootManagerID].ID == "" {
		return errors.New("target context must identify exactly one active root Manager")
	}
	for _, manager := range target.Managers {
		if manager.Parent != "" {
			if _, exists := managerIDs[manager.Parent]; !exists {
				return fmt.Errorf("target Manager %q refers to unknown parent %q", manager.ID, manager.Parent)
			}
		}
	}
	contractIDs := map[string]bool{}
	for _, contract := range target.PublicContracts {
		if strings.TrimSpace(contract.ID) == "" || strings.TrimSpace(contract.Description) == "" || contract.Uses == nil || contract.Requires == nil {
			return errors.New("target public contract context requires stable identity, description, and explicit references")
		}
		if contractIDs[contract.ID] {
			return fmt.Errorf("target context repeats public contract %q", contract.ID)
		}
		contractIDs[contract.ID] = true
		if _, exists := managerIDs[contract.Owner]; !exists {
			return fmt.Errorf("target public contract %q refers to unknown owner %q", contract.ID, contract.Owner)
		}
	}
	seenPaths := map[string]bool{}
	for _, file := range target.ModelFiles {
		if err := validateRepoPath(file); err != nil || !strings.HasPrefix(file, projectwork.ModelRoot+"/") || path.Ext(file) != ".yaml" && path.Ext(file) != ".yml" {
			return fmt.Errorf("target model path %q is not a canonical selected YAML file", file)
		}
		key := strings.ToLower(file)
		if seenPaths[key] {
			return fmt.Errorf("target model paths repeat or alias %q", file)
		}
		seenPaths[key] = true
	}
	copy := target
	copy.Digest = ""
	if !validDigest(target.Digest) || digestValue(copy) != target.Digest {
		return errors.New("target context digest mismatch")
	}
	return nil
}

// ValidateDistillationTarget checks an optional Distillation target binding
// against the exact fixed target Project. Older non-agent reports may omit
// all target fields; agent-generated reports are target-bound by validation.
func ValidateDistillationTarget(report Distillation, target *projectwork.Project) error {
	if report.TargetBasis == "" && report.TargetRevision == "" && report.TargetContextDigest == "" {
		if report.Method == "agent-assisted" {
			return errors.New("agent-assisted distillation must bind its fixed target project and context")
		}
		return nil
	}
	if !validDigest(report.TargetBasis) || !validFullCommit(report.TargetRevision) || !validDigest(report.TargetContextDigest) {
		return errors.New("target-bound distillation must bind a project digest, full revision, and target-context digest together")
	}
	targetContext, err := TargetContextForProject(target)
	if err != nil {
		return fmt.Errorf("validate fixed target context: %w", err)
	}
	if report.TargetBasis != targetContext.ProjectDigest || report.TargetRevision != targetContext.Revision || report.TargetContextDigest != targetContext.Digest {
		return errors.New("distillation target basis, revision, or context differs from the loaded fixed target project")
	}
	return nil
}
