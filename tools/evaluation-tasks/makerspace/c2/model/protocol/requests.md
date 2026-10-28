---
kind: std/Rule
internal: true
---
# Requests

A request is a UTF-8 JSON object with exactly three members:

```json
{"id":"q1","op":"registerMember","args":{"actor":"admin","at":"2026-03-02T09:00Z","memberId":"ada","name":"Ada Byron","tier":"plus"}}
```

- `id`: a nonempty string chosen by the caller. It is echoed in the response. It is only a correlation value, not a domain identifier and not an idempotency key; two requests with the same `id` are two separate requests.
- `op`: the wire name of one operation of this model. Names are case-sensitive.
- `args`: a JSON object with the operation's arguments. Every operation takes `actor` ([[values|identifier]], the acting user) and `at` ([[values|timestamp]], the request time) in addition to the arguments it lists. All arguments are required; no operation has optional arguments.

The operations are those of the module interfaces: [[members/members-interface]], [[equipment/equipment-interface]], [[billing/billing-interface]], [[training/training-interface]], [[bookings/bookings-interface]], [[materials/materials-interface]], [[safety/safety-interface]], [[reporting/reporting-interface]].
