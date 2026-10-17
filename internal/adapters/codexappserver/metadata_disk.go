package codexappserver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const (
	maxExecutableBytes  int64 = 512 << 20
	maxSchemaBytes      int64 = 16 << 20
	maxSchemaFileBytes  int64 = 2 << 20
	maxSchemaFiles            = 1024
	maxDirectories            = 1024
	maxDirectoryEntries       = 65536
	maxPathDepth              = 64
)

type invalidMetadata struct{ message string }

func (e *invalidMetadata) Error() string       { return e.message }
func invalid(format string, args ...any) error { return &invalidMetadata{fmt.Sprintf(format, args...)} }

type observedDiskPath struct {
	info  os.FileInfo
	names []string
}
type metadataDisk struct {
	ctx                context.Context
	paths              map[string]observedDiskPath
	dirs               map[string]*heldMetadataDirectory
	nameObservations   map[*heldMetadataDirectory][]string
	beforeOpen         func(stage, path string) // private deterministic regression seam
	schemaBytes        int64
	files, directories int
}

func newMetadataDisk(ctx context.Context) *metadataDisk {
	return &metadataDisk{ctx: ctx, paths: map[string]observedDiskPath{}, dirs: map[string]*heldMetadataDirectory{}, nameObservations: map[*heldMetadataDirectory][]string{}}
}
func (d *metadataDisk) close() {
	for _, dir := range d.dirs {
		_ = dir.file.Close()
	}
}

func sameObservation(a, b os.FileInfo) bool {
	return os.SameFile(a, b) && a.Mode() == b.Mode() && a.Size() == b.Size() && a.ModTime().Equal(b.ModTime())
}

func (d *metadataDisk) names(abs string) ([]string, error) {
	if err := d.ctx.Err(); err != nil {
		return nil, err
	}
	dir := d.dirs[abs]
	if dir == nil {
		return nil, fmt.Errorf("directory handle not acquired %q", abs)
	}
	return d.directoryNames(dir, true)
}

