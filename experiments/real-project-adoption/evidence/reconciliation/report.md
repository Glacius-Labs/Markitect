# Reconciliation evidence

The lifecycle used the published Markitect v0.12.0 Windows binary (SHA-256 `b03424560caa460322e3580785d6abc9dfbf2137df878e03ddedcf76b3a8fa47`) with the configured built-in `markitect-render` adapter. Every command record in this directory includes argv, working directory, exit code, complete stdout/stderr, and SHA-256 file-tree maps before and after the command. The source-tree hash excludes `.git`, `.artifacts`, `bin`, and `obj`.

| Step | Exit | Result |
|---|---:|---|
| observe | 1 | `status: drift`; source-tree hash stayed `cd66b0556f5cd98fd3bb6f6ec29541e4e55887d488922d291c6715ee99caa972` |
| plan-1 | 0 | Complete plan; same unchanged source-tree hash |
| plan-2 | 0 | Byte-identical stdout to plan-1; SHA-256 `b9ccd3969cd7dfd8efcbf535bf9b10478d14fa879a3c237280c5a8ac2d7f2e0e` |
| apply-no-write | 2 | Rejected with `reconcile apply requires explicit --write`; unchanged source-tree hash |
| stale-plan-apply | 2 | Rejected with `reconciliation plan is stale or was modified`; attempted against temporary Skill hash `6ebf373aa409472afdc50d576ec7f76a02ff082c864714e8dd5caa5c2ddba50e` |
| plan-after-restore | 0 | Recomputed from restored canonical bytes; complete |
| apply-write | 0 | `status: applied`; source-tree hash changed from `cd66b0556f5cd98fd3bb6f6ec29541e4e55887d488922d291c6715ee99caa972` to `b789f773ddad37a3917b273f59369ebd278d8cad30c9e0b063817a1e4658ff20` |
| verify | 0 | `status: verified`, exactly 72 operations/outputs; source-tree hash unchanged after apply |
| plan-converged | 0 | `status: complete`, `operations: []`; source-tree hash unchanged |

The canonical Skill description bytes were restored exactly: original SHA-256 `e3d72b4a231b35f781acfaaf1958ffcf7434993f58a0f61852b82dac851a6ef3`, restored SHA-256 identical. See `stale-plan-restoration.json` and the original-byte backup `skill-original.bin`. Plan stdout equality is recorded in `plan-determinism.json`.

The final branch commit is `fd94a689aab857704c3a4a45ac2abe0fc0e3185c`. The 646 upstream files declared in `source-manifest.json` still match exactly at 711,461 bytes; details are in `source-manifest-check.json`.
