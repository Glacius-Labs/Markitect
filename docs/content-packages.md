# Content packages

This guide describes the content-package model in Markitect 0.3.0. The immutable v0.1.0 release remains historical. See [GitHub Releases](https://github.com/Glacius-Labs/Markitect/releases) for available distributions.

Content packages let a Project consume a selected set of reusable Markitect resources and their declared file inputs from an offline archive. A Project pins each direct package by exact name, version, provenance coordinate, repository-relative archive path, and SHA-256. There is no floating resolution or second content lock. `markitect.lock.yaml` continues to describe the Markitect CLI distribution only.

## Package manifest and archive

A package archive contains `markitect-package.yaml`, canonical resource YAML, and only the ordinary UTF-8 files declared by those resources. Its manifest uses `kind: Package`, a DNS-label `metadata.name`, and `spec.version`, `spec.areas`, `spec.exports`, plus optional package-local `spec.bindings`. Export references identify resources by package-local namespace, kind, and name. Resources and file inputs remain inside the archive.

The package model is intentionally direct and offline. A package cannot import another package or reference consumer resources. It cannot declare checks, render targets, or rule adapters. External rule adapters are rejected in this initial slice; a consumer can add a local wrapper and declare its own rule relationship. Imported Rules do not activate themselves or apply to unrelated consumer resources.

Archive reads are bounded: a ZIP is at most 64 MiB compressed, at most 128 MiB expanded, contains at most 10,000 files, and each member is at most 8 MiB. Across all direct packages in a Project, expanded content is limited to 128 MiB and 10,000 files total.

## Pin packages in a Project

Use the package's suggested pin as a starting point, then choose a committed archive path and provenance that match the exact bytes you reviewed. Here, “vendored” means that the archive is checked into the consumer repository; do not place it in a directory literally named `vendor`:

```yaml
apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata:
  name: sample-project
spec:
  areas:
    - name: product
      path: docs/product
  packages:
    - name: shared-guidance
      version: 1.2.0
      source: git:0123456789abcdef0123456789abcdef01234567
      archive: packages/shared-guidance-1.2.0.zip
      sha256: 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
```

The version and digest are exact values, not ranges. `source` records provenance and is never fetched. The archive must exist in the selected Project snapshot and match its digest. Markitect excludes root-level `vendor/`, `.artifacts/`, and `.cache/`, along with build/cache directories such as `bin` and `obj`; files there cannot serve as package archives or package inputs. Put the archive under an included path such as `packages/` and commit it with its pin so a fixed revision selects both. The single Project pin list avoids maintaining a second lock or asking Markitect to reconcile two configuration sources.

An exported package resource can be used explicitly:

```yaml
apiVersion: markitect.example.org/v1alpha1
kind: Skill
metadata:
  name: product-review
  namespace: product
spec:
  description: Review product changes against the shared review workflow.
  uses:
    - package: shared-guidance
      namespace: core
      kind: Workflow
      name: review-change
  text: |
    Apply the selected shared workflow to the product change and report findings
    with the evidence and limits described by that workflow.
```

This is a complete Skill resource with the required description and text. The `package` field qualifies its Workflow reference. The package must be pinned, and the target must be one of its declared exports. An unqualified reference inside a package resolves within that package. Local identities remain `namespace/kind/name`; an imported graph entry has the origin-qualified identity `package::namespace/kind/name`.

Consumer Project bindings may select an exported package Contract and implementation. A consumer cannot replace package-local bindings. Package-to-package references and direct package-to-consumer references are invalid; declare the dependency from the consumer side using an export.

## Build and use a package

Build from an immutable Git revision. `--output` must name a ZIP file that does not already exist:

```powershell
markitect pack --repo . --revision 0123456789abcdef0123456789abcdef01234567 --output .\out\shared-guidance.zip
```

`pack` validates the closed package graph, creates a deterministic archive from the fixed Git snapshot, and emits a suggested Project pin containing the exact digest. The suggestion uses the source revision as provenance and `packages/<name>-<version>.zip` as the archive path; it does not install the archive or edit a Project. Review the package and archive bytes, place them at the suggested path in the consumer repository, then copy or adapt the suggested pin. Create the output's parent directory first if needed; the command only creates the ZIP itself and refuses to overwrite an existing file.

Use `--package` with `context`, `explain`, or `review` to select an exported package entry. `find` can list exported package entries and filter by package. For example:

```powershell
markitect find --repo . --query review --package shared-guidance
markitect explain --repo . --package shared-guidance --namespace core --kind Workflow --name review-change
markitect context --repo . --package shared-guidance --namespace core --kind Workflow --name review-change
```

Package dependencies shown in rendered views are plain text with the corresponding CLI selection instruction. Imported package resources and files are read-only inputs: `render` and `format` write only consumer-owned resources and views.

## Fixed inputs, impact, and trust

The Project's Git snapshot remains the source snapshot. Verified archive members are tracked separately as immutable package inputs; they do not become repository paths or writable outputs. Context records package origin, package manifest, exact pin and archive identity, and selected resource/file byte hashes. Only exports are selectable directly through consumer queries, while dependencies of an export may also appear in compiled context.

Impact and review eligibility include package content. Any pin or archive change may conservatively invalidate the whole Project in this initial slice. The old/new graph union and unknown-input conservatism still apply.

An archive digest establishes byte integrity against the Project pin. It does not authenticate a publisher, prove that a provenance claim is true, or make the package confidential. Exported context can contain internal dependencies required by an exported resource. Review package contents and provenance before accepting a pin; no network fetch, model call, script execution, or provider activation occurs as part of package loading or selection.

## Executable examples

[The package fixture](../examples/content-package/README.md) shows a fixed-revision pack operation. [The consumer fixture](../examples/package-consumer/README.md) shows a committed archive pin, a local Skill wrapper around the exported Workflow, and package-qualified context selection.

See [Usage](usage.md) for the complete Project and CLI contract and [Architecture](architecture.md) for implementation boundaries.
