package apply

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const registryDirectory = ".markitect/registry/"

func registryOwner(path snapshot.Path) (realization.Owner, error) {
	p := path.String()
	if p == registryDirectory+"registry.yaml" {
		return realization.RootOwner(), nil
	}
	if strings.HasPrefix(p, registryDirectory) && strings.HasSuffix(p, "/registry.yaml") {
		return realization.PackageOwner(strings.TrimSuffix(strings.TrimPrefix(p, registryDirectory), "/registry.yaml"))
	}
	return realization.Owner{}, fmt.Errorf("not a registry shard path: %q", p)
}

func prepareRegistry(checked model.Result, raw RawInputs) registryPreparation {
	r := registryPreparation{status: model.Passed}
	if checked.Status != model.Passed {
		r.add(model.NotCompleted, "registry.model_not_passed", snapshot.Path{}, nil, "Registry preparation requires a passed bound model stage.")
		return r
	}
	topology, err := organization.BuildTopology(checked)
	if err != nil {
		r.add(model.Failed, "registry.model_invalid", snapshot.Path{}, nil, err.Error())
		return r
	}
	actual := make(map[snapshot.Path]snapshot.Entry)
	for _, entry := range raw.Census.Entries() {
		if !strings.HasPrefix(entry.Path().String(), registryDirectory) {
			continue
		}
		owner, err := registryOwner(entry.Path())
		if err != nil {
			r.add(model.Failed, "registry.layout", entry.Path(), nil, err.Error())
			continue
		}
		actual[entry.Path()] = entry
		r.discovered = append(r.discovered, entry.Path())
		packagePath := organization.Root
		if name, ok := owner.Package(); ok {
			packagePath = name
		}
		if _, found := topology.Manager(packagePath); !found {
			r.add(model.Failed, "registry.unknown_owner", entry.Path(), nil, "Registry shard belongs to a Package outside the current local inventory.")
		}
	}
	sources := slices.Clone(raw.Registry)
	slices.SortFunc(sources, func(a, b SourceFile) int {
		return strings.Compare(a.RepositoryPath.String(), b.RepositoryPath.String())
	})
	seen := make(map[snapshot.Path]bool)
	var registrations []realization.Registration
	for _, source := range sources {
		path := source.RepositoryPath
		if seen[path] {
			r.add(model.Failed, "registry.duplicate_source", path, nil, "A registry shard was captured more than once.")
			continue
		}
		seen[path] = true
		owner, err := registryOwner(path)
		if err != nil {
			r.add(model.Failed, "registry.source_path", path, nil, err.Error())
			continue
		}
		entry, found := actual[path]
		if !found {
			r.add(model.NotCompleted, "registry.source_missing", path, nil, "The selected registry shard is absent from Census.")
			continue
		}
		if entry.Mode() != source.File.Mode() || !bytes.Equal(entry.Bytes(), source.File.Bytes()) {
			r.add(model.Failed, "registry.source_mismatch", path, nil, "Registry source bytes/mode differ from Census.")
			continue
		}
		registrations = append(registrations, r.decodeRegistry(source, owner, checked, raw.Census)...)
	}
	for _, path := range r.discovered {
		if !seen[path] {
			r.add(model.NotCompleted, "registry.source_uncaptured", path, nil, "A Census registry shard has no captured source observation.")
		}
	}
	registry, err := realization.NewRegistry(registrations)
	if err != nil {
		r.add(model.Failed, "registry.records_invalid", snapshot.Path{}, nil, err.Error())
	}
	if r.status == model.Passed {
		r.registry = registry
	}
	slices.SortFunc(r.diagnostics, func(a, b RegistryDiagnostic) int {
		if d := strings.Compare(a.Source.String(), b.Source.String()); d != 0 {
			return d
		}
		ap, bp := model.Position{}, model.Position{}
		if a.Position != nil {
			ap = *a.Position
		}
		if b.Position != nil {
			bp = *b.Position
		}
		if ap.Line != bp.Line {
			return ap.Line - bp.Line
		}
		if ap.Column != bp.Column {
			return ap.Column - bp.Column
		}
		if d := strings.Compare(a.Code, b.Code); d != 0 {
			return d
		}
		return strings.Compare(a.Message, b.Message)
	})
	return r
}
