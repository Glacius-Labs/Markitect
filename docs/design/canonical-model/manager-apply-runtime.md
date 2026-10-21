# Implemented runtime contract of Apply

**Status, 20 October 2026:** this page describes the base Pass implementation integrated by PR #530, Main `780dccb7`, of the [accepted workflow](manager-apply-workflow.md) ([DEC-064](../../concepts/register.md#dec-064-checked-recursive-apply-passes-and-manager-teams)), with the [evidence-driven hardening](manager-apply-workflow.md#evidence-driven-hardening-adopted-20-october-2026) of the [hardening assignment](../../development/claude-apply-hardening-handoff-20261020.md): Orchestrator-produced probe receipts, claim-level evidence, independent dispositions of adverse receipts, active counterexample mandates, waits on any Manager and early ends of unsatisfiable waits. It supersedes draft PR #536, whose observation fields and cycle regression it carries in changed form. Its [first provider runs](../../validation/apply-pass-process-20261020.md) include successes and a Verified result retaining a genuine defect; the bounded hardening evaluation is recorded in the [hardening report](../../validation/apply-hardening-20261020.md). Stable Apply and owner acceptance are not established. The earlier contract (one Execute phase, Verify rounds, start-drift repair, question ban and `model_defect`) remains in [Git at Main b59abad3](https://github.com/Glacius-Labs/Markitect/blob/b59abad30409601075f4a597457b707cbf47ba72/docs/design/canonical-model/manager-apply-runtime.md); its earlier real runs retain their original build scope.

