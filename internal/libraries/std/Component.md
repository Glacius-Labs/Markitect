---
kind: Kind
fields:
  Instances: {required: true}
  Implemented by: {required: true, links: Package}
parts:
  offers: std/Offer
  requirements: std/Requirement
  needs: std/Need
  rules: std/Rule
---
# Component

A Component Definition says in one sentence what exists when the system runs: a process, a client in a browser, a store. It lives under `architecture/` and is not a Package. **Instances:** how many, and when that is fixed. **Implemented by:** the Packages whose code runs in it, or in words that all of the project's does.

Its [[std/Offer|Offers]], [[std/Requirement|Requirements]] and [[std/Need|Needs]] live in the folder of its name, under `offers/`, `requirements/` and `needs/`; without Offers it offers nothing, without Requirements it requires nothing. A [[std/Rule]] of its behavior lives under `rules/`.

Verified when it exists and its parts hold.
