"""Explicit Docker smoke test of the study command: one provider-free study.

  python3 -B tests/smoke_study.py [--study S.json] [--keep]

Runs `python -m playground study` for a fake-agent study file (default
examples/study-fake-roombook.json: both arms, two roombook stations) with fake reviewers
and a throwaway Codex login `{"generation": 0, "secret": ...}` passed as --codex-auth.
The fake agent bumps the generation on every call, as a token refresh would. Checks: exit
0, the schedule order, both assessments, the comparison with matching fairness, a
complete study.json, the login carried over from run to run, the throwaway login nowhere
in the study folder and the source never written, the login folder removed and no
labelled container left, every run and assessment on the study's pre-registered evaluation
tree. Runs from a Git checkout whose evaluation files are committed and unchanged. Needs
Docker, and Git and Go for the Markitect arm (built from the checkout holding the
playground unless the study file names another); makes no model call. Not picked up by
unittest discovery.
"""
from __future__ import annotations

import argparse
import json
import platform
import re
import secrets
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
GENERATION = re.compile(r'"fakeLoginGeneration": (\d+)')


def files_containing(root: Path, needle: bytes) -> list[str]:
    return [path.relative_to(root).as_posix() for path in root.rglob("*")
            if path.is_file() and not path.is_symlink() and needle in path.read_bytes()]


