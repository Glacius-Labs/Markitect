# Public trajectory: Roombook
Only .study/station.json releases work. All requirements are public. Each item includes tests/docs, freely chosen implementation and integration into isolated main.

## S1: one item
- R01: Implement README book/list: validation, half-open overlap, sorting, IDs, durable state and errors without mutation. No cancel/summary yet.

## S2: three items
- R02 (R01): README cancel: idempotence, missing ID, history and freed interval.
- R03 (R01): README summary: active-only counts/exact minutes, lexical rooms.
- R04 (R02,R03): integrate regression tests and ordinary run/failure/state/limitation docs.

## S3: seven items, actual team station
At least two executing agents must overlap on independent groups and integrate contributions. Choose native roles/prompts/planning/branches yourself; retain TEAMWORK.md with identities/intervals/SHAs/merges/conflicts.
- R05 (R04; query group): list --status active|canceled, preserving unfiltered ordering; compose with --room.
- R06 (R04; export group): export --csv PATH writes UTF-8 CSV header id,room,start,end,title,status, in list order, with comma/newline quoting, returns {"exported":COUNT}, never changes DB; destination errors fail clearly.
- R07 (R04; reporting group): summary --room NAME selects normalized case-sensitive room; unknown gives rooms=[]; unfiltered behavior unchanged.
- R08 (R05): test status/room combinations, unknown rooms and stable order across processes.
- R09 (R06): test CSV escaping, empty export/canceled history, DB nonmutation and output failures.
- R10 (R07): test filtered counts/minutes after cancel/adjacent bookings.
- R11 (R08,R09,R10): integrate all groups, regression/docs and actual team/merge/conflict evidence. No prescribed architecture or allocation recipe.

## S4: one final domain rename
- R12 (R11): Change active domain term reservation to booking. Default list becomes {"bookings":[...]}; active code/tests/docs/config and Markitect model/views consistently use booking. Existing commands book/list/cancel/summary/export, returned record fields and raw stored JSON remain readable/compatible, without rewriting on read. For one compatibility period list --legacy returns {"reservations":[...]} with identical records, composable with room/status. Explain compatibility before implementation; test both interfaces on pre-change data and every behavior. Historical logs and deliberate legacy aliases are permitted; grep alone is not acceptance.

12 items, 1/3/7/1 waves, one longitudinal Greenfield case. Do not implement unreleased waves.
