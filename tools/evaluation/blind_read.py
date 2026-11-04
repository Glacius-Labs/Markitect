"""Blind reads of kept results: fresh reader sessions, neutral labels, the same questions (concept 3.0 §13).

  python -I -B blind_read.py prepare --store DIR --names NAME,NAME,... --requirements FILE --out NEW_DIR
  python -I -B blind_read.py run --material DIR --sandbox IMAGE_ID --variant-code tools/evaluation [--login FILE] [--series FILE]
                                 [--model claude:MODEL_ID] [--deadline-seconds N] [--stop-after-points N] [--lock FILE]

`prepare` gives each result a random label and a copy of its program without anything that names how it was made: no
history, no `.markitect/`, no agent notes and no compiled files. The key stays in `key.json` beside the material, which no
session sees, and a person should not read it before the reads are through.

`run` starts one fresh session per label, one at a time, through the suite's sandbox and `reviewer.py`: Read, Glob and
Grep only, no memory, on one model. The same prompt goes to all. The reader sees one result and the requirements, and
never hears that there are others, so that what it says is about this program and not about a comparison. Each attempt
lands in `reads/LABEL/aN/`, whatever became of the session. Running it again starts only the labels without a valid read.
It stops before the next session once the weekly reading has risen by --stop-after-points, or when a request is refused.

A score here says where to look. The difference between two results is for the owner and the session to go through.
"""
import argparse
import json
import pathlib
import secrets
import shutil
import sys
import tempfile

# Isolated mode (-I) leaves the script's directory off the path.
sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import kept  # noqa: E402

LABELS = "ABCDEFGHJKLMNPQRSTUVWXYZ"
DEADLINE = 3000
STRIPPED_FOLDERS = {".git", ".markitect", ".arm", ".claude", ".codex"}
STRIPPED_FILES = {"task.md", "CLAUDE.md", "AGENTS.md", "GEMINI.md"}
BINARY_SUFFIXES = {".exe", ".dll", ".so", ".dylib", ".test", ".out", ".a"}
RUBRIC = """# Questions

Answer each from the program itself, as an experienced engineer would who has to take over this program and keep changing it for years.

1. **Architecture.** Do the modules have clear jobs and clean boundaries? Does a change stay in one place, or would it have to reach through several? Is anything in the wrong place?
2. **Naming.** Do files, types, functions and variables say what they are? Is one thing called one thing?
3. **Cleanliness.** Is the code as simple as the job allows? Is there repetition, dead code, needless layers, or code that only looks finished?
4. **Tests.** Would the tests catch a mistake in a change? Do they test behaviour, or only that code runs?
5. **Taking it over.** Would you want to maintain this program, and what would you change first?

For each of the first four, give a score from 1 (poor) to 5 (excellent), what is good and what is weak, and name files. Say only what you saw in a file you opened. Do not guess how the program was made.
"""
PROMPT = """Read the Go program in TOKEN_DIR. REQUIREMENTS holds what it was built to do, and RUBRIC holds the questions to answer about it. It passes its own tests and a separate set of behaviour tests, so do not look for whether it meets the requirements: judge how it is made.

You have Read, Glob and Grep. Open enough of the program to back what you say, starting from its largest packages and following what you find. Do not stop at the first files.

Your last message is only this JSON object:

{"architecture": {"score": 1, "good": ["..."], "weak": ["..."], "files": ["..."]},
 "naming": {"score": 1, "good": ["..."], "weak": ["..."], "files": ["..."]},
 "cleanliness": {"score": 1, "good": ["..."], "weak": ["..."], "files": ["..."]},
 "tests": {"score": 1, "good": ["..."], "weak": ["..."], "files": ["..."]},
 "takingOver": {"wouldMaintain": "yes, with doubts or no", "why": "...", "changeFirst": ["..."]},
 "filesOpened": 0}
"""


class ReadError(Exception):
    pass


def strip(tree):
    """Remove from an exported tree what names how it was made."""
    tree = pathlib.Path(tree)
    for p in sorted(tree.rglob("*"), reverse=True):
        rel = p.relative_to(tree)
        if any(part in STRIPPED_FOLDERS for part in rel.parts):
            p.rmdir() if p.is_dir() else p.unlink()
        elif p.is_file() and (p.name in STRIPPED_FILES or p.suffix.lower() in BINARY_SUFFIXES):
            p.unlink()


