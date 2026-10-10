# Current product purpose over inherited compatibility

Direct user decision on 9 October 2026: there are no existing users requiring compatibility. Delete functions retained solely for compatibility instead of preserving, hiding or merely deprecating them. Designer owns implementation in the active native-working package. Root edits coordination only. Existing proof limits remain unchanged; studies, other owners, automatic observation, Main merge and releases remain stopped.

## Source-backed retention findings

Read-only audit of Designer source through local 6121a8ad, with public PR head b61eb445:

- The canonical `.markitect/areas/development/repository-boundaries.rule.yaml`, introduced in this form by eec465ed, explicitly requires preserving historical commands, installable manifest types and records. Its generated development rule repeats the requirement. This inherited instruction can drive retention without a fresh user requirement; update the canonical rule and regenerate its views. Latest human instructions override the older rule.
- `docs/project-workflow.md` distinguishes historical top-level initialization from `markitect project init`. Parallel formats and commands introduce onboarding ambiguity. Retain only currently needed functions; age alone is not the criterion. A previous public product proof selected the historical initialization path and stopped on incompatible coverage.
- `internal/host/projectadoption/reverse.go` grants repository-wide evidence routing to root iterations with omitted delegation pools, expressly to preserve older iterations. Remove compatibility-only omission behavior and associated markers where unnecessary. Define a clear safe new authority contract; reject ambiguous old state rather than silently expanding its authority. Historical records remain evidence, not a promise of executable compatibility.
- Root also explicitly required proposal-only compatibility and a model-first router in earlier briefs. Those requirements are withdrawn. Designer should remove them if they have no independent current use, rather than infer that Root still requires them.

For each affected surface, report current purpose, keep/delete decision and resulting ordinary user workflow. Remove corresponding dead code, fixtures, tests and instructions; replace old preservation tests with focused current-contract checks where needed. Preserve useful compiler/Host reuse, independent read-only reviews, ownership/freshness checks, guarded Apply and protection of adopting-project content. Do not rewrite functioning internals merely to rename them. Preserve immutable historical releases, source pins and original evidence. No quantitative development-cost attribution follows from this audit.

## Readiness snapshot and Main gate

Public b61eb445 has successful Linux/Windows CI 37925341863. Local native-working commits 91bd9cf6 and 6121a8ad are later candidates, not covered by that CI. Native work is still unproven at this audit: scoped candidate trees lack Git history, deletion/binary output support and native helper lifecycle support. Model-declared Managers are still distinct Host-scheduled actors. Failed Manager receipts were corrected locally after independent review; targeted checks do not prove normal autonomous delivery.

Before recommending Main, require a coherent pushed final source, required CI on that exact head, closed independent review, and the authorized small real native proof with honest remaining limits. The normal user workflow must support actionable compiler/model repair and actual candidate/Review/Verify/Apply rather than claim it from structural checks alone. Case Studies and every future interface are not prerequisite gates. This decision does not authorize automatic merge or release.

## Two separate interface directions

For agents and humans invoking Markitect, retain CLI for normal local work, CI and diagnostics. A thin local MCP tool adapter over the same application services is a promising later interface: typed arguments/results and actionable diagnostics, without duplicating business logic or introducing a remote server solely for its own sake. MCP supports structured tool results and error reporting: [official tools specification](https://modelcontextprotocol.io/specification/2025-11-25/server/tools).

For Markitect controlling Codex Managers, consider App Server for fresh threads, tool/lifecycle events and usage notifications. These protocol capabilities suit managed work better than relying solely on incomplete one-shot JSON events. The installed version still requires validation of actual helper accounting and termination; documentation is not runtime proof. The Python SDK can provide a client, but is not necessary for the current package: [App Server](https://learn.chatgpt.com/docs/app-server), [Codex SDK](https://learn.chatgpt.com/docs/codex-sdk).

This is an interface recommendation, not an implementation grant for MCP, App Server migration, new provider integrations or installations. A working CLI-based first Main candidate can be useful without those additions.
