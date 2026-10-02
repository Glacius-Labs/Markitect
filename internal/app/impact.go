package app

import (
	"sort"

	"github.com/Glacius-Labs/Markitect/internal/render"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type Impact struct {
	Base      string   `yaml:"base"`
	Candidate string   `yaml:"candidate"`
	Changed   []string `yaml:"changed"`
	Affected  []string `yaml:"affected"`
	Reason    string   `yaml:"reason"`
}

func Changes(before, after *Project) *Impact {
	result := &Impact{Base: before.Snapshot.ID, Candidate: after.Snapshot.ID, Reason: "Union of old and new dependency closures; configuration or inventory changes conservatively affect all resources."}
	result.Changed = snapshot.Compare(before.Snapshot, after.Snapshot).Paths()
	changed := map[string]bool{}
	for _, name := range result.Changed {
		changed[name] = true
	}
	all := false
	seeds := map[string]bool{}
	ownedInputs := map[string]map[string]bool{}
	for _, p := range []*Project{before, after} {
		for _, r := range p.Resources {
			if r.Package == "" && changed[r.Path] {
				seeds[r.GraphKey()] = true
				if r.Kind == "Project" {
					all = true
				}
			}
		}
		fileOwners, err := impactFileOwners(p)
		if err != nil {
			all = true
			continue
		}
		for file, owners := range fileOwners {
			for owner := range owners {
				if ownedInputs[file] == nil {
					ownedInputs[file] = map[string]bool{}
				}
				ownedInputs[file][owner] = true
			}
		}
	}
	// New/deleted symbols may affect global constraints and scope-level discovery.
	if len(before.Graph.Resources) != len(after.Graph.Resources) {
		all = true
	}
	for k := range before.Graph.Resources {
		if _, ok := after.Graph.Resources[k]; !ok {
			all = true
		}
	}
	for k := range after.Graph.Resources {
		if _, ok := before.Graph.Resources[k]; !ok {
			all = true
		}
	}
	// Unmodelled inputs may contain normative contracts, router membership or
	// checker code. Until their independence is declared, never reuse evidence.
	modelled := map[string]bool{"markitect.yaml": true}
	for _, p := range []*Project{before, after} {
		for _, r := range p.Resources {
			if r.Package == "" {
				modelled[r.Path] = true
			}
		}
	}
	for name := range changed {
		if owners := ownedInputs[name]; len(owners) > 0 {
			for owner := range owners {
				seeds[owner] = true
			}
		} else if !modelled[name] {
			all = true
		}
	}
	if all {
		for _, p := range []*Project{before, after} {
			for k := range p.Graph.Resources {
				seeds[k] = true
			}
		}
	}
	for again := true; again; {
		again = false
		for _, p := range []*Project{before, after} {
			for from, edges := range p.Graph.Edges {
				if seeds[from] {
					continue
				}
				for _, to := range edges {
					if seeds[to] {
						seeds[from] = true
						again = true
						break
					}
				}
			}
		}
	}
	for k := range seeds {
		result.Affected = append(result.Affected, k)
	}
	sort.Strings(result.Affected)
	return result
}

// impactFileOwners maps only paths whose content is already represented by a
// typed resource or by an explicit Spec.Files declaration. Unknown generated
// files remain broad-impact inputs until their ownership is modeled.
func impactFileOwners(p *Project) (map[string]map[string]bool, error) {
	owners := map[string]map[string]bool{}
	add := func(file, key string) {
		if file == "" || key == "" {
			return
		}
		if owners[file] == nil {
			owners[file] = map[string]bool{}
		}
		owners[file][key] = true
	}
	if p == nil {
		return owners, nil
	}
	for _, r := range p.Resources {
		if r == nil || r.Kind == "Project" || r.Package != "" {
			continue
		}
		key := r.GraphKey()
		for _, file := range r.Spec.Files {
			add(file, key)
		}
		for _, file := range p.InputFiles[key] {
			add(file, key)
		}
	}
	_, generatedOwners, err := render.GenerateWithOwners(p.Graph, p.Snapshot.Files)
	if err != nil {
		return nil, err
	}
	for file, resources := range generatedOwners {
		for _, resource := range resources {
			add(file, resource)
		}
	}
	return owners, nil
}