def prepare(store, names, requirements, out):
    store, out = pathlib.Path(store), pathlib.Path(out)
    if out.exists():
        raise ReadError(f"{out} exists")
    if len(names) > len(LABELS):
        raise ReadError(f"{len(names)} results, and only {len(LABELS)} labels")
    labels = list(LABELS[:len(names)])
    secrets.SystemRandom().shuffle(labels)
    key = {}
    (out / "material").mkdir(parents=True)
    for label, name in zip(labels, names):
        token = secrets.token_hex(6)
        dest = out / "material" / label
        (dest / token).parent.mkdir(parents=True)
        with tempfile.TemporaryDirectory(ignore_cleanup_errors=True) as tmp:
            kept.clone(store / name, pathlib.Path(tmp) / "repo")
            shutil.copytree(pathlib.Path(tmp) / "repo", dest / token, ignore=shutil.ignore_patterns(*STRIPPED_FOLDERS))
        strip(dest / token)
        shutil.copyfile(requirements, dest / "requirements.md")
        (dest / "rubric.md").write_text(RUBRIC, encoding="utf-8")
        key[label] = {"name": name, "token": token}
    (out / "key.json").write_text(json.dumps(key, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    return sorted(key)


def valid_answer(answer):
    """Why an answer does not have the agreed form, or None."""
    if not isinstance(answer, dict):
        return "no answer"
    for name in ("architecture", "naming", "cleanliness", "tests"):
        part = answer.get(name)
        if not isinstance(part, dict) or part.get("score") not in (1, 2, 3, 4, 5):
            return f"{name} has no score from 1 to 5"
    if not isinstance(answer.get("takingOver"), dict):
        return "takingOver is missing"
    return None


def run(material_root, model, box, deadline=DEADLINE, stop_after=None):
    """One session per label, one at a time. A failed session is recorded and does not stop the others."""
    import reviewing  # the suite's modules, only needed here
    import series_runner

    root = pathlib.Path(material_root)
    key = json.loads((root / "key.json").read_text(encoding="utf-8"))
    results = {}
    first_reading = 0
    for label in sorted(key):
        reads = root / "reads" / label
        earlier = sorted(reads.glob("a*/read.json"))
        done = [json.loads(p.read_text(encoding="utf-8")) for p in earlier]
        if any(r["valid"] for r in done):
            results[label] = next(r for r in done if r["valid"])
            continue
        # A failed attempt stays where it is, and a later `run` adds another.
        out = reads / f"a{len(earlier) + 1}"
        out.mkdir(parents=True)
        token = key[label]["token"]
        session = reviewing.session("read", PROMPT, token, root / "material" / label, out, model, box, reviewing.REVIEWER, deadline)
        problem = valid_answer(session["answer"]) if session["valid"] else "the session gave no valid answer"
        quota = series_runner.quota(out)
        # In points; a session the provider sent no weekly figure to reads 0 and says nothing.
        reading = round(quota["sevenDay"] * 100, 1)
        read = {"label": label, "valid": problem is None, "problem": problem, "answer": session["answer"], "reported": session["reported"],
                "wallSeconds": session.get("wallSeconds"), "weeklyReading": reading or None}
        (out / "read.json").write_text(json.dumps(read, indent=2) + "\n", encoding="utf-8")
        results[label] = read
        first_reading = first_reading or reading
        if quota["refused"]:
            raise ReadError("a request was refused: the reads stop here")
        if stop_after is not None and reading and reading - first_reading >= stop_after:
            raise ReadError(f"the weekly reading rose from {first_reading} to {reading}: the reads stop here")
    return results


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="cmd", required=True)
    p = sub.add_parser("prepare")
    p.add_argument("--store", required=True)
    p.add_argument("--names", required=True)
    p.add_argument("--requirements", required=True)
    p.add_argument("--out", required=True)
    r = sub.add_parser("run")
    r.add_argument("--material", required=True)
    r.add_argument("--sandbox", required=True)
    r.add_argument("--variant-code", required=True)
    r.add_argument("--login")
    r.add_argument("--series", help="the series' safety limit and egress; by default the suite's")
    r.add_argument("--engine", default=shutil.which("docker") or "docker")
    r.add_argument("--model", default="claude:claude-opus-5-5")
    r.add_argument("--deadline-seconds", type=int, default=DEADLINE)
    r.add_argument("--stop-after-points", type=float)
    r.add_argument("--lock")
    a = parser.parse_args(argv)
    try:
        if a.cmd == "prepare":
            print(json.dumps({"labels": prepare(a.store, a.names.split(","), a.requirements, a.out)}))
            return 0
        import sandbox as sandboxing  # the suite's sandbox, only needed here

        login = a.login or str(pathlib.Path.home() / ".claude" / ".credentials.json")
        provider = a.model.partition(":")[0]
        series = sandboxing.load_series(a.series or sandboxing.DEFAULT_SERIES)
        box = sandboxing.Sandbox([a.engine], a.sandbox, series, a.variant_code, login, provider)
        with sandboxing.engine_lock(a.lock, "evaluation blind read", str(pathlib.Path(a.material).resolve())):
            results = run(a.material, a.model, box, a.deadline_seconds, a.stop_after_points)
        print(json.dumps({label: {"valid": r["valid"], "problem": r["problem"]} for label, r in results.items()}, indent=2))
        return 0 if all(r["valid"] for r in results.values()) else 1
    except Exception as e:
        # Whatever it is, the wrapper's second start must not spend sessions on it.
        print(f"blind_read: {type(e).__name__}: {e}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
