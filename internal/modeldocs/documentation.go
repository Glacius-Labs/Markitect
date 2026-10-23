// Package modeldocs prepares a generated canonical v1 view and an explicit
// source/target-bound write plan. It is unrelated to realization publication.
package modeldocs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const ViewPath = "docs/markitect/project.md"

type Source struct {
	Boundary       string        `json:"boundary"`
	Path           string        `json:"path"`
	RepositoryPath string        `json:"repositoryPath,omitempty"`
	ContentDigest  string        `json:"contentDigest"`
	Mode           snapshot.Mode `json:"mode"`
}

// Target records actual filesystem bytes and permissions for a working view,
// or the historical Census entry for a fixed preview. Bytes are owned copies.
type Target struct {
	Path          string `json:"path"`
	Exists        bool   `json:"exists"`
	ContentDigest string `json:"contentDigest"`
	Mode          uint32 `json:"mode"`
	Bytes         []byte `json:"-"`
}

type Preview struct {
	Status               model.Status       `json:"status"`
	Path                 string             `json:"path"`
	ContentDigest        string             `json:"contentDigest"`
	PreviewDigest        string             `json:"previewDigest"`
	CanonicalModelDigest string             `json:"canonicalModelDigest"`
	Markdown             string             `json:"markdown"`
	Sources              []Source           `json:"sources"`
	Target               Target             `json:"target"`
	Diagnostics          []model.Diagnostic `json:"diagnostics,omitempty"`
	Written              bool               `json:"written"`
	CompletedPaths       []string           `json:"completedPaths,omitempty"`
}

type WriteResult struct{ CompletedPaths []string }

// Writer observes the target and applies one selected file while its captured
// target remains unchanged. validate checks the actual source/target plan again
// under the independent documentation writer lock. This is no Apply guard.
type Writer interface {
	Observe(context.Context, string, string) (Target, error)
	Write(context.Context, string, Target, []byte, func() error, func() error) (WriteResult, error)
}

type Service struct {
	models      *project.ModelService
	preparation *project.PreparationService
	writer      Writer
}

func New(models *project.ModelService, preparation *project.PreparationService, writer Writer) *Service {
	return &Service{models, preparation, writer}
}

func (s *Service) Preview(ctx context.Context, request project.PreviewRequest) (Preview, error) {
	p, _, _, err := s.prepare(ctx, request)
	return p, err
}

func (s *Service) Write(ctx context.Context, request project.PreviewRequest, expected string) (Preview, error) {
	p, _, controlsPassed, err := s.prepare(ctx, request)
	if err != nil || p.Status != model.Passed {
		return p, err
	}
	if request.Revision != "" {
		p.Status = model.Failed
		return p, errors.New("docs --write requires a working preview, never a historical source")
	}
	if expected == "" || expected != p.PreviewDigest {
		p.Status = model.Failed
		return p, errors.New("documentation previewDigest changed; review a new source/target-bound preview")
	}
	for _, source := range p.Sources {
		if source.RepositoryPath == "" {
			p.Status = model.NotCompleted
			return p, errors.New("documentation write requires repository-bound sources; target disjointness from an unbound model, control or instruction input is not established")
		}
	}
	if !controlsPassed {
		p.Status = model.NotCompleted
		return p, errors.New("documentation write requires validated execution declarations to protect instruction inputs")
	}
	if s.writer == nil {
		p.Status = model.NotCompleted
		return p, errors.New("documentation writer unavailable")
	}
	validate := func() error {
		current, _, controls, err := s.prepare(ctx, request)
		if err != nil {
			return err
		}
		if current.Status != model.Passed || !controls || current.PreviewDigest != expected {
			return errors.New("documentation sources or target changed after preview")
		}
		return nil
	}
	validateSources := func() error {
		current, _, controls, err := s.prepare(ctx, request)
		if err != nil {
			return err
		}
		if current.Status != model.Passed || !controls || !slices.Equal(current.Sources, p.Sources) || current.ContentDigest != p.ContentDigest {
			return errors.New("documentation sources changed during the guarded write")
		}
		return nil
	}
	result, err := s.writer.Write(ctx, request.Selection.Repository, p.Target, []byte(p.Markdown), validate, validateSources)
	p.CompletedPaths = slices.Clone(result.CompletedPaths)
	p.Written = slices.Contains(p.CompletedPaths, p.Path)
	if err != nil {
		p.Status = model.NotCompleted
		return p, err
	}
	return p, nil
}

