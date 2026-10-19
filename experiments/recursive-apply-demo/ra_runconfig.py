"""Write one Apply's run configuration. Usage: ra_runconfig.py ATTEMPT LABEL SHA DEADLINE_RFC3339"""
import json
import sys
from pathlib import Path

attempt, label, sha, deadline = Path(sys.argv[1]), sys.argv[2], sys.argv[3], sys.argv[4]
runtime = next(line.split(" ", 1)[0] for line in (attempt / "binding.txt").read_text().splitlines() if line.startswith("runtime_image="))
runtime = runtime.split("=", 1)[1]
config = {
    "formatVersion": 2,
    "repositoryIdentity": "ra-library-" + attempt.name,
    "productBuild": sha,
    "ownerRequest": "Apply the committed Library model.",
    "ownerDecisions": json.loads((attempt / "frozen" / "owner-decisions.json").read_text()),
    "stateDirectory": str(attempt / ("state-" + label)),
    "deadline": deadline,
    "engineExecutable": "/usr/bin/docker",
    "workspaceHelper": {"name": "ra-ws", "image": runtime, "labels": {"markitect.store": "ra-library"}, "user": "1000:1000",
                        "cpus": 1, "memoryBytes": 536870912, "pids": 128, "tmpBytes": 134217728, "network": "none"},
    "harvest": {"maxEntries": 50000, "maxBytes": 268435456},
    "credentials": {"ra-claude-token": {"provider": "claude-code", "mode": "dedicated-runtime-token", "path": str(attempt / "secret" / "claude-token")}},
}
target = attempt / ("run-config-" + label + ".json")
target.write_text(json.dumps(config, indent=1))
print(target)
