# Harness task and validator contract

The external hidden evaluator command is:

    powershell -File oracle/evaluate.ps1 -Repo <arm-repository-root> -Task <two-digit-task-id> -Base <base-commit-sha> -Output <run-record.yaml>

The external evaluator owns the hidden executable oracle and drift vector. It is not implemented or stored in this seed. Invoke it against the acting arm's repository after each task and preserve its YAML output with the trial record; never copy evaluator code, task expectations, or comparison metadata into the actor checkout.

The harness presents the same prompt text from task-set.yaml to each arm in the same sequence. It creates separate fresh repositories from the committed arm seed and passes the same model, settings, permissions, tool versions, and environment. It records full base and candidate commits and keeps the task ID and arm identity with every YAML record. Expected classifications, escalation flags, path scopes, and affected-set labels in the task set are evaluator/harness metadata, not actor instructions.

The separate `../parallel-task-set.yaml` overlay defines P01 and P02. Each arm's parallel branch forks from that arm's retained main-run trial 1 state after task 06. P01 maps to main task 07; P02 maps to main task 08. These branches run independently and do not replace or reorder main tasks 07 and 08. A P01 evaluation expects only the completed 01–06 state plus P01's verification-check change; a P02 evaluation expects only 01–06 plus P02's runbook addition. Final integration evaluates both changes together against the cumulative frozen expectations for main tasks 01–08, with P01/P02 satisfying the same 07/08 intent rather than adding duplicate tasks.

Tasks 09–12 are sealed holdouts. Do not disclose their prompts or expected outcomes before their scheduled turn. Do not treat a passing Markitect check, literal project check, or evaluator result as human acceptance. Keep attempted repairs, failures, exclusions, missing measures, and decisions in persisted records.
