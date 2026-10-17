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

## Model adoption with known realization deviations

Recorder's note: a subsequent direct owner discussion on 17 October 2026 in **Usage, Skills, UX**, thread `01a14ac0-f40a-73a0-b401-e7e9d24ee4fc`. The earlier Commands statements above retain their history. This continuation explicitly changes the Extract completion criterion; it does not supply implementation or runtime evidence.

The owner asked:

> Lass uns jetzt die Modellübernahme konkret durchspielen: Wie erstellt der Coding-Agent aus Code, Tests und Doku einen Modellvorschlag, ohne vorhandene Fehler zu Regeln zu machen? Weiterhin keine Implementierung.

Recorder's summary of the assistant proposal: bind the source revision and scope, distinguish authoritative intent from observed behavior, inspect discriminating cases and actual assertions, and independently review the model for source support, omissions and unnecessary implementation constraints. In a hypothetical library, an authoritative five-loan rule and code admitting a sixth justify a correct five-loan model plus a retained realization deviation. They do not justify changing the rule to six. Unclear authority between conflicting 14-day and 21-day sources remains an owner question. The assistant identified that the existing all-conforming Verify completion condition would block adoption of a sound model in the presence of a known implementation defect, and proposed separating model readiness from realization conformity. These examples are reasoning about the design, not observed repository defects or test results.

The owner accepted the workflow and changed the criterion:

> Das klingt gut. Halte den Ablauf in unseren Docs fest und passe das Extract-Abschlusskriterium an: Ein geprüfter Modellvorschlag darf mit bekannten Realisierungsabweichungen übernommen werden. Weiterhin keine Implementierung.

Recorder's interpretation: the current [Extract contract](../../design/canonical-model/commands.md#completion-and-adoption-with-known-deviations) now allows adoption of a structurally valid, reviewed and integrated candidate with adequately supported known realization deviations. Material unresolved intent, inadequate model coverage and evidence gaps still prevent adoption readiness. Verify keeps its actual result, and deviations remain visible for later separately authorized artifact work. This is a documentation-only change to the later-operation design; it grants no runtime, model-writing implementation or artifact repair authority.
