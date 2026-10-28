# Build and evaluation

The Build and evaluation session coordinates the evaluation suite and the variants, and it is the only session that builds ([workstreams](../workstreams.md)). It owns the suite and all four arms: K, P and G since 28 October, and M since lane 1 handed it over the same day. For arm M this includes the host variant's code on `main` (the Run and Apply host, the harness, the Codex and Claude Code adapters, the Linux runner), the V1 records under `docs/validation/` and the [delivery plan](../implementation-plan.md), which owns V1's steps and known deviations. This page holds only the session's state.

The owner, 28 October 2026, on lane 1's handover: "Lane 1 finished what its doing. Can you bring it to main and then we will start discussing and managing the evaluation suite and the three execution variants from here?"

## Now

- **Suite and baseline K:** the [suite](../guides/evaluation-suite.md) runner and the Claude Code baseline are on `main` in [`tools/evaluation`](../../tools/evaluation/README.md), tested offline only; nothing has run against a provider. The assessment, review and analysis in `tools/comparison` are tied to the Library and Makerspace and do not read the suite's output yet; the old-versus-new Makerspace comparison keeps its earlier Codex `gpt-6-sol` runs. Earlier, Phases A to C compared Markitect with one agent and showed no quality advantage at 9 to 10.5 times the time ([report](../validation/v1-comparison-report-20261026.md)).
- **Arm M, V1:** the [technical acceptance record](../validation/v1-technical-acceptance-20261026.md) finds V1 correct and safe on F_final, but not yet reliable at Makerspace size; the frozen source is [F2](../validation/v1-technical-acceptance-20261026.md#new-frozen-source-f2). V1's acceptance covers the current publication; today's handoff and publication stay until the owner adopts the Git variant, which is judged after its prototype ([owner, 27 October](../concepts/sources/owner-evening-decisions-20261027.md#5-v1s-acceptance-scope-2015-utc)).
- **Arm M, Claude:** K1b, the first native Claude Code Apply, passed on its second of three allowed attempts ([record](../validation/v1-k1b-claude-qualification-20261028.md)). Its launch fix is on `main`, but no native Apply ran on `main` itself. Neither Claude change is in F2, and V1's provider scope stays Codex (D18).
- **Arm M, suite:** lane 1's repository work is all on `main`. The suite adapter is not started; lane 1's controller scripts are outside the repository under `Documents\Codex\Markitect-V1-Requalification\`.
- **Arm P:** the plugin and its runner pass 70 offline tests on `variant/plugin`. Smoke attempts 01 and 02 failed before any Apply; both causes are fixed offline. Smoke 03 waits for the owner ([plugin page](plugin-variant.md)).
- **Arm G:** the prototype passed P0 with scripted Managers, 34 of 34. An independent review says go after fixes, with ten blocking findings, recorded next to the code ([review](https://github.com/Glacius-Labs/Markitect/blob/94e28e12b73ee7ee26eff4589ee90d2c758119aa/prototypes/g0/review-20261028.md)). Its design is in [PR #795](https://github.com/Glacius-Labs/Markitect/pull/795). The prototype runs on Codex only, which is out of tokens.
- **Running:** no real run and no sub-session.

## Next

1. **Assessment on the suite's output:** the assessment, the review and the report per area read `suite.json` and `result.bundle`.
2. **First suite dispatch:** the [K smoke on the Library](../validation/eval-k-library-smoke-20261028.md) is written and its task folder is built; it waits for the owner's approval.
3. **Arm M:**
   - The adapter in `tools/arm-m/`, from lane 1's controller: native Apply on Linux, one repository identity per chain and arm, the state directory as its continuation, the candidate committed with a fixed identity, the Apply status and cause reported unchanged, offline tests only.
   - A frozen build that contains the Claude adapter, then three Library runs in a row through the suite, under a bounded dispatch.
   - K1b's follow-ups before counted runs on Claude: a short smoke with a correction round so that `--resume` runs natively, an allow-list instead of the deny-list for the entries agentd watches in the Manager's home, and a check that the access-only login outlasts a long run.
   - V1 step 5, the acceptance package ([steps](../implementation-plan.md#steps)), and the small follow-ups: Capture refuses an empty Package directory; WP3b's dependency order is deferred; whether WP1 may continue after served actions is open.
4. **Arm P:** smoke 03 when the owner approves and the engine is free, then the variant command around `armp.py`.
5. **Arm G:** fix the ten blocking findings offline on `dev/g0-prototype`; no provider run before the owner answers the provider question.
6. **One page:** fold the plugin and Git variant pages into this one as their PRs land.
7. **Before counted runs:** isolation of the variant command and a task portfolio with oracles outside the repository ([limits](../guides/evaluation-suite.md#limits)).

## Decisions

- **Model, 28 October 2026:** every arm runs on Claude Sonnet 5.5 (`claude:claude-sonnet-5-5`); Codex runs are recorded separately once Codex has tokens again. The owner: "Sonnet 5.5 for all".

## Questions for the owner

1. **First K smoke.** May the [dispatch](../validation/eval-k-library-smoke-20261028.md) run: three baseline runs on the Library, one after another, 45 minutes each at most, on the owner's Claude subscription? Recommendation: yes, now; the login is valid long enough and nothing else holds the engine.
2. **P smoke 03.** May it run once nothing else holds the engine? Recommendation: yes; it shows whether the plugin completes an Apply at all.
3. **G's provider.** The prototype runs on Codex only. Recommendation: wait for Codex tokens rather than add a Claude backend now, and fix the review findings meanwhile.
4. **Isolation before counted runs.** Variants run as the user and could read the hidden oracles. Recommendation: isolate the variants and move the oracles out of the repository before step 3, the comparison; name same-family review as a limit while no Codex reviewer is available.
5. **Long-lived Claude token for M.** Recommendation: no; keep the access-only login copy, because a Manager's shell could read a token that is valid for a year.
