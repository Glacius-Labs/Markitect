#!/usr/bin/env python3
"""Offline .NET build and independent behavior check for the Classic Commerce example."""
from __future__ import annotations

import hashlib
import json
import os
import shutil
import subprocess
import sys
import tempfile
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any
import xml.etree.ElementTree as ET

EVIDENCE_ENV = "MARKITECT_CLASSIC_COMMERCE_EVIDENCE_DIR"
SDK_VERSION = "8.0.418"
FRAMEWORK = "net8.0"
STEP_TIMEOUT_SECONDS = 120


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def is_within(path: Path, parent: Path) -> bool:
    try:
        path.relative_to(parent)
        return True
    except ValueError:
        return False


def create_evidence_directory(repo_root: Path) -> Path:
    requested = os.environ.get(EVIDENCE_ENV)
    if requested:
        parent = Path(requested).expanduser()
        if not parent.is_absolute():
            raise ValueError(f"{EVIDENCE_ENV} must be an absolute directory path")
        parent = parent.resolve()
    else:
        parent = Path(tempfile.gettempdir()).resolve()

    if is_within(parent, repo_root):
        raise ValueError("evidence directory must be outside the source checkout and target roots")
    parent.mkdir(parents=True, exist_ok=True)
    return Path(tempfile.mkdtemp(prefix="classic-commerce-check-", dir=parent)).resolve()


def make_directory_build_props(path: Path, output_root: Path) -> None:
    project = ET.Element("Project")
    group = ET.SubElement(project, "PropertyGroup")
    output = output_root.as_posix().rstrip("/")
    ET.SubElement(group, "BaseIntermediateOutputPath").text = f"{output}/obj/$(MSBuildProjectName)/"
    ET.SubElement(group, "MSBuildProjectExtensionsPath").text = f"{output}/obj/$(MSBuildProjectName)/"
    ET.SubElement(group, "BaseOutputPath").text = f"{output}/bin/$(MSBuildProjectName)/"
    ET.indent(project, space="  ")
    path.write_bytes(ET.tostring(project, encoding="utf-8", xml_declaration=True))


def run_step(
    name: str,
    argv: list[str],
    cwd: Path,
    env: dict[str, str],
    evidence_dir: Path,
    receipt: dict[str, Any],
) -> bool:
    stdout_path = evidence_dir / f"{name}.stdout.bin"
    stderr_path = evidence_dir / f"{name}.stderr.bin"
    started = time.monotonic()
    result: dict[str, Any] = {
        "name": name,
        "argv": argv,
        "cwd": str(cwd),
        "timeoutSeconds": STEP_TIMEOUT_SECONDS,
        "stdoutPath": str(stdout_path),
        "stderrPath": str(stderr_path),
    }
    try:
        completed = subprocess.run(
            argv,
            cwd=str(cwd),
            env=env,
            stdin=subprocess.DEVNULL,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            timeout=STEP_TIMEOUT_SECONDS,
            check=False,
            shell=False,
        )
        stdout = completed.stdout
        stderr = completed.stderr
        result["exitCode"] = completed.returncode
        succeeded = completed.returncode == 0
    except subprocess.TimeoutExpired as exc:
        stdout = exc.stdout or b""
        stderr = exc.stderr or b""
        if isinstance(stdout, str):
            stdout = stdout.encode("utf-8", errors="replace")
        if isinstance(stderr, str):
            stderr = stderr.encode("utf-8", errors="replace")
        result["exitCode"] = None
        result["timedOut"] = True
        result["failure"] = f"command exceeded {STEP_TIMEOUT_SECONDS} seconds"
        succeeded = False
    except OSError as exc:
        stdout = b""
        stderr = str(exc).encode("utf-8", errors="replace")
        result["exitCode"] = None
        result["failure"] = f"could not start command: {type(exc).__name__}: {exc}"
        succeeded = False

    stdout_path.write_bytes(stdout)
    stderr_path.write_bytes(stderr)
    result["stdoutSha256"] = sha256(stdout)
    result["stderrSha256"] = sha256(stderr)
    result["durationSeconds"] = round(time.monotonic() - started, 3)
    result["status"] = "passed" if succeeded else "failed"
    receipt["steps"].append(result)
    if not succeeded:
        receipt["status"] = "failed"
        receipt["failedStep"] = name
    return succeeded


