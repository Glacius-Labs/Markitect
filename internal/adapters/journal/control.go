// Package journal implements the physical SQLite journal and repository control.
package journal

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

var (
	ErrReconcile = errors.New("journal control metadata or registered store requires reconciliation")
	ErrCorrupt   = errors.New("journal store is corrupt or has an unsupported format")
	ErrUnsafe    = errors.New("unsafe journal control or state path")
)

const operationTimeout = 5 * time.Second
const busyMilliseconds = 250

// Control has no mutable configuration. StateDirectory must name a dedicated,
// private directory outside the repository; the first acquisition creates it.
type Control struct{}

func New() *Control { return &Control{} }

var _ apply.Control = (*Control)(nil)

type session struct {
	gate              chan struct{}
	binding           apply.ControlBinding
	registration      registration
	registrationBytes []byte
	paths             map[string]string
	lock              *os.File
	db                *sql.DB
	conn              *sql.Conn
	closed            bool
	lost              error
	closeErr          error
}

func (s *session) Binding() apply.ControlBinding { return s.binding }
func (s *session) Journal() run.Journal          { return s }

func (c *Control) Acquire(ctx context.Context, req apply.ControlRequest) (_ apply.ControlSession, err error) {
	if ctx == nil || !validID(req.RepositoryIdentity) || req.Revision == "" || req.Guard == "" || req.Repository == "" || req.StateDirectory == "" {
		return nil, run.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	repo, err := safeAbsolute(req.Repository)
	if err != nil {
		return nil, err
	}
	state, err := safeAbsolute(req.StateDirectory)
	if err != nil {
		return nil, err
	}
	if inside(repo, state) || inside(state, repo) {
		return nil, ErrUnsafe
	}
	// Only an actual checkout root with existing canonical metadata is accepted.
	if _, err := os.Lstat(filepath.Join(repo, ".git")); err != nil {
		return nil, ErrUnsafe
	}
	meta := filepath.Join(repo, ".markitect")
	if err := directory(meta, false); err != nil {
		return nil, err
	}
	control := filepath.Join(meta, "state")
	if err := directory(control, true); err != nil {
		return nil, err
	}
	s := &session{paths: map[string]string{}, gate: make(chan struct{}, 1)}
	defer func() {
		if err != nil {
			err = errors.Join(err, s.Close())
		}
	}()
	regPath, lockPath := filepath.Join(control, "journal.json"), filepath.Join(control, "journal.lock")
	_, regErr := os.Lstat(regPath)
	if regErr != nil && !errors.Is(regErr, os.ErrNotExist) {
		return nil, regErr
	}
	s.lock, err = openLock(lockPath, errors.Is(regErr, os.ErrNotExist))
	if err != nil {
		return nil, err
	}
	if err = lockExclusive(s.lock); err != nil {
		return nil, errors.Join(apply.ErrControlUnavailable, err)
	}
	lockID, err := physicalFileID(s.lock)
	if err != nil {
		return nil, err
	}
	if err = samePhysical(lockPath, lockID); err != nil {
		return nil, err
	}
	for _, path := range []string{repo, meta, control, lockPath} {
		if err = s.remember(path); err != nil {
			return nil, err
		}
	}
	s.paths[lockPath] = lockID
	repoID := s.paths[repo]
	if _, err = os.Lstat(regPath); errors.Is(err, os.ErrNotExist) {
		if err = directory(state, true); err != nil {
			return nil, err
		}
		entries, readErr := os.ReadDir(state)
		if readErr != nil {
			return nil, readErr
		}
		if len(entries) != 0 {
			return nil, ErrReconcile
		}
		var random [32]byte
		if _, err = rand.Read(random[:]); err != nil {
			return nil, err
		}
		s.registration = registration{Version: 1, Repository: req.RepositoryIdentity, RepositoryPath: repo, RepositoryFile: repoID,
			StoreID: hex.EncodeToString(random[:]), Directory: state, Status: "initializing"}
		// Record intent BEFORE creating a database. An interrupted initialization
		// is visible and cannot be mistaken for a genuinely new empty store.
		if err = createRegistration(regPath, s.registration); err != nil {
			return nil, err
		}
		dbPath := filepath.Join(state, "journal.sqlite")
		f, createErr := os.OpenFile(dbPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
		if createErr != nil {
			return nil, createErr
		}
		err = errors.Join(f.Sync(), f.Close())
		if err != nil {
			return nil, err
		}
		if err = private(dbPath); err != nil {
			return nil, err
		}
		s.registration.DatabaseFile, err = physicalID(dbPath)
		if err != nil {
			return nil, err
		}
		s.registration.DirectoryFile, err = physicalID(state)
		if err != nil {
			return nil, err
		}
		if err = s.openDatabase(ctx, true); err != nil {
			return nil, err
		}
		if err = syncDirectory(state); err != nil {
			return nil, err
		}
		s.registration.Status = "ready"
		if err = finishRegistration(regPath, s.registration); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		if err = private(regPath); err != nil {
			return nil, err
		}
		s.registration, _, err = readRegistration(regPath)
		if err != nil {
			return nil, err
		}
		r := s.registration
		if r.Status != "ready" || r.Repository != req.RepositoryIdentity || r.RepositoryPath != repo || r.RepositoryFile != repoID || r.Directory != state {
			return nil, ErrReconcile
		}
		if err = private(state); err != nil {
			return nil, err
		}
		if err = samePhysical(state, r.DirectoryFile); err != nil {
			return nil, err
		}
		if err = samePhysical(filepath.Join(state, "journal.sqlite"), r.DatabaseFile); err != nil {
			return nil, err
		}
		if err = s.openDatabase(ctx, false); err != nil {
			return nil, err
		}
	}
	s.registration, s.registrationBytes, err = readRegistration(regPath)
	if err != nil {
		return nil, err
	}
	for _, path := range []string{regPath, state, filepath.Join(state, "journal.sqlite")} {
		if err = s.remember(path); err != nil {
			return nil, err
		}
	}
	s.binding = apply.ControlBinding{Repository: req.RepositoryIdentity, StoreID: s.registration.StoreID}
	if err = s.acquireEpoch(ctx); err != nil {
		return nil, err
	}
	if err = s.checkPaths(); err != nil {
		return nil, err
	}
	return s, nil
}

func validID(s string) bool {
	return len(s) <= 256 && utf8.ValidString(s) && strings.TrimSpace(s) != "" && !strings.ContainsRune(s, 0)
}

func (s *session) Check(ctx context.Context) error {
	return s.transaction(ctx, func(context.Context, *sql.Tx) error { return nil })
}

func (s *session) Close() error {
	s.gate <- struct{}{}
	defer func() { <-s.gate }()
	if s.closed {
		return s.closeErr
	}
	s.closed = true
	if s.conn != nil {
		s.closeErr = errors.Join(s.closeErr, s.conn.Close())
	}
	if s.db != nil {
		s.closeErr = errors.Join(s.closeErr, s.db.Close())
	}
	if s.lock != nil {
		s.closeErr = errors.Join(s.closeErr, s.lock.Close())
	}
	return s.closeErr
}

func (s *session) remember(path string) error {
	id, err := physicalID(path)
	if err == nil {
		s.paths[path] = id
	}
	return err
}

func (s *session) checkPaths() error {
	for path, id := range s.paths {
		if err := samePhysical(path, id); err != nil {
			return errors.Join(apply.ErrControlBinding, err)
		}
	}
	control := filepath.Join(s.registration.RepositoryPath, ".markitect", "state")
	for _, path := range []string{s.registration.Directory, control, filepath.Join(control, "journal.lock"), filepath.Join(control, "journal.json")} {
		if err := private(path); err != nil {
			return err
		}
	}
	_, data, err := readRegistration(filepath.Join(control, "journal.json"))
	if err != nil || string(data) != string(s.registrationBytes) {
		return errors.Join(apply.ErrControlBinding, err)
	}
	// SQLite sidecars are private regular, single-link files if present.
	for _, suffix := range []string{"", "-wal", "-shm"} {
		path := filepath.Join(s.registration.Directory, "journal.sqlite"+suffix)
		if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) && suffix != "" {
			continue
		}
		if err := private(path); err != nil {
			return err
		}
	}
	return nil
}

func (s *session) poison(err error) error {
	if err != nil {
		s.lost = errors.Join(apply.ErrControlBinding, err)
	}
	return s.lost
}

func samePhysical(path, want string) error {
	got, err := physicalID(path)
	if err != nil || got != want {
		return fmt.Errorf("%w: physical identity changed: %s: %v", ErrReconcile, path, err)
	}
	return nil
}
