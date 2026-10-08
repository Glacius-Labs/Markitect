# Selected-feature metadata R3

R3 ended **reported-config-and-requirements-observed** after exactly one reserved metadata process. The actual config and requirements reads completed; this is a prospective report of those fields, not an enforcement or tool-capability result.

| Actual bounded observation | Reported state |
| --- | --- |
| apps, goals, hooks, memories, multi_agent, plugins | false |
| shell_tool, unified_exec | true |
| Default permissions | :read-only; sessionFlags origin and one matching layer |
| Approval / model / provider / reasoning | never / gpt-6.1-sol / openai / high |
| Requirements object | present |
| featureRequirements | null: no reported additional feature constraint |
| allowedLoginMethods | [chatgpt] |

The precheck explicitly awaited the actual requirements response. Optional missing/null/empty managed-feature states were covered in five changed offline integration tests, alongside missing-response, malformed, unknown, conflicting and independent-origin gates. The accepted adapter is unchanged. [Additive adjudication](../evidence/s1-selected-feature-metadata-20261008-r3/r2-adjudication.md) preserves R2’s stopped outcome and corrects its protocol interpretation and log-hash typo.

Native, worker and outer exits were zero. Session 0.134s; cleanup 0.011s; controller 2.393s; outer receipt/readback 5.937s, within the fixed limits. One schema-valid disabled notification was discarded without reply. 50,145 raw stdout bytes were discarded in RAM; no raw frame/stderr log was retained. [Actual sanitized receipt](../evidence/s1-selected-feature-metadata-20261008-r3/run-1/sanitized-result.json), [timing/accounting summary](../evidence/s1-selected-feature-metadata-20261008-r3/terminal-summary.json), [independent postreview](../evidence/s1-selected-feature-metadata-20261008-r3/postreview.md), and [slot release](../evidence/s1-selected-feature-metadata-20261008-r3/slot-release.json) bind the closure.

Policy metadata trees are now 5 (4+1); CLI metadata operations remain 6. No new Actor, requested model call, product or study execution. Historical total usage and internal native network/inference remain unknown. All six comparison cells remain NOT RUN. All 3,961 prior package files and historical ledgers/receipts remain unchanged. No retry or automatic further step.

Matching values do not establish per-feature origins, inline-key recognition, active permissions, enforcement, real tools, serving identity, historical denial cause or S1 readiness. Full policy coverage, product quality and human acceptance remain unproven.
