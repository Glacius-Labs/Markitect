package gitrepo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Retain already acquired external inputs in the existing candidate directory.
// They never become product files and are never reacquired from mutable paths.
type retainedInputs struct {
	models       []project.ModelSource
	instructions []project.SourceFile
}
type retainedEntry struct {
	Path string
	Mode snapshot.Mode
	Data []byte
}
type retainedModel struct {
	Boundary, Manifest, Root string
	Source                   model.SourceModel
	Files                    []retainedEntry
}
type retainedInstruction struct {
	Boundary, Path string
	File           retainedEntry
}
type retainedDocument struct {
	Version      int
	Models       []retainedModel
	Instructions []retainedInstruction
}

func retainedFrom(p project.Prepared) retainedInputs {
	raw := p.Attempt().Model().Inputs().Raw
	r := retainedInputs{}
	for _, m := range raw.Dependencies {
		if m.RepositoryArea.String() == "" {
			r.models = append(r.models, m)
		}
	}
	for _, f := range raw.Instructions {
		if f.RepositoryPath.String() == "" {
			r.instructions = append(r.instructions, f)
		}
	}
	return r
}
func (r retainedInputs) model(selection project.ModelSelection) (project.ModelSource, bool) {
	for _, m := range r.models {
		if m.Selection == selection {
			return m, true
		}
	}
	return project.ModelSource{}, false
}
func (r retainedInputs) instruction(location project.SourceLocation) (project.SourceFile, bool) {
	for _, f := range r.instructions {
		if f.Location == location {
			return f, true
		}
	}
	return project.SourceFile{}, false
}
func encodedEntry(e snapshot.Entry) retainedEntry {
	return retainedEntry{e.Path().String(), e.Mode(), e.Bytes()}
}
func (r retainedInputs) MarshalJSON() ([]byte, error) {
	d := retainedDocument{Version: 1}
	for _, m := range r.models {
		v := retainedModel{Boundary: m.Selection.Boundary, Manifest: m.Selection.ManifestPath.String(), Root: m.Root.String(), Source: m.Source}
		for _, f := range m.Files.Entries() {
			v.Files = append(v.Files, encodedEntry(f))
		}
		d.Models = append(d.Models, v)
	}
	for _, f := range r.instructions {
		d.Instructions = append(d.Instructions, retainedInstruction{f.Location.Boundary, f.Location.Path.String(), encodedEntry(f.File)})
	}
	return json.Marshal(d)
}
func decodedEntry(f retainedEntry) (snapshot.Entry, error) {
	p, err := snapshot.NewPath(f.Path)
	if err != nil {
		return snapshot.Entry{}, err
	}
	return snapshot.NewEntry(p, f.Mode, f.Data)
}
func (r *retainedInputs) UnmarshalJSON(data []byte) error {
	var d retainedDocument
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&d); err != nil {
		return err
	}
	if d.Version != 1 {
		return fmt.Errorf("unsupported retained input format")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("extra retained input data")
	}
	var result retainedInputs
	for _, m := range d.Models {
		manifest, err := snapshot.NewPath(m.Manifest)
		if err != nil {
			return err
		}
		root, err := directoryBoundary(m.Root)
		if err != nil {
			return err
		}
		var files []snapshot.Entry
		for _, f := range m.Files {
			e, err := decodedEntry(f)
			if err != nil {
				return err
			}
			files = append(files, e)
		}
		tree, err := snapshot.NewTree(files)
		if err != nil {
			return err
		}
		result.models = append(result.models, project.ModelSource{Selection: project.ModelSelection{Boundary: m.Boundary, ManifestPath: manifest}, Root: root, Source: m.Source, Files: tree})
	}
	for _, f := range d.Instructions {
		p, err := snapshot.NewPath(f.Path)
		if err != nil {
			return err
		}
		entry, err := decodedEntry(f.File)
		if err != nil {
			return err
		}
		result.instructions = append(result.instructions, project.SourceFile{Location: project.SourceLocation{Boundary: f.Boundary, Path: p}, File: entry})
	}
	*r = result
	return nil
}

func (g *LocalGit) retained(ctx context.Context, ref project.CandidateRef, p project.Prepared) (retainedInputs, error) {
	if err := ctx.Err(); err != nil {
		return retainedInputs{}, err
	}
	if ref.Binding != p.Binding() {
		return retainedInputs{}, fmt.Errorf("retained inputs require the exact fixed Project binding")
	}
	dir, err := g.candidatePath(ref)
	if err != nil {
		return retainedInputs{}, err
	}
	return g.retainedAt(ctx, dir, p)
}
func (g *LocalGit) inputLimit() (int64, error) {
	limit := g.MaxInputBytes
	if limit == 0 {
		limit = DefaultLimits().TotalBytes
	}
	if limit < 0 || limit > DefaultLimits().TotalBytes {
		return 0, ErrLimit
	}
	return limit, nil
}
func (g *LocalGit) retainedAt(ctx context.Context, dir string, p project.Prepared) (retainedInputs, error) {
	limit, err := g.inputLimit()
	if err != nil {
		return retainedInputs{}, err
	}
	f, err := os.Open(filepath.Join(dir, "inputs.json"))
	if err != nil {
		return retainedInputs{}, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return retainedInputs{}, err
	}
	if int64(len(data)) > limit {
		return retainedInputs{}, ErrLimit
	}
	expected, err := json.Marshal(retainedFrom(p))
	if err != nil {
		return retainedInputs{}, err
	}
	if !bytes.Equal(data, expected) {
		return retainedInputs{}, fmt.Errorf("stored external inputs differ from fixed Project binding")
	}
	var result retainedInputs
	err = json.Unmarshal(data, &result)
	return result, err
}
