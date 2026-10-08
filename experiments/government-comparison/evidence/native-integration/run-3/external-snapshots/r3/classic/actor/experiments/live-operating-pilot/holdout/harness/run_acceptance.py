import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import uuid

HOLDOUT = Path(__file__).resolve().parent.parent
EXPECTED_DOTNET = "10.0.103"
PROJECT_REL = Path("src") / "Commerce" / "Commerce.csproj"


def manifest(root: Path):
    entries = {}
    for path in sorted(root.rglob("*")):
        if path.is_symlink():
            raise RuntimeError(f"symlink is not accepted in the source input: {path.relative_to(root)}")
        if path.is_file():
            rel = path.relative_to(root).as_posix()
            digest = hashlib.sha256(path.read_bytes()).hexdigest()
            entries[rel] = {"sha256": digest, "bytes": path.stat().st_size}
    return entries


def tree_hash(entries):
    payload = "".join(f"{name}\0{item['sha256']}\n" for name, item in sorted(entries.items()))
    return hashlib.sha256(payload.encode("utf-8")).hexdigest()


def run_logged(command, cwd: Path, env, log_path: Path):
    proc = subprocess.run(command, cwd=cwd, env=env, text=True, stdout=subprocess.PIPE,
                          stderr=subprocess.STDOUT, check=False)
    log_path.write_text(proc.stdout, encoding="utf-8")
    return proc


def main():
    parser = argparse.ArgumentParser(description="Offline behavioral acceptance for Commerce.CreateOrderHandler")
    parser.add_argument("artifact_root", help="Explicit artifact root containing src/Commerce/Commerce.csproj")
    parser.add_argument("max_quantity", type=int, choices=(10, 20), help="10 for changed or 20 for baseline")
    parser.add_argument("--label", default="live-pilot", help="Use mechanics-only-stub for harness validation")
    args = parser.parse_args()

    artifact_root = Path(args.artifact_root).expanduser().resolve(strict=True)
    project_path = artifact_root / PROJECT_REL
    if not project_path.is_file():
        raise SystemExit(f"missing required project: {project_path}")
    source_root = project_path.parent
    original_before = manifest(source_root)
    if PROJECT_REL.name not in {Path(k).name for k in original_before}:
        raise SystemExit("project source manifest is empty or incomplete")

    dotnet = shutil.which("dotnet")
    if not dotnet:
        raise SystemExit("dotnet was not found on PATH")
    version_proc = subprocess.run([dotnet, "--version"], text=True, stdout=subprocess.PIPE,
                                  stderr=subprocess.STDOUT, check=False)
    version = version_proc.stdout.strip()
    if version_proc.returncode != 0 or version != EXPECTED_DOTNET:
        raise SystemExit(f"requires installed dotnet {EXPECTED_DOTNET}; found {version!r}")

    run_id = uuid.uuid4().hex
    run_root = HOLDOUT / "runs" / run_id
    copied_root = run_root / "source" / "src" / "Commerce"
    runner_root = run_root / "runner"
    runner_root.mkdir(parents=True)
    shutil.copytree(source_root, copied_root, dirs_exist_ok=True)
    copied_manifest = manifest(copied_root)
    original_after_copy = manifest(source_root)
    copy_matches = original_before == copied_manifest == original_after_copy

    (runner_root / "Acceptance.csproj").write_text('''<Project Sdk="Microsoft.NET.Sdk">\n  <PropertyGroup>\n    <OutputType>Exe</OutputType>\n    <TargetFramework>net10.0</TargetFramework>\n    <ImplicitUsings>enable</ImplicitUsings>\n    <Nullable>enable</Nullable>\n  </PropertyGroup>\n  <ItemGroup>\n    <ProjectReference Include="../source/src/Commerce/Commerce.csproj" />\n  </ItemGroup>\n</Project>\n''', encoding="utf-8")
    (runner_root / "NuGet.Config").write_text('''<?xml version="1.0" encoding="utf-8"?>\n<configuration><packageSources><clear /></packageSources></configuration>\n''', encoding="utf-8")
    (runner_root / "Program.cs").write_text(CSHARP_PROGRAM, encoding="utf-8")

    env = os.environ.copy()
    env.update({
        "DOTNET_CLI_HOME": str(run_root / "dotnet-home"),
        "DOTNET_CLI_TELEMETRY_OPTOUT": "1",
        "DOTNET_NOLOGO": "1",
        "DOTNET_MULTILEVEL_LOOKUP": "0",
        "NUGET_PACKAGES": str(run_root / "packages"),
        "NUGET_HTTP_CACHE_PATH": str(run_root / "http-cache"),
        "NUGET_PLUGINS_CACHE_PATH": str(run_root / "plugins-cache"),
    })
    (run_root / "dotnet-home").mkdir(parents=True, exist_ok=True)
    (run_root / "packages").mkdir(parents=True, exist_ok=True)
    restore = run_logged([dotnet, "restore", "Acceptance.csproj", "--configfile", "NuGet.Config",
                          "--packages", str(run_root / "packages")], runner_root, env, run_root / "restore.log")
    build = None
    execution = None
    csharp_result = None
    if restore.returncode == 0:
        build = run_logged([dotnet, "build", "Acceptance.csproj", "--no-restore", "--configuration", "Release"],
                           runner_root, env, run_root / "build.log")
    if build is not None and build.returncode == 0:
        execution = run_logged([dotnet, str(runner_root / "bin" / "Release" / "net10.0" / "Acceptance.dll"),
                                str(args.max_quantity)], runner_root, env, run_root / "execution.log")
        try:
            csharp_result = json.loads(execution.stdout.strip().splitlines()[-1])
        except Exception:
            csharp_result = {"parseError": "acceptance program did not emit a final JSON object"}

    original_after = manifest(source_root)
    target_unchanged = original_before == original_after
    own_success = (copy_matches and target_unchanged and restore.returncode == 0 and build is not None
                   and build.returncode == 0 and execution is not None and execution.returncode == 0
                   and isinstance(csharp_result, dict) and csharp_result.get("passed") is True)
    result = {
        "schema": "markitect-live-acceptance/v1",
        "label": args.label,
        "evidence_class": "pilot_behavioral_acceptance" if args.label == "live-pilot" else "mechanics_only_not_pilot_evidence",
        "acceptance_status": "passed" if own_success and args.label == "live-pilot" else (
            "failed" if not own_success else "not_a_pilot_result"),
        "max_quantity": args.max_quantity,
        "artifact_root": str(artifact_root),
        "project": str(PROJECT_REL.as_posix()),
        "dotnet_version": version,
        "offline_restore": True,
        "external_packages": "not_used_by_runner; project restore constrained to empty feed list",
        "isolated_run_root": str(run_root),
        "copy_manifest_matches_source_before_and_after_copy": copy_matches,
        "source_unchanged_before_vs_after": target_unchanged,
        "tested_source_tree_sha256": tree_hash(copied_manifest),
        "source_sha256_manifest_before": original_before,
        "tested_source_sha256_manifest": copied_manifest,
        "source_sha256_manifest_after": original_after,
        "restore_exit_code": restore.returncode,
        "build_exit_code": build.returncode if build else None,
        "execution_exit_code": execution.returncode if execution else None,
        "behavior": csharp_result,
        "logs": {"restore": str(run_root / "restore.log"), "build": str(run_root / "build.log"),
                 "execution": str(run_root / "execution.log")},
    }
    result_path = HOLDOUT / "results" / f"{run_id}.json"
    result_path.parent.mkdir(parents=True, exist_ok=True)
    result_path.write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(result, indent=2))
    return 0 if own_success else 1


