# Measurement

This document defines measurement procedure for Markitect product work. The [vision](vision.md) owns the quality and intent-fidelity thesis, and [operating methodology](operating-methodology.md) owns the intended working process; the [roadmap](implementation-plan.md) owns delivery order. Product tests and measurement fixtures use Go; any Markitect-owned persisted measurement record uses YAML.

## Staged method readiness

Establish a usable ontology-to-reconciliation process before measuring comparative benefit. This procedure orders new work; it does not rescore historical cohorts, erase failures or waive contribution and release gates. A real missing capability from the accepted design must be implemented, not replaced by a mock and reported as working.

Before each stage, record the specific obligation, fixed inputs, relevant scope, available capabilities, evidence method, finite exit criteria and known gaps. Select enough representative positive and negative cases to assess the named risk; do not impose a universal pass percentage or exhaustive enumeration without a reason. Repeated stochastic runs need their own predeclared uncertainty treatment. A controlled non-agent mutation and a real agent run answer different questions.

| Stage | Required demonstration | Exit evidence and limit |
|---|---|---|
| 1. Component boundaries | Canonical ownership and structural compilation; relevant context, impact and fanout; explicit target binding; bounded candidate/apply/verify; stale input refusal and drift recognition. | Independent expected outcomes and targeted negative cases for the actual components. Unit/protocol checks establish their boundaries, not real agent usability. |
| 2. One complete real change | An agent changes accepted intent through Markitect, derives affected work, materializes actual representations, receives independent verification, repairs a failure if needed and completes the accounted change. An unchanged-intent repair is also exercised. | Inspectable end-to-end evidence tied to intent and artifacts; no hand-prepared final target or mock replacing the capability being claimed. Include both a precise rule and a broad rule with declared assessment coverage. |
| 3. Recursive composition | Bounded children form a slice and a parent/project candidate using the same responsibilities. Parent obligations detect a relevant integration defect even when child checks pass. | Local and parent outcomes reported separately; one successful hierarchy shows feasibility, not reliability across arbitrary projects. |
| 4. Repeated usable operation | A small predeclared sequence combines local changes, shared rule changes, missed fanout, representation drift and routine repairs. Agents follow the declared process without constant manual steering. | Record completion, conformance, missed obligations, recovery and human intervention. State which evidence supports a reliability assessment and which remains unknown. Readiness is bounded to the tested configuration and scope. |
| 5. Comparative project backlog | Matched Markitect and conventional actors implement an equivalent realistic backlog with ordinary changes and architectural evolution. | Independent quality, drift and process assessment over repeated changes, including upkeep and intervention. This is the benefit study, not another component certification round. |

Once a stage meets its declared exit criteria, move to the next level. Reopen it for a concrete failure or material validity problem, not merely another possible test or a preference for stricter implementation detail. Keep technical prerequisites that genuinely block the next stage explicit. Bound infrastructure diagnosis to a named hypothesis and finite diagnostic attempts; unresolved timeout or environment evidence stays incomplete instead of being turned into a semantic product failure or silently passed.

Correct implementation with a stale model, correct model with a missed target, passing local checks with a broken parent, and agents bypassing Markitect are distinct negative outcomes. Preserve them separately so a change to the model, a fanout repair, an integration check or process enforcement addresses the actual problem. A repaired known failure needs fresh evidence; leave the original result intact and use fresh holdout tasks when tuning affects the comparison.

## Quality-first comparative evaluation

The primary study asks whether the method better addresses the problems it is designed for: losing intent through forgotten obligations, overlooked consequences, contradictory representations or ignored rules. Compare a governed project backlog with a well-equipped conventional agentic approach using the same accepted requirements, quality obligations, tools, model settings and authorized freedom. The conventional baseline may use normal repository guidance, architecture tests and review; do not deliberately deprive it of useful engineering practice. Markitect adds its model and reconciliation method, whose authoring and upkeep count as work.

Assess completed behavior and defects, model-to-representation consistency, rule conformity, impact/fanout misses, integration outcomes, observably followed process and necessary versus avoidable human interventions. Evaluate narrow and broad rules with the same predeclared expectations on both sides, while distinguishing Markitect-specific process requirements. Do not equate a tool-specific command count with overall compliance or favor Markitect merely because its records are easier to inspect.

Keep initial setup, model maintenance, recovery and review effort visible. Record elapsed execution and actual token/cost data as secondary observations; a slower run may still improve quality, and a faster run with missed obligations is not a success. Report uncertainty, unavailable evidence and failures alongside advantages. Synthetic checks, one controlled failure or one real task cannot establish comparative superiority. The existing controlled-comparison procedure below supplies experimental controls; a longer backlog evaluates sustained coherence and usable autonomy.

## Human attention and delegated work

The future product question is whether people can govern more autonomous engineering work without losing architectural intent. No current value for that outcome is established. Fewer interventions count as improvement only when task quality, required escalation, authority and independently checked architecture outcomes are preserved. Silent violations or skipped review requirements are failures, not autonomy.

