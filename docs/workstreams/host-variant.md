# Host variant (lane 1)

The host variant is Markitect as developed so far. The host coordinates the Manager hierarchy derived from the canonical model and keeps the guarantees in host state. Its target is [V1 concept 1.0](../concepts/v1-concept.md), and its code is on `main`. The [delivery plan](../implementation-plan.md) owns its steps, its known deviations and the integration rules. This page holds only the lane's scope, what it does now and next, and its questions for the owner. It links to the pages that own the details.

The lane exists since the owner split Markitect's work into lanes on 27 October 2026 ([owner source](../concepts/sources/owner-evening-decisions-20261027.md#3-lanes-1611-and-1614-utc)). Its lead is the Coordinator session, which may start sub-sessions for the lane's work.

## Scope

- **Owns:**
  - On `main`: the Run and Apply host, the harness and the provider adapters: Codex, and Claude as source only since K1 increment 1.
  - The Linux runner's code.
  - Arm M's adapter for the evaluation suite, in `tools/arm-m/`.
  - The V1 qualification and acceptance records under `docs/validation/`.
  - The delivery plan.
- **Uses, does not own:**
  - The compiler, the model, the Libraries and the evaluation suite are shared. Their changes land on `main` through normal PRs from their own lanes.
  - In the comparison suites, the host variant is arm M.
- **Elsewhere:**
  - The Git variant owns [DEC-078](../concepts/register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review) and its slices. On `main`, today's handoff and publication stay until the owner adopts that variant.
  - Questions about Windows and Linux execution are discussed in the platform lane.

## Now

- **Step 1, the shared yardstick:**
  - Lane 1 agreed to the evaluation lane's arm interface.
  - The evaluation lane now lands the task and run schemas and the run checker on `main`.
  - Arm M's adapter waits for them.
- **Where V1 stands:**
  - **Acceptance:** the [technical acceptance record](../validation/v1-technical-acceptance-20261026.md) of 26 October finds V1 correct and safe on F_final, but not yet reliable at Makerspace size. The owner approved the fix plan for those limits. Its native controls then ran, and the fixes they led to were merged. The record's addendum of 27 October gives no new verdict.
  - **Acceptance scope:** V1's acceptance on `main` covers today's publication, and the Git variant is judged after its prototype ([owner, 27 October](../concepts/sources/owner-evening-decisions-20261027.md#5-v1s-acceptance-scope-late-evening)).
  - **Frozen source:** F2 is `7371cfd9`, with its CI run, binary and images pinned ([F2](../validation/v1-technical-acceptance-20261026.md#new-frozen-source-f2)). No native Apply ran on F2 itself.
  - **Steps:** steps 1 to 3 are done. Step 4 has native evidence on F, F′ and the fix-plan controls. Step 5 is open ([steps](../implementation-plan.md#steps)).
  - **Linux runner:** L1 increment 1, L2 and L3 are done. L1's WSL form and a native Linux engine are open ([Linux Docker way](../implementation-plan.md#linux-docker-way-owner-direction-26-october)).
  - **Claude adapter:** K1 increment 1 is on `main` as source only ([PR #788](https://github.com/Glacius-Labs/Markitect/pull/788)). It is not in F2, and V1's provider scope stays Codex (D18).
- **Sub-sessions:** none running.

## Next

1. **Arm M's adapter.** Step 4's controller moves into `tools/arm-m/` as arm M's adapter for the shared interface.
   - It runs native Apply on Linux.
   - It derives one repository identity per chain and arm.
   - It carries the state directory over as its continuation.
   - It commits the candidate with a fixed identity.
   - It reports the Apply status and cause unchanged.
   - It has offline tests, and runs no engine.
2. **A stable arm M.** One Library task runs end to end through the interface three times in a row without manual help. This needs a bounded dispatch that the owner approves.
3. **K1b.** The native Claude qualification inside the runner, after the owner's yes. Arm P and a Claude suite need it first.
4. **Step 5, the acceptance package:**
   - Guides and code map.
   - The provider matrix, with Claude as source only.
   - The known limits.
   - Linux CI on the frozen head and the manual Windows smoke.
   - An independent architecture review.
   - Owner acceptance follows as a separate decision.
5. **Small follow-ups:**
   - Capture refuses a working state whose Package directory is empty, because Git does not keep empty directories.
   - WP3b's ordering by dependency is deferred.
   - Whether WP1 may continue after served actions is open.

## Questions for the owner

1. **Starting K1b.** Should K1b start? It uses the owner's Claude subscription. The recommendation is one bounded Library smoke in the runner first.
2. **Long-lived Claude token.** Should a long-lived Claude token mode return? The recommendation is no. Keep the access-only copy of the login, because a Manager's shell could read a token that is valid for a year.
3. **Arm M's first dispatch.** Once the adapter is merged, may three Library runs of arm M run on Codex Luna under one bounded dispatch? The recommendation is yes, as step 2's stability check.

The question about Docker Desktop's WSL integration moved to the platform lane.
