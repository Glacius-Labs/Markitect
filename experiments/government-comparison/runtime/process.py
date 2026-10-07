"""Shell-free process-tree deadline. Windows children enter a Job before work starts."""
from __future__ import annotations

import base64
import ctypes
from ctypes import wintypes
import hashlib
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import time


class WindowsJob:
    def __init__(self, process):
        kernel = ctypes.WinDLL("kernel32", use_last_error=True)
        kernel.CreateJobObjectW.argtypes = [ctypes.c_void_p, wintypes.LPCWSTR]
        kernel.CreateJobObjectW.restype = wintypes.HANDLE
        kernel.SetInformationJobObject.argtypes = [wintypes.HANDLE, ctypes.c_int, ctypes.c_void_p, wintypes.DWORD]
        kernel.AssignProcessToJobObject.argtypes = [wintypes.HANDLE, wintypes.HANDLE]
        kernel.CloseHandle.argtypes = [wintypes.HANDLE]
        kernel.TerminateJobObject.argtypes = [wintypes.HANDLE, wintypes.UINT]

        class Basic(ctypes.Structure):
            _fields_ = [("PerProcessUserTimeLimit", ctypes.c_int64), ("PerJobUserTimeLimit", ctypes.c_int64),
                        ("LimitFlags", wintypes.DWORD), ("MinimumWorkingSetSize", ctypes.c_size_t),
                        ("MaximumWorkingSetSize", ctypes.c_size_t), ("ActiveProcessLimit", wintypes.DWORD),
                        ("Affinity", ctypes.c_size_t), ("PriorityClass", wintypes.DWORD),
                        ("SchedulingClass", wintypes.DWORD)]

        class IO(ctypes.Structure):
            _fields_ = [(name, ctypes.c_uint64) for name in ("ReadOperationCount", "WriteOperationCount",
                        "OtherOperationCount", "ReadTransferCount", "WriteTransferCount", "OtherTransferCount")]

        class Extended(ctypes.Structure):
            _fields_ = [("BasicLimitInformation", Basic), ("IoInfo", IO),
                        ("ProcessMemoryLimit", ctypes.c_size_t), ("JobMemoryLimit", ctypes.c_size_t),
                        ("PeakProcessMemoryUsed", ctypes.c_size_t), ("PeakJobMemoryUsed", ctypes.c_size_t)]

        self.kernel, self.handle = kernel, kernel.CreateJobObjectW(None, None)
        limits = Extended()
        limits.BasicLimitInformation.LimitFlags = 0x2000  # JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
        if not self.handle:
            raise ctypes.WinError(ctypes.get_last_error())
        if not kernel.SetInformationJobObject(self.handle, 9, ctypes.byref(limits), ctypes.sizeof(limits)):
            self.close()
            raise ctypes.WinError(ctypes.get_last_error())
        if not kernel.AssignProcessToJobObject(self.handle, wintypes.HANDLE(int(process._handle))):
            self.close()
            raise ctypes.WinError(ctypes.get_last_error())

    def kill(self):
        if not self.kernel.TerminateJobObject(self.handle, 1):
            raise ctypes.WinError(ctypes.get_last_error())

    def close(self):
        if self.handle:
            self.kernel.CloseHandle(self.handle)
            self.handle = None


def bounded(argv, cwd, evidence, wall_seconds, *, stop_path=None, stdin=b"", env=None, max_log_bytes=16777216):
    """One call, no retry. Returns raw-log hashes and an explicit stop reason.

    Evidence must be a new directory. A small launcher waits for JSON on stdin;
    only after Windows Job assignment succeeds is the requested command released.
    This controls process lifetime, not filesystem confidentiality or provider work.
    """
    if not argv or not Path(argv[0]).is_absolute() or not Path(cwd).is_absolute():
        raise ValueError("absolute executable and cwd required")
    if not 0 < wall_seconds <= 180:
        raise ValueError("readiness calls require 0 < wallSeconds <= 180")
    directory = Path(evidence)
    directory.mkdir(parents=True, exist_ok=False)
    started = time.monotonic()
    code, reason, job, process = None, None, None, None
    with (directory / "stdout.log").open("wb") as out, (directory / "stderr.log").open("wb") as err:
        try:
            process = subprocess.Popen([sys.executable, str(Path(__file__).resolve()), "--child"],
                                       cwd=cwd, stdin=subprocess.PIPE, stdout=out, stderr=err, env=env,
                                       start_new_session=os.name != "nt", shell=False)
            if os.name == "nt":
                job = WindowsJob(process)  # Failure cannot release target executable.
            payload = json.dumps({"argv": argv, "stdin": base64.b64encode(stdin).decode("ascii")}).encode() + b"\n"
            if stop_path and Path(stop_path).exists():
                reason = "stop_requested"
                if job:
                    job.kill()
                else:
                    os.killpg(process.pid, signal.SIGKILL)
            else:
                process.stdin.write(payload)
                process.stdin.close()
            while process.poll() is None:
                if stop_path and Path(stop_path).exists():
                    reason = "stop_requested"
                elif time.monotonic() - started >= wall_seconds:
                    reason = "wall_deadline"
                elif any((directory / name).stat().st_size > max_log_bytes for name in ("stdout.log", "stderr.log")):
                    reason = "log_limit"
                if reason:
                    if job:
                        job.kill()
                    else:
                        os.killpg(process.pid, signal.SIGKILL)
                    break
                time.sleep(0.025)
            code = process.wait(timeout=5)
        except BaseException:
            if process and process.poll() is None:
                if job:
                    job.kill()
                elif os.name == "nt":
                    process.kill()  # Only the still-gated launcher exists on Job assignment failure.
                else:
                    os.killpg(process.pid, signal.SIGKILL)
                process.wait(timeout=5)
            raise
        finally:
            if job:
                job.close()  # Also removes surviving descendants after successful parent exit.
            elif process and os.name != "nt":
                try:
                    os.killpg(process.pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass
    receipts = [{"path": str((directory / name).resolve()),
                 "sha256": hashlib.sha256((directory / name).read_bytes()).hexdigest(), "kind": name}
                for name in ("stdout.log", "stderr.log")]
    result = {"argv": argv, "cwd": str(cwd), "returnCode": code,
              "wallSeconds": time.monotonic() - started, "stopReason": reason,
              "processTreeControl": "windows-job-kill-on-close" if os.name == "nt" else "POSIX-process-group",
              "filesystemIsolation": "not established by process wrapper", "automaticRetries": 0,
              "receipts": receipts}
    (directory / "process.json").write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    return result


if __name__ == "__main__" and sys.argv[1:] == ["--child"]:
    instruction = json.loads(sys.stdin.buffer.readline())
    child = subprocess.Popen(instruction["argv"], stdin=subprocess.PIPE, shell=False)
    child.communicate(base64.b64decode(instruction["stdin"], validate=True))
    raise SystemExit(child.returncode)
