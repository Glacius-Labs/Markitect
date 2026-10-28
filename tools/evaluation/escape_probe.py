"""The escape probe: a variant without a model that tries to leave the sandbox.

  python3 -I -B escape_probe.py [--endpoint HOST] TASK_DIR OUT_DIR [STATE_DIR]

Run it through `suite.py run --sandbox`. From inside the sandbox it tries three
escapes, each of which must fail:

- **host files:** a mount that is not one of the suite's, a known host mount
  path, or a repository marker (a .git directory, AGENTS.md, an oracle
  directory) anywhere outside the suite's mounts;
- **GitHub and the web:** resolving github.com or example.com at all, reaching
  them by name, GitHub and public resolvers by address, and GitHub through the
  egress gateway with GitHub's server name;
- **engine and host:** a container engine socket file, DOCKER_HOST, the
  engine's TCP ports on the gateway, the host names and the first address of
  every attached subnet (where an engine puts its host side), and common
  service ports there.

It also opens a TLS handshake with the allowed provider endpoint (no request is
sent), so that blocked escapes are not just a missing network. It writes every
attempt to OUT_DIR/run.json, delivers no result and exits 1 if anything escaped.
"""
import argparse
import json
import os
import pathlib
import socket
import ssl
import sys

SECONDS = 5
SUITE_MOUNTS = ("/task", "/out", "/state", "/variant", "/login", "/tmp")
# The engine's own, in every container; --init mounts its init binary at either path, depending on the engine.
ENGINE_FILES = ("/etc/hosts", "/etc/hostname", "/etc/resolv.conf", "/sbin/docker-init", "/usr/sbin/docker-init")
SYSTEM = ("/proc", "/sys", "/dev")
HOST_PATHS = ("/mnt", "/host_mnt", "/hostfs", "/run/desktop/mnt/host", "/Users", "/c")
MARKER_DIRS = {".git", "library-oracle", "makerspace-oracle", "Markitect-Evaluation"}
MARKER_FILES = {"AGENTS.md"}
GITHUB_ADDRESSES = ("140.82.112.3", "140.82.121.4")
WEB_ADDRESSES = (("1.1.1.1", 443), ("8.8.8.8", 53), ("8.8.8.8", 443))
HOST_PORTS = (22, 53, 80, 443, 3000, 5432, 8080)
ENGINE_SOCKETS = ("/var/run/docker.sock", "/run/docker.sock", "/var/run/docker.sock.raw", "/run/desktop/docker.sock",
                  "/run/podman/podman.sock", "/run/containerd/containerd.sock")
ENGINE_PORTS = (2375, 2376)
ENGINE_HOSTS = ("host.docker.internal", "gateway.docker.internal", "172.17.0.1")


def within(path, roots):
    return any(path == r or path.startswith(r.rstrip("/") + "/") for r in roots)


def foreign_mounts(mountinfo):
    """Mount points from /proc/self/mountinfo that are neither the suite's, the engine's nor the system's."""
    found = []
    for line in mountinfo.splitlines():
        fields = line.split()
        if len(fields) < 5:
            continue
        point = fields[4].replace("\\040", " ")
        if point == "/" or within(point, SUITE_MOUNTS + SYSTEM) or point in ENGINE_FILES:
            continue
        found.append(point)
    return found


def markers(root, skip):
    """Repository markers below root, outside the skipped paths, without following links."""
    found = []
    for top, dirs, files in os.walk(root, onerror=lambda e: None):
        dirs[:] = [d for d in dirs if not within(os.path.join(top, d), skip)]
        found += [os.path.join(top, d) for d in dirs if d in MARKER_DIRS]
        found += [os.path.join(top, f) for f in files if f in MARKER_FILES]
    return sorted(found)


def tcp(host, port, seconds=SECONDS):
    target = f"{host}:{port}"
    try:
        with socket.create_connection((host, port), timeout=seconds):
            return {"target": target, "reached": True, "detail": "connected"}
    except OSError as e:
        return {"target": target, "reached": False, "detail": f"{type(e).__name__}: {e}"}


def tls(host, port, server_name, seconds=SECONDS, context=None):
    """A TLS handshake with server_name; reached only when it completes. No request is sent."""
    target = f"{host}:{port} as {server_name}"
    context = context or ssl.create_default_context()
    try:
        with socket.create_connection((host, port), timeout=seconds) as raw:
            with context.wrap_socket(raw, server_hostname=server_name) as conn:
                return {"target": target, "reached": True, "detail": conn.version()}
    except (OSError, ssl.SSLError) as e:
        return {"target": target, "reached": False, "detail": f"{type(e).__name__}: {e}"}


