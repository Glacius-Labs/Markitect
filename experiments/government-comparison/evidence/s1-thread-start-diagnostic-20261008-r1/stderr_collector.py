"""One-off bounded, privacy-preserving stderr diagnostics for R1 thread/start."""

from __future__ import annotations

from datetime import datetime
import re
from typing import Any


MAX_CAPTURE_BYTES = 16_384
MAX_DIAGNOSTIC_LINES = 8
_MAX_TIMESTAMP = re.compile(rb"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z\Z")
_TARGET_TOKEN = re.compile(rb"([a-z][a-z0-9_]*(?:::[a-z][a-z0-9_]*)*):?\Z")
_VERIFIED_COMPONENTS = frozenset({"codex_core", "codex_protocol", "codex_app_server", "codex_cli", "codex_tui"})
_LEVELS = {
    b"ERROR": ("error", "stderr.level.error"),
    b"WARN": ("warning", "stderr.level.warning"),
    b"WARNING": ("warning", "stderr.level.warning"),
    b"INFO": ("info", "stderr.level.info"),
    b"DEBUG": ("debug", "stderr.level.debug"),
    b"TRACE": ("trace", "stderr.level.trace"),
}
_CODES = (
    "stderr.level.error",
    "stderr.level.warning",
    "stderr.level.info",
    "stderr.level.debug",
    "stderr.level.trace",
    "stderr.unclassified",
    "stderr.structured.unclassified",
    "stderr.invalid_utf8",
    "stderr.empty_line",
)


class StderrCollector:
    """Capture at most 16 KiB transiently and classify at most eight full lines.

    Raw bytes exist only in the bounded in-memory fragment buffer while a line
    is incomplete. They are discarded as soon as a complete line is reduced.
    No method returns, persists, or logs source text or bytes.
    """

    def __init__(self, max_capture_bytes: int, max_lines: int) -> None:
        if type(max_capture_bytes) is not int or not 1 <= max_capture_bytes <= MAX_CAPTURE_BYTES:
            raise ValueError("capture limit must be between 1 and the fixed 16 KiB ceiling")
        if type(max_lines) is not int or not 1 <= max_lines <= MAX_DIAGNOSTIC_LINES:
            raise ValueError("line limit must be between 1 and the fixed eight-line ceiling")
        self._max_capture_bytes = max_capture_bytes
        self._max_lines = max_lines
        self._triggered = False
        self._finished = False
        self._observed_bytes = 0
        self._captured_bytes = 0
        self._discarded_bytes = 0
        self._physical_lines = 0
        self._capture_capped = False
        self._line_cap_reached = False
        self._unterminated_fragment = False
        self._utf8_invalid = False
        self._fragment = bytearray()
        self._lines: list[dict[str, Any]] = []
        self._counts = {code: 0 for code in _CODES}
        self._result: dict[str, Any] | None = None

    @property
    def triggered(self) -> bool:
        """True after the first nonempty stderr byte; the diagnostic is terminal."""
        return self._triggered

    def feed(self, chunk: bytes) -> None:
        """Observe one stderr chunk, including bounded cleanup-only fragments."""
        if self._finished:
            raise RuntimeError("stderr collector is finished")
        if not isinstance(chunk, bytes):
            raise TypeError("stderr chunk must be bytes")
        if not chunk:
            return
        self._triggered = True
        self._observed_bytes += len(chunk)
        available = self._max_capture_bytes - self._captured_bytes
        if available <= 0:
            self._capture_capped = True
            self._discarded_bytes += len(chunk)
            return
        take = min(available, len(chunk))
        self._captured_bytes += take
        self._discarded_bytes += len(chunk) - take
        if take < len(chunk):
            self._capture_capped = True
        self._consume(chunk[:take])

    def _consume(self, data: bytes) -> None:
        for index, byte in enumerate(data):
            if len(self._lines) >= self._max_lines:
                self._line_cap_reached = True
                self._discarded_bytes += len(data) - index
                self._captured_bytes -= len(data) - index
                return
            if byte == 0x0A:
                line = bytes(self._fragment)
                self._fragment.clear()
                if line.endswith(b"\r"):
                    line = line[:-1]
                self._physical_lines += 1
                self._classify(line)
            else:
                self._fragment.append(byte)

    def _classify(self, line: bytes) -> None:
        if len(self._lines) >= self._max_lines:
            self._line_cap_reached = True
            return
        if not line:
            code, severity = "stderr.empty_line", "unknown"
        else:
            try:
                line.decode("utf-8", errors="strict")
            except UnicodeDecodeError:
                self._utf8_invalid = True
                code, severity = "stderr.invalid_utf8", "unknown"
            else:
                tokens = line.split()
                offset = 0
                if tokens and _MAX_TIMESTAMP.fullmatch(tokens[0]):
                    # Validate the calendar fields, while deliberately keeping
                    # neither timestamp value nor source text.
                    stamp = tokens[0][:-1].split(b".", 1)[0]
                    try:
                        datetime.strptime(stamp.decode("ascii"), "%Y-%m-%dT%H:%M:%S")
                    except (UnicodeDecodeError, ValueError):
                        tokens = []
                    else:
                        offset = 1
                level_token = tokens[offset] if len(tokens) > offset else b""
                classified = _LEVELS.get(level_token)
                if classified is None:
                    code, severity = "stderr.unclassified", "unknown"
                else:
                    severity, level_code = classified
                    component = "unknown"
                    target_token = tokens[offset + 1] if len(tokens) > offset + 1 else b""
                    target_match = _TARGET_TOKEN.fullmatch(target_token)
                    if target_match:
                        target = target_match.group(1).decode("ascii")
                        root_component = target.split("::", 1)[0]
                        if root_component in _VERIFIED_COMPONENTS:
                            component = root_component
                    if component == "unknown":
                        code = level_code
                    else:
                        code = "stderr.structured.unclassified"
                        self._counts[code] += 1
                        self._lines.append({
                            "line": len(self._lines) + 1,
                            "code": code,
                            "severity": severity,
                            "component": component,
                        })
                        return
        self._counts[code] += 1
        self._lines.append({
            "line": len(self._lines) + 1,
            "code": code,
            "severity": severity,
            "component": "unknown",
        })

    def finish(self) -> dict[str, Any]:
        """Finalize the safe receipt; an unterminated fragment is never classified."""
        if self._result is not None:
            return self._copy_result(self._result)
        if self._fragment:
            self._unterminated_fragment = True
            self._fragment.clear()
        self._finished = True
        self._result = {
            "triggered": self._triggered,
            "observedBytes": self._observed_bytes,
            "capturedBytes": self._captured_bytes,
            "discardedBytes": self._discarded_bytes,
            "completeLines": self._physical_lines,
            "classifiedLines": self._lines,
            "classificationCounts": dict(self._counts),
            "captureCapped": self._capture_capped,
            "lineCapReached": self._line_cap_reached,
            "truncated": self._capture_capped or self._line_cap_reached,
            "unterminatedFragment": self._unterminated_fragment,
            "utf8Invalid": self._utf8_invalid,
            "rawTextRetained": False,
        }
        return self._copy_result(self._result)

    @staticmethod
    def _copy_result(result: dict[str, Any]) -> dict[str, Any]:
        return {
            **result,
            "classifiedLines": [dict(line) for line in result["classifiedLines"]],
            "classificationCounts": dict(result["classificationCounts"]),
        }
