"""Assess one comparison candidate, the same way for arm M and arm B.

Every candidate is a Git commit in a bundle, with the commit its writer
started from. The assessment runs, in containers without network:

  1. the unchanged Library oracle harness (tools/library-oracle/container/assess.py),
  2. the hidden supplementary probes (probes.py) through the oracle's transport,
  3. the candidate's own `go build ./cmd/library` and `go test ./...` with the
     limits of the Markitect `project-tests` check profile,

then classifies every changed path against the start commit and can write a
blinded copy for the rubric review. Evaluator and probe bytes come from the
frozen Markitect revision, never from the working tree.

  python -I -B assess_candidate.py --candidate BUNDLE --commit SHA --start BUNDLE --start-commit SHA \
      --profile m1|m2 --run-id ID --out NEW_DIR --rev FROZEN_SHA --go-image sha256:... \
      [--seed-lines SEED.json] [--blind DIR --token TOKEN]
"""
import argparse
import hashlib
import importlib.util
import io
import json
import os
import pathlib
import re
import shutil
import subprocess
import sys
import tarfile
import tempfile
import uuid

HERE = pathlib.Path(__file__).resolve().parent
REPO = HERE.parents[1]
_spec = importlib.util.spec_from_file_location("oracle_container", REPO / "tools" / "library-oracle" / "container" / "assess.py")
oracle_container = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(oracle_container)

LIMITS = {"oracle": 1200, "build": 300, "probes": 300, "own_build": 120, "own_test": 120, "cleanup": 20}
# The project-tests check profile of a Markitect Apply: 1 CPU, 2 GiB, 64 processes, 2 minutes.
OWN_CHECK = ["--network", "none", "--user", "1000:1000", "--cap-drop", "ALL", "--security-opt", "no-new-privileges",
             "--read-only", "--tmpfs", "/tmp:rw,exec,nosuid,nodev,mode=1777,size=536870912",
             "--cpus", "1", "--memory", "2147483648", "--pids-limit", "64",
             "-e", "HOME=/tmp", "-e", "GOTOOLCHAIN=local", "-e", "GOCACHE=/tmp/gocache", "-e", "GOPATH=/tmp/gopath",
             "-e", "GOTMPDIR=/tmp", "-e", "GOFLAGS=-buildvcs=false"]
PROTECTED = (".markitect/model.yaml", ".markitect/model-lock.yaml", ".markitect/model/", ".markitect/dependencies/",
             ".markitect/execution.yaml", "requirements.md")
SCOPE_PATTERNS = {"machine_clock": r"\btime\.Now\(", "network": r'"net(/http)?"', "file_writes": r"\bos\.(WriteFile|Create|OpenFile)\("}


def git(repo, *args, binary=False):
    result = subprocess.run(["git", "-c", "core.autocrlf=false", "-C", str(repo), *args], capture_output=True, timeout=120)
    if result.returncode != 0:
        raise RuntimeError("git " + " ".join(args[:2]) + ": " + result.stderr.decode(errors="replace").strip())
    return result.stdout if binary else result.stdout.decode()


def export(repo, commit, dest, exclude=()):
    """Exact blob bytes of a commit, without Git or excluded top-level paths."""
    dest.mkdir(parents=True)
    with tarfile.open(fileobj=io.BytesIO(git(repo, "archive", "--format=tar", commit, binary=True))) as tar:
        members = [m for m in tar.getmembers() if not any(m.name == e or m.name.startswith(e + "/") for e in exclude)]
        tar.extractall(dest, members=members, filter="data")


def census(repo, start, commit):
    """Every changed path against the start commit, by category."""
    out = {"comparable": [], "registry": [], "protected": [], "build_residue": []}
    raw = git(repo, "diff", "--no-renames", "--name-status", "-z", start, commit).split("\0")
    numstat = {}
    for line in git(repo, "diff", "--no-renames", "--numstat", start, commit).splitlines():
        added, removed, path = line.split("\t", 2)
        numstat[path] = (added, removed)
    for status, path in zip(raw[0::2], raw[1::2]):
        entry = {"status": status, "path": path, "added": numstat.get(path, ("?", "?"))[0], "removed": numstat.get(path, ("?", "?"))[1]}
        if any(path == p or (p.endswith("/") and path.startswith(p)) for p in PROTECTED):
            out["protected"].append(entry)
        elif path.startswith(".markitect/registry/"):
            out["registry"].append(entry)
        elif status == "A" and git(repo, "cat-file", "blob", commit + ":" + path, binary=True)[:4] == b"\x7fELF":
            out["build_residue"].append(entry)
        else:
            out["comparable"].append(entry)
    return out


