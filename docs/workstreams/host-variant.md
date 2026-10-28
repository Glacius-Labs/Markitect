# Host variant (lane 1)

The host variant is Markitect as developed so far. The host coordinates the Manager hierarchy derived from the canonical model and keeps the guarantees in host state. Its target is [V1 concept 1.0](../concepts/v1-concept.md), and its code is on `main`. The [delivery plan](../implementation-plan.md) owns its steps, its known deviations and the integration rules. This page holds only the lane's scope, what it does now and next, and its questions for the owner. It links to the pages that own the details.

The lane exists since the owner split Markitect's work into lanes on 27 October 2026 ([owner source](../concepts/sources/owner-evening-decisions-20261027.md#3-lanes-1611-and-1614-utc)). Its lead is the Coordinator session, which may start sub-sessions for the lane's work. Since 28 October, two sessions replace the lanes: lane 1 finishes its current task, then hands arm M over to the Build and evaluation session and ends ([workstreams](../workstreams.md)).

## Scope

- **Owns:**
  - On `main`: the Run and Apply host, the harness and the provider adapters: Codex, and Claude Code, qualified natively once by K1b.
  - The Linux runner's code.
  - Arm M's adapter for the evaluation suite, in `tools/arm-m/`.
  - The V1 qualification and acceptance records under `docs/validation/`.
  - The delivery plan.
- **Uses, does not own:**
  - The compiler, the model, the Libraries and the evaluation suite are shared. Their changes land on `main` through normal PRs from their own lanes.
  - In the comparison suites, the host variant is arm M.
- **Elsewhere:**
  - The Git variant owns [DEC-078](../concepts/register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review) and its slices. On `main`, today's handoff and publication stay until the owner adopts that variant.
  - Questions about Windows and Linux execution are discussed in the Topics session.

## Now

- **K1b, done:** the first native Claude Code Apply, one Library Apply through the runner with the subscription login, passed on its second attempt, on the head of the launch fix in [PR #807](https://github.com/Glacius-Labs/Markitect/pull/807) ([record](../validation/v1-k1b-claude-qualification-20261028.md)). The dispatch allowed three attempts, and two were used. `c87e0660` brings that fix to `main`; no native Apply ran on `main` itself. Each Manager answered one message, so `--resume` across responses was not exercised.
- **Handover:** lane 1's current task is done. The handover of arm M waits for the owner's answer on the [build page](build.md#questions-for-the-owner).
- **Step 1, the shared yardstick:** lane 1 agreed to the evaluation suite's variant command. The suite, [PR #793](https://github.com/Glacius-Labs/Markitect/pull/793), is not on `main` yet, and arm M's adapter waits for it.
- **Where V1 stands:**
  - **Acceptance:** the [technical acceptance record](../validation/v1-technical-acceptance-20261026.md) of 26 October finds V1 correct and safe on F_final, but not yet reliable at Makerspace size. The owner approved the fix plan for those limits. Its native controls then ran, and the fixes they led to were merged. The record's addendum of 27 October gives no new verdict.
  - **Acceptance scope:** V1's acceptance on `main` covers the current publication (working-tree publication and the runner's branch hand-back), and the Git variant is judged after its prototype ([owner, 27 October](../concepts/sources/owner-evening-decisions-20261027.md#5-v1s-acceptance-scope-2015-utc)).
  - **Frozen source:** F2 is `7371cfd9`, with its CI run, binary and images pinned ([F2](../validation/v1-technical-acceptance-20261026.md#new-frozen-source-f2)). No native Apply ran on F2 itself.
  - **Steps:** steps 1 to 3 are done. Step 4 has native evidence on F, F′ and the fix-plan controls. Step 5 is open ([steps](../implementation-plan.md#steps)).
  - **Linux runner:** L1 increment 1, L2 and L3 are done. L1's WSL form and a native Linux engine are open ([Linux Docker way](../implementation-plan.md#linux-docker-way-owner-direction-26-october)).
  - **Claude adapter:** K1 increment 1 and K1b's launch fix are on `main`. Neither is in F2, and V1's provider scope stays Codex (D18).
- **Sub-sessions:** none running.

## Next

These items pass to the Build and evaluation session with the handover.

1. **Arm M's adapter.** Step 4's controller moves into `tools/arm-m/` as arm M's adapter for the shared variant command.
   - It runs native Apply on Linux.
   - It derives one repository identity per chain and arm.
   - It carries the state directory over as its continuation.
   - It commits the candidate with a fixed identity.
   - It reports the Apply status and cause unchanged.
   - It has offline tests, and runs no engine.
2. **A stable arm M.** One Library task runs end to end through the suite three times in a row without manual help, on a frozen build that contains the Claude adapter. This needs a bounded dispatch that the owner approves.
3. **K1b's follow-ups**, before counted suite runs on Claude:
   - One short smoke with a correction round, such as a Verify reject, so that `--resume` runs natively.
   - An allow-list instead of the deny-list for the entries agentd watches in the Manager's home.
   - A check that the access-only login outlasts a long run, since the adapter never refreshes it.
4. **Step 5, the acceptance package:**
   - Guides and code map.
   - The provider matrix, with Claude qualified once by K1b and outside D18's scope.
   - The known limits.
   - Linux CI on the frozen head and the manual Windows smoke.
   - An independent architecture review.
   - Owner acceptance follows as a separate decision.
5. **Small follow-ups:**
   - Capture refuses a working state whose Package directory is empty, because Git does not keep empty directories.
   - WP3b's ordering by dependency is deferred.
   - Whether WP1 may continue after served actions is open.

## Questions for the owner

1. **Long-lived Claude token.** Should a long-lived Claude token mode return? The recommendation is no. Keep the access-only copy of the login, because a Manager's shell could read a token that is valid for a year.

The handover question is on the [build page](build.md#questions-for-the-owner). Arm M's first dispatch passes to Build with the handover.
