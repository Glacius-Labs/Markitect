# Responsibilities, build start and the oracle — owner, 19 October 2026

Recorder's note: verbatim owner statements from the Claude implementation session, given after the first real library demonstration of the recursive Apply ([validation record](../../validation/recursive-apply-demonstration-20261019.md)). [DEC-059](../register.md#dec-059-enforced-responsibilities-build-start-and-an-outside-oracle) records the decision; the [runtime contract](../../design/canonical-model/manager-apply-runtime.md) owns the implemented behavior.

## The oracle observes after each Apply

> Oracle sollte einmal nach jedem Apply schauen, was passiert ist und ob etwas übersehen wurde. Er sollte sich im Prozess nicht einmischen.

## Responsibilities

> Das mit den Zuständigkeiten müssen wir dringend lösen. Dass der Root Manager alles alleine machen will und seinen Kindern sagt sie brauchen nichts tun darf nicht sein.

## Starting situation and the next run

> Startsituation sollten wir ebenfalls entsprechend implementieren.

> Sobald wir das entsprechend haben sollten wir den Lauf nochmal starten und schauen wie es läuft

Recorder's note: "Startsituation" refers to the proposal discussed immediately before: in starting situation 1 of [DEC-057](../register.md#dec-057-three-fundamental-starting-situations), Root prepares the skeleton first and the children build on it. An earlier statement in the same session, "Deadlines kannst du gerne generell auf 20 Minuten hochsetzten", set each Apply's budget to 20 minutes inside the demonstration's unchanged 30-minute limit.

## Execute-first now

Recorder's note: after DEC-058 (execute-first, documentation only) reached Main from a parallel discussion, the Claude session asked whether to implement it before the new run: "Auf Main steht jetzt DEC-058 (Execute zuerst ab Root, keine initiale Verify-Runde, kein Sonderweg für Fall 1), dort mit dem Vermerk „noch keine Implementierung“. Soll ich das vor dem neuen Lauf umsetzen?" The owner chose the option "Jetzt umsetzen (Empfohlen)": one uniform process for every Apply, Root first, each child after its parent's first turn, all Managers examine their areas in Execute, then independent Verify rounds; the initial Verify round and the separate build route are dropped.
