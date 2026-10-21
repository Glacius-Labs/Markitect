"""One ordinary Agent conversation, never a Manager team; no execution during preparation.

    python -B plan_arm.py --ledger DIR --scenario ID --workspace DIR --profile JSON --output DIR

Profiles bind one image and a single-Agent native Codex JSONL or Claude print-JSON
entry point, or an explicit plan-agent/v1 adapter. Stdin is the ordinary prompt;
stdout supplies reply, session and usage. Actual CLI/turn bounds need a separately
authorized real probe before D. There is no callback, owner answer, automatic retry or helper Agent.

run_simulation is only for local process controls and can never consume a provider-mode ledger.
It records simulation even if a stub mimics a provider reply. Output files are outside the workspace.
"""
import argparse
import json
import os
from pathlib import Path, PurePosixPath
import re
import signal
import subprocess
import threading
import time

import d_series
from materialize import git, packet_files
import plan_packet
from scenarios import BY_ID

IMAGE = re.compile(r"sha256:[a-f0-9]{64}\Z")
CONTAINER = re.compile(r"[a-f0-9]{64}\Z")


def stop_process(process):
    if process.poll() is not None:
        return
    if os.name == "nt":
        subprocess.run(["taskkill", "/PID", str(process.pid), "/T", "/F"], capture_output=True, timeout=10)
    else:
        os.killpg(process.pid, signal.SIGKILL)
    process.wait(timeout=10)


def capture(command, workspace, prompt, timeout, max_output):
    """Bound memory/output and kill only our process tree; no shell, retries or inherited stdin."""
    options = {"creationflags": subprocess.CREATE_NEW_PROCESS_GROUP} if os.name == "nt" else {"start_new_session": True}
    process = subprocess.Popen(command, cwd=workspace, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                               stderr=subprocess.PIPE, **options)
    streams = [bytearray(), bytearray()]
    exceeded = threading.Event()
    lock = threading.Lock()

    def drain(pipe, destination):
        while True:
            block = pipe.read(4096)
            if not block:
                break
            with lock:
                remaining = max_output - sum(map(len, streams))
                destination.extend(block[:max(0, remaining)])
                if len(block) > remaining:
                    exceeded.set()

    threads = [threading.Thread(target=drain, args=(pipe, destination), daemon=True)
               for pipe, destination in zip((process.stdout, process.stderr), streams)]
    for thread in threads: thread.start()
    try:
        process.stdin.write(prompt.encode("utf-8")); process.stdin.close()
    except BrokenPipeError:
        pass
    deadline = time.monotonic() + timeout
    status = "completed"
    while process.poll() is None:
        if exceeded.is_set() or time.monotonic() >= deadline:
            status = "output_limit" if exceeded.is_set() else "timeout"
            stop_process(process)
            break
        time.sleep(0.02)
    for thread in threads: thread.join(timeout=2)
    if any(thread.is_alive() for thread in threads):
        raise RuntimeError("process streams remain open; completion/quiescence is unproved")
    process.stdout.close(); process.stderr.close()
    if exceeded.is_set(): status = "output_limit"
    if status == "completed" and process.returncode != 0: status = "error"
    return {"status": status, "exitCode": process.returncode,
            "stdout": streams[0].decode("utf-8", "replace"), "stderr": streams[1].decode("utf-8", "replace")}


