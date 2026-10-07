# Independent review: Q1 setup and policy compatibility

Review scope: the public Q1 setup procedure, its exact canonical study-contract snapshot and provenance, the updated user-goal coverage matrix, and the pinned CLI/schema compatibility packet. This is a documentation and evidence review, not a study-cell result or a readiness approval.

## Q1 procedure

The provenance record binds `docs/design/government/evaluation.md` at commit `5162a678cf151a6c42fb723567472c7e43c226b1`, Git blob `e51cb07255e6a56e0cbd53e3137c17d053098721`, 29,625 bytes, SHA-256 `ed8e50429ba2de41f7074acec10be07f7fe09d1ee34909c87ef788202fb7cf17`. The exported snapshot at [study-contract-2026-10-07-q1-v2.md](study-contract-2026-10-07-q1-v2.md) matches those committed source bytes exactly. The earlier snapshot is retained separately.

[Q1 setup v1](q1-intent-setup-v1.md) implements the added canonical section consistently: before implementation each arm briefly restates only currently released intent, names its sources/cutoff, distinguishes assumptions and open questions, and uses any ordinary setup/planning format. It applies the same release cutoff, clarification rules, recordkeeping and independent comparison across arms. It forbids leaking future cards, adding hidden preferences, treating an assessment as user acceptance, or creating a free session/review. It leaves unanswered decisions open under the existing intervention rules. These are shared setup/observation rules, not new functional task requirements or scoring weights; Conventional may use ordinary Markdown.

The updated [user-goal coverage](user-goal-coverage.md) now describes Q1 as a fixed observation path that remains untested. It correctly limits the claim to replaying already-normalized synthetic intent: genuine elicitation, human model acceptance, and real-user usability remain untested. It identifies the adopted setup procedure without claiming any study cell has run or any human acceptance has occurred. No material contradiction with the bound snapshot or Q1 procedure was found.

## Policy compatibility evidence

The packet records six bounded calls to the pinned local CLI `codex-cli 0.160.1` at `C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/5ea220ae823df3d7/codex.exe`, SHA-256 `3b8f6e33caa75f232558a3cf76ff9b87bb5ef6dbcf4996372f24e55c78b1b916`: five help calls and one static schema export. The raw receipts show exit code 0 for all six. Global help does not list `--permission-profile`; its `--profile` option is described only as a named configuration-file layer. The report records the first receipt’s terminal-encoding display failure and a later local lookup of an absent optional schema filename; neither triggered another CLI call.

The schema manifest records 440 members and archive SHA-256 `cd24042eec4696f6b73368cfc6ce930726f3e10c28bae63cdbd1b2f502ff1d97`. I checked the ZIP directly: it contains 440 files and every member matches its manifest hash. The schemas expose named-profile interfaces: thread start `permissions`, command `permissionProfile`, and managed `allowedPermissionProfiles`/`defaultPermissions` in `ConfigRequirementsReadResponse`. `ConfigReadResponse.Config` allows additional properties, so an undeclared configuration field is not evidence that support is absent.

These schemas establish interface presence only. The evidence does not show that the CLI parser accepts a corresponding config key, that app-server profile fields work on the frozen `exec` route, or what effective/inherited Windows or managed permissions applied. It does not prove read-only execution, preservation of `approval_policy="never"`, successful access to either sentinel file, or a supported configuration correction. The report states these limits and leaves S1 open. No Actor, inference, app-server RPC, permission change, or further discovery call is evidenced here.

## Disposition

The Q1 documentation and provenance are consistent. The policy report is appropriately bounded: it recognizes the schema fields while leaving parser, effective runtime policy, inherited rights and successful read-only behavior unresolved. The six-call metadata allowance is exhausted; this review authorizes no further probe. No study cell has run, and S1 remains open.