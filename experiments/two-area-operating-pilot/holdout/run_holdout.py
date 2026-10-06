import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import uuid
import xml.etree.ElementTree as ET

HOLDOUT = Path(__file__).resolve().parent
EXPECTED_DOTNET = "10.0.103"
PROJECTS = ("Orders", "Fulfillment")
EXCLUDED_DIRS = {"bin", "obj", ".git"}


def manifest(root: Path):
    entries = {}
    for path in sorted(root.rglob("*")):
        relative = path.relative_to(root)
        if any(part.lower() in EXCLUDED_DIRS for part in relative.parts):
            continue
        if path.is_symlink():
            raise RuntimeError(f"symlink is not accepted in source input: {relative.as_posix()}")
        if path.is_file():
            name = relative.as_posix()
            raw = path.read_bytes()
            entries[name] = {"sha256": hashlib.sha256(raw).hexdigest(), "bytes": len(raw)}
    return entries


def tree_hash(project_manifests):
    payload = "".join(
        f"{project}/{name}\0{item['sha256']}\n"
        for project, entries in sorted(project_manifests.items())
        for name, item in sorted(entries.items())
    )
    return hashlib.sha256(payload.encode("utf-8")).hexdigest()


def run_logged(command, cwd: Path, env, log_path: Path):
    proc = subprocess.run(command, cwd=cwd, env=env, text=True, stdout=subprocess.PIPE,
                          stderr=subprocess.STDOUT, check=False)
    log_path.write_text(proc.stdout, encoding="utf-8")
    return proc


def copy_sources(source: Path, destination: Path):
    destination.mkdir(parents=True, exist_ok=True)
    for path in sorted(source.rglob("*")):
        relative = path.relative_to(source)
        if any(part.lower() in EXCLUDED_DIRS for part in relative.parts):
            continue
        if path.is_symlink():
            raise RuntimeError(f"symlink is not accepted in source input: {relative.as_posix()}")
        target = destination / relative
        if path.is_dir():
            target.mkdir(parents=True, exist_ok=True)
        elif path.is_file():
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(path, target)


def classify_project_reference(area_root: Path, include: str, condition: str = ""):
    raw = (include or "").strip()
    if condition or not raw or any(token in raw for token in ("$", "%", "@", "*", "?", ";")):
        return {"include": raw, "condition": condition, "status": "not_established"}
    target = Path(raw)
    if not target.is_absolute():
        target = area_root / target
    resolved = target.resolve()
    try:
        relative = resolved.relative_to(area_root.resolve())
    except ValueError:
        return {"include": raw, "condition": condition, "resolved_path": str(resolved), "status": "outside_area"}
    if not resolved.is_file():
        return {"include": raw, "condition": condition, "resolved_path": str(resolved), "status": "not_established"}
    return {"include": raw, "condition": condition, "resolved_path": str(resolved),
            "area_relative_path": relative.as_posix(), "status": "within_area"}


