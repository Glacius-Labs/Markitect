---
kind: std/Rule
---
# Household discount

The membership line of an invoice that `generateInvoices` creates has `discountPercent` 10 when the member belongs, at the request time of that request, to a household with 3 or more members. Otherwise it is 0. The payer counts as a member. Every member of such a household gets the discount on their own membership line, not only the payer. Households are judged as they are when the request is made, whatever month is invoiced. Only the membership line is affected; the line amount follows [[money]].

Example: a household of 3 whose members are on `basic`, `plus` and `pro` gets membership lines of 2250, 4410 and 8910 cents (`(2500 × 90 + 50) div 100`, `(4900 × 90 + 50) div 100` and `(9900 × 90 + 50) div 100`). With 2 members the same lines are 2500, 4900 and 9900. A member who belongs to no household gets no discount.
