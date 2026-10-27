# Evaluation suite (lane 4)

The evaluation suite gives every Markitect variant and the classic baseline the same job and judges every outcome the same way. A variant joins the suite through one arm interface: the same inputs, the same output bundle, and then one shared assessment, review and analysis that no arm runs or influences. This page holds the lane's scope, the interface, a short state and its open owner questions, and links to the pages that own the details.

The lane exists since the owner split Markitect's work into lanes on 27 October 2026. Its lead is the "Lane 4: Evaluation suite" session.

## Scope

- **Owns:**
  - [`tools/comparison`](../../tools/comparison/README.md): the shared assessment ([`assess_candidate.py`](../../tools/comparison/assess_candidate.py)), the review rubric and prompts, the ledger and [`analyze.py`](../../tools/comparison/analyze.py).
  - The arm interface on this page and its shared checks.
  - The baseline arm K, classic agentic coding: one agent with ordinary documentation of the same meaning. Today's runner is [`runner.py`](../../tools/comparison/runner.py) (arm B in the [comparison protocol](../validation/v1-comparison-protocol-20261024.md)).
  - The task portfolio and the study designs of later suites.
- **Uses, does not own:**
  - The hidden oracles in [`tools/library-oracle`](../../tools/library-oracle/README.md) and [`tools/makerspace-oracle`](../../tools/makerspace-oracle/README.md). They change only through their own reviewed PRs, never during counted runs.
  - The arms of the other lanes. Each lane owns its arm's adapter; the suite owns only the contract the adapter meets.
