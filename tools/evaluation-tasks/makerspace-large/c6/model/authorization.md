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
- the operation has an `invoiceId` argument and the invoice's member is the actor;
- the operation has a `loanId` argument and the loan's member is the actor;
- the operation has a `depositId`, `rentalId`, `guestPassId`, `badgeId`, `noticeId`, `ticketId` or `entryId` argument and that deposit, locker rental, guest pass, badge, notice, event ticket or volunteer time entry is the actor's;
- the operation has a `guardianId` argument and it equals the actor's identifier; where an operation has `guardianId`, ownership is judged on it alone;
- the operation has an `ownerId` argument and it equals the actor's identifier;
- the operation has a `projectId` argument and the project's owner is the actor; where an operation has `projectId`, ownership is judged on it alone, except create project, whose ownership is judged on `ownerId` alone;
- an operation that says on which of its arguments ownership is judged, such as `admitGuest` on `badgeId`, is judged on that argument alone.

A "none" permission is refused at check 4, an ownership violation at check 6 of the order of checks. Both use the code [[refusal-codes|`forbidden`]].

The roles:

- `member`: Members are the workshop's customers.
- `staff`: Staff run the front desk and the workshop floor.
- `admin`: Admins also manage the catalog, credits and users.
