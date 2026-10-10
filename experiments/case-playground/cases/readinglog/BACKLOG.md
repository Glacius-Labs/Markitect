# Backlog: Readinglog
Only .study/station.json releases work. All requirements are public. Preserve supplied working add/list baseline and stored records. Each item includes tests/docs and isolated main integration; choose internal organization freely.

## S1: one item
- B01: README finish and list --status; preserve no-filter order, idempotence, add/list/persistence/failure semantics.

## S2: three items
- B02 (B01): README atomic all-or-nothing import --csv, exact columns/normalization/duplicates and input order.
- B03 (B01): README lexical-author summary, all books/pages and finished count.
- B04 (B02,B03): integrate baseline/regression tests and ordinary usage/failure/state docs.

## S3: seven items, actual team station
At least two executing agents must overlap on independent groups and integrate contributions. Native roles/prompts/planning/branches are yours. Retain TEAMWORK.md with identities/intervals/SHAs/merges/conflicts.
- B05 (B04; query group): list --author NAME, normalized/case-sensitive and composable with status, preserving insertion order.
- B06 (B04; export group): export --csv PATH writes UTF-8 header id,title,author,pages,status in insertion order, quotes comma/newline text, returns {"exported":COUNT}, never changes DB; destination errors fail clearly.
- B07 (B04; reporting group): summary --author NAME selects the normalized case-sensitive author, unknown gives authors=[]; unfiltered behavior unchanged.
- B08 (B05): test author/status combinations, unknown authors and process restart.
- B09 (B06): test quoting, both statuses/empty export, DB nonmutation and output failures.
- B10 (B07): test counts/pages after finish/import and exact author filtering.
- B11 (B08,B09,B10): integrate all groups, regression/docs and actual team/merge/conflict evidence; no architecture prescription.

## S4: one final domain rename
- B12 (B11): Change active domain term book to entry. Default list becomes {"entries":[...]}; summary author objects use "entries" instead of "books". Update active behavior/interfaces/code/tests/docs/config and any project model/views. Existing command names and raw stored records remain readable/compatible, without rewriting on read. For one compatibility period list --legacy returns {"books":[...]} and summary --legacy retains author "books" counts, both composable with filters. Explain compatibility before implementation, then test new/legacy modes against pre-change data and all commands. Historic logs and deliberate aliases are allowed; grep alone is not acceptance.

12 items in four waves (1/3/7/1). Do not implement unreleased waves.
