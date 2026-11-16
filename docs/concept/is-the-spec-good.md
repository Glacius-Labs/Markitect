# Is the spec good?

A spec is judged like code, so it gets the same kind of analysis code gets. `check` is the compiler: is it valid? `vet`, named after Go's, is the static analyzer: is it well designed? It warns or notes, never refuses, since granularity stays the author's choice, and its thresholds are provisional ([DEC-102](../decisions.md#dec-102-what-vet-measures)).

| Measure | Code analogy | `vet` |
|---|---|---|
| Efferent and afferent coupling, instability, per Package | Clean Architecture | Warns: `vet.coupling`, `vet.unstable_dependency` |
| Size of a Definition and a Package | Long functions, large modules | Warns: `vet.definition_size`, `vet.package_size`; notes long lists of statements, `vet.statements` |
| Packages with no paths, or no Definitions and one child | Empty namespaces | Warns: `vet.no_paths`, `vet.thin_package` |
| Fan-in of a Definition, Root included | God class, `utils` file | Warns: `vet.hub` |
| A Definition named in prose without a link | An undeclared dependency | Notes `vet.unlinked_mention`; the auditor tells missing links from ordinary words |
| A Definition all of whose users lie in one Package below it | Misplaced code | Notes `vet.placement` |
| Change hotspots: revisions between verdicts times fan-in | Churn | Notes `vet.hotspot`; revisions come from the history, since the tool reads no Git history |
| Two Definitions whose texts keep changing together with no link between them | Change coupling | Notes `vet.change_coupling`; co-changes come from the history |
| Abstractness and distance from the main sequence | Martin's metrics | Shown by `list --deps`, no warning |
| Size against what the evidence demonstrates | CRAP | Notes `vet.risk`, from the reviewer's counts on `mark` |
| A field value out of the form its Kind leaves to `vet`, a refusal code no Rule states, a State no Outcome reaches, a **Who** without a Role | Type and consistency checks | Notes `vet.field_value`, `vet.refusal_code`, `vet.unreached_state`, `vet.who_role` ([DEC-117](../decisions.md#dec-117-a-kind-declares-the-form-of-a-fields-value)) |
| The spec's dependencies against the code's imports | Architecture tests | Not yet |

**Package cycles are a compile error**, as in Go. [The spec language guide](../guides/spec-language.md#vet-warnings-and-notes) gives every code and threshold, and [what makes a good spec](../guides/good-spec.md) says what each measure is for.

**Why no central page of values.** On the Makerspace specs, one page, `values`, with thirteen unrelated value formats, was linked by 60 Definitions, and by 268 on a project five times as large: a `utils` file. Split into one Definition per format, a change to one format lists between 2 and 52 Definitions instead of 61 ([measurement](https://github.com/Glacius-Labs/Markitect/blob/d0c23a31efc928e95f3acc0633deb01a5f887884/docs/validation/spec-design-measures-20261107.md)). So a value rule of the project's own is a small Definition of its own, where it is used.

**What analysis can't say:** whether the spec is right. No threshold has been checked against outcomes yet, so a clean report doesn't show that a spec leads to better work.
