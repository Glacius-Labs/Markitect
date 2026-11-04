"""The pilot's arms S and F: Markitect's model and checks without Manager agents, Git handoffs or containers of its own.

  python3 -I -B one_agent_arm.py --markitect BIN [--shape single|federal] [--claude BIN] [--credentials FILE] [--effort LEVEL]
                                 [--rounds N] TASK_DIR OUT_DIR [STATE_DIR]

Run it through the suite's sandbox, as K: the suite's one container per launch wraps it, and it starts none. One graph:

  impact -> make -> gate -> fresh check -> land, with a bounded loop from a refusal back to make

The host, out of the agents' reach, holds impact and the generated prompts; the gate after make has ended (the model
untouched, a verdict per item, compile, required checks); starting the checker; the loop's limit; all Git; and the
landing. An agent edits files and writes its answer to a file in `.arm/`, which Git excludes, so it never lands.

- S (`single`): make is one session with the whole checklist and its context.
- F (`federal`): make is one prompt per working Package with its bounded context, in parallel in one folder, and after
  they have all ended one Root prompt that integrates. A rejection goes back to Root alone.

The checker is a fresh session the host starts, never a subagent of the maker. Agents may use subagents, as K may. It
answers each verdict `confirmed`, `wrong`, `not-shown` or `waived`. Only `wrong` refuses the work; `not-shown` and
`waived` land and are recorded as findings, so that a person sees them. A ruling stands for the Run: a later round
re-checks only what was refused and the items whose verdict the maker changed since, and a later checker reopens another
item only if the diff since the last review breaks it. A `model-problem` verdict the checker confirms ends the Run at once
without landing, and any other answer to one refuses it, since nothing was made for that item. Exit codes and the record
are the 2.0 arm's (markitect_arm.py), so the suite classes and reads these arms as it does that one.
"""
import argparse
import concurrent.futures
import json
import os
import pathlib
import re
import shutil
import signal
import subprocess
import sys
import tempfile
import time

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import claude_code as cc  # noqa: E402
import classify  # noqa: E402
import markitect_arm as arm  # noqa: E402

TOOLS = ["Bash", "Read", "Edit", "Write", "Grep", "Glob", "Agent", "TodoWrite"]
DENIED = ["WebFetch", "WebSearch"]
ROOT = ""
MARGIN = 120
MAX_INLINE_DIFF = 64 << 10
LINK = re.compile(r"\[\[([^\]|#]+)")
EDITABLE = re.compile(r"^\.markitect/(registry\.md$|registry/)")
PROVIDER_CLASS = {"rate_limit": "rate-limit", "overloaded": "capacity", "overloaded_error": "capacity", "server_error": "server-error",
                  "api_error": "server-error", "authentication_failed": "login", "billing_error": "login",
                  "status 429": "rate-limit", "status 529": "capacity", "status 401": "login", "status 403": "login"}
ANSWERS = ("confirmed", "wrong", "not-shown", "waived")
RECORDED = ("not-shown", "waived")

COVERS = """## What a verdict covers

- **The model, in full.** A verdict covers the item's whole Definition, its examples included, and the Definitions it links to. Where they contradict each other, nothing can conform, and the only right verdict is `model-problem`. It is no review of the whole model: stay with each item and its links.
- **Evidence.** For `conforms` and `changed`, the evidence names how each example in the item's Definition is met, for example by which test.
"""
REPOSITORY = """## The repository

- The model is in `.markitect/model/`. It, the manifest, the lock and `execution.yaml` are the owner's: never change them. A gap without effect at the project boundary is yours to decide and document.
- Every file needs a row on a registry page, `.markitect/registry.md` or `.markitect/registry/PACKAGE.md`, on the page of its owner: the Package whose `paths` pattern matches it most closely, else Root. `markitect check --repo .` says what is missing.
- Do not commit: the host commits your work when you have ended.
"""
ANSWER = """# Answer

Write your verdicts to `{file}`, which the host reads and never commits, as JSON:

{{"verdicts": [{{"item": "ITEM", "verdict": "conforms", "evidence": "..."}}]}}

One entry per item you answer for. The verdict is `conforms` (it already conformed; you changed nothing for it), `changed` (you changed files so that it conforms) or `model-problem` (the model contradicts itself or leaves open behavior at the project boundary for this item, so it cannot be settled within the model). When you end, the host runs the required checks on the whole repository, and a fresh checker re-checks your verdicts.
"""


