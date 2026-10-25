package ms_test

import "testing"

func TestBulkDiscountTiers(t *testing.T) {
	e := shop(t)
	now := "2026-03-02T09:00Z"
	e.ok("sam", now, "addMaterial", m{"materialId": "ply", "name": "Plywood", "unitPriceCents": 199})
	e.ok("sam", now, "addMaterial", m{"materialId": "bolt", "name": "Bolt", "unitPriceCents": 37})
	e.ok("sam", now, "restockMaterial", m{"materialId": "ply", "quantity": 3000})
	e.ok("sam", now, "restockMaterial", m{"materialId": "bolt", "quantity": 1000})
	cases := []struct {
		material      string
		quantity      int
		discount, amt int
	}{
		{"ply", 1, 0, 199}, {"ply", 9, 0, 1791}, {"ply", 10, 5, 1891}, {"ply", 24, 5, 4537},
		{"bolt", 25, 12, 814}, {"ply", 25, 12, 4378}, {"ply", 1000, 12, 175120},
	}
	for _, c := range cases {
		out := e.ok("ada", now, "purchaseMaterial", m{"memberId": "ada", "materialId": c.material, "quantity": c.quantity})
		p := obj(out["purchase"])
		if num(p["discountPercent"]) != c.discount || num(p["amountCents"]) != c.amt {
			t.Errorf("%s x %d: %v, want %d%% and %d", c.material, c.quantity, p, c.discount, c.amt)
		}
	}
}

func TestBulkDiscountIsPerPurchaseAndTierIndependent(t *testing.T) {
	for _, tier := range []string{"basic", "pro"} {
		e := shop(t)
		now := "2026-03-02T09:00Z"
		e.ok("sam", now, "changeTier", m{"memberId": "ada", "tier": tier})
		e.ok("sam", now, "addMaterial", m{"materialId": "ply", "name": "Plywood", "unitPriceCents": 199})
		e.ok("sam", now, "restockMaterial", m{"materialId": "ply", "quantity": 100})
		// separate purchases are never added together
		for i := 0; i < 3; i++ {
			p := obj(e.ok("ada", now, "purchaseMaterial", m{"memberId": "ada", "materialId": "ply", "quantity": 9})["purchase"])
			if num(p["discountPercent"]) != 0 || num(p["amountCents"]) != 1791 {
				t.Fatalf("%s: %v", tier, p)
			}
		}
		p := obj(e.ok("ada", now, "purchaseMaterial", m{"memberId": "ada", "materialId": "ply", "quantity": 10})["purchase"])
		if num(p["discountPercent"]) != 5 {
			t.Fatalf("%s: %v", tier, p)
		}
	}
}

func TestBulkDiscountOnInvoiceLine(t *testing.T) {
	e := shop(t)
	now := "2026-03-02T09:00Z"
	e.ok("sam", now, "addMaterial", m{"materialId": "ply", "name": "Plywood", "unitPriceCents": 199})
	e.ok("sam", now, "restockMaterial", m{"materialId": "ply", "quantity": 100})
	e.ok("ada", now, "purchaseMaterial", m{"memberId": "ada", "materialId": "ply", "quantity": 10})
	inv := obj(list(e.ok("sam", "2026-04-01T08:00Z", "generateInvoices", m{"month": "2026-03"})["invoices"])[0])
	line := obj(list(inv["lines"])[2])
	if line["kind"] != "material" || num(line["discountPercent"]) != 5 || num(line["amountCents"]) != 1891 || line["ref"] != "pur-1" {
		t.Fatalf("%v", line)
	}
	rep := e.ok("sam", "2026-04-01T08:00Z", "revenueReport", m{"month": "2026-03"})
	if num(rep["materialCents"]) != 1891 {
		t.Fatalf("%v", rep)
	}
}
