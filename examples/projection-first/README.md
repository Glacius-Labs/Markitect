# Projection-first Dispatch fixture

This greenfield consumer example gives Markitect an unfamiliar vocabulary:
a DispatchBoard contains Listings, each Listing has one Board and exactly one
Owner, and a ListingView selects and orders the rows shown to a person. The
Domain declares those references and singleton bounds. Its board capacity
policy selects one through five explicit listings, while the ListingView has
exactly one Board and one DispatchProcess context dependency.

The canonical ListingView is a typed resource stored as JSON syntax in a
.yaml file. JSON is a YAML subset accepted by the repository's YAML decoder,
and it lets the independent checker read the same canonical bytes using only
Python's standard library. The finite contract selects status Open, sorts by
openedAt then id in ascending order, preserves each complete row, and writes
one JSON value to stdout. The checker derives expected rows from that resource
and the fixed cases in fixtures/listings.json; it also checks the literal
commands from the typed DispatchProcess resource in CI and the pre-commit
hook.

tools/check_contract.py and its focused checker tests are independently
owned verification tooling. AI materializers own the declared
implementation, implementation tests, CI workflow, and hook targets. The Rule
and Skill remain canonical typed Markitect resources. The existing registered
Markdown and Agent Rules renderers own their generated documentation and
provider views; those paths are listed separately in projections.config.
No AI identity is authorization.

## Finite checks

From this directory, after a candidate implementation and its process files
exist:

- python -m unittest discover -s tools -p test_contract_checker.py checks
  the independent checker helpers.
- python -m unittest discover -s tests runs implementation tests owned by
  the materializer.
- python tools/check_contract.py exercises the finite JSON examples against
  scripts/list_open_listings.py, verifies complete-row preservation and the
  declared CI/hook command literals, and runs the implementation in a fresh
  temporary working directory.

The checker is a bounded fixture, not a general semantic analyzer. It checks
only the named JSON cases, configured status/order/output fields, expected
command text, candidate exit/stdout, bound-file hashes around the full run,
and writes in the temporary current directory. It does not prove that
arbitrary code has no other filesystem side effects, establish runtime
behavior for unlisted inputs, verify GitHub Actions or shell semantics, or
establish human acceptance. Both candidate variants are checked independently
against this same unchanged intent and checker; passing does not support a
universal behavior or productivity claim.

Renderer output paths are exact contract targets, not independent AI-owned
files. With the current Markitect CLI, run this from the example directory
to inspect both contracts:

```text
markitect projection --repo . --action plan --config projections.config --coverage markitect-artifacts.yaml
```

Before the AI-owned targets and
fixed-snapshot check evidence exist, the plan is expected to be incomplete;
the deterministic renderer contract can still be converged. Planning does not
apply candidate files or establish verification of the AI materialization.


The checked-in implementation retains the reviewed variant A. [Replay evidence](../../experiments/projection-first/README.md) preserves both candidates, fixed inputs/digests, negative findings and bounded drift/evolution results. `TestProjectionGreenfieldCandidatesConvergeAgainstIndependentEvidence` removes all 20 targets from a fresh copy and replays both exact candidates through Plan, Apply and immutable Verify without model calls. Named CI gates exercise this proof and verify the fixed checked-in example.

The fixture deliberately keeps its custom resource files as JSON-subset YAML so the independent standard-library checker can read them. Markitect's YAML formatter may change their textual syntax; do not use `format --write` as a repair for this projection. That is an explicit authoring/checker integration cost, not new canonical semantics or source analysis in Core.
