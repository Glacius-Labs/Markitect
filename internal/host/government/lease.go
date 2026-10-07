package government

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"strings"
	"time"
	"unicode/utf8"
)

// Lease holds an OS exclusion lock while a Government writer is active. The
// durable record identifies the owner and fencing token; it is retained after
// Close so a later process can distinguish a known same-host lease from an
// unrecognized or foreign legacy lock.
type Lease struct {
	path  string
	token string
	file  *os.File
	owner leaseOwner
}

type leaseOwner struct {
	Version   int       `json:"version"`
	Host      string    `json:"host"`
	User      string    `json:"user"`
	PID       int       `json:"pid"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"createdAt"`
}

func currentLeaseUser() string {
	current, err := user.Current()
	if err != nil || current.Username == "" {
		return ""
	}
	return current.Username
}

// AcquireLease acquires an OS-held exclusion lock at path. If a prior valid
// record exists, obtaining the OS lock proves its process no longer holds the
// lease on this host. Records from another host or records without this format
// are refused. Filesystem lock semantics are provided by the host OS and are
// not an OS sandbox or a cross-volume transaction.
func AcquireLease(path string) (*Lease, error) {
	if path == "" {
		return nil, errors.New("lease path is required")
	}
	created := false
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err == nil {
		created = true
	} else if errors.Is(err, os.ErrExist) {
		file, err = os.OpenFile(path, os.O_RDWR, 0600)
	}
	if err != nil {
		return nil, fmt.Errorf("open lease record: %w", err)
	}
	failed := func(e error) (*Lease, error) { _ = unlockLeaseFile(file); _ = file.Close(); return nil, e }
	if err := lockLeaseFile(file); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("lease is held by another runner: %w", err)
	}
	if err := verifyLeasePath(file, path); err != nil {
		return failed(err)
	}
	data, err := readLeaseFile(file, 8192)
	if err != nil {
		return failed(fmt.Errorf("read lease record: %w", err))
	}
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		return failed(errors.New("cannot establish local host identity for lease"))
	}
	user := currentLeaseUser()
	if user == "" {
		return failed(errors.New("cannot establish local user identity for lease"))
	}
	if !created {
		previous, err := decodeLeaseOwner(data)
		if err != nil {
			return failed(fmt.Errorf("exclusive repository promotion lock has an unrecognized record; refusing to take it over: %w", err))
		}
		if previous.Host != host || previous.User != user {
			return failed(fmt.Errorf("existing lease belongs to foreign host or user %q/%q", previous.Host, previous.User))
		}
	}
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return failed(fmt.Errorf("create lease fencing token: %w", err))
	}
	owner := leaseOwner{Version: 1, Host: host, User: user, PID: os.Getpid(), Token: hex.EncodeToString(bytes[:]), CreatedAt: time.Now().UTC()}
	encoded, err := json.Marshal(owner)
	if err != nil {
		return failed(err)
	}
	if _, err := decodeLeaseOwner(encoded); err != nil {
		return failed(fmt.Errorf("invalid local lease owner: %w", err))
	}
	if err := file.Truncate(0); err != nil {
		return failed(fmt.Errorf("truncate lease record: %w", err))
	}
	if _, err := file.Seek(0, 0); err != nil {
		return failed(fmt.Errorf("seek lease record: %w", err))
	}
	if _, err := file.Write(append(encoded, '\n')); err != nil {
		return failed(fmt.Errorf("write lease record: %w", err))
	}
	if err := file.Sync(); err != nil {
		return failed(fmt.Errorf("sync lease record: %w", err))
	}
	if err := verifyLeasePath(file, path); err != nil {
		return failed(err)
	}
	return &Lease{path: path, token: owner.Token, file: file, owner: owner}, nil
}

// Token returns the unique fencing token recorded in this held lease.
func (l *Lease) Token() string {
	if l == nil {
		return ""
	}
	return l.token
}

