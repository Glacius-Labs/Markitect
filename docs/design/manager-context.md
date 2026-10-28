# Manager context: child areas, See also and file paths

**Redesign decided, not built:** [DEC-081](../concepts/register.md#dec-081-the-organization-knows-broad-at-the-top-and-deep-below) changes what each Manager receives. This page describes Main today.

Status: accepted by the coordinator on 26 October 2026 for implementation. It answers two of the W1 fitness review's context gaps, which the owner asked to fix ([W1 decisions](../concepts/sources/model-track-w1-decisions-20261026.md), [DEC-074](../concepts/register.md#dec-074-canonical-model-track-parallel-to-v1)). It stays within [DEC-027](../concepts/register.md#dec-027-strict-links-with-an-optional-optimizer) and [DEC-028](../concepts/register.md#dec-028-context-follows-need-to-know-and-starts-simple), and gives a partial answer to [OQ-006](../concepts/register.md#open-questions).

## Problem

1. A parent Manager reviews and integrates its children's handoffs, yet it got none of their Definitions. Context selection took only the Manager's own Package, so Root and empty Packages got an empty selection. The harness's hierarchy listed each Manager's Definition identities, but no Kind, title, body or file.
2. DEC-028 promises "the own Package's bodies, a See also section with linked Definitions, and files to read". The harness rendered one flat list of every selected Definition: own, linked and contract Definitions were mixed together, nothing formed a See also section, and the source path it showed was relative to the model's acquisition boundary, so it was not necessarily a path in the Manager's working copy.

## What each Manager gets

The selection rules do not change: strict authored links, one hop of outgoing references and links, local incoming ones, type contracts and modeled scope. The harness renders four sections:

| Section | Content | Form |
|---|---|---|
| Own Definitions | Definitions of the Manager's own Package | Full: original body, typed fields, provenance and reasons |
| See also | Definitions selected by outgoing, incoming or modeled-scope reasons | Full |
| Type contracts | Definitions selected only as Kind or Property contracts | Full |
| Child areas | For each direct child Package, every local Definition in that child's subtree | Compact: identity, Kind, title and file; no body, no fields |

- **One section per Definition.** A Definition with several reasons appears once, in the first matching section among the first three. A child Definition that the Manager also links to appears in full under See also and compactly in its child area, so each child area stays a complete index.
- **Empty children.** A child whose subtree has no Definitions appears with an explicit empty list.
- **Files.** Every entry names its repository file: the owning model's repository area joined with the Definition's source path. This is DEC-028's "files to read", and it sits on each entry instead of in a second list. Only Definitions of an external dependency, which has no repository area, carry no file; they are delivered in full when selected.
- **Labels.** Section labels are neutral data labels. The harness gets no prose on how to use the sections ([concept §3.7](../concepts/v1-concept.md#37-tasks-not-process-d13)).

Example with a model of Packages `a`, `a/child`, `b`, `c`, `empty` and `incoming`, where `a/own` links to `b/near`, `b/near` links to `c/far`, and `incoming/sender` links to `a/own`:

- **Root:** no own Definitions and no See also. Child areas: `a` lists `a/child/child` and `a/own`; `b` lists `b/near`; `c` lists `c/far`; `incoming` lists `incoming/sender`; `empty` lists nothing.
- **Parent `a`:** own `a/own`; See also `b/near` (outgoing) and `incoming/sender` (incoming); the Kind and Property contracts they use; child area `a/child` lists `a/child/child`. `c/far` stays out, because it is a second hop.
- **Leaf `a/child`:** own `a/child/child` and its contracts; no child areas.

## Why the whole subtree, in compact form

A child's handoff commit contains its descendants' integrated work, so a parent reviews the child's whole subtree; an index of direct children alone would hide that work. Full bodies of the subtree, on the other hand, would hand Root the whole model, against need to know (DEC-028). Compact entries with files follow OQ-006's hypothesis of current-state context on demand: each file is in the Manager's working copy.

## Size

- Own, See also and type contracts: same content as before, regrouped, plus one file path per entry.
- Child areas: one line of about 100–150 bytes per descendant local Definition. A Manager's addition is bounded by its subtree, Root's by the local model's Definition count, and the whole tree's by N × depth lines. For the self-model, Root gains about 40 lines.
- Nothing is truncated, and no new size limit is introduced ([concept §15.4–15.5](../concepts/v1-concept.md#154-context)).

## Implementation

- **`internal/organization`.** `ContextSelection.Children()` returns one `ContextChild` per direct child, in topology order. Each one holds `ContextSummary` values (identity, Kind, title, source) in source-path order, built from the local model's Definitions in that child's subtree. Summaries are not part of the selected set: they carry no reasons and pull in no type contracts. The zero selection returns nil, and the returned values are copies.
- **`internal/harness`.** Renders the four sections, and the harness version goes from 3 to 4. To find each Definition's repository area, it matches the Definition's source origin to the local or dependency model source.
- **`internal/adapters/cli`.** The `markitect context` report includes the children, so inspection shows what a Manager receives.

Out of scope, to start simple: registered artifacts as files to read, recorded targeted reads, and links proposed by an Optimizer.
