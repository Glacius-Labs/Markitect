"""Validate the readinglog2 evaluation files against the references and the mutants.

    python -I -B validate.py [--jobs N] [--only TEXT ...]

1. ground-truth.json matches the case's STATIONS.json, and every holdout names a known
   item and rule and is released at its item's station.
2. Every reference wave S<n> passes all public checks and all holdouts at station n,
   twice with the same statuses (determinism).
3. The seed baseline passes every baseline and cross-cutting holdout at station 1.
4. Every mutant (a reference wave with exactly one obligation broken, see mutants/)
   fails exactly the holdouts it declares and no other.

Prints one table and exits 0 only when everything is green. Standard library only;
nothing outside a temporary folder is written.
"""
import argparse
from concurrent.futures import ThreadPoolExecutor
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

HERE = Path(__file__).resolve().parent
PLAYGROUND = HERE.parents[1]
CASE = PLAYGROUND / "cases" / "readinglog2"
ACCEPTANCE = PLAYGROUND / "cases" / "common" / "checks" / "acceptance.py"
HOLDOUT = HERE / "holdout.py"
REFERENCE = HERE / "reference"
MUTANTS = HERE / "mutants"
RULES = {"R1", "R2", "R3"}
CHECK_KEYS = {"id", "status", "item", "rule", "detail"}
STATUSES = {"PASS", "FAIL", "ERROR"}


def load_registry():
    spec = importlib.util.spec_from_file_location("readinglog2_holdout", HOLDOUT)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def static_problems(module):
    """Ground truth against STATIONS.json, holdout metadata against the ground truth."""
    problems = []
    truth = json.loads((HERE / "ground-truth.json").read_text(encoding="utf-8"))
    stations = json.loads((CASE / "STATIONS.json").read_text(encoding="utf-8"))["stations"]
    if truth.get("case") != "readinglog2":
        problems.append("ground truth: case is not readinglog2")
    if {rule.get("id") for rule in truth.get("rules", [])} != RULES:
        problems.append("ground truth: rules are not R1-R3")
    waves = [(wave.get("station"), [item.get("id") for item in wave.get("items", [])])
             for wave in truth.get("waves", [])]
    if waves != [(station["id"], station["items"]) for station in stations]:
        problems.append(f"ground truth: waves {waves} do not match STATIONS.json")
    release = {}
    for number, wave in enumerate(truth.get("waves", []), 1):
        for item in wave.get("items", []):
            release[item.get("id")] = number
            for field in ("obligations", "areas", "mustNotChange"):
                values = item.get(field)
                if not isinstance(values, list) or not all(isinstance(v, str) and v for v in values) or (
                        field != "mustNotChange" and not values):
                    problems.append(f"ground truth {item.get('id')}: {field} must be a list of texts")
            for rule in item.get("rules", []):
                if rule.get("id") not in RULES or not isinstance(rule.get("expectation"), str):
                    problems.append(f"ground truth {item.get('id')}: bad rule entry {rule}")
    seen = set()
    for entry in module.HOLDOUTS:
        name = entry["id"]
        if name in seen:
            problems.append(f"holdout {name}: duplicate id")
        seen.add(name)
        if entry["item"] not in release and entry["item"] not in ("baseline", None):
            problems.append(f"holdout {name}: unknown item {entry['item']}")
        if entry["item"] in release and entry["since"] != release[entry["item"]]:
            problems.append(f"holdout {name}: released at S{entry['since']}, its item at S{release[entry['item']]}")
        if entry["rule"] not in RULES and entry["rule"] is not None:
            problems.append(f"holdout {name}: unknown rule {entry['rule']}")
        if not entry["source"]:
            problems.append(f"holdout {name}: no public source sentence")
        if entry["until"] is not None and entry["until"] < entry["since"]:
            problems.append(f"holdout {name}: retired before it is released")
    return problems, len(release)


# --- running the checks ------------------------------------------------------------------

def python(script, *args, timeout):
    proc = subprocess.run([sys.executable, "-I", "-B", str(script), *map(str, args)], capture_output=True,
                          timeout=timeout)
    return proc.returncode, proc.stdout.decode("utf-8", "replace"), proc.stderr.decode("utf-8", "replace")