def scope_findings(candidate):
    found = {}
    for path in sorted(candidate.rglob("*.go")):
        if path.name.endswith("_test.go"):
            continue
        text = path.read_text(encoding="utf-8", errors="replace")
        for name, pattern in SCOPE_PATTERNS.items():
            if re.search(pattern, text):
                found.setdefault(name, []).append(path.relative_to(candidate).as_posix())
    return found


def docker(engine, args, timeout):
    try:
        result = subprocess.run([engine, *args], capture_output=True, timeout=timeout)
        return result.returncode, result.stdout, result.stderr, False
    except subprocess.TimeoutExpired as exc:
        return None, exc.stdout or b"", exc.stderr or b"", True


def own_checks(engine, image, module, out):
    results = {}
    for name, command in (("build", "go build -o /tmp/library ./cmd/library"), ("test", "go test ./...")):
        code, stdout, stderr, timed_out = docker(engine, ["run", "--rm", *OWN_CHECK,
                                                          "--mount", f"type=bind,src={module},dst=/src,readonly",
                                                          "--entrypoint", "/bin/sh", image, "-c", "cp -R /src /tmp/m && cd /tmp/m && " + command],
                                                 LIMITS["own_" + name])
        (out / f"own-{name}.log").write_bytes(stdout + stderr)
        results[name] = {"exit": code, "timed_out": timed_out, "passed": code == 0 and not timed_out}
    return results


def probes(engine, image, module, out, repo, rev):
    exports = out / "frozen"
    commit, evaluator = oracle_container.export_evaluator(str(repo), rev, exports)
    archive = git(repo, "archive", "--format=tar", rev, "tools/comparison", binary=True)
    with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
        tar.extractall(exports, filter="data")
    harness = exports / "tools" / "comparison"
    protected = {name: hashlib.sha256(path.read_bytes()).hexdigest()
                 for name, path in {"evaluator/assess.py": evaluator / "assess.py", "evaluator/driver.py": evaluator / "driver.py",
                                    "evaluator/oracle.py": evaluator / "oracle.py", "harness/probes.py": harness / "probes.py"}.items()}
    volume = "mk-cmp-probe-" + uuid.uuid4().hex[:12]
    report = {"commit": commit, "protected_sha256": protected}
    docker(engine, ["volume", "create", volume], LIMITS["cleanup"])
    try:
        docker(engine, ["run", "--rm", "--user", "0:0", "--network", "none", "--entrypoint", "/bin/chown",
                        "--mount", f"type=volume,src={volume},dst=/out", image, "1000:1000", "/out"], LIMITS["cleanup"])
        code, stdout, stderr, timed_out = docker(engine, ["run", "--rm", *oracle_container.HARDENING, "--entrypoint", "/usr/local/go/bin/go",
                                                          "--mount", f"type=bind,src={module},dst=/src,readonly",
                                                          "--mount", f"type=volume,src={volume},dst=/out", "-w", "/src",
                                                          "-e", "GOCACHE=/tmp/gocache", "-e", "GOPATH=/tmp/gopath", "-e", "GOFLAGS=-buildvcs=false",
                                                          image, "build", "-o", "/out/library", "./cmd/library"], LIMITS["build"])
        (out / "probe-build.log").write_bytes(stdout + stderr)
        if code != 0:
            report.update(built=False, passed=False, timed_out=timed_out)
            return report
        code, stdout, stderr, timed_out = docker(engine, ["run", "--rm", *oracle_container.HARDENING,
                                                          "--mount", f"type=volume,src={volume},dst=/out,readonly",
                                                          "--mount", f"type=bind,src={module},dst=/assess/realization,readonly",
                                                          "--mount", f"type=bind,src={evaluator},dst=/evaluator,readonly",
                                                          "--mount", f"type=bind,src={harness},dst=/harness,readonly",
                                                          oracle_container.PYTHON_IMAGE, "python", "-I", "/harness/probes.py",
                                                          "--source", str(module), "--commit", commit,
                                                          "--limits", json.dumps(oracle_container.LIMITS), "--protected", json.dumps(protected)],
                                                 LIMITS["probes"])
        (out / "probes-report.json").write_bytes(stdout)
        (out / "probes.log").write_bytes(stderr)
        try:
            data = json.loads(stdout or b"{}")
        except ValueError:
            data = {"passed": False, "errors": ["probe output is not JSON"]}
        report.update(built=True, timed_out=timed_out, passed=bool(data.get("passed")) and not timed_out,
                      errors=data.get("errors"), cases={c["id"]: {"passed": c["passed"], "errors": c["errors"][:5]} for c in data.get("cases", [])})
        return report
    finally:
        docker(engine, ["volume", "rm", "-f", volume], LIMITS["cleanup"])


