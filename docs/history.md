# History

Main describes only the current state ([DEC-071](decisions.md#dec-071-main-holds-only-the-current-state)). Everything earlier is in Git.

## Earlier concepts

Each earlier concept has an archive branch at the last Main commit where it was the target, with a note at the top of its README. Its code and documentation describe that concept.

| Branch | On Main | In short |
|---|---|---|
| `archive/concept-1.0` | 24 to 28 October 2026 | A hierarchy of agents per Package over a spec in typed YAML, each change run in isolated environments |
| `archive/concept-2.0` | 28 October to 4 November 2026 | The spec as typed Markdown; one run per change with a maker and a checker per Package; a desktop app |
| `archive/concept-3.0` | 4 to 6 November 2026 | One implementer and one reviewer per change in an isolated environment, and a gate in the tool |

The current concept has been on Main since 6 November 2026.

## Everything else

- Superseded decisions, plans, designs, owner statements and dated validation records are in Main's history. The documentation as it stood before it was cut to the current state is at [`d0c23a31`](https://github.com/Glacius-Labs/Markitect/tree/d0c23a31efc928e95f3acc0633deb01a5f887884/docs), with the full archive index in `docs/archive.md` there.
- To find a removed file, run `git log --diff-filter=D --format=%h -1 -- PATH` and read it at the parent of that commit.
- The history before 23 October 2026 was reset. Its commits, among them the source of the retired releases 0.1.0 to 0.14.1, are kept only by the `archive/pre-reset-*` refs.
- Links from Main into history always use a full commit SHA, never a branch name.
