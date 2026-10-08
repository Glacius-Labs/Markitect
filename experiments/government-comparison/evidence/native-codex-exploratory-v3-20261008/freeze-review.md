# Independent freeze binding review

Reviewer: prep-003  
Protocol: `native-codex-exploratory-v3-20261008`  
Scope: frozen source/support bindings, current grant and configuration, six cell starts, current Actor inputs, pinned product artifacts, authority and resource envelope. Static/read-only review; no tests or study Actors started. Private oracle content was not opened or parsed.

## Result

**PASS for common-preparation closure and immediate first-cell start. No freeze blocker found.** This is a preparation/freeze decision only; no native product invocation, cell outcome, or human acceptance is established here.

## Verified bindings

- `freeze.json` SHA-256 is `18fd3a2af9356fd83c9b73c9fdfd1b67976146565659366959ca3b415627a484`; `protocol.md` SHA-256 is `78cd11a0578f8e708258d794de8235ca31dc1138b6ce0f79eb8446d7f11098ce`. All 149 entries in the freeze file map and all 115 per-cell support pins exist and match their recorded SHA-256 values (264 checked entries total; no missing files or mismatches).
- All six cell repositories are distinct, clean Git repositories on their assigned `codex/native-v3-*` feature branches. The three Greenfield cells are at base `0416e366626d31933443ce2901d6a3696261db7b`; the three Brownfield cells are at base `5d21ef02cd9d5485c26bc4053a35c61ad33024c1`.
- Each Actor `inputs` tree contains only shared public setup/method files and stage 1: the current `01-create-order.md`, architecture, brief, checks, and release manifest (plus the Brownfield provenance where applicable). No later task card, private oracle, or other-cell output appears in these copied Actor inputs. Future cards in the Scientist source remain outside those trees. Private oracle digests are represented only in the freeze metadata; their contents were not read.
- Both real product artifacts are pinned and byte-verified in their cell support: Classic v0.14.1 at held source `c91363b7ac4decbe87212ff0f588b5451581a152`, runtime source `7dbd599c81540c8203a1b7f83afbc335174f4f1f`, binary SHA-256 `2cad55efad64f15d7f57638bea78312918fbf7d181c730f188b9921504da71c4`; Government G5 source `04e225d5caee78c2a198607143863fca1e829750`, binary SHA-256 `12241f325e4af59451e4021b31d9e6f5b829b5de06c94d35eaabfb9d663aa51f`. Frozen product docs/modules and source identities are included in the manifest pins. The main source `a97cbd5ef3e0b22b9e6397501047a4e01dc90204` is recorded separately.
- The prior run-ID concern is resolved from both held source commits: `randomID` encodes 16 random bytes with `hex.EncodeToString`, yielding a 32-character hexadecimal ID compatible with the mailbox's alphanumeric check.
- The canonical authorization record names the direct human request to run the case studies, authorizes six finite trials, and explicitly permits first-cell execution after prospective freeze verification without another Root go. Current configuration consistently records inherited `gpt-6.1-sol/high`, no overrides, serving identity and usage unknown, and cooperative shared-host access. The harmless dummy cross-directory probe succeeded, so this is not OS isolation; the protocol states that limit accurately.
- Resource values agree across the authorization, protocol, freeze, cell identities, and ledger: 1,200 seconds / 12 activations per task; 7,200 seconds / 72 per trial; 432 across six trials; four concurrent study actors; two semantic repair rounds; 600 active human seconds per trial. The preparation ledger reserves exactly the three authorized common activations. Its `correctionBatches: 1` and `affectedTestInvocations: 2` match the authorization envelope; both recorded invocations passed all three focused checks. No more test invocation is needed or authorized for this review.
- Both Government cells specify the same two final Ressort votes: correctness and maintainability. The protocol requires distinct role contexts, prior mandates, and explicit final assent for the exact candidate/evidence/round, including unaffected assent.
- The task-6 mutation target is described as precommitted Scientist-side protocol state and does not appear in any copied Actor input. The freeze uses the 149-file manifest; it does not introduce an old app-server S1 requirement or the previously discussed 217 ancestry-pin inventory.

## Boundary for execution

Proceed with the first actual cell promptly under the existing grant and dispatch ledger. Treat the native capability result as evidence of local read/write/build/run only; it does not establish that either pinned product can complete a real role invocation. Preserve the first invocation/response as product-method evidence. Maintain the recorded cooperative isolation and null usage/serving-identity limitations.
