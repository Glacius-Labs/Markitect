"""Shared parts of the series judges: one judge for every step of a change series.

A series judge is told its step by the suite (EVALUATION_STEP, from
`suite.py judge --step`) and judges the result against every hidden case of
that step and of every earlier step, as that step renders them. On top of the
three areas of every judge (correctness, buildAndTests, scope) it answers:

  step                 the step it judged
  correctness.byStep   planned, passed and failed cases per step that introduced them, for every step up to this one
  drift                forgottenRules: cases of an earlier step that fail now;
                       staleFiles: retired terms still in the program's source;
                       unaccounted: files in the whole tree that neither the build (its Go and embedded
                       files), the module files, vendored code, test data, the docs nor Markitect's
                       bookkeeping account for

`drift` is computed on the whole tree, not on the step's diff, so a judgment of
a chain's last step is the series' drift check, the same for every variant.
"""
import hashlib
import json
import os
import pathlib
import re
import shutil
import signal
import subprocess
import time

FORMAT = 1
LOG_TAIL = 4000
# Markitect's own bookkeeping: registry pages (Root's is registry.md), the ignore list and Results.
# Anything else under .markitect/ is a protected input.
BOOKKEEPING = (".markitect/registry.md", ".markitect/registry/", ".markitect/ignore.md", ".markitect/results/")
TASK_INPUTS = ("task.md", "TASK.md")
# Files outside the program that every repository may carry.
REPOSITORY_FILES = re.compile(r"(^|/)(\.gitignore|\.gitattributes)$")
DOCS = re.compile(r"\.md$", re.IGNORECASE)
GOOS = ("linux", "windows", "darwin")


def git(repo, *args):
    p = subprocess.run(["git", "-c", "core.autocrlf=false", "-C", str(repo), *args], capture_output=True)
    if p.returncode != 0:
        raise RuntimeError(f"git {' '.join(args[:2])}: {p.stderr.decode(errors='replace').strip()}")
    return p.stdout.decode()


