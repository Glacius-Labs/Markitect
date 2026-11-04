"""Does each step's model compile with Markitect's own compiler, as a project that holds no code yet?

  python -I -B compile_steps.py --markitect BIN [--step STEP ...]

`pages.py check` knows this project's page form; `markitect check` is what a Command runs on the spec, and it also finds what
the page form doesn't, such as Packages that depend on each other in a cycle. Every step must pass both. Each step is checked
in a fresh repository: `.markitect/` with the step's model and registry pages, and no code, so nothing is unregistered.
"""
import argparse
import json
import pathlib
import shutil
import subprocess
import sys
import tempfile

HERE = pathlib.Path(__file__).resolve().parent
TASKS = HERE.parent / "evaluation-tasks" / "makerspace-large"
STEPS = ("w0", "c1", "c2", "c3", "c4", "c5", "c6", "c7", "c8")
ENTRIES = ("manifest.yaml", "lock.yaml", "execution.yaml", "model")


def compile_step(markitect, step):
    source = TASKS / step
    registry = source / "registry" if (source / "registry").is_dir() else TASKS / "registry"
    with tempfile.TemporaryDirectory(prefix="ms-compile-", ignore_cleanup_errors=True) as tmp:
        repo = pathlib.Path(tmp)
        dot = repo / ".markitect"
        dot.mkdir()
        for name in ENTRIES:
            if (source / name).is_dir():
                shutil.copytree(source / name, dot / name)
            else:
                shutil.copyfile(source / name, dot / name)
        shutil.copytree(registry, dot / "registry")
        subprocess.run(["git", "init", "-q", str(repo)], check=True)
        p = subprocess.run([markitect, "check", "--repo", str(repo), "--json"], capture_output=True, text=True, encoding="utf-8", errors="replace")
    try:
        report = json.loads(p.stdout)
    except ValueError:
        return [{"code": "no-report", "message": f"exit {p.returncode}: {p.stderr[-400:]}"}]
    return report.get("errors") or []


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--markitect", required=True)
    parser.add_argument("--step", action="append", choices=STEPS)
    a = parser.parse_args(argv)
    bad = 0
    for step in a.step or STEPS:
        errors = compile_step(a.markitect, step)
        print(f"{step}: {len(errors)} errors")
        for e in errors:
            bad += 1
            print("  ", e.get("code"), e.get("file"), e.get("line"), str(e.get("message"))[:400])
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
