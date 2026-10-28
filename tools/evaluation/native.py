"""The native arm mode: the variant's host runs natively on the machine, its own containers on the suite's network.

`suite.py run --native IMAGE_ID` and `chain --native IMAGE_ID` are for a
variant whose host is trusted, reviewed code that starts its agents only in
containers of its own, such as the concept 2.0 arm. The suite stands up the
same egress-limited network as for the sandbox (sandbox.open_egress): an
internal network without a route out and the egress gateway, which forwards
only the provider's endpoints. It then starts the variant command as a host
process with, besides EVALUATION_MODEL:

  EVALUATION_LOGIN      an access-only copy of the provider login, deleted after the run
  EVALUATION_NETWORK    the internal network every container of the variant joins
  EVALUATION_HOSTS      HOST:ADDRESS pairs, comma-separated, for --add-host: the provider endpoints at the gateway
  EVALUATION_DNS        the resolver address that resolves nothing (--dns)
  EVALUATION_LABEL      KEY=VALUE, a label the variant puts on its containers
  EVALUATION_ENGINE     the engine command, as a JSON list

While the variant runs, the suite records the engine's events. Afterwards every
container started during the run, other than the suite's own, must have joined
the internal network or no network, and nothing else; one that joined another
network is an escape, and the run is not valid for counting. Containers and
networks the variant left are residue: removed and counted. The engine lock
keeps other runners' containers out of the record.

Hidden material stays off the machine during runs: the series names where it
would lie (`--hidden-absent PATH`), and the run refuses to start, and is not
valid, while any of them exists. The variant's host itself has the machine's
files and network: that is why only trusted variant code runs natively.
"""
import json
import os
import pathlib
import subprocess
import tempfile
import time
import uuid

import sandbox as sandboxing

EVENTS_SECONDS = 10
ALLOWED_NONE = "none"


