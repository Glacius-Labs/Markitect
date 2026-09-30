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

An explicitly declared `.yaml` or `.yml` may be ordinary input, including a Kubernetes manifest, when it is valid YAML and is not recognizable as a Markitect resource. The Markitect API group and an incomplete envelope with a known kind plus mapping `metadata` and `spec` remain parse errors; malformed YAML also fails closed. Do not list a valid typed resource's source file as an input—reference that resource through the graph.

`markitect schema` checks the generated draft 2020-12 JSON Schema vocabulary emitted as YAML. The schemas help editors with object shape and reject undeclared object properties; parser and graph checks remain authoritative for semantic constraints. These files are not Kubernetes CRDs.

## Commands

Examples run from the standalone Markitect repository. Replace `../consumer` and sample commits with the intended consumer checkout and available commit IDs. Build `bin/markitect` once when recording context and review evidence so all steps use the same executable.

| Command | Purpose |
|---|---|
| `check` | Validate the typed YAML graph, declared file inputs, and generated outputs. |
| `verify` | Run `check` plus the fixed repository checks selected by the Project profile. Requires an immutable revision. |
| `inventory` | List legacy Markdown candidates and, when configured, typed resources; it does not infer dependencies or assert semantic validity. |
| `find` | Search valid typed resources by literal text, kind and namespace; return concise canonical locations. |
| `explain` | Show one resource's area, direct resolved relationships, declared files and Contract implementations. |
| `authoring` | Compile the core authoring Skill shipped inside this executable; no repository required. |
| `context` | Compile the dependency closure and its declared file inputs for one resource. |
| `impact` | Compare two immutable snapshots and report changed paths and affected resources. |
| `review` | Record an advisory AI report or determine whether its declared inputs still permit reuse. |
| `render` | Check generated output drift, or write managed outputs from the working tree. |
| `format` | Check YAML canonicalization or write canonical YAML from the working tree. |
| `migrate` | Plan the existing Markdown-to-YAML migration using `--profile konfyra` (default) or `--profile cockpit`, or apply it with `--write` on a non-protected branch. Dependencies still need explicit review. |
| `schema` | Check schema output drift, or write generated schema YAML. |
| `package` | Build a deterministic source archive and its YAML version/SHA-256 lock file. |
| `bundle` | Build the complete five-file distribution from `--revision COMMIT` into an absent `--output FILE.zip`; version must match that source. |
| `install` | Validate `--bundle FILE.zip --sha256 HASH` and preview a complete pin change; `--write` applies it on a non-protected Git branch. |
| `version` | Print the CLI version and target platform. |

`markitect --help`, `markitect help COMMAND`, and `markitect COMMAND --help` return usage without reading a repository. Command help shows only applicable flags. Follow [consumer integration](../integration/README.md) for release authentication, installation, upgrades and rollback. A bundle checksum detects changed bytes; trusted release provenance establishes which bytes to accept.

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

The consumer bootstrap uses this exact pair: copy the archive to `tools/markitect/source.zip`, the lock to the repository root as `markitect.lock.yaml`, and the runner from `integration/run-markitect.go` to `scripts/run-markitect.go`. Copy its paired test from `integration/run-markitect_test.go` to `scripts/markitect-bootstrap_test.go`. From the consumer repository root, run it with Go, for example `go run scripts/run-markitect.go check`. The runner verifies the archive against the lock, builds that pinned Go source into a digest-keyed `.artifacts/markitect/` cache, and forwards the command. Its flat YAML `build-stamp-<id>.yaml` records `version`, `source_sha256`, `toolchain`, `build_policy` and `executable_sha256`. Cache reuse requires every identity field to match. The bootstrap selects the toolchain from the verified module before lookup and pins that selection during the build. Builds use portable native settings with CGO disabled and ignore ambient compiler/workspace overrides. The outer `go run` needs a writable Go cache before the bootstrap starts; see [consumer integration](../integration/README.md).

