"""Bounded sequential JSONL transport. No model, policy or Apply behavior lives here."""

import base64
import json
import math
import os
import queue
import subprocess
import threading
import time
from dataclasses import dataclass


class TransportFailure(Exception):
    pass


@dataclass(frozen=True)
class Limits:
    request_seconds: float = 2
    process_seconds: float = 30
    output_bytes: int = 262144
    request_bytes: int = 16384

    def __post_init__(self):
        for value in (self.request_seconds, self.process_seconds):
            if isinstance(value, bool) or not math.isfinite(value) or not 0 < value <= 300:
                raise ValueError("timeouts must be finite, positive and at most 300 seconds")
        for value in (self.output_bytes, self.request_bytes):
            if type(value) is not int or not 0 < value <= 16777216:
                raise ValueError("byte limits must be positive integers at most 16 MiB")


def strict_json(data):
    def pairs(items):
        result = {}
        for key, value in items:
            if key in result:
                raise ValueError("duplicate JSON object key")
            result[key] = value
        return result

    def constant(value):
        raise ValueError("nonfinite JSON number: " + value)

    return json.loads(data, object_pairs_hook=pairs, parse_constant=constant)


class Process:
    """One initially empty library per instance; captures exact bounded output bytes."""

    def __init__(self, command, cwd, limits=Limits()):
        self.limits = limits
        self.deadline = time.monotonic() + limits.process_seconds
        self.lines = queue.Queue()
        self.lock = threading.Lock()
        self.raw = {"stdout": bytearray(), "stderr": bytearray()}
        self.failure = None
        self.pairs = []
        self.closed = False
        # No inherited provider credentials. This is environment filtering, not a sandbox.
        env = {key: os.environ[key] for key in ("SystemRoot", "WINDIR") if key in os.environ}
        self.child = subprocess.Popen(command, cwd=cwd, env=env, stdin=subprocess.PIPE,
                                      stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                      bufsize=0, shell=False)
        self.readers = []
        for name in self.raw:
            thread = threading.Thread(target=self._read, args=(name,), daemon=True)
            thread.start()
            self.readers.append(thread)

    def _read(self, name):
        pending = bytearray()
        try:
            while True:
                chunk = os.read(getattr(self.child, name).fileno(), 4096)
                if not chunk:
                    break
                with self.lock:
                    remaining = self.limits.output_bytes - sum(map(len, self.raw.values()))
                    self.raw[name].extend(chunk[:remaining])
                    if len(chunk) > remaining:
                        self.failure = self.failure or "output byte limit exceeded (trace truncated)"
                        self.child.kill()
                        break
                if name == "stdout":
                    pending.extend(chunk)
                    while b"\n" in pending:
                        line, _, pending = pending.partition(b"\n")
                        self.lines.put(bytes(line))
            if name == "stdout" and pending:
                self.failure = self.failure or "unterminated stdout line"
        except (OSError, ValueError) as exc:
            self.failure = self.failure or "stream read failed: " + str(exc)
        finally:
            if name == "stdout":
                self.lines.put(None)

    def _remaining(self):
        if self.failure:
            raise TransportFailure(self.failure)
        remaining = min(self.limits.request_seconds, self.deadline - time.monotonic())
        if remaining <= 0:
            raise TransportFailure("process deadline exceeded")
        return remaining

    def request(self, request=None, raw=None):
        if not self.lines.empty():
            raise TransportFailure("unsolicited stdout or premature EOF")
        wire = (raw if raw is not None else json.dumps(request, ensure_ascii=False)).encode("utf-8") + b"\n"
        if len(wire) > self.limits.request_bytes or b"\n" in wire[:-1]:
            raise TransportFailure("request is not one bounded JSONL line")
        pair = {"request_base64": base64.b64encode(wire).decode(), "response_base64": None}
        self.pairs.append(pair)
        sent = queue.Queue(maxsize=1)

        def write():
            try:
                view = memoryview(wire)
                while view:
                    count = self.child.stdin.write(view)
                    if not count:
                        raise BrokenPipeError("stdin closed")
                    view = view[count:]
                sent.put(None)
            except (OSError, ValueError) as exc:
                sent.put(str(exc))

        writer = threading.Thread(target=write, daemon=True)
        writer.start()
        deadline = time.monotonic() + self._remaining()
        try:
            error = sent.get(timeout=max(0.001, deadline - time.monotonic()))
            if error:
                raise TransportFailure("stdin write failed: " + error)
            line = self.lines.get(timeout=max(0.001, deadline - time.monotonic()))
        except queue.Empty as exc:
            raise TransportFailure("request timeout / missing response") from exc
        self._remaining()
        if line is None:
            raise TransportFailure("missing response / premature EOF")
        pair["response_base64"] = base64.b64encode(line + b"\n").decode()
        try:
            response = strict_json(line.decode("utf-8"))
        except (ValueError, UnicodeError, RecursionError) as exc:
            raise TransportFailure("malformed response: " + str(exc)) from exc
        if not isinstance(response, dict):
            raise TransportFailure("response must be a JSON object")
        return response

    def finish(self):
        self.child.stdin.close()
        try:
            self.child.wait(timeout=self._remaining())
        except subprocess.TimeoutExpired as exc:
            raise TransportFailure("EOF exit timeout") from exc
        for thread in self.readers:
            thread.join(timeout=self._remaining())
            if thread.is_alive():
                raise TransportFailure("stream did not close after process exit")
        self._remaining()
        if self.child.returncode != 0:
            raise TransportFailure("nonzero exit: " + str(self.child.returncode))
        while not self.lines.empty():
            if self.lines.get_nowait() is not None:
                raise TransportFailure("extra stdout response")

    def close(self):
        if self.closed:
            return
        self.closed = True
        if self.child.poll() is None:
            self.child.kill()
        self.child.wait(timeout=5)
        for stream in (self.child.stdin, self.child.stdout, self.child.stderr):
            stream.close()
        for thread in self.readers:
            thread.join(timeout=0.2)

    def trace(self):
        with self.lock:
            return {"streams_base64": {k: base64.b64encode(v).decode() for k, v in self.raw.items()},
                    "pairs": self.pairs, "exit_code": self.child.poll(), "transport_error": self.failure}