def bounded_inspection(source_roots):
    report = {}
    io_pattern = re.compile(r"\b(?:System\.IO|File|Directory|FileStream|HttpClient|WebRequest|Socket|TcpClient|UdpClient)\b")
    mutable_pattern = re.compile(r"\bstatic\s+(?!readonly\b|const\b)[^;=]+(?:=|;)")
    for project, root in source_roots.items():
        project_file = root / f"{project}.csproj"
        try:
            xml_root = ET.parse(project_file).getroot()
            package_refs = [node.attrib.get("Include", node.attrib.get("Update", ""))
                            for node in xml_root.iter() if node.tag.split("}")[-1] in {"PackageReference", "PackageDownload"}]
            project_ref_nodes = [node for node in xml_root.iter() if node.tag.split("}")[-1] == "ProjectReference"]
            project_refs = []
            for node in project_ref_nodes:
                include = node.attrib.get("Include", node.attrib.get("Update", ""))
                project_refs.append(classify_project_reference(root, include, node.attrib.get("Condition", "")))
            target_frameworks = [value.strip() for node in xml_root.iter() if node.tag.split("}")[-1] == "TargetFramework" for value in [(node.text or "")]]
            target_frameworks += [value.strip() for node in xml_root.iter() if node.tag.split("}")[-1] == "TargetFrameworks" for value in (node.text or "").split(";") if value.strip()]
            imports = [node.attrib.get("Project", "") for node in xml_root.iter()
                       if node.tag.split("}")[-1] == "Import"]
        except Exception as exc:
            package_refs, project_refs, imports, target_frameworks = [], [{"status": "not_established", "error": f"{type(exc).__name__}: {exc}"}], [], []
        io_mentions = []
        static_mutable_mentions = []
        for source_file in sorted(root.rglob("*.cs")):
            if any(part.lower() in EXCLUDED_DIRS for part in source_file.relative_to(root).parts):
                continue
            try:
                text = source_file.read_text(encoding="utf-8-sig")
            except UnicodeDecodeError:
                continue
            for line_number, line in enumerate(text.splitlines(), 1):
                if io_pattern.search(line):
                    io_mentions.append({"file": source_file.relative_to(root).as_posix(), "line": line_number,
                                        "text": line.strip()[:240]})
                if mutable_pattern.search(line):
                    static_mutable_mentions.append({"file": source_file.relative_to(root).as_posix(),
                                                    "line": line_number, "text": line.strip()[:240]})
        report[project] = {
            "package_references_in_project_xml": package_refs,
            "project_references_in_project_xml": project_refs,
            "imports_in_project_xml": imports,
            "target_frameworks_in_project_xml": target_frameworks,
            "io_related_source_text_mentions": io_mentions,
            "static_mutable_source_text_mentions": static_mutable_mentions,
        }
    return report


def project_reference_policy_status(inspections):
    states = [reference.get("status", "not_established")
              for item in inspections.values() for reference in item["project_references_in_project_xml"]]
    if "outside_area" in states:
        return "failed"
    if "not_established" in states:
        return "not_established"
    return "passed"


