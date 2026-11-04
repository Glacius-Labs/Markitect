// Package hazmat holds the hazard classifications of materials, the hazard
// gate they impose on purchases, and the operations on them.
package hazmat

import (
	"makerspace/equipment"
	"makerspace/materials"
	"makerspace/policies"
	"makerspace/training"
)

// Classification is the hazard classification of one material.
type Classification struct {
	MaterialID, FamilyID string
	MonthlyLimit         int
}

// Module is the hazmat module: it owns every classification.
type Module struct {
	equipment *equipment.Module
	materials *materials.Module
	training  *training.Module
	byMat     map[string]*Classification
}

// New returns the module in the initial state: no classification.
func New(eq *equipment.Module, mat *materials.Module, tr *training.Module) *Module {
	return &Module{equipment: eq, materials: mat, training: tr, byMat: map[string]*Classification{}}
}

// HazardRefusal gives the refusal code of the hazard gate that applies to a
// purchase of the quantity of the material by the member at the request time
// at, or "" when none applies.
func (m *Module) HazardRefusal(memberID, materialID string, quantity, at int) string {
	cl := m.byMat[materialID]
	if cl == nil {
		return ""
	}
	if !m.training.Certified(memberID, cl.FamilyID, policies.DayOf(at)) {
		return "hazard_not_certified"
	}
	if m.materials.UnitsBought(memberID, materialID, policies.MonthOf(at))+quantity > cl.MonthlyLimit {
		return "hazard_limit_exceeded"
	}
	return ""
}

func view(cl *Classification) policies.Rec {
	return policies.Rec{"materialId": cl.MaterialID, "familyId": cl.FamilyID, "monthlyLimit": cl.MonthlyLimit}
}