func (s *Service) prepare(ctx context.Context, request project.PreviewRequest) (Preview, project.PreviewedModel, bool, error) {
	p := Preview{Status: model.NotCompleted, Path: ViewPath}
	m, err := s.models.PreviewModel(ctx, request, model.Limits{})
	if err != nil {
		var manifest *project.ManifestPreflightError
		if errors.As(err, &manifest) {
			inspection := manifest.Inspection()
			p.Status, p.Diagnostics = inspection.Status, inspection.Diagnostics
		}
		return p, m, false, err
	}
	raw := m.Inputs().Raw
	controlsPassed := true
	// A document can describe a model without execution configuration. Where
	// actual execution exists, reuse the real preparation closure to acquire
	// its controls and instructions; a write cannot ignore malformed policy.
	execution, _ := snapshot.NewPath(".markitect/execution.yaml")
	if _, present := raw.Census.Lookup(execution); present {
		prepared, e := s.preparation.Preview(ctx, request, model.Limits{})
		attempts := prepared.Attempts()
		if len(attempts) != 0 {
			a := attempts[len(attempts)-1]
			m = a.Model()
			controlsPassed = a.Execution().Status() == model.Passed
			raw = m.Inputs().Raw
		}
		if e != nil {
			return p, m, false, e
		}
	}
	checked := m.Checked()
	p.Status = checked.Status
	p.Diagnostics = checked.Diagnostics
	if checked.Status != model.Passed {
		return p, m, false, nil
	}
	authored := m.DecodeExecution().Partial()
	for _, ref := range documentationInstructionRefs(authored) {
		if ref == ViewPath {
			p.Status = model.Failed
			return p, m, false, errors.New("documentation target is an authored instruction input")
		}
	}
	for _, input := range append([]project.ModelSource{raw.Local}, raw.Dependencies...) {
		root := repoPath(input.RepositoryArea.String(), input.Root.String())
		if root != "" && (root == "." || ViewPath == root || strings.HasPrefix(ViewPath, root+"/")) {
			p.Status = model.Failed
			return p, m, false, errors.New("documentation target is inside a protected canonical model root")
		}
		for _, file := range input.Files.Entries() {
			p.Sources = append(p.Sources, Source{Boundary: input.Selection.Boundary, Path: file.Path().String(), RepositoryPath: repoPath(input.RepositoryArea.String(), file.Path().String()), ContentDigest: file.ContentID().String(), Mode: file.Mode()})
		}
	}
	for _, group := range [][]project.SourceFile{raw.Execution, raw.Registry, raw.Ignore, raw.RuntimeProfiles, raw.Instructions} {
		for _, file := range group {
			if file.RepositoryPath.String() == ViewPath {
				p.Status = model.Failed
				return p, m, false, errors.New("documentation target is a protected control or instruction input")
			}
			p.Sources = append(p.Sources, Source{Boundary: file.Location.Boundary, Path: file.Location.Path.String(), RepositoryPath: file.RepositoryPath.String(), ContentDigest: file.File.ContentID().String(), Mode: file.File.Mode()})
		}
	}
	slices.SortFunc(p.Sources, func(a, b Source) int {
		if a.Boundary < b.Boundary {
			return -1
		}
		if a.Boundary > b.Boundary {
			return 1
		}
		if a.Path < b.Path {
			return -1
		}
		if a.Path > b.Path {
			return 1
		}
		return 0
	})
	for _, scope := range checked.Models {
		if scope.Local {
			p.CanonicalModelDigest = scope.Digest
		}
	}
	p.Markdown = render(checked)
	p.ContentDigest = digest([]byte(p.Markdown))
	if request.Revision != "" {
		p.Target = Target{Path: ViewPath}
		if file, ok := raw.Census.Lookup(mustPath(ViewPath)); ok {
			p.Target.Exists = true
			p.Target.Bytes = file.Bytes()
			p.Target.ContentDigest = file.ContentID().String()
			p.Target.Mode = 0644
			if file.Mode() == snapshot.Executable {
				p.Target.Mode = 0755
			}
		}
	} else {
		if s.writer == nil {
			p.Status = model.NotCompleted
			return p, m, controlsPassed, errors.New("documentation target observer unavailable")
		}
		p.Target, err = s.writer.Observe(ctx, request.Selection.Repository, ViewPath)
		if err != nil {
			p.Status = model.NotCompleted
			return p, m, controlsPassed, err
		}
	}
	plan := struct {
		Content  string
		Sources  []Source
		Target   Target
		Revision string
	}{p.ContentDigest, p.Sources, p.Target, request.Revision}
	data, err := json.Marshal(plan)
	if err != nil {
		p.Status = model.NotCompleted
		return p, m, controlsPassed, err
	}
	p.PreviewDigest = digest(data)
	return p, m, controlsPassed, nil
}

