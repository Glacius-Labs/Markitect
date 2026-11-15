---
name: markitect-catchup
description: Tell a person what changed in a project's spec since they last looked - for every changed item what is now true, how it was before, and why where that is recorded - without assuming they remember the old spec. Their last look is kept outside the project, in their own profile. It changes nothing in the project. Use when the user asks what changed, what is new, or to catch up on the spec or the project.
allowed-tools: Bash(markitect list), Bash(markitect list *), Bash(markitect doc *), Bash(markitect status), Bash(markitect status *)
---
Tell the user what changed in the spec since they last looked, from what `markitect` prints and their own copy of the spec. Change no file of the project.

**Their last look** is personal, so it lies outside the project, in a folder of their own per project folder. Run each block below as one command at the project's root in the Bash tool: its first line names that folder `S`, since a shell keeps no variable from one command to the next. `$S/last/` holds what `markitect status --all` printed then, as `status.txt`, a copy of `.markitect/spec/` as `spec/`, and `questions.md`, empty when the project had none. Commands on these copies may ask the user for permission.

**Start** by saving the state the user is about to see as `$S/next/`, and what changed since `last/` as `$S/next/changes.txt`:

```bash
S="$HOME/.claude/markitect-seen/$(pwd -P | tr -c 'A-Za-z0-9' '-')"
rm -rf "${S:?}/next" && mkdir -p "$S/next" && cp -R .markitect/spec "$S/next/spec"
markitect status --all > "$S/next/status.txt"; cp .markitect/questions.md "$S/next/" 2>/dev/null || : > "$S/next/questions.md"
items() { awk '/^[^ ].*@[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]$/ { if (k != "") print k; k = $0; s = 0; next } /^  with its / { next } /^  / && k != "" && !s { k = k " | " substr($0, 3); s = 1 } END { if (k != "") print k }' "$1/status.txt" | sort; }
[ -d "$S/last" ] && { diff <(items "$S/last") <(items "$S/next"); diff -rq --strip-trailing-cr "$S/last/spec" "$S/next/spec"; diff --strip-trailing-cr "$S/last/questions.md" "$S/next/questions.md"; } > "$S/next/changes.txt"
```

If `$S/last/` does not exist, there is nothing to compare: say so, offer the markitect-tour skill, and offer to remember this state as seen.

**What changed** is what `changes.txt` lists, and you cover all of it. Its first part names each item, `DEFINITION@FINGERPRINT | STATE`, that is new, gone, has another fingerprint or another state: its own text or a part's, its Kind, a Definition it links, a Rule that binds it or its files changed, or a reviewer recorded a verdict on it. The second names each file of the spec added, removed or changed, line endings aside, which the tool ignores too. The third is the questions. When the spec has errors, `status` lists no items: say so, and go by the files.

**Tell it** in the order of the markitect-tour skill's chapters, each statement with the Definition it comes from, as a link to its file:

- For each item whose own file or a part's was added, removed or changed: what is now true, in plain words; what it said before, from `diff -u --strip-trailing-cr` of its file in `$S/last/spec` and `$S/next/spec`; and why, which the project does not record yet, so say `why: not recorded` and do not guess one.
- The items whose own files did not change, in one list grouped by what they follow: a Definition it links or a Rule that binds it, among the changed ones, as `markitect doc --json DEFINITION` shows them, or what its line `changed:` in `next/status.txt` names. Where neither says, say that its files or its Kind changed and the copy does not show which.
- Each item whose state changed, with the state before and now, such as `no record yet` before and `ok` now.
- The questions added to `.markitect/questions.md` and those answered.

Draw a picture where it shows a change better than words, such as the states of a Concept before and after; draw only what the two versions of the spec state. End by checking what you told against `changes.txt` and naming each line of it you did not cover, or say that none was left.

**Remember it as seen** only when the user says they have seen it, and not while `next/status.txt` reports errors in the spec, or the next catch-up would count every item as new:

```bash
S="$HOME/.claude/markitect-seen/$(pwd -P | tr -c 'A-Za-z0-9' '-')"
rm -rf "${S:?}/old"; { [ ! -d "$S/last" ] || mv "$S/last" "$S/old"; } && if mv "$S/next" "$S/last"; then rm -rf "$S/old"; else mv "$S/old" "$S/last" 2>/dev/null; false; fi
```

If they stop before, remove `$S/next` the same way, with `S` named first, so that the next catch-up still starts from their last look. When another skill offers it, save the state as it is now, as under **Start**, and move it to `last/` the same way. Two sessions catching up on the same project folder at once share `next/`.
