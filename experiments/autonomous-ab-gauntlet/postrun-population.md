# Postrun population and identity

This document fixes how the final descriptive analysis identifies observations. It does not change the frozen product, tasks, oracles, actor budgets or eligibility decisions. The [analysis contract](analysis-contract.yaml) defines the dimensions. The [static audit](results/analysis-contract-static-audit-v1.json) explains why concatenating report totals would be incorrect.

The planned main population remains three projects, two arms, three trials and twelve tasks: 216 planned cells, including 144 development cells and 72 reserved holdout cells. A planned cell is not a completed task. A fresh measurement correction is another attempt at the same planned cell, not an additional independent trial. Report attempted trajectories separately from this planned denominator.

Every observation must carry `(arena identity, freeze digest, run ID, project, arm, trial, task ID, run kind)`. Neither run ID nor project/arm/trial alone identifies an observation across measurement corrections. Each source report is pinned by exact bytes and analysis-source identity. All reports and exclusions remain available; the portfolio must not discard negative history.

The following development trajectories are designated independently of their eventual outcomes:

| Project | Designated source arena | Main development cells | Treatment |
|---|---|---|---|
| Engineering operations | `r1-ops-oracle-v3`, freeze `a615190fb253cc8ff0ebcb3c02573aa9328e712d67c9f24e985610d9441b8dd6` | A/B trials1–3, tasks01–08 | Fresh measurement-corrected trajectories. Terminal availability and any later explicit exclusion remain binding; designation does not imply eligibility or success. |
| Modular service | `r1-service-oracle-v3`, freeze `1daa5bb43552c0437bcfd085d8afa4916094f90f03e3370f1ab2ecabdc339ee9` | A/B trials1–3, tasks01–08 | Trial1 is controller-excluded in both arms. Trials2/3 remain designated regardless of outcomes. Trial1's passing A checkpoint remains descriptive. |
| Vertical slices | `r1-paths-v3-ddd-t1-control-rerun`, freeze `4faee78795986559e91a0e233bd7321a9cf48526d331edba621ca98eee11638a` | A/B trials1–3, tasks01–08 | Trial1 is the fresh same-freeze replacement for the original excluded pair. Trials2/3 have explicit controller/containment exclusions; unprepared and unresolved tasks remain unavailable. Component evidence in the original arena must be joined explicitly, never substituted for a missing task. |

Earlier arenas are historical descriptive populations, including excluded or unavailable attempts. A final report must list their coverage and retained negative findings; it must not pool them into the designated paired comparison. If another measurement defect requires a new cohort, record a new explicit designation and paired exclusion before dispatch. Do not choose a favorable trajectory after comparing outcomes.

Sequential tasks, independent forks and mechanical integrations are separate populations. Initial and final captures are repeated observations of one boundary. A mechanical integration does not become an additional task agent merely because it has a native-format record; no actor response or model call is invented. Ordinary task08's expected-set or escalation values do not establish the truth of a P01/P02 integration. Preserve raw values and mark that interpretation unavailable unless a separately bound integration contract supplies it.

Apply exact freeze-bound exclusions before comparative interpretation. Retain raw oracle/helper status alongside excluded eligibility. An oracle pass cannot rescue protocol-invalid, timed-out, controller-excluded or unavailable evidence. Unknown evidence does not mean zero violations. Missing Context/Impact output is unavailable; separately reconstructed context cannot prove an actor consumed it.

Holdout tasks09–12 remain closed until the explicit global product fix/no-fix decision. This population note does not open that gate. Any later holdout comparison must identify its candidate, source arena, baseline trajectory and accumulated start state separately. No productivity, human-attention, token-saving or production claim follows from a counted synthetic task.
