package gitrepo

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type workspaceFake struct {
	volumes                                   map[string]container.VolumeObservation
	trees                                     map[string]snapshot.Tree
	mounts                                    []string
	createErr, writeErr, censusErr, removeErr error
	removeNoop                                bool
	corrupt                                   bool
	truncated                                 bool
	startMount                                bool
	helpers                                   []container.ContainerSpec
	removals                                  int
	initialTree                               snapshot.Tree
	censusCalls                               int
	directories                               map[string][]string
}

func (f *workspaceFake) InspectVolume(_ context.Context, name string) (container.VolumeObservation, error) {
	if observed, ok := f.volumes[name]; ok {
		return observed, nil
	}
	return container.VolumeObservation{Name: name}, nil
}
func (f *workspaceFake) CreateVolumeExclusive(_ context.Context, name string, labels map[string]string) (container.VolumeObservation, error) {
	if observed, ok := f.volumes[name]; ok {
		return observed, errors.New("already exists")
	}
	observed := container.VolumeObservation{Name: name, Exists: true, Labels: cloneWorkspaceLabels(labels)}
	f.volumes[name] = observed
	f.trees[name] = f.initialTree
	return observed, f.createErr
}
func (f *workspaceFake) VolumeMounts(context.Context, string) ([]string, error) { return f.mounts, nil }
func (f *workspaceFake) RemoveVolume(_ context.Context, name string) error {
	f.removals++
	if !f.removeNoop {
		delete(f.volumes, name)
	}
	return f.removeErr
}
func (f *workspaceFake) RunHelper(_ context.Context, spec container.ContainerSpec, argv []string, stdin io.Reader, stdout io.Writer) error {
	f.helpers = append(f.helpers, spec)
	volume := spec.Mounts[0].Volume
	if argv[0] == "/bin/tar" {
		if f.writeErr != nil {
			return f.writeErr
		}
		var entries []snapshot.Entry
		reader := tar.NewReader(stdin)
		for {
			header, err := reader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			if header.Typeflag == tar.TypeDir {
				if header.Mode != 0755 || !strings.HasSuffix(header.Name, "/") {
					return errors.New("directory header is not mode 0755 with a trailing slash")
				}
				if f.directories == nil {
					f.directories = make(map[string][]string)
				}
				f.directories[volume] = append(f.directories[volume], strings.TrimSuffix(header.Name, "/"))
				continue
			}
			data, err := io.ReadAll(reader)
			if err != nil {
				return err
			}
			path, err := snapshot.NewPath(header.Name)
			if err != nil {
				return err
			}
			mode := snapshot.Regular
			if header.Mode&0100 != 0 {
				mode = snapshot.Executable
			}
			entry, err := snapshot.NewEntry(path, mode, data)
			if err != nil {
				return err
			}
			entries = append(entries, entry)
		}
		var err error
		f.trees[volume], err = snapshot.NewTree(entries)
		return err
	}
	if argv[0] != "/usr/local/bin/node" || argv[2] != workspaceCensusScript || !spec.Mounts[0].ReadOnly {
		return errors.New("unexpected census helper")
	}
	f.censusCalls++
	for _, entry := range f.trees[volume].Entries() {
		data := entry.Bytes()
		if f.corrupt {
			data = append(data, '!')
		}
		if err := writeWorkspaceFrame(stdout, workspaceFrame{Path: base64.StdEncoding.EncodeToString([]byte(entry.Path().String())), Type: "file", Size: int64(len(data)), Execute: entry.Mode() == snapshot.Executable}); err != nil {
			return err
		}
		if _, err := stdout.Write(data); err != nil {
			return err
		}
	}
	if f.startMount && f.censusCalls > 1 {
		f.mounts = []string{"late-container"}
	}
	if !f.truncated || f.censusCalls == 1 {
		if err := writeWorkspaceFrame(stdout, workspaceFrame{End: true}); err != nil {
			return err
		}
	}
	if f.censusCalls > 1 {
		return f.censusErr
	}
	return nil
}
func writeWorkspaceFrame(writer io.Writer, f workspaceFrame) error {
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(data)))
	if _, err = writer.Write(header[:]); err != nil {
		return err
	}
	_, err = writer.Write(data)
	return err
}
func workspaceEntry(t *testing.T, p string, mode snapshot.Mode, data string) snapshot.Entry {
	t.Helper()
	path, err := snapshot.NewPath(p)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := snapshot.NewEntry(path, mode, []byte(data))
	if err != nil {
		t.Fatal(err)
	}
	return entry
}
func workspaceTree(t *testing.T, entries ...snapshot.Entry) snapshot.Tree {
	t.Helper()
	tree, err := snapshot.NewTree(entries)
	if err != nil {
		t.Fatal(err)
	}
	return tree
}
func workspaceAdapter(t *testing.T) (*Workspaces, *workspaceFake) {
	t.Helper()
	f := &workspaceFake{volumes: make(map[string]container.VolumeObservation), trees: make(map[string]snapshot.Tree)}
	w, err := NewWorkspaces(f, container.ContainerSpec{Name: "runtime", Image: "sha256:" + strings.Repeat("a", 64), User: "1000:1000", Labels: map[string]string{"markitect.store": "store"}, CPUs: 1, MemoryBytes: 1 << 28, Pids: 32, TmpBytes: 1 << 20, Network: "none"})
	if err != nil {
		t.Fatal(err)
	}
	return w, f
}
func TestWorkspaceHelperNamesDifferAcrossAdaptersWithTheSameHelper(t *testing.T) {
	// Two processes with the same Runtime configuration each start their
	// sequence at 1; the workspace part keeps their helper names apart.
	tree := workspaceTree(t, workspaceEntry(t, "a", snapshot.Regular, "a"))
	first, firstFake := workspaceAdapter(t)
	second, secondFake := workspaceAdapter(t)
	if _, err := first.Materialize(context.Background(), run.WorkspaceSpec{ID: "run-a-work", Tree: tree}); err != nil {
		t.Fatal(err)
	}
	if _, err := second.Materialize(context.Background(), run.WorkspaceSpec{ID: "run-b-work", Tree: tree}); err != nil {
		t.Fatal(err)
	}
	names := make(map[string]bool)
	for _, spec := range append(slices.Clone(firstFake.helpers), secondFake.helpers...) {
		if names[spec.Name] || !strings.HasPrefix(spec.Name, "runtime-workspace-") {
			t.Fatalf("helper name %q repeats or lost its prefix", spec.Name)
		}
		names[spec.Name] = true
	}
	if len(names) < 2 {
		t.Fatal("no helpers observed")
	}
}

