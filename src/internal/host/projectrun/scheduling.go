package projectrun

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

type managerAction struct {
	managerID string
	phase     string
}

// managerDependencies returns prerequisite Manager IDs keyed by dependent
// Manager ID. Only explicit Statement Uses/Requires references create edges.
// Parent/child relationships remain governed by the work and integration
// phases and are omitted from this cross-branch graph.
func managerDependencies(report projectmodel.Report, tasks []ManagerTask) (map[string][]string, error) {
	taskByID := make(map[string]ManagerTask, len(tasks))
	for _, task := range tasks {
		if task.ManagerID == "" {
			return nil, fmt.Errorf("manager task has an empty Manager ID")
		}
		if _, exists := taskByID[task.ManagerID]; exists {
			return nil, fmt.Errorf("duplicate manager task %q", task.ManagerID)
		}
		taskByID[task.ManagerID] = task
	}

	statements := make(map[string]projectmodel.Statement, len(report.Statements))
	for _, statement := range report.Statements {
		if statement.ID == "" {
			return nil, fmt.Errorf("statement has an empty ID")
		}
		if _, exists := statements[statement.ID]; exists {
			return nil, fmt.Errorf("duplicate statement %q", statement.ID)
		}
		statements[statement.ID] = statement
	}

	dependencies := make(map[string]map[string]bool, len(tasks))
	for _, task := range tasks {
		dependencies[task.ManagerID] = map[string]bool{}
	}
	for _, statement := range report.Statements {
		consumer := statement.Owner
		if _, selected := taskByID[consumer]; !selected {
			continue
		}
		refs := append(append([]string(nil), statement.Requires...), statement.Uses...)
		sort.Strings(refs)
		for _, ref := range refs {
			target, exists := statements[ref]
			if !exists {
				return nil, fmt.Errorf("statement %s references missing statement %s", statement.ID, ref)
			}
			dependency := target.Owner
			if dependency == "" {
				return nil, fmt.Errorf("dependency statement %s has no Manager owner", target.ID)
			}
			if dependency == consumer || managersRelated(taskByID, consumer, dependency) {
				continue
			}
			if _, selected := taskByID[dependency]; !selected {
				return nil, fmt.Errorf("Manager %s depends on unselected Manager %s through statement %s", consumer, dependency, ref)
			}
			dependencies[consumer][dependency] = true
		}
	}

	result := make(map[string][]string, len(dependencies))
	for managerID, prerequisites := range dependencies {
		for prerequisite := range prerequisites {
			result[managerID] = append(result[managerID], prerequisite)
		}
		sort.Strings(result[managerID])
	}
	if cycle := dependencyCycle(result); len(cycle) > 0 {
		return nil, fmt.Errorf("manager dependency cycle: %s", strings.Join(cycle, " -> "))
	}
	return result, nil
}

// readyManagerActions chooses a deterministic batch. Integrations take
// precedence and are ordered bottom-up; otherwise it returns independent
// ready work actions, up to maxParallel. Scope overlap conservatively prevents
// two actions from sharing a writable path tree.
func readyManagerActions(tasks []ManagerTask, dependencies map[string][]string, maxParallel int) ([]managerAction, error) {
	if maxParallel < 1 {
		return nil, fmt.Errorf("maximum parallel Managers must be positive")
	}
	taskByID := make(map[string]ManagerTask, len(tasks))
	children := make(map[string][]string, len(tasks))
	for _, task := range tasks {
		if task.ManagerID == "" {
			return nil, fmt.Errorf("manager task has an empty Manager ID")
		}
		if _, exists := taskByID[task.ManagerID]; exists {
			return nil, fmt.Errorf("duplicate manager task %q", task.ManagerID)
		}
		taskByID[task.ManagerID] = task
	}
	for _, task := range tasks {
		if task.ParentTask == "" {
			continue
		}
		if _, exists := taskByID[task.ParentTask]; !exists {
			return nil, fmt.Errorf("Manager %s has missing parent task %s", task.ManagerID, task.ParentTask)
		}
		children[task.ParentTask] = append(children[task.ParentTask], task.ManagerID)
	}
	for parent := range children {
		sort.Strings(children[parent])
	}
	if cycle := dependencyCycle(dependencies); len(cycle) > 0 {
		return nil, fmt.Errorf("manager dependency cycle: %s", strings.Join(cycle, " -> "))
	}
	for managerID, prerequisites := range dependencies {
		if _, exists := taskByID[managerID]; !exists {
			return nil, fmt.Errorf("dependency graph contains unknown Manager %s", managerID)
		}
		for _, prerequisite := range prerequisites {
			if _, exists := taskByID[prerequisite]; !exists {
				return nil, fmt.Errorf("Manager %s depends on unknown Manager %s", managerID, prerequisite)
			}
			if managersRelated(taskByID, managerID, prerequisite) {
				return nil, fmt.Errorf("dependency graph contains parent/child edge %s -> %s", managerID, prerequisite)
			}
		}
	}

	var readyIntegrations []ManagerTask
	for _, task := range tasks {
		if !hasDirectChildren(children, task.ManagerID) || !integrateEligible(task.State) {
			continue
		}
		ready := true
		for _, childID := range children[task.ManagerID] {
			child := taskByID[childID]
			if !managerTreeComplete(child, children) {
				ready = false
				break
			}
		}
		if ready {
			readyIntegrations = append(readyIntegrations, task)
		}
	}
	if len(readyIntegrations) > 0 {
		sort.Slice(readyIntegrations, func(i, j int) bool {
			if readyIntegrations[i].Depth != readyIntegrations[j].Depth {
				return readyIntegrations[i].Depth > readyIntegrations[j].Depth
			}
			return readyIntegrations[i].ManagerID < readyIntegrations[j].ManagerID
		})
		return selectNonOverlapping(readyIntegrations, "integrate", maxParallel), nil
	}

	var readyWork []ManagerTask
	for _, task := range tasks {
		if !workEligible(task.State) {
			continue
		}
		if task.ParentTask != "" {
			parent, exists := taskByID[task.ParentTask]
			if !exists || !workComplete(parent.State) {
				continue
			}
		}
		ready := true
		for _, prerequisite := range dependencies[task.ManagerID] {
			dependency := taskByID[prerequisite]
			if !managerTreeComplete(dependency, children) {
				ready = false
				break
			}
		}
		if ready {
			readyWork = append(readyWork, task)
		}
	}
	sort.Slice(readyWork, func(i, j int) bool {
		if readyWork[i].Depth != readyWork[j].Depth {
			return readyWork[i].Depth < readyWork[j].Depth
		}
		return readyWork[i].ManagerID < readyWork[j].ManagerID
	})
	return selectNonOverlapping(readyWork, "work", maxParallel), nil
}

