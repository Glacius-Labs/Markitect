package guardedwrite

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/recordstore"
)

// ApplyRecordChecked preserves the sealed Git/source authorization and shared
// Host lease for one exploration record, while recordstore owns publication.
func ApplyRecordChecked(root string, capture *Capture, name recordstore.Name, data []byte, validate func() error) (Result, error) {
	result := Result{CompletedPaths: []string{}}
	if name.Kind != recordstore.Exploration || name.Role != "record.json" {
		return result, errors.New("guarded record Apply permits one exploration record")
	}
	p, err := name.Path()
	if err != nil {
		return result, err
	}
	if capture == nil || capture.rootIdentity == nil || capture.seal == "" {
		return result, errors.New("guarded record requires sealed capture")
	}
	originalSeal := capture.seal
	seal, err := guardedCaptureSeal(capture)
	if err != nil || seal != originalSeal {
		return result, errors.New("guarded record capture changed")
	}
	before, ok := capture.Files[p]
	if !ok {
		return result, errors.New("record target was not captured")
	}
	store, err := recordstore.Open(root)
	if err != nil {
		return result, err
	}
	defer store.Close()
	if !os.SameFile(capture.rootIdentity, store.Identity()) {
		return result, errors.New("record root differs from capture")
	}
	if err = store.CheckWrite(name.Kind); err != nil {
		return result, err
	}
	writer, err := OpenRoot(root)
	if err != nil {
		return result, err
	}
	defer writer.Close()
	check := func() error {
		seal, err := guardedCaptureSeal(capture)
		if err != nil || seal != originalSeal {
			return errors.New("guarded record capture changed")
		}
		if err = writer.CheckIdentity(); err != nil {
			return err
		}
		if !os.SameFile(capture.rootIdentity, writer.identity) {
			return errors.New("guarded record root identity changed")
		}
		if err = ensureGuardedGitState(root, capture.Identity, capture.Branch, capture.Head); err != nil {
			return err
		}
		return compareGuardedFiles(writer, capture.Files)
	}
	if err = check(); err != nil {
		return result, err
	}
	unlock, err := writer.lockWriterAt(".markitect", "write.lock")
	if err != nil {
		return result, err
	}
	defer unlock()
	if err = check(); err != nil {
		return result, err
	}
	expected, err := store.Read(name, 0)
	if err != nil {
		return result, err
	}
	if expected.Exists != before.Exists || before.Exists && (!bytes.Equal(expected.Bytes, before.Bytes) || expected.Mode != before.Mode) {
		return result, errors.New("record differs from original sealed capture")
	}
	err = store.Replace(name, expected, data, func() error {
		if err := check(); err != nil {
			return err
		}
		if validate != nil {
			if err := validate(); err != nil {
				return err
			}
		}
		return check()
	})
	if err == nil || recordstore.WasPublished(err) {
		result.CompletedPaths = append(result.CompletedPaths, p)
	}
	if err != nil {
		return result, fmt.Errorf("guarded record publication: %w", err)
	}
	expectedFiles := cloneGuardedWriteFiles(capture.Files)
	written, err := readGuardedWriteFile(writer, p)
	if err == nil {
		expectedFiles[p] = written
		err = compareGuardedFiles(writer, expectedFiles)
	}
	if err == nil {
		err = ensureGuardedGitState(root, capture.Identity, capture.Branch, capture.Head)
	}
	if err != nil {
		return result, &recordstore.PublicationError{Path: p, Outcome: recordstore.Published, Cause: err}
	}
	return result, nil
}
