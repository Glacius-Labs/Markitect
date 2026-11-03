---
kind: std/Rule
---
# Quote validity

A [[quotes/quote]] that is sent has `sentAt` equal to the request time and `validUntil` equal to the date 30 days after the date of the request time ([[time]]).

A sent quote can be accepted on every request date up to and including `validUntil`. On the first date after `validUntil` an acceptance is refused with `quote_expired`. A rejection is never refused for age. A quote that has passed `validUntil` stays `sent`.

Example: a quote sent at `2026-03-02T15:00Z` is valid until `2026-04-01`. It can be accepted at `2026-04-01T23:59Z` and not at `2026-04-02T00:00Z`.