func TestWorkspaceMaterializesExactArchiveAndReadback(t *testing.T) {
	w, f := workspaceAdapter(t)
	tree := workspaceTree(t, workspaceEntry(t, "lib/λ [a].go", snapshot.Regular, "\x00\xff\r\n"), workspaceEntry(t, "bin/run", snapshot.Executable, "#!/bin/sh\n"), workspaceEntry(t, "empty", snapshot.Regular, ""))
	got, err := w.Materialize(context.Background(), run.WorkspaceSpec{ID: "step / arbitrary UTF8 λ", Tree: tree})
	if err != nil || got.Uncertain || got.Tree != tree.Digest() || got.Volume == "" {
		t.Fatalf("materialize %+v %v", got, err)
	}
	if !f.trees[got.Volume].Equal(tree) {
		t.Fatal("archive changed exact content or owner execute mode")
	}
	for _, spec := range f.helpers {
		if spec.Image != w.helper.Image || spec.User != "1000:1000" || spec.CPUs != 1 || spec.MemoryBytes != 1<<28 || spec.Pids != 32 || spec.TmpBytes != 1<<20 || spec.Network != "none" || len(spec.Mounts) != 1 || spec.Mounts[0].Target != "/work" {
			t.Fatalf("lost trusted helper configuration: %+v", spec)
		}
	}
	before := len(f.helpers)
	if _, err = w.Materialize(context.Background(), run.WorkspaceSpec{ID: got.ID, Tree: snapshot.Tree{}}); err == nil || len(f.helpers) != before {
		t.Fatal("existing volume was overwritten")
	}
	h, err := w.Harvest(context.Background(), run.HarvestRequest{ID: got.ID, MaxEntries: 10, MaxBytes: 100})
	if err != nil || !h.Complete() || !h.Tree.Equal(tree) {
		t.Fatalf("harvest %+v %v", h, err)
	}
	closed, err := w.Close(context.Background(), got.ID)
	if err != nil || !closed.Closed {
		t.Fatalf("close %+v %v", closed, err)
	}
	closed, err = w.Close(context.Background(), got.ID)
	if err != nil || !closed.Closed || f.removals != 1 {
		t.Fatal("close not idempotent")
	}
}
func TestWorkspaceMaterializesEmptyDirectoriesWithoutChangingTheTree(t *testing.T) {
	w, f := workspaceAdapter(t)
	tree := workspaceTree(t, workspaceEntry(t, "go.mod", snapshot.Regular, "module m\n"), workspaceEntry(t, "lending/keep.go", snapshot.Regular, "package lending\n"))
	got, err := w.Materialize(context.Background(), run.WorkspaceSpec{ID: "w", Tree: tree, Directories: []string{"lending/reminders", "library", "lending/reminders", "lending"}})
	if err != nil || got.Uncertain || got.Tree != tree.Digest() {
		t.Fatalf("materialize %+v %v", got, err)
	}
	if want := []string{"lending", "lending/reminders", "library"}; !slices.Equal(f.directories[got.Volume], want) {
		t.Fatalf("directories %q, want %q with ancestors, parents first and no duplicates", f.directories[got.Volume], want)
	}
	h, err := w.Harvest(context.Background(), run.HarvestRequest{ID: "w", MaxEntries: 10, MaxBytes: 100})
	if err != nil || !h.Complete() || !h.Tree.Equal(tree) {
		t.Fatalf("empty directories changed the harvested tree: %+v %v", h, err)
	}
}

