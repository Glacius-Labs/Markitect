# Parallel wave: GitHub repository metadata

**Candidate baseline:** `0ba7a7218f2ceae65b8db90bb8133c2ffa583ada`.
**Scope:** one offline, read-only Repository metadata conformance consumer in
`cmd/markitect-adapter-github/`. This is an implementation candidate, not a
Markitect release, GitHub integration, or evidence of adopter benefit.

## Mapping and observed claim

The adapter consumes the frozen external command request/model protocol and
uses one adapter-owned parameter mapping:

```yaml
parameters:
  repositories:
    - resource: <exact semantic-model identity.key>
      evidenceFile: <exact Project-declared captured JSON input>
```

The mapping defines the check scope. It cannot supply alternate expected
values. Desired `fullName` and `defaultBranch` come only from the mapped,
normalized canonical `Repository` resource's top-level `data` properties.
The evidence file is a sanitized capture envelope containing one request path,
the pinned GitHub REST API version, response status, and selected body fields
`full_name` and `default_branch`. The adapter checks the request and response
identity against canonical `fullName`, then compares the default branch.

The GitHub REST reference defines `GET /repos/{owner}/{repo}`, documents
`full_name` and `default_branch` in the response, and currently shows API
version `2026-03-10` in the request example: [Get a repository](https://docs.github.com/en/rest/repos/repos?apiVersion=2026-03-10).
The fixture input is only an exact local capture record. Its contents do not
prove source authenticity, that a request was made, API authorization, or
continuing remote state.

## Lifecycle and evidence limits

`observe` reads only mapped captured files staged as declared inputs. It emits
canonical scope, selected observed values, a digest per exact capture file,
and findings. A completed default-branch difference is a `default-branch-drift`
finding, not an operation. `plan` rechecks the same fixed capture and emits no
operations. `verify` requires the matching model/adapter/observation/plan,
replays the fixed evidence and returns failed status for outstanding drift or
changed capture digests. No HTTP client, credential lookup, mutation, apply
argv, retries, shared target registry, or another adapter's output is used.

Missing or malformed inputs, unsupported capture versions/statuses, `403`,
`404`, wrong request/response identity, duplicate mappings, unknown parameters,
oversized files, duplicate JSON keys at any depth (including case-folded key
collisions, to avoid Go struct-field matching ambiguity), and unsafe paths
remain incomplete; they cannot silently pass. The process test serializes the actual
`app.AdapterRequest` and complete `app.SemanticModel` DTO, including the fields
the adapter does not interpret, before invoking the built executable.
Each captured JSON input is capped at 1 MiB; the adapter also applies a local
10 MiB stdin request cap. The latter is an executable-specific prototype
bound, not a shared Markitect protocol limit or a scale guarantee.
The bounded implementation does not inspect branch protection, permissions,
topics, security settings, API rate limits, or GitHub Enterprise behavior.

## Validation

The adapter's package tests cover deterministic successful observation,
canonical desired-value sourcing, read-only no-op plans, failed verification
for completed drift, fixed-capture digest drift, unauthorized/missing/malformed
and wrong-target evidence, exact mapping validation, parameter/path bounds,
unsupported Apply/version, duplicate capture keys, and a separately built
executable consuming the unchanged v1alpha1 protocol. Run:

```powershell
go test ./cmd/markitect-adapter-github -count=1
New-Item -ItemType Directory -Force .artifacts/adapters | Out-Null
go build -o .artifacts/adapters/markitect-adapter-github.exe ./cmd/markitect-adapter-github
```

No shared source, Core semantics, CLI, schema, Project configuration, DTO, or
another adapter is changed. The focused executable build demonstrates use of
the baseline command seam; it does not establish release inclusion or verify a
real GitHub response. Live fetch, credential handling, and any write capability
remain separately gated work.