class ArmError(Exception):
    pass


def fence(text):
    """A code fence longer than any run of backticks in the text."""
    return "`" * max([3] + [len(run) + 1 for run in re.findall(r"`{3,}", text)])


def model_text(repo, ids):
    """The full text of these Definitions and of those they link to, once each; Library Definitions have no file here."""
    root, order = repo / ".markitect" / "model", []
    for wanted in ids:
        path = root / (wanted + ".md")
        for name in [wanted] + (LINK.findall(path.read_text(encoding="utf-8")) if path.is_file() else []):
            if name not in order and (root / (name + ".md")).is_file():
                order.append(name)
    parts = []
    for name in order:
        text = (root / (name + ".md")).read_text(encoding="utf-8").rstrip("\n")
        marks = fence(text)
        parts.append(f"### `{name}`\n\n{marks}markdown\n{text}\n{marks}\n")
    return "\n".join(parts)


def stale_item(s):
    subject = s.get("target") if s.get("code") == "stale.definition" else s.get("file")
    return f"{s.get('code')} {subject or ''} on {s.get('pageFile')}".replace("  ", " ")


def checklist(impact):
    """Each reached Definition once, with no inherited duplicates, and each Finding and stale entry: its item, its
    Package, whether it is the Package's own or inherited for others, and why it is on the list."""
    items, inherited = {}, {}
    for c in impact.get("checklists") or []:
        for i in c.get("own") or []:
            items[i["definition"]] = {"item": i["definition"], "package": c["package"], "why": "; ".join(i.get("reasons") or [])}
        inherited[c["package"]] = [i["definition"] for i in c.get("inherited") or []]
        for n, f in enumerate(c.get("findings") or []):
            name = f"finding {n + 1} on {f.get('definition')}"
            items[name] = {"item": name, "package": c["package"], "why": f"observed {f.get('observed')}; expected {f.get('expected')}"}
        for s in c.get("stale") or []:
            items[stale_item(s)] = {"item": stale_item(s), "package": c["package"], "why": "a stale registry entry"}
    return list(items.values()), inherited


def table(items):
    rows = "".join(f"| `{i['item']}` | {i['why'].replace('|', ' ')} |\n" for i in items)
    return "## Checklist\n\n| Item | Why |\n|---|---|\n" + rows


def required_checks(repo):
    """The required checks of `execution.yaml`, as flat as the model language writes them."""
    path = repo / ".markitect" / "execution.yaml"
    blocks = re.split(r"^\s*-\s+", path.read_text(encoding="utf-8"), flags=re.M)[1:] if path.is_file() else []
    return [dict(re.findall(r"^\s*(\w+):[ \t]*(.+?)\s*$", block, re.M)) for block in blocks]


def checks_table(checks):
    rows = "".join(f"| {c.get('name')} | `{c.get('command')}` | `{c.get('directory', '.')}` |\n" for c in checks)
    return "## Required checks\n\n| Check | Command | Directory |\n|---|---|---|\n" + rows


def seconds_of(text):
    match = re.fullmatch(r"(\d+)([smh])", str(text or "").strip())
    return int(match[1]) * {"s": 1, "m": 60, "h": 3600}[match[2]] if match else 600


def run_session(argv, prompt, cwd, env, stream, stderr, seconds):
    """One session, in a process group of its own: stopping what it left behind must not stop the sessions beside it."""
    started = time.monotonic()
    with open(prompt, "rb") as inp, open(stream, "wb") as out, open(stderr, "wb") as err:
        proc = subprocess.Popen(argv, cwd=cwd, env=env, stdout=out, stderr=err, stdin=inp, start_new_session=os.name != "nt")
        try:
            code, timed_out = proc.wait(timeout=seconds), False
        except subprocess.TimeoutExpired:
            code, timed_out = None, True
        if os.name == "nt":
            subprocess.run(["taskkill", "/PID", str(proc.pid), "/T", "/F"], capture_output=True)
        else:
            try:
                os.killpg(proc.pid, signal.SIGKILL)
            except (ProcessLookupError, PermissionError):
                pass
        try:
            proc.wait(timeout=15)
        except subprocess.TimeoutExpired:
            pass
    return {"exitCode": proc.returncode if timed_out else code, "timedOut": timed_out, "sessionSeconds": round(time.monotonic() - started, 1)}


