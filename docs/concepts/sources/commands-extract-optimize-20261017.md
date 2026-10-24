# Commands, Extract and Optimize — 17 October 2026

Recorder's note: direct owner discussion in the Codex chat **Commands**, thread `01a14ac0-d4c1-7cb0-a070-d677f0033b45`. Owner quotations retain their original wording. Assistant explanations and the synthesis are identified separately. The final request authorizes documenting this direction on Main, not implementation or runtime execution. The [command design](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/commands.md) owns the synthesis; the first reliable Apply remains the delivery target.

## Owner command examples

> Hier ist wie ich mir das vorgestellt habe:
>
> - Apply arbeitet entlang des Modells und sorgt dafür dass alles zum Modell passt. Das könnte man als statischen Prompt verstehen.
> - Cleanup würde für jeden Manager heruntergehen und sie jeweils anweisen, nach Möglichkeiten zum Cleanup zu suchen aber halt so, dass es immer noch zum Modell passt. Auch das ist quasi ein static translated Prompt.
> - Bughunt würde jeweils nach Bugs suchen und sie dann entsprechend fixen. Auch das wieder static translated prompt.
> - Refactor such nach entlang des Baum nach Refactoring Möglichkeiten. Auch das als static translated Prompt.
> - Verify: Quasi wie Apply mit Befunden in der Hoffnung, dass alle Manager sagen passt so. Auch das als static translated Prompt.
> - Custom Commands: Prompt, welcher dann von oben nach unten in jeden Bereich durchgereicht und auf den entsprechenden Bereich konzentriert gemacht wird.
> - Extract analyisiert die Codebase, erstellt ein kanonisches Modell, führt verify aus, schaut sich das Ergebnis an und refined solange das kanonische Modell, bis ein Verify sagt, dass es so passt. Das würde man in Brownfield Projekten verwenden um Markitect einzuführen.
>
> Das sind grobe Beispiele für das was mir einfällt.

## Owner clarification of Verify

> Verify sollte nur prüfen und Befunde liefern. Es prüft nur ob der Stand dem kanonischen Modell entspricht und falls nicht wieso nicht.

The assistant summarized Verify as a read-only conformity assessment with supported conformity, concrete deviation, or insufficient evidence. It separated model refinement by Extract from assessment by Verify. The owner continued:

> Genau. Lass uns jetzt Extract genauer durchgehen: Wie entsteht das erste Modell, und wie werden die Verify-Befunde genutzt, um es schrittweise zu verfeinern?

## Extract describes the existing stand

> Extract soll erstmal den bestehenden Istzustand beschreiben. Wenn Code und dokumentierte Absicht sich widersprechen, soll es das als Frage vorlegen. Wie verhindern wir konkret, dass das Modell zu oberflächlich wird und Verify trotzdem passt sagt?

Recorder's note: the assistant first proposed a separate source-to-model coverage assessment. The owner steered toward reuse:

> Wir könnten die Detailtiefe entsprechend erweitern und den AI Agent, welcher das Modell ja erstellt soll ein vernünftiges Modell machen. Dafür brauchen wir eine Lösung und elegant wäre es natürlich, die bestehenden Capabilities von Markitect dafür zu reusen

Recorder's summary of the resulting assistant proposal: treat the extracted model as the candidate produced by the existing Manager organization. Initial discovery proposes Packages; Managers draft their model areas from existing evidence; independent local review checks accuracy, omissions, appropriate detail and unsupported assumptions; existing rework corrects findings; parents integrate and independently review the combined model; read-only Verify findings drive further model revisions. The same detail-depth objective goes to author and reviewer. Model-quality review asks whether important observed behavior could change without violating the proposed model. No separate permanent completeness catalogue is needed. This is assistant synthesis, subsequently covered by the documentation request below, not a verbatim owner instruction for each mechanism.

## Publication request and Optimize

> Dann halte das zusammen mit den anderen Commands in den Docs auf main fest.
> Ausserdem sollte es noch eine Art Optimize geben, welches das kanonische Modell untersucht und nach fehlenden Links sucht oder generell das kanonische Modell optimieren soll, ohne zu viel dazuzudichten.

Recorder's interpretation: record the agreed command direction and the proposed reuse for Extract, plus a model-focused Optimize that seeks justified missing links and improvements without unsupported invention. This authorizes the documentation delivery on Main despite the prior discussion-only scope. It does not authorize a new command implementation queue, a change to the artifact-only Apply target, or provider, Docker, Apply, comparison or study runs. Exact future schemas, settings and draft execution contracts remain design work.

## Optimize workflow clarification

The owner asked:

> Lass uns Optimize genauer durchgehen: Wie finden die Manager fehlende Links und sinnvolle Modellverbesserungen, ohne neue Anforderungen oder Beziehungen zu erfinden?

