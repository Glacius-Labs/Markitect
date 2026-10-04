# Commerce module map

Commerce is a fictional .NET 8 modular monolith with three independently owned modules.

| Owner | Responsibility | Published contract |
|---|---|---|
| Orders | Order aggregate, placement, cancellation, and order reads | Order-facing contracts, if later needed |
| Inventory | Stock and reservations | `IStockAvailability` |
| Billing | Payment authorization and capture | `IPaymentAuthorizer` |

Orders may call Inventory and Billing through their public contracts. It must not read their storage or call their Application/Domain implementation. Core owns only stable shared abstractions, such as `IClock`. Commands and queries have separate Application slices; aggregate lifecycle rules stay in Orders Domain. Module ownership changes require an explicit human decision. The executable check validates selected behavior and source references, not runtime deployment or a complete architecture proof.

The initial policy selects exactly the `engineering/UseCase:create-order` resource, identified by `validation: required`, and requires selected resources to be Commands. Validators are permitted but optional. The proposed v2 policy adds a Validator requirement for that same unchanged cohort. A policy finding cannot be waived by relabeling or changing selection.
