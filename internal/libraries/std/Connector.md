---
kind: Kind
fields:
  Between: {required: true, links: [std/Component, std/ExternalSystem]}
  Protocol: {required: true}
  Limits: {required: true}
parts:
  errors: std/Error
---
# Connector

A Connector Definition states how [[std/Component|Components]] interact, under `architecture/`. **Between:** the Components, or a Component and an External system. **Protocol:** who starts, what goes back and forth, and whether the caller waits. **Limits:** time limits and sizes, with values, or "none".

What each side sees when the other fails are its [[std/Error|Errors]], under `errors/`.

Verified when the Components interact as stated.
