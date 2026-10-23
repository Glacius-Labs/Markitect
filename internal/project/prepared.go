package project

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// CandidateRef names self-contained stored Git objects. It owns no live resource.
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
func (p Prepared) ValidateCandidate(a PreviewPreparationAttempt) error {
	if !p.Valid() {
		return fmt.Errorf("missing prepared project")
	}
	before, after := p.attempt.Model().Inputs().Raw, a.Model().Inputs().Raw
	checked := a.Model().Checked()
	t, err := organization.BuildTopology(checked)
	if err != nil || !reflect.DeepEqual(p.topology.Managers(), t.Managers()) || p.topology.ModelID() != t.ModelID() || p.topology.ModelDigest() != t.ModelDigest() {
		return fmt.Errorf("candidate changes checked model or complete Package inventory")
	}
	protections, _ := p.attempt.Files().Protections()
	for _, change := range snapshot.Compare(before.Census, after.Census) {
		path := change.Path()
		if strings.HasPrefix(path.String(), ".markitect/registry/") {
			continue
		}
		for _, protection := range protections {
			if protection.Covers(path) {
				return fmt.Errorf("candidate changes protected input %s", path)
			}
		}
	}
	if a.Error() != nil || structuralPreparationStatus(checked.Status, a.Registry().Status(), a.Files().Status(), a.Execution().Status(), a.Contexts().Status()) != model.Passed {
		return fmt.Errorf("candidate structural checks failed")
	}
	return nil
}