Use repeated realistic task/change sequences with a materially simpler baseline, such as owner guidance plus existing architecture tests. Freeze task/rubric, source and policy history, actual agent/model settings, tool versions, permissions and environment. Counterbalance run order and use fresh isolated sessions. Include setup, discovery, recovery, model authoring and recurring upkeep, not just the implementation interval. A single run or an unmatched policy history cannot establish sustained attention savings.

| Future metric | Required observation and limit |
|---|---|
| Autonomous agent-hours / human intervention-hours | Record actual execution intervals and measured human attention separately; distinguish waiting and overlapping parallel work. An open agent session is not active work. Report both quantities and unavailable or zero denominators; do not substitute elapsed workflow time for attention. |
| Completed changes / human review actions | Count independently assessed task outcomes and actual human reviews, corrections and acceptance decisions. Keep automated agent reviews separate. A green command or merged PR is not automatically an accepted change. |
| Architecture-compliant changes before human intervention | State the finite model assertions and specialist implementation checks that ran, their fixed inputs and independent review coverage. Keep passed, failed, waived, incomplete and unknown evidence distinct; do not call modeled conformance complete implementation correctness. |
| Architectural escalation and intervention | Record decisions versus routine clarifications, source/scope correction, steering, manual review, reconciliation and recovery. Score missing necessary escalations and unnecessary escalations against an owner-defined rubric. Fewer events alone is not better. |
| Human time allocation | Measure implementation, routine review/synchronization, specification, architecture decisions, initial setup and recurring model/adapter maintenance. Record total attention as well as its distribution; moving effort into YAML authoring is not automatically a saving. |
| Synchronization or reconciliation work eliminated | Identify a real prior manual step and owner, the replacement operation, outcome and subsequent maintenance. Generated file counts or several views of one owner do not prove a truth source or human step was removed. |
| Drift detection and context/impact usefulness | Predeclare expected rules, violations, inputs and affected sets where independently knowable. Record misses, unnecessary inclusions, review noise and causes. Separate direct policy subjects from conservative impact; unassessed resources are not automatically false positives. |

Record human interventions where they occur and preserve failures, exclusions, missing evidence and later corrections. Measure repeated architecture evolution as well as ordinary implementation so package, exception, mapping and projection upkeep is visible. Tokens and model/tool calls remain secondary observations only when instrumented; bytes, helper read events and agent duration are not token usage or cognition.

The [real-code pilot](validation/real-project-adoption-pilot.md), [matched comparison](validation/agents-md-vs-markitect.md) and [consumer inventory](validation/parallel-wave-konfyra.md) retain contrary evidence and missing instrumentation. They do not establish human-attention reduction. Future studies must allow the simpler baseline to win and ask whether modeling costs more than the consistency work it replaces. This section specifies evaluation, not a telemetry implementation, benchmark run or automatic acceptance policy.

## Evaluating the method separately from its runtime

This is a proposed procedure. Markitect's product owner endorsed it on 10 October 2026, and the plan is still pending ([DEC-010](concepts/register.md#dec-010-evaluate-the-method-separately-from-its-execution-runtime)). Until the owner decides the plan, studies of the delegated method should follow it.

