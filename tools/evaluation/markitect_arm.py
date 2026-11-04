"""The concept 3.0 arm: one `markitect apply` per task, its host native, its agents in the Command's container on the suite's network.

  python -I -B markitect_arm.py --markitect BIN --image IMAGE_ID [--effort LEVEL] [--verify] TASK_DIR OUT_DIR [STATE_DIR]

Run it only through `suite.py run --native` or `chain --native` (native.py),
which hands it the evaluation's model, the network every container joins, the
provider endpoints at the gateway, a resolver that resolves nothing, a label
and the engine. The login is the owner's token, which the arm reads from
MARKITECT_CLAUDE_TOKEN in its own environment and passes to the host under the
same name; it refuses to start without it. The token is never written to a
file, a record or a log, and never put on a command line. Per task the arm:

1. puts the task's start into a private bare repository, the Command's target,
   and clones it;
2. commits the owner's input with the owner's identity, which is not the arm's
   work: the task's model/ as .markitect/ (manifest, lock, execution.yaml,
   model/ and the registry pages that give each Package its paths). A start
   that holds files but no model, a fixed reference start, is first adopted:
   the task's base-model/ (the model the start realizes; without one, the
   task's model) with its registry pages, which an owner wrote for that
   reference: every file with its owner and the Definitions it realizes. Only
   a file no page names gets a row without links, on the page `markitect
   check` names as its owner's; the record says which was used. Each page also
   gets the task's model's patterns for paths the start does not hold yet, so
   that the implementer can write where the model wants files. The model then
   changes in a second commit. A start that already holds a model keeps its
   registry pages, whose rows the implementer wrote;
3. runs `markitect apply`: `--full` on a fresh build or an adopted start
   without a base model, `--base` the adoption commit after it, and the
   default base, the work Markitect last landed, when the start is this arm's
   own last landing (the state directory records it);
4. delivers the landed commit, or the start unchanged when the Command did not
   land, so that a stop scores as "did not land". The Result goes to
   OUT_DIR/results/RUN.json (the suite's Result contract, results.py), the
   agents' transcripts to OUT_DIR/transcripts/ and what the arm knows about
   itself to OUT_DIR/run.json.

The host gets MARKITECT_NETWORK, MARKITECT_HOSTS and MARKITECT_DNS from the
suite's EVALUATION_NETWORK, EVALUATION_HOSTS and EVALUATION_DNS, the image from
--image as MARKITECT_MANAGER_IMAGE, the provider and model from
EVALUATION_MODEL and the engine from EVALUATION_ENGINE; the Command is named
after EVALUATION_LABEL, so its containers carry the suite's run ID in
`markitect.run`. Every other ANTHROPIC_*, CLAUDE_*, OPENAI_*, CODEX_*,
MARKITECT_* and EVALUATION_* variable is cleared: the owner's token only,
never an API key.

With --verify the arm checks the task's start as it is instead: the independent
`markitect verify --full` that reads No drift from a chain's end state,
whichever arm made it. A start that holds a model is checked against the task's
model with its own registry pages; one that holds files but no model is first
adopted, every file without links on its owner's page; an empty start gets the
model alone. The Result goes to OUT_DIR/results/RUN.json as for apply, and the
start is delivered unchanged: verify changes nothing. Exit 0: verify ended
`verified` or `findings`; 1: it ended otherwise; 2 as below.

Exit 0: the Command landed, found nothing to do or stopped on a spec problem.
Exit 1: it ended otherwise (refused, kept, incomplete) and the start was
delivered. A `refused` Command is the agents' work, such as required checks
that still fail after the repair rounds.
Exit 2: it could not run, and nothing was delivered: the owner's input does not
compile, the host could not start the Command, the start did not compile for
the host (`check-failed`, no agent started), or the Command did not land and
no session finished. None of these is the arm's work, and an unchanged start
would score as a stop. The series protocol voids them with a variant runtime
signature on `reported.status` or `reported.noAgentTurn` (classify.py).
"""
import argparse
import fnmatch
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
# A registry page's front matter.
PAGE_FRONT = re.compile(r"\A---\npaths:(?: \[\]\n|\n((?:  - \S+\n)+))---\n")
# Time kept back from the task's deadline for setting up and delivering.
MARGIN_SECONDS = 180
CLEARED = ("ANTHROPIC_", "CLAUDE_", "CLAUDECODE", "OPENAI_", "CODEX_", "MARKITECT_", "EVALUATION_", "GIT_", "GH_", "GITHUB_")
# The owner's token is the only login a Command takes, and claude the only provider with a route for it (internal/agents/README.md).
TOKEN_VARIABLE = "MARKITECT_CLAUDE_TOKEN"
PROVIDERS = ("claude",)
# How a Command ended on purpose: delivered and exit 0. Any other state is delivered and exit 1 (internal/run/result.go).
DELIBERATE = ("landed", "nothing-to-do", "spec-problem")
# How a verify Command ended when it checked every item it had: every item conforms, or something does not.
CHECKED = ("verified", "findings")
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


