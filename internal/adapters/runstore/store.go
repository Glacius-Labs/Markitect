// Package runstore stores closed Run records, an index of completed Verify
// approvals and independently appended Notes.
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
	"time"

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

// Save stores r. A completed Verify approval is also indexed under what it
// approved, so a later command finds it without reading every stored Run. The
// index entry comes first: one without its record names nothing readable.
func (s *Store) Save(ctx context.Context, r run.Result) error {
	p, err := s.path(r.Project, "runs", r.ID)
	if err != nil {
		return err
	}
	if key, ok := r.Approval(); ok {
		index, err := s.path(r.Project, filepath.Join("approvals", key), r.ID)
		if err == nil {
			err = write(ctx, index, r.ID)
		}
		if err != nil {
			return err
		}
	}
	return write(ctx, p, r)
}

// Approvals reads the stored results indexed under one approval key. A record
// that cannot be read is no evidence and is left out; the caller validates
// every returned result itself.
func (s *Store) Approvals(ctx context.Context, project, key string) ([]run.Result, error) {
	if !valid(key) {
		return nil, fmt.Errorf("invalid approval key")
	}
	p, err := s.path(project, filepath.Join("approvals", key), "unused")
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
	var out []run.Result
	for _, e := range entries {
		id, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || !valid(id) {
			continue
		}
		r, err := s.Read(ctx, project, id)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if err == nil {
			out = append(out, r)
		}
	}
	return out, nil
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

// Append stores one Note under an ID that starts with its creation time, so
// the files sort in creation order without any lock between processes. Notes
// beyond the size limits are refused with the reason: a stored Note must always
// fit into a page the tool transport can carry.
func (s *Store) Append(ctx context.Context, n run.Note) (run.Note, error) {
	if !valid(n.Run) || n.Manager == "" || n.Text == "" || n.ID != "" {
		return run.Note{}, fmt.Errorf("invalid Note provenance or content")
	}
	if len(n.Text) > run.MaxNoteText {
		return run.Note{}, fmt.Errorf("Note text is %d bytes, above the limit of %d; shorten it or split it into several Notes", len(n.Text), run.MaxNoteText)
	}
	if len(n.Context) > run.MaxNoteContext {
		return run.Note{}, fmt.Errorf("Note context is %d bytes, above the limit of %d; name the affected area more briefly", len(n.Context), run.MaxNoteContext)
	}
	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return run.Note{}, err
	}
	n.ID = fmt.Sprintf("%020d-%s", time.Now().UTC().UnixNano(), hex.EncodeToString(suffix[:]))
	p, err := s.path(n.Project, "notes", n.ID)
	if err == nil {
		err = write(ctx, p, n)
	}
	return n, err
}

// Notes returns the page of project Notes that follows the cursor after (empty
// for the first page), ordered by ID. The page ends before the Note that would
// exceed the byte or count limit; Next then names the last Note of the page.
// Only the files of the page are read.
func (s *Store) Notes(ctx context.Context, project, after string) (run.NotePage, error) {
	page := run.NotePage{Notes: []run.Note{}}
	if after != "" && !valid(after) {
		return page, fmt.Errorf("invalid cursor %q; pass the next value of the previous page, or omit it for the first page", after)
	}
	p, err := s.path(project, "notes", "unused")
	if err != nil {
		return page, err
	}
	dir := filepath.Dir(p)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return page, nil
	}
	if err != nil {
		return page, err
	}
	var ids []string
	for _, e := range entries {
		if id, ok := strings.CutSuffix(e.Name(), ".json"); ok && id > after {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	size := 0
	for _, id := range ids {
		var n run.Note
		if err = read(ctx, filepath.Join(dir, id+".json"), &n); err != nil {
			if ctx.Err() != nil {
				return page, err
			}
			return page, fmt.Errorf("stored Note %s is unreadable (%w); continue after it with cursor %q", id, err, id)
		}
		if n.Project != project || n.ID != id {
			return page, fmt.Errorf("stored Note %s does not match its identity; continue after it with cursor %q", id, id)
		}
		encoded, err := json.Marshal(n)
		if err != nil {
			return page, err
		}
		if len(page.Notes) == 0 && len(encoded) > run.NotePageBytes {
			return page, fmt.Errorf("stored Note %s is %d bytes, more than a page can carry; continue after it with cursor %q", id, len(encoded), id)
		}
		if len(page.Notes) == run.NotePageCount || size+len(encoded) > run.NotePageBytes {
			page.Next = page.Notes[len(page.Notes)-1].ID
			return page, nil
		}
		size += len(encoded)
		page.Notes = append(page.Notes, n)
	}
	return page, nil
}
