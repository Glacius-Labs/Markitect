package project

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// CandidateRef names self-contained stored Git objects. It owns no live resource.
var ErrCandidateInvalid = errors.New("candidate violates fixed project intent or accounting")

type CandidateRef struct {
	Project string `json:"project"`
	Key     string `json:"key"`
	Format  string `json:"format"`
	Commit  string `json:"commit"`
	Tree    string `json:"tree"`
	Binding string `json:"binding"`
}

// Prepared is constructed only from complete checked acquisition. It retains
// exact source bytes and full Package inventory, independently of model digest.
type Prepared struct {
	attempt  CapturedPreparationAttempt
	topology organization.Topology
	binding  string
}

func Prepare(a CapturedPreparationAttempt) (Prepared, error) {
	if a.Error() != nil || structuralPreparationStatus(a.Model().Checked().Status, a.Registry().Status(), a.Files().Status(), a.Execution().Status(), a.Contexts().Status()) != model.Passed || a.Model().Inputs().Guard == "" {
		return Prepared{}, fmt.Errorf("project requires complete checked committed preparation")
	}
	t, err := organization.BuildTopology(a.Model().Checked())
	if err != nil {
		return Prepared{}, err
	}
	// Source entries include empty directories, modes, dependencies and controls.
	type source struct {
		Selection  ModelSelection
		Root, Area string
		Model      model.SourceModel
		Files      string
	}
	raw := a.Model().Inputs().Raw
	var sources []source
	for _, m := range append([]ModelSource{raw.Local}, raw.Dependencies...) {
		sources = append(sources, source{m.Selection, m.Root.String(), m.RepositoryArea.String(), m.Source, m.Files.Digest().String()})
	}
	data, err := json.Marshal(struct {
		Sources  []source
		Managers []organization.Manager
		Controls [][]SourceFile
	}{sources, t.Managers(), [][]SourceFile{raw.Execution, raw.Registry, raw.Ignore, raw.Instructions}})
	if err != nil {
		return Prepared{}, err
	}
	// SourceFile has immutable entry internals; bind their actual bytes explicitly.
	h := sha256.New()
	h.Write(data)
	fmt.Fprintf(h, "census:%s\n", raw.Census.Digest().String())
	for _, group := range [][]SourceFile{raw.Execution, raw.Registry, raw.Ignore, raw.Instructions} {
		for _, f := range group {
			fmt.Fprintf(h, "%q:%q:%q:%d:%s\n", f.Location.Boundary, f.Location.Path.String(), f.RepositoryPath.String(), f.File.Mode(), f.File.ContentID().String())
		}
	}
	return Prepared{a, t, hex.EncodeToString(h.Sum(nil))}, nil
}
func (p Prepared) Binding() string                     { return p.binding }
func (p Prepared) Valid() bool                         { return p.binding != "" }
func (p Prepared) Topology() organization.Topology     { return p.topology }
func (p Prepared) Attempt() CapturedPreparationAttempt { return p.attempt }
func (p Prepared) Selection() InputSelection           { return p.attempt.Model().Request().Selection }
func (p Prepared) Context(pkg string) (organization.ContextSelection, bool) {
	all, ok := p.attempt.Contexts().Selections()
	if ok {
		for _, c := range all {
			if c.Package() == pkg {
				return c, true
			}
		}
	}
	return organization.ContextSelection{}, false
}
func (p Prepared) Checks() []ExecutionCheck {
	d, ok := p.attempt.Execution().Document()
	if !ok || d.Checks == nil {
		return nil
	}
	return d.Checks.Value
}

