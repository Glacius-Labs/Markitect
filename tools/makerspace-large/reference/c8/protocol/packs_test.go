package protocol_test

import "testing"

func TestSellAndListPacks(t *testing.T) {
	e := shop(t)
	at := "2026-03-02T09:00Z"
	prod := obj(e.ok("admin", at, "addPackProduct", m{"productId": "p60", "name": "Hour", "minutes": 60, "priceCents": 2500, "validDays": 30})["product"])
	assertKeys(t, prod, "productId", "name", "minutes", "priceCents", "validDays")
	e.refused("admin", at, "addPackProduct", m{"productId": "p60", "name": "Hour", "minutes": 60, "priceCents": 2500, "validDays": 30}, "already_exists")
	sold := e.ok("sam", at, "sellPack", m{"memberId": "ada", "productId": "p60"})
	pack := obj(sold["pack"])
	assertKeys(t, pack, "packId", "memberId", "productId", "minutesLeft", "issuedOn", "expiresOn")
	if pack["packId"] != "pak-1" || pack["expiresOn"] != "2026-04-01" {
		t.Fatalf("%v", pack)
	}
	charge := obj(sold["charge"])
	if charge["kind"] != "pack" || charge["ref"] != "pak-1" {
		t.Fatalf("%v", charge)
	}
	e.refused("sam", at, "sellPack", m{"memberId": "ada", "productId": "nope"}, "not_found")
	e.ok("sam", at, "sellPack", m{"memberId": "ada", "productId": "p60"})
	packs := list(e.ok("sam", at, "listMemberPacks", m{"memberId": "ada"})["packs"])
	if len(packs) != 2 || obj(packs[1])["packId"] != "pak-2" {
		t.Fatalf("%v", packs)
	}
}
