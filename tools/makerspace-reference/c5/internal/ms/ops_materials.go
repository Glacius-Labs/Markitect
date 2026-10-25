package ms

func materialOps() []*opDef {
	return []*opDef{
		{name: "addMaterial", mutation: true, perms: "naan",
			args: []argSpec{id("materialId"), text("name"), integer("unitPriceCents", 1, 1000000)},
			run: func(c *call) (rec, string, *refusal) {
				mid := c.str("materialId")
				if c.s.materials[mid] != nil {
					return nil, "", refuse("already_exists", "A material with identifier %s already exists.", mid)
				}
				m := &material{id: mid, name: c.str("name"), unit: c.num("unitPriceCents")}
				c.s.materials[mid] = m
				return rec{"material": materialView(m)}, mid, nil
			}},
		{name: "restockMaterial", mutation: true, perms: "naan",
			args: []argSpec{idRef("materialId", refMaterial), integer("quantity", 1, 100000)},
			run: func(c *call) (rec, string, *refusal) {
				m := c.s.materials[c.str("materialId")]
				m.stock += c.num("quantity")
				return rec{"material": materialView(m)}, m.id, nil
			}},
		{name: "purchaseMaterial", mutation: true, perms: "oaan",
			args: []argSpec{idRef("memberId", refMember), idRef("materialId", refMaterial),
				integer("quantity", 1, 10000)},
			run: func(c *call) (rec, string, *refusal) {
				mem := c.member("memberId")
				mat := c.s.materials[c.str("materialId")]
				qty := c.num("quantity")
				if mem.standing != "good" {
					return nil, "", refuse("member_suspended", "The member is suspended.")
				}
				if c.s.blocked(mem.user.id, c.at) {
					return nil, "", refuse("account_overdue", "The member has an invoice that is more than 30 days overdue.")
				}
				if mat.stock < qty {
					return nil, "", refuse("insufficient_stock", "The stock is lower than the quantity.")
				}
				mat.stock -= qty
				p := &purchase{id: c.s.nextID("pur"), memberID: mem.user.id, materialID: mat.id,
					quantity: qty, unit: mat.unit, discount: bulkDiscount(qty), at: c.at}
				p.amount = chargeAmount(p.quantity, p.unit, p.discount)
				c.s.purchases = append(c.s.purchases, p)
				c.s.addCharge(mem.user.id, "material", p.id, p.quantity, p.unit, p.discount, c.at)
				return rec{"purchase": purchaseView(p), "material": materialView(mat)}, p.id, nil
			}},
		{name: "listMaterials", perms: "aaaa",
			run: func(c *call) (rec, string, *refusal) {
				out := []rec{}
				for _, k := range sortedKeys(c.s.materials) {
					out = append(out, materialView(c.s.materials[k]))
				}
				return rec{"materials": out}, "", nil
			}},
	}
}

// bulkDiscount is the discount percent of one purchase by its quantity.
func bulkDiscount(quantity int) int {
	switch {
	case quantity >= 25:
		return 12
	case quantity >= 10:
		return 5
	}
	return 0
}