def profile_data(path):
    p = json.loads(Path(path).read_text(encoding="utf-8"))
    protocols = {"plan-agent/v1", "codex-exec-jsonl", "claude-print-json"}
    if p.get("formatVersion") != 1 or p.get("protocol") not in protocols or p.get("mode") != "provider":
        raise ValueError("an explicit supported provider-mode profile is required")
    if p.get("provider") not in ("codex", "claude") or not p.get("model") or not p.get("effort"):
        raise ValueError("provider/model/effort must be bound")
    if not IMAGE.fullmatch(p.get("imageID", "")) or not p.get("engineExecutable"):
        raise ValueError("an immutable image id and engine are required")
    engine = Path(p["engineExecutable"])
    if not engine.is_absolute() or d_series.sha(engine) != p.get("engineSha256"):
        raise ValueError("engine executable binding differs")
    limits = p.get("limits") or {}
    for key in ("milliCPUs", "memoryBytes", "processes", "maxOutputBytes"):
        if type(limits.get(key)) is not int or limits[key] <= 0:
            raise ValueError("positive resource bound required: " + key)
    if limits["maxOutputBytes"] > 16 * 1024 * 1024:
        raise ValueError("output bound exceeds 16 MiB")
    argv = p.get("argv")
    if (not isinstance(argv, list) or not argv or not all(isinstance(a, str) for a in argv)
            or not argv[0].startswith("/") or any(value not in " ".join(argv) for value in ("{model}", "{effort}"))
            or not p.get("singleAgentTurnBoundVerified")):
        raise ValueError("bind one approved entry point and its enforced turn-limit argument")
    if p["protocol"] != "codex-exec-jsonl" and "{maxTurns}" not in " ".join(argv):
        raise ValueError("this protocol needs an enforced maxTurns argument")
    if p.get("network", "none") not in ("none", "bridge"):
        raise ValueError("ordinary provider network must be explicitly none or bridge")
    if p["protocol"] == "codex-exec-jsonl":
        expected = ["exec", "--json", "--model", "{model}", "--config", 'model_reasoning_effort="{effort}"',
                    "--disable", "multi_agent", "--disable", "apps", "--disable", "memories",
                    "--sandbox", "danger-full-access", "-"]
        normalized = ["--config" if a == "-c" else a for a in argv[1:]]
        if p["provider"] != "codex" or normalized != expected:
            raise ValueError("Codex native argv must match the closed fresh-exec template without extra overrides")
    if p["protocol"] == "claude-print-json":
        pairs = list(zip(argv, argv[1:]))
        tools = [value for flag, value in pairs if flag == "--tools"]
        allowed = {"Read", "Write", "Edit", "Glob", "Grep", "Bash"}
        expected = ["-p", "--output-format", "json", "--model", "{model}", "--effort", "{effort}",
                    "--max-turns", "{maxTurns}", "--tools", tools[0] if tools else ""]
        normalized = ["-p" if a == "--print" else a for a in argv[1:]]
        if (p["provider"] != "claude" or normalized != expected
                or len(tools) != 1 or not tools[0] or not set(tools[0].split(",")) <= allowed):
            raise ValueError("Claude requires fresh print JSON, native limits and an ordinary builtin tool set")
    return p


def create_command(profile, workspace, attempt, max_turns, operator_root=None):
    limits = profile["limits"]
    if "," in str(workspace):
        raise ValueError("commas cannot be represented safely in a bind mount source")
    command = [profile["engineExecutable"], "create", "--name", "mk-plan-" + attempt,
               "--label", "markitect.d.attempt=" + attempt, "--interactive", "--log-driver", "none",
               "--cpus", str(limits["milliCPUs"] / 1000), "--memory", str(limits["memoryBytes"]),
               "--pids-limit", str(limits["processes"]), "--cap-drop", "ALL",
               "--security-opt", "no-new-privileges", "--network", profile.get("network", "none"),
               "--workdir", "/work", "--mount", "type=bind,source=%s,target=/work" % workspace,
               "--mount", "type=bind,source=%s,target=/work/plan.md,readonly" % (Path(workspace) / "plan.md"),
               "--tmpfs", "/tmp:rw,nosuid,nodev,size=67108864", "--env", "PYTHONDONTWRITEBYTECODE=1"]
    command += ["--tmpfs", "/run/provider-auth:rw,nosuid,nodev,mode=1777,size=67108864", "--env",
                ("CODEX_HOME" if profile["provider"] == "codex" else "CLAUDE_CONFIG_DIR") + "=/run/provider-auth"]
    # Authentication files are explicit, read-only and never a source/control directory mount.
    for auth in profile.get("authFiles") or []:
        source = Path(auth["source"]).resolve()
        target = auth["target"]
        parts = target.split("/")
        if (not source.is_file() or not target.startswith("/run/provider-auth/")
                or any(part in (".", "..", "") for part in parts[1:])
                or str(PurePosixPath(target)) != target or "\\" in target
                or any(c in target + str(source) for c in (",", "\n", "\r"))):
            raise ValueError("authentication mounts must be individual files under /run/provider-auth")
        if (HERE_SOURCE in source.parents or Path(workspace) in source.parents
                or operator_root and (source == Path(operator_root) or Path(operator_root) in source.parents)):
            raise ValueError("source/control/workspace files cannot be mounted as authentication")
        command += ["--mount", "type=bind,source=%s,target=%s,readonly" % (source, auth["target"])]
    argv = [a.replace("{model}", profile["model"]).replace("{effort}", profile["effort"])
            .replace("{maxTurns}", str(max_turns)) for a in profile["argv"]]
    return command + ["--entrypoint", argv[0], profile["imageID"], *argv[1:]]


HERE_SOURCE = Path(__file__).resolve().parents[2]


