package run

import (
	"fmt"
	"strings"
)

// Closure names are Run-internal. Execute records them and ValidateClosed
// requires them; no other package names a Run resource.
const (
	workersResource    = "manager-workers"
	gitServiceResource = "git-service"
)

func sessionResource(pkg string) string { return "session:" + pkg }
func copyResource(pkg string) string    { return "copy:" + pkg }

// ValidateClosed is the one check that a returned or stored result is complete,
// bound to request, technically completed, fully closed and carries a decision
// valid for its mandate with the Findings that decision requires. Callers must
// open no further Run or publish before it passes. Review may retain identical
// objects under a new key.
func ValidateClosed(value Result, request Request) error {
	if !request.Prepared.Valid() {
		return fmt.Errorf("closed result requires checked Project preparation")
	}
	var managers []string
	for _, m := range request.Prepared.Topology().Managers() {
		managers = append(managers, m.Package)
	}
	return validateClosed(value, request, request.Prepared.Binding(), managers)
}

func validateClosed(value Result, request Request, binding string, managers []string) error {
	if value.ID != request.ID || value.Project != request.Project || value.Binding != binding || value.Mandate != request.Mandate || value.Input != request.Input || value.Technical != Completed || len(value.Errors) != 0 || value.Candidate == nil {
		return fmt.Errorf("Run did not return a complete bound durable result")
	}
	c := *value.Candidate
	if c.Project != request.Project || c.Binding != value.Binding || c.Key == "" || c.Format != request.Input.Format || c.Commit == "" || c.Tree == "" {
		return fmt.Errorf("Run candidate binding mismatch")
	}
	if request.Mandate != Implement && (c.Commit != request.Input.Commit || c.Tree != request.Input.Tree) {
		return fmt.Errorf("review changed its fixed input candidate")
	}
	valid := value.Decision == ModelBlocker || value.Decision == Reject
	if request.Mandate == Implement {
		valid = valid || value.Decision == Submitted
	} else {
		valid = valid || value.Decision == Approve
	}
	if !valid {
		return fmt.Errorf("mandate returned an invalid decision")
	}
	if value.Decision == Reject || value.Decision == ModelBlocker {
		concrete := false
		for _, f := range value.Findings {
			concrete = concrete || strings.TrimSpace(f.Text) != ""
		}
		if !concrete {
			return fmt.Errorf("rejection or model blocker requires mandatory Findings")
		}
	}
	closed := map[string]bool{}
	for _, c := range value.Cleanup {
		if !c.Closed || c.Error != "" || closed[c.Resource] {
			return fmt.Errorf("Run cleanup is uncertain or duplicated")
		}
		closed[c.Resource] = true
	}
	for _, pkg := range managers {
		if !closed[sessionResource(pkg)] || !closed[copyResource(pkg)] {
			return fmt.Errorf("Manager resources not confirmed closed")
		}
	}
	if !closed[workersResource] || !closed[gitServiceResource] || len(value.Handoffs) != len(managers) {
		return fmt.Errorf("complete hierarchy or Git closure missing")
	}
	return nil
}
