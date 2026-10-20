# Terminology and file accounting discussion — 20 October 2026

**Current disposition:** the owner adopted the complete discussed proposals and later corrections on 20 October 2026, and requested preparation for a Claude implementation session. The [accepted workflow](../../design/canonical-model/manager-apply-workflow.md) is now the behavior owner, [DEC-064](../register.md#dec-064-checked-recursive-apply-passes-and-manager-teams) records adoption, and the [handoff](../../development/claude-apply-implementation-handoff-20261020.md) prepares implementation. The chronological statuses below describe the conversation at the time; they are not remaining approval gates. No runtime change or new provider evidence is claimed by this document.

Recorder's note: local discussion record requested by the owner in this chat. Inspected integrated Main: `0c28097c030bc5d031b501002bc85634cd66a4c8`. The owner subsequently chose strict, blocking initial file accounting, superseding DEC-062's repairable-start exception, and accepted the vocabulary below, including its use in code. The inspected implementation still has the exception and the old identifiers. This worktree records accepted owner directions and source observations; it changes no runtime behavior and starts no implementation, issue, PR or publication.

## Owner instruction: consistent names

> Wir sollten es einheitlich check oder compile nennen. Der Code darf hier nicht abweichen. Halte das lokal in einem Worktree fest. Dasselbe Problem vermute ich auch an anderen Stellen wie host vs orchestrator und apply vs reconcile. Wir wollen hier einheitlich in der Benamung sein. Prüfe die entsprechenden Stellen und lege irgendwo was an, damit wir das nicht vergessen.

Recorder's interpretation: use one name consistently across conversation, documentation, CLI, code and reports for the same operation or responsibility. Do not treat genuinely different responsibilities as synonyms. The later accepted vocabulary below resolves the initial naming question; the source observations retain the inspected names for traceability.

## Source observations: naming

| Terms | Observed source and boundary | Question retained |
| --- | --- | --- |
| `check` / `compile` | [CLI command](../../../internal/adapters/cli/command.go) exposes `check` for model, registry and file accounting. [model.Compile](../../../internal/model/compile.go) validates only the supplied model sources, dependencies and lock. [PreparationService](../../../internal/apply/preparation.go) composes the broader operation. | Select the name of the complete deterministic check and give its actual substeps explicit, consistent names; stop using the whole operation and one substep interchangeably. |
| Additional uses of `check` | [Execute tool check](../../../internal/run/checktool.go) only peeks and accounts the live shared tree, as advice. [Required checks](../../../internal/run/checks.go) execute configured project commands on a candidate. | Distinguish these different operations from initial model/repository checking. They do not all compile the model. |
| Host / Orchestrator | [DEC-042](../register.md#dec-042-orchestrator-names-the-deterministic-coordination-component) already names the deterministic coordinator Orchestrator, while explicitly preserving then-current API/config/CLI names and literal history. [Architecture](../../architecture.md) uses Orchestrator; current [runtime contract](../../design/canonical-model/manager-apply-runtime.md) and [profile types](../../../internal/apply/host_profiles.go) still use Host. | Apply the owner's new consistency requirement to code as well. Decide which remaining Host uses mean the coordinator and which mean the execution machine, profile catalog or infrastructure. Keep historical quotations unchanged. |
| Apply / Reconcile | [apply.Reconcile](../../../internal/apply/reconcile.go) prepares the engine config and publishes its verified result. [run.Reconcile](../../../internal/run/reconcile.go) runs Execute/capture/Verify without publishing and is also used by Extract. [Run guide](../../../internal/run/README.md) documents that boundary. | Choose consistent names for the complete Apply and its shared engine; retain the actual publication boundary and avoid naming Extract's shared engine as an artifact-publishing Apply. |

The findings establish naming overlap and deliberate scope differences. They do not establish that every occurrence is an erroneous synonym. A later rename should follow the selected meaning across code, configuration, CLI, reports, tests and current docs together; no rename implementation or delivery queue is started here.

## Owner question: complete file accounting

> Als nächstes war mal festgelegt, dass ein valides Modell nur dann valide ist, wenn jede Datei entweder in .markitectignore steht oder in der zusammengesetzten Registry fest zugeordnet ist. Ist das korrekt? Wie heißt der Schritt in Apply? Wird er in Apply überhaupt ausgeführt? Sollte es zu compile gehören oder ein eigener Schritt sein? Wenn das failed sollte dasselbe passieren, wie wenn ein compile fehlschlägt

## Source observations: accounting and failure

- The implemented exact-file policy is `.markitect/ignore.yaml`, not `.markitectignore` or `.gitignore`. [Decoder](../../../internal/apply/file_check_ignore.go) requires observed exact paths and reasons; it does not accept general ignore patterns.
- [AccountFiles](../../../internal/realization/accounting.go) checks a captured tree against the composed registry, the compiled model, explicit input/ignore classifications and the persistent layout. Canonical model and Markitect control inputs are classified separately; they must not be registered or ignored as artifacts. Technical Git administration is outside the captured project tree.
- File coverage alone is insufficient: registration owners and Definition links must resolve; paths must exist; protected inputs cannot also be artifacts; conflicting/duplicate accounting and layout ownership violations block.
- [Compile](../../../internal/model/compile.go) checks model-language validity without a repository registry. A passed model is a prerequisite for [file checking](../../../internal/apply/file_check_run.go); it is not a claim that every repository file is accounted for.
- Apply runs this accounting during [preparation](../../../internal/apply/preparation_capture.go) and again on every captured candidate through [Candidates.Account](../../../internal/apply/candidates.go). A failed required preparation returns before Manager work in [bootstrap](../../../internal/bootstrap/execution.go). Candidate accounting findings block a green Verify and publication.
- Existing exception, [DEC-062](../register.md#dec-062-start-stand-drift-is-the-owners-first-work): under a declared layout, repairable drift of committed files (`file.unexplained`, `registry.missing_file`, `registry.dangling_basis`, `registry.outside_area`) is retained as owner-addressed initial work rather than refusing Apply. Strict `markitect check` still fails. Invalid/unreadable control inputs, invalid layout and `layout.no_owner` still refuse preparation. A Git-ignored untracked file is not repairable committed drift.
- Observed reason for that exception: [RA-R1 and the owner's response](start-stand-drift-20261019.md) explicitly chose repair over refusal after an unregistered committed file prevented Apply from starting. [RA-R3](../../validation/apply-evidence-haiku-20261019.md) and [RA-R6](../../validation/apply-stability-20261020.md) report successful drift repairs. Those runtime reports are bounded Library evidence, not runs performed in this discussion.

## Owner decision: strict initial accounting and unambiguous ownership

> Es soll nichts repariert werden in diesen Fällen. Das sind klare Hinweise darauf, dass das Modell nicht ganz sauber ist und könnte unseren Prozess gefährden. Es sollte definitiv ein eigener blockierender deterministischer Schritt sein. Keine Ausnahme. Neue Dateien dürfen während des Applies hinzukommen und müssen bis zum Ende des Applys entsprechend zugeordnet sein. Jeder Manager muss immer wissen, welche Dateien ihm gehören und welche nicht. Hier darf es keine Uneinigkeit geben. Das müssen wir so festhalten.

Recorder's account of the accepted requirement:

1. File accounting is its own deterministic, blocking step after model-language validation and before Execute. Every existing captured project file must have exactly one valid classification: canonical/control input, registered artifact with valid owner and Definition basis, or exact explicitly ignored file. Ownership of artifacts must be unambiguous to every Manager.
2. An initial accounting error stops Apply with structured diagnostics, as a failed model check does. No Manager repairs an invalid initial registry or accounting during that Apply, and nothing is published. There is no repairable-start exception; this explicitly replaces the direction of DEC-062 in this conversation.
3. New artifacts may be created during Execute. Their owner must already be unambiguous when created, through the declared responsibility and write boundaries. This is distinct from the registry entry: new artifacts must have their complete, valid registrations by completion. Existing candidate accounting continues to block successful Verify and publication while any captured file remains unexplained, conflicted or assigned to the wrong owner.
4. Each Manager must know its own areas and the areas it does not own throughout Apply. Initial file registration and persistent ownership must agree; later communication cannot create competing ownership or authorize writes to another Manager's area.
5. This requirement addresses structural/accounting faults in the starting stand. Semantic repair of an owned, registered realization against the accepted model remains part of Execute.

Status: accepted owner direction recorded locally; not implemented or integrated. The concept register and source on the frozen Main still describe DEC-062; this local record makes the supersession explicit without rewriting historical evidence. No claim of newly observed runtime safety or reliability follows from this decision.

## Owner request: distinguish the process and its steps

> Wir scheinen hier ausserdem viele Namen zu vermischen, die wir festziehen sollten.
> Der Prozess den wir hier beschreiben heißt Apply. Hierbei könnte man argumentieren, dass Apply der Command ist und viele Teilschritte beinhaltet. Gleichzeitig höre ich auch reconcile. Wir müssen hier eindeutig sein und unterscheiden können was wann ist. Dasselbe bei den anderen genannten Beispielen. Lass uns das einmal kurz geradeziehen, da das sonst zu Verwirrung führt beim Verstehen des Prozesses.

## Owner acceptance of the vocabulary

> Die Benennung passt so, halte das entsprechend fest. Der Code muss das reflektieren und dann entsprechend "Compile" und "Reconcile" entfernen. Diese Dinge sind zu nah an den anderen Begriffen und stiften für Verwirrung.

The following vocabulary is accepted. It applies to conversation, current documentation, CLI, reports and code identifiers. `Compile` and `Reconcile` must be removed as competing current operation/API names in a later implementation. This is a binding naming requirement recorded locally, not a rename implemented in this discussion. Historical quotations and fixed-source observations retain their original words.

| Term | One meaning |
| --- | --- |
| Apply | The complete process and command, from initial checks through Execute, Verify and guarded publication. |
| Model check | Deterministic model-language validation and resolution. Use this operation name consistently, including in code (for example `CheckModel`); remove the competing `Compile` operation name. |
| File check | The separate deterministic accounting and ownership gate, before Execute and again on captured candidates. Accounting describes its responsibility rather than an additional process. |
| Execute | Managers realize and integrate artifacts in their areas. |
| Verify | Fresh independent semantic assessment of the immutable candidate, including child integration. |
| Required checks | The explicitly declared project checks, such as build and tests, on the candidate. |
| Publish | Guarded writing of the verified candidate to the target repository with read-back. |
| Orchestrator | The deterministic component that controls participation, turns, messages, budgets and phase transitions. |
| Runtime | The execution infrastructure for containers, provider processes and volumes. Host refers only to the machine/environment where this infrastructure runs. |

`check` is the standalone command that runs Model check and File check. Apply is the whole process and command; `Reconcile` must not remain a competing operation/API name or an additional phase. The shared engine currently called `run.Reconcile` executes the Execute/Verify cycle for Apply and Extract; its replacement name must preserve that shared responsibility and the fact that it does not publish. Renaming the complete Apply entry and the shared cycle does not make them the same operation. Exact replacement signatures and the implementation remain unassigned.

Model validity, complete file accounting and semantic realization remain separate results. Registry coverage does not prove that a file fulfills its linked Definition; Verify supplies the semantic assessment.

## Owner clarification: local registries, repository layout and foundation readiness

> 1. Wir hatten mal festgelegt, dass es keine globale registry geben soll sonder lokale pro package, welche dann bei Bedarf hochkompiliert werden können. So vermeiden wir Schreibprobleme. Ausserdem gelten dort immer relative Pfade vom jeweiligen Package aus. Ausserdem sollten direkte Definitions in einem Bereich explizit gekennzeichnet sein als "dein Bereich" oder ähnlich und andere nur als Kontext oder mit zugehörigen Manager auftauchen.
> 2. Ein klassisches Repo hat nicht diese Struktur die du hier beschreibst. Stattdessen hätte es src, docs, tests, etc. Je nachdem wie es organisiert werden soll. Manche arbeiten auch ganz anders und das hängt oft auch von Konventionen der Technologie ab. Das dürfen wir nicht einfach so festschreiben ohne Ausweichmöglichkeit.
> 3. Der Kinderstart darf erst loslegen, sobald der Root Manager sein Zeug vorbereitet hat. Das umfasst auch, dass er seine registry aktualisiert hat mit allen Dateien.
> 4. Was Arbeitsbereiche angeht, könnte man für jeden Ordner Pfade zu Ordnern oder gar Dateien geben. Wer zuständig ist, leitet sich aus Spezifizität ab. Der Speziellste gewinnt und darf schreiben.

Recorder's account of points 1 to 3 as owner requirements:

- Each Package's Manager writes only its local registry. Root may have its own local registry for Root-owned artifacts; that is not a project-wide registry for other Managers to edit. A combined registry is a deterministic derived view of the local registries, not another independently maintained writable authority.
- The owner initially requested Package-relative artifact paths; the subsequent annotations below explicitly accept repository-relative paths. That resolves the base question without requiring a physical Package folder or changing local registry ownership.
- Briefs explicitly separate own Definitions (the Manager's obligations) from contextual Definitions, identifying the responsible Manager for the latter. Root coordinates child responsibilities and cannot take over or waive them.
- Model Package hierarchy and physical repository layout are separate. A mirrored layout may be a convenient default, but `src`, `docs`, `tests`, technology-specific layouts and other accepted structures must be supported. `realization/...` is the Library fixture's configured layout, not a mandatory Markitect project structure.
- Children may start only after Root has prepared the foundation they need and updated its local registry for every file it created. A successful provider turn ending is insufficient. The readiness transition must include valid file accounting for the prepared basis; it is not final completion of Root's later child-integration responsibility. The owner subsequently accepted the semantic transition and checked-completion mechanism below; exact state/event identifiers remain unspecified.

Source comparison on the fixed Main:

- Per-owner registry shards already exist. [Registry paths](../../../internal/apply/registry_encode.go) name Root's own shard and each Package's shard. [Registry decoding](../../../internal/apply/registry_record.go) currently interprets artifact paths against the repository census, and [the Brief's example](../../../internal/apply/brief.go) explicitly uses paths from the repository root. Package-relative paths are therefore not the inspected format.
- Own Definitions versus included context are already labeled in the Brief, including the owning Package for included Definitions. This is an existing property to preserve.
- The Library fixture declares `realization` as its artifact root; the current layout supports multiple roots and explicit exceptions. It nevertheless derives mirrored Package areas by default, so arbitrary physical organization needs explicit ownership mappings rather than an assumption that Package paths and folders coincide.
- The current [child release](../../../internal/run/execute.go) follows the first successful parent turn, or a disabled parent, without checking the foundation's registry as a separate readiness condition. It does not implement the clarified gate.

## Candidate: explicit ownership paths with specificity

Point 4 is retained as an owner-proposed option, not silently adopted as a final mapping format:

- Declare directory subtrees and exact files with their owning Package, independently of model hierarchy. A Package may own several physical areas, such as source, tests and documentation.
- The most specific matching path owns the file and determines its sole writer. A deeper directory overrides an enclosing directory; an exact file overrides its enclosing directory. The File check must reject equal-specificity assignments to different owners and uncovered artifacts, rather than choosing an arbitrary winner.
- The resolved ownership used by the File check, Manager Briefs, local registry validation and runtime write protection must agree. Declaring an owner is not sufficient if the runtime cannot enforce that owner's write boundary.
- Current [layout code](../../../internal/realization/layout.go) already chooses the deepest mapping, but file exceptions can only be owned by Root; normal enforced writable areas are directories. General exact-file ownership/write support is therefore not established by the existing implementation.

Benefit and tradeoff: explicit paths allow conventional or technology-specific repositories while preserving one writer per file. They require maintained mappings and an unambiguous rule for new files. The current mirrored default avoids some authoring work; it should not remove the explicit-layout escape route. A smallest later check would resolve one Package's source, tests and docs across separate folders and one exact-file override, confirming the same owner and write rights everywhere. No probe or implementation is started here.

The previously open relative-path base question is resolved by the owner's acceptance of repository-relative paths below. Local registry ownership remains independent of the path base.

## Owner annotations: path selectors, derived registry and semantic completion

### Accepted path clarification

> Repository relativ ist auch okay. Man sollte aber auch wie in .gitingnore ganze Ordner angeben können statt nur files.

> Wie gesagt, repositoryrelativ geht auch für mich.

Recorder's account: repository-relative artifact paths are accepted; a separate Package-relative base is no longer required. Directory entries must be possible rather than requiring one authored entry per file. The comparison to Gitignore establishes directory support, not automatic adoption of its entire pattern/negation language or its treatment of untracked files. Folder ownership, artifact registration and explicit ignoring remain different meanings and must not become interchangeable.

Open directory-entry semantics: the File check must still inspect actual files and resolve one owner for each. A directory selector cannot count another Manager's more-specific area as the declaring Manager's own artifacts. The artifact form, Definition basis and reason must remain valid for each covered file; naming a folder does not establish semantic conformance or silently justify every future artifact in it. A deterministic derived exact-file view can expose what each selector currently covers. Pattern syntax, recursion and exceptions remain unspecified; no schema is implemented here.

### Accepted mechanism: checked completion and a derived registry view

> Wir könnten hier auch etwas einführen. Beispielsweise jedes mal, wenn ein Manager abschließt und diese Dateien geändert hat, wird ein Teilcheck oder ein Check Files für seine lokale Datei ausgeführt um sicherzustellen, dass es passt. Danach wird in /registry oder einem ähnlichen Ordner das globale registry neu kompiliert. Oder man kompiliert es neu und checked dann. Idee wäre es dass wir das Tooling nutzen um sowas sicherzustellen und dass jeder immer schauen kann, welche Dinge wem gehören. Dabei müssten wir nur aufpassen, dass sich die Manager nicht in die Quere kommen, gerade wenn sie parallel arbeiten.

Initial status: owner proposal. Subsequently accepted in this conversation, together with the status/event distinction below:

> Das passt so, halte die Mechanik entsprechend fest.

Status: accepted owner mechanism, recorded locally; not implemented or integrated. Use the accepted name File check, and describe the derived registry as being assembled or refreshed rather than adding another Compile operation.

Accepted sequence:

1. A Manager explicitly reports a semantic result, such as a prepared foundation or completed own work. An ordinary provider turn ending is not that report.
2. The Orchestrator runs a scoped File check over that Manager's local registry and actual owned files, against the fixed model and ownership mapping. Checking only the YAML file would miss unregistered artifacts, wrong ownership and dangling entries.
3. Once the scoped result passes, the Orchestrator refreshes a read-only project registry view from coherent, identified local registry versions. Local registries remain the only authored sources. A materialized project view, if useful, is derived output owned by the Orchestrator, not a shared Manager-written registry. No storage directory is selected here.
4. Only then is the relevant semantic result accepted and any dependent work released. Scope and source version of a local pass stay visible; it is not a whole-repository pass.

Parallelism boundary: each Manager remains the sole writer of its local registry and areas. The Orchestrator alone produces the combined view; it must not read half-written local files or label a mixed, unvalidated set as checked. Compose from coherent local versions, or detect a changed input and withhold/retry the refresh. This may briefly serialize producing the view, not all Manager implementation work. Subsequent changes invalidate the corresponding checked view/result until reassessed. A full File check on the immutable candidate still precedes successful Verify/publication; the local check cannot replace it.

Open choice: local validation can precede assembly, with cross-Package collision checks on assembly, or assembly and a scoped/global check can be one operation. The observable requirement is that no accepted result or displayed checked registry hides an ownership conflict, invalid local registration or changed input. No second authoritative registry or general transaction framework is selected.

### Accepted mechanism: status transitions and events

> Das hier klingt für mich nach einem Event oder einem Status. Wir sollten hier eventuell generell mit Events oder Status arbeiten statt jeden Turn als Abschluss zu werten.

Status: accepted by the owner with the mechanism above; not implemented or integrated. A semantic state states what is currently true, for example that the parent's foundation is ready; an event records that the Orchestrator accepted that transition. Provider turns remain technical execution intervals and may end for waiting, interruption or continuation without completing a responsibility.

The accepted parent-readiness mechanism is an explicit Manager readiness report, deterministic File check, refreshed checked registry view, and an accepted state transition/event that releases the children. The report alone establishes no checked readiness, and readiness of the foundation is distinct from final completion after child integration. Exact states, event names, storage paths and tool contracts remain open. This does not adopt an event-sourced architecture, a new role or another review stage.

Mechanism assessment: explicit checked readiness addresses the observed code behavior that a successful first turn (or even a disabled parent) releases children without validating the parent's foundation registry. It could replace inference of completion from turn endings while preserving sole ownership, child participation and later independent Verify. The benefit is a meaningful release condition and traceable checked registry view; the cost is a small explicit transition and handling a failed check or changed basis. A smallest later observation would show one parent whose invalid local registry prevents child release and whose corrected, prepared foundation releases children, while another branch continues independently. This record authorizes no execution of that observation.

## Owner clarification: access, routing and the initiating AI

> Das klingt alles korrekt und sauber. Gerade das mit ein Container pro Agent und mounting halte ich für eine elegante Lösung. Generell sollte jeder Agent natürlich alle Dateien des Arbeitsstands lesen dürfen. Schreiben nur in seinem Bereich. Kommunikation darf ausschließlich über Eltern- und Kindmanager erfolgen. Rückfragen an den User sollten stattdessen immer an den Vatermanager gehen, welche das entweder beantwortet oder weiter eskaliert. Root Manager eskaliert an den User, bzw die AI des Users mit dem markitect ausgeführt wurde, welcher es dann mit dem User abstimmt und wieder runter gibt.

Recorder's account of the clarified direction:

- Every Manager's Agent may read every project file in the shared working stand. Writes are restricted to its explicitly owned areas and local registry, enforced through the Runtime. A focused initial Brief does not restrict the Agent's read rights or mean every file must be inserted into its prompt.
- Agent communication goes only to direct parent/child Managers. An answer outside the local authority goes to the parent; that parent resolves it within the model, coordinates a child, or escalates further. Knowing another file's owner does not grant a sibling communication route or write permission.
- Root is the sole boundary to the owner or the user's initiating AI. That AI is an external caller/owner-facing channel, not Root and not the deterministic Orchestrator. It can discuss an escalation with the user and pass the resulting answer down through Root and the Manager tree. This discussion records no concrete transport or authorization to message another chat.
- Source difference: the frozen Main's DEC-063 refuses all mid-run owner questions in Apply and ends on a model-defect report. The owner's desired answer-return path is a new direction relative to that implementation; the later explicit question policy below resolves this discussion. Answers never override the model.

## Owner ideas: recursive passes and deterministic tools

> - Aufteilen von execute in plan, prepare, implement. Damit könnte man kollisionen vermeiden. Jeder Step wäre hiermit ja dann wieder ein Durchlauf von Root runter und wieder hoch und allem was dazwischen hoch und runter geht. Das sollte generell der Kern von Markitects Apply sein bzw von seiner Manager Struktur für jede Art von Durchlauf. Dabei muss jeder rekursive Durchlauf sich auf ein Ergebnissignal einigen, sodass jeder Manager im Baum dasselbe Ergebnis meldet und damit der Durchlauf beendet ist. Dann startet entsprechend der nächste. Und jeder Durchlauf muss einen model und file check überstehen. Ansonsten kommt ein zwischendurchlauf welcher das wieder heruntergibt, bis der Endzustand des Durchlaufs "grün" ist. Dabei wäre beispielsweise plan ein Durchlauf. Apply wäre eine Gruppe oder ein Workflow von Durchläufen.
> - Wir könnten uns überlegen welche Dinge Agenten jeweils generell machen oder wissen bzw erfragen wollen und dafür tooling bereitstellen. Vieles sollte durch Compiler, Modell und unsere technischen files leicht zu beantworten sein.

Status: ideas for assessment, not adopted phases or a generic workflow-engine implementation. The agreed vocabulary, strict initial checks, readiness gates, ownership, independent Verify, guarded Publish and finite bounds remain applicable.

### Candidate: a common recursive pass protocol

Recorder's interpretation for discussion: each pass has one fixed purpose and result contract, travels through the derived Manager tree and integrates results upward. A parent can accept completion for its subtree only after its own required result, all required children and their integration are accounted for. Root's accepted aggregate result closes the whole pass. Turns remain execution intervals; messages and dependencies can require several turns within one pass.

One common result schema/target is distinct from identical content or forced agreement. Managers retain their own findings and scope. A child reporting blocked cannot be hidden by a parent's local ready report. Passing Model check and File check establishes structural validity, not that an Agent's plan is semantically sound or that independent Verify is unnecessary.

Possible separation of Execute:

| Candidate pass | Intended result |
| --- | --- |
| Plan | Each area identifies its realization approach, needed owner-provided interfaces, dependencies and unresolved questions within the fixed model. Root and parents integrate cross-area choices. |
| Prepare | Actual owners provide and register the concrete foundations/interfaces on which dependent work can start. A parent's preparation never authorizes implementing a child's Definitions in the parent's area. |
| Implement | Managers realize their areas using those foundations, maintain local registries and integrate children. |

Problem and property: cross-area duplication, unclear foundations and expensive late repairs are real observations in the Library reports. Earlier agreement on interfaces may reduce them. A separate Plan/Prepare could replace repeated ad hoc coordination, but adds whole-tree work and barriers even when an existing foundation is already suitable. The frozen Main's [DEC-060](../register.md#dec-060-owners-realize-others-use-parents-settle-cross-area-matters) already names prior planning as a fallback, not an adopted default; the later [stability report](../../validation/apply-stability-20261020.md) also records improvement from clearer ownership prompts without such passes. Neither proves the extra passes necessary or useless.

Preservation and bounds: a failed initial Model check or File check still stops before Agent work, with no repair pass. Accounting defects introduced during authorized Execute may be returned to their owners within the same bounded pass, after which the exact corrected result is checked again. A model contradiction cannot be repaired by Agents through another pass. Never repeat until green without budget, deadline and explicit Incomplete/Failed outcomes. With unchanged protected model sources, whether to rerun Model check or reuse an exactly bound successful result is an open implementation choice, not permission to skip its validity requirement.

Smallest later observation: on the same cross-area dependency, compare the prepared result and late-repair work with and without an explicit planning/preparation boundary, retaining wall time, messages, ownership violations and findings. This is a possible future observation only; no run or measurement series is authorized here. Exact pass names, result fields, barriers, plan storage and general reuse across Commands remain open.

### Candidate: query known facts through tools

Prefer deterministic answers for facts already owned by the model, checked path mappings, local registries or Orchestrator state. Candidate questions include: who owns this exact path; which paths may I write; which actual files does a directory entry cover; which Definitions are my obligations versus another Manager's context; where is an obligation stated; which required checks apply; what registered foundation has my parent made ready; what is the current checked registry view and its source stand; and does my changed area pass File check?

Each answer should state its source and stand. Querying ownership grants no new authority. A semantic question such as which interface best satisfies the model remains the Managers' job, routed through the tree; a tool cannot invent intent or infer an authoritative new rule.

Assessment: such queries can replace repeated relay messages about known facts and manual registry inspection, while preserving one authoritative source for each fact. They cost a small maintained query surface and must avoid stale answers or a competing knowledge store. Start, if later assigned, with the unanswered recurring question revealed by real work rather than a general tool catalog. One distinguishing observation would be a Manager answering an existing ownership/registration question correctly through a source-bound query without a parent relay. This record starts no tooling implementation.

## Owner decision: questions are allowed; the model remains law

> Generell sollten Rückfragen erlaubt sein und an den Vater Manager gehen. Generell gilt auch das Modell als Gesetz. Wenn der Root Manager der Meinung ist, das Modell ist fehlerhaft oder widersprüchlich sollte er den Apply Run abbrechen. Falls es sich lediglich um Unterspezifikation handelt oder um Fragen zur Umsetzung soll wie besprochen eskaliert werden an Vater bis User.

Recorder's account of the accepted policy:

- Questions are allowed during Apply and are always addressed to the direct parent. A parent answers within the bound model and its authority or escalates. Root may ask the owner through the user's initiating AI; an answer returns through Root and the tree, and the dependent work can continue.
- If Root judges the model erroneous or contradictory, Root ends Apply without publication and reports the problem. Managers do not repair or override the model during that run.
- Mere underspecification and implementation questions may be clarified through escalation, including questions that reach the user. A missing detail is no longer by itself a mandatory abort merely because it affects boundary behavior. This replaces that part of DEC-063 as well as the blanket ban on mid-run questions in this local design direction. An answer must still be compatible with every existing model rule.
- The question, answer, authority, affected responsibility and actual source stand must remain visible. A question does not silently waive an obligation or transfer ownership. A clarification that requires changing an existing model rule cannot be applied as a run-only override; it requires owner model work and a new bound starting stand.
- The transport through the initiating AI, waiting/cancellation behavior and exact question/result identifiers remain to be designed. No existing run has been resumed or changed by recording this direction.

## Owner priority and request for a complete workflow view

> Halte das auch so fest und lass uns nochmal durchgehen wie das Apply arbeitet. Angefangen bei markitect apply. Was umfasst apply, wie sieht der Workflow aus und wie sehen die Durchläufe aus?
> Auch Zustände bzw TurnResults oder ähnlich halte ich für etwas, dass wir möglichst einfach und mächtig einführen sollten. Und Tools wären auch eine gute Sache, die sicher helfen könnte. Generell ist uns Aufwand und Zeit egal, da wir Haiku und Luna verwenden und unser Ziel darin besteht, das Modell möglichst sicher und vollständig und vertrauenswürdig umzusetzen, sodass man darauf vertrauen kann, dass nichts ausgelassen wurde. Zumindest mehr darauf vertrauen kann als wenn man nur einen Prompt in einen Agent reinhaut.
> Generell sag mir auch gerne immer wie du das siehst und einschätzt

Recorder's account: prioritize faithful, complete, evidence-supported realization and less omitted intent over minimizing turns, elapsed time or implementation effort. Haiku and Luna are the owner's intended execution models, not a new runtime grant or evidence of their performance. Keep finite outcomes and observability so a blocked run cannot claim success or loop invisibly. Continue giving source-bound independent assessments rather than silently adopting every idea.

## Workflow explanation candidate, not implemented

Apply is the complete Command/workflow: bind the source and explicit invocation, pass the strict Model check and File check, prepare the complete Manager team/context/rights, Execute, stop all writers and capture one immutable candidate, run full File check and Required checks plus fresh recursive Verify, repair through owners within bounds or end honestly, then guarded Publish/read-back and a report. It never commits, merges or pushes.

The candidate decomposition of Execute is Plan -> Prepare -> Implement. In Plan the Managers integrate one consistent realization approach, dependencies and owner-provided contracts within the model. In Prepare actual owners establish/register needed foundations and explicit readiness releases dependent work. In Implement they realize and integrate artifacts. Existing conforming areas may assess and preserve their stand in every pass. The complete required team participates; missing participation cannot be disguised as a no-change result. This is a proposed target view, not a claim that the frozen product implements these three passes.

Each pass needs its purpose, explicit input stand/model binding, expected result and allowed writes. Context and authority travel down the fixed tree; local results, findings and integration travel up; parent/child messages may occur between these boundaries. A Manager's own result and the completion of its entire subtree remain distinct. The Orchestrator accepts Root's whole-pass completion only after all required responsibilities and integration, pass checks and unresolved blockers are accounted for. No forced identical prose or agreement hides a finding.

Model validity/protected-input preservation and File check remain gates for every accepted pass result. Initial failures stop Apply. Defects introduced by authorized work may return to their owners for a bounded correction on a newly checked stand. A model error identified by Root ends Apply instead of starting an Agent model-repair pass. Model and file validity alone do not prove a plan or implementation semantically complete; independent Verify and concrete obligation evidence remain essential.

### Small result model for discussion

Candidate semantic report kinds, with names and wire format still open:

| Kind | Meaning |
| --- | --- |
| Continue | Progress is recorded; the current assigned result is not ready yet. |
| Waiting | A named dependency or a question prevents the next action; the report identifies what is needed and its parent/child route. |
| Ready | The Manager proposes that its currently assigned result is ready and supplies scope, outcome and evidence. Orchestrator checks and child/integration completion determine acceptance. |
| Model problem | A suspected model error goes to the parent; Root's confirmed judgment ends Apply. |

A provider error, timeout or interrupted container is a technical observation, not an Agent's successful semantic result. A turn ending without a valid report must not silently become Ready. Readiness does not bypass File check or become independent Verify. Findings and questions carry their content and references instead of multiplying status labels for every cause. The Orchestrator records accepted transitions/events and derives aggregate completion; the Agent cannot declare the entire tree finished on its own.

Proposed tools can expose checked model context and obligations, ownership/write boundaries, local and derived registry views, check declarations/results, routed questions/messages, File check and semantic result reporting. Answers must refer to actual source/stand and preserve routing and authority. Semantic adequacy remains Agent work, not an invented deterministic fact. No specific tool API, state machine library, generic workflow framework or implementation queue is selected here.

Recorder's assessment: a common recursive result/acceptance protocol is a direct response to the observed turn-end-as-readiness gap. The three Execute passes are plausible for making dependencies and foundations inspectable, especially under the owner's reliability-first priority; their benefit must come from concrete outputs and integration evidence, not the number of passes. Repeating generic prompts through the tree would add activity without establishing completeness. A plan can identify how each obligation will be realized and independently assessed; File check covers files, while Verify must still detect missing or incorrectly realized obligations. These complementary responsibilities are what could support more trust than an unconstrained one-prompt workflow. They establish no general reliability result before real evidence.

## Owner request: elaborate states, Passes and Agent methods

> Wie wäre es mit Working, Waiting, Ready, Problem und einem Event bei einem Change davon. Der Orchestrator reagiert auf das Event irgendwie. Probleme müssen die Agenten selbst klassifizieren und entscheiden ob sie eskalieren oder es selsbt beantworten. ModelProblem wäre also nur ein Problem wie jedes andere.
> Alles was Ready ist wird dann immer gestartet. Eventuell fehlt noch Done oder sowas. Oder Success und Fail.

> Wie nennen wir einen solchen Durchlauf auf Englisch am besten?

> Das müssten wir entsprechend ausdefinieren, wie diese Durchläufe funktionieren

> Wie wäre es eher mit der Richtung:
> - look-up
> - check
> - escalate
>
> Oder irgendwie sowas. Wir müssen uns fragen, was ein Agent generell machen will und soll und welche Tools er hierfür brauchen würde. Fast schon ähnlich wie Methoden. Skills wären hier auch eine Idee.

> Wir müssen das entsprechend ausarbeiten

Status: authorization to elaborate the local design, not to implement it. The [discussion draft](../../design/canonical-model/manager-apply-workflow.md) now owns the proposed Pass contract, lifecycle/result distinction, Plan/Prepare/Implement mechanics and tool/Skill roles. Its proposals are not silently accepted by being written down. The earlier Continue/Ready/ModelProblem table above is a superseded assistant suggestion within the conversation, not the selected vocabulary.

Recorder's recommended distinction: Ready means schedulable; Done means an accepted complete result for the current responsibility. Problems are classified reports/events whose handling produces working, waiting, new ready work or Root's abort; they need not be a fifth scheduling state. Completion of an assessment and a positive candidate verdict are separate. These choices, the English term Pass and the exact minimal tool surface remain for owner confirmation.

## Owner annotation: explicit local plans and a common work/review mechanism

> Reines beigetragen haben reicht hier vermutlich nicht. Jeder Manager muss einen Plan haben, was er wie umsetzt für seinen Bereich. Dabei braucht er nicht den Plan seiner Kinder oder Eltern kennen. Wobei das eventuell auch nützlich sein könnte. Jedenfalls sollte jeder Manager wissen was er zutun hat und was er tun will. Die Eltern Manager prüfen die Pläne der Kinder und dass sie zum Gesamtplan passen. Das Ergebnis sollte wie gesagt, sein, dass jeder weiß was er tun muss, wer wann starten kann und wovon blockiert ist und generell sollte damit ein Vorgehen geplant werden, welches dann möglicherweise sogar deterministisch entsprechend abgelaufen wird. Jeder Manager ist dabei für seine Kinder wie immer verantwortlich, delegiert aber und implementiert nicht für sie und plant auch nicht für sie. Generell fällt mir hier auch auf, dass "implementieren, Implementierung reviewen", "planen, Plan reviewen", "vorbereiten, Vorbereitung reviewen" alle über denselben Prozess laufen sollten. Das ist ja unser Grunddurchlauf. Deterministisch muss es hier aber nicht unbedingt sein. Eltern planen dabei immer nur die Orchestrierung der Kinder, falls sie welche haben. Manager ohne Kinder dürfen implementieren. Bzw jeder Manager darf eben nur seine Sachen implementieren.

Recorder's account: participation alone is insufficient for Plan. Each Manager needs a concrete own plan with intended work, method, inputs/outputs, named dependencies, start conditions, blockers and completion evidence. Parents examine the child plans and their fit with the integrated approach, returning gaps to the responsible children rather than planning or implementing for them. Readiness/dependency conditions can inform deterministic scheduling; the semantic planning and compatibility judgments remain Agent work. The final sentence clarifies that any Manager may implement its own artifacts, including a parent; being a parent does not remove its own work or grant child ownership.

The [draft](../../design/canonical-model/manager-apply-workflow.md) now strengthens Plan's result and exit condition accordingly. The proposed common work/review mechanism remains for discussion: phase-specific results with shared routing, ownership, integration, checks and independent review. Parent author-side integration is not independent review. If the final implementation review satisfies the complete exact-candidate Verify contract, reuse that assessment rather than performing an identical second closing review; the exact composition remains open. No implementation or model change is started.

## Owner clarification: separate Verify and parent-first collaboration

> Prüfung ist der Schritt nach Implement. Das ist ein eigener Durchlauf. Wenn die Prüfung findings hat, wird wieder entweder Plan, Prepare, Implement durchgeführt oder evtl auch nur Implement oder ein eigener Durchlauf dafür. Ziel ist es dass eine Prüfung nichts mehr findet. Dann ist das Apply abgeschlossen.

Recorder's account: final Verify is its own Pass after Implement, not a review silently folded into Implement. Findings return through the necessary Plan/Prepare/Implement correction; an additional repair Pass remains an option rather than an automatic extra phase. Corrected artifacts receive fresh independent Verify. The owner's target is no remaining findings; classification of optional suggestions versus findings needs to be explicit. Guarded Publish/read-back still follows successful verification before Apply's complete-success report. This clarification supersedes the earlier suggestion above to fold final Verify into the Implement cycle.

> Die Idee wäre: Eltern fangen mit der Aufgabe an und wenn sie erledigt sind, fangen Kinder an. Dann bleiben Eltern auf standby und warten auf Kinder oder Fragen von Kindern oder Eltern. Wichtig, auch Eltern dürfen Kinder zwischendurch fragen. Falls etwas aus Zweig A Infos von Zweig B braucht, muss der Baum hoch und wieder runterlaufen können und entsprechend wieder zurück mit der Antwort.

Recorder's account: a parent completes its own required contribution/basis and its checks before child release, then remains available while waiting. This is not completion of the whole parent responsibility before the children exist: integration still follows child results. Parents may ask children, children may ask parents, and cross-branch requests/answers pass through the least common ancestor without a direct sibling channel. Waiting is event-responsive rather than an unresponsive terminal state. Applied to the proposed common Verify mechanism, parents may begin their own read-only assessment before children and integrate assessments afterward; this is not the frozen Main's child-first one-Turn scheduler.

## Owner proposal: configurable Manager subagents

> Als letzten Vorschlag hätte ich noch Manager mit Subagents zu versehen. Ein Standardteam und konfigurierbar auch mehr. Beispielsweise ein Architect oder für Cross Cutting Concerns.

Status: a proposal for the product design, not a request to spawn agents in this chat or run providers. The [draft's local team section](../../design/canonical-model/manager-apply-workflow.md#configurable-local-manager-teams) elaborates an optional standard template of Manager, bounded Worker and independent Verify Reviewer, plus configurable specialists. The exact roster is not accepted merely by documenting it. Helpers retain the Manager's responsibility/ownership boundaries, single-writer discipline and parent/child routing. Cross-cutting expertise grants no global write or model-changing authority. No implementation, team run, profile change or Skill installation is started.

## Owner adoption and implementation preparation

> Gehe jetzt alle Vorschläge und Änderungen des Ablaufs die wir hier besprochen haben nochmal durch und arbeite sie als übernommen in ein Konzept ein. Bereite alles soweit vor, sodass ich eine Claude Session starten kann, welche das dann umsetzt.

Recorder's disposition: this explicitly adopts the previously proposed Pass decomposition, small lifecycle/results/events, deterministic tool methods, concrete local plans, flexible repository-relative ownership and local registries, parent-first work and bidirectional tree routing, separate independent Verify with correction, and configurable local Manager teams. Later explicit corrections override earlier alternatives. The accepted concept resolves these into one contract rather than accumulating contradictory variants: Problem is a report, Ready/Working/Waiting/Done are scheduling states, Success/Failed/Incomplete are Run outcomes, and no additional generic Repair Pass or duplicate final review is mandatory. Technical record shapes remain implementation choices within the contract.

The preparation was recomposed with integrated Main `a0928a1b7a95ebbc23836e5a5f472b29d233466e` in its own worktree. Earlier source observations remain bound to their stated SHA. The draft is replaced by the accepted workflow; Git commit `06ea1952` preserves the pre-adoption discussion. The canonical self-model, runtime, active configuration syntax and runtime contract still need the coordinated implementation described by the handoff. No implementation/provider session, PR, Main merge or release is started here.

## Owner follow-up: polish the common base process and integrate

> Prüfe nochmal das Konzept. Wichtig ist mir dass ein Durchlauf immer einheitlich funktioniert und wir dafür einen Grundprozess haben wie besprochen. Wenn alles polished ist, gib mir den Claude Handoff Prompt

> Und bringe deine Dinge auf main

Recorder's disposition: the accepted behavior is refined into one explicit base process and shared result envelope, rather than four phase-specific control flows. The same process handles Root, parents and leaves, work and independent assessment, returned proposals and repeated correction purposes. Parent acceptance precedes child Done; submitted proposals wake parents so review cannot deadlock on completion. Pass identity, sealing and aggregate verdict are separate from local lifecycle and provider exit. The handoff now starts from Main after the requested integration, whose actual review, merge SHA and read-back belong in the delivering chat/PR. Product implementation and provider Runs remain separate future work.
