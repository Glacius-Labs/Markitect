package materials

import "makerspace/policies"

func materialView(m *material) policies.Rec {
	return policies.Rec{"materialId": m.id, "name": m.name, "unitPriceCents": m.unit, "stock": m.stock}
}

func purchaseView(p *purchase) policies.Rec {
	return policies.Rec{"purchaseId": p.id, "memberId": p.memberID, "materialId": p.materialID,
		"quantity": p.quantity, "unitPriceCents": p.unit, "discountPercent": p.discount,
		"amountCents": p.amount, "at": policies.FormatTime(p.at)}
}
