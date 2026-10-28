"""Compile the Makerspace v2 model with the real `markitect check`.

The model folder is copied into a fresh Git repository as `.markitect/` and
checked with every vendored Library passed by `--dependency`. `check` exits
nonzero unless the model compiles and `model-lock.yaml` pins the current
Library digests; `lock` rewrites `model-lock.yaml` from the digests `check`
reports. Both also require each vendored Library to equal its source when
both have the same version. A vendored Library older than its source is frozen
gate material: it is named in a note, and the lock still pins its digest.

On Windows, set TMP and TEMP to %LOCALAPPDATA%\\Temp first if Git refuses the
system temporary folder.

Usage, from the repository root:
  python -I -B tools/comparison/makerspace-v2/check.py check [STEP]
  python -I -B tools/comparison/makerspace-v2/check.py lock

STEP is w0 (default, `model/`) or a change step such as c1 (`c1/model/`); every
step shares the manifest, lock and execution configuration.
"""
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
CASCADE = ("_not_passed",)  # follow-on findings that only repeat an earlier failure


def files(root):
    return {p.relative_to(root).as_posix(): p.read_bytes() for p in sorted(pathlib.Path(root).rglob("*")) if p.is_file()}


def same_tree(a, b):
    return files(a) == files(b)


def version(root):
    for line in (pathlib.Path(root) / "model.yaml").read_text(encoding="utf-8").splitlines():
        if line.startswith("version:"):
            return line.split(":", 1)[1].strip().strip('"')
    return None


def git(repo, *args):
    done = subprocess.run(["git", "-c", "commit.gpgsign=false", *args], cwd=repo, capture_output=True, text=True)
    if done.returncode != 0:
        raise SystemExit(f"git {' '.join(args)} failed: {done.stderr.strip()}")


def step_model(step):
    return HERE / "model" if step == "w0" else HERE / step / "model"


def run_check(lock_text, step="w0"):
    with tempfile.TemporaryDirectory() as tmp:
        repo = pathlib.Path(tmp)
        control = repo / ".markitect"
        shutil.copytree(step_model(step), control / "model")
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
    return [c for c in codes if not c.split(":", 1)[0].endswith(CASCADE)] or codes


def lock_text(found):
    lines = ["formatVersion: 1", "models:"]
    manifest = (HERE / "model.yaml").read_text(encoding="utf-8")
    for name in LIBRARIES:
        version = manifest.split(name + ":", 1)[1].split("\n", 1)[0].strip().strip('"')
        lines += [f"  {name}:", f"    version: {version}", f"    digest: {found[name]}"]
    return "\n".join(lines) + "\n"


def main(argv):
    step = argv[2] if len(argv) == 3 and argv[1] == "check" else "w0"
    if len(argv) not in (2, 3) or argv[1] not in ("check", "lock") or not step_model(step).is_dir():
        print(__doc__)
        return 2
    vendored = sorted(p.name for p in (HERE / "libraries").iterdir() if p.is_dir())
    if vendored != sorted(LIBRARIES):
        print("vendored Library folders differ from the expected set:", ", ".join(vendored))
        return 1
    frozen = [(name, version(HERE / "libraries" / name), version(source)) for name, source in LIBRARIES.items()
              if version(HERE / "libraries" / name) != version(source)]
    for name, kept, current in frozen:
        print(f"note: vendored {name} {kept} is kept frozen; the source is {current}")
    stale = [name for name, source in LIBRARIES.items()
             if name not in {f[0] for f in frozen} and not same_tree(HERE / "libraries" / name, source)]
    if stale:
        print("vendored Libraries differ from their sources:", ", ".join(stale))
        return 1
    current = (HERE / "model-lock.yaml").read_text(encoding="utf-8") if (HERE / "model-lock.yaml").exists() else "formatVersion: 1\nmodels: {}\n"
    code, out, err = run_check(current if argv[1] == "check" else "formatVersion: 1\nmodels: {}\n", step)
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
    print(f"makerspace-v2 {step} compiles; model digest {found.get('makerspace', '?')}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
