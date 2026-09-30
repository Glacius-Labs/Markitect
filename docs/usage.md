# Using Markitect

Run the CLI built from this standalone repository against an explicitly selected consumer with `--repo PATH`. The source repository itself is not a configured consumer; [the minimal example](../examples/minimal/README.md) is.

## Resource model

The format has six namespaced content kinds—`Text`, `Rule`, `Contract`, `Workflow`, `Skill`, and `Agent`—plus the unnamespaced `Project` configuration in `markitect.yaml`. Every resource has `apiVersion`, `kind`, `metadata`, and `spec`. Content resources need a DNS-label `name`; in a project graph, their DNS-label `namespace` must match the owning area. Project metadata requires only `name` and forbids `namespace`. Content `spec` requires nonempty `text`; Project `spec` requires a `profile`.

```yaml
apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata:
  name: example
spec:
  profile: generic
  areas:
    - name: general
      path: docs/general
    - name: product
      path: docs/products/example
      imports: [general]
```

A content resource can hold text directly. Save this example under the `general` area's path and use the area's name as its namespace:

```yaml
apiVersion: markitect.example.org/v1alpha1
kind: Text
metadata:
  name: filing-basics
  namespace: general
spec:
  text: |
    Explain the filing process and its key dates.
```

The paths under `spec.areas` establish ownership. A resource's `metadata.namespace` must match the most specific area containing its YAML path. Area paths use normalized repository-relative POSIX syntax. `imports` explicitly permits references to another area's resources; imports do not pass transitively. A resource receives the `rules` explicitly listed by every area containing its path, including parent paths. This is mandatory path scope; namespace names do not imply inheritance and resources have no override mechanism.

Each content kind has a limited set of `spec` fields. The YAML parser rejects unknown fields, duplicate keys, extra YAML documents, and unsafe YAML features. References are resolved against the complete project graph. A Contract declares an interface, for example:

```yaml
apiVersion: markitect.example.org/v1alpha1
kind: Contract
metadata:
  name: review
  namespace: general
spec:
  kind: Agent
  input: [change]
  output: [findings]
  text: Describe the review expected from an implementation.
```

Project bindings connect a Contract to an Agent, Skill, or Workflow. The implementation must declare that it implements the Contract, and its `input` and `output` lists must match the Contract exactly.

`spec.files` declares additional repository-relative UTF-8 text files needed by a resource. Paths must be explicit normalized paths without globs. Markitect checks that each file exists in the same or an imported area and includes it in that resource's compiled context. These declarations do not infer Markdown links, directory contents, or navigation relationships. A generated Markdown companion for a typed resource must be reached through the typed resource reference instead of `spec.files`.

`markitect schema` checks the generated draft 2020-12 JSON Schema vocabulary emitted as YAML. The schemas help editors with object shape and reject undeclared object properties; parser and graph checks remain authoritative for semantic constraints. These files are not Kubernetes CRDs.

## Commands

Examples run from the standalone Markitect repository. Replace `../consumer` and sample commits with the intended consumer checkout and available commit IDs. Build `bin/markitect` once when recording context and review evidence so all steps use the same executable.

| Command | Purpose |
|---|---|
| `check` | Validate the typed YAML graph, declared file inputs, and generated outputs. |
| `verify` | Run `check` plus the fixed repository checks selected by the Project profile. Requires an immutable revision. |
| `inventory` | List legacy Markdown candidates and, when configured, typed resources; it does not infer dependencies or assert semantic validity. |
| `context` | Compile the dependency closure and its declared file inputs for one resource. |
| `impact` | Compare two immutable snapshots and report changed paths and affected resources. |
| `review` | Record an advisory AI report or determine whether its declared inputs still permit reuse. |
| `render` | Check generated output drift, or write managed outputs from the working tree. |
| `format` | Check YAML canonicalization or write canonical YAML from the working tree. |
| `migrate` | Plan the current Konfyra Markdown-to-YAML migration, or apply it on a non-protected working branch. Reports dependency candidates for human review. |
| `schema` | Check schema output drift, or write generated schema YAML. |
| `package` | Build a deterministic source archive and its YAML version/SHA-256 lock file. |
| `version` | Print the CLI version and target platform. |

```powershell
# Provisional working-tree check
go run ./cmd/markitect check --repo ../consumer

# Immutable check and repository-profile gates at one fixed commit
go run ./cmd/markitect verify --repo ../consumer --revision 0123456789abcdef0123456789abcdef01234567

# Compile context for one Skill
go run ./cmd/markitect context --repo ../consumer --kind Skill --name example --namespace product

# Compare fixed base and candidate commits
go run ./cmd/markitect impact --repo ../consumer --base 0123456789abcdef0123456789abcdef01234567 --revision 89abcdef0123456789abcdef0123456789abcdef

# Check generated views, then write them from the working tree when ready
go run ./cmd/markitect render --repo ../consumer
go run ./cmd/markitect render --repo ../consumer --write

# Check YAML canonicalization, then write normalized YAML on a working branch
go run ./cmd/markitect format --repo ../consumer
go run ./cmd/markitect format --repo ../consumer --write

# Preview the supported Konfyra migration; --write applies it to a working tree
go run ./cmd/markitect migrate --repo ../consumer
go run ./cmd/markitect migrate --repo ../consumer --write

# Write generated schemas, then check for drift
go run ./cmd/markitect schema --repo . --write
go run ./cmd/markitect schema --repo .

# Package the standalone module into a new output directory
go run ./cmd/markitect package --repo . --output .artifacts/release-candidate
```

