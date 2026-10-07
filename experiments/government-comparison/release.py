"""Materialize only released public inputs outside the actor Git repository."""
import argparse
import ast
import json
from pathlib import Path
import shutil

from prepare import ROOT, digest


def staged_checks(cutoff):
    tree = ast.parse((ROOT / "public" / "checks.py").read_text(encoding="utf-8"))
    # Truncate at explicit stage boundaries; later implementation/rule text is absent.
    for node in ast.walk(tree):
        if isinstance(node, ast.FunctionDef) and node.name == "run":
            output = []
            for statement in node.body:
                text = ast.unparse(statement.test) if isinstance(statement, ast.If) else ""
                boundary = ((cutoff == 1 and text == "through < 2") or
                            (cutoff in (2, 3) and text == "through < 4") or
                            (cutoff == 4 and text == "through >= 5"))
                if boundary:
                    output.append(ast.Return(value=ast.Name(id="checkpoint", ctx=ast.Load())))
                    break
                output.append(statement)
            node.body = output
    source = ast.unparse(ast.fix_missing_locations(tree)) + "\n"
    source = source.replace("choices=range(1, 7)", f"choices=range(1, {cutoff + 1})")
    return source


def release(cutoff, destination, condition="greenfield"):
    if cutoff not in range(1, 7):
        raise ValueError("task cutoff must be 1..6")
    destination = Path(destination).resolve()
    destination.mkdir(parents=True, exist_ok=False)
    for name in ("brief.md", "architecture.md"):
        shutil.copyfile(ROOT / "public" / name, destination / name)
    if condition == "brownfield":
        shutil.copyfile(ROOT / "public" / "brownfield-provenance.md", destination / "brownfield-provenance.md")
    elif condition != "greenfield":
        raise ValueError("unknown condition")
    for card in sorted((ROOT / "public" / "tasks").glob("*.md"))[:cutoff]:
        shutil.copyfile(card, destination / card.name)
    (destination / "checks.py").write_text(staged_checks(cutoff), encoding="utf-8")
    manifest = {"schemaVersion": 1, "throughTask": cutoff, "condition": condition,
                "inputs": [{"path": str(p), "sha256": digest(p)} for p in sorted(destination.iterdir()) if p.is_file()]}
    (destination / "release.json").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
    return manifest


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--through-task", type=int, required=True)
    parser.add_argument("--destination", required=True)
    parser.add_argument("--condition", choices=("greenfield", "brownfield"), required=True)
    args = parser.parse_args()
    if not Path(args.destination).is_absolute():
        parser.error("destination must be absolute")
    print(json.dumps(release(args.through_task, args.destination, args.condition), indent=2))
