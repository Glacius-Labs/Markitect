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
  EVALUATION_LABEL      KEY=VALUE, the run's ID, for the variant's container labels or names
  EVALUATION_ENGINE     the engine command, as a JSON list

While the variant runs, the suite records the engine's events, and inspects
every network created during the run as soon as its event arrives. Afterwards
every container started during the run, other than the suite's own, must have
joined only the internal network, internal networks the variant created itself
during the run (such as a Run's own network between its remote and its agents),
or no network; one that joined any other network is an escape, and the run is
not valid for counting. Networks are told apart by ID; a network that could not
be inspected counts as not internal, and a reader of the record that stopped
fails the check. An internal network reaches no other network; unless its
bridge's gateway mode is isolated, as the suite's own is, its containers can
reach the engine's host at the bridge address, which the record shows
(`armNetworks`). Containers, networks and volumes the variant left are residue:
removed and counted; so is what the variant's host reports it left (run.json
`leftovers`), such as a Run branch or remote its cleanup could not remove. The engine lock keeps other runners' containers out of the
record.

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
import threading
import time
import uuid

import sandbox as sandboxing

EVENTS_SECONDS = 10
# The event record starts this early, so that an engine clock behind the host's misses nothing; account() skips what
# came before the run's own network.
EVENTS_MARGIN = 300
ALLOWED_NONE = "none"


def reported_leftovers(out):
    """The leftovers the variant reports in OUT/run.json (`leftovers`, a list of texts); none when it reports none."""
    path = pathlib.Path(out) / "run.json"
    if path.is_symlink() or not path.is_file():
        return []
    try:
        value = json.loads(path.read_text(encoding="utf-8")).get("leftovers")
    except (OSError, ValueError, AttributeError, RecursionError):
        return []
    return [str(x)[:300] for x in value[:20]] if isinstance(value, list) else []


