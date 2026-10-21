# D input and execution preparation, 21 October 2026

This is provider-free preparation for the owner's four selected D slots: **A2 preservation, ARTIFACTS, DEP-SEMANTIC and R1**. It authorizes no execution. A's separate small Codex retest and C's method/harness retain their own scopes and budgets. Earlier #564/#580 inputs, historical attempts and evidence remain identifiable in Git and their original records.

## Inputs and independent controls

[Workshop](workshop/README.md) supplies the one R1 model-only case, W-G. Its actual count is 26 Definitions, six named Packages and seven Manager roles, including Root. A blind model-only review found no material contradiction in the valid workflows. A separate reference/oracle review independently recomputed invoice, half-even VAT, loyalty-discount and report amounts. The ambiguous meaning of semantically malformed persisted records remains an explicit coverage limit; it is not settled by assuming the reference's internal JSON schema. W-N and the three isolated W-M faults provide positive/negative controls, not extra D execution slots. No meter, CX-SEM, Depth or Fan inputs were adopted.

The other three slots reuse existing inputs and their controls. The artifact slot retains the isolated green a2/issue-exact and a4/list-order controls. DEP-SEMANTIC retains H-I's locally green children and failed parent reserve agreement, separately from DEP's routing/version criterion. A2 retains its unattractive conforming D-U stand. The evaluator's ownership grouping now recognizes physical child directories such as `realization/rooms/` and `realization/billing/` beneath their exceptional parent Package; this changes grouping, not packet bytes or intent.

## Frozen input manifest

From the repository root, after committing the source candidate and refreshing the lock:

```text
python -B experiments/apply-trials/freeze.py lock
python -B experiments/apply-trials/freeze.py check
python -B experiments/apply-trials/d_series.py prepare ABSOLUTE_EXTERNAL_DIRECTORY --series-id EXPLICIT_D_SCOPE
```

`prepare` creates the twelve matched writer workspaces, separate hidden control packets and `manifest.json`. It verifies each scenario packet head against the lock and records complete file/tree/head hashes, source commit, kit/lock, case version, oracle profile, policy, budgets, existing assessment controls and fixed arm order. Each slot rotates direct/staged/plan order before execution. There are no retry slots. The shared dependencies and intended content are identical between Markitect strategies; only explicit direct/staged selection differs. R1's 180-minute, 600-Agent-Turn/12-Pass packet does not expand D's total ceiling.

Writer mounts contain only the selected packet. For the ordinary arm, `plan.md` contains the full generated requirements/provenance and is mounted read-only over the writable workspace. Reference implementations, checks, faults, other arms, hidden controls, manifest and ledger remain outside that mount. The actual Markitect CLI independently checks model preparation and plan coverage; a second rendering by the same formatter is not that proof.

`--bindings JSON` optionally binds operator-supplied profiles. Provider initialization requires direct/staged/plan descriptors with provider, exact model/effort, immutable image, observed profile digest and equal CPU/memory/process bounds. Markitect descriptors additionally bind the product binary/revision and run configuration. All referenced file digests are checked, and actual profile contents must agree with the descriptors for every Agent role. For this static comparison, the actual Runtime file passed to Apply must use JSON encoding of the existing markitect.runtime/v1 YAML schema (JSON is a YAML subset), with agentProfiles containing the existing codexAppServer or claudeCode sections. A sidecar assertion is insufficient. Existing YAML-producing wrappers must emit that actual JSON input before freezing; this harness does not replace the product profile validator. Ordinary profile bytes and compute settings are checked again at execution. No real profile/image/model is silently selected by this document; absent values remain a readiness dependency.

## One ordinary conversation and its result

