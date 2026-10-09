"""Execute the separately authorized finite Conventional wrapper pilot.

The plan fixes public tasks and runtime resources. This controller neither repairs
the candidate nor changes the user request. It keeps external raw evidence and
assesses immutable main independently at each station.
"""
from datetime import datetime, timedelta, timezone
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import time

import lifecycle

ROOT = Path(__file__).resolve().parent
CHECK_SPEC = importlib.util.spec_from_file_location("pilot_public_checker", ROOT / "public/common/checks/acceptance.py")
CHECKER = importlib.util.module_from_spec(CHECK_SPEC)
CHECK_SPEC.loader.exec_module(CHECKER)


def now():
    return datetime.now(timezone.utc)


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def save(path, value):
    path.write_bytes((json.dumps(value, ensure_ascii=False, indent=2) + "\n").encode("utf-8"))


def git(repo, *args):
    return subprocess.check_output(["git", "-C", str(repo), *args], timeout=600).decode("utf-8").strip()


def setup(case, backend, destination, plan, executable, version, source_sha, overall_expiry):
    repo = destination / case / "repo"
    audit = destination / case / "audit"
    (destination / case).mkdir()
    begun = now()
    profile = {"id": "codex-luna-high", "modelRequested": plan["model"], "effortRequested": plan["effort"],
               "nativeBackend": backend, "entrance": "wrapper CLI", "referenceEquivalence": "unverified"}
    prepared = lifecycle.prepare(ROOT / "public", repo, audit, case=case, method="Conventional", profile=profile,
                                 pins={"playgroundSourceCommit": source_sha},
                                 authorization={"execution_authorized": True, "actualOrderRef": plan["id"],
                                                "scope": "Conventional-only finite pilot; no Markitect or old-grant reopening"})
    git(repo, "checkout", "main")
    agents = repo / "AGENTS.md"
    agents.write_bytes(agents.read_bytes() + b"\n" + (ROOT / "public/conventional/AGENTS.fragment.md").read_bytes() +
                      b"\nThe installed finite runtime profile is .study/runtime-profile.json. Keep its model/effort and resources for all native roles.\n")
    save(repo / ".study/runtime-profile.json", {"model": plan["model"], "effort": plan["effort"],
         "nativeHelperModel": plan["model"], "nativeHelperEffort": plan["effort"],
         "jobWallSeconds": plan["jobWallSeconds"], "turnWallSeconds": plan["turnWallSeconds"],
         "sharedPilotStartAllowance": plan["startAllowance"], "nativeStartScope": "parents/helpers/final/failed starts",
         "nativeStartEnforcement": "cooperative plus observed receipts; no hard OS/global start counter",
         "privateData": "Prior solutions, other arms, private assessments and credentials are not inputs.",
         "userMemory": "disabled in child-local config to exclude old study context; user config unchanged"})
    git(repo, "add", "AGENTS.md", ".study/runtime-profile.json")
    subprocess.run(["git", "-C", str(repo), "diff", "--cached", "--check"], check=True, timeout=600)
    git(repo, "commit", "-m", "Install ordinary Conventional rules and frozen runtime profile")
    setup_main = git(repo, "rev-parse", "main")
    git(repo, "checkout", "codex/backlog")
    git(repo, "merge", "--ff-only", "main")
    order_path = audit / "actual-order.json"
    config_path = audit / "wrapper-config.json"
    sources = [ROOT / "conventional" / name for name in ("service.py", "backends.py", "mcp.py")]
    sources += [ROOT / "conventional_wrapper.py", Path(__file__).resolve(), ROOT / "lifecycle.py",
                ROOT / "public/common/checks/acceptance.py", executable]
    user_config = Path("C:/Users/Consiliari/.codex/config.toml")
    options = {"sandbox": "workspace-write", "approvalPolicy": "never", "memoryEnabled": False,
               "nativeHelperModel": plan["model"], "nativeHelperEffort": plan["effort"]}
    config = {"schema": 1, "execution_authorized": True, "actualOrderPath": str(order_path),
              "backend": backend, "command": [str(executable)], "filePins": {str(path): digest(path) for path in sources},
              "cwd": str(repo), "audit": str(audit), "model": plan["model"], "effort": plan["effort"],
              "timeoutSeconds": plan["turnWallSeconds"], "requestTimeoutSeconds": 600,
              "threadOptions": {}, "turnOptions": {}, "runtimeOptions": options,
              "runtimeBinding": {"executableVersion": version, "sourceCommit": source_sha,
                   "userConfigSha256": digest(user_config) if user_config.exists() else None,
                   "effectiveModel": None, "effectiveEffort": None, "configurationChanges": "child-local overrides only",
                   "userConfigBindingPolicy": "observed full-file hash; mutable UI config is not an admission pin; inherited/effective loaded values unknown",
                   "serverManagement": "owned_stdio" if backend == "codex-app-server" else "native_exec",
                   "externalEndpointRequired": False, "sharedHost": True, "referenceEquivalence": "unverified"}}
    save(config_path, config)
    job_expiry = min(begun + timedelta(seconds=plan["jobWallSeconds"]), overall_expiry)
    expires = job_expiry - timedelta(seconds=900)
    save(order_path, {"execution_authorized": True, "actualOrderRef": plan["id"],
                     "configSha256": digest(config_path), "runId": prepared["runId"],
                     "expiresAt": expires.isoformat(), "maxTurns": 4})
    manifest = {"case": case, "backend": backend, "sourceCommit": source_sha,
                "preparedRunId": prepared["runId"], "seedCommit": prepared["seed"]["mainCommit"],
                "setupMainCommit": setup_main, "setupStartedAt": begun.isoformat(), "setupEndedAt": now().isoformat(),
                "configSha256": digest(config_path), "orderSha256": digest(order_path),
                "runtimeOptions": options, "jobExpiresAt": job_expiry.isoformat(),
                "actorExpiresAt": expires.isoformat(), "captureAssessmentReserveSeconds": 900, "externalEndpoint": None, "serverOwner": "Conventional wrapper",
                "runtimeReady": "not established until native launch and protocol/work receipts",
                "sourceAndInputManifest": lifecycle._manifest(repo, exclude_git=True),
                "effectiveValues": "unknown unless native-reported; requested values are not proof"}
    save(audit / "setup-binding.json", manifest)
    lifecycle.record_setup(audit, owner="Scientist ordinary Conventional setup", status="complete", details=manifest)
    return repo, audit, config_path, expires


