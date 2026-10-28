---
kind: std/Rule
---
# Authorization

Every user has exactly one role, fixed when the user is registered: member, staff or admin.

For each operation and role, the operation's permissions state one permission:

- **all:** the role may use the operation for any record.
- **own:** the role may use the operation only for records that belong to the actor.
- **none** (no permission listed for the role): the role may not use the operation.

A record concerned belongs to the actor when:

- the operation has a `memberId` argument and it equals the actor's identifier;
- the operation has a `reservationId` argument and the reservation's member is the actor;
- the operation has an `invoiceId` argument and the invoice's member is the actor.

A "none" permission is refused at check 4, an ownership violation at check 6 of the order of checks. Both use the code [[refusal-codes|`forbidden`]].

The roles:

- `member`: Members are the workshop's customers.
- `staff`: Staff run the front desk and the workshop floor.
- `admin`: Admins also manage the catalog, credits and users.