func documentationInstructionRefs(e project.AuthoredExecution) []string {
	var result []string
	if e.Defaults.Execute.Instructions != nil {
		result = append(result, e.Defaults.Execute.Instructions.Value...)
	}
	if e.Defaults.Verify.Instructions != nil {
		result = append(result, e.Defaults.Verify.Instructions.Value...)
	}
	if e.Packages != nil {
		for _, override := range e.Packages.Value {
			if override.Execute.Instructions != nil {
				result = append(result, override.Execute.Instructions.Value...)
			}
			if override.Verify.Instructions != nil {
				result = append(result, override.Verify.Instructions.Value...)
			}
		}
	}
	return result
}

func repoPath(area, path string) string {
	if area == "" {
		return ""
	}
	if area == "." {
		return path
	}
	if path == "." {
		return area
	}
	return area + "/" + path
}
func mustPath(path string) snapshot.Path { p, _ := snapshot.NewPath(path); return p }
func digest(data []byte) string          { return fmt.Sprintf("sha256:%x", sha256.Sum256(data)) }

func render(checked model.Result) string {
	var b bytes.Buffer
	b.WriteString("# Canonical Markitect model\n\nGenerated from acquired canonical v1 source bytes. This view is not execution or acceptance evidence.\n")
	for _, scope := range checked.Models {
		fmt.Fprintf(&b, "\n## %s\n\nCanonical model digest: `%s`.\n\nManifest: `%s` (original bytes `%s`).\n", scope.ID, scope.Digest, scope.Manifest.Path, scope.Manifest.Digest)
		for _, definition := range scope.Definitions {
			fmt.Fprintf(&b, "\n### %s/%s\n\nKind: `%s/%s`. Source: `%s` (original bytes `%s`).\n\n", definition.Identity.Package, definition.Identity.ID, definition.Kind.Package, definition.Kind.ID, definition.Source.Path, definition.Source.Digest)
			b.WriteString(definition.Body)
			b.WriteByte('\n')
			if len(definition.Spec) > 0 {
				b.WriteString("\nFields:\n\n")
				for _, f := range definition.Spec {
					fmt.Fprintf(&b, "- `%s`: %s\n", f.Name, renderValue(f.Value))
				}
			}
		}
	}
	return b.String()
}

func renderValue(v model.Value) string {
	if text, ok := v.Text(); ok {
		b, _ := json.Marshal(text)
		return string(b)
	}
	if integer, ok := v.Integer(); ok {
		return integer
	}
	if boolean, ok := v.Boolean(); ok {
		return fmt.Sprint(boolean)
	}
	if number, ok := v.Number(); ok {
		return number.Coefficient + "e" + number.Exponent
	}
	if reference, ok := v.Reference(); ok {
		return "`" + reference.Package + "/" + reference.ID + "`"
	}
	if list, ok := v.Items(); ok {
		var out []string
		for _, item := range list {
			out = append(out, renderValue(item))
		}
		return "[" + strings.Join(out, ", ") + "]"
	}
	if fields, ok := v.Fields(); ok {
		var out []string
		for _, f := range fields {
			out = append(out, f.Name+": "+renderValue(f.Value))
		}
		return "{" + strings.Join(out, ", ") + "}"
	}
	return ""
}
