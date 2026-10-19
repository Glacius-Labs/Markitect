# Recursive Apply after DEC-058 and DEC-059: three repeated demonstrations — 19 October 2026

**Scope:** three real, timed library demonstrations (M1 → A1, limit 5 → 3, M2 → A2) on the merged Main `5710c2ee`. That Main implements execute-first ([DEC-058](../concepts/register.md#dec-058-apply-always-executes-before-independent-verify)) and enforced responsibilities with the oracle outside the process ([DEC-059](../concepts/register.md#dec-059-enforced-responsibilities-build-start-and-an-outside-oracle)). The setup and harness are those of the [first record](recursive-apply-demonstration-20261019.md), with three changes:
- The required checks are `go build ./cmd/library` and `go test ./...`.
- The library oracle assesses each published Apply only afterwards.
- Each Execute container can write only its own areas and registry shard.

Implementation work, local tests, hosted CI and these product runs are reported separately. No human acceptance is recorded. This record is evidence for these runs only.

## Verdict

The behavior is now reproducible. In three consecutive runs, both Applies were verified in **one** independent Verify round each, with no blocking finding, published with read-back, and passed the oracle afterwards. The total time was 683–776 s per run, well inside the 30-minute limit. No owner question, intervention or repair was needed.

The responsibility problem of the first demonstration did not recur:
- Root only laid the foundation and coordinated.
- No file lies outside its area.
- Two attempted out-of-area writes failed on read-only mounts.

The remaining quality gap is **cross-area rules**. They are duplicated instead of being used through their owner, and fresh Verify does not block that. The evidence is n = 3 on one example with one model.

## Setup

| Item | Value |
| --- | --- |
| Product | Main `5710c2ee` (PR #472); the same frozen harness for all three attempts |
| Agents | Sonnet `claude-sonnet-5-5`, high effort, for every Execute and Verify turn, observed in the provider init. No Codex or OpenAI inference. |
| Start | M1 model, dependencies and `execution.yaml` only; no artifacts or prepared APIs |
| Layout | Root `realization`; exception `realization/cmd/library` → `library`. Enforced through Docker volume-subpath mounts. |
| Bound inputs | The two owner decisions from earlier runs, the Host system prompt (standard library only) and the owner request |
| Limits | One `timeout` outside all prompts (27.5 min, then cleanup within 30 min); 20-minute budget per Apply under a shared absolute deadline |
| Owner step | Scripted: commit A1 unchanged, change only `maxOpenLoans`, commit M2 |
| Oracle | `experiments/runtime-baseline/assess.py` after the timed window; no feedback to the Agents |

## Results

| Attempt | Total | A1 | A2 | Verify rounds A1 / A2 | Turns A1: Execute / Verify | Turns A2: Execute / Verify | Blocking findings | Oracle A1 / A2 / A2 as m1 | Cost estimate |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| RA-D2 | 683 s | 416 s | 267 s | 1 / 1 | 15 / 6 | 12 / 6 | 0 / 0 | 12/12, 12/12, 10/12 | 5.68 USD |
| RA-D3 | 776 s | 530 s | 246 s | 1 / 1 | 18 / 6 | 11 / 6 | 0 / 0 | 12/12, 12/12, 10/12 | 8.03 USD |
| RA-D4 | 719 s | 412 s | 307 s | 1 / 1 | 16 / 6 | 12 / 6 | 0 / 0 | 12/12, 12/12, 10/12 | 6.62 USD |
| RA-D1 (first record, older build) | 932 s | 652 s | 280 s | 3 / 2 | 14 / 18 | 6 / 12 | 29 / 7 | 12/12, 12/12, 10/12 | 8.72 USD |

The oracle's discrimination check passed for A1 and A2 in every run. A2 assessed against the m1 profile fails as expected, which shows that the published behavior changed with the model.

**Agent concurrency:** at most 4–5 Agent turns ran in parallel.

**A2 changes:** 5–6 files per run: lending's limit, tests, documentation and shard reason, the affected library test, and in some runs Root's README line about the limit. Members, catalog and reminders were assessed and left unchanged every time.

**Cost:** Claude Code's per-turn estimates for the runtime turns only. Billing for the OAuth runtime token was not observed.

**Separate real runs, not part of the timed demonstrations:**
- Plumbing smoke S2 (enforced ownership, before execute-first, `f94ab83c`): 435 s, 4.84 USD.
- Plumbing smoke S3 (final code `f8a3963c`): 419 s, one green Verify round, 4.26 USD.

## Observed behavior

- **Root first, then the children.** In A1, Root's first turn lasted 42–95 s. It created `realization/go.mod` and a README and sent each child the concrete basis. The children started when that turn ended, and reminders started after lending's first turn. In A2, Root's first turn (31–33 s) assessed the existing foundation and preserved it. In some runs a later Root turn updated the README line that stated the limit.
- **Coordination through the tree.** Children reported their APIs. Root relayed them to the siblings that needed them (catalog and members to lending and library), and lending relayed its query API to reminders. The engine delivered child "done" summaries to the parent, which integrated and checked the whole build.
- **Ownership.**
  - No published file lies outside its owner's area.
  - In RA-D3, Root's first turn probed its mounts with `touch` and got read-only refusals on child areas.
  - In RA-D3 A2, a lending command (`sed -i`) in a directory outside lending's area failed on the read-only mount.
  - Attribution is from the Execute transcripts kept outside the repository. It is approximate, because Agents mostly write through Bash.
- **Verify.** Every round was green at the first attempt. One advisory finding remained open (catalog availability, RA-D2 A1). No blocking finding was raised in any of the three runs.

## Artifact quality (source audit of RA-D2, A2)

A separate Sonnet agent, the same model family as the authors, audited the published A2 against the model, as in the first record. Overall **B+**, with a shifted profile:

| | RA-D1 | RA-D2 |
| --- | --- | --- |
| Fidelity | A− | A− |
| Structure / ownership | B+ | B− |
| Tests | B+ | A− |
| Documentation | B− | A− |
| Registry | B+ | B |
| A2 minimality | A | A− |

Gaps that fresh Verify did not block:
- **Membership eligibility (`members/eligibility`):** lending re-implements the rule, and the owner's function is dead code.
- **Availability (`catalog/availability`):** it is held twice, derived from loans and as a mutable flag in the catalog.
- **Registry:** `realizes` links for such cross-area rules are missing.
- **Smaller points:** stale test messages after the A2 change, and no test for existing loans above a lowered limit.

RA-D3 and RA-D4 were not source-audited.

## Assessment

| Aspect | Grade | Reason |
| --- | --- | --- |
| Reproducibility of the mechanism | A | 3/3 runs with both Applies verified in one round, oracle-clean, within the limit and without interventions |
| Responsibilities | A− | Root no longer takes over child work; enforcement blocked the out-of-area writes that were attempted. Cross-area rule placement is still weak. |
| Generated artifacts | B+ | Behavior and documentation are good; cross-area duplication and registry links remain |
| Strength of the evidence | C+ | n = 3, one small example, one model, bound oracle-derived owner decisions, estimated cost only; no owner question or interrupted run exercised |
| **Overall** | **A−** | The simplified, execute-first strategy with enforced responsibilities works reliably on this example. |

## Next levers

1. **Cross-area rules.** A rule owned by one area but enforced in another should be used through its owner and linked in the registry. Whether Verify must block duplication is a model and policy question for the owner.
2. **Broader evidence.** Use another example, a second model, and real owner questions and interruptions.
3. **Then Extract and the self-model reorganization**, as recorded on Main.
