"""The concept 2.0 arm: one `markitect apply` per task, its host native, its agents in containers on the suite's network.

  python -I -B markitect_arm.py --markitect BIN --remote-binary BIN --manager-image IMAGE [--effort LEVEL] TASK_DIR OUT_DIR [STATE_DIR]

Run it only through `suite.py run --native` or `chain --native` (native.py),
which hands it the evaluation's model, an access-only login, the network every
container joins, the provider endpoints at the gateway, a resolver that
resolves nothing, a label and the engine. Per task it:

1. puts the task's start into a private bare repository, the Run's target, and
   clones it;
2. commits the owner's input with the owner's identity, which is not the arm's
   work: the task's model/ as .markitect/ (manifest, lock, execution.yaml,
   model/ and the registry pages that give each Package its paths). A start
   that holds files but no model, a fixed reference start, is first adopted:
   the task's base-model/ (the model the start realizes; without one, the
   task's model) and one registry row without links for every file, on the
   page `markitect check` names as its owner's. The model then changes in a
   second commit. A start that already holds a model keeps its registry pages,
   whose tables are its Managers';
3. runs `markitect apply`: `--full` on a fresh build or an adopted start
   without a base model, `--base` the adoption commit after it, and the
   default impact, from the last landing, when the start is this arm's own
   last landing (the state directory records it);
4. delivers the landed commit as result.bundle, or the start unchanged when
   the Run did not land, so that a stop scores as "did not land". The Result
   goes to OUT_DIR/results/RUN.json (the suite's Result contract,
   results.py), the agents' transcripts to OUT_DIR/transcripts/ and what the
   arm knows about itself to OUT_DIR/run.json.

The host gets MARKITECT_NETWORK, MARKITECT_HOSTS and MARKITECT_DNS from the
suite's EVALUATION_NETWORK, EVALUATION_HOSTS and EVALUATION_DNS, the login
from EVALUATION_LOGIN, the provider and model from EVALUATION_MODEL and the
engine from EVALUATION_ENGINE; the Run is named after EVALUATION_LABEL, so its
containers carry the suite's run ID in `markitect.run`. Every ANTHROPIC_*,
CLAUDE_*, OPENAI_* and CODEX_* variable is cleared: the account login only,
never an API key.

Exit 0: the Run landed, found nothing to do or stopped on a model defect.
Exit 1: it ended otherwise (incomplete, kept) and the start was delivered.
Exit 2: it could not run, and nothing was delivered: the owner's input does
not compile, the host could not start the Run, its gate failed, or the Run
did not land and no agent completed a turn. None of these is the arm's work,
and an unchanged start would score as a stop. The series protocol voids them
with a variant runtime signature on `reported.status` or
`reported.noAgentTurn` (classify.py).
"""
import argparse
import json
import os
import pathlib
import re
import shutil
import subprocess
import sys
import tempfile
import time
import uuid

OWNER = ("Owner", "owner@markitect.invalid")
STATE_FILE = "markitect-arm.json"
# The owner's model files: everything under .markitect/ but the registry pages, the ignore page and the Results.
MODEL_ENTRIES = ("manifest.yaml", "lock.yaml", "execution.yaml", "model")
# Time kept back from the task's deadline for setting up and delivering.
MARGIN_SECONDS = 180
CLEARED = ("ANTHROPIC_", "CLAUDE_", "CLAUDECODE", "OPENAI_", "CODEX_", "MARKITECT_", "GIT_", "GH_", "GITHUB_")
PROVIDERS = ("claude", "codex")
# How the Run ended: delivered and exit 0, or delivered with exit 1 (results.py and internal/run/result.go).
DELIBERATE = ("landed", "nothing-to-do", "model-defect")
UNREGISTERED = re.compile(r"list it on (\S+), the page of its owner")


class ArmError(Exception):
    pass


