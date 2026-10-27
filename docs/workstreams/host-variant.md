# Host variant (lane 1)

The host variant is Markitect as developed so far. The host coordinates the Manager hierarchy derived from the canonical model and keeps the guarantees in host state. Its target is [V1 concept 1.0](../concepts/v1-concept.md), and its code is on `main`. The [delivery plan](../implementation-plan.md) owns its steps, its known deviations and the integration rules. This page holds only the lane's scope, a short state and its open owner questions, and links to the pages that own the details.

The lane exists since the owner split Markitect's work into lanes on 27 October 2026. Its lead is the Coordinator session, which may start sub-sessions for the lane's work.

## Scope

- **Owns:**
  - On `main`: the Run and Apply host, the harness and the provider adapters: Codex, and Claude as source only since K1 increment 1.
  - The Linux runner's code.
  - The V1 qualification and acceptance records under `docs/validation/`.
  - The delivery plan.
- **Uses, does not own:**
  - The compiler, the model, the Libraries and the evaluation suite are shared. Their changes land on `main` through normal PRs from their own lanes.
  - In the comparison suites the host variant is one arm.
- **Elsewhere:**
  - The Git variant owns [DEC-078](../concepts/register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review) and its slices G0 to G4. On `main`, today's handoff and publication stay until the Git variant's work lands there, which the owner decides.
  - Questions about Windows and Linux execution are discussed in the platform lane.

## State

- **Acceptance:** the [technical acceptance record](../validation/v1-technical-acceptance-20261026.md) of 26 October finds V1 correct and safe on F_final, but not yet reliable at Makerspace size. The owner approved the fix plan for those limits. Its native controls then ran, and the fixes they led to were merged. The record's addendum of 27 October gives no new verdict.
- **Frozen source:** F2 is `7371cfd9`, with its CI run, binary and images pinned ([F2](../validation/v1-technical-acceptance-20261026.md#new-frozen-source-f2)). No native Apply ran on F2 itself.
- **Steps:** steps 1 to 3 are done. Step 4 has native evidence on F, F′ and the fix-plan controls. Step 5 is open ([steps](../implementation-plan.md#steps)).
- **Linux runner:** L1 increment 1, L2 and L3 are done. L1's WSL form and a native Linux engine are open ([Linux Docker way](../implementation-plan.md#linux-docker-way-owner-direction-26-october)).
- **Claude adapter:** K1 increment 1 is on `main` as source only ([PR #788](https://github.com/Glacius-Labs/Markitect/pull/788)). It is not in F2, and V1's provider scope stays Codex (D18).
- **Sub-sessions:** none running.

## Next

1. **Step 5, the acceptance package:**
   - Guides and code map.
   - The provider matrix, with Claude as source only.
   - The known limits.
   - Linux CI on the frozen head and the manual Windows smoke.
   - An independent architecture review.
   - Owner acceptance follows as a separate decision.
2. **The next comparison study:** the host variant is one arm. The evaluation lane runs the study on F2 or a later freeze ([plan](../implementation-plan.md#product-fixes-before-the-next-study-owner-approval-26-october)).
3. **K1b:** the native Claude qualification inside the runner, after the owner's yes.
4. **Small follow-ups:**
   - Capture refuses a working state whose Package directory is empty, because Git does not keep empty directories.
   - WP3b's ordering by dependency is deferred.
   - Whether WP1 may continue after served actions is open.

## Open owner questions

1. **V1 acceptance and the Git variant.** Does V1's acceptance on `main` cover today's publication, with the Git variant judged after its prototype? The recommendation is yes. The delivery plan's [question 5](../implementation-plan.md#git-remote-collaboration-owner-direction-27-october) still recommends that acceptance cover the new publication. It was written before the owner answered the G0 questions on 27 October with "prototype first", which the Git variant's G0 design records. Lane 1 updates question 5 once that design is on `main`.
2. **Long-lived Claude token.** Should a long-lived Claude token mode return? The recommendation is no. Keep the access-only copy of the login, because a Manager's shell could read a token that is valid for a year.
3. **Starting K1b.** Should K1b start now? It uses the owner's Claude subscription. The recommendation is one bounded Library smoke in the runner first.
4. **Docker Desktop's WSL integration.** The runner's WSL form needs it. This is a platform question for the platform lane.