func TestWorkspaceRefusesInvalidDirectoriesBeforeCreatingAnything(t *testing.T) {
	tree := workspaceTree(t, workspaceEntry(t, "docs", snapshot.Regular, "file"), workspaceEntry(t, "a/b", snapshot.Regular, "file"))
	for name, directory := range map[string]string{
		"empty": "", "absolute": "/lending", "parent": "../lending", "dot": "lending/./x", "double slash": "lending//x", "trailing slash": "lending/",
		"backslash": `lending\x`, "git": "lending/.git", "state": ".markitect/state", "state child": ".markitect/state/x",
		"same as a file": "docs", "below a file": "docs/x", "deep below a file": "a/b/c/d",
	} {
		t.Run(name, func(t *testing.T) {
			w, f := workspaceAdapter(t)
			got, err := w.Materialize(context.Background(), run.WorkspaceSpec{ID: "w", Tree: tree, Directories: []string{"ok", directory}})
			if err == nil || len(f.volumes) != 0 || len(f.helpers) != 0 || got.Uncertain {
				t.Fatalf("invalid directory accepted or left effects: %+v %v volumes=%d helpers=%d", got, err, len(f.volumes), len(f.helpers))
			}
		})
	}
}

func TestWorkspaceRetainsPartialAndUncertainVolumes(t *testing.T) {
	for _, test := range []string{"creation response lost", "write failed", "readback mismatch", "missing terminal frame", "helper failed", "mount appeared"} {
		t.Run(test, func(t *testing.T) {
			w, f := workspaceAdapter(t)
			switch test {
			case "creation response lost":
				f.createErr = errors.New(test)
			case "write failed":
				f.writeErr = errors.New(test)
			case "readback mismatch":
				f.corrupt = true
			case "missing terminal frame":
				f.truncated = true
			case "helper failed":
				f.censusErr = errors.New(test)
			case "mount appeared":
				f.startMount = true
			}
			got, err := w.Materialize(context.Background(), run.WorkspaceSpec{ID: "w", Tree: workspaceTree(t, workspaceEntry(t, "a", snapshot.Regular, "contents"))})
			if err == nil || !got.Uncertain || got.Reason == "" || f.removals != 0 || !f.volumes[got.Volume].Exists {
				t.Fatalf("failed materialization lost retained effects %+v %v", got, err)
			}
			if test != "creation response lost" && test != "write failed" && f.trees[got.Volume].EntryCount() != 1 {
				t.Fatal("failure did not exercise retention after materialization wrote content")
			}
		})
	}
}
func TestWorkspaceRefusesImageCopyUpBeforeExtraction(t *testing.T) {
	w, f := workspaceAdapter(t)
	f.initialTree = workspaceTree(t, workspaceEntry(t, "preexisting", snapshot.Regular, "image-owned bytes"))
	result, err := w.Materialize(context.Background(), run.WorkspaceSpec{ID: "w", Tree: workspaceTree(t, workspaceEntry(t, "preexisting", snapshot.Regular, "replacement"))})
	if err == nil || !result.Uncertain || !f.trees[result.Volume].Equal(f.initialTree) || f.removals != 0 {
		t.Fatalf("image contents were overwritten or discarded: %+v %v", result, err)
	}
	if len(f.helpers) != 1 || !f.helpers[0].Mounts[0].ReadOnly {
		t.Fatal("extraction ran before the empty-volume observation")
	}
}

func TestWorkspacePeekObservesMountedWorkspace(t *testing.T) {
	w, f := workspaceAdapter(t)
	ctx := context.Background()
	tree := workspaceTree(t, workspaceEntry(t, "a", snapshot.Regular, "contents"))
	if _, err := w.Materialize(ctx, run.WorkspaceSpec{ID: "w", Tree: tree}); err != nil {
		t.Fatal(err)
	}
	f.mounts = []string{"running-step"}
	h, err := w.Peek(ctx, run.HarvestRequest{ID: "w", MaxEntries: 10, MaxBytes: 100})
	if err != nil || !h.Complete() || !h.Tree.Equal(tree) {
		t.Fatalf("peek %+v %v", h, err)
	}
	if _, err = w.Peek(ctx, run.HarvestRequest{ID: "unknown", MaxEntries: 10, MaxBytes: 100}); err == nil {
		t.Fatal("peek of an absent workspace succeeded")
	}
}

