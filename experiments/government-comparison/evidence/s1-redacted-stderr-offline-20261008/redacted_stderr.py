"""Offline-only prospective stderr excerpt; importing never starts a process.

Receipt metadata is public-safe by contract. Excerpt text is private: a later
explicit runtime grant must bind its external location, targeted read and removal.
Redaction is deliberately conservative, not a guarantee against unknown secrets.
"""
from __future__ import annotations

import copy
import json
from pathlib import Path
import re
import threading
import unicodedata

INPUT_LIMIT = 1024
LINE_LIMIT = 4
OUTPUT_LIMIT = 2048
REPO = Path(__file__).resolve().parents[4]
_SGR = re.compile(r"\x1b\[(?:[0-9]{1,3}(?:;[0-9]{1,3})*)?m")
_ENCODED_CHAR = re.compile(r"\\(?:u[0-9A-Fa-f]{4}|U[0-9A-Fa-f]{8}|x[0-9A-Fa-f]{2})")
_SENSITIVE = re.compile(
    r"\b(?:authorization|bearer|basic|password|passwd|pwd|secret|token|auth[_-]?code|api[_-]?key|credential|private[_ -]?key|signing[_ -]?key)s?\b"
    r"|\b(?:cookie|set-cookie)\s*:"
    r"|\b[\w-]*(?:token|secret|password|passwd|pwd|auth[_-]?code|authorization[_-]?code|api[_-]?key|access[_-]?key|credential|account[_-]?key|sharedaccesssignature)[\w-]*[\"']?\s*[:=]"
    r"|\b(?:sig|signature)[\"']?\s*="
    r"|-----\s*(?:BEGIN|END)\b[^\r\n]*\b(?:KEY|CERTIFICATE)\s*-----"
    r"|\b[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b"
    r"|\bsk-(?:proj-|svcacct-)?[A-Za-z0-9_-]{8,}"
    r"|\b(?:gh[pousr]_[A-Za-z0-9_]{8,}|github_pat_[A-Za-z0-9_]{8,})"
    r"|\b(?:AKIA|ASIA)[A-Z0-9]{16}\b"
    r"|\bxox[baprs]-[A-Za-z0-9-]{8,}"
    r"|\bAIza[A-Za-z0-9_-]{20,}"
    r"|\b[a-z][a-z0-9+.-]*://[^\s/]*:[^\s/]*@"
    r"|^\s*[A-Za-z0-9+/]{40,}={0,2}\s*$",
    re.IGNORECASE,
)
_URL = re.compile(r"\b(?:[a-z][a-z0-9+.-]*://|www\.)[^\s<>\"']+", re.IGNORECASE)
_EMAIL = re.compile(r"(?<![\w.+-])[\w.!#$%&'*+/=?^`{|}~-]+@[\w-]+(?:\.[\w-]+)+", re.UNICODE)
_QUOTED_PATH = re.compile(r'''(["'])(?:[A-Za-z]:[\\/]|\\\\|/)[^"']*\1''')
# Unknown unquoted paths may contain spaces: redact their entire remaining tail
# rather than guessing where the private path ends.
_WINDOWS_PATH = re.compile(r"(?<!\w)(?:[A-Za-z]:[\\/]|\\\\|//)[^\r\n]*")
_POSIX_PATH = re.compile(r"(?<![\w/:])/(?!/)[^\r\n]*")


