# Trust

**Trust is measured, not granted** ([DEC-125](../decisions.md#dec-125-trust-is-measured-not-granted)). Nobody approves the agents' work, and no trust comes from a person looking through it. A mechanism counts only when it delivers a fact that can turn out negative without anyone reading: a run, a figure, a deviation. Another agent's opinion is no such fact.

The image is a testing laboratory. Nobody there looks at every result. People trust it because testing is apart from making, because control samples with a known result run along, and because the results hold up in use.

| Testing laboratory | Markitect |
|---|---|
| Production | The builder realizes a story |
| Testing, apart from production | A fresh reviewer checks and confirms with evidence |
| A test procedure derived from the standard, not from the maker | Counter-checks, derived from the spec alone |
| A control sample with a known result | Planted deviations the reviewer must find |
| Use and complaints | Findings from use, and the observers that keep watching |

## Two halves

- **Do the artifacts match the spec?** Confirmation and use secure this without people, and counter-checks, control samples and spot checks measure it once the runs bring them.
- **Does the spec say what is meant?** No mechanism and no reader can judge this without the people who decide. It shows in use, through findings that name the property "correct" ([§5](is-the-spec-good.md)). People set the intent when they plan with the Architect and make the new decisions in conversation ([§9](the-work.md)); they read neither artifacts nor every spec change.

## Confirmation

A fresh reviewer per round checks the build against the story's plan and target picture and against every level of the spec above it, up to the global one. It reads each document the story concerns, what it links to and its artifacts, takes its list from `status`, and treats what the builder says as claims to check. It changes nothing that was built and returns `comments`. When builder and reviewer agree, it runs the targeted tests and binds its verdict with `confirm` ([§6](the-state.md#binding-a-verdict)). The tests a builder writes are realizations: a green one is evidence, not proof. Alone, a reviewer is an opinion; the mechanisms below turn it into figures.

**Counter-checks, control samples and spot checks come later, with the runs.** Building with the way of working does not wait for them; they measure it once they exist ([roadmap](../roadmap.md)).

## Counter-checks

A counter-checker derives checks from the spec alone, before the builder starts and without seeing any artifact: tests for code, and questions with an expected answer for documents such as a handbook. A competency question the spec answers can serve directly: the same question put to the artifact must give the same answer.

- They run against what is built, and a failing one is a `fail`. So they catch the error that builder and reviewer make together when both read the spec the same wrong way.
- They are files of the project and stay when Markitect is removed. The state names them per document, apart from its artifacts.
- The builder may read them and may not change them.
- A document's counter-checks are its cases. Whether a document has too few is a signal against the reference value of its type, not a fixed threshold.

**Architecture tests.** Where a rule of the spec names modules as the code names them, and says what may depend on what, the counter-checker writes architecture tests in the tool the project already has, such as ArchUnit, dependency-cruiser or Import Linter. They are red until it is built, like any test written before the code, and run in CI. A rule that names no modules fixes nothing to test and stays with the reviewer. So whether the code follows the architecture is checked by a test, and Markitect needs nothing specific to a programming language for it.

## Control samples

A sampler copies the artifacts into a throwaway copy and plants known deviations from the spec in it, in code as in handbooks, and notes which and where. The reviewer checks the copy as usual, without knowing what was planted. The list of planted deviations and the reviewer's verdicts are kept in the run's context; the copy never lands. The hit rate computed from the two is the measured answer to "what is an `ok` worth?". It measures the checker and leaves the work as it is.

## Spot checks and full reviews

A spot check checks confirmed documents again at random, with a fresh reviewer. When the new verdict contradicts the old one on a document that did not change, that is a fact: one of the two verdicts is wrong. The rate of such contradictions says how far the confirmed rows of the state can be relied on. A full review checks every document, rarely.

## Convergence over time

Observers watch the project on a schedule, and every finding goes through the same circle ([§8](the-circle.md)). A single run may leave errors; the measure is that deviations shrink over time. Regular work could also converge on what the agents misread, and then everything looks calm. Control samples show whether the reviewer finds what it should, and counter-checks whether builder and reviewer share a misreading. `status`, the findings and these figures show whether the project converges ([§10](overview.md)).

## Who checks

| Agent | Does | Changes |
|---|---|---|
| **Architect** | Works out the spec and the plan with a person | The spec and the plan |
| **Builder** | Realizes a story on its branch | Artifacts, on the branch that carries the story's spec change |
| **Reviewer** | Checks the build in rounds, returns `comments`, and confirms with evidence | Nothing that was built; it writes `confirm` |
| **Counter-checker**, later | Derives counter-checks from the spec | Its checks, never the artifacts |
| **Sampler**, later | Plants deviations in a throwaway copy | Nothing that lands but the list |

The model of each agent stands at its Agent document in the spec ([§12](the-project.md)): by default a smaller model for the builder and the reviewer, and a larger one for the agents on the spec, which is little text that everything else follows from. Control samples and counter-checks show whether another model catches more.

## How firm it is

Markitect guards against forgetting, which is what has been seen, and against a shortcut or a deliberate bypass only by making it visible.

| What people want | What secures it | How firm |
|---|---|---|
| The spec is correct in itself before it is built | `status` gives no list while `check` finds errors | Firm |
| Every document is confirmed at the state it is in | The state's hashes, and `confirm` binding a verdict to the fingerprint the reviewer read | Firm for one step of links in both directions and for every file; the code's imports are not followed |
| Nothing grows without a reason | Every file mapped or ignored, else unexplained | Shown by `status`; an entry becomes a finding |
| Work outside the usual way is seen | Deviations noted by the agent at hand, and an observer of what happened on the main branch ([§8](the-circle.md#deviations)) | Shown, not prevented |
| The reviewer finds deviations | Control samples | Measured, as a hit rate, once they run |
| Builder and reviewer do not share a misreading | Counter-checks from the spec alone | Good where the spec states what it means, once they run |
| Confirmed rows can be relied on | Spot checks | Measured, as a rate of contradictions, once they run |
| The spec says what was meant | Findings from use | Shown late, not prevented |
| The code is right | Reviewer, the project's tests and later counter-checks | Not guaranteed |
