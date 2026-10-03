"""Freeze operator inputs. Does not run agents or copy hidden oracles into checkouts."""
import argparse
import hashlib
import json
import io
import subprocess
import zipfile
from pathlib import Path

EXPERIMENT = Path(__file__).resolve().parent
REPO = EXPERIMENT.parent.parent
V1 = "fd94a689aab857704c3a4a45ac2abe0fc0e3185c"
V2 = "a0f89e7c0ce43a1be556c17c41cd19f8e7f32d7f"
INTEGRATED = "a23db5f48854cbf7723789843aaa10ddf06cdb0b"


def sha(data):
    return hashlib.sha256(data).hexdigest()


def git(repo, *args):
    return subprocess.check_output(["git", "-C", str(repo), *args])


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--source-repo", type=Path, required=True)
    parser.add_argument("--tool", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    if args.output.exists():
        raise SystemExit("Refusing to replace an existing frozen manifest")
    # Archive fixed trees once; avoid one native Git process per blob on Windows.
    inputs = ("src", "docs/architecture-decision-log", "README.md", "LICENSE")
    trees = [zipfile.ZipFile(io.BytesIO(git(args.source_repo, "archive", "--format=zip",
                                          revision, "--", *inputs)))
             for revision in (V1, V2)]
    paths = sorted(name for name in trees[0].namelist() if not name.endswith("/"))
    files = []
    for path in paths:
        content = trees[0].read(path)
        if content != trees[1].read(path):
            raise SystemExit(f"Source changed between task baselines: {path}")
        files.append({"path": path, "sha256": sha(content), "bytes": len(content)})
    if len(files) != 646:
        raise SystemExit(f"Expected the selected 646-file source, found {len(files)}")
    frozen = []
    for path in sorted(EXPERIMENT.rglob("*")):
        if not path.is_file() or path == args.output.resolve():
            continue
        if any(part in ("bin", "obj", "results", "evidence", "__pycache__")
               for part in path.relative_to(EXPERIMENT).parts):
            continue
        frozen.append({"path": path.relative_to(REPO).as_posix(), "sha256": sha(path.read_bytes())})
    for relative in (
        "experiments/real-project-adoption/checks/check-module-project-references.ps1",
        "experiments/real-project-adoption/checks/project-map.json",
        "experiments/real-project-adoption/evidence/source-manifest.json",
    ):
        frozen.append({"path": relative, "sha256": sha((REPO / relative).read_bytes())})
    oracles = [entry for entry in frozen if "/oracles/" in entry["path"]]
    if len(oracles) != 3:
        raise SystemExit("Freeze requires exactly three hidden task oracles")

    def asset(source, destination, arm, trials, phase="base"):
        path = EXPERIMENT / source
        return {"source": path.relative_to(REPO).as_posix(), "destination": destination,
                "sha256": sha(path.read_bytes()), "arm": arm, "trial": trials, "phase": phase}

    def context(kind, name, namespace, failing=False):
        arguments = ["context", "--repo", "{repo}", "--revision", "{candidate}",
                     "--namespace", namespace, "--kind", kind, "--name", name]
        if kind == "UseCase":
            arguments += ["--api-version", "architecture.mymeetings.example/v1alpha1"]
        if failing:
            arguments.append("--analyze-policy-failures")
        return {"args": arguments, "output": "context.yaml", "expectedExitCode": int(failing),
                "stdoutContains": ["policy-failure-analysis", "policyStatus: failed"] if failing else [],
                "scope": {"kind": kind, "name": name, "namespace": namespace}}

    preparation = []
    plan = "{externalEvidenceRoot}/render-plan.stdout.txt"
    for action in ("observe", "plan", "apply", "verify"):
        arguments = ["reconcile", "--repo", "{workspace}", "--action", action,
                     "--adapter", "markitect-render"]
        if action in ("apply", "verify"):
            arguments += ["--plan", plan]
        if action == "apply":
            arguments.append("--write")
        record = {"name": f"render-{action}", "executable": "{markitect}",
                  "args": arguments, "expectedExitCode": int(action == "observe")}
        if action == "apply":
            record.update(requiresPlanReview=True, reviewPlan="render-plan.stdout.txt")
        preparation.append(record)
    impact = {"args": ["impact", "--repo", "{repo}", "--base", "{base}",
                       "--revision", "{candidate}", "--analyze-policy-failures"],
              "output": "impact.yaml", "expectedExitCode": 1,
              "stdoutContains": ["policy-failure-analysis", "directPolicySubjectCount: 2"],
              "scope": {"comparison": "reconstructed exact v1 to v2 pin trees"}}
    prompt = """Work only inside this fresh checkout. Do not inspect parent directories, other trials,
operator files, hidden acceptance criteria, memories, or prior sessions. Do not write outside
this checkout or bypass the sandbox. The task below is the current authoritative request.
Use .tools/read.ps1 -Path <relative-path> for content reads so the experiment can meter them;
normal rg searches are allowed and separately recorded. The helper log is not a complete OS
I/O monitor. Do not alter the helper or supplied .agent-input files.
Use the existing governance supplied in this checkout. If .agent-input/context.yaml exists,
read that compiled task context first; also read .agent-input/impact.yaml when present.
Failed-policy analysis can produce useful output with exit 1: it does not mean compliance.
The local .tools/markitect.exe, when present, is the integrated candidate, not a release.
Keep existing governance coherent with the change; do not introduce a different mechanism.
Run focused checks and a Meetings Application build from src, honoring src/global.json.
The common bounded existing-test facade is available with:
dotnet test ../.tools/architecture-check/MeetingsArchitectureCheck.csproj
Run that command from src. It links the checkout architecture tests; hidden task oracles
are not in this facade. Do not edit the facade itself; edit project-owned tests when needed.
Use the same public NuGet source https://api.nuget.org/v3/index.json when restore is needed.
Do not publish, change unrelated modules, waive a policy just to inspect it, or create an API.
There is no interactive operator assistance. If blocked, preserve the failure and explain it.
Finish with changed files, checks and their exact outcomes, architecture decisions, missing
or irrelevant guidance, governance upkeep, and the limits of what was actually verified."""
    manifest = {
        "schemaVersion": 1, "experimentId": "agents-md-vs-markitect-2026-10-03",
        "sourceBundleSha256": "af9dbdc750f376b5250b61a451c1d124fe363fd87cce87684804b292149311cc",
        "sourceFiles": files,
        "sourceManifestSha256": sha(json.dumps(files, sort_keys=True, separators=(",", ":")).encode()),
        "frozenFiles": frozen,
        "oracleFreezeSha256": sha(json.dumps(oracles, sort_keys=True, separators=(",", ":")).encode()),
        "markitect": {"sourceCommit": INTEGRATED, "sha256": sha(args.tool.read_bytes())},
        "prompts": {"genericBoundary": prompt, "genericBoundarySha256": sha(prompt.encode())},
        "executionOrder": ["t1-simple", "t1-markitect", "t2-markitect", "t2-simple", "t3-simple", "t3-markitect"],
        "assets": [
            asset("guidance/AGENTS.md", "AGENTS.md", "simple", ["t1", "t2", "t3"]),
            asset("guidance/architecture-check/MeetingsArchitectureCheck.csproj", ".tools/architecture-check/MeetingsArchitectureCheck.csproj", ["simple", "markitect"], ["t1", "t2", "t3"]),
            asset("guidance/architecture-check/TestBase.cs", ".tools/architecture-check/TestBase.cs", ["simple", "markitect"], ["t1", "t2", "t3"]),
            asset("guidance/architecture-check/LICENSE", ".tools/architecture-check/LICENSE", ["simple", "markitect"], ["t1", "t2", "t3"]),
            asset("guidance/policy-v1.md", "docs/engineering-policy.md", "simple", ["t1", "t3"]),
            asset("guidance/policy-v2.md", "docs/engineering-policy.md", "simple", ["t2"]),
            asset("guidance/architecture-review.skill.yaml", "agent-guidance/architecture-review.skill.yaml", "markitect", ["t1"], "candidate")],
        "trials": [
            {"id": "t1", "revision": V1, "task": "experiments/agents-md-comparison/tasks/task1.md",
             "contextArgs": [context("Skill", "architecture-review", "engineering")],
             "preparationCommands": preparation},
            {"id": "t2", "revision": V2, "task": "experiments/agents-md-comparison/tasks/task2.md",
             "markitectBaseRevision": V1, "markitectOverlayRevision": V2, "markitectOverlayPath": "markitect.yaml",
             "contextArgs": [context("UseCase", "add-meeting-attendee", "architecture", True), impact]},
            {"id": "t3", "revision": V1, "task": "experiments/agents-md-comparison/tasks/task3.md",
             "contextArgs": [context("UseCase", "get-meeting-attendees", "architecture")]}]
    }
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8", newline="\n")
    print(json.dumps({"frozenFiles": len(frozen), "sourceFiles": len(files),
                      "manifestSha256": sha(args.output.read_bytes()),
                      "oracleFreezeSha256": manifest["oracleFreezeSha256"]}))


if __name__ == "__main__":
    main()
