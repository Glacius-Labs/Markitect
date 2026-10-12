// Package projectwork owns the selected project-model frontend and controlled
// model writes. The structural Core remains source- and provider-independent.
package projectwork

import "github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"

// InitPlan is an application-only plan for initializing an adopting project.
type InitPlan struct {
	APIVersion string                    `json:"apiVersion"`
	Name       string                    `json:"name"`
	Digest     string                    `json:"digest"`
	Files      []projectvalue.FileChange `json:"files"`
	Written    []string                  `json:"written,omitempty"`
}
