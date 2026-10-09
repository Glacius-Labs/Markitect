"""Admission binding for this finite, explicitly issued adapter delivery order."""
import hashlib
import json
from pathlib import Path
from datetime import datetime, timezone

ORDER_SHA256="e6c2b11ce4758d9c315d6babb01612f43ebef23e492da018424cb92732f08d73"
EXE_SHA256="3553cd6e7df5a093d8cb8301cd8088a57e0971aba71ddbe0e67f7f44a15cdf68"
SNAPSHOT_ORDER_SHA256="58cf8d06944223b77f51062bfa48ecb224bee9e5f5651abec18597dff1664d63"
SNAPSHOT_DESTINATION="C:/Users/Consiliari/Documents/Conventional-Workspace-Snapshot-Study-20261009"
SNAPSHOT_LEDGER="C:/Users/Consiliari/Documents/Conventional-Workspace-Snapshot-Execution-20261009.reservation.json"

def reserve(plan, destination):
    """Reserve this separately ordered case once, without rewriting old grants."""
    if plan.get("id")!="conventional-workspace-snapshot-execution-20261009":return
    if Path(destination).resolve()!=Path(SNAPSHOT_DESTINATION).resolve():
        raise ValueError("snapshot destination differs from the frozen owned path")
    if plan.get("executionLedgerPath")!=SNAPSHOT_LEDGER:
        raise ValueError("snapshot reservation path changed")
    data={"schema":1,"order":plan["id"],"reservedAt":datetime.now(timezone.utc).isoformat(),"destination":str(destination),
          "planSha256":hashlib.sha256(json.dumps(plan,sort_keys=True).encode()).hexdigest(),"scope":"one fresh trajectory; preserve reservation on any failure"}
    with Path(SNAPSHOT_LEDGER).open("xb") as stream:
        stream.write((json.dumps(data,indent=2)+"\n").encode())

def _verify_snapshot(plan, executable):
    authority=plan.get("authority",{})
    path=Path(authority.get("orderPath",""))
    if not path.is_absolute():raise ValueError("snapshot order path must be absolute")
    raw=path.read_bytes()
    if authority.get("verifiedRootOrderSha256")!=SNAPSHOT_ORDER_SHA256 or hashlib.sha256(raw).hexdigest()!=SNAPSHOT_ORDER_SHA256:
        raise ValueError("snapshot v2 authority bytes changed")
    order=json.loads(raw)
    if order.get("revision")!=2 or order.get("id")!=plan.get("id") or authority.get("directUserTurn")!=order["authority"]["directUserTurn"]:
        raise ValueError("snapshot order revision/id/human-turn mismatch")
    if plan.get("completionTarget")!="workspace_snapshot" or len(plan.get("trajectories",[]))!=1:
        raise ValueError("snapshot order permits one frozen workspace-snapshot trajectory")
    if plan.get("destination")!=SNAPSHOT_DESTINATION or plan.get("executionLedgerPath")!=SNAPSHOT_LEDGER:
        raise ValueError("snapshot owned destination/reservation binding changed")
    if Path(SNAPSHOT_LEDGER).exists():raise ValueError("snapshot single trajectory reservation already consumed")
    values={"jobWallSeconds":14400,"turnWallSeconds":5400,"implementationSeconds":10800,"freshFinalSeconds":2700,
            "captureAssessmentReserveSeconds":900,"overallWallSeconds":14400,"startAllowance":256,"outerTurnsPerTrajectory":4,"maxRuntimeStartupChecks":2}
    if any(plan.get(key)!=value for key,value in values.items()):
        raise ValueError("snapshot finite resources changed")
    if plan.get("providerFreeChecks")!={"maximum":2,"alreadyConsumed":0,"perTrajectoryMaximum":2,"nativeModelCanaries":0}:
        raise ValueError("snapshot startup check accounting changed")
    for key in ("absoluteEndUtc","lastStartUtc"):
        if plan.get(key)!=order["limits"][key]:raise ValueError("snapshot absolute time binding changed")
    if datetime.now(timezone.utc)>=datetime.fromisoformat(plan["lastStartUtc"].replace("Z","+00:00")):
        raise ValueError("snapshot last start deadline exceeded")
    if plan.get("executableSha256")!=EXE_SHA256 or hashlib.sha256(Path(executable).read_bytes()).hexdigest()!=EXE_SHA256:
        raise ValueError("snapshot existing executable binding changed")
    if plan.get("limits",{}).get("maxConcurrentAgents")!=4 or plan["limits"].get("maxDepth")!=2:
        raise ValueError("snapshot shared concurrency/depth changed")
    choice=plan["trajectories"][0]
    cfg=choice.get("config",{})
    if choice.get("method")!="Conventional" or choice.get("case")!="readinglog" or cfg.get("backend")!="codex-app-server" or cfg.get("model")!="gpt-6-luna" or cfg.get("effort")!="high":
        raise ValueError("snapshot frozen Conventional AppServer case/model changed")
    if choice.get("commandArgs",[])!=[] or choice.get("factory")!="conventional.adapter:ConventionalAdapter" or choice.get("setupSourceRoot")!="public/conventional":
        raise ValueError("snapshot native command/factory/setup changed")
    opts=cfg.get("runtimeOptions",{})
    if opts!={"sandbox":"workspace-write","approvalPolicy":"never","memoryEnabled":False,"nativeHelperModel":"gpt-6-luna","nativeHelperEffort":"high","windowsSandbox":"mxc","nativeMaxConcurrentAgents":3,"allowLoginShell":False}:
        raise ValueError("snapshot declared native runtime changed")
    closure=Path(__file__).resolve().parent/"pilots/conventional-workspace-snapshot-execution-20261009/closure.json"
    if closure.exists():raise ValueError("snapshot order already closed; no retry")
    return raw

