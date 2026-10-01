# Project artifact inputs

Markitect can connect AI-facing resources to exact project files without
turning every file into a typed resource. A resource may declare ordinary UTF-8
files that belong to its area or a directly imported area in `spec.files`.
These bytes enter compiled context, and their changes seed conservative impact
analysis. The path is explicit and repository-relative. Source code, schemas,
configuration, infrastructure, CI files, and documentation follow the same
mechanism. Arbitrary binary files, globs, content inference, and semantic truth
checking are not part of this feature. See the [product boundary](architecture.md#project-artifact-boundary).

Code can stay in its normal source directory. For example, a Project may give
`src/` its own area and let the area containing `docs/` import it. The owning
documentation resource then lists the exact `src/...` file it needs. The import
permits that dependency; it does not load the entire source area or infer code
relationships. The executable example places its small source file beside its
documentation to keep the initial model to two independent areas.

The [Go input example](../examples/documentation/README.md)
connects a Go source file to a `Text` resource, then to a `Workflow`. Changing
the source file affects the declared `Text` and its dependent `Workflow`, while
independent documentation in a second area stays outside the selected context
and impact set. Structural checks still pass when the prose no longer matches
the implementation. A targeted semantic review is needed to decide whether the
documentation should change.

Markitect's review-evidence API can record an advisory report for a fixed
context and determine whether changed inputs affect that entry. It does not
call a model, interpret report prose, establish correctness, or imply human
acceptance. The example test exercises the existing context, impact, and
review-reuse APIs; it records no model or productivity metrics.

## Fixed-run task and artifact inputs

For one scoped task, a committed ContextRun manifest can add a work-item
snapshot and a bounded list of exact project artifact paths to the normal
entry closure. Keep the manifest in the same repository and commit as the
inputs. The manifest field is named `sources`, but its selected UTF-8 files
are not limited to source code. Pass the immutable commit separately so the
manifest does not need to contain its own commit id:

```yaml
version: markitect.example.org/context-run/v1alpha1
entry: operations/Skill/change-review
task:
  id: change-1234
  path: work-items/1234.md
sources:
  - path: api/openapi.yaml
    reason: declared API contract for the selected change
  - path: deployment/service.yaml
    reason: deployment configuration affected by the change
```

```powershell
markitect context --repo . --revision <full-commit-id> --run docs/context-runs/change-1234.yaml
```

The manifest has one entry, one required task id and file, and at most 64
unique, normalized repository-relative artifact paths. Every selected path is
required by default; set `optional: true` only when absence is acceptable. The
report includes the resolved revision, snapshot digest, manifest path and hash,
task id, and each selected task/artifact path with its reason, status, byte hash
and captured text. Missing required paths produce an `incomplete` report and
exit code 2. Invalid UTF-8 or NUL-containing inputs are identified as invalid
text in the report. Selected task/artifact bytes are limited to 16 MiB in total.
The context digest includes the run selection, including requested paths that
are missing. No globbing or artifact inference is performed.

Markitect hashes and captures the work-item file, but it does not interpret
acceptance criteria or prove that the selected artifact set is sufficient. A
human still needs to assess task scope, artifact relevance and resulting changes.
