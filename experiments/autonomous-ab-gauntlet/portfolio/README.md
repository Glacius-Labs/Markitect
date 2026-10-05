# Post-run portfolio

The versioned validator in `v1` accepts an explicit manifest of already-produced analysis reports. It verifies each report's exact bytes, report schema, declared analysis-source hash, freeze digest, and arena/report identity, then copies only manifest-designated task rows into a descriptive YAML portfolio record. This follows the experiment population contract in `postrun-population.md`: 216 planned cells, split into 144 development cells and 72 reserved holdout cells. The planned denominator is reported separately from designated trajectories and observed rows.

The manifest is an exact index, not a query language. Each trajectory names its `arena_id` indirectly through one report entry and binds one project, arm, trial, run ID, run kind, and explicit task-ID list. Duplicate report, trajectory, or task identities are refused. Multiple reports for an arena are allowed only for nonduplicate observations: the global observation identity is arena, freeze, run ID, project, arm, trial, task ID, and run kind. Repeated run IDs in different arenas remain separate report/trajectory records. Sequential trajectories require one run identity per project/arm/trial across all reports for the same arena and freeze; forks and integrations remain separate by run kind and run ID. Missing known task rows remain explicit unavailable slots; unknown and holdout task IDs are refused. Initial/final captures and derived snapshots are never expanded into extra task rows. Output counts distinguish manifest-designated trajectories, report-backed identities, unavailable identities, selected task slots, and observed rows; none is labeled an independent trial or actor-attempt count.

`portfolio/v1` does not collect actor data, invoke validators or evaluators, or derive a score, pass rate, comparison eligibility, or winner. It preserves each selected raw row and all report-level exclusion metadata. Excluded report/task rows remain `excluded-descriptive`; raw statuses are never rewritten, and the portfolio-level comparison eligibility flag is always false. All other rows are also descriptive until a separately reviewed comparison contract exists.

Integration Task08 retains the source report's expected-set and escalation fields inside `raw_row`, but labels both interpretations `unavailable-integration-card`. The static analysis contract audit found that the existing analysis card selection can use ordinary Task08 expectations for an integration run. This tool does not repair that source or treat those values as integration evidence.

The finite task allowlist includes only ordinary tasks 01–08 and fork tasks P01/P02. Holdout IDs and unknown identities cannot be designated. The validator trusts the pinned analysis report's contents and does not prove that the report's own source collection was complete or that its claims match an actor's consumed inputs. It does not reconcile reports into one pooled denominator. The manifest and report hashes provide byte identity, not authorization or human acceptance. Manifest completeness and outcome-independent trajectory selection require separate review; this CLI never chooses reports or trajectories from their outcomes.

Identity components use a restricted ASCII name form (`A-Z`, `a-z`, digits, `.`, `_`, and `-`, with an alphanumeric first character). This keeps the finite tuple-key representation unambiguous; path-valued fields remain separate and are not identity components.

The manifest records one or more immutable report inputs and exact trajectory selectors:

```yaml
schema: autonomous-ab-portfolio-manifest/v1
reports:
  - arena_id: source-arena-a
    freeze_digest: <64-hex-freeze-digest>
    report_id: report-a-v1
    report_path: C:/evidence/report-a.yaml
    report_sha256: <64-hex-report-digest>
    report_schema: autonomous-ab-analysis/v2
    report_source_version: analysis-source-sha256:<64-hex-analysis-source-digest>
    trajectories:
      - trajectory_id: arena-a-run-01
        project: modular-service
        arm: A
        trial: 1
        run_id: modular-service-a-t01
        run_kind: sequential
        task_ids: ["01", "02", "03", "04", "05", "06", "07", "08"]
```

Example invocation:

```powershell
go run ./experiments/autonomous-ab-gauntlet/portfolio/v1 --manifest <portfolio-manifest.yaml> --out <new-portfolio.yaml>
```

Output creation is exclusive. Synthetic fixture tests exercise the selector and refusal boundaries without reading any arena.