// Share one name observation per held directory during acquisition. Verify
// starts a fresh observation phase. Identity checks and native no-follow opens
// remain live; names describe the beginning/end observations, not every instant
// between them. The unchecked variant avoids recursive namespace verification.
func (d *metadataDisk) directoryNames(dir *heldMetadataDirectory, verify bool) ([]string, error) {
	abs := dir.abs
	if verify {
		if err := d.confirmDirectory(dir); err != nil {
			return nil, err
		}
	}
	if names, ok := d.nameObservations[dir]; ok {
		return names, nil
	}
	if d.beforeOpen != nil {
		d.beforeOpen("before-enumeration-open", abs)
	}
	// Reopen '.' relative to the held directory, never through its absolute
	// namespace. Verify the handle AND every held namespace edge before reading.
	f, err := openMetadataChild(dir.file, ".", true)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if d.beforeOpen != nil {
		d.beforeOpen("after-enumeration-open", abs)
	}
	opened, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if redirected(opened) || !opened.IsDir() || !os.SameFile(dir.info, opened) {
		return nil, fmt.Errorf("directory handle changed %q", abs)
	}
	if verify {
		if err := d.confirmDirectory(dir); err != nil {
			return nil, err
		}
	}
	var names []string
	for {
		if err := d.ctx.Err(); err != nil {
			return nil, err
		}
		batch, err := f.Readdirnames(256)
		names = append(names, batch...)
		if len(names) > maxDirectoryEntries {
			return nil, fmt.Errorf("finite directory entry limit at %q", abs)
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	if verify {
		if err := d.confirmDirectory(dir); err != nil {
			return nil, err
		}
	}
	slices.Sort(names)
	d.nameObservations[dir] = names
	return names, nil
}

// Check every native component, including ancestors of a selected boundary.
// Exact stored names reject case, normalization and platform short-name aliases.
func (d *metadataDisk) inspect(abs string) (os.FileInfo, error) {
	if err := d.ctx.Err(); err != nil {
		return nil, err
	}
	if !literalAbsolute(abs) {
		return nil, invalid("nonliteral absolute path %q", abs)
	}
	if dir := d.dirs[abs]; dir != nil {
		if err := d.confirmDirectory(dir); err != nil {
			return nil, err
		}
		return dir.info, nil
	}
	parent, err := d.root(filepath.Dir(abs))
	if err != nil {
		return nil, err
	}
	name := filepath.Base(abs)
	if err := d.exactName(parent, name); err != nil {
		return nil, err
	}
	info, err := statMetadataChild(parent.file, name)
	if err != nil {
		return nil, err
	}
	if redirected(info) {
		return nil, invalid("redirected path %q", abs)
	}
	if old, ok := d.paths[abs]; ok && !sameObservation(old.info, info) {
		return nil, fmt.Errorf("selected path changed %q", abs)
	}
	d.paths[abs] = observedDiskPath{info: info}
	return info, nil
}

type heldMetadataDirectory struct {
	file      *os.File
	info      os.FileInfo
	abs, name string
	parent    *heldMetadataDirectory
}

func statMetadataChild(parent *os.File, name string) (os.FileInfo, error) {
	f, err := openMetadataChild(parent, name, false)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	return info, errors.Join(err, f.Close())
}

func (d *metadataDisk) exactName(parent *heldMetadataDirectory, want string) error {
	// This is only lookup within an already held object. The actual child
	// open/read and inventory enumeration each confirm the namespace chain.
	names, err := d.directoryNames(parent, false)
	if err != nil {
		return err
	}
	if !slices.Contains(names, want) {
		key, _ := model.NameCollisionKey(want)
		for _, name := range names {
			other, ok := model.NameCollisionKey(name)
			if ok && key == other {
				return invalid("aliased stored name %q", want)
			}
		}
		// A truly absent input is unknown acquisition, while an existing path
		// under another stored spelling (e.g. Windows 8.3) is a refused alias.
		if _, err := statMetadataChild(parent.file, want); err != nil {
			return err
		}
		return invalid("exact stored path name absent %q", filepath.Join(parent.abs, want))
	}
	key, ok := model.NameCollisionKey(want)
	if !ok {
		return invalid("invalid path name %q", want)
	}
	for _, other := range names {
		if other == want {
			continue
		}
		k, valid := model.NameCollisionKey(other)
		if valid && k == key {
			return invalid("aliased sibling of %q", want)
		}
	}
	return nil
}

func (d *metadataDisk) confirmDirectory(dir *heldMetadataDirectory) error {
	for current := dir; current != nil; current = current.parent {
		if err := d.confirmDirectoryEdge(current); err != nil {
			return err
		}
	}
	return nil
}

func (d *metadataDisk) confirmDirectoryEdge(dir *heldMetadataDirectory) error {
	if err := d.ctx.Err(); err != nil {
		return err
	}
	opened, err := dir.file.Stat()
	if err != nil {
		return err
	}
	if redirected(opened) || !opened.IsDir() || !os.SameFile(dir.info, opened) || dir.info.Mode() != opened.Mode() {
		return fmt.Errorf("held directory changed %q", dir.abs)
	}
	var atPath os.FileInfo
	if dir.parent == nil {
		atPath, err = os.Lstat(dir.abs)
	} else {
		if err := d.confirmStoredName(dir.parent, dir.name); err != nil {
			return err
		}
		atPath, err = statMetadataChild(dir.parent.file, dir.name)
	}
	if err != nil {
		return err
	}
	if redirected(atPath) || !atPath.IsDir() || !os.SameFile(dir.info, atPath) || dir.info.Mode() != atPath.Mode() {
		return fmt.Errorf("directory namespace changed %q", dir.abs)
	}
	return nil
}

func (d *metadataDisk) confirmFileName(parent *heldMetadataDirectory, name string) error {
	if err := d.confirmDirectory(parent); err != nil {
		return err
	}
	return d.confirmStoredName(parent, name)
}

func (d *metadataDisk) confirmStoredName(parent *heldMetadataDirectory, name string) error {
	names, err := d.directoryNames(parent, false)
	if err != nil {
		return err
	}
	if !retainedMetadataName(names, name) {
		return fmt.Errorf("stored spelling or collision changed %q", filepath.Join(parent.abs, name))
	}
	return nil
}

func retainedMetadataName(names []string, want string) bool {
	if !slices.Contains(names, want) {
		return false
	}
	key, ok := model.NameCollisionKey(want)
	if !ok {
		return false
	}
	for _, name := range names {
		if name == want {
			continue
		}
		other, valid := model.NameCollisionKey(name)
		if valid && other == key {
			return false
		}
	}
	return true
}

func (d *metadataDisk) root(abs string) (*heldMetadataDirectory, error) {
	if err := d.ctx.Err(); err != nil {
		return nil, err
	}
	if !literalAbsolute(abs) {
		return nil, invalid("nonliteral boundary %q", abs)
	}
	if dir := d.dirs[abs]; dir != nil {
		// Retaining a handle performs no path access. Uses below confirm all
		// edges at the actual file-open/read or directory-enumeration boundary.
		return dir, nil
	}
	volume := filepath.VolumeName(abs) + string(filepath.Separator)
	parts := strings.Split(strings.TrimPrefix(abs, volume), string(filepath.Separator))
	if len(parts) > maxPathDepth {
		return nil, fmt.Errorf("finite path depth limit at %q", abs)
	}
	var parent *heldMetadataDirectory
	name := filepath.Base(abs)
	var before os.FileInfo
	var err error
	if abs == volume {
		before, err = os.Lstat(volume)
	} else {
		parent, err = d.root(filepath.Dir(abs))
		if err != nil {
			return nil, err
		}
		if err = d.exactName(parent, name); err != nil {
			return nil, err
		}
		before, err = statMetadataChild(parent.file, name)
	}
	if err != nil {
		return nil, err
	}
	if redirected(before) || !before.IsDir() {
		return nil, invalid("redirected or non-directory boundary %q", abs)
	}
	if d.beforeOpen != nil {
		d.beforeOpen("before-directory-open", abs)
	}
	var file *os.File
	if parent == nil {
		file, err = openMetadataVolume(volume)
	} else {
		file, err = openMetadataChild(parent.file, name, true)
	}
	if err != nil {
		return nil, err
	}
	opened, err := file.Stat()
	if err != nil || redirected(opened) || !opened.IsDir() || !os.SameFile(before, opened) {
		_ = file.Close()
		return nil, fmt.Errorf("directory changed while opening %q", abs)
	}
	dir := &heldMetadataDirectory{file: file, info: opened, abs: abs, name: name, parent: parent}
	if err = d.confirmDirectory(dir); err != nil {
		_ = file.Close()
		return nil, err
	}
	d.dirs[abs] = dir
	d.paths[abs] = observedDiskPath{info: opened}
	return dir, nil
}

func (d *metadataDisk) readAbsolute(abs string, limit int64, keep bool) (apply.ProfileMetadataFile, []byte, error) {
	boundary := filepath.Dir(abs)
	r, err := d.root(boundary)
	if err != nil {
		return apply.ProfileMetadataFile{}, nil, err
	}
	return d.read(r, boundary, filepath.Base(abs), limit, keep)
}

func (d *metadataDisk) read(root *heldMetadataDirectory, boundary, rel string, limit int64, keep bool) (apply.ProfileMetadataFile, []byte, error) {
	var result apply.ProfileMetadataFile
	abs := filepath.Join(boundary, filepath.FromSlash(rel))
	before, err := d.inspect(abs)
	if err != nil {
		return result, nil, err
	}
	if !before.Mode().IsRegular() {
		return result, nil, invalid("not a regular file %q", abs)
	}
	if before.Size() > limit {
		return result, nil, fmt.Errorf("finite file byte limit at %q", abs)
	}
	parent, err := d.root(filepath.Dir(abs))
	if err != nil {
		return result, nil, err
	}
	if d.beforeOpen != nil {
		d.beforeOpen("before-file-open", abs)
	}
	f, err := openMetadataChild(parent.file, filepath.Base(abs), false)
	if err != nil {
		return result, nil, err
	}
	defer f.Close()
	if err := d.confirmFileName(parent, filepath.Base(abs)); err != nil {
		return result, nil, err
	}
	opened, err := f.Stat()
	if err != nil {
		return result, nil, err
	}
	atPath, err := statMetadataChild(parent.file, filepath.Base(abs))
	if err != nil {
		return result, nil, err
	}
	if redirected(opened) || !opened.Mode().IsRegular() || redirected(atPath) || !sameObservation(before, opened) || !os.SameFile(opened, atPath) {
		return result, nil, fmt.Errorf("file changed while opening %q", abs)
	}
	h := sha256.New()
	var raw bytes.Buffer
	buffer := make([]byte, 32<<10)
	var size int64
	for {
		if err := d.ctx.Err(); err != nil {
			return result, nil, err
		}
		n, readErr := f.Read(buffer)
		if n > 0 {
			size += int64(n)
			if size > limit {
				return result, nil, fmt.Errorf("finite file byte limit at %q", abs)
			}
			_, _ = h.Write(buffer[:n])
			if keep {
				_, _ = raw.Write(buffer[:n])
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return result, nil, readErr
		}
	}
	after, err := f.Stat()
	if err != nil {
		return result, nil, err
	}
	atPath, err = statMetadataChild(parent.file, filepath.Base(abs))
	if err != nil {
		return result, nil, err
	}
	if redirected(atPath) || !sameObservation(opened, after) || !sameObservation(after, atPath) || size != after.Size() {
		return result, nil, fmt.Errorf("file changed while reading %q", abs)
	}
	if err := d.confirmFileName(parent, filepath.Base(abs)); err != nil {
		return result, nil, err
	}
	path, err := snapshot.NewPath(rel)
	if err != nil {
		return result, nil, invalid("unrepresentable metadata path %q", rel)
	}
	result = apply.ProfileMetadataFile{Location: apply.SourceLocation{Boundary: boundary, Path: path}, Size: size, Mode: snapshot.Regular}
	if after.Mode().Perm()&0111 != 0 {
		result.Mode = snapshot.Executable
	}
	copy(result.ContentID[:], h.Sum(nil))
	d.paths[abs] = observedDiskPath{info: after}
	return result, raw.Bytes(), nil
}

func (d *metadataDisk) inventory(boundary string, selected []string) (apply.ProfileMetadataInventory, bool, error) {
	result := apply.ProfileMetadataInventory{Boundary: boundary}
	root, err := d.root(boundary)
	if err != nil {
		return result, false, err
	}
	var failures error
	var walk func(string, int)
	walk = func(rel string, depth int) {
		if err := d.ctx.Err(); err != nil {
			failures = errors.Join(failures, err)
			return
		}
		d.directories++
		if depth > maxPathDepth || d.directories > maxDirectories {
			failures = errors.Join(failures, fmt.Errorf("finite schema directory/depth limit"))
			return
		}
		abs := filepath.Join(boundary, filepath.FromSlash(rel))
		held, err := d.root(abs)
		if err != nil {
			failures = errors.Join(failures, err)
			return
		}
		info := held.info
		if !info.IsDir() {
			failures = errors.Join(failures, invalid("schema directory is not regular %q", abs))
			return
		}
		names, err := d.names(abs)
		if err != nil {
			failures = errors.Join(failures, err)
			return
		}
		d.paths[abs] = observedDiskPath{info: info, names: names}
		if len(names) == 0 {
			failures = errors.Join(failures, invalid("unexpected empty schema directory %q", abs))
			return
		}
		for _, name := range names {
			child := name
			if rel != "." {
				child = rel + "/" + name
			}
			if _, err := snapshot.NewPath(child); err != nil {
				failures = errors.Join(failures, invalid("unrepresentable schema member %q", child))
				continue
			}
			dir, err := d.root(abs)
			if err != nil {
				failures = errors.Join(failures, err)
				return
			}
			info, err := statMetadataChild(dir.file, name)
			if err != nil {
				failures = errors.Join(failures, err)
				continue
			}
			if redirected(info) {
				failures = errors.Join(failures, invalid("redirected schema member %q", child))
				continue
			}
			if info.IsDir() {
				walk(child, depth+1)
				continue
			}
			d.files++
			if d.files > maxSchemaFiles {
				failures = errors.Join(failures, fmt.Errorf("finite schema file count limit"))
				return
			}
			remaining := maxSchemaBytes - d.schemaBytes
			if remaining <= 0 {
				failures = errors.Join(failures, fmt.Errorf("finite schema total byte limit"))
				return
			}
			metadata, data, err := d.read(root, boundary, child, min(maxSchemaFileBytes, remaining), slices.Contains(selected, child))
			if err != nil {
				failures = errors.Join(failures, err)
				continue
			}
			result.Files = append(result.Files, metadata)
			d.schemaBytes += metadata.Size
			if d.schemaBytes > maxSchemaBytes {
				failures = errors.Join(failures, fmt.Errorf("finite schema total byte limit"))
				return
			}
			if slices.Contains(selected, child) && !json.Valid(data) {
				failures = errors.Join(failures, invalid("selected schema is invalid JSON %q", child))
			}
		}
	}
	walk(".", 0)
	for _, path := range selected {
		found := false
		for _, file := range result.Files {
			if file.Location.Path.String() == path {
				found = true
				break
			}
		}
		if !found && failures == nil {
			failures = errors.Join(failures, invalid("required original schema file absent %q", path))
		}
	}
	slices.SortFunc(result.Files, func(a, b apply.ProfileMetadataFile) int {
		return strings.Compare(a.Location.Path.String(), b.Location.Path.String())
	})
	return result, failures == nil, failures
}

func (d *metadataDisk) verify() error {
	defer clear(d.nameObservations)
	if err := d.ctx.Err(); err != nil {
		return err
	}
	// Reobserve every relevant held parent once. Ordinary parents are checked
	// only for used basenames/collision siblings; only inventory directories
	// compare complete membership. No observations survive across calls.
	clear(d.nameObservations)
	var failures error
	paths := make([]string, 0, len(d.paths))
	for p := range d.paths {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	for _, path := range paths {
		if err := d.ctx.Err(); err != nil {
			return errors.Join(failures, err)
		}
		old := d.paths[path]
		var now os.FileInfo
		var err error
		if dir := d.dirs[path]; dir != nil {
			err = d.confirmDirectoryEdge(dir)
			if err == nil {
				now, err = dir.file.Stat()
			}
		} else if parent := d.dirs[filepath.Dir(path)]; parent != nil {
			err = d.confirmStoredName(parent, filepath.Base(path))
			if err == nil {
				now, err = statMetadataChild(parent.file, filepath.Base(path))
			}
		} else {
			err = fmt.Errorf("metadata path has no held parent %q", path)
		}
		if err != nil {
			failures = errors.Join(failures, err)
			continue
		}
		if redirected(now) || !os.SameFile(old.info, now) || old.info.Mode() != now.Mode() {
			failures = errors.Join(failures, fmt.Errorf("observed path changed %q", path))
			continue
		}
		if !now.IsDir() && !sameObservation(old.info, now) {
			failures = errors.Join(failures, fmt.Errorf("observed file changed %q", path))
		}
		if old.names != nil {
			names, err := d.directoryNames(d.dirs[path], false)
			if err != nil {
				failures = errors.Join(failures, err)
			} else if !slices.Equal(old.names, names) {
				failures = errors.Join(failures, fmt.Errorf("schema directory membership changed %q", path))
			} else {
				old.names = names
				d.paths[path] = old
			}
		}
	}
	return failures
}

// path-size-sha256-nul-v1: ordinal relative '/' paths, UTF8 without BOM;
// path NUL decimal whole-byte size NUL lowercase SHA256 NUL, no line separators.
func inventoryDigest(files []apply.ProfileMetadataFile) snapshot.ContentID {
	ordered := slices.Clone(files)
	slices.SortFunc(ordered, func(a, b apply.ProfileMetadataFile) int {
		return strings.Compare(a.Location.Path.String(), b.Location.Path.String())
	})
	h := sha256.New()
	for _, file := range ordered {
		for _, text := range []string{file.Location.Path.String(), strconv.FormatInt(file.Size, 10), file.ContentID.String()} {
			_, _ = io.WriteString(h, text)
			_, _ = h.Write([]byte{0})
		}
	}
	var result snapshot.ContentID
	copy(result[:], h.Sum(nil))
	return result
}
func digestLiteral(s string) snapshot.ContentID {
	data, err := hex.DecodeString(s)
	if err != nil || len(data) != 32 {
		panic("invalid built-in metadata pin")
	}
	var result snapshot.ContentID
	copy(result[:], data)
	return result
}
