"""Save pure serializer outputs and exact pinned Host source; no role launch."""
import hashlib
import json
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
AREA = Path(__file__).resolve().parent
sys.path.insert(0, str(ROOT / "runtime"))
from fixtures.government_positive import deterministic_delegate as fixture
from test_government_response_serialization import invocation


def write_new(path, raw):
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("xb") as stream:
        stream.write(raw)
    return {"path": str(path.resolve()), "sha256": hashlib.sha256(raw).hexdigest()}


samples = []
for name, phase, options in (
        ("executor-proposed", "execute", {}),
        ("review-passed", "review", {}),
        ("review-failed", "review", {"good_candidate": False}),
        ("review-incomplete", "review", {"subjects": False}),
        ("vote-assent", "vote", {}),
        ("vote-objection", "vote", {"good_candidate": False}),
        ("vote-incomplete", "vote", {"identity": False})):
    request = invocation(phase, **options)
    input_pin = write_new(AREA / "wire-samples" / (name + ".invocation.json"),
                          (json.dumps(request, sort_keys=True, separators=(",", ":")) + "\n").encode())
    output_pin = write_new(AREA / "wire-samples" / (name + ".response.json"),
                           fixture.response_bytes(request, phase))
    samples.append({"case": name, "phase": phase, "input": input_pin, "serializedOutput": output_pin,
                    "classification": "source-derived pure-function example; no process invocation or Host admission"})
source = "04e225d5caee78c2a198607143863fca1e829750"
sources = []
for name in ("internal/host/agentexec/types.go", "internal/host/agentexec/runner.go",
             "internal/host/agentexec/json.go", "internal/host/government/execution/run.go"):
    raw = subprocess.check_output(["git", "show", source + ":" + name], cwd=ROOT)
    blob = subprocess.check_output(["git", "rev-parse", source + ":" + name], cwd=ROOT, text=True).strip()
    pin = write_new(AREA / "pinned-host-source" / name, raw)
    sources.append({"sourceCommit": source, "sourcePath": name, "gitBlob": blob, "snapshot": pin})
manifest = {"classification": "offline serializer evidence only", "sourceCommit": source,
            "sourceSnapshots": sources, "wireSamples": samples,
            "newNativeStarts": 0, "newWrapperStarts": 0, "newDelegateProcesses": 0,
            "newProviderRuns": 0, "newMetadataSessions": 0, "newStudyCells": 0}
write_new(AREA / "offline-evidence-manifest.json",
          (json.dumps(manifest, sort_keys=True, separators=(",", ":")) + "\n").encode())
print(json.dumps({"wireSamples": len(samples), "pinnedSourceBlobs": len(sources), "newRoleOrProductProcesses": 0}))