class RedactedStderrCollector:
    def __init__(self, stop_event: threading.Event, known_paths: tuple[str, ...] = ()) -> None:
        self._stop = stop_event
        home = str(Path.home())
        paths = (*known_paths, home, home.replace("\\", "/"),
                 "C:/Users/Consiliari/Documents/Scientist-Probes/s1-common-runner-read-tool-20261008-r1/actor")
        self._known_paths = tuple(sorted({p for p in paths if p}, key=len, reverse=True))
        self._triggered = False
        self._finished = False
        self._observed = 0
        self._selected_bytes = 0
        self._complete = 0
        self._pending = bytearray()
        self._pem = False
        self._lines: list[dict] = []
        self._dropped: list[dict] = []
        self._byte_cap = False
        self._line_cap = False
        self._input_truncated = False
        self._incomplete = False
        self._output_truncated = False
        self._rendered: bytes | None = None

    def feed(self, chunk: bytes, cleanup: bool = False) -> None:
        if self._finished:
            raise ValueError("collector-finished")
        if type(chunk) is not bytes:
            raise TypeError("bytes-required")
        if not chunk:
            return
        # This precedes every scan, redaction, output and possible parser error.
        self._stop.set()
        if self._triggered and not cleanup:
            raise ValueError("stderr-fragment-outside-cleanup")
        self._triggered = True
        self._observed += len(chunk)
        for byte in chunk:
            if self._complete >= LINE_LIMIT:
                self._line_cap = True
                break
            if self._selected_bytes >= INPUT_LIMIT:
                self._byte_cap = True
                self._input_truncated = True
                self._drop_pending("input-cap")
                break
            self._selected_bytes += 1
            if byte == 10:
                raw = bytes(self._pending)
                self._pending.clear()
                self._complete += 1
                if raw.endswith(b"\r"):
                    raw = raw[:-1]
                self._accept_line(raw, self._complete)
            else:
                self._pending.append(byte)
        self._byte_cap |= self._selected_bytes == INPUT_LIMIT

    def _drop_pending(self, reason: str) -> None:
        if self._pending:
            self._pending.clear()
            self._incomplete = True
            self._dropped.append({"line": self._complete + 1, "reason": reason})

    def _accept_line(self, raw: bytes, number: int) -> None:
        try:
            text = raw.decode("utf-8", errors="strict")
        except UnicodeDecodeError:
            self._dropped.append({"line": number, "reason": "invalid-utf8"})
            return
        try:
            result, reason = self._redact(text)
        except Exception:
            # Fail closed without emitting exception text or the source line.
            self._dropped.append({"line": number, "reason": "redaction-failure"})
            return
        if reason:
            self._dropped.append({"line": number, "reason": reason})
        else:
            self._lines.append({"line": number, **result})

    def _redact(self, text: str) -> tuple[dict, str | None]:
        normalized, count = _SGR.subn("", text)
        if any(unicodedata.category(char).startswith("C") for char in normalized):
            return {}, "control-or-escape"
        if _ENCODED_CHAR.search(normalized):
            return {}, "encoded-structured-line"
        object_start = normalized.find("{")
        if object_start >= 0:
            try:
                json.loads(normalized[object_start:])
            except (ValueError, RecursionError):
                return {}, "malformed-structured-line"
            # Escaped keys/values can conceal sensitive labels or paths. Do not
            # invent a second JSON message parser or publish their raw encoding.
            if "\\" in normalized:
                return {}, "encoded-structured-line"
        begin = re.search(r"-----\s*BEGIN\b.*\b(?:KEY|CERTIFICATE)\s*-----", normalized, re.IGNORECASE)
        end = re.search(r"-----\s*END\b.*\b(?:KEY|CERTIFICATE)\s*-----", normalized, re.IGNORECASE)
        if self._pem or begin:
            self._pem = not bool(end)
            return {}, "sensitive"
        if _SENSITIVE.search(normalized):
            return {}, "sensitive"
        markers: list[str] = []
        for pattern, replacement, marker in [(_URL, "[url]", "url"), (_EMAIL, "[email]", "email")]:
            normalized, hits = pattern.subn(replacement, normalized)
            if hits:
                markers.append(marker)
        for pattern in (_QUOTED_PATH, _WINDOWS_PATH, _POSIX_PATH):
            normalized, hits = pattern.subn("[path]", normalized)
            if hits and "path" not in markers:
                markers.append("path")
        for path in self._known_paths:
            # Also redact exact caller-known forms not recognized as absolute.
            for variant in {path, path.replace("\\", "/"), path.replace("/", "\\")}:
                normalized, hits = re.subn(re.escape(variant), "[path]", normalized, flags=re.IGNORECASE)
                if hits and "path" not in markers:
                    markers.append("path")
        return {"text": normalized, "redacted": markers, "sgrRemoved": bool(count)}, None

    def _metadata(self) -> dict:
        return {"triggered": self._triggered, "observedBytes": self._observed,
                "selectedInputBytes": self._selected_bytes, "completeLines": self._complete,
                "includedLines": len(self._lines), "dropped": self._dropped,
                "byteCapReached": self._byte_cap, "lineCapReached": self._line_cap,
                "inputTruncated": self._input_truncated, "incompleteLineDropped": self._incomplete,
                "outputTruncated": self._output_truncated,
                "rawTextPersisted": False, "contentFingerprintsPersisted": False,
                "redactionGuarantee": "known-patterns-only"}

    def _encode(self) -> bytes:
        return (json.dumps({**self._metadata(), "lines": self._lines}, ensure_ascii=False,
                           separators=(",", ":")) + "\n").encode("utf-8")

    def _finalize(self) -> None:
        if self._finished:
            return
        self._drop_pending("input-cap" if self._byte_cap else "incomplete")
        self._input_truncated |= self._byte_cap and self._incomplete
        self._finished = True
        encoded = self._encode()
        while len(encoded) > OUTPUT_LIMIT and self._lines:
            removed = self._lines.pop()
            self._dropped.append({"line": removed["line"], "reason": "output-cap"})
            self._output_truncated = True
            encoded = self._encode()
        # Metadata is bounded by four line records; never cut an encoded line.
        if len(encoded) > OUTPUT_LIMIT:
            raise ValueError("metadata-output-cap")
        self._rendered = encoded

    def finish(self) -> dict:
        """Metadata-only receipt; never include excerpt text in normal results."""
        self._finalize()
        return copy.deepcopy(self._metadata())

    def render_private(self) -> bytes:
        """Sanitized text for the explicitly authorized private read path only."""
        self._finalize()
        return self._rendered

    def write_private_excerpt(self, directory: Path) -> Path:
        """Write one sanitized file exclusively in a caller-bound fresh external dir.

        No ACL/isolation guarantee: the runtime grant owns destination access and
        removal. No console/logging/public receipt/callback gets excerpt text.
        """
        directory = Path(directory)
        if not directory.is_absolute() or not directory.is_dir():
            raise ValueError("external-directory-required")
        if directory.is_symlink() or directory.is_junction():
            raise ValueError("linked-directory-forbidden")
        resolved = directory.resolve()
        if resolved == REPO or REPO in resolved.parents or any(directory.iterdir()):
            raise ValueError("fresh-external-directory-required")
        path = directory / "redacted-stderr.json"
        with path.open("xb") as output:
            output.write(self.render_private())
        return path
