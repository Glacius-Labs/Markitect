"""Prepare D's four inputs; account for every authorized start, without running a provider.

    python -B d_series.py prepare DIR --series-id ID [--bindings JSON]
    python -B d_series.py init MANIFEST LEDGER_ROOT --authorization JSON
    python -B d_series.py reserve LEDGER --scenario ID --arm direct|staged|plan
    python -B d_series.py finish LEDGER ATTEMPT --result JSON

Preparation grants no execution. A separate authorization binds the exact manifest hash, series id,
provider/simulation mode and finite limits. Pending starts consume their full reservation until an
observed ending is recorded. No automatic retry, stale-lock removal, ledger reset or provider call.
"""
import argparse
from contextlib import contextmanager
from datetime import datetime, timezone
import hashlib
import json
import math
import os
from pathlib import Path
import re
import time
import uuid

import freeze
from materialize import commit, materialize, apply_edits, git
import plan_packet
from scenarios import BY_ID, ALL_ARMS, POLICY
from cases import A2_TEST, A4_ORDER

HERE = Path(__file__).resolve().parent
SLOTS = ("A2", "ARTIFACTS", "DEP-SEMANTIC", "R1")
MAX_STARTS, MAX_SECONDS = 12, 12 * 3600
ID = re.compile(r"[A-Za-z0-9][A-Za-z0-9_-]{0,79}\Z")


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def write_new(path, data):
    with Path(path).open("x", encoding="utf-8", newline="\n") as stream:
        json.dump(data, stream, indent=2, ensure_ascii=False)
        stream.write("\n")
        stream.flush()
        os.fsync(stream.fileno())


def seconds(text):
    match = re.fullmatch(r"([1-9][0-9]*)([smh])", text)
    if not match:
        raise ValueError("unsupported finite duration: " + text)
    return int(match[1]) * {"s": 1, "m": 60, "h": 3600}[match[2]]


def now():
    return datetime.now(timezone.utc).isoformat()


def prepare(directory, series_id, bindings=None):
    if not ID.fullmatch(series_id):
        raise ValueError("invalid series id")
    directory = Path(directory).resolve()
    root = HERE.parent.parent
    if directory == root or root in directory.parents:
        raise ValueError("D preparation must be outside the source repository")
    if git(root, "status", "--porcelain").strip():
        raise ValueError("commit the source candidate before preparing its D manifest")
    lock = json.loads(freeze.LOCK.read_text(encoding="utf-8"))
    if lock["kit"] != freeze.kit_files():
        raise ValueError("kit differs from the lock")
    directory.mkdir(parents=True, exist_ok=False)
    # Rotate the three arm orders across slots; one predetermined start for each pair, no retry slots.
    order = [{"scenario": scenario, "arm": ALL_ARMS[(index + offset) % 3]}
             for index, scenario in enumerate(SLOTS) for offset in range(3)]
    manifest = {"formatVersion": 1, "seriesId": series_id, "executionAuthority": False,
                "sourceRevision": git(root, "rev-parse", "HEAD").strip(), "policy": POLICY,
                "lockSha256": sha(freeze.LOCK), "kitDigest": lock["kitDigest"], "order": order,
                "ceilings": {"starts": MAX_STARTS, "seconds": MAX_SECONDS}, "scenarios": {},
                "bindings": bindings, "executionReadiness": "unverified: no provider transport run",
                "controls": {}, "assessmentControls": lock["assessmentControls"]}
    for scenario_id in SLOTS:
        expected = lock["scenarios"][scenario_id]
        arms = {}
        for arm in ALL_ARMS:
            target = directory / "packets" / scenario_id / arm
            if arm == "plan":
                described = plan_packet.build_from_packet(arms["direct"]["repo"], target)
                if described["head"] != expected["plan"]["head"]:
                    raise ValueError("plan packet differs from lock: " + scenario_id)
                arms[arm] = {"repo": str(target), **expected["plan"]}
            else:
                described = freeze.scenario_packet(scenario_id, target, arm)
                if described["head"] != expected["markitect"][arm]["head"]:
                    raise ValueError("packet differs from lock: " + scenario_id + "/" + arm)
                arms[arm] = {"repo": str(target), **expected["markitect"][arm]}
        manifest["scenarios"][scenario_id] = {"case": expected["case"], "version": expected["version"],
            "budget": expected["budget"], "profile": BY_ID[scenario_id].profile or __import__("cases").CASES[expected["case"]].profile,
            "arms": arms}
    # Hidden control inputs stay in the operator preparation, never in a writer's mounted packet.
    for case_id in ("S-N", "H-N", "W-N", "W-M"):
        path = directory / "controls" / case_id
        materialize(case_id, path)
        manifest["controls"][case_id] = freeze.describe(path)
    for fault in (A2_TEST, A4_ORDER):
        path = directory / "controls" / ("clause-" + fault.id)
        materialize("S-N", path)
        apply_edits(path, fault.edits, "clause control")
        commit(path, "Accepted project artifacts")
        manifest["controls"]["clause-" + fault.id] = freeze.describe(path)
    write_new(directory / "manifest.json", manifest)
    return manifest


