# Authoring

The owner's changes to the project's `.markitect` inputs made through the [cockpit](../adapters/cockpit/README.md) are proposals ([DEC-065](../../docs/concepts/register.md#dec-065-work-through-the-cockpit-agent-proposed-model-changes-and-configuration), [concept](../../docs/design/cockpit.md#working-through-the-cockpit)). This package makes, checks and decides them; bootstrap composes it with the repository's capture and publisher.

## One proposal path

1. **Capture.** A proposal starts from the committed source, captured as Apply captures it: the working tree must equal its head, and the capture carries the guard a publication is checked against. A dirty working tree refuses with `ErrNotClean`. A proposal made from the head the owner read reuses that capture for 15 minutes; reuse is safe because acceptance checks the guard, and a started publication drops the capture.
2. **Change.** A producer builds the proposed tree. The configuration producer edits `.markitect/execution.yaml`; a proposal may change only the paths its producer permits.
3. **Check.** `apply.CheckProposal` prepares the proposed tree as `markitect check` prepares the repository: Model check, registry, File check, execution settings and context selection. The proposal keeps the structural status and every finding with its stage, code and location.
4. **Review.** The proposal lists each changed file with its line difference (three lines of context, longer unchanged runs cut).
5. **Decide.** Discarding writes nothing. Accepting needs a passing check: bootstrap's fence takes the lock of every observed state directory, then Apply's publisher preflights the guard, writes exactly the changes and reads every path back. A source that changed since the capture refuses the publication; only an applied (or no-change) receipt accepts the proposal. A fence that cannot be taken leaves the proposal open; every other unapplied outcome fails it with its receipt. Nothing is committed and no Apply starts.

Proposals live in the cockpit process; what was accepted is in the working tree, ready to be committed.

## Execution settings

`Settings` reads the budget, role profiles, per-Package profile overrides and team of the committed `.markitect/execution.yaml`, with the model's local Packages. `ProposeSettings` changes exactly those fields in the document's YAML node tree, so comments, order and every other field (ownership, top-level and per-Package instructions, scope, checks) stay as they are: scalars change in place, a Package entry left empty is removed, the team is written anew, and a team of `nil` removes the authored team, which restores the product's default team. The product's strict decoding judges the result in the check.
