# Primary goal and conformance evaluation, 21 October 2026

## Origin and status

Direct owner statements in the current concept assessment conversation. The owner commissioned documentation and assignment changes, followed by coordination with the Apply planning chat. No runtime implementation or additional provider series is commissioned here. Recorder notes below distinguish the accepted goal and evaluation direction from bounded execution planning and measured results.

## Owner statement: primary goal

> Das wichtigste ist, dass Markitect in der Lage ist das kanonische Modell zu lesen und zuverlässig dafür sorgen kann, dass sämtliche Artefakte damit konsistent sind. Dabei will ich dass nichts übersehen und nichts widersprüchlich ist. Daher die Aufteilung und unabhängige Prüfung pro Bereich und die Integration und Prüfung durch Eltern. Das ist die Hauptaufgabe von Markitect.
>
> Das umfasst natürlich:
>
> - Markitect kann ein Software System bzw Projekt mit seinem Kanonischen Modell ausreichend spezifizieren und darstellen
> - Das Kanonische Modell kann sowohl strukturell als auch inhaltlich zuverlässig auf Stimmigkeit, Vollständigkeit und Einheitlichkeit geprüft werden => Wobei es immer noch leichter ist ein kanonisches Modell sauber zu halten als unzählige Ausprägungen davon.
> - Das Kanonische Modell kann einfach und angenehm gepflegt werden (UX, hauptsächlich durch Agent, UI um es sich anzuschauen)
>
> Es geht primär um das Vertrauen in die Agenten durch Struktur und systematisches Vorgehen. Sekundär geht es darum, das Software Projekt auf Dauer widerspruchsfrei, einheitlich, konsistent und sauber zu halten.
>
> Wenn ich einen Agenten nehme und ihm sage "Prüfe ob das Software Projekt widerspruchsfrei, einheitlich, konsistent und sauber  ist und dass es das tut was es soll", was passiert dann?
>
> Was heißt "tut was es soll?".
>
> Agent plant eventuell Subagenten so wie es ihm gerade einfällt und schickt sie los.
>
> Jeder findet irgendwas und sucht ewig nach Widersprüchen, Cleanups etc.
>
> Jeder neue Run findet wieder etwas neues, was der vorherige übersehen hat.
>
> So wirklich sicher, dass alles sauber ist ist man sich nie wirklich und was jetzt gilt und was sein soll und was ist ist immer sehr neblig.
>
> Das soll Markitect lösen.

## Owner adoption and coordination request

> Halte dieses Hauptziel und die vorgeschlagenen Prüfungen im Konzept fest und passe die Aufträge A–D entsprechend an. Anschließend sollten wir auch den Apply Ablauf Chat darüber informieren, dieser plant nämlich die Umsetzung. Wir wollen hier on the same page sein.

## Recorder notes: adopted evaluation direction

The assessment proposed checking omitted material conditions despite complete Definition IDs and green ordinary tests; contradictions across code, tests, documentation and configuration; semantic parent integration despite locally passing children; and preservation without optional cleanup. Review derives expectations independently from binding model content rather than accepting the author's claim list as complete. Valid file accounting and receipts alone cannot prove semantic conformance.

Model adequacy needs an explicit project purpose and requirements as its reference, including boundaries and cross-area rules. Later bounded authoring and longitudinal exercises should assess this and carry each arm's actual output through creation, cross-area change, removal of a requirement and unchanged Apply. These directions do not launch either exercise. Quality, missed obligations, false blockers and unsupported success precede time/cost comparisons; readable evidence must expose assessed scope and unknowns.

The owning documents translate this direction into concrete scope: [vision](../../vision.md#primary-goal-and-prerequisites), [workflow](../../design/canonical-model/manager-apply-workflow.md#semantic-coverage-and-parent-integration), [measurement](../../measurement.md#evidence-for-the-primary-goal), and [A–D supplement](../../development/claude-direct-runtime-handoff-20261021.md#primary-goal-supplement-and-current-handoff). D's next selection prioritizes existing preservation, multi-artifact and semantic-integration cases plus the bounded realistic case; it still requires a frozen scope and execution authorization.

## Recorder notes: coordination boundary

The latest owner instruction read in the existing Apply planning chat, `Apply-Ablauf nachvollziehen` (`01a157ed-1baf-7401-a1d0-9dc760529eea`), says:

> Dann machen wir das so und nehmen überall Sol High.
> Ich habe für Claude Session A gesagt, dass sie zuende implementieren und ein paar kurze runs machen soll am Ende. Sie soll nicht mittendrin abbrechen. Erst dann übergeben wir. Die Session B läuft noch. Session C und D habe ich beendet.
> Wie machen wir jetzt weiter?

Recorder interpretation of that chat's subsequent planning: A and B finish before handoff; A's limited short closing runs remain authorized. Codex C can take over the stopped candidate without a new provider series, coordinating shared production integration through A. D waits for integrated inputs and a separate scope/model freeze. Sol High is the selected coding-session setting, not automatic replacement of the evaluation profiles or an execution grant. The concept update must be relayed to that chat after integration.
