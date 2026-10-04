# Artifact coverage module

`artifactcoverage` evaluates explicit ownership for a bounded set of repository
paths. It is a pure module: it does not read the filesystem, Git, Project
resources, or renderer state. The Host supplies the config bytes and composes
one `Inventory` from the already parsed Core model, resolved opaque inputs,
renderer output ownership, and the fixed snapshot file bytes.

## Contract

The Host decodes policy bytes with `ParseConfig`, sets `Config.SourcePath` to
the exact repository-relative path from which those bytes came, then calls
`Check(config, inventory)`. `SourcePath` is Host context and is not serialized
in the versioned YAML API. The config has this shape:

```yaml
apiVersion: markitect.example.org/artifact-coverage/v1alpha1
kind: ArtifactCoverage
spec:
  roots:
    - content
  tooling:
    - path: tooling/check.go
      owner: artifact coverage checker
  exclusions:
    - path: content/retained-fixture.md
      reason: retained historical example
```

Roots are exact repository-relative file or directory paths. Tooling ownership
and exclusions also name exact paths; exclusions require a human-readable
reason. Globs and broad directory exclusions are invalid. The policy file
itself must be inside a managed root and explicitly tooling-owned. Resource
inputs come from the supplied `InputOwners` facts and are not duplicated in
policy. Multiple resources may consume one input path without creating an
ownership collision. Canonical resource ownership and generated-output owners
are facts from the same Host-composed model.

## Findings and evidence limits

The module reports unmanaged files, missing declared inputs, missing renderer
outputs, renderer-marked orphan projections, stale tooling/exclusion entries,
missing roots, and conflicting classifications. It checks exact path spelling
and portable case aliases. Renderer orphan-marker recognition is limited to
`.md` and `.toml` projections; other generated artifacts need an explicit
tooling owner or reasoned exclusion. Unknown generated-looking YAML remains
unmanaged unless explicitly classified.

The inventory's `SnapshotDigest` binds a report to the snapshot selected by
the Host. The `Files` map must represent all files the Host intends this check
to cover, including untracked files when checking a live working tree. In a
materialized fixed snapshot, it can only report files present in that
snapshot; it cannot discover files outside it. The module does not establish
how the snapshot was selected, independently verify the digest, inspect
symlinks or file modes, or infer ownership by reading artifact contents.
Those responsibilities and claims belong to the Host and its evidence
boundary.
