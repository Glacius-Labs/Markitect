---
kind: Kind
fields:
  Environment: {required: true}
  Runs: {required: true, links: std/Component}
parts:
  provisions: std/Provision
---
# Deployment

A Deployment Definition says where the system runs, under `architecture/`. **Environment:** the place, in one sentence. **Runs:** the [[std/Component|Components]] that run there.

What the place provides are its [[std/Provision|Provisions]], under `provisions/`, matched against each Component's Needs.

Verified when the project can be brought to run there as stated.
