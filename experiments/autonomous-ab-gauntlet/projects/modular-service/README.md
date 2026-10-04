# Modular service A/B seed

This directory is experiment-control material. `arm-a/` and `arm-b/` are paired
actor seeds; task cards are shared in `task-set.yaml`. Keep `oracle/` out of
actor workspaces. The owner freezes and reviews these artifacts before scoring.

Both arms begin with the same Go application, tests, architecture gate, CI, and
human-authored baseline guidance. Arm B adds a Markitect Project, typed
architecture resources, a workflow and Skill, declared source ownership, and
generated Markdown/Codex context. Its public checks use the frozen v0.13.0 CLI
and the matching frozen artifact-coverage helper through PATH, plus the local
package fixture; no network access is needed.

A standalone parallel overlay is in `parallel-task-set.yaml`; the 12 sequential task cards stay in `task-set.yaml`. The evaluator accepts main tasks `01`–`12` and parallel tasks `P01`/`P02`, with `-Repo`, `-Task`, `-Output`, `-Base`, and for parallel forks `-PriorTasksThrough 06`. `oracle/` contains frozen evaluator code and hidden vectors.

Each actor card has an 1800-second maximum to cover the observed native readiness check and up to two fresh repair attempts. This is a task timeout, not a claim about human attention or completed work duration.