def provider_reply(stdout, profile, max_turns):
    if profile["protocol"] == "codex-exec-jsonl":
        events = [json.loads(line) for line in stdout.splitlines() if line.strip()]
        sessions = [e.get("thread_id") for e in events if e.get("type") == "thread.started"]
        endings = [e for e in events if e.get("type") == "turn.completed"]
        messages = [(e.get("item") or {}).get("text") for e in events if e.get("type") == "item.completed"
                    and (e.get("item") or {}).get("type") == "agent_message"]
        if (len(sessions) != 1 or not sessions[0] or len(endings) != 1 or not messages or not messages[-1]
                or any(e.get("type") in ("error", "turn.failed") for e in events)):
            raise ValueError("no single completed Codex input Turn/session/final message")
        return {"reply": messages[-1], "turns": 1, "turnUnit": "codex-input-turn",
                "usage": endings[0].get("usage"), "sessionId": sessions[0]}
    if profile["protocol"] == "claude-print-json":
        data = json.loads(stdout)
        if (data.get("type") != "result" or data.get("subtype") != "success" or data.get("is_error")
                or not data.get("session_id") or not isinstance(data.get("result"), str) or not data["result"].strip()
                or type(data.get("num_turns")) is not int or not 0 < data["num_turns"] <= max_turns):
            raise ValueError("no successful bounded Claude print result")
        return {"reply": data["result"], "turns": data["num_turns"], "turnUnit": "claude-reported-turn",
                "usage": {**(data.get("usage") or {}), "total_cost_usd": data.get("total_cost_usd")},
                "sessionId": data["session_id"]}
    data = json.loads(stdout)
    provider = data.get("provider") or {}
    if (data.get("protocol") != "plan-agent/v1" or data.get("status") != "completed"
            or not isinstance(data.get("reply"), str) or not data["reply"].strip()
            or any(provider.get(k) != profile[k] for k in ("provider", "model", "effort"))
            or not provider.get("sessionId") or data.get("simulation") is not False
            or type(data.get("turns")) is not int or not 0 < data["turns"] <= max_turns):
        raise ValueError("missing/mismatched provider ending, session or turn-bound evidence")
    return dict(data, sessionId=provider["sessionId"], turnUnit="adapter-reported-turn")


def workspace_binding(workspace, expected):
    workspace = Path(workspace).resolve()
    if workspace != Path(expected["repo"]).resolve():
        raise ValueError("workspace differs from the exact frozen attempt path")
    if git(workspace, "rev-parse", "HEAD").strip() != expected["head"] or git(workspace, "status", "--porcelain").strip():
        raise ValueError("workspace is not the clean frozen start")
    actual = {p: d_series.sha(workspace / p) for p in packet_files(workspace)}
    if actual != expected["files"]:
        raise ValueError("workspace contents differ from the frozen packet")
    plan_packet.audit(workspace)
    return d_series.sha(workspace / "plan.md")


