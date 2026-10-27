---
description: Markitect Apply (comparison arm P) - check, Assessment, Implement, Verify and guarded publication through the compiled Manager tree
allowed-tools: Bash, Read, Write, Workflow
---
Run one Markitect Apply in the current repository. You only connect deterministic steps; you make no domain decision and change no repository file yourself. This invocation authorizes the Workflow call below.

1. Run `python "${CLAUDE_PLUGIN_ROOT}/scripts/mkp.py" prepare --run-id "$MARKITECT_RUN_ID"`. If it exits non-zero, stop and answer with its output.
2. Call the Workflow tool with `name: "markitect:apply"` and `args` set to the complete JSON object that prepare printed, unchanged. If the tool does not know the name, use `scriptPath: "${CLAUDE_PLUGIN_ROOT}/workflows/apply.js"` with the same args. Wait until the workflow has finished.
3. Write the workflow's returned JSON value, exactly as returned, to the file `workflow-result.json` in the `stateDir` named by the prepare record.
4. Run `python "${CLAUDE_PLUGIN_ROOT}/scripts/mkp.py" publish --run-id "$MARKITECT_RUN_ID" --result "<stateDir>/workflow-result.json"`.
5. Answer with the JSON that publish printed and nothing else.
