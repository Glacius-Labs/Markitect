# Build and evaluation

The Build and evaluation session coordinates the evaluation suite and the variants, and it is the only session that builds ([workstreams](../workstreams.md)). It owns the suite and arms K, P and G now, and arm M once lane 1 hands over. This page holds only its state; the suite, the variants and the delivery plan keep their details on their own pages and branches.

## Now

- **Suite and baseline K:** [PR #793](https://github.com/Glacius-Labs/Markitect/pull/793) holds the suite runner, the Claude Code baseline and the evaluation page. 18 offline tests pass and both reviews are applied. It is not on `main`, has had no Linux CI and has never run against a provider.
- **Arm M (lane 1):** K1b, the first native Claude Code Apply, passed on its second attempt after the launch fix in [PR #807](https://github.com/Glacius-Labs/Markitect/pull/807); the record is [PR #808](https://github.com/Glacius-Labs/Markitect/pull/808). The suite adapter for M is not started.
- **Arm P:** the plugin and its runner pass 70 offline tests on `variant/plugin`. Smoke attempts 01 and 02 failed before any Apply; both causes are fixed offline. Smoke 03 waits for the owner ([plugin page](plugin-variant.md)).
- **Arm G:** the prototype passed P0 with scripted Managers, 34 of 34. An independent review says go after fixes, with ten blocking findings, recorded next to the code ([review](https://github.com/Glacius-Labs/Markitect/blob/94e28e12b73ee7ee26eff4589ee90d2c758119aa/prototypes/g0/review-20261028.md)). Its design is in [PR #795](https://github.com/Glacius-Labs/Markitect/pull/795). P1 and P2 are paused by the owner until lane 1 is done, and the prototype runs on Codex only, which is out of tokens.
- **Running:** no real run and no sub-session.

## Next

1. **Suite on `main`:** Linux CI on #793's head, freeze, merge.
2. **First suite dispatch:** build the Library task folder with `suite.py task` and write the bounded dispatch for three K runs in a row on Claude Sonnet 5.5, then ask the owner.
3. **Arm M:** after the owner says lane 1 is done, take over the adapter `tools/arm-m/` and a frozen build that contains the Claude adapter, then three Library runs through the suite.
4. **Arm P:** smoke 03 when the owner approves and the engine is free, then the variant command around `armp.py`.
5. **Arm G:** fix the ten blocking findings offline on `dev/g0-prototype`; no provider run before the owner answers the provider question.
6. **One page:** fold the evaluation, plugin, Git and host variant pages into this one as their PRs land.

## Questions for the owner

1. **Lane 1 handover.** K1b passed. May lane 1 merge #807 and #808, hand arm M to Build and end? Recommendation: yes; it unblocks arm M and ends the pause on G.
2. **P smoke 03.** May it run once nothing else holds the engine? Recommendation: yes; it shows whether the plugin completes an Apply at all.
3. **Model for the comparison.** Recommendation: Claude Sonnet 5.5 for every arm; Codex runs are recorded separately and return when Codex has tokens again.
4. **G's provider.** The prototype runs on Codex only. Recommendation: wait for Codex tokens rather than add a Claude backend now, and fix the review findings meanwhile.
5. **Isolation before counted runs.** Variants run as the user and could read the hidden oracles. Recommendation: isolate the variants and move the oracles out of the repository before step 3, the comparison; name same-family review as a limit while no Codex reviewer is available.
