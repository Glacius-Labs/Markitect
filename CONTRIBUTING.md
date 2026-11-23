# Contributing

Markitect is a small Go tool with a Claude Code plugin, developed as an ordinary project: agentic coding, tests and ordinary documentation, with no spec of itself ([DEC-076](docs/decisions.md#dec-076-markitect-is-developed-as-an-ordinary-software-project)). This page is the whole development process: how to make, check, review and land a change, and where things go.

## Make a change

1. **Start clean.** Work in a worktree of your own, from the current `origin/main`, on a branch with a prefix such as `dev/`, `fix/` or `docs/`. Check the folder, the branch and the base commit before you write. Never use `git stash`: it is shared by every worktree. Set work aside with a commit instead.
2. **Find the owner.** Each Go package has one row in the [code map](docs/development/code-map.md) with the page that owns its behavior, and each topic has one page ([below](#where-things-go)). Change the behavior, its tests and its page together.
3. **Keep it simple.** Choose the clean design even when it is more work, add no aliases or compatibility paths, and delete what is obsolete together with its tests and pages ([DEC-139](docs/decisions.md#dec-139-quality-simplicity-and-stability-over-effort-and-compatibility)). Fix a failure in how agents work with Markitect in the way of working, the `factory` template, and the skill built from it, not with prompt text beside them ([DEC-147](docs/decisions.md#dec-147-the-way-of-working-stands-in-the-spec-and-skills-realize-it)).
4. **Work in parallel only apart.** Several sessions may work at once when their files do not overlap. Shared files, such as `go.mod`, the [architecture rule table](internal/tooling/architecture/rules.go) and generated pages, have one writer at a time.

Product code and tests that handle a project's `.markitect/` folder use fixtures, never this repository's own files.

## Where things go

| New item | Place | Also |
|---|---|---|
| Go package | `internal/<responsibility>/` | A row in the architecture rule table, a package doc comment, and the code map regenerated ([architecture](docs/architecture.md#adding-changing-or-removing-a-package)) |
| Shipped command | `cmd/<name>/` | A row in the rule table |
| File the binary embeds | Inside the embedding package, such as `internal/library/` | |
| Fixture of one package | `<package>/testdata/` | |
| Check that must not import product code | `tools/` | A CI step |
| Maintainer script | `scripts/` | Its tests beside it |
| Documentation | The page that owns the topic, below | |

| Topic | Page |
|---|---|
| Why Markitect exists | [Vision](docs/vision.md) |
| What Markitect builds on: standards, literature and known practice | [Foundations](docs/foundations.md) |
| The product as a whole | [Concept](docs/concept/README.md) |
| Decisions in force and open questions | [Decisions](docs/decisions.md) |
| The spec language and the commands | [Language reference](docs/guides/language-reference.md) |
| The library built into the tool: packages, templates and the skills of the standard | [Library](docs/guides/library.md) |
| The signals, the topic view, reference values, question catalogs and change signals | [Signals](docs/guides/signals.md) |
| The project: finding it, `init`, the `.markitect/` layout, the walk of its files, their hashes, the state with `status`, `confirm` and `move`, the history and `compare` | [Project reference](docs/guides/project-reference.md) |
| The plugin as built: the launcher that fetches the tool, and its start hooks | [The plugin](docs/guides/plugin.md) |
| What makes a good spec, and how to write one | [Is the spec good?](docs/concept/is-the-spec-good.md) |
| Packages, layers and the repository layout | [Architecture](docs/architecture.md) and the generated [code map](docs/development/code-map.md) |
| Releases | [Releasing](docs/releasing.md) |
| Where Main is and what comes next | [Roadmap](docs/roadmap.md) |
| Where earlier concepts and records are | [History](docs/history.md) |
| How Markitect is developed | This page |

**Rules for documentation.**

- Write in English ([DEC-069](docs/decisions.md#dec-069-documentation-is-written-in-english)).
- One page per topic. Link to it instead of repeating it.
- Main holds only the current state ([DEC-071](docs/decisions.md#dec-071-main-holds-only-the-current-state)): no history, no dated records, no quotes, and one concept without version names. A page that relies on a dated result says it in one sentence and links its record: the PR that reported it, or a file at a full commit SHA. What a real run showed goes into its PR.
- A new decision gets the next free number in the [decisions](docs/decisions.md), and the concept changes with it in the same change.
- Generated pages, such as the code map, are regenerated from their source, never edited.
- A new maintained page goes into `DEFAULT_DOCUMENTS` in [`scripts/check-docs.py`](scripts/check-docs.py), so that its links are checked.

## Check a change

Run the checks the change calls for, and record their results in the PR.

- **Documentation only:** `python -B scripts/check-docs.py`, which checks the local links and anchors of the maintained pages (name other pages as arguments), and `git diff --check`.
- **Go code:** `gofmt` on the changed files, `go vet ./...` and the tests of the changed packages. `go test ./internal/tooling/...` whenever a package, a doc comment or an owning page moves. The commands build for Windows too, so a change to code they reach also runs `GOOS=windows go vet ./...`.
- **The code map** after a change of the rule table or a package doc comment: `go test ./internal/tooling/architecture -run TestCodeMapPage -update`.
- **The command smoke** in a fresh, empty folder outside the repository: `go run ./tools/smoke -repo . -work ABSOLUTE_EMPTY_FOLDER`. It builds `markitect` from the repository and drives its commands over a small project. It starts no agent.

The full set, as CI runs it:

```text
go mod verify
go test ./... -count=1 -timeout=20m
go vet ./...
go build ./...
gofmt -l $(git ls-files '*.go')
git diff --check
go run ./tools/smoke -repo . -work ABSOLUTE_EMPTY_FOLDER
python -B -m unittest discover -s scripts -p test_*.py
python -B scripts/check-docs.py
```

CI also runs the tests with the race detector.

CI runs on Linux only, with quality checks and the race detector, in about five minutes ([DEC-077](docs/decisions.md#dec-077-linux-ci-run-on-request)). It starts by hand, and the release workflow runs it for every tag:

```text
gh workflow run ci.yaml --ref BRANCH
```

Local checks and the independent review are the basis for a merge, and CI is optional for ordinary changes. When you run it, run it on the reviewed head and merge only when it is green. The tests run in about a minute on Windows too; Git is started only by `TestWalkMatchesGit`, which compares the tool's walk with Git and is skipped where Git is not installed.

## Review and land

1. **Open a focused PR:** what changed and why, the checks you ran with their results, and what is not verified. Tests and smokes show that the code works; they are not a real agent run or a release.
2. **Get one independent review of the complete diff,** by someone who did not write it: a person, or a fresh agent that has not seen your reasoning. Fix what it finds, and review again if the fixes are substantial.
3. **Merge the reviewed head:** with a merge commit and the head matched, `gh pr merge NUMBER --merge --match-head-commit SHA`, without bypassing branch protection. Then read Main back.

**Real agent runs.** The skills of the standard, `markitect-assistant`, `markitect-capture`, `markitect-architect`, `markitect-finish-planning`, `markitect-build`, `markitect-review`, `markitect-analyze`, `markitect-accept`, `markitect-observe`, `markitect-create-observer`, `markitect-triage`, `markitect-record`, `markitect-brief` and `markitect-housekeeper`, start real agents, apart from `markitect-capture`, `markitect-finish-planning` and `markitect-create-observer`, which run inside the session that calls them. Run them only in a project set up with `init` that is a committed test project, in your own Claude Code session or a worktree of that project, bounded, and with the product owner's word. Before a run, name the project and its source commit, the limits, when to abort and the cleanup; after it, report what happened, failures included, in the PR it informs. The tool's commands start no agent and run anywhere.

**Releases** follow [Releasing](docs/releasing.md) ([DEC-153](docs/decisions.md#dec-153-markitect-holds-the-work-init-lays-out-templates-and-skills-and-the-plugin-fetches-its-own-version)). There is no deployment.

## Reporting

Report defects and propose changes in [GitHub Issues](https://github.com/Glacius-Labs/Markitect/issues); [SUPPORT.md](SUPPORT.md) says where questions go, and [SECURITY.md](SECURITY.md) how to report a vulnerability privately.