Successful commands return 0, validation findings return 1, and commands that cannot run or receive invalid arguments return 2. Results are YAML on standard output; command errors go to standard error.

## Snapshot and evidence boundaries

### Discover and author resources

```powershell
# Core guidance, compiled from the resources embedded in this release
./bin/markitect.exe authoring

# Literal, case-insensitive match in identity, path, description or body
./bin/markitect.exe find --repo examples/minimal --query rollback --kind Rule

# Inspect canonical location, governing area and direct incoming/outgoing relationships
./bin/markitect.exe explain --repo examples/minimal --namespace sample --kind Rule --name rollback-review
./bin/markitect.exe explain --repo examples/minimal --namespace sample --kind Contract --name rollback-assessment
./bin/markitect.exe explain --repo examples/minimal --kind Project --name rollback-example
```

`find` without query text lists all matches; `--kind` and `--namespace` are exact optional filters. It returns summaries, not duplicated prose. Both commands emit a YAML envelope with tool/version, revision/provisional state, snapshot digest and a `result` containing the facts. A missing match is an empty successful result; an unknown exact identity in `explain` is an error. Both require a structurally valid project; use `check` diagnostics to repair invalid inputs first. They do not require freshly rendered views, which makes them useful while editing.

Both queries accept `--revision COMMIT` and report the selected snapshot. `explain` names the longest owning area for a content resource. Project has no namespace or owning area. Ownership here is structural, not human authorization. Relationships record their resolved source/target and declaration origin, including applicable area rules and selected bindings. `resourceLine` is the start of the declaring YAML resource, not the line of its nested reference. These are direct relationships; `context` follows the complete dependency closure and reports an inclusion reason for each input. Use `impact` for old-and-new consequences of an actual committed change.

`authoring` accepts no flags and reads no consumer repository. Its `revision: embedded` is a source marker, not a Git commit. Its snapshot and tool digests identify the bundled guidance. The core Skill, Workflow, Rule and Text are canonical YAML compiled through the same parser and graph as other resources. Consumers need no copied installation or optional package to read them. Their own rules and delivery authority still apply to every edit.

### Repository snapshots

Omitting `--revision` reads the working tree and marks the result `provisional: true`. This is useful for local feedback, but it is not immutable evidence. Supplying `--revision` makes Markitect resolve a Git commit and read that committed tree. `impact` requires both `--base` and `--revision`. `verify` requires `--revision` and runs the selected profile's fixed checks against a materialized copy of that same snapshot.

Coverage is reported with command results. `check` covers the typed graph, declared inputs, and Markitect-owned outputs. `verify` adds the configured repository gates; the `konfyra` and `cockpit` profiles have adapters, while `generic` currently has no repository gate adapter. Cockpit snapshots containing `scripts/check-cockpit.go` and `scripts/check-cockpit_test.go` run their native Go checker and tests. Historical Cockpit snapshots without both files retain their existing Python checks; an incomplete pair fails before execution. Konfyra retains its existing Python renderer and regressions as consumer-owned compatibility. Both profiles add `go test -count=1` for the bootstrap when the snapshot contains both `scripts/run-markitect.go` and `scripts/markitect-bootstrap_test.go`; an incomplete pair also fails. Markitect itself, including bootstrap and tests, is Go. The Konfyra profile retains its existing renderer as the owner of provider outputs; Markitect checks and renders its adjacent typed-resource views.

Every gate has a ten-minute timeout, a streaming 1 MiB output limit and a two-second pipe-wait limit. Failure reports preserve completed and failed gate output with the fixed snapshot identity. A normal nonzero gate exits 1 with `status: failed`; missing tools, missing gate support, incomplete integration, timeout or output overflow exit 2 with `status: incomplete` and a `verify.<kind>` diagnostic. Remaining gates do not run after a failure. [Operations](operations.md) owns recovery and precise process-isolation limits. A passing structural or repository check does not establish semantic review or external acceptance.

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
