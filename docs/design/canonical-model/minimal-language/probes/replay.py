"""Replay bounded historical design probes without changing their evidence files."""
from pathlib import Path
import argparse
import hashlib
import json
import os
import shutil
import subprocess
import sys
import tempfile


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def read_json(path):
    return json.loads(path.read_text(encoding="utf-8"))


def verify_inputs(directory, report):
    for name, expected in report["inputSha256"].items():
        actual = hashlib.sha256((directory / name).read_bytes()).hexdigest()
        require(actual == expected, f"Input changed: {directory.name}/{name}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--with-go", action="store_true")
    args = parser.parse_args()
    probes = Path(__file__).resolve().parent
    base = probes.parent
    repo = base.parents[3]
    archive = probes / "archive"
    bootstrap = read_json(base / "bootstrap/probe.json")
    compact = read_json(base / "bootstrap/compact-probe.json")
    identity = read_json(base / "identities/probe.json")
    for directory, report in [(base / "bootstrap", bootstrap),
                              (base / "bootstrap", compact),
                              (base / "identities", identity)]:
        verify_inputs(directory, report)
    expected_scripts = {
        "markitect-bootstrap-probe.py": bootstrap["prototypeSha256"],
        "markitect-inline-property-probe.py": compact["normalizerPrototypeSha256"],
        "markitect-identity-walkthrough-probe.py": identity["resolverPrototypeSha256"],
    }
    for name, expected in expected_scripts.items():
        require(hashlib.sha256((archive / name).read_bytes()).hexdigest() == expected,
                f"Archived prototype changed: {name}")
    require(compact["bootstrapPrototypeSha256"] == bootstrap["prototypeSha256"],
            "Compact/bootstrap provenance differs")
    require(identity["structuralPrototypeSha256"] == bootstrap["prototypeSha256"],
            "Identity/bootstrap provenance differs")

    with tempfile.TemporaryDirectory(prefix="markitect-design-replay-") as location:
        scratch = Path(location)
        runtime = scratch / "runtime"
        runtime.mkdir()
        for name in expected_scripts:
            shutil.copyfile(archive / name, runtime / name)
        env = dict(os.environ, TEMP=str(runtime), TMP=str(runtime),
                   PYTHONDONTWRITEBYTECODE="1")
        for name in ["bootstrap", "identities"]:
            destination = scratch / name
            destination.mkdir()
            for source in (base / name).glob("*.yaml"):
                shutil.copyfile(source, destination / source.name)
        cases = [
            ("markitect-bootstrap-probe.py", "bootstrap", "probe.json", bootstrap),
            ("markitect-inline-property-probe.py", "bootstrap", "compact-probe.json", compact),
            ("markitect-identity-walkthrough-probe.py", "identities", "probe.json", identity),
        ]
        for script, folder, report_name, expected in cases:
            subprocess.run([sys.executable, "-B", str(runtime / script),
                            str(scratch / folder)], cwd=repo, env=env, check=True)
            actual = read_json(scratch / folder / report_name)
            comparable = lambda value: {k: v for k, v in value.items() if k != "sourceRevision"}
            require(comparable(actual) == comparable(expected), f"Report drift: {script}")
            print(f"Historical result reproduced: {folder}/{report_name}")

        if args.with_go:
            require(shutil.which("go"), "Go is required for --with-go")
            for source_name, report_name in [
                ("probe.go.txt", "structural-probe.json"),
                ("exchange-probe.go.txt", "exchange-walkthrough-probe.json"),
            ]:
                expected = read_json(base / report_name)
                verify_inputs(base, expected)
                overlay = scratch / "overlay.json"
                virtual = repo / "src/cmd/markitect/language_design_probe.go"
                overlay.write_text(json.dumps({"Replace": {
                    str(virtual): str(archive / source_name)}}), encoding="utf-8")
                result = subprocess.run(
                    ["go", "run", "-overlay", str(overlay), str(virtual),
                     str(archive / "inputs.json")], cwd=repo, env=env,
                    check=True, text=True, capture_output=True)
                actual = json.loads(result.stdout)
                for key, value in actual.items():
                    require(expected.get(key) == value, f"Go report drift: {report_name}/{key}")
                print(f"Historical Go result reproduced: {report_name}")
    print("Bounded design probes passed; no product or semantic-index conformance claimed.")


if __name__ == "__main__":
    main()
