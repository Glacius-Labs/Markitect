# Commands, Extract and Optimize — 17 October 2026

Recorder's note: direct owner discussion in the Codex chat **Commands**, thread `01a14ac0-d4c1-7cb0-a070-d677f0033b45`. Owner quotations retain their original wording. Assistant explanations and the synthesis are identified separately. The final request authorizes documenting this direction on Main, not implementation or runtime execution. The [command design](../../design/canonical-model/commands.md) owns the synthesis; the first reliable Apply remains the delivery target.

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
