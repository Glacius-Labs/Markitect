# Trust

**Trust is measured, not granted** ([DEC-125](../decisions.md#dec-125-trust-is-measured-not-granted)). Nobody approves the agents' work, and no trust comes from a person looking through it. A mechanism counts only when it delivers a fact that can turn out negative without anyone reading: a run, a figure, a deviation. Another agent's opinion is no such fact.

The image is a testing laboratory. Nobody there looks at every result. People trust it because testing is apart from making, because control samples with a known result run along, and because the results hold up in use.

| Testing laboratory | Markitect |
|---|---|
| Production | The builder realizes the spec |
| Testing, apart from production | A fresh reviewer confirms with evidence |
| A test procedure derived from the standard, not from the maker | Counter-checks, derived from the spec alone |
| A control sample with a known result | Planted deviations the reviewer must find |
| Use and complaints | Reports from use, and the jobs that keep running |

## Two halves

- **Do the artifacts match the spec?** Confirmation, counter-checks, control samples, spot checks and use secure this without people.
- **Does the spec say what is meant?** No mechanism and no reader can judge this without the owner. It shows in use, through reports that name the property "correct" ([§5](is-the-spec-good.md)). The owner sets the intent in conversations and makes the new decisions ([§9](reports-and-actions.md)); they read neither artifacts nor every spec change.

## Confirmation

A fresh reviewer takes its list from `status`, reads each document, what it links to and its artifacts, runs the project's tests and the counter-checks, and binds its verdict with `mark` ([§6](the-state.md#binding-a-verdict)). It treats what the builder says as claims to check. The tests a builder writes are realizations: a green one is evidence, not proof. Alone, a reviewer is an opinion; the three mechanisms below turn it into figures.

## Counter-checks

A counter-checker derives checks from the spec alone, before the builder starts and without seeing any artifact: tests for code, and questions with an expected answer for documents such as a handbook. A competency question the spec answers can serve directly: the same question put to the artifact must give the same answer.

- They run against what is built, and a failing one is a `fail`. So they catch the error that builder and reviewer make together when both read the spec the same wrong way.
- They are files of the project, in the folder the settings name, and stay when Markitect is removed. The state names them per document, apart from its artifacts.
- The builder may read them and may not change them ([§12](jobs-and-runners.md#rights-come-from-the-steps)).
- A document's counter-checks are its cases. Whether a document has too few is a signal against the reference value of its type, not a fixed threshold.

**Architecture tests.** Where a rule of the spec names modules as the code names them, and says what may depend on what, the counter-checker writes architecture tests in the tool the project already has, such as ArchUnit, dependency-cruiser or Import Linter. They are red until it is built, like any test written before the code, run in CI and belong to the landing rule. A rule that names no modules fixes nothing to test and stays with the reviewer. So whether the code follows the architecture is checked by a test, and Markitect needs nothing specific to a programming language for it.

## Control samples

A sampler copies the artifacts into a throwaway copy and plants known deviations from the spec in it, in code as in handbooks, and notes which and where. The reviewer checks the copy as usual, without knowing what was planted. The list of planted deviations and the reviewer's verdicts land as a file in `.markitect/`; the copy never lands. The tool computes the hit rate from the two, so that it is no agent's claim. The hit rate is the measured answer to "what is an `ok` worth?". It measures the checker and leaves the work as it is.

## Spot checks and full reviews

A spot check checks confirmed documents again at random, with a fresh reviewer. When the new verdict contradicts the old one on a document that did not change, that is a fact: one of the two verdicts is wrong. The rate of such contradictions says how far the confirmed rows of the state can be relied on. A full review checks every document, rarely.

## Convergence over time

Regular jobs build, review, hunt bugs, clean up and improve ([§8](the-factory.md#the-jobs)). A single run may leave errors; the measure is that deviations shrink over time. Regular runs could also converge on what the agents misread, and then everything looks calm. Control samples show whether the reviewer finds what it should, and counter-checks whether builder and reviewer share a misreading. `status`, the reports and these figures show whether the project converges ([§10](overview.md)).

## Roles

| Role | Does | Sees |
|---|---|---|
| **Architect** | Writes the spec, in a conversation with the owner or as a job | The spec and the project |
| **Counter-checker** | Derives counter-checks from the spec | The spec, never the artifacts |
| **Builder** | Realizes the spec | Everything; changes only artifacts |
| **Reviewer** | Confirms with evidence and binds verdicts | Everything; changes nothing |
| **Sampler** | Plants deviations in a throwaway copy | Everything; nothing it writes lands but the list |
| **Reader** | Reads and questions the spec; findings become reports | The spec |

The model per role is a setting: by default a smaller model for the builder and the reviewer, and a larger one for the roles on the spec, which is little text that everything else follows from. Control samples and counter-checks show whether another model catches more.

## How firm it is

Markitect guards against forgetting and shortcuts, which is what has been seen, and against a deliberate bypass only by making it visible.

| What the owner wants | What secures it | How firm |
|---|---|---|
| The spec is correct in itself before it is built | `status` gives no list while `check` finds errors | Firm |
| Every document is confirmed at the state it is in | The state's hashes, and `mark` binding a verdict to the fingerprint the reviewer read | Firm for one step of links in both directions and for every file; the code's imports are not followed |
| Nothing grows without a reason | Every file mapped or ignored, else unexplained | Firm |
| No role does more than its part | Each role a step; the tool checks every change against the write table, read from the target branch | A fact in CI, where the runner is a script no agent can change; in a session it makes shortcuts visible and proves nothing |
| What lands makes nothing worse | The landing rule ([§8](the-factory.md#the-landing-rule)) | A fact where CI checks it before the merge |
| The reviewer finds deviations | Control samples | Measured, as a hit rate |
| Builder and reviewer do not share a misreading | Counter-checks from the spec alone | Good where the spec states what it means; a counter-check that derives what the spec does not say is a signal about the spec |
| Confirmed rows can be relied on | Spot checks | Measured, as a rate of contradictions |
| The spec says what was meant | Reports from use | Shown late, not prevented |
| The code is right | Reviewer, counter-checks and the project's tests | Not guaranteed |

**Anyone who wants more** runs `status`, the counter-checks and the project's tests where no agent can reach them, in CI. That is also where the landing rule becomes a fact.
