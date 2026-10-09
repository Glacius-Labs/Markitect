"""One-time input authoring, not a nest.py rerun or a dispatch validation."""
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
from datetime import datetime, timezone

PACKET = Path(__file__).resolve().parents[1]
PUBLIC = PACKET / "public"
DEST = Path("C:/Users/Consiliari/Documents/Luna-Work-Item-Nests-20261009")

def git(path, *args):
    return subprocess.check_output(["git", "-C", str(path), *args], text=True, encoding="utf-8", timeout=45).strip()

def commit(path, message):
    git(path, "add", ".")
    git(path, "diff", "--cached", "--check")
    git(path, "-c", "user.name=Study Seed", "-c", "user.email=study@example.invalid", "commit", "-m", message)

def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def files(path):
    return {p.relative_to(path).as_posix(): {"sha256": digest(p), "bytes": p.stat().st_size}
            for p in sorted(path.rglob("*")) if p.is_file() and ".git" not in p.relative_to(path).parts}

def write(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2, ensure_ascii=False) + "\n", encoding="utf-8", newline="\n")

DEST.mkdir(parents=True, exist_ok=True)
manifest = {"kind": "Input authoring only, no corrected nest invocation or success claim",
            "createdUtc": datetime.now(timezone.utc).isoformat(), "seeds": [], "cells": [],
            "actuals": 0, "markitectInstallation": "NOT RUN", "dispatchValidation": "BLOCKED after 2 failed batches"}
for case in ("roombook", "readinglog"):
    seed = DEST / "seeds" / case
    if not seed.exists():
        shutil.copytree(PUBLIC / "common", seed)
        shutil.copytree(PUBLIC / "cases" / case, seed, dirs_exist_ok=True)
        subprocess.run(["git", "init", "-b", "main", str(seed)], check=True, capture_output=True, timeout=45)
        git(seed, "config", "core.autocrlf", "false")
        commit(seed, f"Author shared {case} seed and public trajectory")
    seed_commit = git(seed, "rev-parse", "HEAD")
    bundle = PACKET / "evidence" / f"{case}-seed.bundle"
    if not bundle.exists():
        git(seed, "bundle", "create", str(bundle), "--all")
    manifest["seeds"].append({"case": case, "repo": str(seed), "commit": seed_commit,
                             "tree": git(seed, "rev-parse", "HEAD^{tree}"),
                             "files": files(seed), "bundleSha256": digest(bundle)})
    stations = json.loads((seed / "STATIONS.json").read_text(encoding="utf-8-sig"))["stations"]
    for arm in ("conventional", "markitect"):
        repo = DEST / "cells" / f"{case}-{arm}"
        state_dir = DEST / "state" / f"{case}-{arm}"
        if state_dir.exists():
            raise SystemExit("Refuse replacing an already authored cell state")
        if not repo.exists():
            subprocess.run(["git", "-c", "core.autocrlf=false", "clone", "--no-hardlinks", str(seed), str(repo)], check=True, capture_output=True, timeout=45)
            git(repo, "remote", "remove", "origin")
            git(repo, "config", "core.autocrlf", "false")
            git(repo, "checkout", "-b", "work/backlog")
            if arm == "conventional":
                with (repo / "AGENTS.md").open("a", encoding="utf-8", newline="\n") as handle:
                    handle.write("\n" + (PUBLIC / "conventional/AGENTS.fragment.md").read_text(encoding="utf-8"))
            else:
                shutil.copytree(PUBLIC / "models" / case, repo, dirs_exist_ok=True)
            shutil.copytree(PUBLIC / "native", repo / ".codex")
            write(repo / ".study/station.json", stations[0])
        else:
            # Only recover the exact partial input-authoring cell, never an actor's work.
            if case != "roombook" or arm != "conventional" or git(repo, "rev-parse", "HEAD") != seed_commit:
                raise SystemExit("Unexpected pre-existing repository; preserve without mutation")
            for path in repo.rglob("*"):
                if path.is_file() and ".git" not in path.relative_to(repo).parts:
                    text = path.read_text(encoding="utf-8-sig")
                    path.write_text(text.replace("\r\n", "\n"), encoding="utf-8", newline="\n")
        commit(repo, f"Author {arm} case inputs; no product installation or app work")
        prepared = git(repo, "rev-parse", "HEAD")
        state = {"case": case, "arm": arm, "repo": str(repo), "stateDir": str(state_dir),
                 "seedCommit": seed_commit, "preparedCommit": prepared,
                 "preparedTree": git(repo, "rev-parse", "HEAD^{tree}"),
                 "createdUtc": datetime.now(timezone.utc).isoformat(), "status": "prepared",
                 "sessions": [], "interventions": [], "productReadiness": None, "initialSetup": [],
                 "trialGrant": None, "sessionId": None, "deadlineUtc": None,
                 "implementationDeadlineUtc": None, "evaluationDeadlineUtc": None,
                 "tokens": None, "cost": None, "servingModel": None,
                 "stationIndex": 0, "stations": stations, "snapshots": []}
        write(state_dir / "state.json", state)
        manifest["cells"].append({"case": case, "arm": arm, "repo": str(repo), "state": str(state_dir),
                                  "preparedCommit": prepared, "files": files(repo),
                                  "main": git(repo, "rev-parse", "main"),
                                  "status": git(repo, "status", "--porcelain"), "installation": "NOT RUN"})
write(PACKET / "evidence/prepared-inputs.json", manifest)
print(json.dumps({"seeds": len(manifest["seeds"]), "cells": len(manifest["cells"]),
                  "destination": str(DEST), "actuals": 0, "dispatchValidation": "BLOCKED"}, indent=2))