Recorder's summary of the assistant proposal: Managers start with their own Definitions and search beyond existing links through a simple model inventory, text search and targeted reading. Similar terms are candidates rather than relationships. Ordinary proposals explain source support, the change, preserved meaning and context benefit. Independent review checks unambiguous targets, stronger or lost obligations, preserved scopes and whether a link supplies useful context. Implementation observations can raise a model question but do not automatically become requirements. Managers retain their own write responsibility; parents coordinate cross-area questions and obtain independent integration review. Reuse ordinary findings, rework, structural checks and context previews. Outcomes are reviewed improvement, a grounded question or justified no-op; link counts are not success measures.

The owner accepted this and requested the remaining questions:

> Klingt gut, halte das so fest und schaue was es sonst noch zu klären gibt

Recorder's note: this accepts the preceding Optimize workflow and authorizes its documentation under the existing Main publication request. The remaining command questions and recommendations in the owning design are assistant analysis, not further accepted decisions. No implementation or runtime authority is added.

## Model adoption with known realization deviations

Recorder's note: a subsequent direct owner discussion on 17 October 2026 in **Usage, Skills, UX**, thread `01a14ac0-f40a-73a0-b401-e7e9d24ee4fc`. The earlier Commands statements above retain their history. This continuation explicitly changes the Extract completion criterion; it does not supply implementation or runtime evidence.

The owner asked:

> Lass uns jetzt die Modellübernahme konkret durchspielen: Wie erstellt der Coding-Agent aus Code, Tests und Doku einen Modellvorschlag, ohne vorhandene Fehler zu Regeln zu machen? Weiterhin keine Implementierung.

Recorder's summary of the assistant proposal: bind the source revision and scope, distinguish authoritative intent from observed behavior, inspect discriminating cases and actual assertions, and independently review the model for source support, omissions and unnecessary implementation constraints. In a hypothetical library, an authoritative five-loan rule and code admitting a sixth justify a correct five-loan model plus a retained realization deviation. They do not justify changing the rule to six. Unclear authority between conflicting 14-day and 21-day sources remains an owner question. The assistant identified that the existing all-conforming Verify completion condition would block adoption of a sound model in the presence of a known implementation defect, and proposed separating model readiness from realization conformity. These examples are reasoning about the design, not observed repository defects or test results.

The owner accepted the workflow and changed the criterion:

> Das klingt gut. Halte den Ablauf in unseren Docs fest und passe das Extract-Abschlusskriterium an: Ein geprüfter Modellvorschlag darf mit bekannten Realisierungsabweichungen übernommen werden. Weiterhin keine Implementierung.

Recorder's interpretation: the current [Extract contract](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/commands.md#completion-and-adoption-with-known-deviations) now allows adoption of a structurally valid, reviewed and integrated candidate with adequately supported known realization deviations. Material unresolved intent, inadequate model coverage and evidence gaps still prevent adoption readiness. Verify keeps its actual result, and deviations remain visible for later separately authorized artifact work. This is a documentation-only change to the later-operation design; it grants no runtime, model-writing implementation or artifact repair authority.

## Optimize search space and targeted reading

The owner asked:

> Lass uns als Nächstes den Suchraum von Optimize klären: Wie weit dürfen die Manager über bestehende Links hinaus lesen, ohne unnötig viel Kontext zu laden oder relevante Zusammenhänge zu übersehen?

Recorder's summary of the assistant proposal: distinguish searchable evidence, loaded context and change authority. Search the selected local model, explicitly bound dependencies and permitted artifacts from the fixed repository stand; load own obligations, explicit references and targeted candidate sources rather than every searchable byte. Every Manager assesses its own area. Concrete questions drive model-wide inventory/text search, followed by source matches, candidate Definitions and necessary contracts. Further questions justify further reads within the existing grant. Existing communication, parent integration and independent review examine cross-area meanings and differently named concepts; lexical similarity alone grants no relationship or write authority. Search results remain bounded with visible continuation. Budget/context limits and unfinished searches do not prove completeness. Ordinary reports retain relevant sources, conclusions and open questions without a second search catalogue. Start without a fixed link-depth limit or semantic-index subsystem; real use would supply evidence for later extensions.

The owner accepted the proposal and requested Main documentation:

> Klingt gut. Halte diesen Suchraum und den Ablauf für gezieltes Nachlesen in den Docs auf main fest. Weiterhin keine Implementierung.

Recorder's interpretation: this selects the [search and targeted-reading workflow](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/commands.md#find-candidates-beyond-existing-links), resolving its earlier open product-scope question. Existing model authority, write ownership, source bindings and budgets remain. Exact search APIs, output formats and numeric limits are not selected; no implementation, provider, Docker, Apply or study authority is granted.