- **Arms:**

  | Arm | Variant | Lane | Adapter today |
  |---|---|---|---|
  | M | Markitect with the host as coordinator | 1 | Step 4's controller (`requalify.py`), kept in the evidence root, outside the repository |
  | P | Markitect Apply as a Claude Code plugin | 2 | [`armp.py`](https://github.com/Glacius-Labs/Markitect/blob/c76d2b77/tools/comparison/armp.py) on branch `dev/markitect-plugins-assessment-779f8e`, not on `main` |
  | G | Agents collaborate through a Git remote ([DEC-078](../concepts/register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review)) | 3 | None yet; G is `markitect apply` from lane 3's branch, so arm M's adapter with another binary |
  | K | Classic agentic coding, the baseline | 4 | `runner.py` |

## The arm interface

Every arm is one command that runs one task and writes one output directory. Everything before it (building the task) and after it (checking, assessing, reviewing, analyzing) is shared and identical for every arm.

```text
ARM run --task TASK.json --out NEW_DIR
```

### Inputs: the task

The suite writes `TASK.json`. An arm reads nothing else from the suite.

| Field | Meaning |
|---|---|
| `taskId` | Suite, project, chain and step, for example `suite-1/library/chain-2/S3` |
| `start` | The target repository as a Git bundle, its commit and tree. The arm works on this commit and nothing else |
| `intent` | The step's intent in both forms, built from one canonical model snapshot: the model (`.markitect/`, with the execution configuration) and the ordinary document (`requirements.md`) rendered from it with the renderer's equivalence check. A model arm (M, P, G) gets only the model in `start`; arm K gets only the document. The same meaning is the fairness condition, proven by the renderer's check |
| `provider`, `model`, `effort` | The model every arm of the suite uses, for example Codex Luna at high effort ([owner decision](../concepts/sources/owner-morning-decisions-20261027.md)) |
| `limits` | Wall deadline, aggregate CPU, memory and process ceiling, network policy, and the shared engine lock. Equal for all arms |
| `login` | The path of the account login. The arm passes it on without its refresh token and never writes it to the output |
| `previous` | The arm's own output directory of the chain's previous step, or none. What an arm carries over (K resumes its thread, M keeps its state directory) is its own business, but it reads only its own previous output |

The suite, not the arm, builds the start of every chain step: the arm's own previous candidate, or its previous start when there was none, plus one commit with the step's intent change. Both arms of a chain therefore start each step from their own work under the same rule.

### Output: the run bundle

The arm writes `NEW_DIR`, which must not exist before:

| Path | Content |
|---|---|
| `run.json` | The run record, below |
| `start.bundle` | The start it received, unchanged |
| `result.bundle` | The candidate: one commit that descends from the start. Absent when there is no candidate |
| `transcripts/` | Every agent session's raw transcript. Usage and the model actually used are recomputed from these |
| `continuation/` | Whatever the arm needs for the next step. Opaque to the suite |
| `arm/` | Arm-specific evidence, such as an Apply report, a state directory or `work.bundle`. Opaque to the suite |

`run.json` holds:

- `arm`, `armRevision` (commit and digest of the arm's code or binary), `taskId`;
- `provider`, `model`, `effort` as requested;
- `start` and `candidate` as commit and tree, with the bundles' SHA-256;
- `outcome`: `candidate`, `no_candidate`, `model_blocker` (with the blocker's text) or `infrastructure_error` (with its cause);
- `startedUtc`, `endedUtc`, `wallSeconds`;
- `usage` as the arm reports it: input, cached, output and reasoning tokens, sessions, turns and tool calls;
- `retries`: inside the product and by the arm's runner, counted separately;
- `quarantine`: the result of the arm's own secret scan.

### Shared after the arm

1. **Run check.** A shared checker validates `run.json`, recomputes every hash, checks that the candidate descends from the start, that the intent inputs are unchanged, that the transcripts name the requested model, that usage matches the transcripts and that no secret is in the output. A run that fails is an infrastructure error and never counts as a candidate.
2. **Assessment.** [`assess_candidate.py`](../../tools/comparison/assess_candidate.py) from the frozen assessor revision: the hidden oracle and probes, the candidate's own build and tests in the frozen Go image, the change census and a blinded copy.
3. **Review.** The blinded rubric review, the mechanical check of critical findings ([`rubric_check.py`](../../tools/comparison/rubric_check.py)) and a confirming reviewer. A model blocker is classified here as justified or false.
4. **Ledger and analysis.** One builder turns run check, assessment and review into ledger rows, and [`analyze.py`](../../tools/comparison/analyze.py) applies the suite's predeclared reading. The arm never writes a ledger row.

The outcome of a run is decided by these shared steps, never by the arm's own report.

## State

- **Done:** Phases A, B and C compared arm M with one agent on the Library and Makerspace ([report](../validation/v1-comparison-report-20261026.md)). No quality advantage for M was shown; M cost 9 to 10 times the time and 9 to 42 times the tokens.
- **Shared tools:** assessment, review and analysis exist and are tested. They are tied to the Library and Makerspace profiles and to a ledger builder that lives in the evidence root, not in the repository.
- **Arms:**
  - K's `result.json` and P's `result.json` are close to the run record; neither writes `run.json` yet.
  - M's adapter is outside the repository.
  - G has no adapter.
- **Pilot:** the Makerspace old-versus-new gate cells stay on `gpt-6-sol` and are not rerun on Luna ([delivery plan](../implementation-plan.md#acceptance-and-evaluation-owner-directive-25-october)).
- **Study design:** a draft evaluation contract (estimands per task type, fixed sample size, calibrated instruments, a model-quality gate, new projects written after the source freeze) and a draft follow-up study on consistency over long change chains exist as the lead's working drafts and are not yet in the repository.

## Next

1. **Agree the interface** with the leads of lanes 1, 2 and 3.
2. **Shared parts on `main`:** the `TASK.json` and `run.json` schemas, the run checker, the chain-start builder and the ledger builder in `tools/comparison`, with offline tests. No engine runs.
3. **Adapters:**
   - K: `runner.py` writes `run.json` (lane 4).
   - P: `armp.py` writes `run.json` (lane 2).
   - M: step 4's controller moves into the repository as arm M's adapter (lane 1).
   - G: lane 3's binary through M's adapter, once G1 lands.
4. **Portfolio and contract:** the evaluation contract and the follow-up study go into `docs/validation/` for owner review; new projects and their oracles follow it.
5. **First bounded dispatch,** after the owner's approval: one Library task per arm on the suite's model, as a smoke of the interface, not as a counted run.

## Open owner questions

1. **Arm P and the suite's model.** Arm P runs only in Claude Code, so it cannot run on Codex Luna. The recommendation: a suite fixes one provider and model for every arm; P takes part in a Claude suite in which M runs through the Claude adapter (K1) and K through one Claude Code session, and Luna suites hold M, G and K. A Claude suite uses the owner's Claude subscription.
2. **The baseline's continuation.** Does K continue one agent thread across a chain (as in Phase C) or start a fresh session per step with the repository as its only memory? The recommendation is to decide after the early probe that compares both on Phase C material.
3. **The evaluation contract.** Does the owner want the contract reviewed and approved as binding before the first counted suite? The recommendation is yes, so that the decision rule is fixed before any data.
