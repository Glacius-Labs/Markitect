"""The egress gateway of the evaluation sandbox: the sandbox network's only route out.

  python3 -I -B egress.py --allow HOST [--allow HOST ...] [--port 443]

The sandbox runs on an internal network that has no route out. The suite maps
each allowed host name to this gateway's address in the sandbox's /etc/hosts, so
every tool connects here on port 443 by itself, without a proxy setting. The
gateway reads the TLS ClientHello and, only when its server name is allowed,
forwards the connection unchanged to that host; anything else is closed. TLS
stays end to end: the gateway sees the server name, never the content. It
writes one JSON line per connection to standard output.
"""
import argparse
import json
import socket
import struct
import sys
import threading
import time

RECORD_LIMIT = 5 + 16384
HELLO_SECONDS = 10
CONNECT_SECONDS = 15
CHUNK = 65536
DRAIN_SECONDS = 30
CONNECTIONS = 64  # two threads each, well inside the gateway's process limit

_print = threading.Lock()
slots = threading.BoundedSemaphore(CONNECTIONS)


def log(entry):
    with _print:
        sys.stdout.write(json.dumps(entry) + "\n")
        sys.stdout.flush()


def server_name(hello):
    """The server name of a TLS ClientHello record, lower case, or None."""
    try:
        if len(hello) < 5 or hello[0] != 0x16:
            return None
        body = hello[5:5 + struct.unpack(">H", hello[3:5])[0]]
        if body[0] != 0x01:
            return None
        p = 4 + 2 + 32  # handshake header, client version, random
        p += 1 + body[p]  # session id
        p += 2 + struct.unpack(">H", body[p:p + 2])[0]  # cipher suites
        p += 1 + body[p]  # compression methods
        end = p + 2 + struct.unpack(">H", body[p:p + 2])[0]
        p += 2
        while p + 4 <= end:
            kind, size = struct.unpack(">HH", body[p:p + 4])
            p += 4
            if kind == 0:  # server_name: a list of (type, name)
                q = p + 2
                while q + 3 <= p + size:
                    name_type, length = body[q], struct.unpack(">H", body[q + 1:q + 3])[0]
                    if name_type == 0:
                        return body[q + 3:q + 3 + length].decode("ascii").lower()
                    q += 3 + length
                return None
            p += size
    except (IndexError, struct.error, UnicodeDecodeError):
        return None
    return None


def read_hello(conn):
    """The first TLS record, read whole: the ClientHello."""
    data = b""
    while len(data) < 5 or len(data) < 5 + struct.unpack(">H", data[3:5])[0]:
        chunk = conn.recv(RECORD_LIMIT - len(data))
        if not chunk:
            break
        data += chunk
        if len(data) >= RECORD_LIMIT:
            break
    return data


def pipe(src, dst, entry, key):
    try:
        while True:
            data = src.recv(CHUNK)
            if not data:
                dst.shutdown(socket.SHUT_WR)
                return
            dst.sendall(data)
            entry[key] += len(data)
    except OSError:
        # One side failed: close both, so that the other direction ends too.
        for s in (src, dst):
            try:
                s.shutdown(socket.SHUT_RDWR)
            except OSError:
                pass


def handle(client, peer, allowed, upstream_port):
    started = time.monotonic()
    entry = {"peer": peer[0], "sni": None, "allowed": False, "bytesUp": 0, "bytesDown": 0}
    upstream = None
    try:
        client.settimeout(HELLO_SECONDS)
        hello = read_hello(client)
        entry["sni"] = name = server_name(hello)
        if name not in allowed:
            return
        entry["allowed"] = True
        upstream = socket.create_connection((name, upstream_port), timeout=CONNECT_SECONDS)
        upstream.sendall(hello)
        entry["bytesUp"] = len(hello)
        client.settimeout(None)
        upstream.settimeout(None)
        up = threading.Thread(target=pipe, args=(client, upstream, entry, "bytesUp"), daemon=True)
        up.start()
        pipe(upstream, client, entry, "bytesDown")
        # The provider has finished; a client that never closes its side must not hold the connection open.
        up.join(DRAIN_SECONDS)
    except (OSError, RuntimeError) as e:
        entry["error"] = f"{type(e).__name__}: {e}"
    finally:
        for s in (client, upstream):
            if s is not None:
                try:
                    s.shutdown(socket.SHUT_RDWR)
                except OSError:
                    pass
                s.close()
        entry["seconds"] = round(time.monotonic() - started, 3)
        log(entry)
        slots.release()


def serve(allowed, port, upstream_port=443, ready=None):
    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    server.bind(("0.0.0.0", port))
    server.listen(128)
    log({"ready": True, "port": server.getsockname()[1], "allow": sorted(allowed)})
    if ready:
        ready(server)
    while True:
        try:
            client, peer = server.accept()
        except OSError:
            return
        # Beyond the cap, or when no thread can start, the connection is refused and the gateway lives on.
        if not slots.acquire(blocking=False):
            log({"peer": peer[0], "sni": None, "allowed": False, "error": "connection limit"})
            client.close()
            continue
        try:
            threading.Thread(target=handle, args=(client, peer, allowed, upstream_port), daemon=True).start()
        except RuntimeError as e:
            log({"peer": peer[0], "sni": None, "allowed": False, "error": f"RuntimeError: {e}"})
            client.close()
            slots.release()


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--allow", action="append", required=True)
    parser.add_argument("--port", type=int, default=443)
    a = parser.parse_args(argv)
    serve({h.lower() for h in a.allow}, a.port)


if __name__ == "__main__":
    main()