def initialize(manifest_path, ledger_root, authorization_path):
    manifest_path, authorization_path = Path(manifest_path).resolve(), Path(authorization_path).resolve()
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    grant = json.loads(authorization_path.read_text(encoding="utf-8"))
    if not ID.fullmatch(manifest["seriesId"]):
        raise ValueError("invalid series id")
    if (grant.get("seriesId") != manifest["seriesId"] or grant.get("manifestSha256") != sha(manifest_path)
            or not grant.get("authorizationReference") or grant.get("mode") not in ("simulation", "provider")):
        raise ValueError("authorization must bind this exact manifest, series and execution mode")
    starts, duration = grant.get("maxStarts"), grant.get("maxSeconds")
    if type(starts) is not int or not 0 < starts <= MAX_STARTS or type(duration) is not int or not 0 < duration <= MAX_SECONDS:
        raise ValueError("authorization exceeds finite D ceilings")
    if grant["mode"] == "provider":
        validate_bindings(manifest.get("bindings"))
    directory = Path(ledger_root).resolve() / manifest["seriesId"]
    directory.mkdir(parents=True, exist_ok=False)
    header = {"kind": "series", "seriesId": manifest["seriesId"], "manifest": str(manifest_path),
              "manifestSha256": sha(manifest_path), "authorizationSha256": sha(authorization_path),
              "authorizationReference": grant["authorizationReference"], "mode": grant["mode"],
              "maxStarts": starts, "maxSeconds": duration, "order": manifest["order"], "created": now()}
    with (directory / "ledger.jsonl").open("x", encoding="utf-8", newline="\n") as stream:
        stream.write(json.dumps(header) + "\n")
        stream.flush(); os.fsync(stream.fileno())
    return directory


@contextmanager
def locked(directory):
    path = Path(directory) / "writer.lock"
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    try:
        os.close(fd)
        yield
    finally:
        path.unlink()


def records(directory):
    path = Path(directory) / "ledger.jsonl"
    rows = [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines()]
    if not rows or rows[0].get("kind") != "series":
        raise ValueError("missing series header")
    if sha(rows[0]["manifest"]) != rows[0]["manifestSha256"]:
        raise ValueError("frozen manifest changed")
    return rows


def append(directory, row):
    with (Path(directory) / "ledger.jsonl").open("a", encoding="utf-8", newline="\n") as stream:
        stream.write(json.dumps(row) + "\n")
        stream.flush(); os.fsync(stream.fileno())


def reserve(directory, scenario_id, arm):
    with locked(directory):
        rows = records(directory); header = rows[0]
        starts = [r for r in rows if r["kind"] == "start"]
        endings = {r["attempt"]: r for r in rows if r["kind"] == "end"}
        if any(r["attempt"] not in endings for r in starts):
            raise ValueError("pending start: record its ending or explicit abandonment first")
        index = len(starts)
        if index >= min(header["maxStarts"], len(header["order"])):
            raise ValueError("start budget exhausted")
        if header["order"][index] != {"scenario": scenario_id, "arm": arm}:
            raise ValueError("start differs from the frozen order; no implicit replay")
        used = sum(endings[r["attempt"]]["chargedSeconds"] for r in starts)
        remaining = header["maxSeconds"] - used
        cap = min(seconds(BY_ID[scenario_id].budget.execution), remaining)
        if cap <= 0:
            raise ValueError("series time budget exhausted")
        row = {"kind": "start", "attempt": str(uuid.uuid4()), "number": index + 1, "scenario": scenario_id,
               "arm": arm, "started": now(), "reservedSeconds": cap, "mode": header["mode"], "replayOf": None}
        append(directory, row)
        return row