CSHARP_PROGRAM = r'''using System.Text.Json;
using Commerce;

if (args.Length != 1 || !int.TryParse(args[0], out var maxQuantity) || (maxQuantity != 10 && maxQuantity != 20))
{
    Console.Error.WriteLine("Expected exactly one maximum quantity argument: 10 or 20.");
    return 2;
}
var quantities = new SortedSet<int>
{
    int.MinValue, int.MinValue + 1, -2, -1, 0,
    maxQuantity + 1, maxQuantity + 2, int.MaxValue - 1, int.MaxValue
};
for (var q = 1; q <= maxQuantity; q++) quantities.Add(q);
var forward = quantities.ToArray();
var reverse = forward.Reverse().ToArray();
var errors = new List<string>();
var calls = 0;

void Exercise(string name, IEnumerable<int> inputs)
{
    var observed = new List<int>();
    var handler = new CreateOrderHandler(q => observed.Add(q));
    foreach (var q in inputs)
    {
        var before = observed.Count;
        bool actual;
        try { actual = handler.Execute(q); }
        catch (Exception ex)
        {
            errors.Add($"{name}: Execute({q}) threw {ex.GetType().FullName}: {ex.Message}");
            calls++;
            continue;
        }
        calls++;
        var expected = q >= 1 && q <= maxQuantity;
        if (actual != expected) errors.Add($"{name}: Execute({q}) returned {actual}; expected {expected}.");
        var delta = observed.Count - before;
        var expectedDelta = expected ? 1 : 0;
        if (delta != expectedDelta) errors.Add($"{name}: Execute({q}) invoked callback {delta} times; expected {expectedDelta}.");
        if (expected && (delta != 1 || observed[^1] != q))
            errors.Add($"{name}: callback argument for Execute({q}) was not the unchanged quantity.");
    }
}

// Different instances and call orders expose callback or state leakage; each has its own callback ledger.
Exercise("instance-forward", forward);
Exercise("instance-reverse", reverse);
Exercise("instance-repeated", forward.Concat(reverse).Concat(new[] { 1, maxQuantity, 0, maxQuantity + 1 }));

var result = new
{
    passed = errors.Count == 0,
    maxQuantity,
    testedQuantities = forward,
    instanceCount = 3,
    callCount = calls,
    failedCheckCount = errors.Count,
    errors
};
Console.WriteLine(JsonSerializer.Serialize(result));
return errors.Count == 0 ? 0 : 1;
'''

if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:
        print(json.dumps({"schema": "markitect-live-acceptance/v1", "acceptance_status": "failed",
                          "error": f"{type(exc).__name__}: {exc}"}), file=sys.stderr)
        raise
