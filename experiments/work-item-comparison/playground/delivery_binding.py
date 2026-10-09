"""Admission binding for this finite, explicitly issued adapter delivery order."""
import hashlib
import json
from pathlib import Path

ORDER_SHA256="e6c2b11ce4758d9c315d6babb01612f43ebef23e492da018424cb92732f08d73"
EXE_SHA256="3553cd6e7df5a093d8cb8301cd8088a57e0971aba71ddbe0e67f7f44a15cdf68"

def verify(plan,executable):
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
    return raw