The object of evaluation is the delegated method: computed scope, bounded Managers, independent review, integration, verification and guarded Apply. One particular execution runtime is not the object. The October 2026 comparison attempts were dominated by environment, runtime and harness failures and allow no conclusion about the method ([DEC-008](concepts/register.md#dec-008-the-delegated-method-has-not-yet-been-tested)). Earlier matched comparisons evaluated earlier product forms; their findings stand.

- **Stable environment.** Use a stable, isolated environment that is the same for every arm. Current evidence points to Linux. Classify failures as follows; neither kind is an outcome of the method:
  - A failure of Markitect's own runtime is a product finding, recorded with its cause.
  - A harness or environment failure is a measurement limitation.
- **The product supplies the method.** A study harness chooses and configures the executors and models that the product offers. It does not reimplement Markitect's scoping, review, integration or verification. This follows the [work-item comparison decision](design/work-item-comparison-20261009.md). The product-side boundary for exchangeable executors is [ENH-005](concepts/register.md#enh-005-exchangeable-executor-for-the-delegated-method).
- **Change sequences.** Apply the repeated, realistic change sequences required [above](#human-attention-and-delegated-work) to one long-lived project.
  - Include cross-cutting rules, a refactoring and a late change that touches earlier decisions.
  - Predeclare ground truth where it is independently knowable: the obligations, files and Managers each change should affect; seeded cross-cutting rules; and hidden holdout checks. Whether to focus on such series is still an open question ([OQ-005](concepts/register.md#open-questions)).
- **What to score.**
  - Missed and unnecessary inclusions, rule violations and contradictions between representations.
  - Regressions, required and unnecessary escalations, and human attention.
  - Cost as a secondary observation.
- **Configurations as factors.** Treat executor choice and per-role model choice as explicit experimental factors once the product offers them ([ENH-004](concepts/register.md#enh-004-per-role-agent-configuration-and-model-mixing)). Keep each run's actual configuration fixed and recorded.

## Correctness first

Use a synthetic project with explicit shared policy and two separately scoped areas. Record expected effects before running the tool. Cover local and shared Rule changes, removed dependencies, Contract-binding changes, and an unmodelled input change. Unknown inputs must remain conservative until they are declared.

Go scenario tests compare exact expected context and impact sets. Go benchmarks measure deterministic operations after fixture construction. They can report allocations, context bytes, and affected-resource count; they do not measure human authoring speed or model savings. Keep the source revision, command, platform, and raw output together in excluded development artifacts.

```powershell
go test ./internal/app -run AuthoringScenario -count=1
go test ./internal/app -run DocumentationScenario -count=1
go test ./internal/app -run '^$' -bench Authoring -benchmem -count=3
```

The [code and documentation example](documentation.md) exercises exact source-file inputs with a predeclared affected set and an unrelated area. Its source-only change intentionally leaves stale prose structurally valid: the scenario verifies routing to review, not semantic correction by the compiler.

CI executes one iteration of each authoring benchmark to keep the measurement fixtures valid as the resource model evolves. This is a correctness smoke check, with no timing threshold. For timing reports, run the repeated command above on a fixed source commit and record the host; the measured section excludes Git reads, parsing, agent interaction, and provider calls.

After each immutable published release at v0.9.0 or later, `.github/workflows/release-benchmark.yaml` runs the current and immediately previous attested platform binaries on Windows and Linux. Current releases use `benchmark/fixtures/v2`; the previous binary uses v1 when its release predates 0.9.0 and v2 otherwise. The immutable v1 fixture preserves the pre-0.9 layout, while v2 exercises the selected Markdown output with canonical resources beneath `.markitect/areas/sample/` and an ordinary input beneath `docs/`. The harness gives both binaries one shared prepared repository when fixture version and bytes match. Across a version transition it prepares each compatible fixture independently, records the version, root, digest, base commit, and candidate commit for each release, and suppresses performance-change percentages because workloads differ. See the [benchmark contract](../benchmark/README.md) for invocation and metric definitions.

Each command (`check`, `render`, `context`, `impact`, and `verify`) runs once as the first fresh process and three more times as fresh processes, with release order alternating. Records include wall time, sampled peak working set, exit status, status text, and bounded output. JSON and Markdown summaries are workflow artifacts; no asset is added to the immutable release. A maintainer can dispatch the workflow with an exact published stable tag to repeat a failed measurement; it verifies the same immutable release and assets without changing them. The first run does not clear operating-system caches, and a short process can exit before memory sampling, which is recorded as null. There is no performance threshold or release-blocking benchmark gate.

## Authoring exercise

Copy the complete minimal example into an isolated Git repository and commit a baseline. Give an agent a normal requirement, access to the CLI, and the bundled `authoring` context. An independent reviewer checks the changed owner, required relationships, unaffected scopes, generated outputs, and fixed impact. Retain failures and corrections. This is a product usability exercise, not an A/B benchmark or provider-runtime certification.

## Controlled model comparison

Prepare the task oracle outside the actor's context. Run baseline and Markitect variants from identical fixed snapshots with the same model, effort, permissions, and task. Use separate workspaces and fresh sessions; include discovery and recovery costs. An optional check-only variant can isolate the effect of context and impact queries.

Start with a small set of local Rule edits, shared rules, and dependency changes, then extend to renames, binding/scope changes, code-to-document inputs, provider-output drift, and stale references. Repeat each variant at least three times before describing a trend. Report successes and invalid runs alongside duration and token counts; separate correctness from speed.

Capture only observed data: model-reported token usage, calls, tool calls, searches, files read, wall time, context bytes, affected entries, review reuse, and human interventions. Missing values remain unavailable; do not estimate tokens from bytes. Context/impact precision and recall require a known expected set. Report undefined ratios with numerator and denominator rather than substituting 100%.

Review reuse should be exercised for identical inputs, relevant changes, removed edges, tool/config changes, and unknown files. Reuse makes no model call. Compare the full repeated task before claiming sustained savings.

The [2026-10-01 documentation pilot](authoring-pilot.md) records an incomplete first exercise and a six-run repeat with the mutation boundary stated in the task card. All six repeated candidates passed fixed structural and output checks and the four-path boundary. The blind semantic review found a further wording difference between task and rubric, while token and cost measures remained unavailable. State every scored requirement in both task card and review rubric before running another comparison; keep structural validity, requested semantics, and a narrow mutation boundary as separate results.

## Optional local records

No telemetry is installed or enabled. Consider local opt-in YAML records only after a controlled exercise establishes useful fields. Counts, durations, identifiers, and hashes are enough; prompts, credentials, and source documents do not belong in generic metrics. Do not rank individual contributors. A statistics CLI, remote collection, and database are separate future decisions.