def verify(plan,executable):
    if plan.get("id")=="conventional-workspace-snapshot-execution-20261009":
        return _verify_snapshot(plan,executable)
    authority=plan.get("authority",{})
    path=Path(authority.get("orderPath",""))
    if not path.is_absolute():raise ValueError("delivery requires exact absolute Root order path")
    raw=path.read_bytes()
    if authority.get("verifiedRootOrderSha256")!=ORDER_SHA256 or hashlib.sha256(raw).hexdigest()!=ORDER_SHA256:
        raise ValueError("delivery authority bytes changed")
    order=json.loads(raw)
    if order["id"]!=plan["id"] or authority.get("directUserTurn")!=order["authority"]["directUserTurn"]:
        raise ValueError("delivery id/human-turn mismatch")
    checks=plan.get("providerFreeChecks",{})
    if checks!={"maximum":6,"alreadyConsumed":2,"remainingAtSourceFreeze":4,"perTrajectoryMaximum":2,"nativeModelCanaries":0}:
        raise ValueError("new provider-free startup accounting differs from finite order")
    if plan.get("maxRuntimeStartupChecks")!=2 or 2+2*len(plan["trajectories"])>6:
        raise ValueError("cumulative six startup-check limit exceeded")
    native=hashlib.sha256(Path(executable).read_bytes()).hexdigest()
    if plan.get("executableSha256")!=EXE_SHA256 or native!=EXE_SHA256:
        raise ValueError("existing native executable binding changed")
    if plan.get("limits",{}).get("concurrency")!=4 or plan["limits"].get("depth")!=2:
        raise ValueError("shared concurrency/depth limit changed")
    for choice in plan["trajectories"]:
        opts=choice["config"].get("runtimeOptions",{})
        if (choice["method"]!="Conventional" or opts.get("windowsSandbox")!="mxc" or
            opts.get("sandbox")!="workspace-write" or opts.get("approvalPolicy")!="never" or
            opts.get("nativeMaxConcurrentAgents")!=3 or opts.get("memoryEnabled") is not False or
            choice["config"].get("model")!="gpt-6-luna" or choice["config"].get("effort")!="high"):
            raise ValueError("delivery child-local runtime stratum changed")
    closure=Path(__file__).resolve().parent/"pilots/conventional-variant-adapter-delivery-20261009/closure.json"
    if closure.exists() and json.loads(closure.read_text(encoding="utf-8")).get("state")=="closed":
        raise ValueError("delivery order is closed; no automatic quota refill or actual replay")
    return raw
