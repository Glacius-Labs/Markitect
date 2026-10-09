// Package projectknowledge builds a bounded projection over explicitly selected facts.
package projectknowledge

import (
	"encoding/json"
	"errors"
	"github.com/Glacius-Labs/Markitect/internal/core"
)

const GraphVersion = "projectknowledge/v1"
const QueryVersion = "projectknowledge/walk/v1"

const (
	maxNodes        = 5_000
	maxEdges        = 20_000
	maxPayloadBytes = 8 << 20
	maxDepth        = 32
	maxSteps        = 50_000
	maxResults      = 5_000
)

const (
	FactKnown   = "known"
	FactPartial = "partial"
	FactUnknown = "unknown"
)

var ErrNotFound = errors.New("project knowledge item not found")

// Fact is a neutral caller-owned node, such as a selected file, ownership
// claim, or declaration. State describes completeness of this supplied fact
// only; it says nothing about freshness or operational evidence. ID must be
// unique and must not equal a Core identity key. Omitted Properties differ from
// explicitly empty JSON values. Graph semantics belong in selected Relations.
type Fact struct {
	ID         string                     `json:"id"`
	Kind       string                     `json:"kind"`
	State      string                     `json:"state"`
	Properties map[string]json.RawMessage `json:"properties,omitempty"`
	Source     core.Source                `json:"source,omitempty"`
}

// Relation is a caller-supplied relation with opaque provenance. Basis records
// the adapter's derivation method; it is not an ownership authority.
type Relation struct {
	From     string      `json:"from"`
	To       string      `json:"to"`
	Property string      `json:"property"`
	Source   core.Source `json:"source,omitempty"`
	Basis    string      `json:"basis,omitempty"`
}

type ProjectFacts struct {
	// SnapshotDigest binds the exact source snapshot from which these facts came.
	SnapshotDigest string `json:"snapshotDigest"`
	// ProjectDigest is optional when the caller has no project mapping digest.
	ProjectDigest string `json:"projectDigest,omitempty"`
	// Digest binds the exact facts and relations supplied to Build.
	Digest    string     `json:"digest"`
	Facts     []Fact     `json:"facts,omitempty"`
	Relations []Relation `json:"relations,omitempty"`
}

type EdgeKey struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Property string `json:"property"`
}

// NodeSelection explicitly names one node and the fields it exposes. Purpose
// and Source are opt-in; identity is always retained for selected nodes.
type NodeSelection struct {
	ID             string   `json:"id"`
	Fields         []string `json:"fields,omitempty"`
	IncludePurpose bool     `json:"includePurpose,omitempty"`
	IncludeSource  bool     `json:"includeSource,omitempty"`
}

// Scope is a closed allowlist. Empty scope never means all nodes or edges.
type Scope struct {
	ID    string          `json:"id"`
	Nodes []NodeSelection `json:"nodes"`
	Edges []EdgeKey       `json:"edges"`
}

type Binding struct {
	GraphVersion   string `json:"graphVersion"`
	ModelDigest    string `json:"modelDigest"`
	Revision       string `json:"revision,omitempty"`
	SnapshotDigest string `json:"snapshotDigest"`
	ProjectDigest  string `json:"projectDigest,omitempty"`
	FactsDigest    string `json:"factsDigest"`
	ScopeID        string `json:"scopeId"`
	GraphDigest    string `json:"graphDigest"`
}

type Node struct {
	ID         string                     `json:"id"`
	Kind       string                     `json:"kind"`
	FactState  string                     `json:"factState,omitempty"`
	Properties map[string]json.RawMessage `json:"properties,omitempty"`
	Purpose    string                     `json:"purpose,omitempty"`
	Source     *core.Source               `json:"source,omitempty"`
}

type Edge struct {
	From     string      `json:"from"`
	To       string      `json:"to"`
	Property string      `json:"property"`
	Source   core.Source `json:"source,omitempty"`
	Basis    string      `json:"basis,omitempty"`
}

type Index struct {
	binding Binding
	nodes   []Node
	edges   []Edge
	byID    map[string]int
	out     map[string][]int
	in      map[string][]int
}
