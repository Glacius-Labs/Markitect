"""Assess one trial realization against one expectation profile; see README.md.

    python -B oracle.py --project stockroom|depot|fleet-w<W>-d<D> --profile NAME --realization PATH/TO/realization
                        [--previous PATH/TO/previous/realization] [--python CMD] [--output FILE]

Prints a JSON report with pass, fail or skip per check and exits 0 only when
every check passed. Operator material: never part of an Agent's repository.
"""
import argparse
import importlib.util
import json
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))

from trial_kit import python_command  # noqa: E402

PROJECTS = ("stockroom", "depot")


def load_checks(project):
    if project.startswith("fleet-"):
        import fleet_checks
        return fleet_checks.build(project)
    spec = importlib.util.spec_from_file_location(project + "_checks", HERE / project / "checks.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module.CHECKS


def assess(project, profile, realization, python=None, previous=None):
    checks = load_checks(project)
    if profile not in checks.profiles:
        raise SystemExit("unknown profile %s for %s; known: %s" % (profile, project, sorted(checks.profiles)))
    realization = Path(realization).resolve()
    if not (realization / "app.py").is_file():
        selected = checks.selected(profile)
        return {"project": project, "profile": profile, "realization": str(realization), "passed": False,
                "summary": dict(total=len(selected), passed=0, failed=len(selected), skipped=0),
                "error": "realization/app.py does not exist", "checks": []}
    extra = {"previous": str(Path(previous).resolve())} if previous else {}
    return checks.assess(realization, profile, python or python_command(""), extra)


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--project", required=True, help="stockroom, depot or a generated fleet-w<W>-d<D>")
    parser.add_argument("--profile", required=True)
    parser.add_argument("--realization", required=True)
    parser.add_argument("--previous", help="realization of the previous model version, for upgrade checks")
    parser.add_argument("--python", default="", help="interpreter command (default: this interpreter)")
    parser.add_argument("--output", help="also write the JSON report to this file")
    args = parser.parse_args(argv)
    report = assess(args.project, args.profile, args.realization, python_command(args.python), args.previous)
    text = json.dumps(report, indent=2, ensure_ascii=False)
    print(text)
    if args.output:
        Path(args.output).write_text(text + "\n", encoding="utf-8")
    return 0 if report["passed"] else 1


if __name__ == "__main__":
    sys.exit(main())
