# Policy-failure analysis replay

**Captured result for candidate `1549608438f9c0d24d29d43608515bce388c14e7`, not the final post-review result.** The MyMeetings replay completed successfully against this candidate. A separate impact-scope issue was subsequently found in review and is being corrected; the replay must be repeated with the corrected candidate before the implementation is considered final.

## Evidence and boundary

This replay asks one narrow question: can a reviewer inspect the exact structurally valid but policy-failing MyMeetings package update before fixing or waiving it? It does not measure productivity, code correctness, application runtime behavior, safety, or business benefit.

The adopter is the public MIT-licensed [modular-monolith-with-ddd](https://github.com/kgrzybek/modular-monolith-with-ddd) project, source commit `91c8ef24b4cb6ef558c95d8267fa07d68c7059f8`. The earlier pilot selected a faithful subset and retained its license notice at [upstream-LICENSE.MIT](../real-project-adoption/evidence/upstream-LICENSE.MIT). This replay reads the preserved [adopter snapshot bundle](../real-project-adoption/evidence/adopter-snapshots.bundle), SHA-256 `af9dbdc750f376b5250b61a451c1d124fe363fd87cce87684804b292149311cc`. It does not modify anything under `experiments/real-project-adoption/`.

The candidate executable is `Markitect 0.12.0` for Windows/amd64, SHA-256 `229687b9419396d9af659f8c2721d70d334cd43d801f64db0f9be1e0ec5ea44d`. Go build metadata in [candidate-buildinfo.txt](results/20261003T054342Z/candidate-buildinfo.txt), SHA-256 `a86596e1903a5d74e1ebada7493de3ba6ce8d1257e8980ae20a0199307aad9fe`, reports `vcs.revision=1549608438f9c0d24d29d43608515bce388c14e7` and `vcs.modified=false`. The script verifies those fields against the supplied source commit. In the presentation copy, the executable path is normalized to `<candidate-executable>`, trailing spaces/tabs are removed from each line, and exactly one final newline is used; metadata values remain unchanged. The version string reflects embedded version metadata; this is a locally built candidate, not a published release artifact.

For provenance, v1 has snapshot digest `594d527fe69074a9fd7357983ce7c7a8d50307812e638a517c939382673de8ef`, config digest `sha256:9034f3767d0773047d2239ae3a38cbd86f3c804cdc5d2a4d20e77e90b68976e9`, and model digest `sha256:d115aeef1780a8bdf9aaec2dca2d67c696b4892740c7e59830d6dcd035c89e94`. Failing v2 has snapshot digest `dda2001e4249f458414f3b260e72636a188221424e69ad111a645ff8e62f5bf4`, config digest `sha256:3726cae54f94af37c56cf9d6316d27091ddb7919cdf2e262814190fad3cfd0a4`, and model digest `sha256:9e03161abf413beacf9ddc47fa92033032bc2a93990301b6f4cb7ddb8ee6beb7`. The diagnostic Context digest is `sha256:4879b01b4fa01324aeaa7b507e4dc6a87060efa4fd89389b74049662597dc151`; its snapshot digest, tool digest, and candidate model digest match the v2 identities above. The Impact analysis repeats the same base and candidate model/configuration identities, showing that the policy delta and context refer to the same normalized snapshots.

The replay verifies these exact package archives in each historical snapshot: v1 SHA-256 `0c7395055aac7f4c35afcff404481a77e2ce6475c863a435afc2586dd388b1cc`, v2 SHA-256 `0ed97c9f0b21e10b5e2a0ff232bdb910bd468ac04f84441ae24d5b7cefb2b504`.

## Observed lifecycle

The replay exercised v1 `fd94a689aab857704c3a4a45ac2abe0fc0e3185c`, failing v2 pin `a0f89e7c0ce43a1be556c17c41cd19f8e7f32d7f`, first-Validator revision `3e138c146736dad1ddb11dc7c509a61d95ab4398`, explicit-waiver revision `65c1d959e6f5d1b866effbc5188c353c82b2405d`, and final revision `c507172b24da9005904422c03cc3d66a2b5efcec`.

At v1, the normalized model reported structural and policy status `passed`; strict `check` exited 0. At the v2 pin, the normalized model reported `structuralStatus: passed`, `policyStatus: failed`, and two failed PolicyResults. Strict `check`, default Context, and default Impact each exited 1. Context and Impact with `--analyze-policy-failures` also exited 1, while producing completed diagnostic YAML. The exit code therefore continued to signal that the candidate fails policy.

The two failed results are `selected-commands-require-validator` for `UseCase/add-meeting-attendee` and `UseCase/cancel-meeting`. The v1 Domain did not define this constraint, so the direct delta is `not-applicable` (`constraint-not-defined`) to `failed`; the separate `selected-validation-cohort-is-command` results changed from not applicable to passed. The v2 validator rule is defined by Domain `mymeetings-architecture-v2`, exact package version 2.0.0, archive SHA above, package source `git:e0ae3c5181323e0d2a03312524010cc817652330`, and constraint digest `sha256:8d4e312bdb9a2f5569371b9a6be101ee76a9ea0d4007b0087e441031d248f2a8`. Its selector is UseCase resources labeled `validation: required`; the assertion requires at least one `hasValidator` relation. The base Domain was `mymeetings-architecture-v1`, exact package version 1.0.0 and archive SHA above.

Diagnostic Context for `UseCase/add-meeting-attendee` completed with 17 inputs and four in-scope PolicyResults: one failed Validator requirement and three passing results. Its local closure includes the UseCase, Feature, Handler, Module, linked architecture narratives, package Domain, and declared project files. It includes the matching failed PolicyResult with message, constraint and subject digests, plus the constraint definition and exact package/Domain source. Existing `via` entries identify inclusion through `usesFeature`, `hasHandler`, `belongsToModule`, and `documents`. The candidate header marks the overall policy state failed and states this is not verification or acceptance. The other failing Command is not pulled into this selected Context closure.

Impact from v1 to the failing v2 candidate completed and reported:

| Measure | Observed |
|---|---:|
| Changed input paths | 1 (`markitect.yaml`) |
| Policy changes | 4 result records: two newly failed Validator requirements and two newly passed Command guards |
| Direct policy subjects | 2 (`add-meeting-attendee`, `cancel-meeting`) |
| Conservative affected GraphKeys | 69 |
| Affected UseCases | 12 |
| UseCases outside the selected Validator cohort | 10 |

The conservative set includes `Project`, all modeled architecture resources, the package resource, and declared Skills/Workflows. Its reported causes are a project `configuration` change at `markitect.yaml` and the corresponding `resource-change`; the impact reason says configuration changes can conservatively affect all resources. It therefore makes the broad review set visible without presenting all 69 resources as direct Validator implementation targets. The ten affected UseCases outside the policy cohort are `accept-proposal`, `authenticate`, `create-meeting`, `create-price-list-item`, `get-meeting-attendees`, `get-meeting-fees`, `get-meeting-group-details`, `get-member`, `propose-meeting-group`, and `register-new-user`.

No exception was added for the failing v2 state: the v1-to-v2 changed-path set contains no exception path, and the failing model has no waived PolicyResult. Adding the first Validator leaves one failed result for `cancel-meeting`; diagnostic Context for that subject completed and exited 1. The historical explicit waiver yields policy status `waived`; model, strict Context, diagnostic Context and v1-to-waiver diagnostic Impact exited 0. At the final revision, the model, strict `check`, strict Context and strict Impact passed; diagnostic Context and Impact also exited 0.

## Read-only and strictness evidence

The successful run record is [run.json](results/20261003T054342Z/run.json). It records stdout/stderr/exit files for 20 invocations. All commands matched their expected exit code. Recorded command arguments replace only the exact temporary checkout path with `<replay-checkout>`; the subprocesses ran with the actual checkout path. The executable path on the build-info first line is similarly represented as `<candidate-executable>`; trailing whitespace in this presentation copy is normalized and the metadata values are preserved. These presentation-only normalizations do not alter the captured command stdout/stderr or their hashes. For each invocation the disposable checkout’s Git tree, clean status, file count and recursive regular-file fingerprint were unchanged before and after; `.git` metadata was excluded from the recursive fingerprint. The failing-v2 checkout had 794 regular files. This checks ignored files and projections as well as tracked files. The script performed only `model`, `check`, `context`, `impact`, revision checkout and read-only Git inspection. It did not run render, reconciliation, Apply, or any application-code command.

This replay demonstrates that the exact pilot migration can be inspected without treating failed PolicyResults as passing and without using an exception as an analysis key. It confirms strict command exits and read-only output behavior for this fixture. It does not itself test structurally malformed models; those remain a separate test obligation. The unrelated Impact resource-to-collection comparison found in review also remains outside this MyMeetings replay and must be fixed and revalidated separately.

## Captures

The timestamped `results/20261003T054342Z/` directory retains each raw stdout, stderr, and exit code. Key captures are `v2-model.stdout.txt`, `v2-check-strict.stdout.txt`, `v2-context-strict.stdout.txt`, `v1-to-v2-impact-strict.stdout.txt`, `v2-context-analysis.stdout.txt`, `v1-to-v2-impact-analysis.stdout.txt`, `first-validator-model.stdout.txt`, `waiver-model.stdout.txt`, `final-check-strict.stdout.txt`, and the final Context/Impact outputs. The separate replay tool is [replay.ps1](replay.ps1).
