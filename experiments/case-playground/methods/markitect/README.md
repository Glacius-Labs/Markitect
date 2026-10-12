# Markitect method setup

What `playground/methods.py` does for `method: markitect`, inside the container,
before the agent starts. It runs the product's own onboarding the way the product
docs describe it and installs nothing else. It does not author a business model: the
initial model holds only the product's root Manager, and the agent models the project
as part of its normal work.

All commands run as the `agent` user in `/work/<case>`. Every command, exit code,
duration and raw output is recorded in `results/setup/`.

This file is for people. It is not staged into the container, so the Markitect agent
never reads these notes.

## Steps

1. **Binary.** Copy `/in/bin/markitect` (a Linux build of the pinned commit, made by
   the host) to `/usr/local/bin/markitect` and record its SHA-256.
2. **Inner native Codex, when needed.** For each selected Markitect role using Codex
   without an explicit `providerExecutable`, setup locates the native binary in the
   pinned CLI package. `command -v codex` is the npm Node launcher (a symlink to a `#!`
   script), while `config` accepts only a direct executable; setup therefore uses the
   package's ELF binary under `vendor/.../bin/codex`. This inner executable lookup is
   skipped when every role uses `process`, regardless of outer agent kind. A Codex outer
   agent separately uses its CLI and login as the outer runner.
3. **Feature branch.** Product writes refuse `main`/`master`, so the setup works on
   `markitect-setup`.
4. **Init.** `markitect init --repo R --name <case>` (preview), then the same
   with `--expect <digest> --write`. The harness checks that the write returns
   the preview digest.
5. **Onboard.** Onboard the providers actually used: `codex` for a Codex/fake outer
   agent or any selected Codex role, `claude` for a Claude outer agent with all three
   roles on `process`, and `both` for a Claude outer agent with at least one Codex role.
   Preview `markitect onboard --repo R --provider <provider>`, then write with
   `--expect <digest> --write`. This merges the corresponding guidance into `AGENTS.md`
   and adds the matching `.agents/skills/*` and `.markitect/workflows/model-first.md`.
6. **Runtime setup.** For the Codex default configuration, run
   `markitect config --repo R --provider codex --model <model> --effort <effort>
   --provider-executable <native codex>
   --input-micros-per-million 1000000 --output-micros-per-million 1000000
   --max-cost-micros <totalSeconds x 50000>` (preview), then the same with
   `--expect <editPlan.digest> --write`. When role profiles are present, setup supplies
   them through `--input <role-options.json>` using the existing role-options contract;
   with three explicit process profiles, their executables come from that input and no
   inner Codex binary lookup is needed.
   It records the input, preview and write commands and the hash of the resulting
   `.markitect/runtime.yaml`; that written file supplies each role's executor, model,
   effort, command, arguments, effective cost mode and explicitly requested process
   provider version. A mismatch blocks setup. Codex roles and Codex/fake outer agents
   use one of the supported native Codex efforts; an unused shared `innerEffort` does
   not need to be Codex-supported when all roles have explicit profiles. See the
   [manifest role profiles](../../README.md#manifest-schema-1).
7. **Commit.** "Install Markitect project workflow", fast-forward the start branch and
   delete `markitect-setup`.
8. **Observation.** `markitect check --repo R`. Right after setup it exits 1:
   whole-repository coverage is not conforming because the root Manager owns no
   artifacts yet. This is recorded, not treated as a failure.
9. **MCP server for the outer Codex:** `[mcp_servers.markitect]` with command
   `/usr/local/bin/markitect` and args `mcp --repo /work/<case>`. This is the
   config-file form of the documented `codex mcp add markitect -- markitect mcp
   --repo <root>`.

If a product command fails, setup returns `blocked` with the product's error and the
run stops. That is a product finding; the harness does not work around it.

## Budget values

`config` requires caller-supplied cost weights and a positive cost cap; they are
an estimate, not a price or a billing limit. We weight every token as one micro-unit
(1,000,000 per million tokens), so the product's cost counter equals its token count.
The cap is `limits.totalSeconds x 50,000` (720 million for the 4-hour example), high
enough that it cannot bind before the wall-clock limit that both methods share.

## Product constraints at commit 669cecd2

These are historical constraints of the product commit named in the heading, not the
current playground contract:

- Setup accepts only model `gpt-6-luna`, effort `high` and Codex CLI `0.162.0`.
- Setup needs the onboarding output first (it pins the generated Codex instructions
  and the Codex binary by digest in `runtime.yaml`).
- Inner Manager, review and verifier roles run Codex App Server with the inherited
  environment, so they use the agent's Codex home and login.
- Files generated by onboarding and init fail `git diff --check` (blank line at end of
  `.agents/skills/*` files, trailing spaces in `docs/markitect/project.md`). The setup
  records this and commits anyway; the agent meets the same issue when its project
  rules ask for `git diff --cached --check`.

## Current role-profile contract

The manifest may define separate `markitect.roles.manager`, `.reviewer` and `.verifier`
profiles. Each explicit profile selects a provider, a full versioned model ID and effort;
provider-specific executable, arguments, version and cost mode follow the manifest
rules above. Roles without profiles use `markitect.innerModel` and `markitect.innerEffort`,
or the Codex outer model and effort. A Claude outer agent must specify the shared inner
model and effort unless all three role profiles are explicit. Runtime Codex use is
conditional on the selected profiles: a missing role profile defaults to Codex, while
three explicit process profiles need no native Codex executable or login. Codex effort
validation applies to each selected Codex role and to Codex/fake outer agents; an unused
generic `innerEffort` may be outside that set.

Setup reuses the existing `markitect config --input` role-options contract. It records
the input file, preview/write commands and the written runtime YAML hash, then reports
each role's executor, model, effort, command, arguments and effective cost mode, plus an
explicitly requested process provider version. A mismatch blocks setup. This is
configuration evidence only: it does not show that a provider used the requested
effective model, certify Markitect's role lifecycle, or authorize a paid provider run,
study or protocol. Existing image pins and inventory remain the source for the installed
tools.

## Final conformance

`assess.final` copies the frozen `main` into a scratch Git repository and runs
`markitect check --repo <scratch>` (as the agent user, with a fresh empty HOME). Exit code, report status and coverage
conformance are recorded as method conformance, separate from the public checks.

## Sources (read at commit 669cecd2)

- `docs/project-workflow.md`: onboard, then register the MCP server; preview/write
  with returned digests.
- `docs/project-operations.md`: `mcp --repo ABSOLUTE_ROOT`, setup rules, cost
  weights.
- `docs/provider-adapters.md`: `codex mcp add markitect -- ... mcp --repo ...`.
- `docs/usage.md` ("Initialize a project") and `docs/onboarding.md`: init preview, then
  write, on a feature branch.
- `src/internal/host/projectcli/options.go`, `run.go`, `mcp.go`, `operations.go`: flags,
  `--expect` rules, JSON output and exit codes.
- `src/internal/host/projectsetup/setup.go`: native executable rules, model/effort/
  version checks, runtime defaults.
- `src/internal/host/projectwork/init.go` and `src/internal/host/write_safety.go`:
  init files and the protected-branch rule.
