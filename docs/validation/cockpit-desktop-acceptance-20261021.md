# Cockpit desktop app: first real runs — 21 October 2026

**Scope:** the owner approved one bounded acceptance of [DEC-066](../concepts/register.md#dec-066-the-cockpit-is-markitects-desktop-app-and-starts-and-stops-apply) with Sonnet, a small example and at most two runs. It exercises the cockpit's project list, guided setup, start, live observation and stop against a real provider, container engine and credential, which the slice records ([run control](cockpit-run-control-20261021.md), [projects and setup](cockpit-projects-20261021.md), [window and notifications](cockpit-window-notify-20261021.md)) left unproved. It is one run per condition, not a reliability measure.

## Setup

- **Host:** WSL Ubuntu with Docker 29.4.1, shared with another session's evaluation series; that session gave a slot with an idle engine for the duration.
- **Build:** Main `b9cdd6d893fa9487a09016a3520c3c3442660c90` (contains the cockpit hub, window, notifications, start/stop and projects), its Linux CLI and the Python example image `sha256:5f478c0f…` from the existing build of that commit; nothing was rebuilt.
- **Project:** the apply-trials case S-N (stockroom: Root and one Package, expected no-change Verified result), materialized from the same commit.
- **Cockpit:** `markitect cockpit --listen 127.0.0.1:7370` without project flags in WSL, with an isolated user configuration directory. It opened its app window on the Windows desktop. The page was driven from Windows in headless Chrome against the WSL address; the API calls for adding and setting up the project are the ones the forms send.
- **Setup through the cockpit:** runtime directory outside the repository, the Python image, a private copy of the dedicated runtime token, Agent model `claude-sonnet-5-5`, effort high, engine `/usr/bin/docker`, check profile `python-tests=/opt/markitect/bin/python`, store label `mk-cockpit`. Setup passed in 3.6 s, bound the image, agentd, Claude CLI and engine versions, wrote `runtime.yaml` and `run.json`, and the project reopened with runs enabled.

## Results

1. **Start, observe, finish.** The confirmation named the committed source `cef215771357`, execution time up to 40m, turn timeout 8m, 120 Agent turns, up to 12 Passes, no run-configuration deadline and Root's Execute profile `project-agent` with model `claude-sonnet-5-5`. *Start the run* started run `20261021T125155Z-6b50606a`; the Runs card, the running list and the inspector followed it live through Assess, Plan, Prepare, Implement and Verify. It ended **success / verified, publication no_change** after 17 min 50 s: 322 records, 30 Agent turns, 148 provider steps and 8.45 USD reported. This matches the case's expected result.
2. **Deliberate stop.** A second run `20261021T131037Z-8876730c` was started the same way and, after about two minutes in Plan Pass p2, stopped through the two-step *Stop the run*. It ended **incomplete, "the run was cancelled"**, with its report and 79 records written, no publication, 27 provider steps and 0.50 USD reported. The card then showed the last run as ended incomplete and offered a new start. No container with the run's store label remained and the repository stayed clean.

## Not established

- Notifications and their permission prompt with a person at the window (nobody granted permission during the runs).
- A model session from the cockpit on this setup, a project switch while something runs, and a run that changes files.
- Native Windows execution: the runs used WSL; Apply on Windows without WSL remains unproved.
- Owner use: the owner has not yet operated the cockpit end to end.
