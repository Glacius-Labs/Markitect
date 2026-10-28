---
kind: std/Concept
---
# Refusal codes

The refusal codes every module shares. Each area module's interface lists the codes it adds.

| Code | Meaning |
| --- | --- |
| `already_exists` | A record with the supplied identifier already exists. |
| `forbidden` | The actor's role may not use the operation, or may use it only for the actor's own records and the record concerned belongs to another member. |
| `invalid_argument` | An argument is missing, `null`, of the wrong type, or outside its format or range. |
| `invalid_interval` | An interval's end is not after its start, or a report range is longer than allowed. |
| `invalid_state` | The record is not in a state that allows the operation. |
| `not_found` | A referenced record does not exist. |
| `starts_in_past` | An interval starts before the request time. |
| `time_regression` | `at` is earlier than the request time of the latest request answered `ok`. |
| `unknown_actor` | `actor` does not identify a user. |
