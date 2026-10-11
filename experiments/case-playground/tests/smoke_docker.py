"""Explicit Docker smoke test: one provider-free run with a fake agent.

  python experiments/case-playground/tests/smoke_docker.py [--manifest M.json] [--keep]

Runs `python -m playground host run` for a fake-agent manifest (default
examples/fake-roombook.json; `fake` stands in for Codex, `fake-claude` for Claude Code)
and checks the end-to-end contract for every station the run plans (the manifest's
`stations`, default every station of the case's STATIONS.json). A
`fake-claude` run gets a throwaway token file, which must reach the fake agent and must
not survive anywhere in the run folder. Runs from a Git checkout whose evaluation files
are committed and unchanged (the run must be pre-registered). Needs Docker (and Go for a
Markitect manifest); makes no model call. Not picked up by unittest discovery.
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
FAKE_KINDS = ("fake", "fake-claude")


def files_containing(root: Path, needle: bytes) -> list[str]:
    return [path.relative_to(root).as_posix() for path in root.rglob("*")
            if path.is_file() and not path.is_symlink() and needle in path.read_bytes()]


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__,
                                     formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--manifest", type=Path, default=ROOT / "examples" / "fake-roombook.json")
    parser.add_argument("--keep", action="store_true", help="keep the run folder on success")
    args = parser.parse_args()

    manifest_path = args.manifest.resolve()
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    kind = manifest.get("agent", {}).get("kind")
    if kind not in FAKE_KINDS:
        parser.error("the smoke only runs fake-agent manifests (no model calls)")
    plan = json.loads((ROOT / "cases" / manifest["case"] / "STATIONS.json").read_text(encoding="utf-8"))
    count = manifest.get("stations") or len(plan["stations"])
    temp = Path(tempfile.mkdtemp(prefix="mpg-smoke-"))
    out = temp / "run"
    command = [sys.executable, "-B", "-m", "playground", "host", "run", "--manifest", str(manifest_path),
               "--out", str(out)]
    token = None
    if kind == "fake-claude":  # a throwaway value, never a real login
        token = f"smoke-token-{secrets.token_hex(16)}"
        (temp / "claude-token").write_text(token + "\n", encoding="utf-8")
        command += ["--claude-token", str(temp / "claude-token")]
    print(f"smoke: manifest {manifest_path} ({count} stations)\nsmoke: out {out}", flush=True)
    host = subprocess.run(command, cwd=ROOT)

    failures: list[str] = []

    def check(ok: bool, message: str) -> None:
        print(f"  [{'ok' if ok else 'FAIL'}] {message}")
        if not ok:
            failures.append(message)

    print(f"smoke: host exit code {host.returncode}")
    check(host.returncode == 0, f"host run exited 0 (all {count} stations ran)")
    report_path = out / "results" / "report.json"
    check(report_path.is_file(), "results/report.json exists")
    check((out / "results" / "report.md").is_file(), "results/report.md exists")
    report = json.loads(report_path.read_text(encoding="utf-8")) if report_path.is_file() else {}
    check((report.get("setup") or {}).get("status") == "ready",
          f"method setup is ready ({(report.get('setup') or {}).get('error')})")
    check((report.get("classification") or {}).get("class") == "none",
          f"run classified none ({report.get('classification')})")
    final = report.get("final") or {}
    check(final.get("total") is not None and final.get("station") == count,
          f"final assessment ran the public checks of S{count} (station {final.get('station')})")
    if manifest.get("method") == "markitect":
        check((final.get("conformance") or {}).get("reportStatus") is not None,
              "final assessment ran markitect check")
        check(bool(report.get("roles")), f"Markitect roles recorded from runtime.yaml ({report.get('roles')})")
    stations = report.get("stations") or []
    check(len(stations) == count and (report.get("totals") or {}).get("stationsPlanned") == count
          and (report.get("fairness") or {}).get("stationsPlanned") == count,
          f"{count} stations ran and are a fairness field (got {len(stations)})")
    commits = [s.get("newMainCommits") for s in stations]
    check(len(commits) == count and all(isinstance(c, int) and c > 0 for c in commits),
          f"every station added commits to main {commits}")
    finals = [path.parent for path in out.rglob(f"FAKE_S{count}.md")
              if all((path.parent / f"FAKE_S{n}.md").is_file() for n in range(1, count + 1))]
    check(bool(finals), f"a captured main checkout holds FAKE_S1..S{count}.md")
    tokens = (report.get("totals") or {}).get("tokens") or {}
    every = (report.get("totals") or {}).get("tokensAllSessions") or {}
    check(tokens.get("input") == count * 1200 and every.get("input") == count * 1300,
          f"per-wave tokens from session records (outer {tokens}, all {every})")
    leftovers = [s.get("leftoverProcessesKilled") for s in stations]
    check(len(leftovers) == count and all(isinstance(n, int) and n > 0 for n in leftovers),
          f"leftover agent processes were killed after each station {leftovers}")
    host_record = json.loads((out / "host.json").read_text(encoding="utf-8")) if (out / "host.json").is_file() else {}
    check(host_record.get("status") == "completed", "host.json records status completed")
    image = (host_record.get("image") or {}).get("tag") or ""
    check("-claude-" in image, f"image tag names both CLI versions ({image})")
    pre = host_record.get("preRegistration") or {}
    check(pre.get("status") == "registered" and re.fullmatch(r"[0-9a-f]{40}|[0-9a-f]{64}",
                                                             pre.get("evaluationTree") or "") is not None
          and (host_record.get("rules") or {}).get("exploratory") is False,
          f"host.json records the pre-registration of the evaluation files ({pre})")
    here = {"system": platform.system(), "machine": platform.machine()}
    check(host_record.get("hostPlatform") == here and (report.get("fairness") or {}).get("hostPlatform") == here,
          f"host.json and the report's fairness record the host platform {here}")
    if manifest.get("method") == "markitect":
        product = (host_record.get("manifest") or {}).get("markitect") or {}
        check(Path(product.get("sourceRepo") or "").is_absolute()
              and re.fullmatch(r"[0-9a-f]{40}", product.get("commit") or "") is not None,
              f"host.json records sourceRepo as an absolute path and the full commit ({product})")
    if token is not None:
        events = out / "results" / "stations" / "S1" / "events.jsonl"
        text = events.read_text(encoding="utf-8") if events.is_file() else ""
        check('"fakeTokenSeen": true' in text, "the token reached the claude process environment")
        check("[redacted:CLAUDE_CODE_OAUTH_TOKEN]" in text, "the token the fake printed was redacted")
        leaked = files_containing(out, token.encode("utf-8"))
        check(not leaked, f"the token is nowhere in the run folder {leaked[:5]}")
        check(((report.get("setup") or {}).get("claudeRouter") or {}).get("added") is True,
              "the CLAUDE.md router was added")
        check((report.get("fairness") or {}).get("claude") is not None, "the Claude Code version is recorded")
    if kind == "fake":
        events = [out / "results" / "stations" / f"S{n}" / "events.jsonl" for n in range(1, count + 1)]
        hidden = [path.is_file() and '"fakeResultsVisible": false' in path.read_text(encoding="utf-8")
                  for path in events]
        check(all(hidden), f"the agent could not open /out in any station {hidden}")
    remaining = subprocess.run(["docker", "ps", "--all", "--quiet", "--filter", "label=markitect-playground=1",
                                "--filter", f"name=mpg-{manifest['id']}"],
                               capture_output=True, text=True, encoding="utf-8").stdout.split()
    check(not remaining, f"no labelled container mpg-{manifest['id']} remains")

    if failures:
        print(f"smoke: FAILED ({len(failures)} check(s)); run folder kept at {out}")
        return 1
    print("smoke: passed")
    if args.keep:
        print(f"smoke: run folder kept at {out}")
    else:
        shutil.rmtree(temp, ignore_errors=True)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
