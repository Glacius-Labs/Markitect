// Package runstore stores closed Run records and independently appended Notes.
package runstore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

const maxBytes = 16 << 20

type Store struct{ root string }

func New(root string) (*Store, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return nil, fmt.Errorf("state root must be an explicit absolute directory")
	}
	return &Store{root}, nil
}
func valid(s string) bool {
	if s == "" || s == "." || s == ".." || len(s) > 120 {
		return false
	}
	for _, r := range s {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_.", r) {
			return false
		}
	}
	return true
}
func (s *Store) path(project, kind, id string) (string, error) {
	if s == nil || !valid(project) || !valid(id) {
		return "", fmt.Errorf("invalid storage identity")
	}
	return filepath.Join(s.root, project, kind, id+".json"), nil
}

// write publishes one complete immutable record with create-if-absent semantics.
// Unique Notes never overwrite another process; readers never see temp records.
func write(ctx context.Context, path string, value any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(data) > maxBytes {
		return fmt.Errorf("record byte limit exceeded")
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".pending-")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	// A hard link publishes the fully flushed inode atomically and refuses an
	// existing name on Windows and Unix. No replace/last-writer-wins behavior.
	return os.Link(tmp, path)
}
func read(ctx context.Context, path string, value any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > maxBytes {
		return fmt.Errorf("invalid stored record")
	}
	d := json.NewDecoder(io.LimitReader(f, maxBytes+1))
	d.DisallowUnknownFields()
	if err = d.Decode(value); err != nil {
		return err
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return fmt.Errorf("extra stored record data")
	}
	return ctx.Err()
}
func (s *Store) Save(ctx context.Context, r run.Result) error {
	p, err := s.path(r.Project, "runs", r.ID)
	if err != nil {
		return err
	}
	return write(ctx, p, r)
}
func (s *Store) Read(ctx context.Context, project, id string) (run.Result, error) {
	p, err := s.path(project, "runs", id)
	if err != nil {
		return run.Result{}, err
	}
	var r run.Result
	err = read(ctx, p, &r)
	if err == nil && (r.ID != id || r.Project != project) {
		err = fmt.Errorf("record identity mismatch")
	}
	return r, err
}
func (s *Store) Append(ctx context.Context, n run.Note) (run.Note, error) {
	if !valid(n.Run) || n.Manager == "" || n.Text == "" || n.ID != "" {
		return run.Note{}, fmt.Errorf("invalid Note provenance or content")
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return run.Note{}, err
	}
	n.ID = hex.EncodeToString(id[:])
	p, err := s.path(n.Project, "notes", n.ID)
	if err == nil {
		err = write(ctx, p, n)
	}
	return n, err
}
func (s *Store) Notes(ctx context.Context, project string) ([]run.Note, error) {
	p, err := s.path(project, "notes", "unused")
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Dir(p))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(entries) > 10000 {
		return nil, fmt.Errorf("Note count exceeds finite read limit")
	}
	var notes []run.Note
	total := 0
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		var n run.Note
		if err = read(ctx, filepath.Join(filepath.Dir(p), e.Name()), &n); err != nil {
			return nil, err
		}
		if n.Project != project || e.Name() != n.ID+".json" {
			return nil, fmt.Errorf("Note identity mismatch")
		}
		total += len(n.Text) + len(n.Context)
		if total > maxBytes {
			return nil, fmt.Errorf("Note byte limit exceeded")
		}
		notes = append(notes, n)
	}
	sort.Slice(notes, func(i, j int) bool { return notes[i].ID < notes[j].ID })
	return notes, nil
}