def main():
    parser = argparse.ArgumentParser(description="Offline two-project behavior holdout for Orders and Fulfillment")
    parser.add_argument("artifact_root", help="Artifact root containing src/Orders and src/Fulfillment")
    parser.add_argument("max_quantity", type=int, choices=(10, 20), help="10 for changed policy or 20 for baseline")
    parser.add_argument("--label", choices=("candidate", "mechanics-only-not-pilot"), default="candidate")
    args = parser.parse_args()

    artifact_root = Path(args.artifact_root).expanduser().resolve(strict=True)
    original_roots = {name: artifact_root / "src" / name for name in PROJECTS}
    for name, root in original_roots.items():
        if not (root / f"{name}.csproj").is_file():
            raise SystemExit(f"missing required project: {root / (name + '.csproj')}")

    source_before = {name: manifest(root) for name, root in original_roots.items()}
    for name, entries in source_before.items():
        if f"{name}.csproj" not in entries:
            raise SystemExit(f"source manifest is incomplete for {name}")

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
    copied_roots = {name: run_root / "source" / "src" / name for name in PROJECTS}
    runner_root = run_root / "runner"
    runner_root.mkdir(parents=True)
    for name in PROJECTS:
        copy_sources(original_roots[name], copied_roots[name])
    copied_manifest = {name: manifest(root) for name, root in copied_roots.items()}
    source_after_copy = {name: manifest(root) for name, root in original_roots.items()}
    copy_matches = source_before == copied_manifest == source_after_copy
    inspections = bounded_inspection(copied_roots)
    no_external_packages = all(not item["package_references_in_project_xml"] for item in inspections.values())
    target_frameworks_ok = all(item["target_frameworks_in_project_xml"] == ["net10.0"] for item in inspections.values())
    project_ref_policy = project_reference_policy_status(inspections)

    (runner_root / "Acceptance.csproj").write_text('''<Project Sdk="Microsoft.NET.Sdk">\n  <PropertyGroup>\n    <OutputType>Exe</OutputType>\n    <TargetFramework>net10.0</TargetFramework>\n    <ImplicitUsings>enable</ImplicitUsings>\n    <Nullable>enable</Nullable>\n  </PropertyGroup>\n  <ItemGroup>\n    <ProjectReference Include="../source/src/Orders/Orders.csproj" />\n    <ProjectReference Include="../source/src/Fulfillment/Fulfillment.csproj" />\n  </ItemGroup>\n</Project>\n''', encoding="utf-8")
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
    build = execution = None
    behavior = None
    if restore.returncode == 0:
        build = run_logged([dotnet, "build", "Acceptance.csproj", "--no-restore", "--configuration", "Release",
                            ], runner_root, env, run_root / "build.log")
    if build is not None and build.returncode == 0:
        execution = run_logged([dotnet, str(runner_root / "bin" / "Release" / "net10.0" / "Acceptance.dll"), str(args.max_quantity)],
                               runner_root, env, run_root / "execution.log")
        try:
            behavior = json.loads(execution.stdout.strip().splitlines()[-1])
        except Exception:
            behavior = {"parseError": "acceptance program did not emit a final JSON object"}

    source_after = {name: manifest(root) for name, root in original_roots.items()}
    target_unchanged = source_before == source_after
    behavior_ok = isinstance(behavior, dict) and behavior.get("passed") is True
    own_success = (copy_matches and target_unchanged and no_external_packages
                   and target_frameworks_ok and project_ref_policy != "failed" and restore.returncode == 0 and build is not None and build.returncode == 0
                   and execution is not None and execution.returncode == 0 and behavior_ok)
    manifest_record = {
        "artifact_root": str(artifact_root),
        "source_before": source_before,
        "copied_source": copied_manifest,
        "source_after_copy": source_after_copy,
        "source_after_run": source_after,
        "copy_matches_source_before_and_after_copy": copy_matches,
        "source_unchanged_before_vs_after_run": target_unchanged,
        "copied_source_tree_sha256": tree_hash(copied_manifest),
    }
    (run_root / "source-manifests.json").write_text(json.dumps(manifest_record, indent=2) + "\n", encoding="utf-8")
    classification = "baseline_max_20" if args.max_quantity == 20 else "changed_max_10"
    result = {
        "schema": "markitect-two-area-holdout/v1",
        "label": args.label,
        "evidence_class": "technical_behavior_only_not_human_acceptance",
        "classification": classification,
        "status": "passed" if own_success else "failed",
        "max_quantity": args.max_quantity,
        "artifact_root": str(artifact_root),
        "projects": ["src/Orders/Orders.csproj", "src/Fulfillment/Fulfillment.csproj"],
        "dotnet_version": version,
        "offline_restore": True,
        "external_packages": "project XML inspected; restore constrained to empty feed list and isolated package cache",
        "isolated_run_root": str(run_root),
        "source_manifest_file": str(run_root / "source-manifests.json"),
        "copy_matches_source_before_and_after_copy": copy_matches,
        "source_unchanged_before_vs_after_run": target_unchanged,
        "no_package_references_in_project_xml": no_external_packages,
        "target_frameworks_are_net10": target_frameworks_ok,
        "project_reference_area_policy_status": project_ref_policy,
        "bounded_source_inspection": inspections,
        "copied_source_tree_sha256": tree_hash(copied_manifest),
        "restore_exit_code": restore.returncode,
        "build_exit_code": build.returncode if build else None,
        "execution_exit_code": execution.returncode if execution else None,
        "behavior": behavior,
        "logs": {"restore": str(run_root / "restore.log"), "build": str(run_root / "build.log"),
                 "execution": str(run_root / "execution.log")},
    }
    result_path = HOLDOUT / "results" / f"{run_id}.json"
    result_path.parent.mkdir(parents=True, exist_ok=True)
    result_path.write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(result, indent=2))
    return 0 if own_success else 1


