"""Assess one comparison candidate, the same way for arm M and arm B.

Every candidate is a Git commit in a bundle, with the commit its writer
started from. The assessment runs, in containers without network:

  1. the unchanged Library oracle harness (tools/library-oracle/container/assess.py),
  2. the hidden supplementary probes (probes.py) through the oracle's transport,
     or, for Phase B's profiles m3-m5, instead of 1 and 2, every carried and
     new hidden case of phase_b_cases.py through the same transport,
  3. the candidate's own `go build ./cmd/library` and `go test ./...` with the
     limits of the Markitect `project-tests` check profile,

then classifies every changed path against the start commit and can write a
blinded copy for the rubric review. Evaluator and probe bytes come from the
frozen Markitect revision, never from the working tree.

  python -I -B assess_candidate.py --candidate BUNDLE --commit SHA --start BUNDLE --start-commit SHA \
      --profile m1|m2|m3|m4|m5 --run-id ID --out NEW_DIR --rev FROZEN_SHA --go-image sha256:... \
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


def _load(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


oracle_container = _load("oracle_container", REPO / "tools" / "library-oracle" / "container" / "assess.py")
runner = _load("comparison_runner", HERE / "runner.py")

LIMITS = {"oracle": 1200, "build": 300, "probes": 300, "cleanup": 20}
OWN_CHECK_BASE = ["--network", "none", "--user", "1000:1000", "--cap-drop", "ALL", "--security-opt", "no-new-privileges",
                  "--read-only", "--tmpfs", "/tmp:rw,exec,nosuid,nodev,mode=1777,size=1073741824",
                  "-e", "HOME=/tmp", "-e", "GOTOOLCHAIN=local", "-e", "GOCACHE=/tmp/gocache", "-e", "GOPATH=/tmp/gopath",
                  "-e", "GOTMPDIR=/tmp", "-e", "GOFLAGS=-buildvcs=false"]
# Primary: the writer arm's aggregate ceiling and 600 s. Secondary: the
# project-tests check profile of a Markitect Apply (1 CPU, 2 GiB, 64 processes, 2 minutes).
OWN_CHECK_PROFILES = {
    "primary": (["--cpus", runner.ARM_LIMITS["cpus"], "--memory", runner.ARM_LIMITS["memory"], "--pids-limit", runner.ARM_LIMITS["pids"]], 600),
    "project_tests": (["--cpus", "1", "--memory", "2147483648", "--pids-limit", "64"], 120),
}
PROTECTED = (".markitect/model.yaml", ".markitect/model-lock.yaml", ".markitect/model/", ".markitect/dependencies/",
             ".markitect/execution.yaml", "requirements.md")
SCOPE_PATTERNS = {"machine_clock": r"\btime\.Now\(", "network": r'"net(/http)?"', "file_writes": r"\bos\.(WriteFile|Create|OpenFile)\("}


def git(repo, *args, binary=False):
    result = subprocess.run(["git", "-c", "core.autocrlf=false", "-C", str(repo), *args], capture_output=True, timeout=120)
    if result.returncode != 0:
        raise RuntimeError("git " + " ".join(args[:2]) + ": " + result.stderr.decode(errors="replace").strip())
    return result.stdout if binary else result.stdout.decode()


def export(repo, commit, dest, exclude=(), strip_binaries=False):
    """Exact blob bytes of a commit, without Git or excluded top-level paths."""
    dest.mkdir(parents=True)
    with tarfile.open(fileobj=io.BytesIO(git(repo, "archive", "--format=tar", commit, binary=True))) as tar:
        members = [m for m in tar.getmembers() if not any(m.name == e or m.name.startswith(e + "/") for e in exclude)]
        if strip_binaries:
            members = [m for m in members if not (m.isfile() and tar.extractfile(m).read(4) == b"\x7fELF")]
        tar.extractall(dest, members=members, filter="data")


def binaries(repo, rev):
    rows = git(repo, "ls-tree", "-r", "-z", "--name-only", rev).split("\0")
    return [path for path in rows if path and is_binary(repo, rev, path)]


def is_binary(repo, rev, path):
    """A compiled executable left in the tree: an ELF header at that revision."""
    blob = subprocess.run(["git", "-C", str(repo), "cat-file", "blob", rev + ":" + path], capture_output=True)
    return blob.returncode == 0 and blob.stdout[:4] == b"\x7fELF"


def census(repo, start, commit):
    """Every changed path against the start commit, by category.

    A path that is a compiled executable on either side is build residue,
    whether it was added, rebuilt or deleted.
    """
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
        elif is_binary(repo, commit, path) or is_binary(repo, start, path):
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
    for profile, (limits, timeout) in OWN_CHECK_PROFILES.items():
        for name, command in (("build", "go build -o /tmp/library ./cmd/library"), ("test", "go test ./...")):
            code, stdout, stderr, timed_out = docker(engine, ["run", "--rm", *OWN_CHECK_BASE, *limits,
                                                              "--mount", f"type=bind,src={module},dst=/src,readonly",
                                                              "--entrypoint", "/bin/sh", image, "-c", "cp -R /src /tmp/m && cd /tmp/m && " + command],
                                                     timeout)
            (out / f"own-{profile}-{name}.log").write_bytes(stdout + stderr)
            results.setdefault(profile, {})[name] = {"exit": code, "timed_out": timed_out, "passed": code == 0 and not timed_out}
    return results


def probes(engine, image, module, out, repo, oracle_rev, rev, script="probes.py", extra=()):
    """Build the program and run a harness script of tools/comparison at rev against it."""
    exports = out / "frozen"
    commit, evaluator = oracle_container.export_evaluator(str(repo), oracle_rev, exports)
    archive = git(repo, "archive", "--format=tar", rev, "tools/comparison", binary=True)
    with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
        tar.extractall(exports, filter="data")
    harness = exports / "tools" / "comparison"
    protected = {name: hashlib.sha256(path.read_bytes()).hexdigest()
                 for name, path in {"evaluator/assess.py": evaluator / "assess.py", "evaluator/driver.py": evaluator / "driver.py",
                                    "evaluator/oracle.py": evaluator / "oracle.py", "harness/probes.py": harness / "probes.py"}.items()}
    if script != "probes.py":
        protected["harness/" + script] = hashlib.sha256((harness / script).read_bytes()).hexdigest()
    name = pathlib.Path(script).stem
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
        (out / f"{name}-build.log").write_bytes(stdout + stderr)
        if code != 0:
            report.update(built=False, passed=False, timed_out=timed_out)
            return report
        code, stdout, stderr, timed_out = docker(engine, ["run", "--rm", *oracle_container.HARDENING,
                                                          "--mount", f"type=volume,src={volume},dst=/out,readonly",
                                                          "--mount", f"type=bind,src={module},dst=/assess/realization,readonly",
                                                          "--mount", f"type=bind,src={evaluator},dst=/evaluator,readonly",
                                                          "--mount", f"type=bind,src={harness},dst=/harness,readonly",
                                                          oracle_container.PYTHON_IMAGE, "python", "-I", "/harness/" + script, *extra,
                                                          "--source", str(module), "--commit", commit,
                                                          "--limits", json.dumps(oracle_container.LIMITS), "--protected", json.dumps(protected)],
                                                 LIMITS["probes"])
        (out / f"{name}-report.json").write_bytes(stdout)
        (out / f"{name}.log").write_bytes(stderr)
        try:
            data = json.loads(stdout or b"{}")
        except ValueError:
            data = {"passed": False, "errors": ["probe output is not JSON"]}
        report.update(built=True, timed_out=timed_out, passed=bool(data.get("passed")) and not timed_out,
                      errors=data.get("errors"), cases={c["id"]: {"passed": c["passed"], "errors": c["errors"][:5]} for c in data.get("cases", [])})
        if "discrimination" in data:
            report["discrimination"] = data["discrimination"].get("expected_mismatch_confirmed")
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
    p.add_argument("--profile", choices=["m1", "m2", "m3", "m4", "m5"], required=True)
    p.add_argument("--engine", default=os.environ.get("MARKITECT_ENGINE") or shutil.which("docker") or "docker")
    p.add_argument("--repo", default=str(REPO))
    p.add_argument("--oracle-rev", help="revision of tools/library-oracle, the frozen source F; default --rev")
    p.add_argument("--seed-lines")
    p.add_argument("--blind")
    p.add_argument("--token")
    p.add_argument("--lock", help="exclusive engine lock file shared with the arm-M controller")
    a = p.parse_args(argv)
    a.oracle_rev = a.oracle_rev or a.rev
    if git(a.repo, "rev-parse", a.oracle_rev + ":tools/library-oracle") != git(a.repo, "rev-parse", a.rev + ":tools/library-oracle"):
        p.error("tools/library-oracle differs between --oracle-rev and --rev")
    with runner.native_lock(a.lock, "comparison assessment", a.run_id):
        return assess(a)


def assess(a):
    out = pathlib.Path(a.out).resolve()
    out.mkdir(parents=True, exist_ok=False)
    report = {"run_id": a.run_id, "profile": a.profile, "commit": a.commit, "start_commit": a.start_commit, "rev": a.rev,
              "oracle_rev": a.oracle_rev, "go_image": a.go_image,
              "assessor_sha256": hashlib.sha256(pathlib.Path(__file__).read_bytes()).hexdigest()}
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
                raise SystemExit("--blind needs --token of 6-32 lowercase letters or digits")
            # Arm markers and compiled executables never reach the reviewer.
            blind = pathlib.Path(a.blind) / a.token
            export(repo, a.commit, blind / "candidate", (".markitect", "requirements.md"), strip_binaries=True)
            export(repo, a.start_commit, blind / "start", (".markitect", "requirements.md"), strip_binaries=True)
            hidden = sorted(set(binaries(repo, a.commit)) | set(binaries(repo, a.start_commit)))
            (blind / "diff.patch").write_bytes(git(repo, "diff", "--no-renames", a.start_commit, a.commit, "--", ".",
                                                   ":!.markitect", ":!requirements.md", *[":!" + p for p in hidden], binary=True))
    module = candidate / "realization"
    if not (module / "go.mod").is_file():
        report.update(deliverable=False, mechanical_pass=False)
    else:
        report["deliverable"] = True
        if a.profile not in ("m1", "m2"):
            report["phase_b"] = probes(a.engine, a.go_image, module, out, a.repo, a.oracle_rev, a.rev,
                                       "phase_b_cases.py", ("--profile", a.profile))
            report["own_checks"] = own_checks(a.engine, a.go_image, module, out)
            report["scope"] = scope_findings(module)
            report["mechanical_pass"] = (report["phase_b"].get("passed") is True
                                         and all(r["passed"] for r in report["own_checks"]["primary"].values())
                                         and not report["census"]["protected"])
            return finish(a, out, report, candidate)
        oracle_out = out / "oracle"
        try:
            result = subprocess.run([sys.executable, "-I", str(REPO / "tools/library-oracle/container/assess.py"), "--realization", str(candidate),
                                     "--profile", a.profile, "--out", str(oracle_out), "--label", a.run_id, "--repo", a.repo,
                                     "--rev", a.oracle_rev, "--engine", a.engine, "--go-image", a.go_image],
                                    capture_output=True, text=True, timeout=LIMITS["oracle"])
            summary = json.loads(result.stdout) if result.stdout.strip().startswith("{") else {"output": result.stdout + result.stderr}
            report["oracle"] = {"exit": result.returncode, **summary}
            if summary.get("report") and pathlib.Path(summary["report"]).is_file():
                full = json.loads(pathlib.Path(summary["report"]).read_bytes() or b"{}")
                report["oracle"]["program_sha256"] = ((full.get("manifest") or {}).get("program_sha256") or {}).get("/out/library")
                report["oracle"]["failed_cases"] = [c["id"] for c in full.get("cases", []) if not c.get("passed")]
        except subprocess.TimeoutExpired:
            report["oracle"] = {"exit": None, "timed_out": True}
        report["probes"] = probes(a.engine, a.go_image, module, out, a.repo, a.oracle_rev, a.rev)
        report["own_checks"] = own_checks(a.engine, a.go_image, module, out)
        report["scope"] = scope_findings(module)
        report["mechanical_pass"] = (report["oracle"].get("exit") == 0 and report["oracle"].get("passed") is True
                                     and report["oracle"].get("discrimination") is True and report["probes"].get("passed") is True
                                     and all(r["passed"] for r in report["own_checks"]["primary"].values()) and not report["census"]["protected"])
    return finish(a, out, report, candidate)


def finish(a, out, report, candidate):
    if a.seed_lines:
        report["seed"] = seed_check(candidate, a.seed_lines)
    (out / "assessment.json").write_text(json.dumps(report, indent=2), encoding="utf-8")
    print(json.dumps({k: report.get(k) for k in ("run_id", "deliverable", "mechanical_pass")}, indent=2))
    return 0 if report.get("mechanical_pass") else 1


if __name__ == "__main__":
    sys.exit(main())