def generations(run: Path, count: int) -> list[int | None]:
    found = []
    for number in range(1, count + 1):
        events = run / "results" / "stations" / f"S{number}" / "events.jsonl"
        match = GENERATION.search(events.read_text(encoding="utf-8")) if events.is_file() else None
        found.append(int(match.group(1)) if match else None)
    return found


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--study", type=Path, default=ROOT / "examples" / "study-fake-roombook.json")
    parser.add_argument("--keep", action="store_true", help="keep the study folder on success")
    args = parser.parse_args()

    study_path = args.study.resolve()
    data = json.loads(study_path.read_text(encoding="utf-8"))
    if data.get("agent", {}).get("kind") != "fake":
        parser.error("the smoke only runs studies with the fake agent (no model calls)")
    temp = Path(tempfile.mkdtemp(prefix="mpg-smoke-study-"))
    out = temp / "study"
    login = temp / "login" / "auth.json"
    login.parent.mkdir()
    secret = f"smoke-login-{secrets.token_hex(16)}"
    login.write_text(json.dumps({"generation": 0, "secret": secret}), encoding="utf-8")
    before = login.stat()
    command = [sys.executable, "-B", "-m", "playground", "study", str(study_path), "--out", str(out),
               "--codex-auth", str(login), "--fake-reviewers"]
    print(f"smoke: study {study_path}\nsmoke: out {out}", flush=True)
    done = subprocess.run(command, cwd=ROOT)

    failures: list[str] = []

    def check(ok: bool, message: str) -> None:
        print(f"  [{'ok' if ok else 'FAIL'}] {message}")
        if not ok:
            failures.append(message)

    print(f"smoke: study exit code {done.returncode}")
    check(done.returncode == 0, "the study exited 0")
    record = json.loads((out / "study.json").read_text(encoding="utf-8")) if (out / "study.json").is_file() else {}
    check(record.get("status") == "completed" and record.get("exitCode") == 0,
          f"study.json says completed, exit 0 ({record.get('status')}, {record.get('exitCode')})")
    steps = record.get("steps") or []
    runs = [s for s in steps if s.get("step") == "run"]
    first = data["firstArm"]
    arms = [first] + [arm for arm in data["arms"] if arm != first]
    suffix = {"conventional": "conv", "markitect": "mkt"}
    expected = [f"{data['id']}-p1-{suffix[arm]}" for arm in arms]
    check([s.get("id") for s in runs] == expected, f"runs in schedule order {expected}")
    count = data["stations"]
    check(all(s.get("status") == "completed" and s.get("exitCode") == 0 for s in runs),
          f"every run completed with exit 0 {[(s.get('status'), s.get('exitCode')) for s in runs]}")
    assess = [s for s in steps if s.get("step") == "assess"]
    check(len(assess) == len(runs) and all(s.get("exitCode") == 0 for s in assess)
          and all((out / "runs" / s["id"] / "assessment" / "report.json").is_file() for s in runs),
          "every run was assessed")
    comparison = out / "comparisons" / "p1.md"
    text = comparison.read_text(encoding="utf-8") if comparison.is_file() else ""
    here_text = f"Host platform: {platform.system()} {platform.machine()}."
    check("Fairness fields match" in text and f"{expected[0]}" in text and here_text in text,
          f"comparisons/p1.md compares the pair with matching fairness fields ({here_text})")
    pre = record.get("preRegistration") or {}
    tree = pre.get("evaluationTree") or ""
    check(pre.get("status") == "registered" and re.fullmatch(r"[0-9a-f]{40}|[0-9a-f]{64}", tree) is not None,
          f"study.json records the pre-registration ({pre})")
    hosts = [json.loads((out / "runs" / run_id / "host.json").read_text(encoding="utf-8"))
             if (out / "runs" / run_id / "host.json").is_file() else {} for run_id in expected]
    check(all((h.get("preRegistration") or {}).get("evaluationTree") == tree for h in hosts),
          "every run records the study's pre-registration")
    reports = [json.loads((out / "runs" / run_id / "assessment" / "report.json").read_text(encoding="utf-8"))
               if (out / "runs" / run_id / "assessment" / "report.json").is_file() else {} for run_id in expected]
    check(all((r.get("evaluation") or {}).get("source") == "registered"
              and (r.get("evaluation") or {}).get("tree") == tree
              and (r.get("rules") or {}).get("preRegistered") is True
              and (r.get("rules") or {}).get("reviewersIndependent") is True for r in reports),
          f"every assessment judged with the registered tree and independent reviewers "
          f"{[(r.get('evaluation') or {}).get('source') for r in reports]}")
    versions = record.get("versions") or {}
    here = {"system": platform.system(), "machine": platform.machine()}
    complete = (bool((versions.get("playground") or {}).get("commit")) and versions.get("python")
                and versions.get("hostPlatform") == here and (versions.get("docker") or {}).get("arch")
                and versions.get("go") and (versions.get("image") or {}).get("id")
                and re.fullmatch(r"[0-9a-f]{64}", (versions.get("markitect") or {}).get("sha256") or ""))
    check(bool(complete), f"study.json records the versions {versions}")
    checks = {c.get("check"): c.get("status") for c in (record.get("preflight") or {}).get("checks") or []}
    check((record.get("preflight") or {}).get("status") == "passed" and checks.get("image build") == "ok"
          and checks.get("markitect binary") == "ok", "the preflight passed and built the image and binary once")
    check(all(s.get("startedAt") and s.get("endedAt") and s.get("seconds") is not None for s in steps)
          and len(steps) == 2 * len(runs) + 1, "every step records its times, status and exit")
    parameters = record.get("parameters") or {}
    check(Path((parameters.get("markitect") or {}).get("sourceRepo") or "").is_absolute()
          and re.fullmatch(r"[0-9a-f]{40}", (parameters.get("markitect") or {}).get("commit") or "") is not None,
          "the resolved sourceRepo and full commit are recorded")
    manifests = sorted(path.name for path in (out / "manifests").glob("*.json"))
    check(manifests == sorted(f"{run_id}.json" for run_id in expected), f"one normalized manifest per run {manifests}")
    seen = [generations(out / "runs" / run_id, count) for run_id in expected]
    flat = [g for run in seen for g in run]
    check(flat == list(range(len(flat))), f"the login was carried over from run to run (generations {seen})")
    check([(s.get("login") or {}).get("promotion") for s in runs] == ["promoted"] * len(runs),
          f"each run's refreshed login became the working copy {[(s.get('login') or {}) for s in runs]}")
    leaked = files_containing(out, secret.encode("utf-8"))
    check(not leaked, f"the throwaway login is nowhere in the study folder {leaked[:5]}")
    after = login.stat()
    check(json.loads(login.read_text(encoding="utf-8")) == {"generation": 0, "secret": secret}
          and (after.st_mtime_ns, after.st_size) == (before.st_mtime_ns, before.st_size),
          "the source login was never written")
    copies = (record.get("logins") or {}).get("copies") or {}
    check(bool(copies.get("folder")) and not Path(copies["folder"]).exists() and copies.get("removed") is True,
          f"the login folder is gone ({copies.get('folder')})")
    check(record.get("logins", {}).get("codex") == str(login.resolve()), "study.json names the login path only")
    note = (record.get("logins") or {}).get("note") or ""
    check(note.startswith("Codex refreshed its login during the study") and str(login.resolve()) in note
          and "run `codex login`" in note, f"study.json tells to run `codex login` ({note})")
    check(not (Path.home() / ".markitect-playground" / "study.lock").exists(), "the study lock was released")
    names = [f"mpg-{run_id}" for run_id in expected] + [f"mpg-assess-{run_id}" for run_id in expected]
    listed = subprocess.run(["docker", "ps", "--all", "--filter", "label=markitect-playground=1", "--format",
                             "{{.Names}}"], capture_output=True, text=True, encoding="utf-8").stdout.split()
    remaining = sorted(set(names) & set(listed))
    check(not remaining, f"no labelled container of the study remains {remaining}")

    if failures:
        print(f"smoke: FAILED ({len(failures)} check(s)); study folder kept at {out}")
        return 1
    print("smoke: passed")
    if args.keep:
        print(f"smoke: study folder kept at {out}")
    else:
        shutil.rmtree(temp, ignore_errors=True)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
