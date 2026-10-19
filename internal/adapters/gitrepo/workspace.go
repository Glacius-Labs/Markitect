package gitrepo

import (
	"archive/tar"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// WorkspaceRuntime is the container engine capability used by Workspaces.
// VolumeMounts includes every container, including stopped containers and
// read-only mounts. CreateVolumeExclusive must refuse an existing name and
// return observed labels even when creation has an uncertain outcome.
type WorkspaceRuntime interface {
	InspectVolume(context.Context, string) (container.VolumeObservation, error)
	CreateVolumeExclusive(context.Context, string, map[string]string) (container.VolumeObservation, error)
	VolumeMounts(context.Context, string) ([]string, error)
	RemoveVolume(context.Context, string) error
	RunHelper(context.Context, container.ContainerSpec, []string, io.Reader, io.Writer) error
}

const workspaceOwnerLabel = "markitect.workspace-owner"

// Workspaces realizes run.Workspaces using fresh runtime volumes. Ownership
// receipts stay in this adapter instance; reconstruction cannot claim an
// existing volume. Failed or uncertain operations retain their volume.
type Workspaces struct {
	mu      sync.Mutex
	runtime WorkspaceRuntime
	helper  container.ContainerSpec
	owned   map[run.WorkspaceID]string
	seq     uint64
}

var _ run.Workspaces = (*Workspaces)(nil)
var _ WorkspaceRuntime = (*container.Engine)(nil)

// NewWorkspaces binds the trusted helper image, numeric user and resource
// limits. The engine validates the complete spec before starting a helper.
// Mounts are supplied by this adapter; the helper gets only /work. Name and
// Labels identify this Host store and must remain stable for this instance.
// Construction performs no engine or filesystem operation.
func NewWorkspaces(runtime WorkspaceRuntime, helper container.ContainerSpec) (*Workspaces, error) {
	if runtime == nil || helper.Name == "" || helper.Image == "" || len(helper.Labels) == 0 || len(helper.Mounts) != 0 {
		return nil, fmt.Errorf("workspace runtime needs a named, labeled trusted helper without mounts")
	}
	helper.Labels = cloneWorkspaceLabels(helper.Labels)
	return &Workspaces{runtime: runtime, helper: helper, owned: make(map[run.WorkspaceID]string)}, nil
}

func cloneWorkspaceLabels(labels map[string]string) map[string]string {
	copy := make(map[string]string, len(labels))
	for key, value := range labels {
		copy[key] = value
	}
	return copy
}

func (w *Workspaces) volume(id run.WorkspaceID) string {
	digest := sha256.Sum256([]byte(w.helper.Name + "\x00" + string(id)))
	return "markitect-workspace-" + hex.EncodeToString(digest[:])
}

func (w *Workspaces) spec(volume string, readOnly bool) container.ContainerSpec {
	w.seq++
	spec := w.helper
	name := spec.Name
	if len(name) > 95 {
		name = name[:95]
	}
	spec.Name = name + "-workspace-" + strconv.FormatUint(w.seq, 10)
	spec.Labels = cloneWorkspaceLabels(spec.Labels)
	spec.Mounts = []container.Mount{{Volume: volume, Target: "/work", ReadOnly: readOnly}}
	spec.Workdir = "/work"
	return spec
}

func (w *Workspaces) Materialize(ctx context.Context, request run.WorkspaceSpec) (result run.WorkspaceObservation, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	result.ID = request.ID
	if !request.Valid() {
		return result, fmt.Errorf("invalid workspace spec")
	}
	result.Volume = w.volume(request.ID)
	for _, entry := range request.Tree.Entries() {
		if workspaceExcluded(entry.Path().String(), false) {
			return result, fmt.Errorf("workspace input contains excluded path %q", entry.Path())
		}
	}
	directories, err := workspaceDirectories(request)
	if err != nil {
		return result, err
	}
	var secret [32]byte
	if _, err = rand.Read(secret[:]); err != nil {
		return result, err
	}
	token := hex.EncodeToString(secret[:])
	labels := cloneWorkspaceLabels(w.helper.Labels)
	labels[workspaceOwnerLabel] = token
	observed, createErr := w.runtime.CreateVolumeExclusive(ctx, result.Volume, labels)
	if observed.Exists && observed.Name == result.Volume && observed.Labels[workspaceOwnerLabel] == token {
		w.owned[request.ID] = token
	}
	if createErr != nil {
		result.Uncertain, result.Reason = true, createErr.Error()
		return result, createErr
	}
	if w.owned[request.ID] != token {
		err = fmt.Errorf("runtime did not establish ownership of new workspace volume")
		result.Uncertain, result.Reason = true, err.Error()
		return result, err
	}
	if err = w.unmounted(ctx, result.Volume); err != nil {
		result.Uncertain, result.Reason = true, err.Error()
		return result, err
	}
	// The engine may initialize a fresh named volume from the image's /work.
	// Observe it before extraction so image content cannot be overwritten merely
	// because the volume name was freshly created.
	initial, initialErr := w.harvest(ctx, run.HarvestRequest{ID: request.ID, MaxEntries: 1 << 20, MaxBytes: 1 << 32}, true)
	if initialErr != nil || !initial.Complete() || initial.Tree.EntryCount() != 0 {
		err = errors.Join(initialErr, fmt.Errorf("new workspace volume was not observed empty before materialization"))
		result.Uncertain, result.Reason = true, err.Error()
		return result, err
	}
	reader, writer := io.Pipe()
	written := make(chan error, 1)
	go func() {
		archive := tar.NewWriter(writer)
		var writeErr error
		// Directories come first, each owned like the files and mode 0755.
		for _, directory := range directories {
			if writeErr = archive.WriteHeader(&tar.Header{Name: directory + "/", Mode: 0755, Typeflag: tar.TypeDir, Format: tar.FormatPAX}); writeErr != nil {
				break
			}
		}
		for _, entry := range request.Tree.Entries() {
			if writeErr != nil {
				break
			}
			mode := int64(0644)
			if entry.Mode() == snapshot.Executable {
				mode = 0755
			}
			writeErr = archive.WriteHeader(&tar.Header{Name: entry.Path().String(), Mode: mode, Size: entry.ByteLength(), Typeflag: tar.TypeReg, Format: tar.FormatPAX})
			if writeErr == nil {
				_, writeErr = archive.Write(entry.Bytes())
			}
			if writeErr != nil {
				break
			}
		}
		writeErr = errors.Join(writeErr, archive.Close())
		_ = writer.CloseWithError(writeErr)
		written <- writeErr
	}()
	helperErr := w.runtime.RunHelper(ctx, w.spec(result.Volume, false), []string{"/bin/tar", "--extract", "--preserve-permissions", "--no-same-owner", "--file", "-", "--directory", "/work"}, reader, io.Discard)
	_ = reader.CloseWithError(helperErr)
	if err = errors.Join(helperErr, <-written); err != nil {
		result.Uncertain, result.Reason = true, err.Error()
		return result, err
	}
	harvest, harvestErr := w.harvest(ctx, run.HarvestRequest{ID: request.ID, MaxEntries: 1 << 20, MaxBytes: 1 << 32}, true)
	if harvest.Complete() {
		result.Tree = harvest.Tree.Digest()
	}
	if harvestErr != nil || !harvest.Complete() || !request.Tree.Equal(harvest.Tree) {
		err = errors.Join(harvestErr, fmt.Errorf("workspace readback did not establish exact supplied tree"))
		result.Uncertain, result.Reason = true, err.Error()
		return result, err
	}
	return result, nil
}

// workspaceDirectories returns the requested empty directories and all their
// ancestors, parents first. A path must be a valid snapshot path outside the
// excluded Git and state paths and must not be, or lie below, a file of the tree.
func workspaceDirectories(request run.WorkspaceSpec) ([]string, error) {
	seen := map[string]bool{}
	for _, directory := range request.Directories {
		if _, err := snapshot.NewPath(directory); err != nil || workspaceExcluded(directory, true) {
			return nil, fmt.Errorf("workspace directory %q is not a valid included path", directory)
		}
		parts := strings.Split(directory, "/")
		for i := range parts {
			ancestor := strings.Join(parts[:i+1], "/")
			path, err := snapshot.NewPath(ancestor)
			if err != nil {
				return nil, err
			}
			if _, isFile := request.Tree.Lookup(path); isFile {
				return nil, fmt.Errorf("workspace directory %q collides with file %q of the tree", directory, ancestor)
			}
			seen[ancestor] = true
		}
	}
	directories := make([]string, 0, len(seen))
	for directory := range seen {
		directories = append(directories, directory)
	}
	slices.SortFunc(directories, func(a, b string) int { return strings.Compare(a, b) })
	return directories, nil
}

func (w *Workspaces) unmounted(ctx context.Context, volume string) error {
	mounts, err := w.runtime.VolumeMounts(ctx, volume)
	if err != nil {
		return err
	}
	if len(mounts) != 0 {
		return fmt.Errorf("workspace volume remains mounted by %d container(s)", len(mounts))
	}
	return nil
}

func (w *Workspaces) ownedVolume(ctx context.Context, id run.WorkspaceID) (string, container.VolumeObservation, error) {
	volume := w.volume(id)
	observed, err := w.runtime.InspectVolume(ctx, volume)
	if err != nil {
		return volume, observed, err
	}
	if observed.Name != volume {
		return volume, observed, fmt.Errorf("runtime returned a different volume identity")
	}
	if observed.Exists && (w.owned[id] == "" || observed.Labels[workspaceOwnerLabel] != w.owned[id]) {
		return volume, observed, fmt.Errorf("workspace volume ownership is not established")
	}
	return volume, observed, nil
}

func (w *Workspaces) Harvest(ctx context.Context, request run.HarvestRequest) (run.WorkspaceHarvest, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.harvest(ctx, request, true)
}

// Peek takes the Harvest census without requiring the volume to be unmounted.
func (w *Workspaces) Peek(ctx context.Context, request run.HarvestRequest) (run.WorkspaceHarvest, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.harvest(ctx, request, false)
}

func (w *Workspaces) harvest(ctx context.Context, request run.HarvestRequest, stopped bool) (result run.WorkspaceHarvest, err error) {
	result.ID = request.ID
	defer func() {
		if err != nil {
			result.Uncertain, result.Reason = true, err.Error()
		}
	}()
	if !request.Valid() {
		return result, fmt.Errorf("invalid workspace harvest request")
	}
	volume, observed, err := w.ownedVolume(ctx, request.ID)
	if err != nil {
		return result, err
	}
	if !observed.Exists {
		return result, fmt.Errorf("workspace volume is absent")
	}
	if stopped {
		if err = w.unmounted(ctx, volume); err != nil {
			return result, err
		}
	}
	helperCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	reader, writer := io.Pipe()
	done := make(chan error, 1)
	go func() {
		helperErr := w.runtime.RunHelper(helperCtx, w.spec(volume, true), []string{"/usr/local/bin/node", "-e", workspaceCensusScript, strconv.Itoa(request.MaxEntries), strconv.FormatInt(request.MaxBytes, 10)}, nil, writer)
		_ = writer.CloseWithError(helperErr)
		done <- helperErr
	}()
	result, err = readWorkspaceCensus(reader, request)
	if err != nil {
		cancel()
	}
	_ = reader.CloseWithError(err)
	err = errors.Join(err, <-done)
	// A census helper has ended before this second all-container observation.
	if stopped {
		err = errors.Join(err, w.unmounted(ctx, volume))
	}
	_, after, inspectErr := w.ownedVolume(ctx, request.ID)
	if inspectErr == nil && !after.Exists {
		inspectErr = fmt.Errorf("workspace volume disappeared during census")
	}
	err = errors.Join(err, inspectErr)
	return result, err
}

func (w *Workspaces) Close(ctx context.Context, id run.WorkspaceID) (result run.WorkspaceObservation, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	result.ID = id
	defer func() {
		if err != nil {
			result.Uncertain, result.Reason = true, err.Error()
		}
	}()
	if !(run.WorkspaceSpec{ID: id}).Valid() {
		return result, fmt.Errorf("invalid workspace ID")
	}
	volume, observed, err := w.ownedVolume(ctx, id)
	result.Volume = volume
	if err != nil {
		return result, err
	}
	if !observed.Exists {
		delete(w.owned, id)
		result.Closed = true
		return result, nil
	}
	if err = w.unmounted(ctx, volume); err != nil {
		return result, err
	}
	removeErr := w.runtime.RemoveVolume(ctx, volume)
	after, inspectErr := w.runtime.InspectVolume(ctx, volume)
	if inspectErr == nil && after.Name == volume && !after.Exists {
		delete(w.owned, id)
		result.Closed = true
		// Preserve a failed removal response even when absence is established.
		return result, removeErr
	}
	err = errors.Join(removeErr, inspectErr)
	if err == nil {
		err = fmt.Errorf("workspace volume absence was not established")
	}
	return result, err
}
