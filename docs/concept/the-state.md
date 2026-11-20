# The state

**The spec is the interface, and the artifacts are its implementations. The state maps one to the other** ([DEC-124](../decisions.md#dec-124-a-state-maps-each-document-to-its-artifacts-and-status-is-the-plan)). An artifact is any file that comes from the spec: code, tests, configuration, a handbook or a checklist. The unit is one document: a process with five step documents has six verdicts, and the process links its steps, so it becomes suspect when one of them changes.

## One row per document

The state lies in `.markitect/` and holds one row per document:

- its **artifacts**, which the reviewer names when it confirms;
- its **counter-checks**, apart from its artifacts ([§7](trust.md#counter-checks));
- its **verdict**, `ok`, `fail` or `skip`, with the evidence or the reason;
- the **hashes** at its last confirmation: of its own text, its artifacts and its evidence, which tell whether it changed; and of its type and the documents linked in either direction, which tell whether it is suspect.

The mapping stands nowhere else: not in the artifact, not in the spec. The builder keeps no list of files. `markitect show` gives a document with its artifacts and verdict, and a table gives all documents with theirs.

| Verdict | Means |
|---|---|
| `ok` | The document is realized, as the evidence shows, and its counter-checks pass |
| `fail` | It is not, or a counter-check fails; the row says why |
| `skip` | Checked as far as the project can show, with the reason |

A gap or contradiction in the spec is no verdict: it is a report, and the document waits on it ([§9](reports-and-actions.md)).

## What `status` lists

`status` is the plan: what moved since the last confirmation, computed from the folder and the state alone.

| Entry | When | Who acts |
|---|---|---|
| **Changed** | The document's own text, one of its artifacts or its evidence moved, it has no row yet, or its verdict is `fail` | A builder realizes or repairs it, then a reviewer confirms it |
| **Suspect** | A document it links to, or one that links to it, changed, or its type changed | A reviewer judges whether the change means something here, mostly without a builder |
| **Unexplained** | A file is mapped to no document and not ignored, a file of a document that is gone included | The reviewer maps it when it confirms, or it is removed or ignored |
| **Waiting on report X** | An open report about a gap or a contradiction in the spec concerns the document | Triage, or the owner when the report is escalated |

Links count in both directions, as suspect links do in DOORS: a changed rule reaches what it links and what links to it. "Applies to" is an ordinary section, and there is no binding by place. A changed type makes its documents suspect; a new required section missing from them is already an error of `check`.

**Every file is mapped or ignored.** Ignored means listed in `.gitignore` or in `.markitect/ignore`, written like a `.gitignore`. Anything else is unexplained and keeps `status` from being empty, so nothing grows in the project without a reason in the spec. During a build that is the normal state of a new file until the reviewer maps it. Only the architect and the owner write the ignore file ([§12](jobs-and-runners.md#rights-come-from-the-steps)).

**`check` and `status` stay apart**, as `validate` and `plan` do. `check` compares nothing with artifacts; everything that compares spec and artifacts is in `status`. `status` gives no list while the spec has errors.

`status` exits 1 when it lists anything. Documents waiting on the owner are shown and do not make it fail. On the main branch, where every spec change lands with its realization ([§8](the-factory.md#the-landing-rule)), `status` in CI is a check for drift: a changed, suspect or unexplained entry there came past the jobs. Given the state of the target branch, `status` applies the landing rule; given the states before and after a step and its role, it checks the write table ([§11](the-tool.md)).

## Binding a verdict

`status` prints each entry as `DOCUMENT@FINGERPRINT`, the fingerprint being the start of the hash of what the verdict will rest on. `markitect mark DOCUMENT@FINGERPRINT VERDICT` binds the verdict, with the artifacts and the evidence, to exactly that state, and refuses when anything changed since: a builder editing beside the reviewer, or the reviewer's own test run changing a file. The evidence must be files of the folder, and their hashes count, so a weakened test that served as evidence brings its document back. A verdict holds until something it rests on changes, as Go's test cache keeps a result while its inputs are unchanged.

## Moving a document

A document is known by its path. `markitect move` moves it, rewrites every link to it in the spec, the type lines included, and the relative links inside it, and carries its row in the state to the new path. The documents whose links it rewrote changed only mechanically: `move` computes their hashes anew, records the move in the history, and puts none of them on the list.

## The history

At each confirmation the history records, as hashes, which documents were changed at that moment. Only `mark` and `move` write it, and it holds facts only, nothing about how a job ran. From it the tool computes the change signals: a wide change, documents that change together without a link, links dragged along, hotspots, and back and forth where an old hash returns ([§5](is-the-spec-good.md#its-changes)). What a statement said before comes from Git, read by an agent, not by the tool.

**How it merges.** `.markitect/.gitattributes` has Git keep both sides of the state and the history. Of two rows for one document, `status` takes the one whose hashes match the folder; if both match with different verdicts, it lists the document to confirm again.

## The honest limit

`status` reaches one step along links, in both directions, and not into the code's own imports. A document that names another without linking it is a gap: a change of the named one does not reach it. The writing rule, the signal "mention without a link" and counter-checks narrow the gap, and spot checks and full reviews are the net ([§7](trust.md)). An empty `status` says that every document was confirmed at its current state, not that every verdict is right.
