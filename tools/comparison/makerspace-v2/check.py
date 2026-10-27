"""Compile the Makerspace v2 model with the real `markitect check`.

The model folder is copied into a fresh Git repository as `.markitect/` and
checked with every vendored Library passed by `--dependency`. `check` exits
nonzero unless the model compiles and `model-lock.yaml` pins the current
Library digests; `lock` rewrites `model-lock.yaml` from the digests `check`
reports. Both also require each vendored Library to equal its source.

Usage, from the repository root:
  python -I -B tools/comparison/makerspace-v2/check.py check
  python -I -B tools/comparison/makerspace-v2/check.py lock
"""
import filecmp
import json
import pathlib
import shutil
import subprocess
import sys
import tempfile

HERE = pathlib.Path(__file__).resolve().parent
REPO = HERE.parents[2]
LIBRARIES = {  # vendored folder -> source folder in this repository
    "markitect-core": REPO / "testdata" / "core",
    "markitect-values": REPO / "libraries" / "values",
    "markitect-data": REPO / "libraries" / "data",
    "markitect-api": REPO / "libraries" / "api",
    "markitect-practice": REPO / "libraries" / "practice",
    "markitect-architecture": REPO / "libraries" / "architecture",
}
VERSIONS = {"markitect-core": "1.2.0"}


def same_tree(a, b):
    compared = filecmp.dircmp(a, b)
    if compared.left_only or compared.right_only or compared.diff_files or compared.funny_files:
        return False
    return all(same_tree(pathlib.Path(a) / d, pathlib.Path(b) / d) for d in compared.common_dirs)


def git(repo, *args):
    subprocess.run(["git", *args], cwd=repo, check=True, capture_output=True)


def run_check(lock_text):
    with tempfile.TemporaryDirectory() as tmp:
        repo = pathlib.Path(tmp)
        control = repo / ".markitect"
        shutil.copytree(HERE / "model", control / "model")
        shutil.copy(HERE / "model.yaml", control / "model.yaml")
        shutil.copy(HERE / "execution.yaml", control / "execution.yaml")
        (control / "model-lock.yaml").write_text(lock_text, encoding="utf-8")
        (control / "ignore.yaml").write_text("apiVersion: markitect.ignore/v1\nfiles: []\n", encoding="utf-8")
        shard = control / "registry" / "makerspace"
        shard.mkdir(parents=True)
        (shard / "registry.yaml").write_text("apiVersion: markitect.registry/v1\nartifacts: []\n", encoding="utf-8")
        git(repo, "init", "-q")
        git(repo, "add", "-A")
        git(repo, "-c", "user.name=check", "-c", "user.email=check@example.invalid", "commit", "-q", "-m", "model")
        command = ["go", "run", "./cmd/markitect", "check", "--repo", str(repo)]
        for name in LIBRARIES:
            command += ["--dependency", str(HERE / "libraries" / name / "model.yaml")]
        done = subprocess.run(command, cwd=REPO, capture_output=True, text=True, encoding="utf-8")
        return done.returncode, done.stdout, done.stderr


def digests(report):
    found = {}

    def walk(node):
        if isinstance(node, dict):
            digest = node.get("canonicalModelDigest")
            if isinstance(node.get("id"), str) and isinstance(digest, str) and digest.startswith("sha256:"):
                found[node["id"]] = digest
            for value in node.values():
                walk(value)
        elif isinstance(node, list):
            for value in node:
                walk(value)

    walk(json.loads(report))
    return found


def findings(report):
    codes = []

    def walk(node):
        if isinstance(node, dict):
            if isinstance(node.get("Code"), str) and node.get("Message"):
                codes.append(f"{node['Code']}: {node['Message']}")
            for value in node.values():
                walk(value)
        elif isinstance(node, list):
            for value in node:
                walk(value)

    walk(json.loads(report))
    return codes


def lock_text(found):
    lines = ["formatVersion: 1", "models:"]
    manifest = (HERE / "model.yaml").read_text(encoding="utf-8")
    for name in LIBRARIES:
        version = manifest.split(name + ":", 1)[1].split("\n", 1)[0].strip().strip('"')
        lines += [f"  {name}:", f"    version: {version}", f"    digest: {found[name]}"]
    return "\n".join(lines) + "\n"


def main(argv):
    if len(argv) != 2 or argv[1] not in ("check", "lock"):
        print(__doc__)
        return 2
    stale = [name for name, source in LIBRARIES.items() if not same_tree(HERE / "libraries" / name, source)]
    if stale:
        print("vendored Libraries differ from their sources:", ", ".join(stale))
        return 1
    current = (HERE / "model-lock.yaml").read_text(encoding="utf-8") if (HERE / "model-lock.yaml").exists() else "formatVersion: 1\nmodels: {}\n"
    code, out, err = run_check(current if argv[1] == "check" else "formatVersion: 1\nmodels: {}\n")
    try:
        found = digests(out)
    except json.JSONDecodeError:
        print(err or out)
        return 1
    if argv[1] == "lock":
        missing = [name for name in LIBRARIES if name not in found]
        if missing:
            print("Libraries without a digest:", ", ".join(missing))
            print("\n".join(findings(out)))
            return 1
        (HERE / "model-lock.yaml").write_text(lock_text(found), encoding="utf-8", newline="\n")
        code, out, err = run_check(lock_text(found))
    if code != 0:
        print("\n".join(findings(out)) or err)
        return 1
    print(f"makerspace-v2 compiles; model digest {found.get('makerspace', '?')}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
