# Project artifact input example (Go)

This fixture connects one ordinary Go source file to a `Text` resource through
an explicit `spec.files` path. A `Workflow` uses that resource. Independent
operational guidance lives in a separate area, so a source-only change has a
small expected context and impact set.

- [Implementation area](docs/implementation/README.md) owns the behavior description, workflow, and Go input.
- [Operations area](docs/operations/README.md) owns unrelated deployment guidance.

From the Markitect repository root, run:

```powershell
.\bin\markitect.exe check --repo examples/documentation
.\bin\markitect.exe context --repo examples/documentation --kind Workflow --name startup-review --namespace implementation
.\bin\markitect.exe render --repo examples/documentation --write
```

The example uses an exact repository-relative file path. `spec.files` does not
infer code dependencies or support glob patterns. Changing `maxAttempts` from
three to five leaves the graph structurally valid, but makes the explanatory
prose stale. Markitect can include the source in context and report that the
`Text` resource and its dependent `Workflow` are affected; a semantic reviewer
must still decide whether the prose should change.

To inspect fixed-revision `impact` and review-evidence behavior, copy this
directory into a standalone Git repository, commit it, then change only
`docs/implementation/src/worker.go` and commit the candidate. The checked-in
rendered views should remain unchanged because they project the YAML resource,
not the Go source.