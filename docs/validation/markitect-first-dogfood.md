# Markitect-first dogfood

Recorded 2026-10-04. This is a source workflow validation, not a productivity experiment or adopter acceptance. [Vision](../vision.md) owns the product thesis; [measurement](../measurement.md) owns future comparisons. [Release readiness](markitect-first-release-readiness.md) owns release acceptance and distribution proof.

## Fixed checkpoints and independent actor

| Checkpoint | Full commit | Evidence |
|---|---|---|
| Integrated baseline | `5cc9718c740185f61a92fbaf3f26cd8b7a17d031` | Before root self-adoption; published v0.12.0 still current at the start |
| Protocol and root Project | `3cb90535afb47f5a55e58a1c40f384fd9d3e04e8` | Generated entrypoints, helper, configured root checks; fixed actor preflight and Context passed |
| Desired intent | `4c0649375d12f96379fd58177150153ea0cdd33e` | Added the existing managed-artifact proof-limit document as an exact input of the development Rule; fixed check and impact reviewed before the technical regression |
| Implementation | `061f77fde2428b68380477d07c434c83d1191f57` | Fresh actor test change plus root Context/projection integration regression; fixed check, repeated Context, impact and configured verify passed |

The fresh actor received a normal task: strengthen the existing artifact-helper unknown-flag test to assert empty stdout. It started without this coordinator's conversation history, found the root agent pointer, read generated Skills and canonical Workflow, fixed BASE, compiled Context, classified the change as implementation-only, and edited only the Go test. It made no Rule, Domain or package change. Its focused test, direct accounting check and diff check passed. This demonstrates entrypoint discoverability and one bounded task execution, not continuous enforcement of agent behavior.

The actor manually inspected eight local files: the two generated Skills, `docs/markitect-first.md`, the canonical Markitect-first Workflow, the development Skill, the portable authoring Skill, the target Go test, and the accounting config. It also had the task's root AGENTS instructions. The compiled BASE Context contained eight inputs: root Project, two Rules, two Skills/Workflow entrypoints, CONTRIBUTING, architecture and engineering constitution. The accounting config and target test bytes were not included. The config's tooling-owner fact came from manual inspection, not compiled Context. No token, model-call, attention or comparative benefit metric is available.

At the implementation checkpoint, the same binary's repeated selected Context was byte-identical (SHA-256 `846495c1338d606652d60c2585b2369b6fe9023a80761d06ce2f9bf2d3635b28`), 77,992 bytes and nine inputs. The additional input was the managed-artifact design document. This is a dated exact-output observation, not a token estimate or proof of optimal context focus. The regression verifies inclusion reasons, exact ordinary-input hash, canonical Workflow inclusion, generated owner identities, no nested-example context inclusion, repeat determinism and output convergence.

## Impact and coverage

The intent-only comparison `3cb9053 -> 4c06493` affected exactly two resources: `development/Rule/repository-boundaries` and `development/Skill/engineering-change`. Causes identify the changed Rule and the Skill's declared `rules` relation in both snapshots. No configuration or inventory change was hidden behind this count.

The implementation comparison `4c06493 -> 061f77f` affected all ten root resources. The two Go test paths were unowned semantic inputs, so Core conservatively broadened impact. One was explicitly tooling-accounted by the separate helper. Tooling accounting is not a semantic dependency; the ten resources are conservative review impact, not ten independently established implementation changes. This noise remains visible.

An exact temporary file `.markitect/unmanaged-negative-control.txt` was created only after checking the path did not exist. Direct helper execution returned exit 1 with an `unmanaged` finding naming that path. Only that created file was removed; the subsequent report passed. Unit tests additionally cover missing/renamed inputs, missing/orphan generated outputs, collisions, aliases, exact exclusions, stale declarations and deterministic overlapping-root refusal. These prove configured path accounting, not source-code meaning, secret detection or arbitrary filesystem isolation.

At `061f77fde2428b68380477d07c434c83d1191f57`, fixed `check` passed and fixed `verify` passed both configured checks: the artifact helper and `go test ./...`. Verify materialized the reviewed snapshot without Git metadata. Generated outputs were produced by canonical owners and root output checks passed. No external target adapter was configured in this root Project; this result does not establish an external-provider lifecycle. Normal reconciliation tests and packaged fixtures separately exercise explicit Apply, saved-plan freshness and ownership boundaries.

## Friction and corrections

- The first fresh-session attempt found the protocol but could not finish Go preflight because the default local build cache was unavailable. It was not counted as a passed replay. A second fresh session used a writable cache under the already excluded `.cache` root.
- A cache had initially been placed at `.gocache`, which was admissible snapshot input and caused very large captures and slow runs. It was moved to `.cache`; no source-loader exclusion or digest contract was changed. Fixed-snapshot verify remained expensive (the recorded local run took roughly six minutes). Cache/environment setup is not product benefit.
- Existing nested runnable Projects' generated outputs were incorrectly treated as root stale outputs. The correction recognizes a nested Project only when its subtree is disjoint from parent Areas, expected outputs, explicit inputs and local Domains. It changes stale-output inventory only, not graph composition, snapshot acquisition or Core language. Child Projects still require their own checks.
- The embedded authoring Project had to move outside the root resource Area, while retaining its virtual embedded `markitect.yaml` identity and exact package allowlists.
- Independent review caught a nondeterministic overlapping-root error and an adopter Workflow recommending a Markitect-checkout-relative helper command. Both were corrected; adopters build the pinned standalone helper and configure its reviewed binary on PATH. Only root source dogfood uses local `go run`.
- Context routes to the accounting config but does not supply its bytes. The actor still needed manual config and implementation inspection. This missing content is retained as evidence; it is not fixed by broad automatic Context expansion.

## Restricted adopter track

The separately authorized private exercise captured exactly 36 owner-selected files into an isolated external workspace. Digests matched the approved handoff, selection and capture identities; all stored selected bytes were validated; source repository status stayed clean. Interpretation consumed only the supplied capture and produced four bounded candidates with support, qualifying evidence and uncertainty. The existing validator accepted their byte/reference structure. No decision record, canonical adoption or source change was made. Detailed evidence remains private; this public record contains only counts and tooling behavior.

The four candidates require actual owner dispositions. A recurring human coordination burden, real task, expected behavior and scope for implementation evidence are still needed before an adopter comparison can run. No synthetic task substitutes for that owner decision. No direct conflicting or legacy convention was established merely to populate a category. Prior negative MyMeetings and AGENTS.md comparison findings remain unchanged.

## Follow-up

The next experiment should use the published attested binary and exact source bundle. Freeze an owner-selected ordinary task and independent rubric, compare fresh isolated baseline and Markitect-first variants, and measure setup, review, intervention and recurring upkeep separately. The smaller guidance-plus-tests baseline may win. This exercise establishes feasibility and exposes remaining work; it does not establish less human supervision.