def finish(directory, attempt, result_path=None):
    with locked(directory):
        rows = records(directory)
        if any(r.get("attempt") == attempt and r["kind"] == "end" for r in rows):
            raise ValueError("ending already recorded")
        start = next((r for r in rows if r.get("attempt") == attempt and r["kind"] == "start"), None)
        if not start:
            raise ValueError("unknown attempt")
        result = json.loads(Path(result_path).read_text(encoding="utf-8")) if result_path else None
        if result and (result.get("attempt") != attempt or result.get("scenario") != start["scenario"]
                       or result.get("mode") != start["mode"] or result.get("arm") != start["arm"]):
            raise ValueError("result does not bind this attempt/scenario/mode")
        if result and result.get("status") == "completed":
            manifest = json.loads(Path(rows[0]["manifest"]).read_text(encoding="utf-8"))
            if (result.get("manifestSha256") != rows[0]["manifestSha256"]
                    or result.get("head") != manifest["scenarios"][start["scenario"]]["arms"][start["arm"]]["head"]):
                raise ValueError("completed result lacks its frozen manifest/workspace binding")
        if result and result.get("reportPath") and sha(result["reportPath"]) != result.get("reportSha256"):
            raise ValueError("report bytes differ from the result binding")
        elapsed = (datetime.now(timezone.utc) - datetime.fromisoformat(start["started"])).total_seconds()
        # A missing ending or backwards clock consumes the full reservation; no optimistic recovery/reset.
        charged = max(1, math.ceil(elapsed)) if result and elapsed >= 0 else start["reservedSeconds"]
        row = {"kind": "end", "attempt": attempt, "ended": now(), "chargedSeconds": charged,
               "status": result.get("status", "error") if result else "abandoned",
               "overrunSeconds": max(0, charged - start["reservedSeconds"]),
               "result": str(Path(result_path).resolve()) if result_path else None,
               "resultSha256": sha(result_path) if result_path else None}
        append(directory, row)
        return row


def validate_bindings(bindings):
    if not isinstance(bindings, dict) or not all(arm in bindings for arm in ALL_ARMS):
        raise ValueError("all three arm bindings are required")
    identities, builds = set(), set()
    for arm in ALL_ARMS:
        b = bindings[arm]
        for key in ("provider", "model", "effort", "imageID", "profilePath", "profileSha256"):
            if not b.get(key): raise ValueError("missing binding: " + arm + "/" + key)
        if b["provider"] not in ("codex", "claude") or not re.fullmatch(r"sha256:[a-f0-9]{64}", b["imageID"]):
            raise ValueError("invalid provider/image binding")
        if sha(b["profilePath"]) != b["profileSha256"]:
            raise ValueError("profile bytes differ from frozen binding")
        # JSON is a YAML subset accepted by the product Runtime decoder. Require the
        # actual Runtime input in that encoding, rather than trusting a sidecar's claims
        # or implementing another permissive YAML/profile decoder in this harness.
        profile = json.loads(Path(b["profilePath"]).read_text(encoding="utf-8"))
        resources = b.get("resources") or {}
        if any(type(resources.get(k)) is not int or resources[k] <= 0 for k in ("milliCPUs", "memoryBytes", "processes")):
            raise ValueError("explicit CPU/memory/process bounds required for each arm")
        identities.add((b["provider"], b["model"], b["effort"], b["imageID"],
                        *(resources[k] for k in ("milliCPUs", "memoryBytes", "processes"))))
        if arm != "plan":
            for path_key, hash_key in (("productBinary", "productBinarySha256"), ("runConfigPath", "runConfigSha256")):
                if sha(b[path_key]) != b[hash_key]: raise ValueError("product/run binding differs")
            if not re.fullmatch(r"[a-f0-9]{40}", b.get("productRevision", "")):
                raise ValueError("full product revision required")
            builds.add((b["productRevision"], b["productBinarySha256"]))
            if profile.get("apiVersion") != "markitect.runtime/v1" or not profile.get("agentProfiles"):
                raise ValueError("bind the actual JSON-encoded product Runtime profile")
            section = "codexAppServer" if b["provider"] == "codex" else "claudeCode"
            for agent in profile["agentProfiles"].values():
                if set(agent) != {section}:
                    raise ValueError("every Agent profile must use the frozen provider")
                settings = agent[section]; session = settings["session"]
                observed = {"provider": b["provider"], "model": session["model"], "effort": session["effort"],
                            "imageID": settings["imageID"]}
                actual_resources = {"milliCPUs": session["cpus"] * 1000,
                                    "memoryBytes": session["memoryBytes"], "processes": session["pids"]}
                if any(observed[k] != b[k] for k in observed) or actual_resources != resources:
                    raise ValueError("Runtime profile contents disagree with frozen identity/resources")
        else:
            import plan_arm
            profile = plan_arm.profile_data(b["profilePath"])
            if (any(profile[k] != b[k] for k in ("provider", "model", "effort", "imageID"))
                    or any(profile["limits"][k] != resources[k] for k in resources)):
                raise ValueError("ordinary profile contents disagree with frozen identity/resources")
    if len(identities) != 1:
        raise ValueError("the three arms must bind the same provider/model/effort/image/compute bounds")
    if len(builds) != 1:
        raise ValueError("direct and staged must bind the same product revision and binary")