def git(repo, *args, env=None, check=True):
    p = subprocess.run(["git", "-c", "core.autocrlf=false", "-c", "gc.auto=0", "-c", "maintenance.auto=false", "-c", "commit.gpgsign=false",
                        "-C", str(repo), *args], capture_output=True, text=True, env=env)
    if check and p.returncode != 0:
        raise ArmError(f"git {' '.join(args[:3])}: {p.stderr.strip()}")
    return p.stdout.strip()


def descends(repo, ancestor, commit_id, env):
    return subprocess.run(["git", "-C", str(repo), "merge-base", "--is-ancestor", ancestor, commit_id], capture_output=True, env=env).returncode == 0


def git_env(hooks):
    """No system or user Git configuration, no hooks, no credentials, no line-ending conversion; the owner's identity.
    No provider, Markitect or Git variable of the caller's."""
    env = {k: v for k, v in os.environ.items() if not k.upper().startswith(CLEARED)}
    name, email = OWNER
    settings = {"core.autocrlf": "false", "core.hooksPath": str(hooks), "credential.helper": "", "commit.gpgsign": "false"}
    env.update(GIT_CONFIG_NOSYSTEM="1", GIT_CONFIG_GLOBAL=os.devnull, GIT_TERMINAL_PROMPT="0", GIT_AUTHOR_NAME=name, GIT_AUTHOR_EMAIL=email,
               GIT_COMMITTER_NAME=name, GIT_COMMITTER_EMAIL=email, GIT_CONFIG_COUNT=str(len(settings)))
    for i, (key, value) in enumerate(settings.items()):
        env[f"GIT_CONFIG_KEY_{i}"], env[f"GIT_CONFIG_VALUE_{i}"] = key, value
    return env


def provider_model():
    provider, _, model = os.environ.get("EVALUATION_MODEL", "").partition(":")
    if provider not in PROVIDERS or not model:
        raise ArmError(f"EVALUATION_MODEL must be claude:MODEL_ID or codex:MODEL_ID, not {os.environ.get('EVALUATION_MODEL')!r}")
    return provider, model


def run_name():
    """The suite's run ID from EVALUATION_LABEL (KEY=VALUE), so the Run's containers carry it; a fresh one without a label."""
    value = os.environ.get("EVALUATION_LABEL", "").rpartition("=")[2]
    return value if re.fullmatch(r"[a-z0-9][a-z0-9_.-]{0,62}", value) else "eval-" + uuid.uuid4().hex[:12]


def host_env(a, provider, model, transcripts, hooks):
    """What `markitect apply` needs, from the suite's EVALUATION_* variables; no provider key reaches it."""
    engine = json.loads(os.environ.get("EVALUATION_ENGINE") or json.dumps([shutil.which("docker") or "docker"]))
    if not isinstance(engine, list) or len(engine) != 1:
        raise ArmError(f"the host takes one engine program, not {engine!r}")
    login = os.environ.get("EVALUATION_LOGIN")
    if not login:
        raise ArmError("no EVALUATION_LOGIN: run the arm through suite.py run --native")
    env = git_env(hooks)
    for k in ("GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL"):
        env.pop(k)
    env.update(MARKITECT_MANAGER_IMAGE=a.manager_image, MARKITECT_REMOTE_BINARY=a.remote_binary, MARKITECT_ENGINE=engine[0],
               MARKITECT_PROVIDER=provider, MARKITECT_MODEL=model, MARKITECT_LOGIN=login, MARKITECT_EFFORT=a.effort,
               MARKITECT_TRANSCRIPTS=str(transcripts))
    for name in ("NETWORK", "HOSTS", "DNS"):
        if os.environ.get("EVALUATION_" + name):
            env["MARKITECT_" + name] = os.environ["EVALUATION_" + name]
    return env


