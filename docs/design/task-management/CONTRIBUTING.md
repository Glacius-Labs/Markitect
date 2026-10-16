# Development guide

## Project stage and prerequisites

Tasks currently has documentation only. The package layout and commands below are the intended development interface, not existing source files or successful build evidence. Implement the documented first version before claiming any of these checks have run.

Use Python 3.13.x, Git and a normal local filesystem. Runtime and tests use the standard library; no pip dependencies are planned. Windows 11 and Ubuntu 24.04 are the release targets. A virtual environment is optional because no project packages are installed. Verify which interpreter `python` names before running commands.

Read the [product scope](docs/product.md), [domain](docs/domain.md), [CLI](docs/cli.md), [architecture](docs/architecture.md) and [storage contract](docs/storage.md) relevant to your change.

## Intended repository layout

```text
README.md
AGENTS.md
CONTRIBUTING.md
CHANGELOG.md
.gitignore
tasks/                  package and process entrypoint
tests/                  domain, action, storage, CLI and restart tests
docs/                   product, interface, architecture, operations and quality
```

[Architecture](docs/architecture.md#source-layout) describes the module split. Keep files cohesive; do not create framework layers or empty packages. The planned `.gitignore` excludes `__pycache__/`, `*.pyc`, local test output, source release archives and any deliberately local data/backup files. The default home-directory task file is outside the repository; ignoring a repository path does not protect arbitrary `--file` overrides.

## Intended commands

From the future source root:

```text
python --version
python -m compileall -q tasks tests
python -m unittest discover -s tests -p "test_*.py" -v
python -m tasks --file "./scratch-tasks.json" list
```

The final command is a manual smoke using a disposable file in an existing directory. Never run write tests against the normal home-directory file. Tests should use temporary directories, a controlled home for default-location scenarios and separate processes for CLI/restart evidence. Keep test filenames `test_*.py` discoverable from the stated entrypoint. Syntax compilation is not behavioral verification.

## Making a change

1. Describe the requested outcome and identify affected behavior, data and documents. Clarify genuine scope choices without inventing unnecessary process gates.
2. For an intended behavior change, update the appropriate specification and decision rationale before or with the implementation. A repair of already documented behavior does not require redefining that behavior.
3. Work on a focused branch. Implement the smallest coherent change, update affected tests and user/developer instructions, and preserve unrelated local work.
4. Run focused checks while developing and the complete suite before delivery. Investigate failures; do not claim verification from tests that were skipped, cached elsewhere or run against another revision.
5. Obtain an independent review of correctness, storage safety, scope and changed documentation. Resolve material findings and repeat affected checks after repairs.
6. Summarize what changed, why, evidence actually obtained and remaining limitations. The owner decides changes to scope and release readiness.

There is no required external board, hosting service or CI provider. If automated CI is introduced, it should invoke the same documented commands rather than invent a separate behavioral contract.

## Test design and evidence

Use independent expected values. Cover outcomes and file effects, including no-write paths; a test that only mirrors implementation details is insufficient. A controlled storage double supports action tests but does not replace real JSON/filesystem and subprocess tests. The [test strategy](docs/testing.md) owns the scenario inventory and release criteria.

Record evidence against an exact candidate and platform. A Windows pass does not establish Linux behavior, and an automated pass does not establish human usability. Do not report proposed scenarios as completed tests.

## Documentation and decisions

Keep the [documentation map](docs/README.md) navigable. Update affected examples, operations, compatibility notes and changelog with behavior changes. Add or amend a concise decision record when a meaningful tradeoff changes; do not turn every implementation detail into an architecture decision.

No task contents, local backups or real user paths belong in examples or issue reports. Use synthetic fixtures. Release packaging and source updates follow [Operations](docs/operations.md#source-release-and-update).
