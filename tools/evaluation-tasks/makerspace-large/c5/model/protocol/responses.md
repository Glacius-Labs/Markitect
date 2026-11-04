---
kind: std/Rule
internal: true
---
# Responses

Every response is one JSON object on one line, with exactly the members shown for its status.

Success:

```json
{"id":"q1","status":"ok","result":{"member":{"memberId":"ada","name":"Ada Byron","tier":"plus","standing":"good","registeredAt":"2026-03-02T09:00Z"}}}
```

Refusal of a well-formed request:

```json
{"id":"q2","status":"refused","code":"already_exists","reason":"A user with identifier ada already exists."}
```

`code` is exactly one of the refusal codes of this model. `reason` is a nonempty, understandable explanation; its wording is free.

Invalid request:

```json
{"id":null,"status":"invalidRequest","reason":"The line is not valid JSON."}
```
