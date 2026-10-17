package gitrepo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Conversion runs in an external temporary bare repository with no owner,
// system or global configuration. Its only attributes are the closed set
// observed on the requested path. No filter, hook or textconv is executable.
type publicationConverter struct {
	ctx        context.Context
	dir        string
	limits     Limits
	configured bool
	policy     publicationPolicy
}

func newPublicationConverter(ctx context.Context, limits Limits) (*publicationConverter, error) {
	dir, err := os.MkdirTemp("", "markitect-publication-")
	if err != nil {
		return nil, err
	}
	c := &publicationConverter{ctx: ctx, dir: dir, limits: limits}
	if _, err := c.run(nil, "init", "--bare", "--quiet", "--template=", dir); err != nil {
		c.close()
		return nil, err
	}
	return c, nil
}
func (c *publicationConverter) close() { _ = os.RemoveAll(c.dir) }
func (c *publicationConverter) run(data []byte, args ...string) ([]byte, error) {
	prefix := []string{"-c", "core.hooksPath=", "-c", "core.fsmonitor=false", "-c", "core.attributesFile=" + filepath.Join(c.dir, "attributes"), "--git-dir=" + c.dir}
	cmd := exec.CommandContext(c.ctx, "git", append(prefix, args...)...)
	cmd.Env = append(gitEnvironment(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
	cmd.Stdin = bytes.NewReader(data)
	output, stderr := &boundedOutput{limit: c.limits.FileBytes}, &boundedOutput{limit: 16 << 10}
	cmd.Stdout, cmd.Stderr, cmd.WaitDelay = output, stderr, time.Second
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("publication conversion: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return output.Bytes(), nil
}
func (c *publicationConverter) configure(p publicationPolicy) error {
	if c.configured && c.policy == p {
		return nil
	}
	attrs := "*"
	switch p.text {
	case "set":
		attrs += " text"
	case "unset":
		attrs += " -text"
	case "auto":
		attrs += " text=auto"
	}
	if p.pathEOL == "lf" || p.pathEOL == "crlf" {
		attrs += " eol=" + p.pathEOL
	}
	if err := os.WriteFile(filepath.Join(c.dir, "attributes"), []byte(attrs+"\n"), 0600); err != nil {
		return err
	}
	for _, pair := range [][2]string{{"core.autocrlf", p.autocrlf}, {"core.eol", p.eol}, {"core.safecrlf", "false"}} {
		if _, err := c.run(nil, "config", pair[0], pair[1]); err != nil {
			return err
		}
	}
	c.configured, c.policy = true, p
	return nil
}
func (c *publicationConverter) object(data []byte) (string, error) {
	out, err := c.run(data, "hash-object", "-w", "--no-filters", "--stdin")
	return strings.TrimSpace(string(out)), err
}
func (c *publicationConverter) clean(data []byte, p publicationPolicy) ([]byte, error) {
	if err := c.configure(p); err != nil {
		return nil, err
	}
	out, err := c.run(data, "hash-object", "-w", "--path=publication", "--stdin")
	if err != nil {
		return nil, err
	}
	return c.run(nil, "cat-file", "blob", strings.TrimSpace(string(out)))
}
func (c *publicationConverter) checkout(entry snapshot.Entry, p publicationPolicy) ([]byte, error) {
	if entry.ByteLength() > c.limits.FileBytes {
		return nil, ErrLimit
	}
	if err := c.configure(p); err != nil {
		return nil, err
	}
	oid, err := c.object(entry.Bytes())
	if err != nil {
		return nil, err
	}
	data, err := c.run(nil, "cat-file", "--filters", "--path=publication", oid)
	if err != nil {
		return nil, err
	}
	clean, err := c.clean(data, p)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(clean, entry.Bytes()) {
		return nil, fmt.Errorf("%w: requested content is not Git clean form at %q", ErrUnrepresentable, entry.Path())
	}
	return data, nil
}

func publicationParent(d *diskBoundary, relative string, create bool) (*os.Root, string, error) {
	parts := strings.Split(relative, "/")
	root := d.dirs["."].root
	current := ""
	for _, name := range parts[:len(parts)-1] {
		names, err := directoryNames(root, d.limits.Entries)
		if err != nil {
			return nil, "", err
		}
		if !slices.Contains(names, name) {
			if _, err := root.Lstat(name); !errors.Is(err, os.ErrNotExist) {
				return nil, "", ErrUnrepresentable
			}
			key, ok := model.NameCollisionKey(name)
			if !ok {
				return nil, "", ErrUnsafe
			}
			for _, other := range names {
				k, ok := model.NameCollisionKey(other)
				if !ok || key == k {
					return nil, "", ErrUnrepresentable
				}
			}
			if !create {
				return nil, "", os.ErrNotExist
			}
			if err := root.Mkdir(name, 0755); err != nil {
				return nil, "", err
			}
		}
		if current != "" {
			current += "/"
		}
		current += name
		root, err = d.directory(current, true)
		if err != nil {
			return nil, "", err
		}
	}
	return root, parts[len(parts)-1], nil
}

func (r *Repository) publicationWrite(ctx context.Context, g *gitRepository, change snapshot.Change, data []byte) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := g.confirm(); err != nil {
		return false, err
	}
	d, err := openDisk(ctx, g.path, r.limits, nil)
	if err != nil {
		return false, err
	}
	defer d.close()
	root, name, err := publicationParent(d, change.Path().String(), change.Kind() == snapshot.Add)
	if err != nil {
		return false, err
	}
	names, err := directoryNames(root, r.limits.Entries)
	if err != nil {
		return false, err
	}
	before, had := change.Before()
	if had {
		actual, mode, found, err := d.read(change.Path().String(), true)
		if err != nil {
			return false, err
		}
		// Capture requires raw tracked bytes. Before the first write each requested
		// existing path must still be that exact captured file.
		policy, err := publicationPolicies(g, []snapshot.Change{change})
		if err != nil {
			return false, err
		}
		if policy[change.Path().String()].filemode == "false" {
			mode = before.Mode()
		}
		if !found || !bytes.Equal(actual, before.Bytes()) || mode != before.Mode() {
			return false, ErrChanged
		}
	} else {
		if slices.Contains(names, name) {
			return false, ErrChanged
		}
		key, _ := model.NameCollisionKey(name)
		for _, other := range names {
			k, ok := model.NameCollisionKey(other)
			if !ok || k == key {
				return false, ErrUnrepresentable
			}
		}
	}
	if _, has := change.After(); !has {
		if err := root.Remove(name); err != nil {
			return false, err
		}
		return true, nil
	}
	after, _ := change.After()
	// Replace the directory entry rather than truncating an existing inode:
	// a regular captured file can have hard links outside the requested paths.
	// The deliberate remove/create gap is a visible partial publication.
	wrote := false
	if had {
		if err := root.Remove(name); err != nil {
			return false, err
		}
		wrote = true
	}
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	file, err := root.OpenFile(name, flags, 0666)
	if err != nil {
		return wrote, err
	}
	defer file.Close()
	opened, err := file.Stat()
	atPath, pathErr := root.Lstat(name)
	if err != nil || pathErr != nil || redirected(atPath) || !opened.Mode().IsRegular() || !os.SameFile(opened, atPath) {
		return true, ErrChanged
	}
	if err := file.Truncate(0); err != nil {
		return true, err
	}
	wrote = true
	_, writeErr := file.Write(data)
	permission := opened.Mode().Perm() & 0666
	if after.Mode() == snapshot.Executable {
		permission |= (permission & 0444) >> 2
	}
	modeErr := file.Chmod(permission)
	closeErr := file.Close()
	if err := errors.Join(writeErr, modeErr, closeErr); err != nil {
		return wrote, err
	}
	if r.observe != nil {
		r.observe("publication-file-written", change.Path().String())
	}
	// Confirm directory identities without trying to confirm the intentionally
	// changed file's old metadata.
	delete(d.files, change.Path().String())
	delete(d.contents, change.Path().String())
	if err := d.confirm(); err != nil {
		return wrote, err
	}
	return wrote, nil
}