def owner_token():
    """The owner's token from the arm's own environment; the arm does not start without it. Its value goes only into the host's
    environment: never into a message, a record, a file or an argument."""
    token = os.environ.get(TOKEN_VARIABLE, "")
    if not token.strip():
        raise ArmError(f"no {TOKEN_VARIABLE}: the token from `claude setup-token` is the only login a Command takes")
    return token


def provider_model():
    provider, _, model = os.environ.get("EVALUATION_MODEL", "").partition(":")
    if provider not in PROVIDERS or not model:
        raise ArmError(f"EVALUATION_MODEL must be claude:MODEL_ID (a Command has no token route for codex), not {os.environ.get('EVALUATION_MODEL')!r}")
    return provider, model


def run_name():
    """The suite's run ID from EVALUATION_LABEL (KEY=VALUE), so the Command's containers carry it; a fresh one without a label."""
    value = os.environ.get("EVALUATION_LABEL", "").rpartition("=")[2]
    return value if re.fullmatch(r"[a-z0-9][a-z0-9_.-]{0,62}", value) else "eval-" + uuid.uuid4().hex[:12]


def host_env(a, provider, model, transcripts, hooks, token):
    """What `markitect apply` needs, from the suite's EVALUATION_* variables and the owner's token; no provider key reaches it."""
    engine = json.loads(os.environ.get("EVALUATION_ENGINE") or json.dumps([shutil.which("docker") or "docker"]))
    if not isinstance(engine, list) or len(engine) != 1:
        raise ArmError(f"the host takes one engine program, not {engine!r}")
    env = git_env(hooks)
    for k in ("GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL"):
        env.pop(k)
    env.update(MARKITECT_MANAGER_IMAGE=a.image, MARKITECT_ENGINE=engine[0], MARKITECT_PROVIDER=provider, MARKITECT_MODEL=model,
               MARKITECT_EFFORT=a.effort, MARKITECT_TRANSCRIPTS=str(transcripts), **{TOKEN_VARIABLE: token})
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
    """The source's registry pages, Root's registry.md and those in registry/; an existing page stays unless replace."""
    pages = sorted(p for p in (source / "registry").rglob("*.md") if p.is_file()) if (source / "registry").is_dir() else []
    for page in ([source / "registry.md"] if (source / "registry.md").is_file() else []) + pages:
        target = dot / page.relative_to(source)
        if target.exists() and not replace:
            continue
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(page, target)


def page_patterns(text):
    """The path patterns of a registry page; None when its front matter is not in the form pages are written in."""
    front = PAGE_FRONT.match(text)
    if front:
        return (front.group(1) or "").replace("  - ", "").split()
    return None if text.startswith(("---", "﻿")) else []


def pattern_matches(pattern, file):
    """The registry's rule (internal/realization/pattern.go): * and ? stay within a path segment, and ** spans segments,
    a trailing one at least one."""
    def match(pattern, name):
        if not pattern:
            return not name
        if pattern[0] == "**":
            if len(pattern) == 1:
                return bool(name)
            return any(match(pattern[1:], name[i:]) for i in range(len(name) + 1))
        return bool(name) and fnmatch.fnmatchcase(name[0], pattern[0]) and match(pattern[1:], name[1:])
    return match(pattern.split("/"), file.split("/"))


def pattern_specificity(pattern):
    """The registry's order among patterns that match a file: an exact path wins, then more literal characters."""
    return 1 << 30 if not set(pattern) & set("*?") else sum(c not in "*?/" for c in pattern)


def file_owner(file, pages):
    """The page whose pattern owns a file; None for Root, which owns what no pattern claims."""
    found = [(pattern_specificity(p), page) for page, patterns in pages.items() for p in patterns if pattern_matches(p, file)]
    return max(found)[1] if found else None