def seed_check(candidate, seed_file):
    """Which deliberately seeded lines are still present verbatim."""
    remaining = []
    for item in json.loads(pathlib.Path(seed_file).read_text(encoding="utf-8")):
        path = candidate / item["path"]
        if path.is_file() and item["line"] in path.read_text(encoding="utf-8", errors="replace"):
            remaining.append(item)
    return {"seeded_lines": len(json.loads(pathlib.Path(seed_file).read_text(encoding="utf-8"))), "remaining": remaining}


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    for name in ("--candidate", "--commit", "--start", "--start-commit", "--run-id", "--out", "--rev", "--go-image"):
        p.add_argument(name, required=True)
    p.add_argument("--profile", choices=["m1", "m2"], required=True)
    p.add_argument("--engine", default=os.environ.get("MARKITECT_ENGINE") or shutil.which("docker") or "docker")
    p.add_argument("--repo", default=str(REPO))
    p.add_argument("--seed-lines")
    p.add_argument("--blind")
    p.add_argument("--token")
    a = p.parse_args(argv)
    out = pathlib.Path(a.out).resolve()
    out.mkdir(parents=True, exist_ok=False)
    report = {"run_id": a.run_id, "profile": a.profile, "commit": a.commit, "start_commit": a.start_commit, "rev": a.rev,
              "go_image": a.go_image, "assessor_sha256": hashlib.sha256(pathlib.Path(__file__).read_bytes()).hexdigest()}
    with tempfile.TemporaryDirectory() as tmp:
        repo = pathlib.Path(tmp) / "repo"
        subprocess.run(["git", "init", "-q", str(repo)], check=True, capture_output=True)
        for bundle in (a.start, a.candidate):
            git(repo, "bundle", "unbundle", str(pathlib.Path(bundle).resolve()))
        for commit in (a.start_commit, a.commit):
            git(repo, "cat-file", "-e", commit + "^{commit}")
        report["tree"] = git(repo, "rev-parse", a.commit + "^{tree}").strip()
        candidate = out / "candidate"
        export(repo, a.commit, candidate)
        report["census"] = census(repo, a.start_commit, a.commit)
        if a.blind:
            if not re.fullmatch(r"[a-z0-9]{6,32}", a.token or ""):
                p.error("--blind needs --token of 6-32 lowercase letters or digits")
            blind = pathlib.Path(a.blind) / a.token
            export(repo, a.commit, blind / "candidate", (".markitect", "requirements.md"))
            export(repo, a.start_commit, blind / "start", (".markitect", "requirements.md"))
            (blind / "diff.patch").write_bytes(git(repo, "diff", "--no-renames", a.start_commit, a.commit, "--", ".",
                                                   ":!.markitect", ":!requirements.md", binary=True))
    module = candidate / "realization"
    if not (module / "go.mod").is_file():
        report.update(deliverable=False, mechanical_pass=False)
    else:
        report["deliverable"] = True
        oracle_out = out / "oracle"
        try:
            result = subprocess.run([sys.executable, "-I", str(REPO / "tools/library-oracle/container/assess.py"), "--realization", str(candidate),
                                     "--profile", a.profile, "--out", str(oracle_out), "--label", a.run_id, "--repo", a.repo,
                                     "--rev", a.rev, "--engine", a.engine, "--go-image", a.go_image],
                                    capture_output=True, text=True, timeout=LIMITS["oracle"])
            summary = json.loads(result.stdout) if result.stdout.strip().startswith("{") else {"output": result.stdout + result.stderr}
            report["oracle"] = {"exit": result.returncode, **summary}
        except subprocess.TimeoutExpired:
            report["oracle"] = {"exit": None, "timed_out": True}
        report["probes"] = probes(a.engine, a.go_image, module, out, a.repo, a.rev)
        report["own_checks"] = own_checks(a.engine, a.go_image, module, out)
        report["scope"] = scope_findings(module)
        report["mechanical_pass"] = (report["oracle"].get("exit") == 0 and report["oracle"].get("passed") is True
                                     and report["oracle"].get("discrimination") is True and report["probes"].get("passed") is True
                                     and all(r["passed"] for r in report["own_checks"].values()) and not report["census"]["protected"])
    if a.seed_lines:
        report["seed"] = seed_check(candidate, a.seed_lines)
    (out / "assessment.json").write_text(json.dumps(report, indent=2), encoding="utf-8")
    print(json.dumps({k: report.get(k) for k in ("run_id", "deliverable", "mechanical_pass")}, indent=2))
    return 0 if report.get("mechanical_pass") else 1


if __name__ == "__main__":
    sys.exit(main())