def measure(report, events=(), final=None, observations=None):
    """Observed fields only. Turn counts are never silently equated with container starts."""
    run = (report or {}).get("run") or {}
    turns = run.get("turns")
    usage = run.get("usage") or (final or {}).get("usage") or {}
    cost = usage.get("costUSD", usage.get("total_cost_usd"))
    if cost == 0 and not usage.get("costKnown"):
        cost = None  # the product's zero default is not a known zero bill
    containers = (final or {}).get("containers") or (observations or {}).get("containers")
    started = [t.get("started") for t in turns or [] if t.get("started") and not t["started"].startswith("0001")]
    ended = [t.get("ended") for t in turns or [] if t.get("ended") and not t["ended"].startswith("0001")]
    engine_seconds = None
    if started and ended:
        parse = lambda value: datetime.fromisoformat(value.replace("Z", "+00:00"))
        engine_seconds = (max(map(parse, ended)) - min(map(parse, started))).total_seconds()
    elapsed = (observations or {}).get("elapsedSeconds", (final or {}).get("elapsedSeconds"))
    turn_count = len(turns) if isinstance(turns, list) else (final or {}).get("turns")
    return {"evidenceMode": (final or {}).get("mode", (observations or {}).get("mode", "unclassified")),
            "turns": turn_count,
            "turnUnit": "markitect-agent-turn" if isinstance(turns, list) else (final or {}).get("turnUnit"),
            "containers": containers, "usage": usage or None, "costUSD": cost,
            "elapsedSeconds": elapsed, "engineSeconds": engine_seconds,
            "eventRecords": len(events), "unknowns": [name for name, value in (
                ("containers", containers), ("costUSD", cost),
                ("usage", usage or None), ("elapsedSeconds", elapsed), ("turns", turn_count)) if value is None]}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    p = sub.add_parser("prepare"); p.add_argument("directory"); p.add_argument("--series-id", required=True); p.add_argument("--bindings")
    p = sub.add_parser("init"); p.add_argument("manifest"); p.add_argument("ledger_root"); p.add_argument("--authorization", required=True)
    p = sub.add_parser("reserve"); p.add_argument("ledger"); p.add_argument("--scenario", choices=SLOTS, required=True); p.add_argument("--arm", choices=ALL_ARMS, required=True)
    p = sub.add_parser("finish"); p.add_argument("ledger"); p.add_argument("attempt"); p.add_argument("--result")
    p = sub.add_parser("measure"); p.add_argument("--report"); p.add_argument("--events"); p.add_argument("--final"); p.add_argument("--observations")
    a = parser.parse_args()
    if a.command == "prepare":
        result = prepare(a.directory, a.series_id, json.loads(Path(a.bindings).read_text()) if a.bindings else None)
        result = {k: result[k] for k in ("seriesId", "sourceRevision", "lockSha256", "executionAuthority", "executionReadiness")}
    elif a.command == "init": result = str(initialize(a.manifest, a.ledger_root, a.authorization))
    elif a.command == "reserve": result = reserve(a.ledger, a.scenario, a.arm)
    elif a.command == "finish": result = finish(a.ledger, a.attempt, a.result)
    else:
        import evaluate
        events, damaged = evaluate.load_jsonl(a.events)
        paths = {key: getattr(a, key) for key in ("report", "events", "final", "observations") if getattr(a, key)}
        report = evaluate.load_report(a.report)
        final = json.loads(Path(a.final).read_text()) if a.final else None
        observations = json.loads(Path(a.observations).read_text()) if a.observations else None
        result = {"measure": measure(report, events, final, observations), "damagedEventLines": damaged,
                  "inputs": {key: sha(path) for key, path in paths.items()}}
    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