def run(ledger, scenario_id, workspace, profile_path, output, simulation_command=None, simulation_limit=None):
    # Reserve before profile/packet validation: even a refused/missing executable is an accountable start.
    start = d_series.reserve(ledger, scenario_id, "plan")
    began = time.monotonic(); workspace = Path(workspace).resolve(); output = Path(output).resolve()
    record = {"protocol": "plan-result/v1", "attempt": start["attempt"], "scenario": scenario_id,
              "arm": "plan", "mode": start["mode"], "status": "error", "reply": "", "rounds": 1,
              "usage": None, "turns": None, "containers": None, "limitSeconds": start["reservedSeconds"],
              "providerEvidence": "none", "workspace": str(workspace), "started": start["started"]}
    container = None; profile = None; result = None; output_created = False; expected = None; plan_hash = None
    try:
        if output == workspace or workspace in output.parents:
            raise ValueError("operator output must stay outside the writer workspace")
        output.mkdir(parents=True, exist_ok=False)
        output_created = True
        header = d_series.records(ledger)[0]
        manifest = json.loads(Path(header["manifest"]).read_text(encoding="utf-8"))
        expected = manifest["scenarios"][scenario_id]["arms"]["plan"]
        plan_hash = workspace_binding(workspace, expected)
        record.update(head=expected["head"], planSha256=plan_hash, promptSha256=d_series.hashlib.sha256(plan_packet.PROMPT.encode()).hexdigest(),
                      manifestSha256=header["manifestSha256"])
        max_turns = BY_ID[scenario_id].budget.turns
        if simulation_command is not None:
            if start["mode"] != "simulation":
                raise ValueError("simulation cannot execute on a provider ledger")
            record["providerEvidence"] = "simulation"
            result = capture(simulation_command, workspace, plan_packet.PROMPT,
                             min(start["reservedSeconds"], simulation_limit or start["reservedSeconds"]), 65536)
            record.update(status=result["status"], reply=result["stdout"])
        else:
            if start["mode"] != "provider":
                raise ValueError("provider transport cannot execute on a simulation ledger")
            profile = profile_data(profile_path)
            binding = manifest["bindings"]["plan"]
            if binding["profileSha256"] != d_series.sha(profile_path):
                raise ValueError("plan profile differs from the manifest")
            for key in ("provider", "model", "effort", "imageID"):
                if binding[key] != profile[key]: raise ValueError("manifest/profile mismatch: " + key)
            for key in ("milliCPUs", "memoryBytes", "processes"):
                if binding["resources"][key] != profile["limits"][key]: raise ValueError("compute binding differs: " + key)
            record.update(profileSha256=d_series.sha(profile_path), provider={k: profile[k] for k in ("provider", "model", "effort", "imageID")})
            engine = profile["engineExecutable"]
            observed = subprocess.run([engine, "image", "inspect", "--format", "{{.Id}}", profile["imageID"]],
                                      capture_output=True, text=True, check=True, timeout=10).stdout.strip()
            if observed != profile["imageID"]: raise ValueError("observed image id differs")
            created = subprocess.run(create_command(profile, workspace, start["attempt"], max_turns,
                                      Path(header["manifest"]).parent), capture_output=True,
                                     text=True, check=True, timeout=10).stdout.strip()
            if not CONTAINER.fullmatch(created): raise ValueError("engine returned no exact container id")
            container = created
            record["containers"] = {"created": 1, "id": container, "removed": False}
            remaining = start["reservedSeconds"] - (time.monotonic() - began) - 5
            if remaining <= 0: raise ValueError("startup consumed the attempt's remaining time")
            result = capture([engine, "start", "--attach", "--interactive", container], workspace, plan_packet.PROMPT,
                             remaining, profile["limits"]["maxOutputBytes"])
            record["status"] = result["status"]
            if result["status"] == "completed":
                data = provider_reply(result["stdout"], profile, max_turns)
                record.update(reply=data["reply"], usage=data.get("usage"), turns=data["turns"],
                              turnUnit=data["turnUnit"], sessionId=data["sessionId"], providerEvidence="transport-reported")
    except Exception as error:
        record.update(status="error", error=str(error))
    finally:
        if result and output_created:
            (output / "stdout.txt").write_text(result["stdout"], encoding="utf-8")
            (output / "stderr.txt").write_text(result["stderr"], encoding="utf-8")
        if container:
            try:
                # Exact id came from our create; label guards cleanup against foreign resources.
                label = subprocess.run([profile["engineExecutable"], "inspect", "--format",
                    '{{index .Config.Labels "markitect.d.attempt"}}', container], capture_output=True, text=True,
                    check=True, timeout=5).stdout.strip()
                if label != start["attempt"]: raise ValueError("container ownership label differs")
                state = json.loads(subprocess.run([profile["engineExecutable"], "inspect", "--format", "{{json .State}}", container],
                    capture_output=True, text=True, check=True, timeout=5).stdout)
                record["containers"]["stateBeforeRemoval"] = state
                record["containers"]["observedStarted"] = bool(state.get("StartedAt") and not state["StartedAt"].startswith("0001"))
                subprocess.run([profile["engineExecutable"], "rm", "--force", container], capture_output=True, check=True, timeout=5)
                record["containers"]["removed"] = True
            except Exception as error:
                record.update(status="error", cleanupError=str(error))
        if expected and plan_hash:
            try:
                if git(workspace, "rev-parse", "HEAD").strip() != expected["head"] or d_series.sha(workspace / "plan.md") != plan_hash:
                    raise ValueError("Agent changed the frozen HEAD or authoritative plan.md")
                # Observe the closing workspace after the owned container is removed, never while it can write.
                if not container or record["containers"]["removed"]:
                    record["resultFiles"] = {p: d_series.sha(workspace / p) for p in packet_files(workspace)}
            except Exception as error:
                record.update(status="error", error=str(error))
        record["elapsedSeconds"] = round(time.monotonic() - began, 3)
        # Reuse of an output path is a failed start; retain its existing files and place this ending in the ledger.
        final = output / "final.json" if output_created else Path(ledger) / (start["attempt"] + ".json")
        d_series.write_new(final, record)
        d_series.finish(ledger, start["attempt"], final)
    return record


def run_simulation(ledger, scenario, workspace, command, output, limit=None):
    return run(ledger, scenario, workspace, None, output, command, limit)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for key in ("ledger", "scenario", "workspace", "profile", "output"): parser.add_argument("--" + key, required=True)
    args = parser.parse_args()
    result = run(args.ledger, args.scenario, args.workspace, args.profile, args.output)
    print(json.dumps({k: result[k] for k in ("attempt", "mode", "status", "elapsedSeconds", "providerEvidence")}))
    return 0 if result["status"] == "completed" else 1


if __name__ == "__main__":
    raise SystemExit(main())