def mcp_status(config_path, run_id, audit, station):
    """Exercise the real operator MCP facade without launching a second actor."""
    messages = [
        {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25",
         "capabilities":{},"clientInfo":{"name":"pilot-controller","version":"1"}}},
        {"jsonrpc":"2.0","method":"notifications/initialized"},
        {"jsonrpc":"2.0","id":2,"method":"tools/list"},
        {"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"conventional_status","arguments":{"run_id":run_id}}}]
    wire = ("\n".join(json.dumps(message) for message in messages)+"\n").encode("utf-8")
    result = subprocess.run([sys.executable,"-B",str(ROOT/"conventional_wrapper.py"),"--config",str(config_path),"mcp"],
                            input=wire,capture_output=True,timeout=600)
    (audit / f"operator-mcp-{station}.stdout").write_bytes(result.stdout)
    (audit / f"operator-mcp-{station}.stderr").write_bytes(result.stderr)
    responses = [json.loads(line) for line in result.stdout.splitlines()]
    reply = next(response for response in responses if response.get("id")==3)
    if result.returncode or reply.get("result",{}).get("isError"):
        raise RuntimeError("operator MCP status failed; native evidence retained")
    return {"returnCode":result.returncode,"runId":reply["result"]["structuredContent"]["runId"],
            "state":reply["result"]["structuredContent"]["state"],"startsConsumed":0}


def observe_mcp(config_path, run_id, audit, station):
    try:
        return mcp_status(config_path, run_id, audit, station)
    except Exception as exc:
        receipt = {"status": "observation_failed", "detail": type(exc).__name__ + ": " + str(exc),
                   "nativeRunId": run_id, "startsConsumed": 0,
                   "action": "continue waiting on the same owned controller; no native replay"}
        save(audit / f"operator-mcp-{station}.error.json", receipt)
        return receipt


def run_turn(config_path, audit, parent, expiry, station):
    prompt = ROOT / "public/task-prompt.txt"
    command = [sys.executable,"-B",str(ROOT/"conventional_wrapper.py"),"--config",str(config_path),
               "resume" if parent else "start"]
    if parent:
        command.append(parent)
    command += ["--prompt-file",str(prompt)]
    stdout = audit / f"controller-{station}.stdout"
    stderr = audit / f"controller-{station}.stderr"
    begun = now()
    with stdout.open("xb") as out, stderr.open("xb") as err:
        proc = subprocess.Popen(command,stdout=out,stderr=err)
        # Outer expiry includes ample interruption/capture grace; do not truncate
        # a healthy native turn below its frozen remaining window.
        deadline = time.monotonic() + max(0,(expiry-now()).total_seconds()) + 300
        handle = None
        while proc.poll() is None:
            if handle is None:
                lines = stdout.read_bytes().splitlines()
                if lines:
                    try:
                        handle = json.loads(lines[0])["runId"]
                    except (json.JSONDecodeError,KeyError):
                        pass
                    if handle:
                        save(audit/f"start-{station}.json",{"runId":handle,"controllerPid":proc.pid,
                             "startedAt":begun.isoformat(),"mcpStatus":observe_mcp(config_path,handle,audit,station)})
                        print(json.dumps({"event":"native-attempt-accepted","station":station,"runId":handle}),flush=True)
            if time.monotonic() >= deadline:
                # Signal only this owned handle; no stale PID or global cleanup.
                if handle:
                    subprocess.run([sys.executable,"-B",str(ROOT/"conventional_wrapper.py"),"--config",str(config_path),
                                    "cancel",handle],capture_output=True,timeout=600)
                raise RuntimeError("outer finite window expired; owned cancellation requested; outcome requires reconciliation")
            time.sleep(0.25)
    lines = [json.loads(line) for line in stdout.read_bytes().splitlines()]
    result = lines[-1] if lines else {"state":"not_dispatched","detail":stderr.read_text(encoding="utf-8",errors="replace")}
    if "runId" not in result:
        result["state"] = "not_dispatched"
    result.update(controllerExitCode=proc.returncode,controllerStartedAt=begun.isoformat(),controllerEndedAt=now().isoformat())
    save(audit/f"turn-{station}.json",result)
    return result


def trajectory(case, backend, destination, plan, executable, version, source_sha, overall_expiry):
    begun = now()
    repo,audit,config,expiry = setup(case,backend,destination,plan,executable,version,source_sha,overall_expiry)
    result = {"case":case,"backend":backend,"startedAt":begun.isoformat(),"stations":[],
              "status":"running","taskSuccess":None,"methodComparison":"NOT RUN",
              "startAccounting":{"outerAttempts":0,"nativeHelpers":"unknown until receipt review","hardGlobalCounter":False}}
    parent = None
    for number in range(1,5):
        stage = f"S{number}"
        turn = run_turn(config,audit,parent,expiry,stage)
        if "runId" in turn:
            result["startAccounting"]["outerAttempts"] += 1
        lifecycle.record_execution(audit,metadata={"station":stage,"wrapperState":turn["state"],"wrapperRunId":turn.get("runId"),
                                         "nativeSessionId":turn.get("nativeSessionId"),"nativeTurnId":turn.get("nativeTurnId")})
        if turn["state"] in {"uncertain","needs_input"}:
            # Never snapshot changing source as complete or replay an unresolved native turn.
            result["stations"].append({"station":stage,"nativeState":turn["state"],"assessment":"NOT RUN",
                                       "reason":"unresolved native ownership; raw worktree preserved"})
            result["status"] = "blocked_by_runtime"
            break
        captured = lifecycle.snapshot(repo,audit)
        immutable = audit/f"snapshot-{stage}"/"immutable-main"
        assessment = CHECKER.assess(immutable,case,number)
        save(audit/f"assessment-{stage}.json",assessment)
        row = {"station":stage,"nativeState":turn["state"],"mainCommit":captured["immutableMain"]["commit"],
               "snapshotBindingSha256":digest(audit/f"snapshot-{stage}"/"binding.json"),
               "publicChecks":assessment,"usage":turn.get("usage"),"usageScope":turn.get("usageScope"),
               "humanAcceptance":"not established"}
        result["stations"].append(row)
        print(json.dumps({"event":"station-captured","case":case,"station":stage,"nativeState":turn["state"],
                          "checks":{status:sum(f["status"]==status for f in assessment["findings"]) for status in ("PASS","FAIL","NOT RUN","EVALUATION_ERROR")}}),flush=True)
        events = audit / "conventional-execution" / turn.get("runId", "missing") / "events.jsonl"
        native_setup_failure = events.exists() and "helper_unknown_error: setup refresh had errors" in events.read_text(encoding="utf-8", errors="replace")
        if native_setup_failure:
            lifecycle.freeze(repo, audit, reason="known native tool setup failure; completed actor turn; no automatic replay")
            result["status"] = "blocked_by_native_tool_setup"
            result["runtimeFailure"] = "helper_unknown_error: setup refresh had errors"
            break
        if turn["state"] != "completed":
            lifecycle.freeze(repo,audit,reason="known terminal runtime failure; no automatic retry")
            result["status"] = "runtime_failed"
            break
        parent = turn["runId"]
        if number<4:
            lifecycle.advance(repo,audit)
        else:
            final = lifecycle.freeze(repo,audit,reason="four fixed station turns completed; independent assessment follows")
            result.update(status="trajectory_completed",finalMainCommit=final["immutableMain"]["commit"],
                          finalMainPath=str(audit/"final-freeze/immutable-main"))
    result["endedAt"] = now().isoformat()
    save(audit/"trajectory-result.json",result)
    return result


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--plan",required=True,type=Path)
    parser.add_argument("--destination",required=True,type=Path)
    parser.add_argument("--executable",required=True,type=Path)
    parser.add_argument("--version",required=True)
    args=parser.parse_args()
    plan=json.loads(args.plan.read_text(encoding="utf-8"))
    expected=[{"case":"roombook","backend":"codex-cli"},{"case":"readinglog","backend":"codex-app-server"}]
    if plan.get("execution_authorized") is not True or plan.get("trajectories")!=expected:
        raise ValueError("only the explicitly ordered two-trajectory pilot is supported")
    if (plan.get("jobWallSeconds")!=14400 or plan.get("turnWallSeconds")!=5400 or
            plan.get("startAllowance")!=256 or plan.get("model")!="gpt-6-luna" or plan.get("effort")!="high"):
        raise ValueError("pilot does not match the verified finite order")
    destination=args.destination.resolve()
    if not args.destination.is_absolute() or destination.exists() or ROOT in destination.parents:
        raise ValueError("a fresh absolute external destination is required")
    if git(ROOT,"status","--porcelain"):
        raise ValueError("source checkout must be clean and committed before native starts")
    begun = now()
    overall_expiry = begun + timedelta(seconds=plan["overallWallSeconds"])
    destination.mkdir(parents=True)
    source_sha=git(ROOT,"rev-parse","HEAD")
    save(destination/"pilot-binding.json",{"plan":plan,"planSha256":digest(args.plan),"sourceCommit":source_sha,
         "controllerSourceSha256":digest(__file__),"executable":str(args.executable.resolve()),
         "executableSha256":digest(args.executable),"version":args.version,"startedAt":begun.isoformat(), "overallExpiresAt":overall_expiry.isoformat()})
    results=[]
    for choice in expected:
        try:
            results.append(trajectory(choice["case"],choice["backend"],destination,plan,args.executable.resolve(),args.version,source_sha,overall_expiry))
        except Exception as exc:
            results.append({**choice,"status":"controller_error","detail":type(exc).__name__+": "+str(exc),
                            "preservation":"all existing source/receipts retained; no retry or cleanup"})
        save(destination/"pilot-results.json",{"trajectories":results,"interpretation":"case and transport confounded; no comparative winner"})
        if results[-1]["status"] in {"controller_error", "blocked_by_runtime"}:
            audit = destination / choice["case"] / "audit"
            if audit.exists() and not (audit / "trajectory-result.json").exists():
                save(audit / "trajectory-result.json", results[-1])
            # Unresolved ownership must be reconciled before a second actor starts.
            break
    print(json.dumps({"event":"pilot-ended","results":results},ensure_ascii=False),flush=True)


if __name__=="__main__":
    for stream in (sys.stdin,sys.stdout,sys.stderr):
        if hasattr(stream,"reconfigure"): stream.reconfigure(encoding="utf-8")
    main()