func hasDirectChildren(children map[string][]string, managerID string) bool {
	return len(children[managerID]) > 0
}

func workEligible(state string) bool {
	switch state {
	case "queued", "work-retry-ready", "review-rework-ready":
		return true
	default:
		return false
	}
}

func integrateEligible(state string) bool {
	return state == "worked" || state == "integration-retry-ready"
}

func workComplete(state string) bool {
	return state == "worked" || state == "integrated" || state == "complete"
}

func managerTreeComplete(task ManagerTask, children map[string][]string) bool {
	if task.State == "integrated" || task.State == "complete" {
		return true
	}
	// A leaf has no separate integration phase; its worked candidate is the
	// completed output. A Manager with children is complete only after its own
	// integration action has durably incorporated those child candidates.
	return task.State == "worked" && len(children[task.ManagerID]) == 0
}

func selectNonOverlapping(tasks []ManagerTask, phase string, maxParallel int) []managerAction {
	selected := make([]ManagerTask, 0, maxParallel)
	for _, task := range tasks {
		collides := false
		for _, prior := range selected {
			if ownershipScopesOverlap(task.Owns, prior.Owns) {
				collides = true
				break
			}
		}
		if collides {
			continue
		}
		selected = append(selected, task)
		if len(selected) == maxParallel {
			break
		}
	}
	actions := make([]managerAction, 0, len(selected))
	for _, task := range selected {
		actions = append(actions, managerAction{managerID: task.ManagerID, phase: phase})
	}
	return actions
}

func ownershipScopesOverlap(left, right []string) bool {
	for _, a := range left {
		for _, b := range right {
			if scopeOverlaps(a, b) {
				return true
			}
		}
	}
	return false
}

func scopeOverlaps(left, right string) bool {
	left = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(left)), "/")
	right = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(right)), "/")
	if left == "" || left == "." || right == "" || right == "." {
		return true
	}
	return left == right || strings.HasPrefix(left, right+"/") || strings.HasPrefix(right, left+"/")
}

func managersRelated(tasks map[string]ManagerTask, left, right string) bool {
	return managerIsAncestor(tasks, left, right) || managerIsAncestor(tasks, right, left)
}

func managerIsAncestor(tasks map[string]ManagerTask, ancestor, descendant string) bool {
	seen := map[string]bool{}
	for current := descendant; current != ""; {
		if seen[current] {
			return false
		}
		seen[current] = true
		task, exists := tasks[current]
		if !exists || task.ParentTask == "" {
			return false
		}
		if task.ParentTask == ancestor {
			return true
		}
		current = task.ParentTask
	}
	return false
}

func dependencyCycle(dependencies map[string][]string) []string {
	nodes := make([]string, 0, len(dependencies))
	for node := range dependencies {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	state := make(map[string]uint8, len(nodes))
	stack := make([]string, 0, len(nodes))
	positions := make(map[string]int, len(nodes))
	var cycle []string
	var visit func(string) bool
	visit = func(node string) bool {
		state[node] = 1
		positions[node] = len(stack)
		stack = append(stack, node)
		prerequisites := append([]string(nil), dependencies[node]...)
		sort.Strings(prerequisites)
		for _, prerequisite := range prerequisites {
			if state[prerequisite] == 1 {
				cycle = append(cycle, stack[positions[prerequisite]:]...)
				cycle = append(cycle, prerequisite)
				return true
			}
			if state[prerequisite] == 0 && visit(prerequisite) {
				return true
			}
		}
		stack = stack[:len(stack)-1]
		delete(positions, node)
		state[node] = 2
		return false
	}
	for _, node := range nodes {
		if state[node] == 0 && visit(node) {
			return cycle
		}
	}
	return nil
}
