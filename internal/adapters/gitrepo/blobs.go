package gitrepo

import (
	"bufio"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

type blobStream struct {
	cmd    *exec.Cmd
	in     io.WriteCloser
	out    *bufio.Reader
	stderr *boundedOutput
	cache  map[string][]byte
	waited bool
}

func (g *gitRepository) blob(entry gitEntry, limit int64) ([]byte, error) {
	if entry.kind != "blob" || entry.mode != "100644" && entry.mode != "100755" {
		return nil, fmt.Errorf("%w: nonregular Git source %q", ErrUnsafe, entry.path)
	}
	if entry.size > limit {
		return nil, fmt.Errorf("%w: file %q", ErrLimit, entry.path)
	}
	if uint64(entry.size) > uint64(^uint(0)>>1) {
		return nil, ErrLimit
	}
	if g.blobs == nil {
		cmd := g.command("cat-file", "--batch")
		in, err := cmd.StdinPipe()
		if err != nil {
			return nil, err
		}
		out, err := cmd.StdoutPipe()
		if err != nil {
			_ = in.Close()
			return nil, err
		}
		stderr := &boundedOutput{limit: 16 << 10}
		cmd.Stderr = stderr
		if err := cmd.Start(); err != nil {
			_ = in.Close()
			return nil, err
		}
		g.blobs = &blobStream{cmd: cmd, in: in, out: bufio.NewReaderSize(out, 512), stderr: stderr, cache: make(map[string][]byte)}
	}
	b := g.blobs
	if data, ok := b.cache[entry.oid]; ok {
		return data, nil
	}
	if _, err := io.WriteString(b.in, entry.oid+"\n"); err != nil {
		return nil, err
	}
	header, err := b.out.ReadSlice('\n')
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(string(header))
	if len(header) > 256 || len(fields) != 3 || fields[0] != entry.oid || fields[1] != "blob" {
		return nil, fmt.Errorf("%w: unexpected blob header", ErrUnsafe)
	}
	size, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil || size != entry.size {
		return nil, fmt.Errorf("%w: blob size", ErrChanged)
	}
	data := make([]byte, int(size))
	if _, err := io.ReadFull(b.out, data); err != nil {
		return nil, err
	}
	separator, err := b.out.ReadByte()
	if err != nil || separator != '\n' {
		return nil, fmt.Errorf("%w: blob separator", ErrUnsafe)
	}
	framed := append([]byte("blob "+strconv.Itoa(len(data))+"\x00"), data...)
	var oid string
	if g.format == "sha1" {
		hash := sha1.Sum(framed)
		oid = hex.EncodeToString(hash[:])
	} else {
		hash := sha256.Sum256(framed)
		oid = hex.EncodeToString(hash[:])
	}
	if oid != entry.oid {
		return nil, fmt.Errorf("%w: blob content hash", ErrChanged)
	}
	b.cache[entry.oid] = data
	return data, nil
}

func (g *gitRepository) finishBlobs(success bool) error {
	if g.blobs == nil || g.blobs.waited {
		return nil
	}
	b := g.blobs
	_ = b.in.Close()
	if !success {
		_ = b.cmd.Process.Kill()
	}
	var trailing error
	if success {
		if _, err := b.out.ReadByte(); err != io.EOF {
			trailing = fmt.Errorf("%w: unexpected trailing blob output", ErrUnsafe)
		}
	}
	err := b.cmd.Wait()
	b.waited = true
	if g.ctx.Err() != nil && success {
		return g.ctx.Err()
	}
	if trailing != nil {
		return trailing
	}
	if err != nil && success {
		return fmt.Errorf("Git blob stream: %w: %s", err, b.stderr.String())
	}
	return nil
}