def copy_model(source, dot, keep_registry):
    """The owner's model in .markitect/: the model files replaced; registry pages added where the start has none."""
    dot.mkdir(parents=True, exist_ok=True)
    for name in MODEL_ENTRIES:
        target = dot / name
        if target.is_dir():
            shutil.rmtree(target)
        elif target.exists():
            target.unlink()
        if (source / name).is_dir():
            # Fresh timestamps: a copy that kept its source's would look unchanged to Git's stat check when an equally
            # long file of the same time was there before (Library M1 to M2 changes "5" to "3").
            shutil.copytree(source / name, target, copy_function=shutil.copyfile)
        elif (source / name).is_file():
            shutil.copyfile(source / name, target)
    add_pages(source, dot, replace=not keep_registry)


def add_pages(source, dot, replace=False):
    """The source's registry pages in .markitect/registry/; an existing page stays unless replace."""
    for page in sorted(p for p in (source / "registry").rglob("*.md") if p.is_file()) if (source / "registry").is_dir() else []:
        target = dot / "registry" / page.relative_to(source / "registry")
        if target.exists() and not replace:
            continue
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(page, target)


def check(a, repo, env):
    p = subprocess.run([a.markitect, "check", "--repo", str(repo), "--json"], capture_output=True, text=True, encoding="utf-8",
                       errors="replace", env=env)
    try:
        return json.loads(p.stdout), p.returncode
    except ValueError:
        raise ArmError(f"markitect check printed no JSON (exit {p.returncode}): {p.stderr.strip()[-500:]}")


def adopt(a, repo, env):
    """One registry row without links for every file of the start, on the page of its owner as the compiler names it."""
    report, _ = check(a, repo, env)
    rows = {}
    for diag in report.get("errors") or []:
        if diag.get("code") != "file.unregistered":
            continue
        page = UNREGISTERED.search(diag.get("message", ""))
        if not page:
            raise ArmError(f"cannot read the owner page from: {diag}")
        rows.setdefault(page.group(1), []).append(diag["file"])
    for page, files in sorted(rows.items()):
        path = repo / page
        text = path.read_text(encoding="utf-8") if path.exists() else "# Registry\n"
        if "| File |" not in text:
            text = text.rstrip("\n") + "\n\n| File | Realizes | Why |\n|---|---|---|\n"
        text = text.rstrip("\n") + "\n" + "".join(f"| `{f}` | | Reference start, adopted by the owner |\n" for f in sorted(files))
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text, encoding="utf-8", newline="\n")
    return sum(len(f) for f in rows.values())


def commit(repo, message, env):
    git(repo, "add", "-A", env=env)
    if not git(repo, "status", "--porcelain", env=env):
        return None
    git(repo, "commit", "-q", "-m", message, env=env)
    return git(repo, "rev-parse", "HEAD", env=env)


def compiles(a, repo, env, what):
    report, code = check(a, repo, env)
    if code != 0:
        raise ArmError(f"{what} does not compile: " + json.dumps(report.get("errors"))[:2000])


def owner_input(a, task, repo, start, landed, env):
    """The owner's commits on the start, and how apply runs: (apply options, mode, adoption record).

    Every owner commit must compile: a gate that fails on the owner's input is not the variant's work, so the arm
    delivers nothing rather than an unchanged start that would score as a stop."""
    dot = repo / ".markitect"
    model, base_model = task / "model", task / "base-model"
    if not model.is_dir():
        raise ArmError("the task has no model/: the 2.0 arm works from the model")
    files = [p for p in git(repo, "ls-files", env=env).splitlines() if p]
    adoption = None
    if dot.is_dir():
        copy_model(model, dot, keep_registry=True)
        options, mode = ([], "change") if start == landed else (["--full"], "full")
    elif not files:
        copy_model(model, dot, keep_registry=False)
        options, mode = ["--full"], "fresh"
    else:
        copy_model(base_model if base_model.is_dir() else model, dot, keep_registry=False)
        # Every Package's page is there before the rows are placed, so that each file lands on its final owner's page.
        add_pages(model, dot)
        rows = adopt(a, repo, env)
        compiles(a, repo, env, "the adopted start")
        adoption = {"commit": commit(repo, "Owner: adopt the reference start", env), "rows": rows, "baseModel": base_model.is_dir()}
        if base_model.is_dir():
            copy_model(model, dot, keep_registry=True)
            options, mode = ["--base", adoption["commit"]], "adopted"
        else:
            options, mode = ["--full"], "adopted"
    commit(repo, "Owner: the model of this task", env)
    committed(repo, model, env)
    compiles(a, repo, env, "the owner's input")
    return options, mode, adoption


