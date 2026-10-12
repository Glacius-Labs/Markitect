# Examples and executable fixtures

Start with the model-first [Shop example](project-world/README.md) for the current project workflow. The current executable fixtures use the recursive Manager model. Historical Project/Domain fixtures remain available at their original source or release.

## Model-first project workflow

| Example | Purpose |
|---|---|
| [Shop](project-world/README.md) | Recursive Managers, model intent, artifact ownership and deterministic application checks |

The [Project workflow](../docs/project-workflow.md) and [Project operations](../docs/project-operations.md) describe normal usage. Live provider exercises and their dated evidence are indexed under [experiments](../experiments/README.md).

The [scripted method harness](../src/harness/examples/method_exchange_e2e_test.go) builds the real `markitect` and `markitect-exchange-executor` binaries. Disposable projects use private exchange directories and recorded requests, literal file expectations, independent review of changed bytes and declared Git checks. Run it with `go test ./src/harness/examples -run '^TestMethodExchangeCLIEndToEnd$' -count=1 -timeout=10m`. It exercises plan, recursive work, review, integration, verification, guarded Apply, an out-of-scope write, a forgotten file replacement repaired after a failed check, and an integration defect corrected after review. A separate omitted-file scenario proves that a wholly absent required artifact fails structurally before process checks and is not eligible for explicit repair or Apply; the completeness check detects the reparable forgotten replacement.

The Windows/Linux hard-kill scenario stops the real CLI while its process executor is waiting for a response. `resume` must reclaim the dead writer lock and durably block the uncertain invocation without issuing another request. This proves the existing no-replay safety boundary; successful continuation after an in-flight process kill remains unproven. Scripted responses establish local method mechanics, not provider quality or adopting-project acceptance.

For a retained local evidence run, set `MARKITECT_METHOD_EVIDENCE_DIR` to a fresh absolute directory outside the source repository and capture the test's complete `-json` output there. The harness retains CLI child arguments/stdout/stderr, raw exchange requests, literal expectations and durable run state before temporary fixture cleanup. Each scenario refuses an existing evidence directory, preserving earlier attempts.

## Historical Project/Domain fixtures

The earlier structural, navigation, content-package, architecture, projection and selective-adoption examples were retired with the Project/Domain implementation. The [v0.14.1 tag](https://github.com/Glacius-Labs/Markitect/tree/v0.14.1/examples) retains the published fixtures and their original instructions. Use their named historical binary rather than current source.

The retained [release benchmark fixtures](../benchmark/README.md) measure versioned synthetic CLI workloads through historical published binaries. Their input versions and raw records remain unchanged. The [design map](../docs/design/README.md) and [validation map](../docs/validation/README.md) retain the dated evidence boundaries.

The current [example harnesses](../src/harness/examples) and [contribution checks](../CONTRIBUTING.md#verify-a-change) own source regression coverage. The Shop and scripted method-exchange scenarios establish finite local mechanics, not provider quality or adopting-project acceptance.