// Verify confirms the OS handle remains open and the durable token still
// matches. Callers use it immediately before every protected side effect.
func (l *Lease) Verify() error {
	if l == nil || l.file == nil {
		return errors.New("lease is not held")
	}
	if err := verifyLeaseFileLock(l.file); err != nil {
		return fmt.Errorf("lease OS exclusion is not held: %w", err)
	}
	if err := verifyLeasePath(l.file, l.path); err != nil {
		return err
	}
	readFile, err := os.Open(l.path)
	if err != nil {
		return err
	}
	defer readFile.Close()
	readInfo, err := readFile.Stat()
	if err != nil {
		return err
	}
	leaseInfo, err := l.file.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(readInfo, leaseInfo) {
		return errors.New("lease path was replaced during verification")
	}
	data, err := readLeaseFile(readFile, 8192)
	if err != nil {
		return err
	}
	current, err := decodeLeaseOwner(data)
	if err != nil {
		return fmt.Errorf("decode lease owner record: %w", err)
	}
	if current.Version != l.owner.Version || current.Host != l.owner.Host || current.User != l.owner.User || current.PID != l.owner.PID || current.Token != l.token || !current.CreatedAt.Equal(l.owner.CreatedAt) {
		return errors.New("lease fencing record changed")
	}
	return nil
}

// decodeLeaseOwner recognizes only the complete format written by this version.
// Field recognition is exact: encoding/json's aliases and duplicate-key
// replacement must not turn an unrecognized record into takeover authority.
func decodeLeaseOwner(data []byte) (leaseOwner, error) {
	var owner leaseOwner
	if len(data) > 8192 || !utf8.Valid(data) {
		return owner, errors.New("lease record must be bounded UTF-8 JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	start, err := decoder.Token()
	if err != nil || start != json.Delim('{') {
		return owner, errors.New("lease record must be a JSON object")
	}
	fields := map[string]json.RawMessage{}
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return owner, err
		}
		name, ok := key.(string)
		if !ok {
			return owner, errors.New("lease field name must be a string")
		}
		switch name {
		case "version", "host", "user", "pid", "token", "createdAt":
		default:
			return owner, fmt.Errorf("unrecognized lease field %q", name)
		}
		if _, exists := fields[name]; exists {
			return owner, fmt.Errorf("duplicate lease field %q", name)
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return owner, err
		}
		fields[name] = value
	}
	end, err := decoder.Token()
	if err != nil || end != json.Delim('}') || len(fields) != 6 {
		return owner, errors.New("lease record requires all six supported fields")
	}
	if _, err := decoder.Token(); err != io.EOF {
		return owner, errors.New("lease record contains trailing data")
	}
	if err := json.Unmarshal(data, &owner); err != nil {
		return owner, err
	}
	validIdentity := func(value string) bool {
		return value != "" && strings.TrimSpace(value) == value && !strings.ContainsAny(value, "\x00\r\n")
	}
	if owner.Version != 1 || !validIdentity(owner.Host) || !validIdentity(owner.User) || owner.PID <= 0 {
		return owner, errors.New("lease version, host, user or PID is invalid")
	}
	token, err := hex.DecodeString(owner.Token)
	if err != nil || len(token) != 32 || strings.ToLower(owner.Token) != owner.Token {
		return owner, errors.New("lease token must be 64 lowercase hexadecimal characters")
	}
	var stamp string
	if err := json.Unmarshal(fields["createdAt"], &stamp); err != nil || owner.CreatedAt.IsZero() || owner.CreatedAt.Year() < 1 || stamp != owner.CreatedAt.UTC().Format(time.RFC3339Nano) {
		return owner, errors.New("lease createdAt must be a nonzero canonical UTC timestamp")
	}
	return owner, nil
}

func readLeaseFile(file *os.File, maxBytes int64) ([]byte, error) {
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, errors.New("lease record exceeds size limit")
	}
	return data, nil
}

func verifyLeasePath(file *os.File, path string) error {
	linkInfo, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if promotionIsReparsePoint(linkInfo) {
		return errors.New("lease path is or became a reparse point")
	}
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(fileInfo, linkInfo) {
		return errors.New("lease path was replaced while the lease was held")
	}
	return nil
}

// Close releases the OS lock and retains the durable owner/token record.
func (l *Lease) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	file := l.file
	l.file = nil
	err := unlockLeaseFile(file)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}
