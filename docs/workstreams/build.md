# Build and evaluation

The Build and evaluation session builds concept 2.0 and the evaluation suite that measures it, and it is the only session that builds ([workstreams](../workstreams.md)). It owns the suite and all four arms: K, P and G since 28 October, and M since lane 1 handed it over the same day. For arm M this includes the host variant's code on `main` (the Run and Apply host, the harness, the Codex and Claude Code adapters, the Linux runner), the V1 records under `docs/validation/` and the [delivery plan](../implementation-plan.md), which owns the concept 2.0 build steps as well as V1's steps and known deviations. This page holds only the session's state.

The owner, 28 October 2026, on lane 1's handover: "Lane 1 finished what its doing. Can you bring it to main and then we will start discussing and managing the evaluation suite and the three execution variants from here?"

## Now

- **Target:** [concept 2.0](../concepts/v2-concept.md) since 28 October; the owner handed its build to this session through the Topics session. Its steps T1 to T7 are in the [delivery plan](../implementation-plan.md#concept-20-build). The owner chose the shape: one implementation, compared with K. Nothing of 2.0 is built.
- **Suite:** E1 and E2 of [evaluation suite v2](../design/evaluation-suite-v2.md) are on `main` in [`tools/evaluation`](../../tools/evaluation/README.md): one sandbox per run with an egress limit, `judge`, `review`, `report`, the failure classifier and the token parser. The [E1 smoke](../validation/eval-e1-sandbox-smoke-20261028.md) and the [E2 smoke](../validation/eval-e2-judge-smoke-20261028.md) passed: the sandbox held, the Library judge separates its reference from its mutants, and K's runs k-02 to k-04 pass all 19 hidden cases with no critical finding. K's capture fix is confirmed.
- **New project:** authored in the private repository: a fresh build and seven changes, each with requirements, a model, a hidden judge that passes its reference and fails its six mutants, and an equivalence review. Build has not seen its content. Its models are in the concept 1.0 form, its judge image is not pinned, and its checks ran on Windows only.
- **Concept 1.0 arms:** stopped on 28 October with the shape decision: M (V1 on `main` until 2.0 replaces it), P (`variant/plugin`) and G (`dev/g0-prototype`, the start of T3).
- **Running:** no real run and no sub-session.

## Next

1. **T1, language and compiler,** in one writing session.
2. **Beside T1:** T3's hooks from the G0 prototype without its referee, and T6's suite work: change-series tasks, planted model defects and the drift check.
3. **Small follow-ups:** the new project's models converted to the 2.0 form by its author, and its judge image pinned and self-tested on Linux.

## Decisions

- **Model, 28 October 2026:** every arm runs on Claude Sonnet 5.5 (`claude:claude-sonnet-5-5`); Codex runs are recorded separately once Codex has tokens again. The owner: "Sonnet 5.5 for all".
- **Evaluation suite v2, 28 October 2026:** the portfolio is the Library as a smoke only, the Makerspace chain as the main test (its role changed the same day, below) and one new, larger, private project; the suite gets one sandbox for every variant, judges as plug-ins, enough repetitions, a reviewer of another model family when available, and human input as its own area ([design](../design/evaluation-suite-v2.md)). The owner: "Yes lets go with your recommendation and address all of these. The evaluation suite is critical and should be suitable to give us data to base our decisions upon."
- **What is measured, 28 October 2026:** the evaluation judges results against the human attention they need, not the computing they use; time, tokens and machine use are reported, not equalized. Technical failures do not count: each variant is made to run without technical issues, then evaluated ([design](../design/evaluation-suite-v2.md#b-judges-are-plug-ins-too)). The owner: "The goal is not to get better result in less computing. The goal is to get better results in less human attention." and "Technical failures shouldnt count. We should make each of them work without technical issues and then evaluate".
- **Portfolio roles, 28 October 2026:** the Library shows that a variant and the suite work technically; the Makerspace chain is an intermediate for a first idea of what works; the new, unseen project is the real test ([design](../design/evaluation-suite-v2.md#c-portfolio)). The owner: "Library is just there as a small project, to test that a variant technically works and that the eval suite works. Makerspace is an intermediate, where we can get an idea what works without much effort. As you propose the real test should be the new unseen one".
- **Hidden material, 28 October 2026:** the new project's hidden material lives in the private repository `Glacius-Labs/Markitect-Evaluation`, created the same day. The owner: "Private GitHub repo".
- **New project's domain, 28 October 2026:** a local energy community (members, meters, tariffs, shared solar credits, monthly bills). The owner: "Yes, energy".

- **Shape of the 2.0 build, 28 October 2026:** one implementation of concept 2.0, compared with K. The plugin variant and V1's open steps 4 and 5 stop; the Git variant's prototype becomes the start of T3 ([build steps](../implementation-plan.md#concept-20-build)). The owner accepted Build's proposal.

## Questions for the owner

1. **Acceptance portfolio.** Concept 2.0's acceptance names the Library and Makerspace; the suite's decision made the new, unseen project the real test, because arm M was tuned on Makerspace. Recommendation: accept on the Library and Makerspace as §15 says, and add the new project as the unseen check; its tasks then join T6.
