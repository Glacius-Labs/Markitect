"""Write one run configuration. Usage: ra_runconfig.py ATTEMPT LABEL SHA DEADLINE_RFC3339 [OWNER_REQUEST]

Without OWNER_REQUEST the run applies the committed Library model with the bound
owner decisions of the demonstration. With it the run gets that request and no
owner decisions: Extract and its round trip must not see decisions that name
the original model's Definitions."""
import json
import sys
from pathlib import Path

attempt, label, sha, deadline = Path(sys.argv[1]), sys.argv[2], sys.argv[3], sys.argv[4]
if len(sys.argv) > 5:
    request, decisions = sys.argv[5], []
else:
    request = "Apply the committed Library model."
    decisions = json.loads((attempt / "frozen" / "owner-decisions.json").read_text())
runtime = next(line.split(" ", 1)[0] for line in (attempt / "binding.txt").read_text().splitlines() if line.startswith("runtime_image="))
runtime = runtime.split("=", 1)[1]
# The workspace helper carries the same store label as the attempt's Agent containers.
store = next((line.split(":", 1)[1].strip() for line in (attempt / "host" / "host.yaml").read_text().splitlines()
              if line.strip().startswith("store:")), "ra-library")
config = {
    "formatVersion": 2,
    "repositoryIdentity": "ra-library-" + attempt.name,
    "productBuild": sha,
    "ownerRequest": request,
    "stateDirectory": str(attempt / ("state-" + label)),
    "deadline": deadline,
    "engineExecutable": "/usr/bin/docker",
    "workspaceHelper": {"name": "ra-ws", "image": runtime, "labels": {"markitect.store": store}, "user": "1000:1000",
                        "cpus": 1, "memoryBytes": 536870912, "pids": 128, "tmpBytes": 134217728, "network": "none"},
    "harvest": {"maxEntries": 50000, "maxBytes": 268435456},
    "credentials": {"ra-claude-token": {"provider": "claude-code", "mode": "dedicated-runtime-token", "path": str(attempt / "secret" / "claude-token")}},
}
if decisions:
    config["ownerDecisions"] = decisions
target = attempt / ("run-config-" + label + ".json")
target.write_text(json.dumps(config, indent=1))
print(target)