class Events:
    """The engine's event record of one run, written to RUN_DIR/engine-events.jsonl as it comes.

    Every network created after the run's own internal network is inspected as soon as its event arrives, while the
    variant still uses it: `inspected[ID]` is True when it is internal, False when it is not and None when it could
    not be inspected. `gateways[ID]` keeps its IPv4 gateway mode for the record: only an isolated bridge carries no
    address of the engine's host, as the suite's own internal network does (sandbox.open_egress)."""

    def __init__(self, argv, engine, since, internal, run_dir):
        self.engine, self.internal, self.inspected, self.gateways, self.error = engine, internal, {}, {}, None
        self.file, self.err = open(run_dir / "engine-events.jsonl", "wb"), open(run_dir / "engine-events.stderr.log", "wb")
        self.proc = subprocess.Popen([*argv, "events", "--since", since, "--format", "{{json .}}"], stdout=subprocess.PIPE,
                                     stderr=self.err, stdin=subprocess.DEVNULL)
        self.thread = threading.Thread(target=self.read, daemon=True)
        self.thread.start()

    def read(self):
        begun = False
        try:
            for line in self.proc.stdout:
                self.file.write(line)
                self.file.flush()
                try:
                    event = json.loads(line)
                except ValueError:
                    continue
                if not isinstance(event, dict) or event.get("Type") != "network" or event.get("Action") != "create":
                    continue
                actor = event.get("Actor") or {}
                if not begun:
                    begun = (actor.get("Attributes") or {}).get("name") == self.internal
                    continue
                ident = actor.get("ID") or event.get("id")
                if not ident:
                    continue  # nothing to inspect; account() finds no ID for it either
                code, out = self.engine("network-internal", ["network", "inspect", "--format", "{{json .Internal}} {{json .Options}}", ident])
                internal, _, options = out.strip().partition(" ")
                self.inspected[ident] = None if code != 0 else internal == "true"
                try:
                    self.gateways[ident] = (json.loads(options) or {}).get("com.docker.network.bridge.gateway_mode_ipv4") if code == 0 else None
                except (ValueError, AttributeError):
                    self.gateways[ident] = None
        except Exception as e:  # noqa: BLE001 - a reader that stops must fail the check, not end it silently
            self.error = f"{type(e).__name__}: {e}"

    def stop(self):
        """Ends the record; the error that makes the network check incomplete, if any."""
        self.proc.terminate()
        try:
            self.proc.wait(timeout=EVENTS_SECONDS)
        except subprocess.TimeoutExpired:
            self.proc.kill()
        self.thread.join(timeout=EVENTS_SECONDS)
        if self.thread.is_alive():
            self.error = self.error or "the event reader did not end"
        else:
            self.proc.stdout.close()
        self.file.close()
        self.err.close()
        return self.error and f"the engine's event record stopped being read ({self.error}): the network check is incomplete"


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
                since = str(int(time.time()) - EVENTS_MARGIN)
                gateway_ip = sandboxing.open_egress(engine, run_id, label, self.image, self.hosts, networks, containers)
                record["gatewayAddress"] = gateway_ip
                gateway_id = engine.must("gateway-id", ["inspect", "--format", "{{.Id}}", f"{run_id}-gateway"]).strip()
                # From before the network stood up, so that nothing the variant starts is missed.
                events = Events(self.engine, engine, since, internal, run_dir)
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
                    if events.proc.poll() is not None:
                        record.setdefault("error", f"the engine's event record ended early (exit {events.proc.returncode}): "
                                                   "the network check is incomplete")
                    stopped = events.stop()
                    if stopped:
                        record.setdefault("error", stopped)
                    self.account(engine, gateway_id, internal, run_dir / "engine-events.jsonl", record, timing["timedOut"], events.inspected,
                                 events.gateways)
                sandboxing.close_egress(engine, run_id, label, run_dir, record, networks, containers)
                record["residue"] = record["residue"] + record.pop("armResidue", [])
                login_file.unlink(missing_ok=True)
        present = self.present()
        if present:
            record["hiddenPresent"] = present
        # What the variant's host says it left (run.json `leftovers`), such as a Run's remote or branch its cleanup could
        # not remove, is residue like what the engine shows: the run is not valid for counting.
        # A text that holds the login never reaches the record, which is written after the quarantine's search.
        left = [x for x in reported_leftovers(out) if not any(s in x.encode() for s in secrets)]
        if left:
            record["reportedLeftovers"] = left
            record["residue"] = record.get("residue", []) + ["reported by the variant: " + x for x in left]
        sandboxing.quarantine(record, secrets, run_dir, state)
        return timing, record

    def account(self, engine, gateway_id, internal, events_path, record, timed_out, inspected=None, gateways=None):
        """Every container the variant started and the networks it joined; remove what it left.

        What it left after a normal end is residue: containers, networks and volumes; at the deadline the suite
        stops it, as it stops the sandbox. A record that never shows the run's own network failed: the check fails
        closed. `inspected` says, per network ID, whether a network created during the run was internal when it was
        inspected. Networks are told apart by ID, so a name reused by another network does not inherit the first's
        standing."""
        inspected, gateways = inspected or {}, gateways or {}
        started, joined, created_networks, volumes, begun, internal_id = {}, {}, {}, [], False, None
        for line in events_path.read_text(encoding="utf-8", errors="replace").splitlines():
            try:
                event = json.loads(line)
            except ValueError:
                continue
            if not isinstance(event, dict):
                continue
            attrs = (event.get("Actor") or {}).get("Attributes") or {}
            kind, action = event.get("Type"), event.get("Action")
            # Current engines name the object in Actor.ID only; the top-level id is a deprecated copy.
            ident = (event.get("Actor") or {}).get("ID") or event.get("id")
            # The record starts early, so that nothing is missed; what came before this run's network is not its.
            if not begun:
                begun = kind == "network" and action == "create" and attrs.get("name") == internal
                internal_id = ident if begun else None
                continue
            if kind == "container" and action == "start":
                if ident == gateway_id:
                    continue  # the suite's own gateway, by its ID: a label the variant could copy
                started[ident] = attrs.get("name")
            elif kind == "network" and action == "connect":
                joined.setdefault(attrs.get("container"), set()).add((ident, attrs.get("name")))
            elif kind == "network" and action == "create" and attrs.get("name") != internal.removesuffix("-internal") + "-egress":
                created_networks[ident] = attrs.get("name")
            elif kind == "volume" and action == "create" and ident:
                volumes.append(ident)
        if not begun:
            record.setdefault("error", "the engine's event record does not show the run's network: the network check did not run")
        elif not internal_id:
            record.setdefault("error", "the engine's event record does not name the run's network by ID: the network check did not run")
        allowed = ({internal_id} | {nid for nid in created_networks if inspected.get(nid) is True}) - {None}
        record["armContainers"], record["escapes"] = [], []
        for cid, name in started.items():
            nets = joined.get(cid, set())
            entry = {"id": cid[:12], "name": name, "networks": sorted(n or "?" for _, n in nets)}
            record["armContainers"].append(entry)
            if not nets or any(nid not in allowed and n != ALLOWED_NONE for nid, n in nets):
                record["escapes"].append(entry)
        record["armNetworks"] = [{"id": nid[:12], "name": name, "internal": inspected.get(nid), "gatewayMode": gateways.get(nid)}
                                 for nid, name in created_networks.items()]
        left = []
        for cid, name in started.items():
            if engine("arm-inspect", ["inspect", "--format", "{{.Id}}", cid])[0] == 0:
                left.append(name or cid[:12])
                engine("arm-remove", ["rm", "-f", cid], sandboxing.KILL_SECONDS)
        for nid, name in created_networks.items():
            if engine("arm-network-inspect", ["network", "inspect", "--format", "{{.Id}}", nid])[0] == 0:
                left.append(name or nid[:12])
                engine("arm-network-remove", ["network", "rm", nid], sandboxing.KILL_SECONDS)
        for vid in volumes:
            if engine("arm-volume-inspect", ["volume", "inspect", "--format", "{{.Name}}", vid])[0] == 0:
                left.append("volume " + vid)
                engine("arm-volume-remove", ["volume", "rm", "-f", vid], sandboxing.KILL_SECONDS)
        record["armStoppedAtDeadline" if timed_out else "armResidue"] = left
