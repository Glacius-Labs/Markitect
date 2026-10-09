# Local KG handoff

Base: `1495e1be7b0046711531fa42c8407fba67b8e814`; branch `codex/knowledge-graph`.

| Item | State | Ownership |
|---|---|---|
| KG01 reference / exactly twelve questions | Complete: independent Luna High Reader PASS, no material corrections | This workstream, new documentation only |
| KG02 pure projection/index | In progress after KG01 Reader PASS | Reserved new `internal/modules/projectknowledge` files |
| KG03 bounded read-only navigation | In progress with pure KG02 contract | Same pure module, no CLI/Host wiring |
| KG04 vocabulary / continuity | Unallocated | Integration coordination |
| KG05 operational adapters | Unallocated | Integration exclusive |
| KG06/07 CLI/MCP/skills / integration | Unallocated | Integration exclusive after shared product work |
| RF01 mapping | Possible later scope | Reuse KG01; no engine authorization |
| RF02â€“RF06 | Unallocated / conditional | Separate decision; no studies or migration |

Proposed pure seam: `core.Model` plus module-owned `ProjectFacts` and explicit `Scope` values. External adapters own file ownership/membership and scope selection; the module must not import `projectmodel.Report` or Host. Nodes/edges preserve Core identity and provenance. Read-only traversal carries bounded witness paths and never narrows or replaces existing Impact. A single selected graph is not evidence of removed edges across two revisions. Operational DTOs and freshness judgments are deferred to KG05.

Integration confirmed directory/import ownership in chat `01a121a0-0417-71c1-9e4b-be742f8c1146`. P08 moves are handled later by rebasing actual source paths; this work is not part of current Main-readiness integration. Main merge/release, canonical RDF, engine installation and product-role trials are outside this assignment.

KG01 Reader reviewed the four documents against pinned source, confirmed exactly twelve questions, and ran no tests or mutations. No material findings; review was finite.
