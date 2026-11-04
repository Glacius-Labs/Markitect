# Change request: Prepaid minutes become packs

At the desk members keep mixing up their prepaid machine minutes with guest passes. From now on we call a block of prepaid machine minutes a **pack**. This is a renaming only: nothing is to behave differently. Guest passes keep their name and everything that belongs to them, and the words for passing an inspection or an assessment stay too.

| Now | From now on |
| --- | --- |
| operation `addPassProduct` | `addPackProduct` |
| operation `sellPass` | `sellPack` |
| operation `listMemberPasses` | `listMemberPacks` |
| field `passId` | `packId` |
| result `pass` of `sellPack` | `pack` |
| result `passes` of `listMemberPacks` | `packs` |
| charge kind `pass` | `pack` |
| ledger account `revenuePasses` | `revenuePacks` |
| identifiers `pas-1`, `pas-2`, … | `pak-1`, `pak-2`, …, the same counter |
| module `passes` | `packs` |

A pass product is now a pack product. Its `product` result and the fields `productId`, `name`, `minutes`, `priceCents` and `validDays` keep their names, as do `minutesLeft`, `issuedOn` and `expiresOn`. The old names are gone; there are no aliases.
