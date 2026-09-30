# Markitect production delivery assessment

**Recorded:** 2026-09-30; **Status:** Markitect source release `v0.1.0` is published and verified as immutable. This report records source, bundle and bounded consumer-operation evidence; it does not grant overall consumer, provider-runtime or human acceptance.

## Result

The independent Go/YAML source repository now has a durable private release and a repeatable owner-authenticated publication and provisioning path. The release is bound to merge commit [`45e0017a3d25d2abd464de934b5ef3a802732d0a`](https://github.com/Glacius-Labs/Markitect/commit/45e0017a3d25d2abd464de934b5ef3a802732d0a), whose tree matches the independently reviewed candidate [`4adb45964a172a77a1355c6ea5ac2596c325b90d`](https://github.com/Glacius-Labs/Markitect/commit/4adb45964a172a77a1355c6ea5ac2596c325b90d). The source PR gate, main gate, release workflow, and immutable-release attestations all completed successfully for the intended version and source.

The supported boundary is Windows amd64 and Linux amd64. Building the source and running the consumer bootstrap requires Go 1.27.1 or newer; the published native CLI can preview or apply an install without Go. The CLI handles local validation, graph queries, context and impact compilation, formatting, rendering, migration, verification, bundling and installation. It does not certify arbitrary prose, provider behavior, an agent's runtime conduct or a consumer's delivery decision.

## Source and release evidence

| Evidence | Result |
|---|---|
| Reviewed candidate | `4adb45964a172a77a1355c6ea5ac2596c325b90d`; the exact reviewed tree was merged as `45e0017a3d25d2abd464de934b5ef3a802732d0a`. |
| PR hosted CI | [Run 36762498178](https://github.com/Glacius-Labs/Markitect/actions/runs/36762498178) passed on Windows and Linux. |
| Main CI | [Run 36762893903](https://github.com/Glacius-Labs/Markitect/actions/runs/36762893903) passed for the merge commit. |
| Release workflow | [Run 36762948855](https://github.com/Glacius-Labs/Markitect/actions/runs/36762948855) passed source quality and bundle-install/bootstrap checks on both platforms. |
| Immutable release | [`v0.1.0`](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.1.0), release ID `400355906`, published 2026-09-30 19:08:33 UTC; the remote tag resolves to the merge commit above. The release attestation and all four asset attestations verified. |

The release workflow produces a 30-day run artifact and does not publish a GitHub Release. Its `GITHUB_TOKEN` cannot read the admin-only immutable-release setting; the live API returned HTTP 403. The authorized owner therefore published locally through the Go release tool and existing `gh auth` session. No personal access token is stored in workflow secrets. The tool's default mode validates the exact run and artifact and prints a read-only plan; explicit `--publish` repeats validation, creates and fills the draft, checks asset names and GitHub SHA-256 values, publishes, then verifies the immutable release and each asset.

The outer bundle digest is `9bb4d2badedf806faa9bf2150a0bcd78551b1f69238d53e7b58f49282a4fd6b6`. It contains the complete five-file consumer pin, including the source archive whose digest is `c9bc71b0939982e58d2edfe14e18fb5221c0d947dcac4e672e90f2ae612a2628`.

| Published asset | SHA-256 |
|---|---|
| `markitect-v0.1.0-bundle.zip` | `9bb4d2badedf806faa9bf2150a0bcd78551b1f69238d53e7b58f49282a4fd6b6` |
| `markitect-v0.1.0-linux-amd64` | `2c338b3601efe54026d5266e592f69cb3e677bfba051f21182d837229a98581f` |
| `markitect-v0.1.0-windows-amd64.exe` | `a6dab3a59c83c29a0f1317982516177be848055b024e07455cafe025847f3431` |
| `markitect-v0.1.0-provenance.yaml` | `41af323fef7b1c1040c25541eb989616724446c5c3222e21fc00046bf2d1cb6a` |

The provenance YAML records source tag/commit, workflow run and attempt, toolchain and payload digests. It is a self-declared record; GitHub's immutable-release attestation is the authoritative binding of the published tag, commit and assets. The automatically generated GitHub source archive is not the Markitect bundle.

## Implemented product boundary

Markitect is a local compiler and authoring tool. Canonical YAML describes six content kinds—Text, Rule, Workflow, Skill, Agent and Contract—plus a Project. Generated YAML schemas assist editors; strict parsing and graph validation remain authoritative. These resources are not Kubernetes CRDs.

The Project defines area paths, imports, scoped rules and bindings. Paths determine resource ownership, with the longest matching area owning a resource. Imports control direct reference access. Resource `rules` expresses requirements; `uses` declares concrete dependencies; `needs` names a Contract; `implements` declares a signature promise; a Project binding selects an implementation. Agent/provider properties are validated against the supported adapter fields. Resource `files` declares exact ordinary UTF-8 inputs; ordinary YAML is treated as file content only through that declaration. Malformed YAML and recognizable but invalid Markitect resources remain errors. Navigation links in prose are not silently converted into dependencies.

The graph and application API support structural discovery/explanation, fixed-revision context compilation, old/new impact analysis, managed output checks and rendering, profile verification, advisory review-evidence eligibility, and controlled file writes. Context requires a selected entry. Unknown or unmodeled changes invalidate conservatively. Migration exposes dependency candidates for review; it does not infer normative dependencies from prose. The portable core authoring bundle is embedded and tested through the same application path; the standalone example exercises all resource kinds, a Contract binding and an ordinary file input.

The format, render, migration and install writers share a writer lock, revalidate their inputs and use atomic per-file replacement. Multi-file operations are not repository transactions. Install validates the exact release bundle before writing its five pins and refuses partial or ambiguous ownership. Git remains the integration and rollback boundary.

## Provisioning and consumer checks

After publication, the bundle, Windows binary and provenance were downloaded again from the release and verified. The installer then completed two bounded operations against the same bundle and source commit:

- Fresh Cockpit consumer: created all five pins (`markitect.lock.yaml`, paired bootstrap test, bootstrap runner, source archive and `release.yaml`); the plan and applied bytes agreed with the release.
- Konfyra consumer: upgraded the recognized complete RC3 legacy pin set to the same five-file release pin; the plan marked unchanged/replaced/created paths, and the applied set matched the verified bundle.

The initial Cockpit migration at `e112a63bd5562ad1ead4d7562d597ec456dcb33b` passed fixed verification in 6.356 seconds. Its Go documentation checker reported eight tests and its bootstrap suite twelve top-level tests. A real authoring exercise refined the existing workflow and produced functional candidate `fb552254f1b0ddf295816cd29f71bd966a643fd4`: fixed Windows verification passed in 4.792 seconds, context in 554 ms and impact in 740 ms. The pinned Windows tool digest was `c282341230252484e45bf569039c165fe75a150c8648bd01c17662b4205ea088`.

The same `fb552254...` commit was exported as a local Git bundle and verified in a fresh Linux amd64 container in 15.094 seconds, including the outer bootstrap invocation. The already present Go 1.27.1 image had ID `sha256:69a7b9788769bec032d238959b61854e9ae87f57be9029ec04e9885fabf99195`; the container used a read-only bundle mount, temporary checkout, read-only root and bounded resources. All three gates passed. This is local cross-platform evidence, not a hosted Cockpit run. The Cockpit has a prepared two-platform CI workflow but no remote repository.

The Konfyra pin rollback exercise preserved the original exercise branch history and did not push. From upgrade commit `30fd2ce931b774555a97de63268617a43fc418de`, a normal revert to `e0fc27b3b50c36b83e71c1e69dbfbcde87a54604` restored all four RC3 pins and removed the release manifest; bootstrap `version` passed in 4.270 seconds and fixed `verify` in 19.104 seconds. Reverting that rollback produced `b926f020a4e85134f1191d98218c84b9b9a0dc01`, with all five release pins matching the upgrade; `version` passed in 6.527 seconds and fixed `verify` in 20.039 seconds. A native `v0.1.0` install preview on the complete pin set was a no-op with all five files unchanged (358 ms). No gate failed in this exercise.

The review also checked scope behavior using counts only. General context contained five resources (Project, two General Rules, Skill and Workflow) and no customer resource bodies. A selected customer context contained thirteen resources (Project, eight local Rules, two General Rules, Skill and Workflow) and no sibling customer resource bodies. The full Project area/rule topology remains in the manifest; this is scoped context selection, not multitenant security redaction. A temporary combined-gate probe confirmed separate ownership: Markitect rejects unknown marked stale output, while the Cockpit checker rejects unknown unmarked provider files. Neither gate alone claims to detect both classes.

## Bounded performance evidence

The benchmark ran at source candidate `4adb45964a172a77a1355c6ea5ac2596c325b90d`, using Go 1.27.1 on Windows amd64 and an AMD Ryzen 7 7700X. Other task work ran concurrently; this was not a controlled hardware comparison. It used a deterministic synthetic multi-area graph with scoped dependencies and customer isolation. Fixture setup and correctness-oracle checks were outside the timed loops. It measured context compilation and impact calculation only; it did not include Git reads, parsing, rendering, model inference or an end-to-end authoring task.

| Operation | Three `ns/op` samples; median | Reported output | Allocation samples |
|---|---:|---|---|
| Context | 161,764; 162,725; 176,799; median **162,725 ns** (0.163 ms) | 7,537 context bytes; 10 entries | 38,441; 38,440; 38,441 B/op; 468 allocs/op each |
| Impact | 1,090,619; 1,042,038; 1,032,927; median **1,042,038 ns** (1.042 ms) | 481 affected entries | 929,450; 929,449; 929,448 B/op; 7,795 allocs/op each |

These are measurements of the named Go operations on the recorded Windows amd64 machine, not user-visible latency guarantees. No model token savings, semantic quality improvement or broad productivity gain is inferred from them.

## Consumer integration and acceptance boundary

Konfyra's final source candidate is `e7f0f7358e7242c146eab965195b9de5a6e3773a`. Fixed verification passed in 19.903 seconds with snapshot digest `f8ce681d6ddc99a9ef95798d401839d90e5a1cfbacd728fca6ecfcbb1fdf0eb3`. It retains the consumer's provider renderer and existing Python regressions and adds the native Go bootstrap suite. The separately scoped WorkSync gate had passed at ancestor `77ee0e57...` in 81.881 seconds; that is not relabelled as a final-candidate rerun.

Independent bounded authoring and migration reviews found no material issue at the final source, with integrated/reviewed target `c473d18b008a0ba1728861e8f0da5f3b0feb1c21`. The authoring report uses only its compiled context and fixed configuration; its context digest is `sha256:884903fc1b8d82cf107db5170b37300f65970ba428d3edd64fe8f5f1452a86b1`. Recording the actual report took 1.431 seconds; checking reuse on the same candidate returned `reusable` in 1.427 seconds. No model call occurs during that CLI eligibility decision.

Hosted Governance run 26375 succeeded on that exact source. Required PR run 26376 succeeded on merge preview `452225f2ce126593a1ae0cb2aaeeddec05d7c1e2`, combining the final source with later target `745b2f9f5fde0657bde4aae7dc4669b16bb24134`; the policy readback was `approved`. The later target is covered by that hosted merge gate, not retroactively by the earlier bounded migration review. Consumer PR 3007 remains Draft and unmerged. Provider runtime and human acceptance retain their own route. A separate Survey Story still uses its earlier RC2 pin and was preserved rather than silently upgraded.

The consumer's parallel process explicitly removed the unpublished general-migration dossier pending an explicit publish decision. This delivery preserves that decision: older records remain in Git history, actual new reports and API readbacks remain in the consumer's local artifacts, and the PR records the current technical result. No new Board item, restored dossier or unrelated sync recovery is inferred from tool readiness.

Cockpit's migration contains 17 canonical content resources plus Project and ten generated provider files. Its Go checker replaces the legacy Python tooling. The initial independent scope/parity review and a later context-only semantic review assessed the actual authoring workflow; the latter is bounded to the compiled Project, two Rules, Workflow, declared tooling file and fixed question. Additional inspections outside that input set cannot be folded into a reusable review record. The consumer's own delivery report records practical changes and final integration separately from the source release.

The first Cockpit report had also inspected undeclared Skill/view inputs. That report and its probe outputs were preserved but excluded as valid reuse evidence. A new actual context-only review was completed, registered with the same pinned tool (632 ms), and evaluated against the same fixed exercise commits:

| Change from `fb552254...` | Candidate | Result | Wall time |
|---|---|---|---:|
| None | `fb552254f1b0ddf295816cd29f71bd966a643fd4` | `reusable`, exit 0 | 695 ms |
| Known independent project Skill prose | `b6bcc11c0fc15858f92d87cde06fe88e747bcde8` | General context unchanged; only that Skill affected; `reusable`, exit 0 | 710 ms |
| Inherited General Rule prose | `5d31b4c70aa15bbac9b5ee4963cfa5dfdedc6a89` | Context and inherited impact changed; `review-required`, exit 1 | 779 ms |
| New unmodeled file | `375ce1c565df62be6d5b225d163b44214ff4c521` | Context unchanged but inventory conservatively affected; `review-required`, exit 1 | 940 ms |

The isolated exercise branches were preserved locally and the clean migration branch restored. No review report or digest was edited to obtain reuse. The CLI cannot detect material a reviewer reads outside its declared inputs; keeping a bounded question and honest input attribution remains an authoring/review responsibility. These probes establish eligibility behavior, not measured token or monetary savings.

Cockpit documentation commit `c493d079a25fdc1c7ea56dfa9618172add6ccf6a` adds the dated delivery report and README routes to the functional candidate without changing canonical resources or tooling. Its own fixed Windows verification passed in 5.574 seconds. Final delivery commit `b22f83e084a9bd2cc7f8192e41c6e3115a5d07e0` also reconciles the release/dossier pointers and passed fixed verification in 5.048 seconds, snapshot `fc05d584feebb5084a006c2abd4654e7cd4f7a030b6182d13bdc8c63b565b89a`. The earlier Linux result applies to `fb552254...`; it is not relabelled as a run on these documentation commits.

The Windows exercise shell reports Go 1.24.1 for its outer launcher. The bootstrap selects the pinned module's Go 1.27.1 toolchain automatically: the source `go.mod`, verified build stamp and `go version -m` on the actual `c282341...` executable all confirm Go 1.27.1. The launcher version must not be mistaken for the compiler version used to build Markitect.

No customer policy content is included in this source report. Provider runtime/authentication behavior and human acceptance are not claimed; Claude CLI authentication remained unavailable during delivery. GitHub server-side branch rules could not be configured under the current private-repository plan (API response 403); CLI checks for protected branch names do not replace server enforcement. The API group remains the placeholder `markitect.example.org/v1alpha1`, and public distribution-license/API-domain choices remain open. Content packages, template initialization, Kubernetes operators and general plugin frameworks are not implemented.

## Operational follow-up

Use the [consumer integration procedure](../integration/README.md) for release verification, the owner-publish flow, installation, migration and recovery. The older [RC3 source assessment](release-assessment.md) remains historical evidence for its earlier candidate; this report supersedes it for `v0.1.0` release identity and distribution evidence. Consumer acceptance remains owned and recorded by each consumer.
