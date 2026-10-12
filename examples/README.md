# Examples and executable fixtures

Start with the model-first [Shop example](project-world/README.md) for the current project workflow. Other directories exercise particular structural, authoring or experimental contracts. Run each fixture with the source or release named by its guide; a historical fixture is not a promise that every old command remains in current source.

## Model-first project workflow

| Example | Purpose |
|---|---|
| [Shop](project-world/README.md) | Recursive Managers, model intent, artifact ownership and deterministic application checks |

The [Project workflow](../docs/project-workflow.md) and [Project operations](../docs/project-operations.md) describe normal usage. Live provider exercises and their dated evidence are indexed under [experiments](../experiments/README.md).

The [scripted method harness](../src/harness/examples/method_exchange_e2e_test.go) builds the real `markitect` and `markitect-exchange-executor` binaries. Disposable projects use private exchange directories and recorded requests, literal file expectations, independent review of changed bytes and declared Git checks. Run it with `go test ./src/harness/examples -run '^TestMethodExchangeCLIEndToEnd$' -count=1 -timeout=10m`. It exercises plan, recursive work, review, integration, verification, guarded Apply, an out-of-scope write, a forgotten file replacement repaired after a failed check, and an integration defect corrected after review. A separate omitted-file scenario proves that a wholly absent required artifact fails structurally before process checks and is not eligible for explicit repair or Apply; the completeness check detects the reparable forgotten replacement.

The Windows/Linux hard-kill scenario stops the real CLI while its process executor is waiting for a response. `resume` must reclaim the dead writer lock and durably block the uncertain invocation without issuing another request. This proves the existing no-replay safety boundary; successful continuation after an in-flight process kill remains unproven. Scripted responses establish local method mechanics, not provider quality or adopting-project acceptance.

For a retained local evidence run, set `MARKITECT_METHOD_EVIDENCE_DIR` to a fresh absolute directory outside the source repository and capture the test's complete `-json` output there. The harness retains CLI child arguments/stdout/stderr, raw exchange requests, literal expectations and durable run state before temporary fixture cleanup. Each scenario refuses an existing evidence directory, preserving earlier attempts.

## Structural resources, navigation and packages

| Fixture | Purpose |
|---|---|
| [Minimal rollback review](minimal/README.md) | Small synthetic Project with generated Markdown |
| [Repository layout](repository-layout/README.md) | Canonical Areas, human docs and native guidance with Markdown views disabled |
| [Markdown navigation](markdown-navigation/README.md) | Links, images, references and code examples in relocated generated views |
| [Code and documentation](documentation/README.md) | Ordinary Go input connected to a resource and review workflow |
| [Documentation placement](documentation-placement/README.md) | Canonical document ownership and optional router validation |
| [Content package](content-package/README.md) | Versioned exported resources |
| [Package consumer](package-consumer/README.md) | Exact package pins and explicit export composition |
| [Canonical engineering](canonical-engineering/README.md) | Project-owned Software/Delivery vocabularies, policy, context and rendering |
| [Engineering constitution](engineering-constitution/README.md) | Executable reusable engineering constraints |
| [Engineering discovery](engineering-discovery/README.md) | Selected evidence and review before canonical adoption |
| [Software architecture](software-architecture/README.md) | Structural architecture and policy stress cases |
| [Delivery target equality](delivery-target-equality/README.md) | Bounded same-target constraint examples |
| [Published onboarding exercise](onboarding/delivery-service/README.md) | Fixed synthetic parcel-support task and dated measurement |

## Experimental and historical projection scenarios

These fixtures preserve their own contract and source boundaries. The canonical Projection alpha fixtures were removed with the alpha; the [v0.14.1 tag](https://github.com/Glacius-Labs/Markitect/tree/v0.14.1/examples) retains the ones it shipped. Consult the [design map](../docs/design/README.md) and [validation map](../docs/validation/README.md) before interpreting a recorded result as current readiness.

| Fixture | Purpose |
|---|---|
| [Projection-first](projection-first/README.md) | Reviewed candidate and verification lifecycle |
| [Selective adoption](selective-adoption/README.md) | Fixed-source capture and project-owned adoption |
| [Consistency conflict](consistency-conflict/README.md) | Bounded conflicting-intent scenario |

Generated fixture views derive from each fixture's canonical inputs. Preserve expected baselines and immutable workload versions; use a disposable copy for a scenario that writes. The [example harnesses](../src/harness/examples) and [contribution checks](../CONTRIBUTING.md#verify-a-change) own source regression coverage. [Release benchmarks](../benchmark/README.md) measure versioned synthetic CLI workloads separately.