class Host:
    """What the host holds across one launch: the repository, the sessions it started and what they answered."""

    def __init__(self, a, tmp, repo, out, env, session_env, login, model, deadline, secrets):
        self.a, self.tmp, self.repo, self.out, self.env, self.session_env = a, tmp, repo, out, env, session_env
        self.login, self.model, self.deadline, self.secrets = login, model, deadline, secrets
        self.starts, self.notes = [], []
        self.rulings = {}
        self.answers = repo / ".arm"
        self.answers.mkdir()
        with open(repo / ".git" / "info" / "exclude", "a", encoding="utf-8") as f:
            f.write("\n/.arm/\n")

    def left(self):
        return self.deadline - time.monotonic() - MARGIN

    def session(self, name, manager, role, attempt, prompt, seconds=None):
        """One headless session, up to the time left or `seconds`: its record goes into the Result's starts, in the order they ended."""
        if self.left() <= 0:
            self.starts.append({"manager": manager or ".", "role": role, "round": attempt, "reason": "deadline", "agent": f"claude:{self.model}"})
            return "deadline"
        config, folder = self.tmp / "config" / name, self.out / "sessions" / name
        config.mkdir(parents=True)
        folder.mkdir(parents=True)
        (config / ".credentials.json").write_bytes(self.login)
        (self.tmp / "prompts").mkdir(exist_ok=True)
        path = self.tmp / "prompts" / f"{name}.txt"
        path.write_text(prompt, encoding="utf-8")
        argv = [self.a.claude, "-p", "--model", self.model, "--output-format", "stream-json", "--verbose", "--strict-mcp-config",
                "--setting-sources", "project", "--allowedTools", *TOOLS, "--disallowedTools", *DENIED]
        if self.a.effort:
            argv += ["--effort", self.a.effort]
        try:
            record = run_session(argv, path, self.repo, dict(self.session_env, CLAUDE_CONFIG_DIR=str(config)), folder / "session.stream.jsonl",
                                 folder / "session.stderr.log", min(self.left(), seconds or self.left()))
        finally:
            (config / ".credentials.json").unlink(missing_ok=True)
        facts = cc.stream_facts(folder / "session.stream.jsonl")
        if (config / "projects").is_dir():
            shutil.copytree(config / "projects", self.out / "transcripts" / name)
        errors = classify.session_errors(folder)
        reason = "deadline" if record["timedOut"] else "failed" if errors or record["exitCode"] != 0 else "finished"
        start = {"manager": manager or ".", "role": role, "round": attempt, "reason": reason, "agent": f"claude:{facts['model'] or self.model}",
                 "seconds": record["sessionSeconds"], "usage": cc.usage(facts["result"])}
        if errors:
            start.update({"class": PROVIDER_CLASS.get(errors[-1]["error"], "server-error"), "detail": str(errors[-1].get("apiError") or errors[-1]["error"])[:300]})
        self.starts.append(start)
        return reason

    def git(self, *args, check=True):
        return arm.git(self.repo, *args, env=self.env, check=check)

    def commit_work(self, message):
        """Commit what the agents left, as the baseline's capture does: their .gitignore stands, except for sources it would drop."""
        (self.repo / ".git" / "index.lock").unlink(missing_ok=True)
        self.git("add", "-A")
        ignored = self.git("ls-files", "-z", "-o", "-i", "--exclude-standard").split("\0")
        kept = [p for p in ignored if cc.KEPT_IGNORED.search(p)]
        if kept:
            self.git("add", "-f", "--", *kept)
        self.git("commit", "-q", "--allow-empty", "--no-verify", "-m", message)
        return self.git("rev-parse", "HEAD")

    def tidy(self):
        """Back to the commit: a checker changes nothing, and what its checks left is gone."""
        self.git("reset", "-q", "--hard")
        self.git("clean", "-q", "-fd")

    def run_checks(self, commit):
        """The required checks on the whole repository at the commit, in a clean checkout, so that they leave nothing behind."""
        checkout = self.tmp / f"checks-{commit[:12]}"
        self.git("worktree", "add", "-q", "--detach", str(checkout), commit)
        results = []
        try:
            for c in required_checks(self.repo):
                started = time.monotonic()
                try:
                    p = subprocess.run(c.get("command", ""), shell=True, cwd=checkout / c.get("directory", "."), env=self.session_env, capture_output=True,
                                       text=True, errors="replace", timeout=min(seconds_of(c.get("timeout")), max(self.left(), 1)))
                    passed, output = p.returncode == 0, (p.stdout + p.stderr)[-2000:]
                except subprocess.TimeoutExpired:
                    passed, output = False, "the check timed out"
                except OSError as e:
                    # Its directory may not exist yet: that is the work's state, not the host's failure.
                    passed, output = False, f"the check could not run: {e}"
                results.append({"name": c.get("name"), "passed": passed, "seconds": round(time.monotonic() - started, 1), "output": output})
        finally:
            self.git("worktree", "remove", "--force", str(checkout), check=False)
        return results