class Native:
    """The settings of every native run of one series."""

    conditions = "native"

    def __init__(self, engine, image, series, login, provider, hidden=()):
        if not sandboxing.IMAGE_ID.fullmatch(image):
            raise sandboxing.SandboxError(f"--native must name the sandbox image by its engine image ID (sha256:...), not {image!r}")
        if provider not in series["egress"]:
            raise sandboxing.SandboxError(f"the series allows no egress for {provider}")
        self.engine, self.image, self.series, self.provider = list(engine), image, series, provider
        self.hosts = list(series["egress"][provider])
        self.login = pathlib.Path(login)
        self.hidden = [str(pathlib.Path(p).resolve()) for p in hidden]

    def describe(self):
        return {"mode": "native", "image": self.image, "seriesSha256": self.series["sha256"], "egressAllow": self.hosts,
                "hiddenAbsent": self.hidden}

    def valid(self, record):
        """The network stood up, nothing is left, no container left the network and no hidden material was here."""
        return ("error" not in record and "variantEnv" in record and not record["residue"] and not record.get("escapes")
                and not record.get("hiddenPresent"))

    def present(self):
        return [p for p in self.hidden if os.path.lexists(p)]

    def run(self, task, out, state, variant, model, deadline, run_dir, launch):
        """Run the variant command natively; launch(argv, on_deadline, env) runs it under the deadline."""
        run_id = "eval-" + uuid.uuid4().hex[:12]
        label = f"{sandboxing.LABEL_KEY}={run_id}"
        internal = f"{run_id}-internal"
        engine = sandboxing.Engine(self.engine)
        record = {**self.describe(), "runId": run_id, "network": internal, "steps": engine.steps}
        timing = {"exitCode": None, "timedOut": False, "wallSeconds": 0.0}
        networks, containers, secrets, events = [], [], [], None
        with tempfile.TemporaryDirectory() as tmp:
            login_file = pathlib.Path(tmp) / "credentials.json"
            try:
                present = self.present()
                if present:
                    raise sandboxing.SandboxError("hidden material is on the machine: " + ", ".join(present))
                login, secrets = sandboxing.access_login(self.provider, self.login.read_bytes(), deadline)
                login_file.write_bytes(login)
                os.chmod(login_file, 0o600)
                del login
                if engine.must("image", ["image", "inspect", "--format", "{{.Id}}", self.image]).strip() != self.image:
                    raise sandboxing.SandboxError("the image ID differs")
                since = str(int(time.time()) - 1)
                gateway_ip = sandboxing.open_egress(engine, run_id, label, self.image, self.hosts, networks, containers)
                record["gatewayAddress"] = gateway_ip
                events_file = open(run_dir / "engine-events.jsonl", "wb")
                # From before the network stood up, so that nothing the variant starts is missed.
                events = subprocess.Popen([*self.engine, "events", "--since", since, "--format", "{{json .}}"], stdout=events_file,
                                          stderr=subprocess.DEVNULL, stdin=subprocess.DEVNULL)
                env = dict(os.environ, EVALUATION_MODEL=model, EVALUATION_LOGIN=str(login_file), EVALUATION_NETWORK=internal,
                           EVALUATION_HOSTS=",".join(f"{h}:{gateway_ip}" for h in self.hosts), EVALUATION_DNS=sandboxing.NO_DNS,
                           EVALUATION_LABEL=f"{sandboxing.LABEL_KEY}.arm={run_id}", EVALUATION_ENGINE=json.dumps(self.engine))
                record["variantEnv"] = {k: v for k, v in env.items() if k.startswith("EVALUATION_") and k != "EVALUATION_LOGIN"}
                argv = [*variant, str(task), str(out)] + ([str(state)] if state else [])
                record["argv"] = argv
                timing = launch(argv, None, env)
                code, gw = engine("gateway-state", ["inspect", "--format", "{{json .State.Running}}", f"{run_id}-gateway"])
                if code != 0 or gw.strip() != "true":
                    raise sandboxing.SandboxError("the egress gateway stopped during the run")
            except Exception as e:  # noqa: BLE001 - whatever failed, the run is recorded and cleaned up
                record["error"] = f"{type(e).__name__}: {e}"
            finally:
                if events:
                    time.sleep(1)  # the engine reports the last events with a short delay
                    events.terminate()
                    try:
                        events.wait(timeout=EVENTS_SECONDS)
                    except subprocess.TimeoutExpired:
                        events.kill()
                    events_file.close()
                    self.account(engine, run_id, internal, run_dir / "engine-events.jsonl", record, timing["timedOut"])
                sandboxing.close_egress(engine, run_id, label, run_dir, record, networks, containers)
                record["residue"] = record["residue"] + record.pop("armResidue", [])
                login_file.unlink(missing_ok=True)
        present = self.present()
        if present:
            record["hiddenPresent"] = present
        sandboxing.quarantine(record, secrets, run_dir, state)
        return timing, record

    def account(self, engine, run_id, internal, events_path, record, timed_out):
        """Every container the variant started and the networks it joined; remove what it left.

        What it left after a normal end is residue; at the deadline the suite stops it, as it stops the sandbox."""
        started, joined, created_networks, begun = {}, {}, {}, False
        for line in events_path.read_text(encoding="utf-8", errors="replace").splitlines():
            try:
                event = json.loads(line)
            except ValueError:
                continue
            attrs = (event.get("Actor") or {}).get("Attributes") or {}
            kind, action = event.get("Type"), event.get("Action")
            # The record starts a second early, so that nothing is missed; what came before this run's network is not its.
            if not begun:
                begun = kind == "network" and action == "create" and attrs.get("name") == internal
                continue
            if kind == "container" and action == "start":
                if attrs.get(sandboxing.LABEL_KEY) == run_id:
                    continue  # the suite's own gateway
                started[event.get("id") or (event.get("Actor") or {}).get("ID")] = attrs.get("name")
            elif kind == "network" and action == "connect":
                joined.setdefault(attrs.get("container"), set()).add(attrs.get("name"))
            elif kind == "network" and action == "create" and not attrs.get("name", "").startswith(run_id):
                created_networks[(event.get("Actor") or {}).get("ID")] = attrs.get("name")
        record["armContainers"] = [{"id": cid[:12], "name": name, "networks": sorted(joined.get(cid, []))} for cid, name in started.items()]
        record["escapes"] = [c for c in record["armContainers"] if not c["networks"] or set(c["networks"]) - {internal, ALLOWED_NONE}]
        left = []
        for cid, name in started.items():
            if engine("arm-inspect", ["inspect", "--format", "{{.Id}}", cid])[0] == 0:
                left.append(name or cid[:12])
                engine("arm-remove", ["rm", "-f", cid], sandboxing.KILL_SECONDS)
        for nid, name in created_networks.items():
            if engine("arm-network-inspect", ["network", "inspect", "--format", "{{.Id}}", nid])[0] == 0:
                left.append(name or nid[:12])
                engine("arm-network-remove", ["network", "rm", nid], sandboxing.KILL_SECONDS)
        record["armStoppedAtDeadline" if timed_out else "armResidue"] = left