**Accepted target, 21 October 2026:** the [mandatory assessment/closed-Run amendment](manager-apply-workflow.md#mandatory-model-assessment-and-closed-runs) requires a read-only Model assessment through this same Pass mechanism, and removes the external in-Run owner-answer route. Material Root-confirmed gaps/errors terminate without publication; corrected model input starts a new Run. Every production strategy retains Managers. This page describes the earlier implemented baseline below; its owner-question API is not an exception to the new target. [A owns production/self-model migration; C supplies the method](../../development/claude-direct-runtime-handoff-20261021.md#current-policy-supplement-mandatory-assessment-and-closed-runs). No new implementation or provider evidence is claimed here.

The packages own their details: [run](../../../internal/run/README.md), [apply](../../../internal/apply/README.md), [realization](../../../internal/realization/doc.go), the [Claude](../../../internal/adapters/claudecode/README.md) and [Codex](../../../internal/adapters/codexappserver/README.md) adapters.

## Organization and Briefs

The organization is derived, not planned: Root (`.`) and one Manager per local Package, including empty and intermediate ones, with parents from the model hierarchy. Every Manager takes part in every Pass. `run.Config` lists Root first and every parent before its children.

Apply renders each Manager's Brief from the captured model: its own Definitions (its obligations; every result covers each of them), included Definitions labeled with their owning Package, its areas, its local registry and the required checks. Root also receives the project obligations explicitly: every local Definition with its kind, title and owner, and the full text of the Definitions the required checks name. A parent's Brief names each child's areas and Definitions and the Managers below them. Prompts carry the task, inputs and answer format; the process belongs to the Orchestrator ([DEC-051](../../concepts/register.md#dec-051-agents-get-tasks-the-orchestrator-owns-the-process)).

## Ownership, local registries and file accounting

<a id="candidate-accounting-and-protected-inputs"></a>

The Run's authority inputs, the model and its dependency locks, the ownership, the execution settings, the ignore policy and the invocation's authority, are immutable. Ownership is the `ownership` list of `.markitect/execution.yaml`: repository-relative areas, each an exact file, a directory ending in `/` or `./` for the whole repository, owned by Root or one local Package. The most specific area owns a path: an exact file above any directory, a deeper directory above a shallower one. Two areas on one path, a path declared as both file and directory, anything under `.markitect` or `.git` and letter-case collisions are refused. A file no area covers has no owner and fails File check as `layout.no_owner`. `look_up` gives every Manager the same owner for any path, including a new one.

Each Manager writes only its own local registry, `.markitect/registry/registry.yaml` for Root and `.markitect/registry/<package>/registry.yaml` otherwise. Registries are owner-writable Markitect metadata, not authority inputs: an owner changes its registry in a write-enabled Pass, and acceptance binds the shard with its artifacts. An entry is an exact file or a directory ending in `/`; a directory entry covers only its registrant's eligible files below it, other owners' areas are excluded, and the most specific entry registers a file. An entry outside its registrant's area, a missing file, a directory that registers nothing, an unknown Definition, a duplicate entry path and an unexplained file block, each addressed to the Manager that must act.

File check is strict at the start: every accounting finding fails the preparation, and Apply starts nothing. There is no tolerated start drift ([DEC-062](../../concepts/register.md#dec-062-start-stand-drift-is-the-owners-first-work) is superseded). `markitect check` applies the same accounting; without declared ownership it accepts registry-only accounting, which Apply refuses.

Only the Orchestrator assembles the derived registry view (`run.RegistryView`): the content version of every included shard and the effective registration of every artifact. Every acceptance records the shard versions it binds, every Pass seal the full view; `look_up registry` exposes it read-only.

## Write plans and enforcement

Every turn of every Agent runs in its own container with the shared tree at `/work`. A turn may write only in a Pass with writes while its Manager works on its own basis or result. Its write plan mounts `/work` read-only (or writable for an owner of `./`) and overlays, parents before descendants and only where writability changes, its own directory and exact-file areas and its registry directory writable, and `.markitect`, protected inputs and every other owner's areas inside them, directories and exact files, read-only. An exact-file area that needs its own overlay must exist in the start stand: a missing file cannot be mounted, so Apply refuses that ownership shape with `ErrUnsupportedOwnership` instead of trusting prompts. Apply also refuses to start when the container engine does not mount volume subpaths (Docker server 26 or newer). `TestRealEngineEnforcesFileAndDirectoryOverlays` shows on a real engine (Docker 29.4.1) that writes, removals and renames through read-only file and directory overlays fail, that a writable directory still accepts new files, and that a writable file overlay changes its file but cannot create a neighbour.

## Workflow composition

`apply.Apply` checks the capture, binds the organization, write plans, checks and budget, opens the run and composes the Passes:

1. Plan (read-only, plan payload), Prepare and Implement (owned writes), each a Pass of the one process.
2. Verify on the captured stand of the last work Pass, with fresh reviewers.
3. After a work Pass, open genuine findings of work Passes that name its purpose or an earlier one repeat it or restart there; a later purpose they name comes in order, so no forward Pass is skipped and no Verify starts while a work finding is open. A Verify that needs correction restarts the work at the earliest purpose its open genuine findings name (`plan`, `prepare` or `implement`; `implement` by default), and a complete fresh Verify follows.
4. When only fresh disputes keep a Verify from being clean, a new complete Verify with a new Pass identity assesses the same unchanged candidate instead of an artificial correction.
5. A clean Verify ends the run Verified; Apply publishes that candidate. The Pass budget bounds every repetition.

Extract composes its own work purposes with Verify the same way.

## The Pass process

<a id="reconcile-cycle"></a>

Every Pass has a new identity `p<N>` and runs the same base process for every Manager, Root and leaves alike (stages in [run](../../../internal/run/README.md#the-base-process)): release on the parent's accepted basis; productive start once the dependencies of the accepted plan for this purpose are accepted (until then only read-only communication turns); own work and an explicit basis; the Orchestrator's technical check of the submitter's stable area after its turn ended (the File check of that area composed with the accepted stand, judged by the findings addressed to the submitter), then the direct parent's semantic acceptance (Root's by the Orchestrator); release of the children; review of their bases and complete results, each of which wakes the parent before the child is Done; integration and the complete result. A returned proposal goes back to its owner within the Pass; a parent may reopen a Done child; a Manager may revise its accepted basis, which explicitly resets its consumers. States are recorded as `manager.state_changed`; problems as `problem.raised`.

A turn that ends without the report its work needs is incomplete work, never completion: it gets a note, and repeated empty turns end the run Incomplete; a progress report that repeats an earlier one of the Pass counts as empty and is not continued automatically. So do repeated failed turns of a required Manager, a turn without an observed container stop, an unresolvable wait and the deadline. A Manager may wait for an own question, a child or the accepted basis or result of any other Manager; only the named acceptance or other actual work wakes it, so an unchanged wait starts no turn. A wait that closes a cycle with the accepted plans' dependencies and the parent-first and integration order is refused with the cycle named; a Pass that still cannot advance ends Incomplete as stalled. Every turn records its mode (`work`, `communication`, `coordination`, `review`, `helper`). The Pass closes when every Manager is Done and nothing runs: the quiescent shared tree is harvested and must equal the accepted stand, and the Model check and the File check run on it. Any failure ends the run Failed without a repair Pass. All participants then receive the sealed verdict in their next Pass.

## Verify, findings and disputes

A Verify materializes the sealed stand of the last work Pass as an immutable read-only candidate, runs the Model check, the File check and every required check on it, and runs fresh reviewers in the same base process. Failed checks are genuine findings of the Orchestrator owned by the check's owner; they hold for their candidate. Reviewer findings name the owner, the obligation, the observed evidence, whether they are genuine or notes and the earliest correction they need. A submission's findings, closures and disputes take effect only when its parent (for Root, the Orchestrator) accepts it; a submission returned, failed or withdrawn before its acceptance changes no finding, and an accepted effect stays even when that submission is invalidated later; a recorded finding cannot be withdrawn. A finding closes only through a later accepted review submission of its raising responsibility, with reason and evidence: `resolved` on a changed candidate, or `disproved` on the unchanged one. Any other reviewer whose responsibility holds the raiser or the finding's owner may dispute a review finding once per candidate with reason and evidence; the dispute keeps it open and blocking until a fresh reviewer of its responsibility disposes of it. A failed required check cannot be disputed. A sealed Verify is never rewritten. A reviewer can be Done with findings; the Run is then not Verified.

<a id="probe-receipts"></a>

Reviewers derive claims and counterexamples from the bound model and run behavioral probes with `check`. A probe runs in its own container with the execution profile of a required check (image, executable, fixed arguments, environment, limits), without network, on a disposable copy of the candidate with the probe file added in the check's directory; it never replaces a project file. Its claim and expected outcome are recorded before it runs. The Orchestrator records the receipt: the candidate, probe bytes and digest, invocation, observed start, exit, timeout and certainty, output and outcome (`passed`, `failed` for the probe's own assertions, or `error` for anything not established, including an unexecutable or killed probe). Every `done` or `preserved` obligation of a review cites evidence items, each supporting one claim: a passing receipt on its candidate, which brings its claim and supports one item, or an inspected source location with reasoning; a receipt of another stand, a fabricated or foreign receipt and an adverse one support nothing. Parents see each child's evidence and every probe it ran. An adverse receipt on the candidate keeps the Verify from being clean until it is disposed of: a genuine finding citing it, a reasoned expectation error, when the probe itself was wrong (a failed assertion, or an error within the probe's own execution, such as an exceeded time limit, with a passing receipt of the same obligation on the candidate), from or accepted by an independent assessor (the parent, the reviewer of the obligation's owner for an ancestor's probe, or for Root's own otherwise a fresh Root reviewer in a new Verify of the unchanged candidate), or a passing exact rerun (`probe.rerun`) for a Runtime error. Such a disposition falls when its submission is reopened or invalidated. A receipt records an observation; it neither proves the model nor the completeness of the chosen cases.

## Questions and escalation

`send` reaches only the direct parent and children; answers travel back the edge their question came, and a forwarded question correlates with the question it serves (`about`), so a cross-branch exchange takes the ancestor route both ways. A question to a Manager its parent's basis has not released yet is refused. Root asks the invoking AI as `user` through the owner channel: the question is written to `<state>/questions/<id>.json` and printed to stderr with its run identity, and `markitect answer ID --run RUN --text ANSWER --run-config PATH` answers it; an answer to another run's or a lapsed question is refused. A result is refused while its Manager's own question is open; open questions lapse when their Pass closes and the user's are withdrawn. `escalate` sends a classified problem as a question to the parent (Root's to the user). Answers clarify within the model; Root ends the run Failed with `report kind problem, category model, abort` when it judges the model erroneous or contradictory.

## Orchestrator tools

<a id="host-tools"></a>

| Tool | Purpose |
| --- | --- |
| `report` | Progress, a named wait, a basis or complete result (shared envelope: coverage with claim-level evidence, purpose payload, findings, closures, disputes, dispositions of adverse receipts, verdict), a decision on a child's proposal, a reopened child result, a revised basis, a problem. |
| `send` | Question, answer or information along one direct edge; Root also to `user`. |
| `escalate` | A classified problem and the decision needed, as a question to the parent. |
| `look_up` | Definitions, owners, the derived registry view, states, accepted plans and bases, findings, checks, probe receipts, questions, and the reusable method guidance (planning, preparing, implementing, integrating, reviewing, escalating), which the standing context names and which grants no right. |
| `check` | Advisory File check of the caller's live area composed with the accepted stand (write Passes), or one behavioral [probe](#probe-receipts) with an Orchestrator receipt when the project declares a check. |
| `delegate` | A bounded task for one of the Manager's helper roles. |

The provider's own read, edit and shell tools stay available; the Claude adapter withholds `Write` and `Edit` from turns that can write nowhere.

## Local teams

The standard team is the Manager role (Execute profile), a Worker helper and the fresh Verify reviewer (Verify profile). The product default team binds one `worker` role with writes to the Manager's Execute profile and allows one active helper; `team` in `execution.yaml` replaces it with explicit roles (`role`, optional `profile`, `writes`, optional `instructions`) and `maxHelpers`. A delegation names an existing file or directory of the Manager's own writable area outside `.markitect`, disjoint from every other active helper's; the helper starts after the delegating turn ended, runs one turn in its own container with its own home, reads the whole stand, writes only its subset and reports to its Manager. While it works, its subset is read-only for the Manager. Read-only roles, such as an architect, advise only. Helpers add no owner, no registry writes and no tree channel, and their completion completes nothing.

## Budget and stopping

The budget bounds the whole run (`executionTime`, with the operator's earlier `deadline`), each turn (`turnTimeout`), all Agent turns of Managers, reviewers and helpers (`agentTurns`) and the number of Passes (`passes`). The deadline cancels running turns and checks. Every limit ends the run Incomplete with its evidence; nothing is published.

## Ports and adapters

| Port (owner) | Adapter | Notes |
| --- | --- | --- |
| `run.Agents` | `claudecode.Agents`, `codexappserver.Agents`, selected per profile by the bootstrap router | One fresh container per turn with `/work` and `/home/agent` volumes, user `1000:1000`; overlays of directories and files of the work volume; a Verify turn is read-only and may continue only its own reviewer conversation of the same review Pass. The container is stopped before `Run` returns and `Stopped` reports whether its end was observed. |
| `run.Workspaces` | `gitrepo.Workspaces` on container volumes | Materialize with read-back, Harvest, Peek, Close. |
| `run.Checks` | `checks.Runner` | One container per check on a disposable copy of the candidate. |
| `run.Owner` | bootstrap file channel | Ask, Answer, Withdraw. |
| `run.Evidence` | bootstrap file evidence | Append-only events, transcripts, report. |
| `apply.Publisher` | `gitrepo` publisher | Preflight against the captured source guard, Publish, read-back. |

## Configuration

`.markitect/execution.yaml` (`apiVersion: markitect.execution/v1`, `strategy: recursive-passes-v1`) is closed: `budget {executionTime, turnTimeout, agentTurns, passes}`, `profiles {execute, verify}`, optional `instructions`, per-Package `packages` overrides, `ownership`, optional `team {maxHelpers, helpers}`, optional `scope` and `checks`. The [apply guide](../../../internal/apply/README.md#execution-settings) shows the shape and the embedded product defaults. The run configuration is the separate closed JSON file of the [project workflow](../../project-workflow.md#invocation-configuration).

## Evidence and state

<a id="evidence-and-state"></a>

The state directory is held by one run at a time (`apply.lock`) and lies outside the repository.

```text
<state>/apply.lock
<state>/current-run                          run identity of the run holding the lock
<state>/runs/<runID>/events.jsonl            append-only, synced per event
<state>/runs/<runID>/transcripts/<turn>.jsonl  provider stream, secrets redacted
<state>/runs/<runID>/report.json             written once at the end
<state>/questions/<id>.json                  open question of Root to the invoking AI
<state>/answers/<id>.txt                     its answer
```

Events: `run.start`, `pass.start`, `manager.release`, `manager.unblocked`, `manager.state_changed`, `turn.start`, `turn.end`, `report.progress`, `report.wait`, `proposal.submitted`, `proposal.checked`, `proposal.withdrawn`, `proposal.invalidated`, `basis.accepted`, `basis.returned`, `basis.invalidated`, `result.accepted`, `result.returned`, `result.reopened`, `finding.recorded`, `finding.closed`, `finding.disputed`, `question.asked`, `question.answered`, `question.lapsed`, `message`, `problem.raised`, `helper.delegated`, `helper.start`, `helper.end`, `check.requested`, `probe.requested`, `probe.receipt`, `probe.disposed`, `candidate`, `check`, `gate`, `pass.sealed`, `registry.view`, `manager.disabled`, `owner.error`, `workspace.materialize`, `workspace.close`, `home.remove`, `outcome`. Events carry the Pass, Manager and turn they belong to; `turn.start` and `turn.end` carry the turn's mode. The report's run lists every probe receipt with its disposition. A failed record write ends the run Incomplete.

## Publication and read-back

Only a Verified outcome is published, as the exact verified candidate against the committed source captured at the start. The publisher refuses a source that changed since capture and a change set the target cannot represent, writes the changes and reads every path back. Multi-file publication is not atomic; a partial write stays visible and unsuccessful. Apply closes the shared work volume only after an applied or no-change receipt; otherwise it keeps the work as retained. The report's `outcome` is `success` only for an applied or no-change read-back after a clean Verify, `failed` for a definite rejection and `incomplete` otherwise. Apply never commits, merges or pushes.

## Extract: first bounded implementation

<a id="extract-first-bounded-implementation"></a>

Extract (Artifacts → Model, [DEC-057](../../concepts/register.md#dec-057-three-fundamental-starting-situations)) keeps its own mandates, accounting, owner questions and model-only publication and runs them on the same Pass process. Its start is a committed repository with artifacts, the imported vocabulary, a manifest and lock without local Definitions and `.markitect/execution.yaml` with budget, profiles, ownership areas for the artifacts and checks without `requiredBy`; Extract refuses a model that already has Definitions and a start without ownership areas. Unregistered artifacts and artifacts no area covers yet do not fail its preparation.

**Phase one, structure:** Root alone runs the work Pass `structure`, writing `.markitect` except the dependency models: it proposes Packages with one overview Definition each, the `ownership` areas that give each Package its artifacts (the Package owning the project as a whole also owns the project-level artifacts), and proposed checks from documented commands. Everything outside its writable inputs stays byte-identical; in `execution.yaml` only `ownership` and added `checks` may change. A fresh Verify assesses the proposal against the artifacts.

**Phase two, model:** the verified proposal is prepared again, and its Manager tree runs the work Pass `model`: each Manager derives the Definitions of its Package from the artifacts and registers its artifacts in its own registry, writing only its model directory and registry minus its descendants'. Its Verify reports an artifact that deviates from a justified statement as a non-blocking finding with subject `artifact` and its Definition (`ArtifactFindingsAdvisory`), and blocks model defects.

Both phases share the owner channel and the deadline; the structure phase may use at most half of the agent turns (at least 8 in total). A Verified second phase publishes only `.markitect/model`, `.markitect/registry` and `.markitect/execution.yaml`; an artifact is never published or changed, and Extract never commits. The report lists model completion, the last Verify Pass with each Manager's verdict, the deviations per Definition, the proposed checks, open questions, both phases' Passes, turns and usage, and the publication. Restructuring between rounds, handing findings to Apply and running the project's checks in Extract are not implemented.

## Trusted and enforced

| Enforced by the code | Trusted |
| --- | --- |
| Strict Model check and File check before work, on every submission's area and at every Pass boundary; unchanged authority inputs. | The Managers' and reviewers' semantic judgment of obligations, fit and findings. |
| Ownership by most specific area, with directory and exact-file overlays per Agent container; unsupported shapes refused. | That a parent's acceptance reflects a real review of its child's proposal. |
| Parent-first release, explicit reports, parent acceptance, no completion by a turn ending, stale and duplicate reports refused. | That plans name the real dependencies; only declared ones are enforced. |
| Tree-only routing with correlation, user answers bound to their run, lapsed questions withdrawn. | That answers stay within the model's rules. |
| Fresh read-only reviewers on one immutable candidate, findings retained until independent closure, required checks on that candidate. | That the chosen probes and claims are the relevant counterexamples; a receipt shows what ran, not that the cases are complete. |
| Probe receipts made only by the Orchestrator, bound to their stand and bytes; claim-level evidence for every reviewed obligation; adverse receipts kept and blocking until an independent disposition. | That a source-inspection reasoning or an expectation-error disproof is sound; the parent or a fresh reviewer judges it. |
| Waits wake only on named events; unsatisfiable waits are refused, stalled Passes and repeated progress end the run within limits. | That plans declare the real dependencies; undeclared needs surface as waits. |
| Nothing publishes unless Verified, with evidence recorded. | |

## Known limits

- Provider evidence covers few small projects and runs per build; reliability, cost, turn budgets and the usefulness of helpers are not established. The example budgets are unvalidated starting values.
- Probes need a declared check whose profile can run them; without one, reviews cite inspected sources only. A probe runs synchronously inside its tool call and only while its turn keeps time for the report; a parent can probe a child's checked proposal but not its unsubmitted work. Joint submission of a basis and a complete result is not implemented (deferred; see the hardening report).
- An exact-file area owned apart from its directory's owner must exist in the start stand; its owner can change it but not delete or create it.
- Dependencies are enforced only as declared in accepted plans; undeclared needs travel as questions.
- No restore of an interrupted run; a new Apply starts again from the committed repository.
- Every Pass involves every Manager; unaffected owners preserve explicitly rather than being skipped.