[plan_arm.py](plan_arm.py) starts exactly one containerized coding conversation. It adds no Manager, separate assessor, reviewer team or owner-answer callback. A prepared native Codex profile uses one fresh `codex exec` invocation and JSONL completion evidence, as specified by [OpenAI's non-interactive documentation](https://learn.chatgpt.com/docs/non-interactive-mode). It disables `multi_agent`, apps and memories using the flags already present in this repository's pinned Codex adapter. No resume is accepted. The reported unit is **Codex input Turn**; it is not equated with every model/tool iteration or a Markitect Agent Turn.

Claude can instead use its native print JSON result, whose failure result must not be treated as success ([programmatic usage](https://code.claude.com/docs/en/headless)). A separately frozen profile uses the selected executable with `-p`, `--output-format json`, `--model {model}`, `--effort {effort}`, `--max-turns {maxTurns}`, and an ordinary tool allowlist excluding Agent/Task. Verify those flags on the pinned image before setting its transport verification field. A third explicit `plan-agent/v1` adapter contract is supported, but no unavailable adapter is presumed. There is no new Claude subscription allowance.

[The example profile](run/plan-profile.example.json) is deliberately incomplete and refuses provider execution. A real profile needs an exact engine executable/hash, immutable image, frozen provider/model/effort, argv, bounds and separately verified single-Agent turn-bound behavior. Individual credential files may be mounted read-only under `/run/provider-auth/`; no source, control or preparation tree can be used as an authentication mount. Provider homes are fresh temporary container filesystems. No broad host home or engine socket mount is provided. Both temporary filesystems have explicit 64 MiB limits. CPU, memory, process, output and wall limits are bounded; the writable host workspace has no disk quota. Disk usage therefore remains unenforced and must be resolved or explicitly accepted in the later execution grant. Preparation does not claim complete resource isolation.

**Ordinary publication definition:** the result is the closing working tree against the frozen start HEAD, not a Markitect Verified publication and not a new commit. The Driver records first completion once, removes its owned container, checks that HEAD and authoritative `plan.md` are unchanged, and hashes the final workspace. No cleanup or repair follows the first completion. The external oracle/tests then score that stand; the Driver does not give the Agent their answers. Timeout, nonzero exit, excessive output, malformed/missing provider ending, changed protected inputs or uncertain cleanup are unsuccessful endings. Output, diagnostics, provider session/usage when supplied and the workspace are retained; no automatic second attempt occurs.

## Finite scope and measurement

Initialization needs a separate owner authorization file containing `seriesId`, the exact `manifestSha256`, `authorizationReference`, explicit `mode` (`simulation` or `provider`), `maxStarts` and `maxSeconds`. The tool refuses more than 12 starts or 43,200 summed seconds and refuses an existing scope directory. Preparation itself writes `executionAuthority: false`. These are available mechanics for later authorized work, not a grant from B.

```text
python -B experiments/apply-trials/d_series.py init MANIFEST_JSON LEDGER_ROOT --authorization AUTHORIZATION_JSON
python -B experiments/apply-trials/d_series.py reserve LEDGER --scenario A2 --arm direct
python -B experiments/apply-trials/d_series.py finish LEDGER ATTEMPT_ID --result RESULT_JSON
```

Reserve before any acquisition, binding or provider start. The append-only ledger fsyncs a unique attempt id, order position, start timestamp and time reservation, capped by both the scenario and remaining series budget. A pending start blocks the next one. Missing endings require explicit abandonment and consume the full reservation; broken/stale locks are never removed automatically. Observed elapsed time is charged including overhead and recorded overruns; it is not silently clipped to the reservation. Every failed start consumes its slot. A new ledger path, scope name, changed profile, replay or authorization file is not a reset of an earlier owner allowance.

The direct/staged wrapper owned by A/D consumes this reservation and returns a result with `attempt`, `scenario`, `arm`, `mode`, `status`, `manifestSha256`, frozen `head`, and optionally `reportPath`/`reportSha256`. Completed endings must match the manifest/head; report bytes must match their declared hash. This is an external harness handoff, with no product-core change or replacement Apply engine. The plan Driver uses the same ledger and creates its final record itself.

```text
python -B experiments/apply-trials/plan_arm.py --ledger LEDGER --scenario SCENARIO --workspace FROZEN_PLAN_WORKSPACE --profile FROZEN_PROFILE_JSON --output NEW_EXTERNAL_OUTPUT_DIRECTORY
python -B experiments/apply-trials/d_series.py measure --report APPLY_JSON --events EVENTS_JSONL --final OPTIONAL_PLAN_FINAL_JSON --observations OPTIONAL_OPERATOR_TIMING_JSON
```

Measurement binds input-file hashes and separates wall elapsed, recorded engine span, native Turn unit, raw usage, known USD cost, observed container creation/start/removal, and unknown values. Zero default USD is unknown without a known-cost marker. Markitect Turn counts do not prove container starts. A wrapper can supply observed container/timing data in `observations`; otherwise those values remain unknown. Damaged event lines are counted. Prep, artifact verification and owner work outside the provider invocation must be recorded separately rather than donated to an arm.

## Evidence boundary and remaining dependency

Local process controls cover success, missing executable, failure, timeout, output limits, protected-plan changes, existing-output preservation, frozen identity, scope accounting and simulated provider-shaped records. Simulation has a separate ledger mode and final evidence label, which the evaluator retains; it cannot run through a provider ledger. These controls are not real provider, CLI-in-container, authentication, image or subscription evidence.

Before D execution: freeze the integrated A/C product build, actual three-arm profiles/images/model/effort/order and a new finite owner grant; validate ordinary native CLI/credential delivery and turn semantics on that frozen image within an explicitly authorized probe. This implementation needs no change to A's core. A's small full-Apply Codex retest does not by itself prove the ordinary `codex exec` transport or authorize D. Current readiness is **prepared mechanics, real transport unproved**. Detailed local results and integration hashes are recorded in the delivering PR/chat and external evidence directory, without rewriting old attempts.
