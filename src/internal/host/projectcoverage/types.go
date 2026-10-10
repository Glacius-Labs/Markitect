// Package projectcoverage inventories and classifies repository paths without
// depending on the projectwork frontend or any model provider.
package projectcoverage

import (
	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

const (
	APIVersion = "project.markitect.example.org/repository-coverage/v1alpha1"
	IgnorePath = ".markitect/ignore.yaml"
)

type Classification string

const (
	ClassRealization    Classification = "realization"
	ClassCanonicalModel Classification = "canonical-model"
	ClassToolOwned      Classification = "tool-owned"
	ClassIgnored        Classification = "ignored"
	ClassTransitional   Classification = "transitional"
)

// ToolPath is an explicitly registered exact path or directory prefix owned
// by Markitect. Operational entries are classified but excluded from the
// stable coverage digest because the active run mutates them.
type ToolPath struct {
	Selector    string `json:"selector" yaml:"selector"`
	Owner       string `json:"owner" yaml:"owner"`
	Operational bool   `json:"operational,omitempty" yaml:"operational,omitempty"`
}

type TransitionalExclusion struct {
	Path   string `json:"path" yaml:"path"`
	Reason string `json:"reason" yaml:"reason"`
}

// Options are supplied by the Host. Canonical model paths and tool-owned
// selectors are explicit; this package never guesses them from filenames.
type Options struct {
	CanonicalModelPaths []string                `json:"canonicalModelPaths"`
	ToolPaths           []ToolPath              `json:"toolPaths"`
	Transitional        []TransitionalExclusion `json:"transitional"`
}

// FileState describes one path's presence in one repository layer. Digest is
// SHA-256 of bytes, except Index.Digest which is the Git blob object ID.
type FileState struct {
	Present bool   `json:"present"`
	Mode    string `json:"mode,omitempty"`
	Digest  string `json:"digest,omitempty"`
}

type PathState struct {
	Path           string    `json:"path"`
	Head           FileState `json:"head"`
	Index          FileState `json:"index"`
	Worktree       FileState `json:"worktree"`
	OpaqueBoundary bool      `json:"opaqueBoundary,omitempty"`
}

// Universe is a deterministic HEAD/index/worktree path census. IgnoreBytes
// is retained only in memory so candidate overlays can reparse the exact
// policy. It is represented in reports by IgnoreBytesDigest.
type Universe struct {
	Revision       string      `json:"revision"`
	IdentityDigest string      `json:"identityDigest"`
	FixedRevision  bool        `json:"fixedRevision,omitempty"`
	Paths          []PathState `json:"paths"`
	// Snapshot contains the exact observed, nonignored bytes used to derive
	// Worktree digests. It is intentionally in-memory only so Host consumers
	// can compile the same observation without a second filesystem read.
	Snapshot          *snapshot.Snapshot `json:"-"`
	IgnoreBytes       []byte             `json:"-"`
	IgnoreBytesDigest string             `json:"ignoreBytesDigest"`
	Digest            string             `json:"digest"`
}

type Entry struct {
	Path           string         `json:"path"`
	Class          Classification `json:"class"`
	Owner          string         `json:"owner,omitempty"`
	Reason         string         `json:"reason,omitempty"`
	Artifacts      []string       `json:"artifacts"`
	Statements     []string       `json:"statements"`
	Head           FileState      `json:"head"`
	Index          FileState      `json:"index"`
	Worktree       FileState      `json:"worktree"`
	OpaqueBoundary bool           `json:"opaqueBoundary,omitempty"`
	Operational    bool           `json:"operational,omitempty"`
}

type Finding struct {
	Code     string `json:"code"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

type Report struct {
	APIVersion          string                 `json:"apiVersion"`
	Accounted           bool                   `json:"accounted"`
	Conforming          bool                   `json:"conforming"`
	UniverseDigest      string                 `json:"universeDigest"`
	IgnoreBytesDigest   string                 `json:"ignoreBytesDigest"`
	IgnoreMembersDigest string                 `json:"ignoreMembersDigest"`
	Digest              string                 `json:"digest"`
	Entries             []Entry                `json:"entries"`
	Findings            []Finding              `json:"findings"`
	LegacyDiagnostics   []string               `json:"legacyDiagnostics"`
	ModelDigest         string                 `json:"modelDigest"`
	ModelReportDigest   string                 `json:"modelReportDigest"`
	ModelErrors         []projectmodel.Finding `json:"modelErrors,omitempty"`
}

type Request struct {
	Universe         *Universe
	Model            projectmodel.Report
	Options          Options
	LegacyRoots      []string
	LegacyExclusions []TransitionalExclusion
}

// Delta is an explicit candidate path change. Content is needed only when the
// changed path is IgnorePath; all other paths use the supplied digest.
type Delta struct {
	Path    string
	Delete  bool
	Mode    string
	Digest  string
	Content []byte
}