CSHARP_PROGRAM = r'''using System.Text.Json;
using Orders;
using Fulfillment;

if (args.Length != 1 || !int.TryParse(args[0], out var maxQuantity) || (maxQuantity != 10 && maxQuantity != 20))
{
    Console.Error.WriteLine("Expected exactly one maximum quantity argument: 10 or 20.");
    return 2;
}
var quantities = new SortedSet<int> { int.MinValue, int.MinValue + 1, -2, -1, 0, maxQuantity + 1, maxQuantity + 2, int.MaxValue - 1, int.MaxValue };
for (var q = 1; q <= maxQuantity; q++) quantities.Add(q);
var forward = quantities.ToArray();
var reverse = forward.Reverse().ToArray();
var errors = new List<string>();
var calls = 0;

bool Valid(int q) => q >= 1 && q <= maxQuantity;
void Check(bool condition, string message) { if (!condition) errors.Add(message); }

void ExerciseOrders(string name, IEnumerable<int> inputs)
{
    var seen = new List<int>();
    var handler = new CreateOrderHandler(q => seen.Add(q));
    foreach (var q in inputs)
    {
        var before = seen.Count;
        bool actual;
        try { actual = handler.Execute(q); }
        catch (Exception ex) { errors.Add($"{name}: Execute({q}) threw {ex.GetType().FullName}: {ex.Message}"); calls++; continue; }
        calls++;
        var expected = Valid(q);
        Check(actual == expected, $"{name}: Execute({q}) returned {actual}; expected {expected}.");
        var delta = seen.Count - before;
        Check(delta == (expected ? 1 : 0), $"{name}: callback count for Execute({q}) was {delta}; expected {(expected ? 1 : 0)}.");
        if (expected && delta == 1) Check(seen[^1] == q, $"{name}: callback argument changed for Execute({q}).");
    }
}

void ExerciseFulfillment(string name, IEnumerable<int> inputs)
{
    var dispatched = new List<int>();
    var handler = new FulfillOrderHandler(q => dispatched.Add(q));
    foreach (var q in inputs)
    {
        var before = dispatched.Count;
        bool actual;
        try { actual = handler.Execute(q); }
        catch (Exception ex) { errors.Add($"{name}: Execute({q}) threw {ex.GetType().FullName}: {ex.Message}"); calls++; continue; }
        calls++;
        var expected = Valid(q);
        Check(actual == expected, $"{name}: Execute({q}) returned {actual}; expected {expected}.");
        var delta = dispatched.Count - before;
        Check(delta == (expected ? 1 : 0), $"{name}: callback count for Execute({q}) was {delta}; expected {(expected ? 1 : 0)}.");
        if (expected && delta == 1) Check(dispatched[^1] == q, $"{name}: callback argument changed for Execute({q}).");
    }
}

void ExerciseInterleavedOrders()
{
    var seenA = new List<int>(); var seenB = new List<int>();
    var a = new CreateOrderHandler(q => seenA.Add(q)); var b = new CreateOrderHandler(q => seenB.Add(q));
    for (var i = 0; i < forward.Length; i++)
    {
        foreach (var (label, handler, ledger, q) in new[] { ("A", a, seenA, forward[i]), ("B", b, seenB, reverse[i]) })
        {
            var before = ledger.Count; bool actual;
            try { actual = handler.Execute(q); }
            catch (Exception ex) { errors.Add($"interleaved Orders {label}: Execute({q}) threw {ex.GetType().FullName}: {ex.Message}"); calls++; continue; }
            calls++;
            var expected = Valid(q); var delta = ledger.Count - before;
            Check(actual == expected, $"interleaved Orders {label}: Execute({q}) returned {actual}; expected {expected}.");
            Check(delta == (expected ? 1 : 0), $"interleaved Orders {label}: callback count was {delta}.");
            if (expected && delta == 1) Check(ledger[^1] == q, $"interleaved Orders {label}: callback argument changed for {q}.");
        }
    }
}

void ExerciseInterleavedFulfillment()
{
    var seenA = new List<int>(); var seenB = new List<int>();
    var a = new FulfillOrderHandler(q => seenA.Add(q)); var b = new FulfillOrderHandler(q => seenB.Add(q));
    for (var i = 0; i < forward.Length; i++)
    {
        foreach (var (label, handler, ledger, q) in new[] { ("A", a, seenA, forward[i]), ("B", b, seenB, reverse[i]) })
        {
            var before = ledger.Count; bool actual;
            try { actual = handler.Execute(q); }
            catch (Exception ex) { errors.Add($"interleaved Fulfillment {label}: Execute({q}) threw {ex.GetType().FullName}: {ex.Message}"); calls++; continue; }
            calls++;
            var expected = Valid(q); var delta = ledger.Count - before;
            Check(actual == expected, $"interleaved Fulfillment {label}: Execute({q}) returned {actual}; expected {expected}.");
            Check(delta == (expected ? 1 : 0), $"interleaved Fulfillment {label}: callback count was {delta}.");
            if (expected && delta == 1) Check(ledger[^1] == q, $"interleaved Fulfillment {label}: callback argument changed for {q}.");
        }
    }
}

void ExerciseComposition(string name, IEnumerable<int> inputs)
{
    var persisted = new List<int>(); var dispatched = new List<int>();
    var events = new List<(string Kind, int Quantity)>();
    var fulfillment = new FulfillOrderHandler(q => { dispatched.Add(q); events.Add(("dispatch", q)); });
    var orders = new CreateOrderHandler(q => { persisted.Add(q); events.Add(("order-callback", q)); fulfillment.Execute(q); });
    foreach (var q in inputs)
    {
        var p0 = persisted.Count; var d0 = dispatched.Count; var e0 = events.Count;
        bool actual;
        try { actual = orders.Execute(q); }
        catch (Exception ex) { errors.Add($"{name}: Orders.Execute({q}) threw {ex.GetType().FullName}: {ex.Message}"); calls++; continue; }
        calls++;
        var expected = Valid(q);
        Check(actual == expected, $"{name}: Orders.Execute({q}) returned {actual}; expected {expected}.");
        Check(persisted.Count - p0 == (expected ? 1 : 0), $"{name}: upstream callback count for {q} was {persisted.Count - p0}.");
        Check(dispatched.Count - d0 == (expected ? 1 : 0), $"{name}: downstream callback count for {q} was {dispatched.Count - d0}.");
        if (expected)
        {
            if (persisted.Count - p0 == 1) Check(persisted[^1] == q, $"{name}: upstream quantity changed for {q}.");
            if (dispatched.Count - d0 == 1) Check(dispatched[^1] == q, $"{name}: downstream quantity changed for {q}.");
            var actualEvents = events.Skip(e0).ToArray();
            Check(actualEvents.SequenceEqual(new[] { ("order-callback", q), ("dispatch", q) }),
                  $"{name}: callback order/quantities were unexpected for {q}.");
        }
        else Check(events.Count == e0, $"{name}: rejected upstream call {q} caused downstream activity.");
    }
}

ExerciseOrders("Orders-forward", forward);
ExerciseOrders("Orders-reverse", reverse);
ExerciseOrders("Orders-repeated", forward.Concat(reverse).Concat(new[] { 1, maxQuantity, 0, maxQuantity + 1 }));
ExerciseInterleavedOrders();
ExerciseFulfillment("Fulfillment-forward", forward);
ExerciseFulfillment("Fulfillment-reverse", reverse);
ExerciseFulfillment("Fulfillment-repeated", forward.Concat(reverse).Concat(new[] { 1, maxQuantity, 0, maxQuantity + 1 }));
ExerciseInterleavedFulfillment();
ExerciseComposition("composition-forward", forward);
ExerciseComposition("composition-reverse", reverse);
ExerciseComposition("composition-repeated", forward.Concat(reverse).Concat(new[] { 1, maxQuantity, 0, maxQuantity + 1 }));

var result = new { passed = errors.Count == 0, maxQuantity, testedQuantities = forward, callCount = calls,
                   failedCheckCount = errors.Count, errors };
Console.WriteLine(JsonSerializer.Serialize(result));
return errors.Count == 0 ? 0 : 1;
'''

if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:
        print(json.dumps({"schema": "markitect-two-area-holdout/v1", "status": "failed",
                          "error": f"{type(exc).__name__}: {exc}"}), file=sys.stderr)
        raise







