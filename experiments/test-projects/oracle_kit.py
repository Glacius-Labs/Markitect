"""Shared black-box harness of the Roombook and Readinglog oracles.

An oracle drives a realized command line exactly through its public contract,
`python -B app.py --db PATH COMMAND ...` from the realization directory, and
reports a pass or fail per check as JSON. It reads no source of the
realization, never feeds back into Agents and runs only after an Apply.

Each project's `oracle/checks.py` registers checks with the stage that
introduces them and the Definitions they cite; `oracle/oracle.py` assesses one
realization at one stage; `oracle/selfcheck.py` shows that the oracle
discriminates. Checks run in a fresh temporary directory each, are cumulative
(stage N includes every earlier stage) and see the child process only through
stdout, the exit code and the database file.
"""
import argparse
import json
import os
import re
import subprocess
import sys
import tempfile
import traceback
from pathlib import Path

TIMEOUT_SECONDS = 30


class CheckFailure(Exception):
    """An expectation of a check did not hold."""


def fail(message):
    raise CheckFailure(message)


def same(left, right):
    """Strict JSON equality: booleans, integers and floats never compare equal."""
    if type(left) is not type(right):
        return False
    if isinstance(left, dict):
        return left.keys() == right.keys() and all(same(left[key], right[key]) for key in left)
    if isinstance(left, list):
        return len(left) == len(right) and all(same(a, b) for a, b in zip(left, right))
    return left == right


def short(value, limit=300):
    text = value if isinstance(value, str) else json.dumps(value, ensure_ascii=False)
    return text if len(text) <= limit else text[:limit] + "..."


class Result:
    """One process invocation: exit code, raw streams and the parsed stdout object."""

    def __init__(self, args, code, out, err, timed_out=False):
        self.args = list(args)
        self.code = code
        self.out = out
        self.err = err
        self.timed_out = timed_out
        self.obj = None
        self.problem = None
        if timed_out:
            self.problem = "timed out after %d seconds" % TIMEOUT_SECONDS
            return
        try:
            text = out.decode("utf-8")
        except UnicodeDecodeError:
            self.problem = "stdout is not UTF-8"
            return
        if not text.strip():
            self.problem = "stdout is empty"
            return
        try:
            parsed = json.loads(text)
        except ValueError as error:
            self.problem = "stdout is not exactly one JSON value (%s): %s" % (error, short(text, 120))
            return
        if not isinstance(parsed, dict):
            self.problem = "stdout is not a JSON object: %s" % short(text, 120)
            return
        self.obj = parsed

    def describe(self):
        return "exit %s, %s" % (self.code, short(self.obj if self.obj is not None else (self.problem or "")))


class Env:
    """Fresh workspace of one check: database directory, input files, process runner."""

    def __init__(self, realization, python, root):
        self.realization = Path(realization)
        self.python = python
        self.root = Path(root)
        self.dbdir = self.root / "data"
        self.dbdir.mkdir()
        self.db = self.dbdir / "db.json"
        self.inputs = self.root / "inputs"
        self.inputs.mkdir()

    # -- process -----------------------------------------------------------------

    def run(self, *args):
        command = [*self.python, "-B", "app.py", "--db", str(self.db), *map(str, args)]
        environment = dict(os.environ, PYTHONUTF8="1", PYTHONIOENCODING="utf-8", PYTHONDONTWRITEBYTECODE="1")
        try:
            done = subprocess.run(command, cwd=self.realization, capture_output=True, timeout=TIMEOUT_SECONDS,
                                  env=environment, stdin=subprocess.DEVNULL)
            return Result(args, done.returncode, done.stdout, done.stderr)
        except subprocess.TimeoutExpired:
            return Result(args, None, b"", b"", timed_out=True)
        except OSError as error:
            fail("cannot start the command: %s" % error)

    def ok(self, *args):
        """Run a command that must succeed with exactly one JSON object and exit 0."""
        result = self.run(*args)
        crash_guard(result)
        if result.code != 0:
            fail("`%s` must exit 0, got %s" % (" ".join(map(str, args)), result.describe()))
        if result.obj is None:
            fail("`%s`: %s" % (" ".join(map(str, args)), result.problem))
        if set(result.obj) == {"error"}:
            fail("`%s` succeeded with an error object" % " ".join(map(str, args)))
        return result.obj

    def refuse(self, *args, unchanged=True):
        """Run a command that must fail with the error envelope, exit 2 and unchanged data."""
        before = self.raw() if unchanged else None
        result = self.run(*args)
        crash_guard(result)
        label = "`%s`" % " ".join(map(str, args))
        if result.code != 2:
            fail("%s must exit 2, got %s" % (label, result.describe()))
        if result.obj is None:
            fail("%s: %s" % (label, result.problem))
        error = result.obj.get("error")
        if set(result.obj) != {"error"} or type(error) is not str or not error.strip():
            fail("%s must write only a nonempty string `error`, got %s" % (label, short(result.obj)))
        if unchanged and self.raw() != before and not self.still_empty(before):
            fail("%s changed the stored data although it failed" % label)
        return result

    def still_empty(self, before):
        """A missing file is empty state, so a failed command may leave it missing or write an empty-state file."""
        if before is not None or self.raw() is None:
            return False
        result = self.run("list")
        if result.code != 0 or result.obj is None:
            return False
        lists = [value for value in result.obj.values() if isinstance(value, list)]
        return len(lists) == 1 and not lists[0]

    def leftover_temp_files(self):
        """Files beside the database that look like abandoned temporary files of a replacement."""
        found = []
        for path in self.dbdir.iterdir():
            name = path.name.lower()
            if path.name == self.db.name:
                continue
            if (name.startswith(self.db.stem.lower()) or "tmp" in name or "temp" in name or name.startswith(".")
                    or name.endswith(("~", ".new", ".bak", ".swp"))):
                found.append(path.name)
        return sorted(found)

    # -- stored data --------------------------------------------------------------

    def raw(self):
        return self.db.read_bytes() if self.db.exists() else None

    def stat(self):
        status = self.db.stat()
        return (status.st_mtime_ns, status.st_size, status.st_ino)

    def write_db(self, content):
        self.db.write_bytes(content if isinstance(content, bytes) else content.encode("utf-8"))

    def write_input(self, name, content):
        path = self.inputs / name
        path.write_bytes(content if isinstance(content, bytes) else content.encode("utf-8"))
        return str(path)

    def stored_json(self):
        """The database file as JSON; it must exist, be UTF-8 and parse."""
        data = self.raw()
        if data is None:
            fail("the database file does not exist")
        try:
            return json.loads(data.decode("utf-8"))
        except ValueError as error:
            fail("the database file is not UTF-8 JSON: %s" % error)

    def reads_leave_data_untouched(self, commands):
        """Run read-only commands against existing data; bytes and file identity must not change."""
        before_bytes, before_stat = self.raw(), self.stat()
        for command in commands:
            self.ok(*command)
        if self.raw() != before_bytes or self.stat()[:2] != before_stat[:2]:
            fail("a read-only command rewrote the database file")