def read_verdicts(path, known):
    """A maker's answer: {item: (verdict, evidence)} for items on the checklist, and what is wrong with the file."""
    try:
        raw = json.loads(pathlib.Path(path).read_text(encoding="utf-8"))
        entries = raw["verdicts"]
        if not isinstance(entries, list):
            raise TypeError
    except FileNotFoundError:
        return {}, []
    except (ValueError, KeyError, TypeError) as e:
        return {}, [f"{pathlib.Path(path).name} is no verdict file: {type(e).__name__}"]
    found, problems = {}, []
    for e in entries:
        item = e.get("item") if isinstance(e, dict) else None
        if item not in known:
            problems.append(f"{item!r} is not on the checklist")
        elif e.get("verdict") not in ("conforms", "changed", "model-problem"):
            problems.append(f"{item}: {e.get('verdict')!r} is no verdict")
        elif not str(e.get("evidence") or "").strip():
            problems.append(f"{item}: the verdict has no evidence")
        else:
            found[item] = {"verdict": e["verdict"], "evidence": e["evidence"]}
    return found, problems


def gate(host, items, verdicts, owner, problems):
    """After make has ended: the host commits, and refuses work that touches the model, misses a verdict, does not compile or
    fails a required check. Returns the commit, the checks' results and the refusal's reasons."""
    commit = host.commit_work(f"Work of attempt {len([s for s in host.starts if s['role'] == 'maker'])}")
    wrong = list(problems)
    changed = host.git("diff", "--name-only", owner, commit).splitlines()
    wrong += [f"{p} is protected: the model, the manifest, the lock, execution.yaml and the rest of .markitect/ change only through the owner"
              for p in changed if p.startswith(".markitect/") and not EDITABLE.match(p)]
    wrong += [f"{i['item']} has no verdict" for i in items if i["item"] not in verdicts]
    report, code = arm.check(host.a, host.repo, host.env)
    if code != 0:
        wrong += [f"`markitect check`: {d.get('code')} {d.get('file') or ''}: {d.get('message')}" for d in (report.get("errors") or [])[:30]]
    checks = host.run_checks(commit)
    wrong += [f"the required check {c['name']} failed:\n{c['output']}" for c in checks if not c["passed"]]
    return commit, checks, wrong


def maker_prompt(host, items, scope, focus, file, defects=None, reports=None):
    """A maker's prompt: its scope, its checklist with why, what a verdict covers, the repository, the model's text and the answer."""
    parts = ["# Task\n\n" + scope + "\n"]
    if items:
        parts += [table(items)]
    parts += [COVERS, REPOSITORY, checks_table(required_checks(host.repo))]
    if reports:
        parts += ["## What the Packages reported\n\n" + reports]
    if defects:
        parts += ["## Why the last attempt was refused\n\nThe work so far is in the repository. Fix these and answer again for each item named here:\n\n" + defects + "\n"]
    parts += ["# Model\n\n" + (model_text(host.repo, focus) or "Nothing is reached.\n"), ANSWER.format(file=file)]
    return "\n".join(parts)