func TestWorkspaceHarvestRefusesMountedAndCloseRefusesUnowned(t *testing.T) {
	w, f := workspaceAdapter(t)
	ctx := context.Background()
	got, err := w.Materialize(ctx, run.WorkspaceSpec{ID: "w"})
	if err != nil {
		t.Fatal(err)
	}
	f.mounts = []string{"stopped-but-still-mounted"}
	before := len(f.helpers)
	h, err := w.Harvest(ctx, run.HarvestRequest{ID: "w", MaxEntries: 1, MaxBytes: 1})
	if err == nil || !h.Uncertain || len(f.helpers) != before {
		t.Fatal("mounted workspace read")
	}
	closed, err := w.Close(ctx, "w")
	if err == nil || closed.Closed || f.removals != 0 {
		t.Fatal("mounted workspace removed")
	}
	f.mounts = nil
	observed := f.volumes[got.Volume]
	observed.Labels[workspaceOwnerLabel] = "different-owner"
	f.volumes[got.Volume] = observed
	closed, err = w.Close(ctx, "w")
	if err == nil || closed.Closed || f.removals != 0 {
		t.Fatal("replaced volume removed")
	}
	other := w.volume("preexisting")
	f.volumes[other] = container.VolumeObservation{Name: other, Exists: true, Labels: map[string]string{workspaceOwnerLabel: "preexisting"}}
	closed, err = w.Close(ctx, "preexisting")
	if err == nil || closed.Closed || f.removals != 0 {
		t.Fatal("preexisting volume removed")
	}
}
func TestWorkspaceCloseReportsObservedAbsenceAndUncertainty(t *testing.T) {
	w, f := workspaceAdapter(t)
	ctx := context.Background()
	if _, err := w.Materialize(ctx, run.WorkspaceSpec{ID: "w"}); err != nil {
		t.Fatal(err)
	}
	f.removeNoop = true
	closed, err := w.Close(ctx, "w")
	if err == nil || closed.Closed || !closed.Uncertain {
		t.Fatal("removal without absence reported closed")
	}
	f.removeNoop = false
	f.removeErr = errors.New("response lost")
	closed, err = w.Close(ctx, "w")
	if err == nil || !closed.Closed || !closed.Uncertain {
		t.Fatal("observed absence or failed response discarded")
	}
}
func TestWorkspaceCensusPreservesUnsupportedAndBounds(t *testing.T) {
	var data bytes.Buffer
	frames := []struct {
		path, kind, body string
		execute          bool
	}{{"dir", "directory", "", false}, {"dir/regular", "file", "raw\x00", false}, {"run", "file", "x", true}, {"link", "unsupported", "", true}, {"bad\\path", "file", "hidden", false}, {"bad\xff", "file", "bytes", false}, {"empty", "directory", "", false}}
	for _, f := range frames {
		if err := writeWorkspaceFrame(&data, workspaceFrame{Path: base64.StdEncoding.EncodeToString([]byte(f.path)), Type: f.kind, Size: int64(len(f.body)), Execute: f.execute}); err != nil {
			t.Fatal(err)
		}
		data.WriteString(f.body)
	}
	_ = writeWorkspaceFrame(&data, workspaceFrame{End: true})
	request := run.HarvestRequest{ID: "w", MaxEntries: 20, MaxBytes: 100}
	got, err := readWorkspaceCensus(bytes.NewReader(data.Bytes()), request)
	if err != nil || got.Uncertain || got.Tree.EntryCount() != 2 || len(got.Unsupported) != 3 || got.Unsupported[0] != "bad\\path" || got.Unsupported[1] != "bytes:626164ff" || got.Unsupported[2] != "link" {
		t.Fatalf("census %+v %v", got, err)
	}
	expected := workspaceTree(t, workspaceEntry(t, "dir/regular", snapshot.Regular, "raw\x00"), workspaceEntry(t, "run", snapshot.Executable, "x"))
	if !got.Tree.Equal(expected) {
		t.Fatal("regular content or mode changed")
	}
	for _, request := range []run.HarvestRequest{{ID: "w", MaxEntries: 1, MaxBytes: 100}, {ID: "w", MaxEntries: 20, MaxBytes: 2}} {
		got, err = readWorkspaceCensus(bytes.NewReader(data.Bytes()), request)
		if err == nil || !got.Uncertain {
			t.Fatal("bounded census succeeded")
		}
	}
	for _, p := range []string{".git/HEAD", "nested/.git/index", ".markitect/state/a"} {
		if !workspaceExcluded(p, false) {
			t.Fatal("missing exclusion", p)
		}
	}
	if workspaceExcluded(".markitect/state", false) || !workspaceExcluded(".markitect/state", true) || workspaceExcluded("a/.markitect/state/a", false) {
		t.Fatal("state exclusion changed scope")
	}
}
