// Package projectwork owns the selected project-model frontend and controlled
// model writes. The structural Core remains source- and provider-independent.
package projectwork

import (
	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

const ManifestPath = ".markitect/project.yaml"
const ModelRoot = ".markitect/model"
const RuntimePath = ".markitect/runtime.yaml"

type Exclusion struct {
	Path   string `json:"path" yaml:"path"`
	Reason string `json:"reason" yaml:"reason"`
}

type Config struct {
	APIVersion     string      `json:"apiVersion" yaml:"apiVersion"`
	Name           string      `json:"name" yaml:"name"`
	ModelFiles     []string    `json:"modelFiles" yaml:"modelFiles"`
	InventoryRoots []string    `json:"inventoryRoots" yaml:"inventoryRoots"`
	Exclusions     []Exclusion `json:"exclusions" yaml:"exclusions"`
}

type Project struct {
	Root        string              `json:"-"`
	Revision    string              `json:"revision"`
	Provisional bool                `json:"provisional"`
	Digest      string              `json:"digest"`
	Config      Config              `json:"config"`
	Model       core.Model          `json:"model"`
	Report      projectmodel.Report `json:"report"`
	Snapshot    *snapshot.Snapshot  `json:"-"`
}

type FileChange struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Delete  bool   `json:"delete,omitempty"`
}

type Mutation struct {
	APIVersion string       `json:"apiVersion"`
	BaseDigest string       `json:"baseDigest"`
	Actor      string       `json:"actor"`
	Goal       string       `json:"goal"`
	Files      []FileChange `json:"files"`
}

type EditPlan struct {
	APIVersion      string                    `json:"apiVersion"`
	BaseDigest      string                    `json:"baseDigest"`
	Digest          string                    `json:"digest"`
	Mutation        Mutation                  `json:"mutation"`
	CandidateDigest string                    `json:"candidateDigest"`
	Report          projectmodel.Report       `json:"report"`
	Impact          projectmodel.ChangeImpact `json:"impact"`
}

type InitPlan struct {
	APIVersion string       `json:"apiVersion"`
	Name       string       `json:"name"`
	Digest     string       `json:"digest"`
	Files      []FileChange `json:"files"`
	Written    []string     `json:"written,omitempty"`
}
