"""Resume only the same unlaunched Classic controller; never refill its wall/start budget."""
import importlib.util
import json
import os
from pathlib import Path
import time

spec = importlib.util.spec_from_file_location("finite_driver", Path(__file__).with_name("run-native-integration.py"))
d = importlib.util.module_from_spec(spec)
spec.loader.exec_module(d)

def main():
    d.require_freeze()
    review = d.ROOT / "public/classic-prelaunch-recovery-review.md"
    frozen = json.loads((d.EVIDENCE / "classic-recovery-freeze.json").read_bytes())
    for item in frozen["inputs"]:
        if d.binding(item["path"]) != item:
            raise ValueError("recovery input changed")
    if not any(Path(x["path"]).resolve() == review.resolve() for x in frozen["inputs"]):
        raise ValueError("independent recovery review missing")
    a,r,raw,captured,request_path = d.authority("classic")
    ledger = a.ledger()
    with ledger.transaction() as db:
        if db.execute("SELECT COUNT(*) FROM attempts").fetchone()[0] or db.execute("SELECT COUNT(*) FROM controller_processes").fetchone()[0]:
            raise ValueError("recovery requires zero role and native process effects")
    b = d.FixtureBudget(d.EXTERNAL / "native-starts.sqlite", d.EXTERNAL / "released-native-grant.json", d.GRANT_SHA, {
        "government":d.government.PIN["accepted"]["binary"]["path"],
        "classic":d.classic.inspect_packet(d.classic_integration.PACKET)["binary"]["path"]})
    if any(x["product"] == "classic" for x in b.snapshot()["starts"]):
        raise ValueError("any prior Classic start claim forbids this prelaunch recovery")
    evidence = Path(r["evidenceDirectory"])
    bundle = evidence / "controller-bootstrap.json"
    context = d.native_controller.load_context(env={
        d.native_controller.BOOTSTRAP_PATH_ENV:str(bundle),
        d.native_controller.BOOTSTRAP_SHA_ENV:d.binding(bundle)["sha256"]})
    session = d.native_controller.ClassicControllerSession(context,ledger,
        d.classic_integration.bind_request(r,request_path),evidence,evidence/"classic-native",
        d.native_controller.CLASSIC_ACTIONS)
    try:
        execute = d.native_controller.run_classic_step(session,"execute",fixture_budget=b)
        print(json.dumps({"execute":execute,"reviewPath":str(d.EXTERNAL/"classic/execute-review.json")}),flush=True)
        if execute["returnCode"] == 0:
            review_path=d.EXTERNAL/"classic/execute-review.json"
            deadline=time.monotonic()+55
            while not review_path.exists() and time.monotonic()<deadline:
                time.sleep(.1)
            if not review_path.exists():
                raise ValueError("no exact Execute review within unchanged controller window")
            execute_review=json.loads(review_path.read_bytes())
            for action in ("apply","verify","audit","apply-replay"):
                cap=d.native_controller.run_classic_step(session,action,fixture_budget=b,
                    external_review=execute_review if action in {"apply","apply-replay"} else None)
                if action!="apply-replay" and cap["returnCode"]!=0:
                    break
    finally:
        result=d.native_controller.finalize_classic_controller(session)
        d.write_new(d.EVIDENCE/"classic/flow-result.json",result)
        print(json.dumps({"status":result["status"],"gaps":result["gaps"]}),flush=True)

if __name__=="__main__":
    main()
