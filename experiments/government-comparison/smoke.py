"""Finite fixture smoke: starts, staged inputs, adapter gaps, API and restart."""
import argparse
import importlib.util
import json
import os
from pathlib import Path
import socket
import subprocess
import sys
import time

from harness import probe
from prepare import ROOT, digest, prepare
from release import release


def load_checks(path):
    spec = importlib.util.spec_from_file_location("public_checks", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module.Checks


class Service:
    def __init__(self, repo, database, logs):
        self.repo, self.database, self.logs = Path(repo), Path(database), Path(logs)
        with socket.socket() as sock:
            sock.bind(("127.0.0.1", 0))
            self.port = sock.getsockname()[1]
        self.url = f"http://127.0.0.1:{self.port}"
        self.process = None
        self.stream = None

    def start(self):
        self.stream = self.logs.open("ab")
        env = dict(os.environ, ORDERS_DB=str(self.database), ASPNETCORE_URLS=self.url,
                   ASPNETCORE_ENVIRONMENT="Production")
        dll = self.repo / "src" / "Orders.Api" / "bin" / "Debug" / "net10.0" / "Orders.Api.dll"
        self.process = subprocess.Popen(["dotnet", str(dll)], cwd=self.repo, env=env,
                                        stdout=self.stream, stderr=subprocess.STDOUT)
        deadline = time.monotonic() + 20
        checks = load_checks(ROOT / "public" / "checks.py")(self.url)
        while time.monotonic() < deadline:
            if self.process.poll() is not None:
                self.stop()
                raise RuntimeError(f"Service exited; inspect {self.logs}")
            try:
                if checks.http("GET", "/health")[0] == 200:
                    return
            except OSError:
                pass
            time.sleep(0.1)
        self.stop()
        raise RuntimeError("Service readiness timeout")

    def stop(self):
        if self.process and self.process.poll() is None:
            self.process.terminate()
            try:
                self.process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                self.process.kill()
                self.process.wait(timeout=5)
        if self.stream:
            self.stream.close()


def smoke(destination):
    destination = Path(destination).resolve()
    manifest = prepare(destination)
    released = release(1, destination / "released-task1")
    brownfield_release = release(1, destination / "released-brownfield-task1", "brownfield")
    # Concrete negative leakage control: no future rule or unreleased cards copied.
    staged = (destination / "released-task1" / "checks.py").read_text(encoding="utf-8")
    if "new-limit" in staged or "boundary-six" in staged or "parallel-" in staged:
        raise AssertionError("Future checks leaked into task1 handoff")
    events = []
    for cell in manifest["cells"]:
        request = {"schemaVersion": 1, "operation": "probe", "mode": "fixture",
                   "trialId": f"{cell['arm']}-{cell['condition']}-smoke", **cell,
                   "actorRepository": cell["repository"], "task": None,
                   "releasedInputs": (brownfield_release if cell["condition"] == "brownfield" else released)["inputs"], "product": None,
                   "profile": {"model": None, "reasoning": None, "runnerVersion": None},
                   "limits": {"wallSeconds": 30, "maxActorCalls": 0, "maxParallelActors": 1}}
        event = probe(request, [sys.executable, str(ROOT / "fixture_adapter.py")],
                      destination / "probe" / request["trialId"])
        if event["status"] != "readiness_gap":
            raise AssertionError(event)
        events.append({"trialId": event["trialId"], "status": event["status"], "usage": event["usage"]})
    brownfield = destination / "starts" / "brownfield"
    with (destination / "build.log").open("wb") as log:
        subprocess.run(["dotnet", "restore", "src/Orders.Api/Orders.Api.csproj", "--locked-mode"],
                       cwd=brownfield, stdout=log, stderr=subprocess.STDOUT, timeout=120, check=True)
        subprocess.run(["dotnet", "build", "src/Orders.Api/Orders.Api.csproj", "--no-restore", "--nologo"],
                       cwd=brownfield, stdout=log, stderr=subprocess.STDOUT, timeout=120, check=True)
    checks_type = load_checks(destination / "released-task1" / "checks.py")
    service = Service(brownfield, destination / "fixture.db", destination / "api.log")
    try:
        service.start()
        checks = checks_type(service.url)
        checkpoint = checks.run(1, "brownfield")
        # The existing Brownfield inventory route already has this contract.
        # Greenfield receives its check only when task 2 is released.
        inventory_checks = load_checks(ROOT / "public" / "checks.py")(service.url)
        inventory_checks.unknown_inventory()
        checks.passed.extend(inventory_checks.passed)
        service.stop()
        service.start()
        checks.restart(checkpoint)
        canonical = {"items": [{"sku": "WIDGET", "quantity": 2}, {"sku": "GADGET", "quantity": 1}]}
        status, replay = checks.http("POST", "/legacy/orders", canonical, "public-initial")
        checks.expect("legacy idempotency after restart", status == 200 and replay["id"] == checkpoint["order"]["id"])
    finally:
        service.stop()
    # This checks current inherited OS rights with a harmless sentinel, not private oracle content.
    sentinel = destination / "private-access-sentinel.txt"
    sentinel.write_text("nonsecret isolation control\n", encoding="utf-8")
    read_probe = subprocess.run([sys.executable, "-c", "from pathlib import Path; import sys; print(Path(sys.argv[1]).read_text() == 'nonsecret isolation control\\n')", str(sentinel)],
                                cwd=manifest["cells"][0]["repository"], capture_output=True, text=True, check=True)
    result = {"schemaVersion": 1, "kind": "fixture-smoke", "liveTrials": 0, "status": "passed",
              "starts": manifest["starts"], "cellProbes": events, "apiChecks": checks.passed,
              "releasedInputs": released["inputs"],
              "access": {"probe": "same-user subprocess from cell cwd sees external nonsecret sentinel",
                         "externalFileReadable": read_probe.stdout.strip() == "True",
                         "osIsolationEstablished": False, "actualActorRunnerTested": False},
              "limits": ["No actual actor or product adapter invoked", "Brownfield task1/restart only; tasks2-6 not baseline capabilities",
                         "No API fault-injection or full restart guarantee established", "Private oracle stays outside handoff; same OS rights permit access"]}
    (destination / "smoke-result.json").write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--destination", required=True)
    args = parser.parse_args()
    if not Path(args.destination).is_absolute():
        parser.error("destination must be absolute")
    result = smoke(args.destination)
    print(json.dumps({"status": result["status"], "liveTrials": 0,
                      "starts": {k: v["commit"] for k, v in result["starts"].items()},
                      "adapterProbes": len(result["cellProbes"]), "apiChecks": len(result["apiChecks"]),
                      "osIsolationEstablished": False}, indent=2))