def public_checks(repo, station):
    code, out, err = python(ACCEPTANCE, "--repo", repo, "--case", "readinglog2", "--station", station, timeout=600)
    try:
        findings = json.loads(out)["findings"]
    except (ValueError, KeyError):
        return None, f"acceptance.py exit {code}: {err[-300:]}"
    return [(f["check"], f["status"]) for f in findings], None


def holdouts(repo, station, module):
    """Run holdout.py and check its output protocol; returns ({id: status}, details, problem)."""
    code, out, err = python(HOLDOUT, "--repo", repo, "--station", station, timeout=900)
    try:
        data = json.loads(out)
    except ValueError:
        return None, None, f"holdout.py exit {code}, unparsable output: {err[-300:]}"
    if code != 0 or data.get("station") != station:
        return None, None, f"holdout.py exit {code} or wrong station {data.get('station')}"
    checks = data.get("checks")
    expected = [entry["id"] for entry in module.released(station)]
    if not isinstance(checks, list) or [c.get("id") for c in checks] != expected:
        return None, None, "holdout ids differ from the released registry"
    for check in checks:
        if not CHECK_KEYS <= set(check) or check["status"] not in STATUSES:
            return None, None, f"holdout {check.get('id')} breaks the output protocol"
    return ({c["id"]: c["status"] for c in checks}, {c["id"]: c["detail"] for c in checks}, None)


def staged(source, folder):
    target = Path(folder) / "repo"
    shutil.copytree(source, target, ignore=shutil.ignore_patterns("__pycache__", "*.pyc"))
    return target


def joined(text):
    return "\n".join(text) if isinstance(text, list) else text


def apply_edits(repo, edits):
    for edit in edits:
        path = repo / edit["file"]
        text = path.read_text(encoding="utf-8")
        old, new = joined(edit["old"]), joined(edit["new"])
        if text.count(old) != 1:
            raise ValueError(f"{edit['file']}: the text to replace occurs {text.count(old)} times: {old[:80]!r}")
        path.write_text(text.replace(old, new), encoding="utf-8", newline="\n")


def say(text):
    """Print ASCII only, so any console encoding works."""
    print(text.encode("ascii", "backslashreplace").decode("ascii"))


def tally(pairs):
    return f"{sum(status == 'PASS' for _, status in pairs)}/{len(pairs)}"


def reference_job(number, module):
    with tempfile.TemporaryDirectory(prefix="rl2-validate-") as folder:
        repo = staged(REFERENCE / f"S{number}", folder)
        public, problem = public_checks(repo, number)
        first, details, problem2 = holdouts(repo, number, module)
        second, _, problem3 = holdouts(repo, number, module)
    problem = problem or problem2 or problem3
    row = {"kind": "reference", "name": f"S{number}", "station": number}
    if problem:
        return dict(row, ok=False, note=problem)
    row.update(public=tally(public), holdouts=tally(first.items()))
    failing = [name for name, status in public if status != "PASS"] + [n for n, s in first.items() if s != "PASS"]
    if first != second:
        return dict(row, ok=False, note="holdout statuses differ between two runs")
    if failing:
        return dict(row, ok=False, note="failing: " + ", ".join(
            f"{name} ({(details or {}).get(name, '')[:160]})" for name in failing))
    return dict(row, ok=True, note="")


def seed_job(module):
    with tempfile.TemporaryDirectory(prefix="rl2-validate-") as folder:
        repo = staged(CASE, folder)
        public, problem = public_checks(repo, 1)
        statuses, details, problem2 = holdouts(repo, 1, module)
    row = {"kind": "seed", "name": "baseline", "station": 1}
    if problem or problem2:
        return dict(row, ok=False, note=problem or problem2)
    items = {entry["id"]: entry["item"] for entry in module.HOLDOUTS}
    broken = [name for name, status in statuses.items() if items[name] in ("baseline", None) and status != "PASS"]
    row.update(public=tally(public), holdouts=tally(statuses.items()))
    if broken:
        return dict(row, ok=False, note="baseline holdouts fail on the seed: " + ", ".join(
            f"{name} ({details[name][:160]})" for name in broken))
    return dict(row, ok=True, note="baseline holdouts pass; B01 holdouts fail as expected")


