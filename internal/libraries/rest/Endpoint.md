---
kind: Kind
---
# Endpoint

An Endpoint Definition describes one HTTP operation of the project's REST interface: how a caller reaches a Feature over HTTP.

- The title is the method and the path, such as `POST /reservations`.
- Link the [[std/Feature]] it exposes.
- Name the request: path and query parameters, headers the caller must send, and the body's fields with their wire names in backticks.
- Name every response a caller can observe: the status code, the body's fields, and the error codes of each refusal.
- State what the caller may rely on beyond that, such as idempotency or ordering, and nothing more.
