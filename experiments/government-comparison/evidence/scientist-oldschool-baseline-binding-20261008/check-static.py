"""Read-only public setup consistency check; never import/execute study runners."""
import ast
import hashlib
import itertools
import json
from pathlib import Path
import re
import subprocess

AREA = Path(__file__).resolve().parent
ROOT = AREA.parents[1]
REPO = ROOT.parents[1]
BASE = "962bcd12ca3a940517f457a9fd89d44cf5d7c0d3"


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def git(*args, cwd=REPO):
    return subprocess.check_output(["git", "-C", str(cwd), *args])


binding = json.loads((ROOT / "public/oldschool-baseline-binding-v1.json").read_bytes())
assert binding["scientistBaseCommit"] == BASE
values = {}
for node in ast.parse((ROOT / "prepare.py").read_text()).body:
    if isinstance(node, ast.Assign):
        for target in node.targets:
            if isinstance(target, ast.Name) and target.id in {"ARMS", "CONDITIONS"}:
                values[target.id] = ast.literal_eval(node.value)
expected = set(itertools.product(values["ARMS"], values["CONDITIONS"]))
assert len(expected) == 6
assert {(c["arm"], c["condition"]) for c in binding["matrix"]} == expected
assert len(binding["matrix"]) == 6
assert all(c["status"] == "NOT RUN" and c["actorContext"] is None and
           c["isolatedCheckout"] is None and c["actualOutcome"] is None for c in binding["matrix"])

for item in binding["commonInputs"]:
    raw = (ROOT / item["path"]).read_bytes()
    assert digest(raw) == item["sha256"], item["path"]
    rel = (ROOT / item["path"]).relative_to(REPO).as_posix()
    assert raw == git("show", BASE + ":" + rel), rel
assert len([i for i in binding["commonInputs"] if i["path"].startswith("public/tasks/")]) == 6
rubric = json.loads((ROOT / "public/rubric.json").read_bytes())
assert [d["weight"] for d in rubric["scoring"]["result_quality"]["dimensions"]] == [30, 25, 20, 15, 10]
assert [d["weight"] for d in rubric["scoring"]["method_conformance"]["dimensions"]] == [20, 20, 25, 15, 20]
schema = json.loads((ROOT / "public/metrics.schema.json").read_bytes())
assert set(schema["properties"]["arm"]["enum"]) == set(values["ARMS"])
assert set(schema["properties"]["condition"]["enum"]) == set(values["CONDITIONS"])
profile = json.loads((ROOT / "public/measurement-profile-v2.json").read_bytes())
assert set(profile["arms"]) == set(values["ARMS"])
assert profile["unknownTokensAdmission"] == "block"
assert profile["unknownProviderRequestsAdmission"] == "permit-with-null-counter-and-gap"
resources = json.loads((ROOT / "public/resource-proposal.json").read_bytes())
assert set(resources["orderProposal"]) == {a + "-" + c for a, c in expected}
assert binding["sharedProfile"]["newOrExpandedLimits"] is False
assert binding["outcomeAndMethodWeightsChanged"] is False
assert not binding["fixtureValuesAreReferenceMeasurements"]
assert all(v == 0 for v in binding["newExperimentalConsumption"].values())
assert binding["liveStudyTrials"] == 0 and not binding["s1Complete"]

source = binding["canonicalSource"]
raw = git("show", source["sourceCommit"] + ":" + source["path"], cwd=source["repository"])
assert digest(raw) == source["sha256"]
assert git("rev-parse", source["sourceCommit"] + ":" + source["path"], cwd=source["repository"]).decode().strip() == source["gitBlob"]
section = (ROOT / "public" / source["sectionSnapshot"]).resolve().read_bytes()
assert digest(section) == source["sectionSha256"] and section in raw
brief = binding["oldschool"]["publicWorkflowBrief"]
assert digest((ROOT / brief["path"]).read_bytes()) == brief["sha256"]
assert brief["actualUsage"] is None and brief["actualActiveSeconds"] is None
assert brief["zeroConsumptionInstrumented"] is False
links = 0
for rel in ("README.md", "public/oldschool-baseline-binding-v1.md", "public/oldschool-workflow-brief-20261008.md"):
    path = ROOT / rel
    for link in re.findall(r"\]\(([^)]+)\)", path.read_text(encoding="utf-8")):
        if "://" in link or link.startswith("#"):
            continue
        assert (path.parent / link.split("#", 1)[0]).resolve().is_file(), (rel, link)
        links += 1
changed = git("diff", "--name-only", BASE).decode().splitlines()
allowed = {"experiments/government-comparison/README.md",
           "experiments/government-comparison/public/oldschool-baseline-binding-v1.md",
           "experiments/government-comparison/public/oldschool-baseline-binding-v1.json",
           "experiments/government-comparison/public/oldschool-workflow-brief-20261008.md"}
assert all(p in allowed or p.startswith("experiments/government-comparison/evidence/scientist-oldschool-baseline-binding-20261008/") for p in changed), changed
ledger = Path("C:/Users/Consiliari/Documents/Scientist-Probes/native-metadata-fixtures-20261008/native-starts.sqlite")
assert digest(ledger.read_bytes()) == "dd617d58a9021fce0b11482b740d78ccb5ddd143ff0c6fa4fbe08983af7f6705"
print(json.dumps({"package": binding["package"], "status": "passed",
                  "scope": "static AST/JSON/byte/link reads only; no runner imports or experimental starts",
                  "matrixCells": 6, "commonInputBindingsUnchanged": len(binding["commonInputs"]),
                  "rubricWeightsUnchanged": True, "sharedProfileUnchanged": True,
                  "localLinksChecked": links, "canonicalSourceCommit": source["sourceCommit"],
                  "ledgerUnchanged": True, "allCellsNotRun": True}, sort_keys=True))
