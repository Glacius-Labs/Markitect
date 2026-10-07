"""Read-only app-server RPC client: no thread, turn, login or token endpoints."""
import argparse
import json
from pathlib import Path
import subprocess
import sys

METHODS = {"initialize", "initialized", "account/read", "model/list"}


def client(request_file):
    request = json.loads(Path(request_file).read_bytes())
    server = subprocess.Popen(request["argv"], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                              stderr=sys.stderr.buffer, shell=False)
    responses = {}

    def send(value):
        if value["method"] not in METHODS:
            raise ValueError("non-metadata RPC forbidden")
        server.stdin.write((json.dumps(value) + "\n").encode())
        server.stdin.flush()

    def receive(identifier):
        while identifier not in responses:
            raw = server.stdout.readline()
            if not raw:
                raise RuntimeError("app-server closed before metadata response")
            sys.stdout.buffer.write(raw)
            sys.stdout.buffer.flush()
            value = json.loads(raw)
            if "id" in value:
                responses[value["id"]] = value
            if value.get("method", "").startswith(("thread/", "turn/", "item/")):
                raise RuntimeError("unexpected Actor activity in metadata client")
        return responses[identifier]

    try:
        send(request["initialize"])
        if "error" in receive(0):
            raise RuntimeError("metadata initialize rejected")
        send({"method": "initialized", "params": {}})
        send({"method": "account/read", "id": 1, "params": {"refreshToken": False}})
        receive(1)
        cursor = None
        for page in range(3):
            params = {"limit": 100, "includeHidden": True}
            if cursor:
                params["cursor"] = cursor
            send({"method": "model/list", "id": 2 + page, "params": params})
            value = receive(2 + page)
            cursor = value.get("result", {}).get("nextCursor")
            if "error" in value or not cursor:
                break
        else:
            raise RuntimeError("metadata pagination bound reached")
    finally:
        server.stdin.close()
        try:
            server.wait(timeout=3)
        except subprocess.TimeoutExpired:
            server.terminate()
            server.wait(timeout=3)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("request")
    client(parser.parse_args().request)
