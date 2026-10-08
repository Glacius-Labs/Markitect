# Assisted pilot frozen evidence

Candidate: `codex/pilot-assisted` at `fd94a689aab857704c3a4a45ac2abe0fc0e3185c` (`Adopt Markitect for assisted architecture pilot`). The candidate is committed and clean. The pinned published executable is Markitect 0.12.0; its binary SHA-256 and all captured argv/stdout/stderr/exit codes are recorded in `discovery-ledger.json` and the per-command JSON files.

The fixed snapshot digest is `594d527fe69074a9fd7357983ce7c7a8d50307812e638a517c939382673de8ef`; the normalized model digest is `sha256:d115aeef1780a8bdf9aaec2dca2d67c696b4892740c7e59830d6dcd035c89e94`. `context-skill.yaml` is the compiled selection `engineering/Skill/architecture-review` at the exact candidate SHA. It contains 26 input records: 11 typed resources, 3 definition inputs (Project, Package, and Domain), and 12 opaque project artifact files. Its context digest is `sha256:6f78259094ab85df7498aa78e6120521fb7a82ca865d00d116cd2a7319dfe1fa`.

`context-run-copy-me.yaml` compiles committed `context-run.yaml` for task `copy-me-query-convention` (`work-items/copy-me-query-convention.md`). It contains 38 input records: 11 typed resources, 3 definition inputs, 12 project artifact files, and 12 run records (the manifest, task file, and ten declared source files). Its context digest is `sha256:c9ec63e24bd1cf0fdf8b70a92a0e9479f86c4d2eccd6e6d938e157937b09c30b`. The compiled run includes the manifest's exact ten source declarations and Markitect's transitive context closure; breadth was not edited after compilation.

`model.yaml` and `check.yaml` both exited 0; model validation and check status are `passed`. Each command used `--revision fd94a689aab857704c3a4a45ac2abe0fc0e3185c`. All four commands left the live source-tree hash unchanged at `96694d5b1927993ee0cccd3aee1f388ec24e6269d1b8dc53a3aec4da4a9f115f`.

The upstream source manifest verified exactly: 646 of 646 files, 711,461 of 711,461 bytes, from upstream commit `91c8ef24b4cb6ef558c95d8267fa07d68c7059f8`. Package pins and their hashes are in `../setup/package-pins.json`.

See `reconciliation-report.md` for the observe/plan/apply/verify exits, read-only tree hashes, stale-plan rejection and exact Skill byte restoration evidence. Raw command records are adjacent JSON files; `capture-command.ps1` is the re-usable evidence collector.
