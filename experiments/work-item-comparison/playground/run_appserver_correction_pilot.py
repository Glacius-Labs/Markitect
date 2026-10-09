"""Execute only the newly ordered one-time App Server enum correction trajectory."""
import argparse
from datetime import datetime, timedelta
import json
from pathlib import Path
import run_conventional_pilot as pilot


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--plan", required=True, type=Path)
    parser.add_argument("--destination", required=True, type=Path)
    parser.add_argument("--original-pilot", required=True, type=Path)
    args = parser.parse_args()
    plan = json.loads(args.plan.read_text(encoding="utf-8"))
    original = json.loads((args.original_pilot / "pilot-binding.json").read_text(encoding="utf-8"))
    expiry = datetime.fromisoformat(plan["overallExpiresAt"])
    if (plan.get("id") != "conventional-appserver-enum-correction-pilot-20261009" or
        plan.get("execution_authorized") is not True or
        plan.get("trajectories") != [{"case":"readinglog", "backend":"codex-app-server"}] or
        plan.get("startAllowance") != 64 or plan.get("sharedOriginalAllowance") != 256 or
        plan.get("jobWallSeconds") != 14400 or plan.get("turnWallSeconds") != 5400 or
        plan.get("model") != "gpt-6-luna" or plan.get("effort") != "high"):
        raise ValueError("not the explicit one-time correction order")
    if datetime.fromisoformat(original["overallExpiresAt"]) < expiry:
        raise ValueError("cannot extend original overall window")
    if pilot.now() + timedelta(hours=4) > expiry:
        raise ValueError("full four-hour remaining window is required at start")
    destination = args.destination.resolve()
    if not args.destination.is_absolute() or destination.exists() or pilot.ROOT in destination.parents:
        raise ValueError("fresh external destination required")
    if pilot.git(pilot.ROOT, "status", "--porcelain"):
        raise ValueError("source must be clean and committed")
    destination.mkdir(parents=True)
    source = pilot.git(pilot.ROOT, "rev-parse", "HEAD")
    pilot.save(destination / "pilot-binding.json", {"plan":plan, "planSha256":pilot.digest(args.plan),
        "sourceCommit":source, "controllerSourceSha256":pilot.digest(__file__),
        "originalPilotBindingSha256":pilot.digest(args.original_pilot / "pilot-binding.json"),
        "executable":original["executable"], "executableSha256":pilot.digest(original["executable"]),
        "version":original["version"], "overallExpiresAt":expiry.isoformat(), "startedAt":pilot.now().isoformat()})
    result = pilot.trajectory("readinglog", "codex-app-server", destination, plan,
        Path(original["executable"]), original["version"], source, expiry)
    pilot.save(destination / "pilot-results.json", {"trajectory":result,
        "interpretation":"one-time runtime correction; original failures retained; no comparative winner"})
    print(json.dumps({"event":"correction-pilot-ended", "status":result["status"]}), flush=True)


if __name__ == "__main__":
    for stream in (pilot.sys.stdin, pilot.sys.stdout, pilot.sys.stderr):
        if hasattr(stream, "reconfigure"): stream.reconfigure(encoding="utf-8")
    main()
