package recordstore

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
)

type Outcome string

const (
	Published Outcome = "published-unobserved"
	Unknown   Outcome = "publication-unknown"
)

type PublicationError struct {
	Path    string
	Outcome Outcome
	Cause   error
}

func (e *PublicationError) Error() string {
	return fmt.Sprintf("record %s: %s: %v", e.Path, e.Outcome, e.Cause)
}
func (e *PublicationError) Unwrap() error { return e.Cause }
func WasPublished(err error) bool {
	var e *PublicationError
	return errors.As(err, &e) && e.Outcome == Published
}

// WasPublishedAt distinguishes a published record from a created ancestor or
// control leaf whose later observation failed.
func WasPublishedAt(err error, relative string) bool {
	var e *PublicationError
	return errors.As(err, &e) && e.Outcome == Published && e.Path == relative
}
func OutcomeUnknown(err error) bool {
	var e *PublicationError
	return errors.As(err, &e) && e.Outcome == Unknown
}

var errAmbiguous = errors.New("native publication outcome is unknown")

func randomLeaf() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}
	return ".pending-" + hex.EncodeToString(id[:]), nil
}
func removeOwned(d *directory, name string, identity os.FileInfo, expected ...[]byte) error {
	if err := d.check(); err != nil {
		return err
	}
	current, err := d.root.Lstat(name)
	if errors.Is(err, fs.ErrNotExist) {
		return d.confirmMissingName(name)
	}
	if err != nil {
		return err
	}
	if platformIsReparse(current) || !os.SameFile(current, identity) {
		return errors.New("cleanup retains replaced record leaf")
	}
	if len(expected) > 0 {
		file, err := readLeaf(d, name, int64(len(expected[0]))+1)
		if err != nil {
			return err
		}
		if !file.Exists || !bytes.Equal(file.Bytes, expected[0]) {
			return errors.New("cleanup retains changed record bytes")
		}
	}
	return platformRemoveOwned(d, name, identity)
}
func (s *Store) Create(n Name, data []byte) error { return s.publish(n, nil, data, nil) }
func (s *Store) Replace(n Name, expected File, data []byte, validate func() error) error {
	return s.publish(n, &expected, data, validate)
}
func (s *Store) publish(n Name, expected *File, data []byte, validate func() error) (result error) {
	p, err := n.Path()
	if err != nil {
		return err
	}
	if err := s.CheckWrite(n.Kind); err != nil {
		return err
	}
	d, err := s.openDirectory(path.Dir(p), true, true)
	if err != nil {
		return err
	}
	defer d.close()
	parent, err := platformWritableParent(d.file)
	if err != nil {
		return err
	}
	defer parent.Close()
	pi, err := parent.Stat()
	di, e := d.file.Stat()
	if err != nil || e != nil || !os.SameFile(pi, di) {
		return errors.New("publication parent changed while opening writer handle")
	}
	leaf := path.Base(p)
	prior, err := readLeaf(d, leaf, 0)
	if err != nil {
		return err
	}
	if expected == nil && prior.Exists {
		return fs.ErrExist
	}
	if expected != nil && !equalPreimage(prior, *expected) {
		return errors.New("record byte preimage changed")
	}
	tmp, err := randomLeaf()
	if err != nil {
		return err
	}
	if n.Kind == Brownfield {
		tmp = ".record.json.tmp"
	}
	if s.hooks.beforeTempCreate != nil {
		if err := s.hooks.beforeTempCreate(); err != nil {
			return err
		}
	}
	if err := d.check(); err != nil {
		return err
	}
	f, err := platformCreate(d.root, parent, tmp)
	if err != nil {
		return err
	}
	identity, err := f.Stat()
	if err != nil {
		return &PublicationError{Path: path.Dir(p) + "/" + tmp, Outcome: Published, Cause: errors.Join(err, f.Close())}
	}
	published := false
	ambiguous := false
	observedSuccess := false
	defer func() {
		closeErr := f.Close()
		if closeErr != nil {
			if published {
				result = errors.Join(result, &PublicationError{p, Published, closeErr})
			} else {
				result = errors.Join(result, closeErr)
			}
		}
		if !published && !ambiguous {
			result = errors.Join(result, removeOwned(d, tmp, identity, data))
		}
		if observedSuccess && closeErr == nil {
			var finalErr error
			if s.hooks.afterPublicationClose != nil {
				finalErr = s.hooks.afterPublicationClose()
			}
			finalErr = errors.Join(finalErr, d.check())
			if finalErr != nil {
				result = errors.Join(result, &PublicationError{p, Published, finalErr})
			}
		}
	}()
	if err = f.Chmod(0600); err != nil {
		return err
	}
	if _, err = f.Write(data); err != nil {
		return err
	}
	if s.hooks.fileSync != nil {
		if err = s.hooks.fileSync(); err != nil {
			return err
		}
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if s.hooks.beforePublish != nil {
		if err = s.hooks.beforePublish(); err != nil {
			return err
		}
	}
	if err = s.CheckCutover(n.Kind); err != nil {
		return err
	}
	if err = d.check(); err != nil {
		return err
	}
	current, err := readLeaf(d, leaf, 0)
	if err != nil {
		return err
	}
	if !equalPreimage(current, prior) {
		return errors.New("record changed before publication")
	}
	if validate != nil {
		if err = validate(); err != nil {
			return err
		}
	}
	if err = d.check(); err != nil {
		return err
	}
	temporary, err := readLeaf(d, tmp, int64(len(data))+1)
	if err != nil {
		return err
	}
	held, err := f.Stat()
	if err != nil || !temporary.Exists || !os.SameFile(identity, held) || !os.SameFile(identity, temporary.identity) || !held.Mode().IsRegular() || platformIsReparse(held) || !bytes.Equal(temporary.Bytes, data) {
		return errors.New("written temporary record identity or bytes changed")
	}
	if err := platformMode(held, 0600); err != nil {
		return err
	}
	published, err = platformPublish(d.root, parent, f, tmp, leaf, expected != nil && prior.Exists)
	if err != nil {
		ambiguous = errors.Is(err, errAmbiguous)
		if published {
			return &PublicationError{p, Published, err}
		}
		if ambiguous {
			return &PublicationError{p, Unknown, err}
		}
		return err
	}
	if s.hooks.afterPublish != nil {
		if err = s.hooks.afterPublish(); err != nil {
			return &PublicationError{p, Published, err}
		}
	}
	if s.hooks.directorySync != nil {
		if err = s.hooks.directorySync(); err != nil {
			return &PublicationError{p, Published, err}
		}
	}
	if err = platformSyncDir(parent); err != nil {
		return &PublicationError{p, Published, err}
	}
	if err = d.check(); err != nil {
		return &PublicationError{p, Published, err}
	}
	observed, err := readLeaf(d, leaf, int64(len(data))+1)
	if err != nil || !observed.Exists || observed.identity == nil || !os.SameFile(identity, observed.identity) || !bytes.Equal(observed.Bytes, data) {
		if err == nil {
			err = errors.New("published record readback identity or bytes differ")
		}
		return &PublicationError{p, Published, err}
	}
	observedSuccess = true
	return nil
}

// Incomplete checks fixed Brownfield publication remnants without repairing them.
func (s *Store) Incomplete(n Name) error {
	p, err := n.Path()
	if err != nil {
		return err
	}
	d, err := s.openDirectory(path.Dir(p), false, true)
	if isDirectoryAbsent(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer d.close()
	for _, suffix := range []string{".tmp", ".previous"} {
		if _, err := d.root.Lstat("." + path.Base(p) + suffix); err == nil {
			return fmt.Errorf("record publication has an unreconciled %s file", suffix)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return d.check()
}