def crash_guard(result):
    if result.timed_out:
        fail("`%s` %s" % (" ".join(result.args), result.problem))
    if b"Traceback (most recent call last)" in result.err:
        fail("`%s` ended with an unhandled exception" % " ".join(result.args))


def expect_equal(actual, expected, label):
    if not same(actual, expected):
        fail("%s: expected %s, got %s" % (label, short(expected), short(actual)))


def expect_keys(obj, keys, label):
    if not isinstance(obj, dict) or set(obj) != set(keys):
        fail("%s must be an object with exactly the keys %s, got %s" % (label, sorted(keys), short(obj)))


class Registry:
    """The checks of one project, ordered, with the stage that introduces each."""

    def __init__(self, project, package, stages):
        self.project = project
        self.package = package
        self.stages = stages  # {stage number: snapshot directory name}
        self.checks = []

    def check(self, check_id, stage, definitions, title):
        def register(function):
            if any(known["id"] == check_id for known in self.checks):
                raise ValueError("duplicate check " + check_id)
            if stage not in self.stages:
                raise ValueError("unknown stage for " + check_id)
            if not definitions:
                raise ValueError("a check cites at least one Definition: " + check_id)
            self.checks.append({"id": check_id, "stage": stage, "definitions": list(definitions),
                                "title": title, "run": function})
            return function
        return register

    def selected(self, stage):
        return [known for known in self.checks if known["stage"] <= stage]

    def assess(self, realization, stage, python):
        results = []
        for known in self.selected(stage):
            with tempfile.TemporaryDirectory(prefix="oracle-") as root:
                env = Env(realization, python, root)
                try:
                    known["run"](env)
                    status, detail = "pass", ""
                except CheckFailure as error:
                    status, detail = "fail", str(error)
                except Exception:  # an oracle defect must never read as a pass
                    status, detail = "fail", "oracle error: " + traceback.format_exc(limit=3)
            results.append({"id": known["id"], "title": known["title"], "introducedIn": known["stage"],
                            "definitions": ["%s/%s" % (self.package, name) for name in known["definitions"]],
                            "status": status, "detail": detail})
        passed = sum(1 for item in results if item["status"] == "pass")
        return {"project": self.project, "stage": stage, "snapshot": self.stages[stage],
                "realization": str(realization), "passed": passed == len(results),
                "summary": {"total": len(results), "passed": passed, "failed": len(results) - passed},
                "checks": results}


def python_command(text):
    return text.split() if text else [sys.executable]


