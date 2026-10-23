package gitrepo

import (
	"context"
	"fmt"
	"maps"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// LinuxCheckCopies creates one independent, verified Linux input/home volume
// for publication checks. It opens no Run or Manager and invokes no Git. Host
// Git has already reconstructed the fixed tree from durable storage.
type LinuxCheckCopies struct {
	Engine           *container.Engine
	Spec             container.ContainerSpec
	MaxTransferBytes int64
}

func (m *LinuxCheckCopies) Materialize(ctx context.Context, tree snapshot.Tree, dirs []snapshot.Path) (run.DisposableCopy, error) {
	if m == nil || m.Engine == nil || m.MaxTransferBytes <= 0 || m.MaxTransferBytes > 1<<30 {
		return nil, fmt.Errorf("publication check copy needs engine and finite transfer bound")
	}
	key, err := token()
	if err != nil {
		return nil, err
	}
	labels := maps.Clone(m.Spec.Labels)
	if labels == nil {
		labels = map[string]string{}
	}
	labels["markitect.publication-copy"] = key
	adapter := &LinuxGit{Engine: m.Engine, Spec: m.Spec, MaxTransferBytes: m.MaxTransferBytes}
	service := &linuxService{adapter: adapter, volume: "markitect-publication-" + key, labels: labels}
	// Return the partial handle on every subsequent error so Runner confirms
	// ownership and closure even when initialization or transfer failed.
	owned := &disposableCopy{location: run.CopyLocation{Volume: service.volume, Subpath: "checks/pending"}, close: service.Close}
	if err = m.Engine.RequireVolumeSubpaths(ctx); err != nil {
		return owned, err
	}
	if _, err = m.Engine.CreateVolumeExclusive(ctx, service.volume, labels); err != nil {
		return owned, err
	}
	spec := m.Spec
	spec.Name = "publication-copy-" + key
	spec.Labels = maps.Clone(labels)
	spec.Mounts = []container.Mount{{Volume: service.volume, Target: linuxRunRoot, NoCopy: true}}
	spec.Workdir = linuxRunRoot
	service.exec, err = m.Engine.StartOwnedExec(ctx, spec, []string{"/usr/local/bin/node", "-e", "setInterval(()=>{},2147483647)"})
	if err != nil {
		return owned, err
	}
	if service.exec == nil {
		return owned, fmt.Errorf("publication helper returned no handle")
	}
	materialized, err := (&linuxCopy{service: service}).Materialize(ctx, tree, dirs)
	if materialized != nil {
		owned.location = materialized.Location()
		owned.tree = materialized.Tree()
		owned.directories = materialized.Directories()
	}
	return owned, err
}
