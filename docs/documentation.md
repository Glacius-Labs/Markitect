# Documentation inputs

Markitect can connect ordinary documentation to exact source inputs without
turning every file into a typed resource. A resource may declare UTF-8 files
that belong to its area or a directly imported area in `spec.files`; these bytes enter compiled context and
their changes seed conservative impact analysis. The path is explicit and
repository-relative. Globs, content inference, and semantic truth checking are
not part of this feature.

Code can stay in its normal source directory. For example, a Project may give
`src/` its own area and let the area containing `docs/` import it. The owning
documentation resource then lists the exact `src/...` file it needs. The import
permits that dependency; it does not load the entire source area or infer code
relationships. The executable example places its small source file beside its
documentation to keep the initial model to two independent areas.

The [code and documentation example](../examples/documentation/README.md)
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
