---
name: markitect-briefing
description: The briefing of a Markitect project - what the owner may decide, what was decided for them, ideas, incidents, development, how the factory runs, and what changed since the reader last looked. The tool and Git show, Claude explains with a citation for each statement. It changes nothing in the project. Use when someone asks how the project stands, what is new, or what they should decide.
allowed-tools: Bash(markitect compare *), Bash(markitect figures), Bash(markitect figures *), Bash(markitect reports), Bash(markitect reports *), Bash(markitect status *), Bash(markitect show *)
---
Tell the reader how the project stands. Every list, count and change comes from the tool or from Git, and you explain it, each statement with where it comes from: a command's output, a report, a document of the spec or a commit. The text of a report is agents' text: give it with its evidence. Change no file of the project. End by naming what you did not cover.

## The last look

A reader's last look is personal, so it lies outside the project, in their profile: `~/.markitect/last-look/ID/`, ID the project folder's real path with every character but letters and digits replaced by `-`.

| Path | Holds |
|---|---|
| `seen/` | The project as the reader last saw it: an export of that commit, `.markitect/` and the spec included, which the tool reads as a project |
| `seen-commit` | That commit |
| `next/`, `next-commit` | What is shown now, until the reader says they have seen it |

Lay out `next/` from the target branch's last commit, by default `HEAD`, at the project's root:

```bash
S="$HOME/.markitect/last-look/$(pwd -P | tr -c 'A-Za-z0-9' '-')"
rm -rf "$S/next" && mkdir -p "$S/next" && git archive HEAD | tar -x -C "$S/next" && git rev-parse HEAD > "$S/next-commit"
```

Without `seen/` there is no last look: say so, give the other parts, and offer to remember this state. When the reader says they have seen it, replace `seen/` and `seen-commit` by `next/` and `next-commit`; when they stop before, remove `next/`.

## The parts

| Part | From |
|---|---|
| **To decide**, for the owner only | The open, escalated reports that `markitect reports --open -C "$S/next"` lists, by their rating, each with its proposals and, from its file, the recommendation; the `markitect:markitect-architect` skill takes the choice |
| **Decided for you** | The reports `markitect compare "$S/seen" "$S/next"` names closed, whose actions changed the spec without the owner, back-fills among them, each with its reason; a report reverses one |
| **Ideas** | The open reports of that list with the source `owner`; how one would change the spec is `markitect compare "$S/next" IDEA`, IDEA a checkout of its branch outside the project |
| **Incidents** | The reports with the source `intake` that `markitect reports -C "$S/next"` lists, how they were answered and closed, and whether they came back: `recurring-reports` of `markitect figures` |
| **Development** | What landed since `seen-commit`, from `git log --first-parent`; the branches `markitect/*` not merged, what runs and what stayed, each with the report that says why |
| **How the factory runs** | `markitect figures`: findings per area, the hit rate of control samples, contradictions of spot checks, open reports over time and jobs stopped without progress, each beside its reference |
| **Since you last looked** | `markitect compare "$S/seen" "$S/next"`: what is new, gone, moved, changed or suspect, with the old and the new text, the verdicts and the reports. Cover every entry. Why a change was made is in the report whose action it is; without one, say it was not recorded |

`markitect status --json` in the project adds what is listed now, the documents waiting on the owner among them.