def checker_prompt(host, items, verdicts, checks, base, commit, file, owner=None, refused=None):
    """The checker's prompt for these items. `base` is what an earlier review saw, and then `owner` and what it refused are named too."""
    diff = host.git("diff", base, commit)
    if len(diff) <= MAX_INLINE_DIFF:
        marks = fence(diff)
        shown = f"{marks}diff\n{diff}\n{marks}\n"
    else:
        names = "\n".join(f"- {p}" for p in host.git("diff", "--name-only", base, commit).splitlines())
        shown = f"The diff is larger than {MAX_INLINE_DIFF >> 10} KiB. `git diff {base} {commit}` shows it. Changed files:\n\n{names}\n"
    rows = "".join(f"| `{i['item']}` | {verdicts[i['item']]['verdict']} | {str(verdicts[i['item']]['evidence']).replace('|', ' ')} |\n" for i in items)
    ran = "".join(f"- {c['name']}: {'passed' if c['passed'] else 'failed'}\n" for c in checks) or "- none\n"
    earlier = ""
    if owner:
        named = {i["item"] for i in items}
        others = ", ".join(f"`{item}`" for item in verdicts if item not in named) or "none"
        earlier = ("## What an earlier review ruled\n\nAn earlier review of this Run ruled on the other items, and those rulings stand. Re-check the items above: those it refused and "
                   "those whose verdict the maker has changed since. If the diff below breaks another item, add an entry for that item too; `.markitect/model/` holds its Definition. "
                   f"`git diff {owner} {commit}` shows the whole work.\n\nThe other items: {others}.\n\n")
        if refused:
            earlier += "It refused these:\n\n" + "".join(f"- `{item}`: {evidence}\n" for item, evidence in refused.items()) + "\n"
    return f"""# Task

Re-check each verdict below of the submission `{commit}` against the model and the repository, which holds that commit. Change no file. The question is whether the realization does what the Definition says, examples included, so read the code and run what you need; the maker's evidence is only where you start. Answer each verdict with one of these:

- `confirmed`: what you read or ran shows that it holds.
- `wrong`: it does not hold. The realization does something the Definition forbids, leaves out something the Definition says, or matches part of the Definition but not its example. Say what and where. On a `conforms` or `changed` verdict, this is the only answer that refuses the work.
- `not-shown`: neither the evidence nor what you read or ran shows it either way. The work lands, and this is recorded for a person to see.
- `waived`: the realization departs from the letter of the Definition, or the Definition can be read two ways and the realization takes one, and you do not hold it against the work. Say what and why. The work lands, and this is recorded for a person to see.

Where the Definition and what it links to contradict each other, nothing can conform: the maker's only right verdict is `model-problem`, so answer `wrong` to a verdict that claims the item conforms, and never `waived`. Answer a `model-problem` verdict `confirmed` only if the model really contradicts itself or leaves open behavior at the project boundary for this item, and that ends the Run without landing. Otherwise answer it `wrong`: any answer but `confirmed` refuses a `model-problem` verdict, since nothing was made for that item.

{table(items)}
## Submitted verdicts

| Item | Verdict | Evidence |
|---|---|---|
{rows}
{earlier}## Required checks on the whole repository

{ran}
## Diff from `{base}` to `{commit}`

{shown}
# Model

{model_text(host.repo, [i['item'] for i in items])}
# Answer

Write your review to `{file}`, which the host reads and never commits, as JSON:

{{"reviews": [{{"item": "ITEM", "answer": "confirmed", "evidence": "..."}}]}}

One entry per verdict, each with evidence: what you read or ran, and for any answer but `confirmed` what you found and where.
"""


def read_reviews(path, items, known):
    """A checker's answer: ({item: its answer and evidence} for any item of the checklist, the items of `items` it left out)."""
    try:
        entries = json.loads(pathlib.Path(path).read_text(encoding="utf-8"))["reviews"]
        found = {e["item"]: {"answer": e["answer"], "evidence": e.get("evidence") or ""} for e in entries
                 if isinstance(e, dict) and e.get("item") in known and e.get("answer") in ANSWERS}
    except (OSError, ValueError, KeyError, TypeError):
        found = {}
    return found, [i["item"] for i in items if i["item"] not in found]


def findings_of(rulings):
    """What lands for a person to see: the rulings that are neither a confirmation nor a refusal."""
    return [{"definition": item, "answer": r["answer"], "observed": r["evidence"]} for item, r in rulings.items() if r["answer"] in RECORDED]


def fan_out(host, items, inherited, attempt):
    """F's make: one prompt per working Package other than Root, in parallel in the one folder; every one has ended when this returns."""
    packages = [p for p in sorted({i["package"] for i in items} | {p for p, names in inherited.items() if names}) if p != ROOT]

    def one(package):
        mine = [i for i in items if i["package"] == package]
        name = "maker-" + package.replace("/", "-")
        scope = (f"You work for the Package `{package}`, one of {len(packages)} Packages working at the same time in this one folder, each on its own part. "
                 "Make your Package's part of the repository conform to every item below, and give each item a verdict. A verdict means: conforms to this item and to "
                 "the Rules above it. Change only your Package's files, and create what the model says belongs to it. Others are writing theirs beside you, so the whole "
                 "program may not build until everyone has ended: build and test your own package. When all of you have ended, one integrator makes the whole work.")
        rules = inherited.get(package) or []
        if rules:
            scope += " These changed Rules above you bind your code too; the integrator answers for them: " + ", ".join(f"`{r}`" for r in rules) + "."
        file = host.answers / f"{name}-{attempt}.json"
        # The integrator after them needs time too.
        host.session(name, package, "maker", attempt, maker_prompt(host, mine, scope, [i["item"] for i in mine] + rules, file), host.left() * 0.55)
        return file

    with concurrent.futures.ThreadPoolExecutor(max_workers=max(len(packages), 1)) as pool:
        return list(pool.map(one, packages))


