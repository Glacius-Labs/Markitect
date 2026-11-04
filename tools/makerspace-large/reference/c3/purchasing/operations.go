package purchasing

import "makerspace/policies"

// Operations lists the operations of the module.
func (m *Module) Operations() []*policies.Operation {
	return []*policies.Operation{
		{Name: "addSupplier", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.ID("supplierId"), policies.Text("name")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				id := c.Str("supplierId")
				if m.suppliers[id] != nil {
					return nil, "", policies.Refuse("already_exists", "A supplier with identifier %s already exists.", id)
				}
				s := &supplier{id: id, name: c.Str("name"), status: "active"}
				m.suppliers[id] = s
				return policies.Rec{"supplier": supplierView(s)}, id, nil
			}},
		{Name: "setSupplierStatus", Mutation: true, Perms: "nna",
			Args: []policies.ArgSpec{policies.IDRef("supplierId", m.SupplierExists), policies.Enum("status", "active", "blocked")},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				s := m.suppliers[c.Str("supplierId")]
				s.status = c.Str("status")
				return policies.Rec{"supplier": supplierView(s)}, s.id, nil
			}},
		{Name: "createOrder", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("supplierId", m.SupplierExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				s := m.suppliers[c.Str("supplierId")]
				if s.status == "blocked" {
					return nil, "", policies.Refuse("supplier_blocked", "The supplier is blocked.")
				}
				o := &order{id: m.ids.Next("po"), supplierID: s.id, status: "draft", createdBy: c.ActorID}
				m.orders[o.id] = o
				m.orderList = append(m.orderList, o)
				return policies.Rec{"order": orderView(o)}, o.id, nil
			}},
		{Name: "addOrderLine", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("orderId", m.OrderExists),
				policies.IDRef("materialId", m.materials.MaterialExists), policies.Integer("quantity", 1, 100000),
				policies.Integer("unitCostCents", 1, 1000000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				o := m.orders[c.Str("orderId")]
				if o.status != "draft" {
					return nil, "", policies.Refuse("invalid_state", "The order is not a draft.")
				}
				mid := c.Str("materialId")
				if o.lineOf(mid) != nil {
					return nil, "", policies.Refuse("duplicate_line", "The order already has a line for %s.", mid)
				}
				o.lines = append(o.lines, &line{materialID: mid, quantity: c.Num("quantity"), unitCost: c.Num("unitCostCents")})
				return policies.Rec{"order": orderView(o)}, o.id, nil
			}},
		{Name: "submitOrder", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("orderId", m.OrderExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				o := m.orders[c.Str("orderId")]
				if o.status != "draft" {
					return nil, "", policies.Refuse("invalid_state", "The order is not a draft.")
				}
				if len(o.lines) == 0 {
					return nil, "", policies.Refuse("empty_order", "The order has no line.")
				}
				o.status = "submitted"
				return policies.Rec{"order": orderView(o)}, o.id, nil
			}},
		{Name: "approveOrder", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("orderId", m.OrderExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				o := m.orders[c.Str("orderId")]
				if o.status != "submitted" {
					return nil, "", policies.Refuse("invalid_state", "The order is not submitted.")
				}
				if o.createdBy == c.ActorID {
					return nil, "", policies.Refuse("self_approval", "The actor created the order.")
				}
				if o.total() > 100000 && c.ActorRole == "staff" {
					return nil, "", policies.Refuse("approval_requires_admin", "The order total is above 100000 cents.")
				}
				if m.suppliers[o.supplierID].status == "blocked" {
					return nil, "", policies.Refuse("supplier_blocked", "The supplier is blocked.")
				}
				o.status = "approved"
				return policies.Rec{"order": orderView(o)}, o.id, nil
			}},
		{Name: "receiveOrder", Mutation: true, Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("orderId", m.OrderExists),
				policies.IDRef("materialId", m.materials.MaterialExists), policies.Integer("quantity", 1, 100000)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				o := m.orders[c.Str("orderId")]
				if o.status != "approved" {
					return nil, "", policies.Refuse("invalid_state", "The order is not approved.")
				}
				mid := c.Str("materialId")
				l := o.lineOf(mid)
				if l == nil {
					return nil, "", policies.Refuse("not_on_order", "The order has no line for %s.", mid)
				}
				qty := c.Num("quantity")
				if l.rcvd+qty > l.quantity {
					return nil, "", policies.Refuse("over_receipt", "The receipt exceeds the ordered quantity.")
				}
				if m.materials.Counted(mid) {
					return nil, "", policies.Refuse("stocktake_open", "The material has a count in an open stocktake.")
				}
				l.rcvd += qty
				m.materials.AddStock(mid, qty)
				m.ledger.Post("inventory", "payable", qty*l.unitCost, o.id, c.At)
				done := true
				for _, x := range o.lines {
					if x.rcvd < x.quantity {
						done = false
					}
				}
				if done {
					o.status = "received"
				}
				return policies.Rec{"order": orderView(o)}, o.id, nil
			}},
		{Name: "listOrders", Perms: "naa",
			Args: []policies.ArgSpec{policies.IDRef("supplierId", m.SupplierExists)},
			Run: func(c *policies.Call) (policies.Rec, string, *policies.Refusal) {
				sid := c.Str("supplierId")
				out := []policies.Rec{}
				for _, o := range m.orderList {
					if o.supplierID == sid {
						out = append(out, orderView(o))
					}
				}
				return policies.Rec{"supplierId": sid, "orders": out}, "", nil
			}},
	}
}