def unix_socket(path):
    if not os.path.exists(path):
        return {"target": path, "reached": False, "detail": "absent"}
    try:
        with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as s:
            s.settimeout(SECONDS)
            s.connect(path)
            return {"target": path, "reached": True, "detail": "connected"}
    except OSError as e:
        return {"target": path, "reached": True, "detail": f"present, {type(e).__name__}: {e}"}


def resolve(name):
    try:
        return sorted({a[4][0] for a in socket.getaddrinfo(name, 443, socket.AF_INET, socket.SOCK_STREAM)})
    except OSError:
        return []


def route_hosts(table):
    """From /proc/net/route: the default gateway and the first address of every directly attached subnet,
    where an engine puts its bridge's host-side address."""
    hosts = []
    for line in table.splitlines()[1:]:
        fields = line.split()
        try:
            destination, gateway, mask = (int(fields[i], 16).to_bytes(4, "little") for i in (1, 2, 7))
        except (IndexError, ValueError):
            continue
        if destination == bytes(4) and gateway != bytes(4):
            hosts.append(socket.inet_ntoa(gateway))
        elif destination != bytes(4) and gateway == bytes(4) and mask != b"\xff" * 4:
            hosts.append(socket.inet_ntoa((int.from_bytes(destination, "big") + 1).to_bytes(4, "big")))
    return hosts


def local_hosts():
    try:
        return route_hosts(pathlib.Path("/proc/net/route").read_text())
    except OSError:
        return []


def host_files():
    try:
        mounts = foreign_mounts(pathlib.Path("/proc/self/mountinfo").read_text())
    except OSError as e:
        mounts = [f"unreadable mountinfo: {e}"]
    paths = [p for p in HOST_PATHS if os.path.exists(p) and (not os.path.isdir(p) or os.listdir(p))]
    found = markers("/", SUITE_MOUNTS + SYSTEM)
    attempts = ([{"target": m, "reached": True, "detail": "foreign mount"} for m in mounts]
                + [{"target": p, "reached": True, "detail": "host path present"} for p in paths]
                + [{"target": f, "reached": True, "detail": "repository marker"} for f in found])
    return attempts or [{"target": "/", "reached": False, "detail": "only the suite's mounts; no host path or repository marker"}]


def github(gateway):
    attempts = []
    for name in ("github.com", "example.com"):
        names = resolve(name)
        # A name that resolves is a channel out through the engine's resolver, even if no address answers.
        attempts.append({"target": f"dns {name}", "reached": bool(names),
                         "detail": "resolves to " + ", ".join(names) if names else "does not resolve"})
        attempts += [tcp(a, 443) for a in names[:2]]
    attempts += [tcp(a, 443) for a in GITHUB_ADDRESSES] + [tcp(a, p) for a, p in WEB_ADDRESSES]
    if gateway:
        attempts.append(tls(gateway, 443, "github.com"))
        attempts.append(tcp(gateway, 80))
    return attempts


def engine(gateway):
    attempts = [unix_socket(p) for p in ENGINE_SOCKETS]
    if os.environ.get("DOCKER_HOST"):
        attempts.append({"target": "DOCKER_HOST", "reached": True, "detail": "set"})
    hosts = [h for h in (gateway, *local_hosts(), *ENGINE_HOSTS) if h]
    attempts += [tcp(h, p, 2) for h in dict.fromkeys(hosts) for p in ENGINE_PORTS]
    # The host side of the bridge: any service the engine's host offers there would be a way out.
    attempts += [tcp(h, p, 2) for h in dict.fromkeys(local_hosts()) if h != gateway for p in HOST_PORTS]
    return attempts


def probe(endpoint):
    gateway = (resolve(endpoint) or [None])[0]
    checks = {"hostFiles": host_files(), "github": github(gateway), "engine": engine(gateway)}
    result = {name: {"blocked": not any(a["reached"] for a in attempts if not a.get("informational")), "attempts": attempts}
              for name, attempts in checks.items()}
    result["allowedEndpoint"] = tls(endpoint, 443, endpoint)
    result["allowedEndpoint"]["gateway"] = gateway
    return result


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--endpoint", default="api.anthropic.com", help="the provider endpoint the egress gateway allows")
    parser.add_argument("task")
    parser.add_argument("out")
    parser.add_argument("state", nargs="?")
    a = parser.parse_args(argv)
    out = pathlib.Path(a.out)
    out.mkdir(parents=True, exist_ok=True)
    result = probe(a.endpoint)
    contained = all(result[k]["blocked"] for k in ("hostFiles", "github", "engine"))
    record = {"provider": None, "model": None, "status": "contained" if contained else "escaped",
              "egressWorks": result["allowedEndpoint"]["reached"], "probe": result}
    (out / "run.json").write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({k: record[k] for k in ("status", "egressWorks")}))
    return 0 if contained else 1


if __name__ == "__main__":
    sys.exit(main())
