# Documentation placement exercise

This neutral fixture exercises authoring before router validation. Its Project declares two Areas and opts into a `docs` router root. Run `markitect inventory --repo examples/documentation-placement` to find existing pages and `markitect check --repo examples/documentation-placement` to check the navigation structure.

Authoring task: **Document that database schema migrations must remain backward compatible during rolling deployments.** Start at `docs/README.md`, follow the local routers, and inspect the existing owner before editing. The expected smallest change is to the existing Engineering persistence document. A second migration document or a Product copy would duplicate its ownership. The router check can confirm that navigation remains intact; it cannot prove the semantic choice.
