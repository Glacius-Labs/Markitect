---
kind: std/Rule
---
# Hazard gate

A purchase of a material that has a [[hazmat/classification]] passes the hazard gate before the purchase is made. A material without a classification is not gated. The gate has two refusals, judged in this order for the member who buys:

1. `hazard_not_certified` when the member has no [[training/certification]] for the classification's family that is valid on the date of the request time ([[training/certifications]]).
2. `hazard_limit_exceeded` when the units of the material the member bought in the month of the request time, plus the quantity of this purchase, are more than the classification's `monthlyLimit`.

The units bought in the month are the sum of `quantity` over the member's [[materials/purchase|purchases]] of that material whose `at` lies in the month of the request time. Purchases of other members or of other materials are not counted. The stock that consumables draw is not a purchase and is not counted. Buying exactly `monthlyLimit` units in all is allowed.

The classification in force at the request time is the one judged, whatever it was when earlier purchases were made.

Example: `monthlyLimit` is 10 and a certified member bought 6 units on `2026-03-05`. On `2026-03-20` a purchase of 4 units passes the gate, because 6 + 4 is 10, and a purchase of 5 units is refused with `hazard_limit_exceeded`. On `2026-04-01` the count of the month starts again at 0.

[[materials/purchase-material]] applies this gate; the answer is supplied to materials as [[hazmat/hazmat-interface]] states.
