package gitrepo

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var _ run.AssignmentCommitReader = (*Workspaces)(nil)

// ReadAssignmentCommit reads Git objects, never the live working tree or an
// attribute-sensitive archive. Every helper output is bounded. It does not
// commit, check out, publish or alter the source's index or files.
func (w *Workspaces) ReadAssignmentCommit(ctx context.Context, request run.AssignmentCommitRequest) (snapshot.Tree, error) {
	if !commitIDPattern.MatchString(request.Commit) || request.MaxEntries < 1 || request.MaxEntries > 1<<20 || request.MaxBytes < 1 || request.MaxBytes > 1<<32 {
		return snapshot.Tree{}, errors.New("named Git read needs a full commit identity and bounded entries and bytes")
	}
	a, err := w.assignment(request.Source)
	if err != nil {
		return snapshot.Tree{}, err
	}
	resolved, err := w.assignmentOutput(ctx, a, nil, 128, "rev-parse", "--verify", request.Commit+"^{commit}")
	if err != nil || strings.TrimSpace(string(resolved)) != request.Commit {
		return snapshot.Tree{}, errors.Join(err, errors.New("named Git source is not the exact full commit identity"))
	}
	listed, err := w.assignmentOutput(ctx, a, nil, 64<<20, "ls-tree", "-r", "-z", "--full-tree", "--long", request.Commit, "--")
	if err != nil {
		return snapshot.Tree{}, err
	}
	if len(listed) > 0 && listed[len(listed)-1] != 0 {
		return snapshot.Tree{}, errors.New("named Git tree output is incomplete")
	}
	var entries []gitEntry
	var total int64
	var input strings.Builder
	for _, record := range bytes.Split(listed, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		header, name, ok := bytes.Cut(record, []byte{'\t'})
		fields := strings.Fields(string(header))
		if !ok || len(fields) != 4 {
			return snapshot.Tree{}, errors.New("malformed named Git tree record")
		}
		path := string(name)
		if workspaceExcluded(path, false) {
			continue
		}
		if _, err := snapshot.NewPath(path); err != nil {
			return snapshot.Tree{}, err
		}
		if fields[1] != "blob" || (fields[0] != "100644" && fields[0] != "100755") || !commitIDPattern.MatchString(fields[2]) {
			return snapshot.Tree{}, fmt.Errorf("unsupported Git entry at %q", path)
		}
		size, err := strconv.ParseInt(fields[3], 10, 64)
		if err != nil || size < 0 || size > request.MaxBytes-total || len(entries) >= request.MaxEntries {
			return snapshot.Tree{}, ErrLimit
		}
		total += size
		entries = append(entries, gitEntry{path: path, mode: fields[0], kind: fields[1], oid: fields[2], size: size})
		input.WriteString(fields[2] + "\n")
	}
	limit := total + int64(len(entries))*128
	if uint64(limit) > uint64(^uint(0)>>1) {
		return snapshot.Tree{}, ErrLimit
	}
	blobs, err := w.assignmentOutput(ctx, a, strings.NewReader(input.String()), int(limit), "cat-file", "--batch")
	if err != nil {
		return snapshot.Tree{}, err
	}
	reader := bufio.NewReader(bytes.NewReader(blobs))
	var tree []snapshot.Entry
	for _, entry := range entries {
		header, err := reader.ReadString('\n')
		if err != nil || header != entry.oid+" blob "+strconv.FormatInt(entry.size, 10)+"\n" {
			return snapshot.Tree{}, errors.New("named Git blob header does not match its tree")
		}
		data := make([]byte, int(entry.size))
		if _, err = io.ReadFull(reader, data); err != nil {
			return snapshot.Tree{}, err
		}
		separator, err := reader.ReadByte()
		if err != nil || separator != '\n' {
			return snapshot.Tree{}, errors.New("named Git blob is incomplete")
		}
		framed := append([]byte("blob "+strconv.Itoa(len(data))+"\x00"), data...)
		var actual string
		if len(entry.oid) == 40 {
			hash := sha1.Sum(framed)
			actual = hex.EncodeToString(hash[:])
		} else {
			hash := sha256.Sum256(framed)
			actual = hex.EncodeToString(hash[:])
		}
		if actual != entry.oid {
			return snapshot.Tree{}, errors.New("named Git blob content does not match its object identity")
		}
		path, _ := snapshot.NewPath(entry.path)
		mode := snapshot.Regular
		if entry.mode == "100755" {
			mode = snapshot.Executable
		}
		value, err := snapshot.NewEntry(path, mode, data)
		if err != nil {
			return snapshot.Tree{}, err
		}
		tree = append(tree, value)
	}
	if _, err := reader.ReadByte(); err != io.EOF {
		return snapshot.Tree{}, errors.New("unexpected trailing Git blob data")
	}
	return snapshot.NewTree(tree)
}

func (w *Workspaces) assignmentOutput(ctx context.Context, a *assignmentWorkspace, input io.Reader, limit int, args ...string) ([]byte, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	volume, observed, err := w.ownedVolume(ctx, a.id)
	if err != nil || volume != a.volume || !observed.Exists {
		return nil, errors.Join(err, errors.New("named Git source workspace is unavailable"))
	}
	output := &reviewOutput{limit: limit}
	argv := assignmentGitArgs(append([]string{"--no-replace-objects"}, args...)...)
	err = w.runtime.RunHelper(ctx, w.spec(volume, true), argv, input, output)
	if output.exceeded {
		return nil, ErrLimit
	}
	if err != nil {
		return nil, err
	}
	return output.data.Bytes(), nil
}
