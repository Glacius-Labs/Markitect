# Preparation validation, 2026-10-09

Outcome: one additive preparation pass, with no real study, implementing actor, product execution, provider test, installation or account diagnosis. Source began clean at `dc34459dfef9d31f3832fc0dafc14e6851b2f7ab` on private `codex/government-scientist`. The current direct preparation-only mandate and callback authority were checked in Root's `docs/design/playground-refinement-20261009.md` and targeted coordination-state fields. This is source/mechanics validation, not stable product readiness or human acceptance.

## Focused checks

| Check | Result | Evidence and limit |
|---|---|---|
| Final offline fixture suite | 18/18 PASS, exit 0, 51.609 seconds | [Exact command/source hashes/interval](offline-checks.json), [technical log](offline-checks.log); temporary Git repositories and mocked process outputs only |
| Lifecycle fixtures | 13 PASS in final suite | Flat public layout, no overwrite, pinned STATIONS waves, arbitrary branch/main/detached state, raw worktree/index/diffs/refs, LF-preserving main clone under global autocrlf=true, retained partial race failures, immutable snapshot content verification, clean-index advance without committing actor edits, early final freeze and closed transitions, bound repo/audit identity, external metadata passthrough |
| Checker reporting fixtures | 5 PASS in final suite | Missing app is one entrypoint failure with functional checks NOT RUN; return code/stderr precede JSON parsing; invalid JSON is an output failure; error mutation fails; unavailable test process is EVALUATION_ERROR. Every subprocess response is mocked |
| Public input provenance | 32 bindings verified | 30 files copied unchanged from committed source; only shared AGENTS.md and acceptance.py derived in the new copy. [Provenance](provenance.json) records source/destination hashes |
| Source-only sanity | PASS | New JSON parses, Python syntax compiles without bytecode, local Markdown links resolve. Initial model proposals were not validated against a product/schema |
| Historic boundary | PASS | `git diff dc34459dfef9d31f3832fc0dafc14e6851b2f7ab -- experiments/work-item-comparison/luna-nest-20261009` empty; old grants/candidates/outcomes/raw evidence unchanged |

Two bounded developer subagents contributed the mechanical source and independent contract/source review. They were development helpers, not measured study actors. The independent reviewer found a misleading broad `unfinished` flag: a clean selected main could still leave work on another feature ref. The final source uses `selectedHeadUnfinished`, retains every ref and leaves `overallUnfinished` unknown; Root reviewed the correction, and the final suite covers that exact clean-main/unmerged-ref case. Source hashes in the final receipt bind the tested version. There is no claimed provider usage or actual-study timing from these development reviews.

Root source review also corrected initial flat-layout/branch assumptions, early closure, actor-index preservation, pre-checkout LF configuration, mid-copy stability and failed-capture retention. A green initial fixture subset did not establish these missing behaviors. The final suite covers the corrected risks; no historical assessment was rescored.

## Remaining boundaries

The lifecycle contains no model dispatcher or product adapter. It records externally supplied authorization, setup and execution claims without validating or granting them. Actual order, stable integrated new main, executable/config/module/Skills/generated-view/business-model pins and observed serving/effort/tool-rights remain open. Claude Markitect inner-provider support remains unbound; a Codex-only product makes the Claude pair `blocked_by_provider_support` rather than a mixed-provider comparison. Missing usage remains unknown.

Capture verifies observed stable bytes and Git state; the controller must still join/stop its own actors before capture. Hash envelopes detect changes but are not tamper-proof OS isolation. Symlinks/submodules are unsupported by this small fixture protocol. All refs are retained; overall integration is assessed independently. A station transition leaves a declared uncommitted metadata change for the actor's normal integration and does not stage other work.

Markitect `go run`, builds, Go/product tests, installed-product checks, native Codex/Claude/App Server calls and real case acceptance were NOT RUN under the direct preparation-only scope. Repository instruction commands cannot expand that scope. No published release/Main change or human acceptance is claimed. After the clean private checkpoint and one material callback, hold for the next scoped instruction; no automatic trial or refill follows.