def graph(host, shape, items, inherited, owner, rounds):
    """make -> gate -> check, looping on a refusal: (state, why, verdicts, checks, the last commit, the refusals)."""
    known = {i["item"] for i in items}
    verdicts, refusals, defects, reports, attempts, attempt, commit, checks = {}, [], None, None, 0, 0, None, []
    reviewed, last_refused = None, None
    while True:
        attempt += 1
        attempts += 1
        files = []
        if shape == "federal" and attempt == 1:
            for file in fan_out(host, items, inherited, attempt):
                files.append(file)
            if host.left() <= 0:
                return "incomplete", "the deadline passed during make", verdicts, checks, commit, refusals
            failed = [s for s in host.starts if s.get("class")]
            if failed:
                return "incomplete", "a provider failure ended a maker: " + failed[-1].get("detail", ""), verdicts, checks, commit, refusals
            found = {}
            for file in files:
                found.update(read_verdicts(file, known)[0])
            reports = "".join(f"- `{item}`: {v['verdict']}: {v['evidence']}\n" for item, v in sorted(found.items())) or "Nothing yet.\n"
            root = [i for i in items if i["package"] == ROOT]
            scope = ("Every Package has ended; their work is in this folder, and their verdicts stand unless you answer for an item yourself. Integrate it: make the whole "
                     "program build and work as the model says, and the required checks pass, using the Packages' code, not building it a second time. You may change any file "
                     "but the model. Give a verdict for each item below, and for any item of a Package whose verdict no longer holds.")
            focus = [i["item"] for i in items]
        elif attempt > 1:
            root, scope, focus = items, "The work so far did not pass. Fix what is named below, in any file but the model.", [i["item"] for i in items]
        else:
            root = items
            scope = ("Make the repository conform to every item on your checklist, and give each item a verdict. A verdict means: conforms to this item and to the Rules above it, "
                     "so a change that breaks an unchanged Rule does not conform.")
            focus = [i["item"] for i in items]
        file = host.answers / f"maker-{attempt}.json"
        reason = host.session(f"maker-{attempt}", ROOT, "maker", attempt, maker_prompt(host, root, scope, focus, file, defects, reports))
        files.append(file)
        if reason == "deadline":
            return "incomplete", "the deadline passed during make", verdicts, checks, commit, refusals
        if reason == "failed" and host.starts[-1].get("class"):
            return "incomplete", "a provider failure ended the maker: " + host.starts[-1].get("detail", ""), verdicts, checks, commit, refusals
        problems = []
        for path in files:
            found, wrong = read_verdicts(path, known)
            verdicts.update(found)
            problems += wrong
        commit, checks, refused = gate(host, items, verdicts, owner, problems)
        if refused:
            refusals.append({"attempt": attempt, "by": "gate", "reasons": refused})
            # What the last review refused is not fixed yet, whatever the gate says on top of it.
            defects = "\n".join([f"- {r}" for r in refused] + [f"- `{item}`: {evidence}" for item, evidence in (last_refused or {}).items()])
        else:
            listed = [i for i in items if (host.rulings.get(i["item"]) or {}).get("verdict") != verdicts[i["item"]]["verdict"]]
            review, missing, tries = {}, listed, 0
            while missing and tries < 2:
                tries += 1
                path = host.answers / f"checker-{attempt}-{tries}.json"
                prompt = checker_prompt(host, missing, verdicts, checks, reviewed or owner, commit, path, owner if reviewed else None, last_refused)
                if tries > 1:
                    prompt += "\nThe last review left out these items, and an entry for each is needed: " + ", ".join(f"`{m['item']}`" for m in missing) + ".\n"
                reason = host.session(f"checker-{attempt}-{tries}", ROOT, "checker", attempt, prompt)
                host.tidy()
                if reason == "deadline":
                    return "incomplete", "the deadline passed during the check", verdicts, checks, commit, refusals
                if reason == "failed" and host.starts[-1].get("class"):
                    return "incomplete", "a provider failure ended the checker: " + host.starts[-1].get("detail", ""), verdicts, checks, commit, refusals
                found, _ = read_reviews(path, missing, known)
                # The first answer to an item stands: a second session is here only for what the first left out.
                review = {**found, **review}
                missing = [i for i in listed if i["item"] not in review]
            if missing:
                return "incomplete", "the checker left no complete review twice in a row", verdicts, checks, commit, refusals
            reviewed = commit
            for item, r in review.items():
                host.rulings[item] = {**r, "verdict": verdicts[item]["verdict"]}
            # Nothing was made for a model-problem item, so any answer but a confirmation refuses it.
            last_refused = {item: r["evidence"] for item, r in review.items()
                            if r["answer"] == "wrong" or (verdicts[item]["verdict"] == "model-problem" and r["answer"] != "confirmed")}
            for item in last_refused:
                host.rulings.pop(item)
            if last_refused:
                refusals.append({"attempt": attempt, "by": "checker", "items": last_refused})
            problems = [item for item, r in review.items() if r["answer"] == "confirmed" and verdicts[item]["verdict"] == "model-problem"]
            if problems:
                return "model-defect", "the checker confirmed a model problem: " + ", ".join(sorted(problems)) + "; the Run ends without landing", verdicts, checks, commit, refusals
            if not last_refused:
                return "landed", "the checker refused nothing", verdicts, checks, commit, refusals
            defects = "\n".join(f"- `{item}`: {evidence}" for item, evidence in last_refused.items())
            for item in last_refused:
                verdicts.pop(item, None)
        reports = None
        if attempts >= rounds:
            return "incomplete", f"the maker used its {rounds} attempts: the checker or the gate refused its work after each", verdicts, checks, commit, refusals


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--markitect", required=True, help="the markitect program for this platform, for check and impact only")
    p.add_argument("--shape", choices=("single", "federal"), default="single")
    p.add_argument("--claude", default=shutil.which("claude") or "claude")
    p.add_argument("--credentials", default=os.environ.get("EVALUATION_LOGIN") or str(pathlib.Path.home() / ".claude" / ".credentials.json"))
    p.add_argument("--effort", default="high")
    p.add_argument("--rounds", type=int, default=6)
    p.add_argument("task")
    p.add_argument("out")
    p.add_argument("state", nargs="?")
    a = p.parse_args(argv)
    began = time.monotonic()
    task, out = pathlib.Path(a.task), pathlib.Path(a.out)
    out.mkdir(parents=True, exist_ok=True)
    report = {"arm": "one-agent", "shape": a.shape, "effort": a.effort, "command": "apply"}

    def finish(code, **facts):
        report.update(facts)
        (out / "run.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
        return code

    try:
        provider, model = arm.provider_model()
        if provider != "claude":
            raise ArmError("this pilot runs on Claude only")
        report.update(provider=provider, model=model)
        spec = json.loads((task / "task.json").read_text(encoding="utf-8"))
        start, deadline = spec["start"]["commit"], began + spec["deadlineSeconds"]
        state = arm.read_state(a.state)
        name = arm.run_name()
        report["run"] = name
        login, secrets = cc.session_login(pathlib.Path(a.credentials).read_bytes(), spec["deadlineSeconds"])
        with tempfile.TemporaryDirectory(prefix="one-agent-arm-", ignore_cleanup_errors=True) as tmp:
            tmp = pathlib.Path(tmp).resolve()
            for folder in ("hooks", "home", "bin"):
                (tmp / folder).mkdir()
            env = arm.git_env(tmp / "hooks")
            # The agents reach `markitect check` by name; a mounted program may not be executable as it is.
            program = pathlib.Path(a.markitect)
            local = tmp / "bin" / program.name
            shutil.copyfile(program, local)
            local.chmod(0o755)
            a.markitect = str(local)
            session_env = cc.base_env(tmp / "home", tmp / "hooks")
            session_env["PATH"] = str(tmp / "bin") + os.pathsep + session_env.get("PATH", "")
            repo = tmp / "work"
            arm.git(tmp, "clone", "-q", "--no-checkout", str((task / "start.bundle").resolve()), str(repo), env=env)
            arm.git(repo, "config", "core.autocrlf", "false", env=env)
            arm.git(repo, "checkout", "-q", "-B", "main", start, env=env)
            options, mode, adoption = arm.owner_input(a, task, repo, start, state.get("landed"), env)
            owner = arm.git(repo, "rev-parse", "HEAD", env=env)
            report.update(mode=mode, adoption=adoption, ownerCommit=owner)
            base = options[1] if options[:1] == ["--base"] else start
            probe = subprocess.run([a.markitect, "impact", "--repo", str(repo), "--base", base, *(["--full"] if "--full" in options else []), "--json"],
                                   capture_output=True, text=True, encoding="utf-8", env=env)
            if probe.returncode != 0:
                raise ArmError(f"markitect impact (exit {probe.returncode}): {probe.stderr.strip()[-500:]}")
            items, inherited = checklist(json.loads(probe.stdout))
            report["items"] = len(items)
            host = Host(a, tmp, repo, out, env, session_env, login, model, deadline, secrets)
            commit, checks, refusals, verdicts = None, [], [], {}
            if not items:
                status, why = "nothing-to-do", "nothing the change reaches: no agent starts"
            else:
                status, why, verdicts, checks, commit, refusals = graph(host, a.shape, items, inherited, owner, a.rounds)
            landed = commit if status == "landed" else None
            result = {"format": 1, "run": name, "command": "apply", "outcome": "landed" if landed else "not-landed", "state": status, "why": why, "shape": a.shape,
                      "base": base, "start": owner, "landed": landed, "requiredChecks": {"ran": commit is not None, "wholeRepository": commit is not None, "checks": checks},
                      "items": [{"manager": i["package"] or ".", "item": i["item"], "part": "own", "verdict": (verdicts.get(i["item"]) or {}).get("verdict"),
                                 "evidence": (verdicts.get(i["item"]) or {}).get("evidence"), "checker": "checker",
                                 "approved": status in ("landed", "model-defect") and i["item"] in host.rulings,
                                 "review": (host.rulings.get(i["item"]) or {}).get("answer")} for i in items],
                      "notParticipating": [], "findings": findings_of(host.rulings) if status == "landed" else [], "refusals": refusals, "starts": host.starts}
            (out / "results").mkdir(exist_ok=True)
            (out / "results" / f"{name}.json").write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
            report["provider"], report["model"] = arm.reported_model(result, provider, model)
            failed = [{k: s.get(k) for k in ("manager", "role", "round", "class", "detail")} for s in host.starts if s.get("reason") == "failed"]
            costs = [(s.get("usage") or {}).get("costUsd") for s in host.starts]
            report["usage"] = {"costUsd": round(sum(c for c in costs if c), 6) if any(costs) else None, "sessions": len(host.starts)}
            report.update(status=status, cause=why, starts=len(host.starts), failedStarts=failed, refusals=len(refusals), findings=len(result["findings"]), landed=landed,
                          leftovers=[], afterLanding=[], afterLandingReported=True, cleanup=[])
            if status not in arm.DELIBERATE and not any(s.get("reason") in ("finished", "deadline") for s in host.starts):
                return finish(2, noAgentTurn=True)
            history = arm.git(repo, "log", "-p", "--format=%H%n%B", f"{start}..{landed or start}", env=env)
            if any(s.decode() in history for s in secrets) or cc.contains_secret(out, secrets):
                for path in (out / "transcripts", out / "sessions", out / "results"):
                    if path.is_dir():
                        shutil.rmtree(path)
                return finish(1, status="quarantined", quarantine=True)
            arm.bundle(repo, landed or start, out / "result.bundle", env)
            code = 0 if status in arm.DELIBERATE else 1
            report["exit"] = code
            if landed and a.state:
                (pathlib.Path(a.state) / arm.STATE_FILE).write_text(json.dumps({"landed": landed, "run": name}, indent=2) + "\n", encoding="utf-8")
            return finish(code)
    except (ArmError, arm.ArmError, cc.VariantError, OSError, ValueError, KeyError, TypeError, subprocess.SubprocessError) as e:
        if (out / "result.bundle").exists():
            report["recordError"] = f"{type(e).__name__}: {e}"
            return finish(report.get("exit", 1))
        return finish(2, status="could-not-run", cause=f"{type(e).__name__}: {e}")


if __name__ == "__main__":
    sys.exit(main())
