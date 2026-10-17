package run

import (
	"errors"
	"slices"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var ErrInventoryUnavailable = errors.New("recorded basis has no complete Definition inventory")

// RecordedModelInventory is a historical producer projection, not a compiled
// model.Result. It retains identities, own Package inventory and source/content
// bindings, but no Definition bodies, contracts, semantic verdict or task grant.
type RecordedModelInventory struct {
	ID, Digest  string
	Local       bool
	Manifest    model.Source
	Packages    []model.Package
	Definitions []model.Identity
}

// RecordedInventory is bound to the original basis bytes. V1 has no complete
// imported inventory; even an empty Models result cannot be inferred for it.
type RecordedInventory struct {
	basis  BlobRef
	models []RecordedModelInventory
}

func (i RecordedInventory) Basis() BlobRef { return i.basis }

// Models returns owned copies. An unavailable v1 inventory is distinct from a
// v2 model's explicitly complete empty Definitions list. The existing 64 MiB
// payload ceiling bounds this view; Restore additionally charges the whole basis
// against its explicit read budget. Neither accessor compiles or reads sources.
func (i RecordedInventory) Models() ([]RecordedModelInventory, error) {
	if !i.basis.Valid() {
		return nil, ErrInvalid
	}
	if i.models == nil {
		return nil, ErrInventoryUnavailable
	}
	out := slices.Clone(i.models)
	for n := range out {
		out[n].Packages = slices.Clone(out[n].Packages)
		out[n].Definitions = slices.Clone(out[n].Definitions)
	}
	return out, nil
}

// BindingInventory validates the same closed record as BindingOwnerRef and
// Restore, then exposes only recorded inventory facts. Identity/shape agreement
// is not independent proof of the originally supplied compiler's completeness
// or semantic correctness. Original compilation and acquisition remain producer
// assertions; no current compiler, private canonical decoder or settings runs.
// This adds no stored references or Journal retention obligations.
func BindingInventory(payload Payload) (RecordedInventory, error) {
	w, _, err := decodeBindingRecord(payload)
	if err != nil {
		return RecordedInventory{}, err
	}
	i := RecordedInventory{basis: payload.Ref()}
	if w.Version == 2 {
		i.models = make([]RecordedModelInventory, len(w.Compilation))
		for n, scope := range w.Compilation {
			m := RecordedModelInventory{ID: scope.ID, Digest: scope.Digest, Local: scope.Local, Manifest: scope.Manifest, Packages: slices.Clone(scope.Packages), Definitions: make([]model.Identity, len(*scope.Definitions))}
			for d, definition := range *scope.Definitions {
				m.Definitions[d] = definition.Identity
			}
			i.models[n] = m
		}
	}
	return i, nil
}

// recordedManagers checks only the stored topology's agreement with its own
// inventories. It does not resolve organization settings or manufacture a
// partial model.Result marked Passed. V1 takes local identities from Managers;
// v2 independently checks them against the complete recorded local inventory.
func recordedManagers(w bindingWire) ([]organization.Manager, error) {
	local := -1
	seenModels := map[string]bool{}
	seenDefinitions := map[model.Identity]bool{}
	for n, scope := range w.Compilation {
		if scope.Local {
			if local != -1 || !text(scope.ID) || !text(scope.Digest) {
				return nil, ErrInvalid
			}
			local = n
		}
		if w.Version == 1 {
			if scope.Definitions != nil {
				return nil, ErrInvalid
			}
			continue
		}
		if !text(scope.ID) || seenModels[scope.ID] || scope.Definitions == nil || *scope.Definitions == nil {
			return nil, ErrInvalid
		}
		seenModels[scope.ID] = true
		packages := map[string]bool{}
		previous := ""
		for _, p := range scope.Packages {
			if !recordedPackage(p.Name) || p.Name <= previous || p.Source.Origin != scope.Manifest.Origin || !text(p.Source.Path) || p.Source.Digest != "" {
				return nil, ErrInvalid
			}
			packages[p.Name], previous = true, p.Name
		}
		for name := range packages {
			if parent := inventoryParent(name); parent != organization.Root && !packages[parent] {
				return nil, ErrInvalid
			}
		}
		var previousIdentity model.Identity
		for d, definition := range *scope.Definitions {
			id := definition.Identity
			_, pathErr := snapshot.NewPath(id.ID)
			if definition.ModelID != scope.ID || !packages[id.Package] || !text(id.ID) || strings.ContainsAny(id.ID, "/\\:") || pathErr != nil || seenDefinitions[id] || d > 0 && !inventoryLess(previousIdentity, id) {
				return nil, ErrInvalid
			}
			seenDefinitions[id], previousIdentity = true, id
		}
	}
	if local == -1 {
		return nil, ErrInvalid
	}
	scope := w.Compilation[local]
	packages := slices.Clone(scope.Packages)
	sort.Slice(packages, func(i, j int) bool { return packages[i].Name < packages[j].Name })
	managers := []organization.Manager{{Package: organization.Root}}
	index := map[string]int{organization.Root: 0}
	for _, p := range packages {
		if !recordedPackage(p.Name) {
			return nil, ErrInvalid
		}
		if _, exists := index[p.Name]; exists {
			return nil, ErrInvalid
		}
		index[p.Name] = len(managers)
		managers = append(managers, organization.Manager{Package: p.Name, Parent: inventoryParent(p.Name)})
	}
	for _, m := range managers[1:] {
		parent, exists := index[m.Parent]
		if !exists {
			return nil, ErrInvalid
		}
		managers[parent].Children = append(managers[parent].Children, m.Package)
	}
	var definitions []model.Identity
	if w.Version == 2 {
		for _, d := range *scope.Definitions {
			definitions = append(definitions, d.Identity)
		}
	} else {
		for _, m := range w.Managers {
			definitions = append(definitions, m.Definitions...)
		}
	}
	seen := map[model.Identity]bool{}
	for _, id := range definitions {
		n, exists := index[id.Package]
		if !exists || id.Package == organization.Root || id.ID == "" || seen[id] {
			return nil, ErrInvalid
		}
		seen[id] = true
		managers[n].Definitions = append(managers[n].Definitions, id)
	}
	for n := range managers {
		slices.SortFunc(managers[n].Definitions, func(a, b model.Identity) int { return strings.Compare(a.ID, b.ID) })
	}
	return managers, nil
}

func inventoryLess(a, b model.Identity) bool {
	return a.Package < b.Package || a.Package == b.Package && a.ID < b.ID
}

func recordedPackage(name string) bool {
	if name == "" || strings.ContainsAny(name, "\\:\x00") {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func inventoryParent(name string) string {
	if slash := strings.LastIndexByte(name, '/'); slash >= 0 {
		return name[:slash]
	}
	return organization.Root
}