def add_new_paths(source, dot, files):
    """The source's patterns for new paths on the pages at `dot`: (how many it added, how many it left out).

    An adopted start's pages name only its files, so without the owner's patterns the new files of the task's model would
    fall to Root, and no Package would own the paths its model gives it. A pattern stays out if
    a page already has it, since two Packages claiming one is a compile error, or if it matches a file of the start that
    another page or Root owns: every file keeps its owner. A page in another form leaves the pages as they are, since
    the owners cannot be read."""
    registry = dot / "registry"
    targets = {p.relative_to(dot).as_posix(): p for p in registry.rglob("*.md")} if registry.is_dir() else {}
    pages = {name: page_patterns(path.read_text(encoding="utf-8")) for name, path in targets.items()}
    if None in pages.values():
        return 0, 0
    added = left_out = 0
    for name in sorted(targets):
        page = source / name
        new = []
        for pattern in (page_patterns(page.read_text(encoding="utf-8")) or []) if page.is_file() else []:
            if pattern in pages[name]:
                continue
            if any(pattern in patterns for patterns in pages.values()) or any(
                    pattern_matches(pattern, f) and file_owner(f, pages) != name for f in files):
                left_out += 1
                continue
            pages[name].append(pattern)
            new.append(pattern)
        if new:
            front = "---\npaths:\n" + "".join(f"  - {p}\n" for p in pages[name]) + "---\n"
            text = targets[name].read_text(encoding="utf-8")
            targets[name].write_text(front + PAGE_FRONT.sub("", text, count=1), encoding="utf-8", newline="\n")
            added += len(new)
    return added, left_out


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
        raise ArmError("the task has no model/: the arm works from the model")
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
        new_paths, left_out = add_new_paths(model, dot, files)
        rows = adopt(a, repo, env)
        compiles(a, repo, env, "the adopted start")
        listed = len(files) - rows
        adoption = {"commit": commit(repo, "Owner: adopt the reference start", env), "pageRows": listed, "unlinkedRows": rows,
                    "registry": "pages" if not rows else "unlinked" if not listed else "pages+unlinked", "baseModel": base_model.is_dir(),
                    "newPaths": new_paths, "newPathsLeftOut": left_out}
        if base_model.is_dir():
            copy_model(model, dot, keep_registry=True)
            options, mode = ["--base", adoption["commit"]], "adopted"
        else:
            options, mode = ["--full"], "adopted"
    commit(repo, "Owner: the model of this task", env)
    committed(repo, model, env)
    compiles(a, repo, env, "the owner's input")
    return options, mode, adoption


def verify_input(a, task, repo, env):
    """The owner's input for an independent `verify --full` of the start as it is: (mode, adoption record).

    A start that holds a model is checked against the task's model with its own registry pages. One that holds files
    but no model, such as an end state of a variant that keeps none, is adopted first: every file gets a row without
    links on the page of its owner. An empty start gets the model alone. Every owner commit must compile."""
    dot, model = repo / ".markitect", task / "model"
    if not model.is_dir():
        raise ArmError("the task has no model/: verify checks the start against it")
    files = [p for p in git(repo, "ls-files", env=env).splitlines() if p]
    adoption = None
    if dot.is_dir():
        copy_model(model, dot, keep_registry=True)
        mode = "verify"
    elif not files:
        copy_model(model, dot, keep_registry=False)
        mode = "verify-empty"
    else:
        copy_model(model, dot, keep_registry=False)
        rows = adopt(a, repo, env)
        compiles(a, repo, env, "the adopted start")
        listed = len(files) - rows
        adoption = {"commit": commit(repo, "Owner: adopt the start to verify", env), "pageRows": listed, "unlinkedRows": rows,
                    "registry": "pages" if not rows else "unlinked" if not listed else "pages+unlinked", "baseModel": False}
        mode = "verify-adopted"
    commit(repo, "Owner: the model of this task", env)
    committed(repo, model, env)
    compiles(a, repo, env, "the owner's input")
    return mode, adoption


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


def sessions_of(result):
    """The Result's sessions, in the order they ended."""
    sessions = (result or {}).get("sessions")
    return [s for s in sessions if isinstance(s, dict)] if isinstance(sessions, list) else []


