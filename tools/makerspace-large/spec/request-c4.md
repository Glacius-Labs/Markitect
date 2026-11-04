# Change request: Pausing a membership

Members who go abroad for a few months ask to keep their place without paying the full fee. I want a third standing, `paused`.

- `setMemberStanding` also accepts `paused`. As today, any standing can be set from any other, and setting the current one succeeds without a change. Pausing cancels nothing.
- A paused member is not in good standing. Wherever a member who is not in good standing is refused today, a paused member is refused the same way with the same code, `member_suspended`, and what a suspended member can still do, a paused member can do too.
- The monthly fee of a paused member is 500 cents, whatever the tier. When `generateInvoices` finds a member paused at the request time of that request, whatever month is invoiced, the member's membership line has `kind` `membership`, `ref` the member's current tier, `quantity` 1, `unitPriceCents` 500, `discountPercent` as household discount states and `at` the request time. A suspended member keeps paying the tier's full fee.

Example: a paused member on `pro` pays 500 cents, and 450 cents (`(500 × 90 + 50) div 100`) in a household of 3 or more.

There is no new operation and no new refusal code.