def oracle_main(registry, argv=None):
    parser = argparse.ArgumentParser(description="Assess a %s realization at one stage." % registry.project)
    parser.add_argument("--realization", help="directory holding app.py (the realization/ directory)")
    parser.add_argument("--stage", type=int, choices=sorted(registry.stages), help="snapshot stage to assess")
    parser.add_argument("--python", default="", help="interpreter command (default: this interpreter)")
    parser.add_argument("--output", help="also write the JSON report to this file")
    parser.add_argument("--list", action="store_true", help="list the checks of the stage and exit")
    args = parser.parse_args(argv)
    if args.stage is None:
        parser.error("--stage is required")
    if args.list:
        for known in registry.selected(args.stage):
            print(known["id"], known["stage"], ",".join(known["definitions"]), sep="\t")
        return 0
    if not args.realization or not (Path(args.realization) / "app.py").is_file():
        report = {"project": registry.project, "stage": args.stage, "snapshot": registry.stages[args.stage],
                  "realization": args.realization, "passed": False,
                  "summary": {"total": len(registry.selected(args.stage)), "passed": 0,
                              "failed": len(registry.selected(args.stage))},
                  "error": "realization/app.py does not exist", "checks": []}
    else:
        report = registry.assess(Path(args.realization).resolve(), args.stage, python_command(args.python))
    text = json.dumps(report, indent=2, ensure_ascii=False)
    print(text)
    if args.output:
        Path(args.output).write_text(text + "\n", encoding="utf-8")
    return 0 if report["passed"] else 1


# -- discrimination self-check ----------------------------------------------------

DEGENERATE = {
    "silent": "import sys\nsys.exit(0)\n",
    "always-error": "import json, sys\nprint(json.dumps({'error': 'no'}))\nsys.exit(2)\n",
    "empty-object": "import sys\nprint('{}')\nsys.exit(0)\n",
    "crash": "raise RuntimeError('boom')\n",
}


def definition_ids(project_dir, snapshot, package):
    folder = Path(project_dir) / "snapshots" / snapshot / "model" / package
    return {path.stem for path in folder.glob("*.yaml")}


def selfcheck_main(registry, project_dir, argv=None):
    parser = argparse.ArgumentParser(description="Show that the %s oracle discriminates." % registry.project)
    parser.add_argument("--python", default="", help="interpreter command (default: this interpreter)")
    parser.add_argument("--reference", help="realization directory that must pass every check of every stage "
                                            "(keep it outside the repository: it would be a gold result)")
    parser.add_argument("--mutants", help="directory with <stage>/<name>/app.py mutants; each must fail a check")
    args = parser.parse_args(argv)
    python = python_command(args.python)
    problems, report = [], {"project": registry.project}

    # 1. Every cited Definition exists in the snapshot that introduces the check and in every later one.
    for known in registry.checks:
        for stage in sorted(registry.stages):
            if stage < known["stage"]:
                continue
            present = definition_ids(project_dir, registry.stages[stage], registry.package)
            for name in known["definitions"]:
                if name not in present:
                    problems.append("%s cites %s/%s, absent from %s" % (known["id"], registry.package, name,
                                                                         registry.stages[stage]))
    report["citations"] = "checked %d checks" % len(registry.checks)

    # 2. Degenerate implementations must not pass, and none may pass more than half of the checks.
    report["degenerate"] = {}
    for name, source in DEGENERATE.items():
        with tempfile.TemporaryDirectory(prefix="oracle-degenerate-") as folder:
            (Path(folder) / "app.py").write_text(source, encoding="utf-8")
            for stage in sorted(registry.stages):
                outcome = registry.assess(folder, stage, python)
                total, passed = outcome["summary"]["total"], outcome["summary"]["passed"]
                report["degenerate"]["%s@%d" % (name, stage)] = "%d/%d pass" % (passed, total)
                if outcome["passed"] or passed * 2 > total:
                    problems.append("degenerate %s passes %d of %d checks at stage %d" % (name, passed, total, stage))

    # 3. A supplied reference passes every stage.
    if args.reference:
        report["reference"] = {}
        for stage in sorted(registry.stages):
            outcome = registry.assess(Path(args.reference).resolve(), stage, python)
            report["reference"][str(stage)] = "%d/%d pass" % (outcome["summary"]["passed"], outcome["summary"]["total"])
            for item in outcome["checks"]:
                if item["status"] != "pass":
                    problems.append("reference fails %s at stage %d: %s" % (item["id"], stage, item["detail"]))

    # 4. Every supplied mutant fails at least one check of its stage.
    if args.mutants:
        report["mutants"] = {}
        for stage_folder in sorted(Path(args.mutants).iterdir()):
            if not stage_folder.is_dir() or not re.fullmatch(r"\d+", stage_folder.name):
                continue
            stage = int(stage_folder.name)
            for mutant in sorted(stage_folder.iterdir()):
                if not (mutant / "app.py").is_file():
                    continue
                outcome = registry.assess(mutant.resolve(), stage, python)
                killed = [item["id"] for item in outcome["checks"] if item["status"] != "pass"]
                report["mutants"]["%d/%s" % (stage, mutant.name)] = {"killedBy": killed}
                if not killed:
                    problems.append("mutant %d/%s survives every check" % (stage, mutant.name))

    report["problems"] = problems
    report["passed"] = not problems
    print(json.dumps(report, indent=2, ensure_ascii=False))
    return 0 if not problems else 1
