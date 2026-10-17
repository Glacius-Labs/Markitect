"""Assess a library realization with the independent playground oracle.

The realization is Agent-produced code, so it is built and run only inside
containers without network: the example Go runtime image builds
`./cmd/library`, and a Python image runs the oracle from
experiments/library-playground at a fixed Git revision of Markitect.

Usage (from any directory):
  python -I assess.py --realization DIR --profile m1 --out DIR [--label A1]

DIR may be the project root (with realization/go.mod) or the module itself.
"""
import argparse
import datetime
import io
import json
import os
import pathlib
import shutil
import subprocess
import sys
import tarfile

HERE = pathlib.Path(__file__).resolve().parent
DEFAULT_REPO = str(pathlib.Path(__file__).resolve().parents[2])
DEFAULT_ENGINE = os.environ.get("MARKITECT_ENGINE") or shutil.which("docker") or "docker"
PYTHON_IMAGE = "python:3.12-slim@sha256:a6e34c598f2467ed0e9a8d349809fcd8b5c603269512df273a0bb1784edc11b1"
LIMITS = {"request_seconds": 2, "process_seconds": 30, "request_bytes": 16384, "output_bytes": 262144}


def run(cmd, **kw):
    return subprocess.run(cmd, check=True, **kw)


def module_dir(path):
    path = pathlib.Path(path).resolve()
    for candidate in (path / "realization", path):
        if (candidate / "go.mod").is_file():
            return candidate
    sys.exit(f"no realization/go.mod or go.mod under {path}")


def export_evaluator(repo, rev, dest):
    commit = run(["git", "-C", repo, "rev-parse", rev], capture_output=True, text=True).stdout.strip()
    archive = run(["git", "-C", repo, "archive", commit, "experiments/library-playground"], capture_output=True).stdout
    with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
        tar.extractall(dest, filter="data")
    return commit, dest / "experiments" / "library-playground"


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--realization", required=True)
    p.add_argument("--profile", choices=["m1", "m2"], required=True)
    p.add_argument("--out", required=True)
    p.add_argument("--label", default="A")
    p.add_argument("--repo", default=DEFAULT_REPO)
    p.add_argument("--rev", default="origin/main")
    p.add_argument("--engine", default=DEFAULT_ENGINE)
    p.add_argument("--go-image", default="markitect-runtime-go:dev")
    a = p.parse_args()

    out = pathlib.Path(a.out).resolve()
    out.mkdir(parents=True, exist_ok=True)
    module = module_dir(a.realization)
    commit, evaluator = export_evaluator(a.repo, a.rev, out / "evaluator")
    stamp = datetime.datetime.now().strftime("%Y%m%d%H%M%S")
    volume = f"mk-assess-{stamp}"
    docker = a.engine
    run([docker, "volume", "create", volume], capture_output=True)
    try:
        build = subprocess.run([
            docker, "run", "--rm", "--network", "none", "--user", "0:0",
            "--entrypoint", "/usr/local/go/bin/go",
            "--mount", f"type=bind,src={module},dst=/src,readonly",
            "--mount", f"type=volume,src={volume},dst=/out",
            "-w", "/src", "-e", "GOCACHE=/tmp/gocache", "-e", "GOPATH=/tmp/gopath", "-e", "HOME=/tmp",
            "-e", "GOFLAGS=-buildvcs=false", a.go_image, "build", "-o", "/out/library", "./cmd/library",
        ], capture_output=True, text=True)
        (out / f"build-{a.label}.log").write_text(build.stdout + build.stderr, encoding="utf-8")
        if build.returncode != 0:
            print(json.dumps({"label": a.label, "profile": a.profile, "built": False, "log": str(out / f"build-{a.label}.log")}))
            return 2
        result = subprocess.run([
            docker, "run", "--rm", "--network", "none", "--user", "0:0",
            "--mount", f"type=volume,src={volume},dst=/out",
            "--mount", f"type=bind,src={module},dst=/assess/realization,readonly",
            "--mount", f"type=bind,src={evaluator},dst=/evaluator,readonly",
            "--mount", f"type=bind,src={HERE},dst=/harness,readonly",
            PYTHON_IMAGE, "python", "-I", "/harness/run_oracle.py",
            "--label", a.label, "--profile", a.profile, "--source", str(module), "--commit", commit,
            "--limits", json.dumps(LIMITS),
        ], capture_output=True, text=True)
        (out / f"oracle-{a.label}.log").write_text(result.stdout + result.stderr, encoding="utf-8")
        report = subprocess.run([docker, "run", "--rm", "--mount", f"type=volume,src={volume},dst=/out",
                                 PYTHON_IMAGE, "cat", "/out/assessment.json"], capture_output=True)
        target = out / f"assessment-{a.label}-{a.profile}.json"
        target.write_bytes(report.stdout)
        data = json.loads(report.stdout or b"{}")
        cases = data.get("cases", [])
        summary = {
            "label": a.label, "profile": a.profile, "built": True, "evaluator_commit": commit,
            "passed": data.get("passed"), "cases_passed": sum(1 for c in cases if c.get("passed")),
            "cases": len(cases), "discrimination": (data.get("discrimination") or {}).get("expected_mismatch_confirmed"),
            "errors": data.get("errors"), "report": str(target),
        }
        print(json.dumps(summary, indent=2))
        return 0 if data.get("passed") else 1
    finally:
        subprocess.run([docker, "volume", "rm", "-f", volume], capture_output=True)


if __name__ == "__main__":
    sys.exit(main())