def committed(repo, model, env):
    """The owner's commit holds exactly the task's model files, byte for byte."""
    want = {p.relative_to(model).as_posix(): p for name in MODEL_ENTRIES for p in ([model / name] if (model / name).is_file() else
                                                                                    sorted((model / name).rglob("*")) if (model / name).is_dir() else [])
            if p.is_file()}
    have = {p[len(".markitect/"):] for name in MODEL_ENTRIES
            for p in git(repo, "ls-tree", "-r", "--name-only", "HEAD", f".markitect/{name}", env=env).splitlines() if p}
    wrong = sorted(set(want) ^ have) + sorted(rel for rel in set(want) & have
                                              if subprocess.run(["git", "-C", str(repo), "show", f"HEAD:.markitect/{rel}"], capture_output=True,
                                                                env=env).stdout != want[rel].read_bytes())
    if wrong:
        raise ArmError("the owner's commit differs from the task's model: " + ", ".join(wrong[:10]))


def bundle(repo, commit_id, out, env):
    """One complete bundle with the one commit on refs/heads/result."""
    with tempfile.TemporaryDirectory(ignore_cleanup_errors=True) as tmp:
        git(tmp, "init", "-q", "--bare", env=env)
        git(tmp, "fetch", "-q", str(pathlib.Path(repo).resolve()), f"+{commit_id}:refs/heads/result", env=env)
        git(tmp, "bundle", "create", "-q", str(pathlib.Path(out).resolve()), "refs/heads/result", env=env)


def read_state(state):
    if not state:
        return {}
    try:
        value = json.loads((pathlib.Path(state) / STATE_FILE).read_text(encoding="utf-8"))
    except (OSError, ValueError):
        return {}
    return value if isinstance(value, dict) else {}


