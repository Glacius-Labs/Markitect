# The model is authoritative; defects escalate — owner, 19 October 2026

Recorder's note: owner statements in the Claude session that tests and stabilizes Apply, after the status report on the Apply evidence. The contradiction scenario RA-C1 had put a contradictory sentence into the library model ("No member may ever hold more than four open loans" beside `maxOpenLoans: 5`). Root asked the owner, a scripted owner answered that the spec is binding, and the run continued with that run-only answer until its deadline. In RA-Q1, run without bound owner decisions, the Managers decided boundary edge cases themselves without asking. [DEC-063](../register.md#dec-063-the-model-is-authoritative-defects-escalate) records the decision.

## The owner's statements

> Wir wollen das Modell generell nicht reparieren im Apply. Wenn es kaputt oder schlecht ist, so sollten wir abbrechen oder es entsprechend melden oder ähnlich. Nicht sill reparieren. Das ist wie bei Code.

Recorder's note: the session then asked how Apply should react when a Manager finds a contradiction or gap in the model that affects the result at the project boundary. It offered three options: "Melden und abbrechen (Recommended)", "Fragen, Antwort ins Modell" and "Wie heute lassen". The owner first selected "Melden und abbrechen (Recommended)", then interrupted:

> Warte, es kommt darauf an. Wenn das Modell selbst sich widerspricht melden und abbrechen. Wenn es ausserhalb zu Problemen kommt eskalieren. Oder generell eskalieren, dabei gilt aber zwansläufig was im Modell steht. Notfalls wird es bis Root und raus eskaliert und bricht ab. Was denkst du?

## The session's proposal

Recorder's note: the session agreed with general escalation and proposed one rule, summarized here:
- The model always holds. Apply never changes it, and no answer may contradict it.
- A Manager that cannot settle something within the model escalates to its parent. The parent settles it when the model allows, for example which child realizes a shared rule, a tool or environment question, or an internal technical choice. Otherwise it escalates further.
- Root decides between two cases:
  - A model defect, meaning the model contradicts itself or a gap changes behavior at the project boundary. Root reports it to the owner, and the run ends at once Incomplete with a model-defect report. Nothing is published and the work is retained. The owner corrects the model, and a new Apply follows. There is no run-only answer for it.
  - Not a model defect, but not solvable in the tree: Root asks the owner a question, and the run continues with an answer that stays within the model.
- Gaps without effect at the project boundary are realization freedom (DEC-061). The Manager decides and documents them without escalation.

Recorder's note: the session also proposed to first finish the Apply evidence report and then implement the rule.

## The answer

> Ja mache es so
