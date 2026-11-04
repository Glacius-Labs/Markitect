// Package stocktake holds the counts of the stock of the shop, the freeze they
// put on counted materials, and the operations on them.
package stocktake

import (
	"sort"

	"makerspace/ledger"
	"makerspace/materials"
	"makerspace/policies"
)

type count struct {
	materialID string
	counted    int
	difference *int
}

type stocktake struct {
	id     string
	closed bool
	counts map[string]*count
}

// Module is the stocktake module: it owns every stocktake.
type Module struct {
	materials *materials.Module
	ledger    *ledger.Module
	ids       policies.IDs
	byID      map[string]*stocktake
	list      []*stocktake
}

// New returns the module in the initial state.
func New(mat *materials.Module, led *ledger.Module) *Module {
	return &Module{materials: mat, ledger: led, byID: map[string]*stocktake{}}
}

// StocktakeExists reports whether the identifier names a stocktake.
func (m *Module) StocktakeExists(id string) bool { _, ok := m.byID[id]; return ok }

// Counted reports whether the material has a count in an open stocktake.
func (m *Module) Counted(materialID string) bool {
	for _, s := range m.list {
		if !s.closed && s.counts[materialID] != nil {
			return true
		}
	}
	return false
}

func (m *Module) open() *stocktake {
	for _, s := range m.list {
		if !s.closed {
			return s
		}
	}
	return nil
}

func (s *stocktake) sortedCounts() []*count {
	out := make([]*count, 0, len(s.counts))
	for _, c := range s.counts {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].materialID < out[j].materialID })
	return out
}

func view(s *stocktake) policies.Rec {
	status := "open"
	if s.closed {
		status = "closed"
	}
	counts := []policies.Rec{}
	for _, c := range s.sortedCounts() {
		var diff any
		if c.difference != nil {
			diff = *c.difference
		}
		counts = append(counts, policies.Rec{"materialId": c.materialID, "counted": c.counted, "difference": diff})
	}
	return policies.Rec{"stocktakeId": s.id, "status": status, "counts": counts}
}