def command(argv, cwd, seconds, env):
    """Run a step of the result's own code in its own process group; the group goes when it ends."""
    kw = {"creationflags": subprocess.CREATE_NEW_PROCESS_GROUP} if os.name == "nt" else {"start_new_session": True}
    started = time.monotonic()
    proc = subprocess.Popen(argv, cwd=cwd, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, stdin=subprocess.DEVNULL, **kw)
    try:
        out, _ = proc.communicate(timeout=seconds)
        timed_out = False
    except subprocess.TimeoutExpired:
        timed_out = True
    if os.name == "nt":
        if proc.poll() is None:
            subprocess.run(["taskkill", "/PID", str(proc.pid), "/T", "/F"], capture_output=True)
    else:
        try:
            os.killpg(proc.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
    if timed_out:
        try:
            out, _ = proc.communicate(timeout=10)
        except subprocess.TimeoutExpired:
            proc.kill()
            out = b"(output not read: a process outside the step still held it)"
    log = out.decode(errors="replace")
    return {"exit": None if timed_out else proc.returncode, "timedOut": timed_out, "passed": not timed_out and proc.returncode == 0,
            "seconds": round(time.monotonic() - started, 1), "log": log[-LOG_TAIL:]}


def go_env(tmp):
    env = dict(os.environ)
    env.update(HOME=str(tmp), GOENV="off", GOCACHE=str(tmp / "gocache"), GOPATH=str(tmp / "gopath"), GOTMPDIR=str(tmp),
               GOFLAGS="-buildvcs=false", GOTOOLCHAIN="local", GOPROXY="off", GOSUMDB="off")
    return env


def fetch(task, start_bundle, result_bundle, tmp):
    """A bare repository with the task's start and the result's one commit: (repo, start, commit)."""
    spec = json.loads((task / "task.json").read_text(encoding="utf-8"))
    start = spec["start"]["commit"]
    repo = tmp / "repo"
    git(tmp, "init", "-q", "--bare", str(repo))
    git(repo, "fetch", "-q", str(start_bundle), f"+{start}:refs/judge/start")
    heads = {line.split()[0] for line in git(repo, "bundle", "list-heads", str(result_bundle)).splitlines()}
    if len(heads) != 1:
        raise RuntimeError(f"the result bundle carries {len(heads)} commits, not one")
    commit = heads.pop()
    git(repo, "fetch", "-q", str(result_bundle), f"+{commit}:refs/judge/result")
    return repo, start, commit


def checkout(repo, commit, dest):
    """The commit's exact files, without Git."""
    dest.mkdir(parents=True)
    env = dict(os.environ, GIT_INDEX_FILE=str(dest.parent / (dest.name + ".index")))
    for args in (["read-tree", commit], ["checkout-index", "-a", "-f"]):
        p = subprocess.run(["git", "-c", "core.autocrlf=false", "-c", "core.symlinks=false", "--git-dir", str(repo), "--work-tree", str(dest), *args],
                           capture_output=True, env=env)
        if p.returncode != 0:
            raise RuntimeError(f"git {args[0]}: {p.stderr.decode(errors='replace').strip()}")


def is_binary(repo, rev, path):
    """A compiled executable: an ELF or PE header at that revision."""
    blob = subprocess.run(["git", "-C", str(repo), "cat-file", "blob", rev + ":" + path], capture_output=True)
    return blob.returncode == 0 and (blob.stdout[:4] == b"\x7fELF" or blob.stdout[:2] == b"MZ")


def model_copy(task, repo, commit, path):
    """Whether a changed file under .markitect/ is byte-equal to the task's own model file at the same relative path."""
    if not path.startswith(".markitect/"):
        return False
    rest = path[len(".markitect/"):]
    blob = subprocess.run(["git", "-C", str(repo), "cat-file", "blob", f"{commit}:{path}"], capture_output=True).stdout
    for origin in (task / "model" / rest, task / "model" / rest.removeprefix("model/")):
        if origin.is_file() and origin.read_bytes() == blob:
            return True
    return False


def scope(task, repo, start, commit, requested):
    """Every changed path against the start: the requested area, bookkeeping, protected inputs and build residue."""
    changed, unrequested, protected = [], [], []
    rows = git(repo, "diff", "--no-renames", "--name-status", "-z", start, commit).split("\0")
    for status, path in zip(rows[0::2], rows[1::2]):
        if path.startswith(BOOKKEEPING):
            category = "bookkeeping"
        elif path.startswith(".markitect/") or path in TASK_INPUTS:
            category = "protected"
        elif is_binary(repo, commit, path) or is_binary(repo, start, path):
            category = "build_residue"
        else:
            category = "requested" if path.startswith(requested) else "other"
        changed.append({"status": status, "path": path, "category": category})
        if category == "protected" and not (status != "D" and model_copy(task, repo, commit, path)):
            protected.append(path)
        elif category in ("build_residue", "other"):
            unrequested.append(path)
    return {"changedPaths": changed, "unrequested": sorted(unrequested), "protectedTouched": sorted(protected)}


def introduced(case_ids_by_step, steps):
    """The step that introduced each case: the first step whose case set holds it."""
    first = {}
    for step in steps:
        for case in case_ids_by_step[step]:
            first.setdefault(case, step)
    return first


def correctness(cases, step, steps, first, planned=()):
    """The correctness area with byStep for every step up to this one; cases carry introducedIn.

    A planned case that did not run, because the oracle stopped early, fails."""
    upto = steps[:steps.index(step) + 1]
    ran = {c["id"] for c in cases}
    cases = cases + [{"id": c, "passed": False, "errors": ["did not run"]} for c in planned if c not in ran]
    for c in cases:
        c["introducedIn"] = first.get(c["id"], step)
    failed = [c["id"] for c in cases if not c["passed"]]
    by_step = {}
    for s in upto:
        mine = [c for c in cases if c["introducedIn"] == s]
        bad = sum(not c["passed"] for c in mine)
        by_step[s] = {"planned": len(mine), "passed": len(mine) - bad, "failed": bad}
    return {"planned": len(cases), "passed": len(cases) - len(failed), "failed": len(failed), "failedCases": failed,
            "byStep": by_step, "cases": cases}


def go_accounted(module, env):
    """The module's Go files the build or its tests use, on any of the usual systems; None when go list fails."""
    used = set()
    for goos in GOOS:
        p = subprocess.run(["go", "list", "-e", "-json", "./..."], cwd=module, env=dict(env, GOOS=goos, CGO_ENABLED="0"),
                           capture_output=True, timeout=300)
        if p.returncode != 0:
            return None
        decoder, text, i = json.JSONDecoder(), p.stdout.decode(errors="replace"), 0
        while i < len(text):
            while i < len(text) and text[i].isspace():
                i += 1
            if i >= len(text):
                break
            package, i = decoder.raw_decode(text, i)
            directory = pathlib.Path(package.get("Dir", ""))
            for key in ("GoFiles", "CgoFiles", "TestGoFiles", "XTestGoFiles", "EmbedFiles", "TestEmbedFiles", "XTestEmbedFiles"):
                used.update((directory / name).resolve() for name in package.get(key) or [])
    return used


def unaccounted(tree, requested, go_used):
    """Files of the whole tree that nothing accounts for; None when the Go build could not be listed."""
    if go_used is None:
        return None
    found = []
    for path in sorted(p for p in tree.rglob("*") if p.is_file() or p.is_symlink()):
        rel = path.relative_to(tree).as_posix()
        if rel.startswith(".markitect/") or rel in TASK_INPUTS or REPOSITORY_FILES.search(rel) or DOCS.search(rel):
            continue
        if rel.startswith(requested):
            inner = rel[len(requested):]
            if inner in ("go.mod", "go.sum") or "/testdata/" in "/" + inner or inner.startswith("vendor/"):
                continue
            if path.resolve() in go_used:
                continue
        found.append(rel)
    return found


def stale(tree, requested, retired, step, steps):
    """Retired terms in the program's non-test Go source: each a term a step at or before this one removed."""
    due = [r for r in retired if steps.index(r["step"]) <= steps.index(step)]
    hits = []
    if not due:
        return hits
    for path in sorted((tree / requested).rglob("*.go")) if (tree / requested).is_dir() else []:
        if path.name.endswith("_test.go") or "testdata" in path.relative_to(tree).parts:
            continue
        for number, line in enumerate(path.read_text(encoding="utf-8", errors="replace").splitlines(), 1):
            for r in due:
                if re.search(r["pattern"], line):
                    hits.append({"path": path.relative_to(tree).as_posix(), "line": number, "term": r["pattern"],
                                 "retiredIn": r["step"], "cite": r["cite"]})
    return hits


def drift(cases, step, tree, requested, retired, steps, go_used):
    return {"forgottenRules": [c["id"] for c in cases if not c["passed"] and c["introducedIn"] != step],
            "staleFiles": stale(tree, requested, retired, step, steps),
            "unaccounted": unaccounted(tree, requested, go_used)}


def frozen_changed(here):
    """Files of an exported judge folder whose bytes differ from the hashes the export recorded; empty in the repository."""
    try:
        files = json.loads((here / "judge.json").read_text(encoding="utf-8"))["frozen"]["files"]
    except (OSError, ValueError, KeyError, TypeError):
        return []
    return sorted(name for name, digest in files.items()
                  if not (here / name).is_file() or hashlib.sha256((here / name).read_bytes()).hexdigest() != digest)


def judged_step(steps):
    """The step the suite asks for; refuses an unknown one."""
    step = os.environ.get("EVALUATION_STEP")
    if step not in steps:
        raise RuntimeError(f"EVALUATION_STEP is {step!r}, not one of {', '.join(steps)}")
    return step


def stop_everything_else():
    """Kill every other process this user runs in the container: whatever the result started is gone.

    Only in the suite's judge container, which sets EVALUATION_JUDGE_CONTAINER: anywhere else the same call
    would stop every process of the user."""
    if not os.environ.get("EVALUATION_JUDGE_CONTAINER"):
        return
    try:
        os.kill(-1, signal.SIGKILL)
    except (ProcessLookupError, PermissionError):
        pass


def write(out, record, contained):
    """Write judge.json; inside the container, first stop everything else and empty the output folder."""
    if contained:
        stop_everything_else()
        for entry in out.iterdir():
            shutil.rmtree(entry) if entry.is_dir() and not entry.is_symlink() else entry.unlink()
    (out / "judge.json").write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")
    return 0 if "error" not in record else 1
