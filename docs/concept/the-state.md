# The state

**The spec is the interface, and the artifacts are its implementations. The state maps one to the other** ([DEC-124](../decisions.md#dec-124-a-state-maps-each-document-to-its-artifacts-and-status-is-the-plan)). An artifact is any file that comes from the spec: code, tests, configuration, a handbook, a skill or a checklist. The unit is one document: a process with five step documents has six verdicts, and the process links its steps, so it becomes suspect when one of them changes.

The state answers the question "where does it say that it should be so?" for every file: which document a file realizes, and with which evidence that was confirmed.

## One row per document

The state lies in `.markitect/` and holds one row per document:

- its **artifacts**, which the reviewer names when it confirms;
- its **counter-checks**, apart from its artifacts, once the runs bring them ([§7](trust.md#counter-checks));
- its **verdict**, `ok`, `fail` or `skip`, with the evidence or the reason;
- the **hashes** at its last confirmation: of its own text, its artifacts and its evidence, which tell whether it changed; and of its type and the documents linked in either direction, which tell whether it is suspect.

The mapping stands nowhere else: not in the artifact, not in the spec. The builder keeps no list of files. `markitect show` gives a document with its artifacts and verdict, and a table gives all documents with theirs.

| Verdict | Means |
|---|---|
| `ok` | The document is realized, as the evidence shows |
| `fail` | It is not; the row says why |
| `skip` | Checked as far as the project can show, with the reason |

A gap or contradiction in the spec is no verdict: it is a finding ([§9](the-work.md)).

The documents of the work in `.markitect/`, plan items, findings, reports and briefings, are not documents of the spec and have no row. A link from them into the spec makes nothing in the spec suspect, and `move` rewrites those links when it moves the document they name ([moving a document](#moving-a-document)).

## What `status` lists

`status` is Terraform's `plan`: what moved since the last confirmation, computed from the folder and the state alone.

| Entry | When | Who acts |
|---|---|---|
| **Changed** | The document's own text, one of its artifacts or its evidence moved, it has no row yet, or its verdict is `fail` | A builder realizes or repairs it, then a reviewer confirms it |
| **Suspect** | A document it links to, or one that links to it, changed, or its type changed | A reviewer judges whether the change means something here, mostly without a builder |
| **Unexplained** | A file is mapped to no document and not ignored, a file of a document that is gone included | The reviewer maps it when it confirms, or it is removed or ignored |

Links count in both directions, as suspect links do in DOORS: a changed rule reaches what it links and what links to it. "Applies to" is an ordinary section, and there is no binding by place. A changed type makes its documents suspect; a new required section missing from them is already an error of `check`.

**Every file is mapped or ignored.** Ignored means listed in `.gitignore` or in `.markitect/ignore`, written like a `.gitignore`. Anything else is unexplained and keeps `status` from being empty, so nothing grows in the project without a reason in the spec. During a build that is the normal state of a new file until the reviewer maps it.

**`check` and `status` stay apart**, as `validate` and `plan` do. `check` compares nothing with artifacts; everything that compares spec and artifacts is in `status`. `status` gives no list while there are errors in the spec or in the state; errors in the documents of the work block nothing, and `check` reports them.

**`status` only shows** ([DEC-146](../decisions.md#dec-146-a-spec-change-lands-with-its-realization-and-status-only-shows)). Its exit code follows the tool's convention, and nothing uses it as a gate, in CI neither. Since every spec change reaches the main branch with its realization ([§8](the-circle.md#building)), an entry there came past the way of working: a change past the spec, an unexplained file or changed evidence. It becomes a finding like any other observation. In an existing project whose files nobody has mapped yet, `status` shows them all as unexplained; that is expected, and it makes no finding until a run of its own has mapped them.

## Binding a verdict

`status` prints each entry as `DOCUMENT@FINGERPRINT`, the fingerprint being the start of the hash of what the verdict will rest on. When builder and reviewer agree, the reviewer runs the targeted tests and writes `markitect confirm DOCUMENT@FINGERPRINT VERDICT`: for each document the story realizes, the files and the test that shows it. The tool binds them to exactly that state and refuses when anything changed since: a builder editing beside the reviewer, or the reviewer's own test run changing a file. The evidence must be files of the folder, and their hashes count, so a weakened test that served as evidence brings its document back. A verdict holds until something it rests on changes, as Go's test cache keeps a result while its inputs are unchanged.

The reviewer changes nothing that was built; `confirm` is the one thing it writes.

## Moving a document

A document is known by its path. `markitect move` moves it, rewrites every link to it in the spec and in the documents of the work, the type lines included, and the relative links inside it, and carries its row in the state to the new path. The documents whose links it rewrote changed only mechanically: `move` computes their hashes anew, records the move in the history, and puts none of them on the list.

## The history

At each confirmation the history records, as hashes, which documents were changed at that moment. Only `confirm` and `move` write it, and it holds facts only; how an agent ran is in its report ([§9](the-work.md#reports)). From it the tool computes the change signals: a wide change, documents that change together without a link, links dragged along, hotspots, and back and forth where an old hash returns ([§5](is-the-spec-good.md#its-changes)). What a statement said before comes from Git, read by an agent, not by the tool.

**How it merges.** `.markitect/.gitattributes` has Git keep both sides of the state and the history. Of two rows for one document, `status` takes the one whose hashes match the folder; if both match with different verdicts, it lists the document to confirm again.

## The honest limit

`status` reaches one step along links, in both directions, and not into the code's own imports. A document that names another without linking it is a gap: a change of the named one does not reach it. The writing rule and the signal "mention without a link" narrow the gap, and later counter-checks, spot checks and full reviews are the net ([§7](trust.md)). An empty `status` says that every document was confirmed at its current state, not that every verdict is right.
