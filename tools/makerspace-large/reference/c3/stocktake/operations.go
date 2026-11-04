package stocktake

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "startStocktake", Mutation: true, Perms: "naa",
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				if m.open() != nil {
					return nil, "", policies.Refuse("stocktake_running", "A stocktake is already open.")
				}
				s := &stocktake{id: m.ids.Next("stk"), counts: map[string]*count{}}
				m.byID[s.id] = s
				m.list = append(m.list, s)
				return policies.Rec{"stocktake": view(s)}, s.id, nil
			}},
		{Name: "recordCount", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("stocktakeId", m.StocktakeExists),
				policies.IDRef("materialId", m.materials.MaterialExists), policies.Integer("counted", 0, 100000000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				s := m.byID[c.Str("stocktakeId")]
				if s.closed {
					return nil, "", policies.Refuse("invalid_state", "The stocktake is closed.")
				}
				mid := c.Str("materialId")
				if ct := s.counts[mid]; ct != nil {
					ct.counted = c.Num("counted")
				} else {
					s.counts[mid] = &count{materialID: mid, counted: c.Num("counted")}
				}
				return policies.Rec{"stocktake": view(s)}, s.id + "/" + mid, nil
			}},
		{Name: "closeStocktake", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("stocktakeId", m.StocktakeExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				s := m.byID[c.Str("stocktakeId")]
				if s.closed {
					return nil, "", policies.Refuse("invalid_state", "The stocktake is closed.")
				}
				if len(s.counts) == 0 {
					return nil, "", policies.Refuse("nothing_counted", "The stocktake has no count.")
				}
				for _, ct := range s.sortedCounts() {
					diff := ct.counted - m.materials.Stock(ct.materialID)
					ct.difference = &diff
					m.materials.SetStock(ct.materialID, ct.counted)
					price := m.materials.UnitPrice(ct.materialID)
					switch {
					case diff < 0:
						m.ledger.Post("expenseWriteOff", "inventory", -diff*price, s.id, c.At)
					case diff > 0:
						m.ledger.Post("inventory", "expenseWriteOff", diff*price, s.id, c.At)
					}
				}
				s.closed = true
				return policies.Rec{"stocktake": view(s)}, s.id, nil
			}},
		{Name: "getStocktake", Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("stocktakeId", m.StocktakeExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				return policies.Rec{"stocktake": view(m.byID[c.Str("stocktakeId")])}, "", nil
			}},
	}
}
