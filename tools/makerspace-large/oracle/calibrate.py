"""How strong the hidden cases are: plausible bugs seeded in the reference, one at a time, each of which should fail a case.

  python -I -B calibrate.py MUTANTS.json [--only NAME] [--out REPORT.json]

MUTANTS.json is a list of {"name", "module", "file", "old", "new", "why"}: `file` relative to the reference tree, `old` found
exactly once. Each mutant is copied to a scratch tree, built and run against every hidden case. It starts no container and
no agent. Prints the mutants no case caught, which are gaps in the cases or changes without effect.
"""
import argparse
import json
import os
import pathlib
import shutil
import subprocess
import sys
import tempfile

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import catalog  # noqa: E402

REFERENCE = HERE.parent / "reference" / "w0"


def run_mutant(mutant, env):
    with tempfile.TemporaryDirectory(prefix="ms-mutant-", ignore_cleanup_errors=True) as tmp:
        tree = pathlib.Path(tmp) / "tree"
        shutil.copytree(REFERENCE, tree)
        path = tree / mutant["file"]
        text = path.read_bytes().decode("utf-8")
        if text.count(mutant["old"]) != 1:
            return {"name": mutant["name"], "state": "bad-mutant", "detail": f"`old` occurs {text.count(mutant['old'])} times"}
        path.write_bytes(text.replace(mutant["old"], mutant["new"]).encode("utf-8"))
        program = pathlib.Path(tmp) / ("makerspace.exe" if os.name == "nt" else "makerspace")
        built = subprocess.run(["go", "build", "-o", str(program), "./cmd/makerspace"], cwd=tree, env=env, capture_output=True, text=True, errors="replace")
        if built.returncode != 0:
            return {"name": mutant["name"], "state": "bad-mutant", "detail": built.stderr[-300:]}
        report = catalog.assess([str(program)], tmp, workers=8)
        failed = [c["id"] for c in report["cases"] if not c["passed"]]
        return {"name": mutant["name"], "module": mutant.get("module"), "state": "caught" if failed else "survived", "failed": failed[:12], "failedCount": len(failed)}


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    p.add_argument("mutants")
    p.add_argument("--only")
    p.add_argument("--out")
    a = p.parse_args(argv)
    env = dict(os.environ)
    mutants = json.loads(pathlib.Path(a.mutants).read_text(encoding="utf-8"))
    rows = [run_mutant(m, env) for m in mutants if not a.only or m["name"] == a.only]
    for row in rows:
        if row["state"] != "caught":
            print(row["state"], row["name"], row.get("detail", ""))
    caught = sum(r["state"] == "caught" for r in rows)
    print(f"caught {caught} of {len(rows)}")
    if a.out:
        pathlib.Path(a.out).write_text(json.dumps(rows, indent=1) + "\n", encoding="utf-8")
    return 0 if caught == len(rows) else 1


if __name__ == "__main__":
    sys.exit(main())