def reported_model(result, provider, model):
    """(provider, model, models used): the model every session that finished reported, or the configured one when none did.

    A session lists every model that answered, its subagents included. With more than one, `model` is the first of them and
    the list holds all, so that the suite's model check (suite.model_check) holds each against the evaluation's model. A
    session that did not finish may carry the configured name before its CLI reported one, so only finished sessions count."""
    finished = [s for s in sessions_of(result) if s.get("reason") == "finished"]
    named, providers = set(), set()
    for s in finished:
        agent = s["agent"] if isinstance(s.get("agent"), str) else ""
        listed = [m for m in s.get("models") or [] if isinstance(m, str) and m]
        named.update(listed or ([agent.partition(":")[2]] if ":" in agent else []))
        if ":" in agent:
            providers.add(agent.partition(":")[0])
    named = sorted(named - {""})
    if not named:
        return provider, model, []
    return (providers.pop() if len(providers) == 1 else provider), named[0], named if len(named) > 1 else []


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--markitect", required=True, help="the markitect program of the frozen revision")
    p.add_argument("--image", required=True, help="the image with the project's toolchain, by image ID, built FROM packaging/runtime")
    p.add_argument("--effort", default="high")
    p.add_argument("--verify", action="store_true", help="run `markitect verify --full` on the task's start instead of apply")
    p.add_argument("task")
    p.add_argument("out")
    p.add_argument("state", nargs="?")
    a = p.parse_args(argv)
    began = time.monotonic()
    task, out = pathlib.Path(a.task), pathlib.Path(a.out)
    out.mkdir(parents=True, exist_ok=True)
    report = {"arm": "markitect", "effort": a.effort, "command": "verify" if a.verify else "apply"}

    def finish(code, **facts):
        report.update(facts)
        (out / "run.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
        return code

    try:
        token = owner_token()
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
            if a.verify:
                (mode, adoption), options = verify_input(a, task, repo, env), ["--full"]
            else:
                options, mode, adoption = owner_input(a, task, repo, start, state.get("landed"), env)
            report.update(mode=mode, adoption=adoption, ownerCommit=git(repo, "rev-parse", "HEAD", env=env))
            git(repo, "push", "-q", "origin", "main", env=env)
            seconds = int(deadline - (time.monotonic() - began) - MARGIN_SECONDS)
            if seconds < 60:
                raise ArmError(f"only {seconds} s of the deadline are left for the Command")
            env_apply = host_env(a, provider, model, out / "transcripts", tmp / "hooks", token)
            (out / "results").mkdir(exist_ok=True)
            result_file = out / "results" / f"{name}.json"
            command = report["command"]
            argv = [a.markitect, command, "--repo", str(repo), "--run", name, "--deadline", f"{seconds}s", "--json", *options]
            report[command] = argv[1:]
            with open(result_file, "wb") as stdout, open(out / f"{command}.stderr.log", "wb") as stderr:
                code = subprocess.run(argv, stdout=stdout, stderr=stderr, stdin=subprocess.DEVNULL, env=env_apply).returncode
            report[f"{command}Exit"] = code
            try:
                result = json.loads(result_file.read_text(encoding="utf-8"))
            except ValueError:
                result_file.unlink()
                return finish(2, status="could-not-run", cause=f"markitect {command} returned no Result (see {command}.stderr.log)")
            sessions = sessions_of(result)
            report["provider"], report["model"], used = reported_model(result, provider, model)
            if used:
                report["modelsUsed"] = used
            failed = [{k: s.get(k) for k in ("role", "round", "class", "detail")} for s in sessions if s.get("reason") == "failed"]
            report.update(status=result.get("state"), cause=result.get("why"), tokens=result.get("usage"),
                          starts=len(sessions), failedStarts=failed, cleanup=result.get("cleanup"))
            # What a landed Command's cleanup after landing left: definite leftovers, which the suite counts as residue. A
            # host that predates `afterLanding` says nothing: that is not reported, not clean. `cleanup` lists what may
            # remain, recorded above but not counted: the engine's own census finds what really does.
            after = result.get("afterLanding") if isinstance(result.get("afterLanding"), list) else None
            report.update(afterLanding=after, afterLandingReported=result.get("outcome") != "landed" or after is not None,
                          leftovers=[str(x) for x in after or []])
            ran = any(s.get("reason") in ("finished", "deadline") for s in sessions)
            if result.get("state") == "check-failed" and not ran:
                # The host found a compile error in the start, which compiled above: no agent started, nothing is the arm's work.
                # Required checks that still fail after the repair rounds end a Command as `refused`, which is the agents' work.
                return finish(2)
            ended = CHECKED if a.verify else DELIBERATE
            if result.get("state") not in ended and not ran:
                # No session finished: the Command ended on its machinery, not on any agent's work.
                return finish(2, noAgentTurn=True)
            if a.verify:
                # Verify changes nothing: the start is delivered as it was, and no landing is recorded.
                bundle(target, start, out / "result.bundle", env)
                code = 0 if result.get("state") in CHECKED else 1
                report.update(findings=len(result.get("findings") or []), exit=code)
                return finish(code)
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
    except (ArmError, OSError, ValueError, KeyError, TypeError) as e:
        if (out / "result.bundle").exists():
            report["recordError"] = f"{type(e).__name__}: {e}"
            return finish(report.get("exit", 1))
        return finish(2, status="could-not-run", cause=f"{type(e).__name__}: {e}")


if __name__ == "__main__":
    sys.exit(main())
