# Project-world Shop example

This fixture demonstrates project-owned Markitect files under `.markitect/`, a recursive Manager tree, vertical Sales/Orders/Inventory slices, explicit file ownership and realization, and a runnable cancellation example using Python's standard library and SQLite.

The fixture is intentionally small. It has no provider configuration, agent credentials, or automatically starting runtime. The empty `.markitect/runtime.yaml` is a placeholder until the project owner chooses a runner; the business code and tests run independently.

See the repository guide at `docs/project-workflow.md` for the project CLI lifecycle. Run the example tests from this directory with:

```powershell
python -B -m unittest discover -s tests -v
```

```sh
python3 -B -m unittest discover -s tests -v
```

The model is a structural specification. Passing these tests establishes only the finite SQLite behaviors they exercise; it does not establish that the model captures every requirement or that an AI followed the workflow.

The request/response example in `.markitect/agent-fixtures/` is a protocol fixture, not a live invocation or provider configuration. The repository guide shows how a supported agent host can use the CLI as a read/proposal boundary.