`render` without `--write` checks existing managed outputs. Writes use the current working tree and are rejected on `main` or `master`; `render --write` rechecks the source inventory before writing and refuses unmanaged output collisions. `format --write` canonicalizes YAML serialization and line endings in `spec.text`; it requires a non-protected Git branch. `migrate --write` is limited to the current Konfyra migration adapter and requires a non-protected branch. The migration reports explicit Skill/Agent links to Workflow files as `dependencyCandidates` and sets `requiresDependencyReview`; these candidates never become normative `uses` references automatically. Review candidate links and resulting files before committing. The schema writer updates only generated schema files.

`package` requires a new, absent output directory. It writes `tools/markitect/source.zip` and `markitect.lock.yaml` beneath that directory. Source text is validated as UTF-8 and normalized to LF so Windows checkout line endings do not change the archive. The lock records the release-candidate version, archive path, and SHA-256 digest so a bootstrap can pin the exact source archive. The Go bootstrap is a separate integration file and is not included in the source archive.

The consumer bootstrap uses this exact pair: copy the archive to `tools/markitect/source.zip`, the lock to the repository root as `markitect.lock.yaml`, and the runner from `integration/run-markitect.go` to `scripts/run-markitect.go`. From the consumer repository root, run it with Go, for example `go run scripts/run-markitect.go check`. The runner verifies the archive against the lock, builds that pinned Go source into a digest-keyed `.artifacts/markitect/` cache, and forwards the command. Its cache metadata is a flat YAML file named `build-stamp-<id>.yaml` with `version`, `source_sha256`, and `executable_sha256` fields. It requires Go on the machine. The outer `go run` needs a writable Go cache before the bootstrap starts; see [consumer integration](../integration/README.md). This flow uses repository-pinned files and has no release-server URL.

Successful commands return 0, validation findings return 1, and commands that cannot run or receive invalid arguments return 2. Results are YAML on standard output; command errors go to standard error.

## Snapshot and evidence boundaries

Omitting `--revision` reads the working tree and marks the result `provisional: true`. This is useful for local feedback, but it is not immutable evidence. Supplying `--revision` makes Markitect resolve a Git commit and read that committed tree. `impact` requires both `--base` and `--revision`. `verify` requires `--revision` and runs the selected profile's fixed checks against a materialized copy of that same snapshot.

Coverage is reported with command results. `check` covers the typed graph, declared inputs, and Markitect-owned outputs. `verify` adds the configured repository gates; the `konfyra` and `cockpit` profiles have adapters, while `generic` currently has no repository gate adapter. The supported profiles invoke consumer-owned checks that currently require `python` or `python3` on `PATH`. Markitect itself, including its bootstrap and tests, is Go; this runtime dependency belongs to the existing consumer gates. The Konfyra profile retains its existing Python renderer as the owner of provider outputs during the pilot; Markitect checks and renders its adjacent typed-resource views. A passing structural or repository check does not establish semantic review or external acceptance.

Context output includes three distinct digests or identifiers. `snapshotDigest` covers the complete loaded repository snapshot, including sorted paths, file modes, and bytes. `toolDigest` identifies the executable bytes. `digest` fingerprints the compiled entry, tool version and digest, selected resource files, and declared `spec.files` inputs. These digests are not semantic-review results.

## Reuse an advisory AI review

Keep a small review configuration outside the typed resource areas, for example at the repository root as `markitect-review.yaml`, and commit it with the candidate:

```yaml
question: Is this entry's declared context sufficient and internally consistent for its task?
promptVersion: context-review-v1
model: gpt-6-luna
effort: high
allowReuse: true
```

Give the reviewer the fixed `context` output and that exact question/configuration. Save its actual report as UTF-8 text. Record it only after the review has completed, using the executable that compiled its context:

```powershell
markitect review --revision COMMIT --namespace general --kind Skill --name author-ai-mechanism --config markitect-review.yaml --report .artifacts/markitect/reviewer-report.md > .artifacts/markitect/review.yaml
markitect review --revision NEXT_COMMIT --namespace general --kind Skill --name author-ai-mechanism --config markitect-review.yaml --evidence .artifacts/markitect/review.yaml
```

The configuration is read from the fixed candidate, never from a later working-tree edit. Reuse verifies the original snapshot, context and report hashes, compares the requested configuration and executable, recompiles the candidate context, and checks the union of old and new dependency impacts. Unknown inputs or inventory changes invalidate reuse conservatively. `allowReuse: false` always requests a new review. Exit code 0 means reusable, 1 means a new review is needed, and 2 means the evidence could not be evaluated. Recording a report returns 0 when its inputs are valid; it does not assert that the report is positive.

The command makes no model call. A reusable result retains the original report and revision; an agent can use that result without repeating the same model review. Human approvals are never transferred. Evidence is local and advisory: hashes detect mismatch but do not authenticate a reviewer, and an imported success statement is not trusted CI evidence. Every input needed for the question must be declared; reviews that use additional undeclared material are ineligible for reuse. Keep records outside their input snapshot, for example in the excluded `.artifacts/markitect/` directory.

## Normal edit sequence

Select ownership and explicit dependencies, then edit canonical YAML. Run `format --write`, `render --write`, the consumer provider renderer (when separately owned), and `check`/consumer gates. Commit the complete candidate, resolve its SHA once, then run `context`, `verify`, and `impact` using fixed commits. Review the affected semantics and evaluate eligible existing evidence. `check` includes generated-output drift, so it is expected to fail between a source change and regeneration. Changes after the commit belong to a later candidate.

For immutable example runs, copy the complete `examples/minimal` directory into a separate Git repository and commit it. Passing that nested example directory with a revision from the parent source repository is unsupported: Git snapshots use repository-root paths.