// ValidateCandidate preserves fixed intent while allowing registry realization.
// Invalid candidates are legitimate inputs to review, but cannot be submitted
// as a successful Implement. The adapter supplies a fixed revision acquisition.
//
// A stale registry link is realization still to do (StaleLinks): it
// invalidates the state only within responsible's responsibility, Root
// covering every shard. An empty responsible binds none, so the fixed input of
// a command keeps them as work for Implement.
func (p Prepared) ValidateCandidate(a PreviewPreparationAttempt, responsible string) error {
	if !p.Valid() {
		return fmt.Errorf("missing prepared project")
	}
	if a.Error() != nil {
		var manifest *ManifestPreflightError
		if errors.As(a.Error(), &manifest) && manifest.Inspection().Status == model.Failed {
			return fmt.Errorf("%w: invalid candidate manifest: %v", ErrCandidateInvalid, a.Error())
		}
		return fmt.Errorf("candidate acquisition: %w", a.Error())
	}
	before, after := p.attempt.Model().Inputs().Raw, a.Model().Inputs().Raw
	checked := a.Model().Checked()
	if checked.Status == model.NotCompleted {
		return fmt.Errorf("%w: candidate Model observation incomplete", ErrIncompleteInputs)
	}
	t, err := organization.BuildTopology(checked)
	if err != nil || !reflect.DeepEqual(p.topology.Managers(), t.Managers()) || p.topology.ModelID() != t.ModelID() || p.topology.ModelDigest() != t.ModelDigest() {
		return fmt.Errorf("%w: changes checked model or complete Package inventory", ErrCandidateInvalid)
	}
	oldModels := append([]ModelSource{before.Local}, before.Dependencies...)
	newModels := append([]ModelSource{after.Local}, after.Dependencies...)
	if len(oldModels) != len(newModels) {
		return fmt.Errorf("%w: dependency inventory changed", ErrCandidateInvalid)
	}
	for i, old := range oldModels {
		next := newModels[i]
		if old.Source.ManifestPath != next.Source.ManifestPath || old.Root != next.Root || old.Files.Digest() != next.Files.Digest() || !sameSourceEntries(old.Source.Entries, next.Source.Entries) {
			return fmt.Errorf("%w: complete model source inventory changed", ErrCandidateInvalid)
		}
	}
	// External selected instruction bytes cannot drift outside the repository
	// Census while reusing this fixed binding.
	for _, old := range before.Instructions {
		if old.RepositoryPath.String() != "" {
			continue
		}
		found := false
		for _, next := range after.Instructions {
			if next.Location == old.Location && next.RepositoryPath == old.RepositoryPath && next.File.Mode() == old.File.Mode() && next.File.ContentID() == old.File.ContentID() {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%w: external instruction input changed", ErrCandidateInvalid)
		}
	}
	protections, _ := p.attempt.Files().Protections()
	for _, change := range snapshot.Compare(before.Census, after.Census) {
		path := change.Path()
		if strings.HasPrefix(path.String(), ".markitect/registry/") {
			continue
		}
		for _, protection := range protections {
			if protection.Covers(path) {
				return fmt.Errorf("%w: changes protected input %s", ErrCandidateInvalid, path)
			}
		}
	}
	status := structuralPreparationStatus(checked.Status, a.Registry().Status(), a.Files().Status(), a.Execution().Status(), a.Contexts().Status())
	if status == model.Failed {
		return candidateStructuralError(a)
	}
	if status != model.Passed {
		return fmt.Errorf("%w: candidate structural observation incomplete", ErrIncompleteInputs)
	}
	if stale := StaleLinks(a, responsible); len(stale) != 0 {
		return candidateError("registry links to Definitions the checked model no longer has", func(add func(code, path, message string)) {
			for _, f := range stale {
				add(f.Code, f.Path.String(), f.Message)
			}
		})
	}
	return nil
}

// StaleLinks returns the registry links of a checked state to Definitions the
// model no longer has, limited to the shards within manager's responsibility:
// Root covers every shard, another Manager the shards of its Package subtree,
// and an empty manager none.
func StaleLinks(a PreviewPreparationAttempt, manager string) []realization.Finding {
	accounting, ok := a.Files().Accounting()
	if !ok || manager == "" {
		return nil
	}
	var stale []realization.Finding
	for _, f := range accounting.Stale() {
		owner, local := f.Owner.Package()
		if manager == organization.Root || local && packageWithin(owner, manager) {
			stale = append(stale, f)
		}
	}
	return stale
}

// Keep candidate repair feedback at the existing error seam. Only actual
// structural diagnostics are projected, never raw files or runtime inputs.
func candidateStructuralError(a PreviewPreparationAttempt) error {
	return candidateError("structural checks failed", func(add func(code, path, message string)) {
		if accounting, ok := a.Files().Accounting(); ok {
			for _, d := range accounting.Findings() {
				add(d.Code, d.Path.String(), d.Message)
			}
		}
		for _, d := range a.Registry().Diagnostics() {
			add(d.Code, d.Source.String(), d.Message)
		}
		for _, d := range a.Files().Diagnostics() {
			add(d.Code, d.Source.File, d.Message)
		}
		for _, d := range a.Model().Checked().Diagnostics {
			add(d.Code, d.Source.Path, d.Message)
		}
		for _, d := range a.Contexts().Diagnostics() {
			add(d.Code, "", d.Message)
		}
	})
}

// candidateError projects the diagnostics that list adds within the feedback
// size limit.
func candidateError(reason string, list func(add func(code, path, message string))) error {
	const limit = 8192
	var detail strings.Builder
	omitted := false
	clip := func(text string, size int) string {
		if len(text) > size {
			return strings.ToValidUTF8(text[:size], "") + "..."
		}
		return text
	}
	add := func(code, path, message string) {
		if path != "" {
			if _, err := snapshot.NewPath(path); err != nil {
				path = "<external input>"
			}
		}
		line := fmt.Sprintf("\n%s path=%q: %q", clip(code, 128), clip(path, 1024), clip(message, 1024))
		if detail.Len()+len(line) > limit-256 {
			omitted = true
			return
		}
		detail.WriteString(line)
	}
	list(add)
	if omitted {
		detail.WriteString("\nAdditional structural diagnostics omitted (feedback size limit).")
	}
	return fmt.Errorf("%w: %s%s", ErrCandidateInvalid, reason, detail.String())
}

func sameSourceEntries(a, b []model.Entry) bool {
	a = slices.Clone(a)
	b = slices.Clone(b)
	less := func(x, y model.Entry) int {
		return cmp.Or(strings.Compare(x.Path, y.Path), strings.Compare(string(x.Kind), string(y.Kind)))
	}
	slices.SortFunc(a, less)
	slices.SortFunc(b, less)
	return reflect.DeepEqual(a, b)
}

// ChecksFor retains obligation eligibility, but requiredBy alone cannot place
// a check in a private copy. Its declared directory and every nested ownership
// area must also belong to this Manager's subtree. Root runs the complete suite.
func (p Prepared) ChecksFor(manager string) []ExecutionCheck {
	var selected []ExecutionCheck
	for _, c := range p.Checks() {
		if manager == organization.Root || c.RequiredBy != nil && packageWithin(c.RequiredBy.Value.Package, manager) && p.checkDirectoryWithin(c, manager) {
			selected = append(selected, c)
		}
	}
	return selected
}

func packageWithin(pkg, manager string) bool {
	return pkg == manager || strings.HasPrefix(pkg, manager+"/")
}

func (p Prepared) checkDirectoryWithin(c ExecutionCheck, manager string) bool {
	layout, ok := p.attempt.Files().Layout()
	if !ok || c.Directory.Value.IsRoot() {
		return false
	}
	directory := c.Directory.Value.String()
	owner, ok := layout.DirectoryOwner(directory)
	pkg, local := owner.Package()
	if !ok || !local || !packageWithin(pkg, manager) {
		return false
	}
	for _, area := range layout.Areas() {
		if area.Path == directory || strings.HasPrefix(area.Path, directory+"/") {
			pkg, local := area.Owner.Package()
			if !local || !packageWithin(pkg, manager) {
				return false
			}
		}
	}
	return true
}