def mutant_job(path, module):
    spec = json.loads(path.read_text(encoding="utf-8"))
    number = int(spec["base"].lstrip("S"))
    row = {"kind": "mutant", "name": spec["id"], "station": number}
    if spec["id"] != path.stem:
        return dict(row, ok=False, note="id differs from the file name")
    released = {entry["id"]: entry for entry in module.released(number)}
    declared = set(spec["fails"])
    if not declared or not declared <= set(released):
        unknown = sorted(declared - set(released))
        return dict(row, ok=False, note=f"declared holdouts not released at S{number}: {unknown}")
    if not any(released[name]["item"] == spec["item"] for name in declared):
        return dict(row, ok=False, note=f"no declared holdout targets item {spec['item']}")
    with tempfile.TemporaryDirectory(prefix="rl2-validate-") as folder:
        repo = staged(REFERENCE / spec["base"], folder)
        try:
            apply_edits(repo, spec["edits"])
        except ValueError as error:
            return dict(row, ok=False, note=str(error))
        public, problem = public_checks(repo, number)
        statuses, details, problem2 = holdouts(repo, number, module)
    if problem or problem2:
        return dict(row, ok=False, note=problem or problem2)
    failing = {name for name, status in statuses.items() if status != "PASS"}
    errors = sorted(name for name, status in statuses.items() if status == "ERROR")
    row.update(public=tally(public), holdouts=tally(statuses.items()))
    if failing != declared or errors:
        unexpected, missed = sorted(failing - declared), sorted(declared - failing)
        note = []
        if unexpected:
            note.append("also fails " + ", ".join(f"{n} ({details[n][:160]})" for n in unexpected))
        if missed:
            note.append("does not fail " + ", ".join(missed))
        if errors:
            note.append("errors " + ", ".join(errors))
        return dict(row, ok=False, note="; ".join(note))
    return dict(row, ok=True, note="fails " + ", ".join(sorted(declared)))


def main(argv=None):
    parser = argparse.ArgumentParser(description="Validate the readinglog2 holdouts against references and mutants.")
    parser.add_argument("--jobs", type=int, default=min(8, os.cpu_count() or 2))
    parser.add_argument("--only", nargs="*", default=None, help="run only rows whose name contains one of these")
    args = parser.parse_args(argv)
    module = load_registry()
    problems, items = static_problems(module)
    say(f"readinglog2 evaluation: {items} items, {len(module.HOLDOUTS)} holdouts, "
          f"{len(list(MUTANTS.glob('*.json')))} mutants; python {sys.version.split()[0]}")
    for problem in problems:
        say(f"  STATIC: {problem}")

    def wanted(name):
        return args.only is None or any(text in name for text in args.only)
    with ThreadPoolExecutor(max_workers=max(1, args.jobs)) as pool:
        futures = [pool.submit(reference_job, n, module) for n in range(1, 7) if wanted(f"S{n}")]
        if wanted("baseline"):
            futures.append(pool.submit(seed_job, module))
        futures += [pool.submit(mutant_job, path, module) for path in sorted(MUTANTS.glob("*.json"))
                    if wanted(path.stem)]
        rows = [future.result() for future in futures]
    say(f"{'kind':<9} {'name':<38} {'st':>2}  {'public':>7}  {'holdouts':>8}  result")
    for row in rows:
        say(f"{row['kind']:<9} {row['name']:<38} {row['station']:>2}  {row.get('public', '-'):>7}  "
              f"{row.get('holdouts', '-'):>8}  {'ok' if row['ok'] else 'NOT OK'}  {row['note']}"[:400])
    green = not problems and all(row["ok"] for row in rows)
    say("GREEN" if green else "NOT GREEN")
    return 0 if green else 1


if __name__ == "__main__":
    sys.exit(main())