def write_receipt(path: Path, receipt: dict[str, Any]) -> None:
    path.write_text(json.dumps(receipt, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def main() -> int:
    script = Path(__file__).resolve()
    checks_dir = script.parent
    example_dir = checks_dir.parent.resolve()
    repo_root = example_dir.parent.parent.resolve()
    receipt: dict[str, Any] = {
        "schema": "markitect.classic-commerce-check/v1",
        "status": "failed",
        "createdUtc": datetime.now(timezone.utc).isoformat(),
        "sdkRequired": SDK_VERSION,
        "targetFramework": FRAMEWORK,
        "steps": [],
    }

    try:
        evidence_dir = create_evidence_directory(repo_root)
    except Exception as exc:
        print(f"Classic Commerce check could not allocate external evidence: {type(exc).__name__}: {exc}", file=sys.stderr)
        return 2

    receipt_path = evidence_dir / "result.json"
    receipt["evidenceDirectory"] = str(evidence_dir)
    receipt["receiptPath"] = str(receipt_path)
    receipt["fixedInputs"] = [
        str(example_dir / "global.json"),
        str(example_dir / "NuGet.Config"),
        str(checks_dir / "ClassicCommerce.Check.csproj"),
        str(checks_dir / "Program.cs"),
        str(script),
    ]
    project = repo_root / "src" / "Commerce" / "Commerce.csproj"
    candidate_source_paths = [
        repo_root / "src" / "Commerce" / "CreateOrderHandler.cs",
        repo_root / "src" / "Commerce" / "EffectAxis.cs",
    ]
    receipt["candidatePaths"] = [str(project), *(str(path) for path in candidate_source_paths)]
    receipt["candidateDigests"] = {}
    missing = [str(path) for path in [project, *candidate_source_paths] if not path.is_file()]
    if missing:
        receipt["failure"] = "generated candidate files are missing: " + ", ".join(missing)
        write_receipt(receipt_path, receipt)
        print(json.dumps({"status": receipt["status"], "receiptPath": str(receipt_path)}))
        return 1

    for path in [project, *candidate_source_paths]:
        receipt["candidateDigests"][str(path)] = sha256(path.read_bytes())

    dotnet = shutil.which("dotnet")
    if dotnet is None:
        receipt["failure"] = "dotnet executable was not found on PATH"
        write_receipt(receipt_path, receipt)
        print(json.dumps({"status": receipt["status"], "receiptPath": str(receipt_path)}))
        return 1

    build_root = evidence_dir / "build"
    build_root.mkdir()
    props_path = evidence_dir / "Directory.Build.props"
    make_directory_build_props(props_path, build_root)
    probe_project = checks_dir / "ClassicCommerce.Check.csproj"
    nuget_config = example_dir / "NuGet.Config"
    base_env = os.environ.copy()
    base_env.update(
        {
            "DOTNET_CLI_HOME": str(evidence_dir / "dotnet-home"),
            "DOTNET_CLI_TELEMETRY_OPTOUT": "1",
            "DOTNET_NOLOGO": "1",
            "DOTNET_SKIP_FIRST_TIME_EXPERIENCE": "1",
            "DOTNET_CLI_WORKLOAD_UPDATE_NOTIFY_DISABLE": "1",
            "NUGET_PACKAGES": str(evidence_dir / "packages"),
        }
    )
    common_props = [
        f"-p:DirectoryBuildPropsPath={props_path}",
        f"-p:ClassicCommerceProject={project}",
        "-p:UseSharedCompilation=false",
        "--nologo",
    ]
    steps = [
        ("sdk-version", [dotnet, "--version"]),
        (
            "restore-candidate",
            [dotnet, "restore", str(project), "--configfile", str(nuget_config), *common_props],
        ),
        (
            "build-candidate",
            [dotnet, "build", str(project), "--no-restore", "--configuration", "Release", *common_props],
        ),
        (
            "restore-probe",
            [dotnet, "restore", str(probe_project), "--configfile", str(nuget_config), *common_props],
        ),
        (
            "build-probe",
            [dotnet, "build", str(probe_project), "--no-restore", "--configuration", "Release", *common_props],
        ),
        (
            "run-behavior-probe",
            [dotnet, str(build_root / "bin" / "ClassicCommerce.Check" / "Release" / FRAMEWORK / "ClassicCommerce.Check.dll")],
        ),
    ]
    for name, argv in steps:
        if not run_step(name, argv, example_dir, base_env, evidence_dir, receipt):
            write_receipt(receipt_path, receipt)
            print(json.dumps({"status": receipt["status"], "receiptPath": str(receipt_path)}))
            return 1
        if name == "sdk-version":
            sdk_output = (evidence_dir / "sdk-version.stdout.bin").read_bytes().decode("utf-8", errors="replace").strip()
            if sdk_output != SDK_VERSION:
                receipt["steps"][-1]["status"] = "failed"
                receipt["status"] = "failed"
                receipt["failedStep"] = name
                receipt["failure"] = f"SDK {SDK_VERSION} is required; selected SDK was {sdk_output!r}"
                write_receipt(receipt_path, receipt)
                print(json.dumps({"status": receipt["status"], "receiptPath": str(receipt_path)}))
                return 1

    receipt["status"] = "passed"
    receipt["summary"] = "candidate build and independent behavior probe passed"
    write_receipt(receipt_path, receipt)
    print(json.dumps({"status": receipt["status"], "receiptPath": str(receipt_path)}))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