def reported_model(result, provider, model):
    """The model every agent that ran reported, or the configured one when none ran; 'mixed' when they differ.

    A start that did not finish may carry the configured name before its CLI reported one, so only finished starts count."""
    agents = {s.get("agent") for s in (result or {}).get("starts") or [] if s.get("agent") and s.get("reason") == "finished"}
    if not agents:
        return provider, model
    if len(agents) > 1:
        return provider, "mixed: " + ", ".join(sorted(agents))
    p, _, m = agents.pop().partition(":")
    return p, m


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--markitect", required=True, help="the markitect program of the frozen revision")
    p.add_argument("--remote-binary", required=True, help="markitect-remote built for Linux, put into each Run's remote")
    p.add_argument("--manager-image", required=True, help="the Manager image with the project's toolchain, by image ID")
    p.add_argument("--effort", default="high")
    p.add_argument("task")
    p.add_argument("out")
    p.add_argument("state", nargs="?")
    a = p.parse_args(argv)
    began = time.monotonic()
    task, out = pathlib.Path(a.task), pathlib.Path(a.out)
    out.mkdir(parents=True, exist_ok=True)
    report = {"arm": "markitect", "effort": a.effort}

    def finish(code, **facts):
        report.update(facts)
        (out / "run.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
        return code

    try:
        provider, model = provider_model()
        report.update(provider=provider, model=model)
        spec = json.loads((task / "task.json").read_text(encoding="utf-8"))
        start, deadline = spec["start"]["commit"], spec["deadlineSeconds"]
        state = read_state(a.state)
        name = run_name()
        report["run"] = name
        with tempfile.TemporaryDirectory(prefix="markitect-arm-", ignore_cleanup_errors=True) as tmp:
            tmp = pathlib.Path(tmp)
            (tmp / "hooks").mkdir()
            env = git_env(tmp / "hooks")
            target, repo = tmp / "target.git", tmp / "work"
            git(tmp, "init", "-q", "--bare", "-b", "main", str(target), env=env)
            git(target, "fetch", "-q", str((task / "start.bundle").resolve()), f"+{start}:refs/heads/main", env=env)
            git(tmp, "clone", "-q", "-b", "main", str(target), str(repo), env=env)
            git(repo, "config", "core.autocrlf", "false", env=env)
            options, mode, adoption = owner_input(a, task, repo, start, state.get("landed"), env)
            report.update(mode=mode, adoption=adoption, ownerCommit=git(repo, "rev-parse", "HEAD", env=env))
            git(repo, "push", "-q", "origin", "main", env=env)
            seconds = int(deadline - (time.monotonic() - began) - MARGIN_SECONDS)
            if seconds < 60:
                raise ArmError(f"only {seconds} s of the deadline are left for the Run")
            env_apply = host_env(a, provider, model, out / "transcripts", tmp / "hooks")
            (out / "results").mkdir(exist_ok=True)
            result_file = out / "results" / f"{name}.json"
            argv = [a.markitect, "apply", "--repo", str(repo), "--run", name, "--deadline", f"{seconds}s", "--json", *options]
            report["apply"] = argv[1:]
            with open(result_file, "wb") as stdout, open(out / "apply.stderr.log", "wb") as stderr:
                code = subprocess.run(argv, stdout=stdout, stderr=stderr, stdin=subprocess.DEVNULL, env=env_apply).returncode
            report["applyExit"] = code
            try:
                result = json.loads(result_file.read_text(encoding="utf-8"))
            except ValueError:
                result_file.unlink()
                return finish(2, status="could-not-run", cause="markitect apply returned no Result (see apply.stderr.log)")
            report["provider"], report["model"] = reported_model(result, provider, model)
            failed = [{k: s.get(k) for k in ("manager", "role", "round", "class", "detail")} for s in result.get("starts") or []
                      if s.get("reason") == "failed"]
            report.update(status=result.get("state"), cause=result.get("why"), tokens=result.get("usage"),
                          starts=len(result.get("starts") or []), failedStarts=failed, cleanup=result.get("cleanup"))
            if result.get("state") == "check-failed":
                # The gate failed on the owner's input, which compiled above: no agent started, nothing is the arm's work.
                return finish(2)
            if result.get("state") not in DELIBERATE and not any(s.get("reason") in ("finished", "deadline") for s in result.get("starts") or []):
                # No agent completed a turn: the Run ended on its machinery, not on any agent's work.
                return finish(2, noAgentTurn=True)
            landed = git(target, "rev-parse", "main", env=env) if result.get("outcome") == "landed" else None
            if landed and not (len(result.get("landed") or "") >= 7 and landed.startswith(result["landed"]) and descends(target, start, landed, env)):
                raise ArmError(f"the target's main {landed} is not the Result's landed commit {result.get('landed')} on the start")
            bundle(target, landed or start, out / "result.bundle", env)
            code = 0 if result.get("state") in DELIBERATE else 1
            report.update(landed=landed, exit=code)
            # What follows the delivery is record-keeping: its failure must not turn a delivered run into one that could not run.
            try:
                if landed and a.state:
                    (pathlib.Path(a.state) / STATE_FILE).write_text(json.dumps({"landed": landed, "run": name}, indent=2) + "\n", encoding="utf-8")
                (out / "target.log").write_text(git(target, "log", "--all", "--format=%H %s", env=env) + "\n", encoding="utf-8")
            except (ArmError, OSError) as e:
                report["recordError"] = f"{type(e).__name__}: {e}"
            return finish(code)
    except (ArmError, OSError, ValueError, KeyError) as e:
        if (out / "result.bundle").exists():
            report["recordError"] = f"{type(e).__name__}: {e}"
            return finish(report.get("exit", 1))
        return finish(2, status="could-not-run", cause=f"{type(e).__name__}: {e}")


if __name__ == "__main__":
    sys.exit(main())
