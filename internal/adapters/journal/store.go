package journal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/run"
	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE store_meta (singleton INTEGER PRIMARY KEY CHECK(singleton=1), format INTEGER NOT NULL CHECK(format=1), repository TEXT NOT NULL, repository_file TEXT NOT NULL, store_id TEXT NOT NULL, database_file TEXT NOT NULL, epoch INTEGER NOT NULL CHECK(epoch>=0));
CREATE TABLE stream_head (singleton INTEGER PRIMARY KEY CHECK(singleton=1), revision INTEGER NOT NULL CHECK(revision>=0));
CREATE TABLE blobs (digest BLOB PRIMARY KEY CHECK(length(digest)=32), size INTEGER NOT NULL CHECK(size>=0 AND size<=67108864), data BLOB NOT NULL CHECK(length(data)=size)) WITHOUT ROWID;
CREATE TABLE batches (id TEXT PRIMARY KEY, digest BLOB NOT NULL CHECK(length(digest)=32), previous INTEGER NOT NULL UNIQUE CHECK(previous>=0), revision INTEGER NOT NULL UNIQUE CHECK(revision>previous AND revision-previous<=256), data BLOB NOT NULL CHECK(length(data)>0 AND length(data)<=67108864));
CREATE TABLE events (id TEXT PRIMARY KEY, batch TEXT NOT NULL REFERENCES batches(id), revision INTEGER NOT NULL UNIQUE CHECK(revision>0), ordinal INTEGER NOT NULL CHECK(ordinal>=0 AND ordinal<256), UNIQUE(batch, ordinal));
INSERT INTO stream_head VALUES(1,0);`

func (s *session) openDatabase(ctx context.Context, initial bool) error {
	path := filepath.ToSlash(filepath.Join(s.registration.Directory, "journal.sqlite"))
	if runtime.GOOS == "windows" {
		path = "/" + path
	}
	u := url.URL{Scheme: "file", Path: path}
	q := url.Values{"mode": {"rw"}, "_txlock": {"immediate"}, "_pragma": {"busy_timeout(250)", "journal_mode(WAL)", "synchronous(FULL)", "foreign_keys(ON)", "trusted_schema(OFF)"}}
	u.RawQuery = q.Encode()
	var err error
	s.db, err = sql.Open("sqlite", u.String())
	if err != nil {
		return err
	}
	s.db.SetMaxOpenConns(1)
	s.db.SetMaxIdleConns(1)
	s.conn, err = s.db.Conn(ctx)
	if err != nil {
		return err
	}
	if err := s.settings(ctx); err != nil {
		return err
	}
	if initial {
		tx, err := s.conn.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if _, err = tx.ExecContext(ctx, schema); err != nil {
			return err
		}
		r := s.registration
		if _, err = tx.ExecContext(ctx, "INSERT INTO store_meta VALUES(1,1,?,?,?,?,0)", r.Repository, r.RepositoryFile, r.StoreID, r.DatabaseFile); err != nil {
			return err
		}
		if err = tx.Commit(); err != nil {
			return errors.Join(run.ErrUncertain, err)
		}
	} else {
		var result string
		if err := s.conn.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&result); err != nil || result != "ok" {
			return errors.Join(ErrCorrupt, err)
		}
	}
	return nil
}

func (s *session) settings(ctx context.Context) error {
	var mode string
	if err := s.conn.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil || mode != "wal" {
		return errors.Join(ErrCorrupt, err)
	}
	for pragma, want := range map[string]int{"synchronous": 2, "foreign_keys": 1, "busy_timeout": busyMilliseconds, "trusted_schema": 0} {
		var got int
		if err := s.conn.QueryRowContext(ctx, "PRAGMA "+pragma).Scan(&got); err != nil || got != want {
			return fmt.Errorf("%w: ineffective %s: %v", ErrCorrupt, pragma, err)
		}
	}
	return nil
}

func (s *session) meta(ctx context.Context, tx *sql.Tx, expected uint64) (uint64, error) {
	var version int
	var repository, repoFile, store, dbFile string
	var epoch int64
	err := tx.QueryRowContext(ctx, "SELECT format,repository,repository_file,store_id,database_file,epoch FROM store_meta WHERE singleton=1").Scan(&version, &repository, &repoFile, &store, &dbFile, &epoch)
	r := s.registration
	if err != nil || version != 1 || repository != r.Repository || repoFile != r.RepositoryFile || store != r.StoreID || dbFile != r.DatabaseFile || epoch < 0 {
		return 0, errors.Join(ErrCorrupt, err)
	}
	if expected != 0 && uint64(epoch) != expected {
		return uint64(epoch), apply.ErrControlBinding
	}
	return uint64(epoch), nil
}

func (s *session) acquireEpoch(ctx context.Context) error {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	epoch, err := s.meta(ctx, tx, 0)
	if err != nil {
		return err
	}
	if epoch == run.MaxRevision {
		return ErrReconcile
	}
	if _, err := tx.ExecContext(ctx, "UPDATE store_meta SET epoch=? WHERE singleton=1 AND epoch=?", epoch+1, epoch); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return errors.Join(run.ErrUncertain, err)
	}
	s.binding.Epoch = epoch + 1
	return nil
}

// BEGIN IMMEDIATE holds SQLite's writer lock from the epoch check through
// commit, including for reads. Filesystem identity checks detect replacement;
// they do not pretend to atomically lock arbitrary same-user filesystem edits.
func (s *session) transaction(parent context.Context, action func(context.Context, *sql.Tx) error) error {
	if parent == nil {
		return run.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(parent, operationTimeout)
	defer cancel()
	select {
	case s.gate <- struct{}{}:
		defer func() { <-s.gate }()
	case <-ctx.Done():
		return ctx.Err()
	}
	if s.closed {
		return apply.ErrControlClosed
	}
	if s.lost != nil {
		return s.lost
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.checkPaths(); err != nil {
		return s.poison(err)
	}
	if err := s.settings(ctx); err != nil {
		return s.poison(err)
	}
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := s.meta(ctx, tx, s.binding.Epoch); err != nil {
		return s.poison(err)
	}
	if err := action(ctx, tx); err != nil {
		return err
	}
	if err := s.checkPaths(); err != nil {
		return s.poison(err)
	}
	if err := tx.Commit(); err != nil {
		return errors.Join(run.ErrUncertain, s.poison(err))
	}
	if err := s.checkPaths(); err != nil {
		return errors.Join(run.ErrUncertain, s.poison(err))
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(run.ErrUncertain, err)
	}
	return nil
}

func head(ctx context.Context, tx *sql.Tx) (uint64, error) {
	var revision int64
	if err := tx.QueryRowContext(ctx, "SELECT revision FROM stream_head WHERE singleton=1").Scan(&revision); err != nil || revision < 0 {
		return 0, errors.Join(ErrCorrupt, err)
	}
	var last sql.NullInt64
	if err := tx.QueryRowContext(ctx, "SELECT MAX(revision) FROM batches").Scan(&last); err != nil {
		return 0, err
	}
	if last.Valid && last.Int64 != revision || !last.Valid && revision != 0 {
		return 0, ErrCorrupt
	}
	return uint64(revision), nil
}
