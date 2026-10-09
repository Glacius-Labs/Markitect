"""Offline subprocess fixtures for Conventional native transport adapters."""

from __future__ import annotations

import json
import base64
import pathlib
import queue
import sys
import tempfile
import threading
import time
import unittest

PLAYGROUND = pathlib.Path(__file__).resolve().parents[1]
sys.path.insert(0, str(PLAYGROUND))
from conventional.backends import BackendSpecError, run, validate_runtime_options  # noqa: E402


RUNTIME_OPTIONS = {"sandbox": "workspace-write", "approvalPolicy": "never",
                   "memoryEnabled": False, "nativeHelperModel": "fixture-model",
                   "nativeHelperEffort": "high"}


class ConventionalBackendTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.root = pathlib.Path(self.temp.name)
        self.events: list[dict] = []

    def tearDown(self) -> None:
        self.temp.cleanup()

    def fixture(self, body: str) -> list[str]:
        path = self.root / "fake_native.py"
        path.write_text(body, encoding="utf-8")
        return [sys.executable, str(path)]

    def spec(self, backend: str, command: list[str], **extra) -> dict:
        return {"backend": backend, "command": command, "cwd": str(self.root),
                "model": "fixture-model", "effort": "high", "timeoutSeconds": 4,
                "requestTimeoutSeconds": 2, **extra}

    def emit(self, event: dict) -> None:
        self.events.append(event)

    def test_cli_start_and_resume_keep_raw_jsonl_and_usage(self) -> None:
        command = self.fixture(r'''
import json, sys
args = sys.argv[1:]
prompt = sys.stdin.read()
thread_id = args[args.index("resume") + 1] if "resume" in args else "fresh-thread"
print(json.dumps({"type":"fixture.args","resume":"resume" in args,"model":"fixture-model" in args,"prompt":prompt,"args":args}), flush=True)
print(json.dumps({"type":"thread.started","thread_id":thread_id}), flush=True)
print(json.dumps({"type":"turn.completed","turn_id":"turn-1","usage":{"input_tokens":3}}), flush=True)
print("fixture diagnostic", file=sys.stderr, flush=True)
''')
        result = run(self.spec("codex-cli", command, runtimeOptions=RUNTIME_OPTIONS), "do the backlog", None,
                     self.emit, threading.Event())
        self.assertEqual("completed", result["state"], (result, self.events))
        self.assertEqual("fresh-thread", result["nativeSessionId"])
        self.assertEqual("turn-1", result["nativeTurnId"])
        self.assertEqual({"input_tokens": 3}, result["usage"])
        args_event = next(e["parsed"] for e in self.events
                          if e.get("parsed", {}).get("type") == "fixture.args")
        self.assertEqual("do the backlog", args_event["prompt"])
        self.assertTrue(args_event["model"])
        self.assertFalse(args_event["resume"])
        for fragment in ('sandbox_mode="workspace-write"', 'approval_policy="never"',
                         "features.memories=false", 'agents.default_subagent_model="fixture-model"',
                         'agents.default_subagent_reasoning_effort="high"'):
            self.assertIn(fragment, args_event["args"])
        self.assertTrue(any(e["stream"] == "stderr" and e["raw"].rstrip("\r\n") == "fixture diagnostic"
                            for e in self.events))

        self.events.clear()
        result = run(self.spec("codex-cli", command, runtimeOptions=RUNTIME_OPTIONS), "continue", "fresh-thread",
                     self.emit, threading.Event())
        self.assertEqual("completed", result["state"], (result, self.events))
        self.assertEqual("fresh-thread", result["nativeSessionId"])
        # The fixture receives argv after the Python script name.
        # Start/resume IDs remain caller-owned and are never silently replaced.
        args_event = next(e["parsed"] for e in self.events
                          if e.get("parsed", {}).get("type") == "fixture.args")
        self.assertEqual("continue", args_event["prompt"])
        self.assertTrue(args_event["resume"])
        self.assertIn("sandbox_mode=\"workspace-write\"", args_event["args"])

    def test_cli_resume_mismatched_thread_is_uncertain(self) -> None:
        command = self.fixture(r'''
import json
print(json.dumps({"type":"thread.started","thread_id":"different-thread"}), flush=True)
print(json.dumps({"type":"turn.completed"}), flush=True)
''')
        result = run(self.spec("codex-cli", command), "continue", "requested-thread",
                     self.emit, threading.Event())
        self.assertEqual("uncertain", result["state"])
        self.assertEqual("requested-thread", result["nativeSessionId"])
        self.assertIsNone(result["nativeTurnId"])
        self.assertIn("different native thread id", result["detail"])

    def test_cli_completed_event_without_native_turn_id_is_supported(self) -> None:
        command = self.fixture(r'''
import json
print(json.dumps({"type":"thread.started","thread_id":"thread-no-turn-id"}), flush=True)
print(json.dumps({"type":"turn.completed"}), flush=True)
''')
        result = run(self.spec("codex-cli", command), "prompt", None,
                     self.emit, threading.Event())
        self.assertEqual("completed", result["state"])
        self.assertEqual("thread-no-turn-id", result["nativeSessionId"])
        self.assertIsNone(result["nativeTurnId"])

    def test_cli_cancel_without_native_interruption_ack_is_uncertain(self) -> None:
        command = self.fixture(r'''
import json, sys, time
print(json.dumps({"type":"thread.started","thread_id":"thread-cancel"}), flush=True)
time.sleep(4)
''')
        cancel = threading.Event()
        def capture(event: dict) -> None:
            self.events.append(event)
            if event.get("parsed", {}).get("type") == "thread.started":
                cancel.set()
        spec = self.spec("codex-cli", command, timeoutSeconds=3)
        result = run(spec, "prompt", None, capture, cancel)
        self.assertEqual("uncertain", result["state"])
        self.assertEqual("thread-cancel", result["nativeSessionId"])
        self.assertEqual("direct-child-only", result["ownedProcessScope"])

    def test_cli_ignores_terminal_for_another_thread(self) -> None:
        command = self.fixture(r'''
import json
def send(x): print(json.dumps(x), flush=True)
send({"type":"thread.started","thread_id":"owned-thread"})
send({"type":"item.completed","thread_id":"other-thread","turn_id":"other-turn"})
send({"type":"turn.completed","thread_id":"other-thread","turn_id":"other-turn"})
''')
        result = run(self.spec("codex-cli", command), "prompt", None,
                     self.emit, threading.Event())
        self.assertEqual("uncertain", result["state"])
        self.assertEqual("owned-thread", result["nativeSessionId"])
        self.assertIsNone(result["nativeTurnId"])

    def test_cli_timeout_is_bounded_when_descendant_keeps_pipe_open(self) -> None:
        command = self.fixture(r'''
import subprocess, sys
subprocess.Popen([sys.executable, "-c", "import time; time.sleep(1.5)"], cwd=sys.prefix)
print("parent exited", flush=True)
''')
        started = time.monotonic()
        result = run(self.spec("codex-cli", command, timeoutSeconds=0.25), "prompt", None,
                     self.emit, threading.Event())
        elapsed = time.monotonic() - started
        self.assertEqual("uncertain", result["state"])
        self.assertLess(elapsed, 1.2, "deadline must apply after direct child exit even while a descendant holds a pipe")
        self.assertIn("descendant", result["detail"])
        # Let the fixture child close the inherited pipe before its temporary
        # workspace is removed; it is intentionally not killed by the backend.
        time.sleep(1.6)

    def test_cli_preserves_invalid_utf8_bytes(self) -> None:
        command = self.fixture(r'''
import sys
sys.stdout.buffer.write(b"bad-json-\xff\r\n"); sys.stdout.flush()
sys.stderr.buffer.write(b"diagnostic-\xfe\n"); sys.stderr.flush()
''')
        result = run(self.spec("codex-cli", command), "prompt", None,
                     self.emit, threading.Event())
        self.assertEqual("uncertain", result["state"])
        stdout = next(e for e in self.events if e["stream"] == "stdout")
        stderr = next(e for e in self.events if e["stream"] == "stderr")
        self.assertEqual(b"bad-json-\xff\r\n", base64.b64decode(stdout["rawBase64"]))
        self.assertEqual(b"diagnostic-\xfe\n", base64.b64decode(stderr["rawBase64"]))
        self.assertEqual(len(base64.b64decode(stdout["rawBase64"])), stdout["rawByteLength"])

    def test_cli_unread_stdin_cannot_bypass_finite_deadline(self) -> None:
        command = self.fixture("import time; time.sleep(2)\n")
        started = time.monotonic()
        result = run(self.spec("codex-cli", command, timeoutSeconds=0.25), "x" * 1000000, None,
                     self.emit, threading.Event())
        self.assertEqual("uncertain", result["state"])
        self.assertIn("delivery", result["detail"])
        self.assertLess(time.monotonic() - started, 1.2)

    def test_app_server_unread_turn_payload_is_bounded(self) -> None:
        command = self.fixture(r'''
import json, sys, time
for line in sys.stdin:
    m=json.loads(line)
    if m.get('method') == 'initialize':
        print(json.dumps({'id':m['id'],'result':{}}),flush=True)
    elif m.get('method') == 'thread/start':
        print(json.dumps({'id':m['id'],'result':{'thread':{'id':'offline'}}}),flush=True)
        time.sleep(2)
        break
''')
        started = time.monotonic()
        result = run(self.spec("codex-app-server", command, timeoutSeconds=0.3), "x" * 1000000, None,
                     self.emit, threading.Event())
        self.assertEqual("uncertain", result["state"])
        self.assertIn("transport failed", result["detail"])
        self.assertLess(time.monotonic() - started, 1.2)

    def test_app_server_handshake_start_and_exact_matching_terminal(self) -> None:
        command = self.fixture(r'''
import json, sys
def send(x): print(json.dumps(x), flush=True)
assert sys.argv[1:] == ['-c', 'sandbox_mode="workspace-write"', '-c', 'approval_policy="never"', '-c', 'features.memories=false', '-c', 'agents.default_subagent_model="fixture-model"', '-c', 'agents.default_subagent_reasoning_effort="high"', 'app-server']
for line in sys.stdin:
    m = json.loads(line); method=m.get("method")
    if method == "initialize": send({"id":m["id"],"result":{}})
    elif method == "initialized": pass
    elif method == "thread/start":
        assert m["params"]["cwd"] == __import__("os").getcwd()
        assert m["params"]["sandbox"] == "workspaceWrite"
        assert m["params"]["approvalPolicy"] == "never"
        send({"id":m["id"],"result":{"thread":{"id":"thread-A","model":"fixture-model","reasoningEffort":"high","modelProvider":"fixture-provider","cwd":__import__("os").getcwd(),"sandbox":"workspaceWrite","approvalPolicy":"never","instructionSources":["AGENTS.md"]}}})
        send({"method":"thread/tokenUsage/updated","params":{"threadId":"thread-A","tokenUsage":{"input_tokens":7}}})
    elif method == "turn/start":
        send({"method":"turn/completed","params":{"threadId":"thread-A","turn":{"id":"wrong","status":"completed"}}})
        send({"id":m["id"],"result":{"turn":{"id":"turn-A"}}})
        send({"method":"turn/completed","params":{"threadId":"thread-A","turn":{"id":"turn-A","status":"completed"}}})
        break
''')
        result = run(self.spec("codex-app-server", command, runtimeOptions=RUNTIME_OPTIONS), "ordinary prompt", None,
                     self.emit, threading.Event())
        self.assertEqual("completed", result["state"])
        self.assertEqual("thread-A", result["nativeSessionId"])
        self.assertEqual("turn-A", result["nativeTurnId"])
        self.assertEqual({"input_tokens": 7}, result["usage"], (result, self.events))
        self.assertEqual("native-reported; aggregate scope unknown", result["usageScope"])
        binding = next(e["parsed"] for e in self.events if e["stream"] == "runtime-binding")
        self.assertEqual("fixture-provider", binding["reported"]["modelProvider"])
        self.assertEqual(["AGENTS.md"], binding["reported"]["instructionSources"])
        calls = [e["parsed"] for e in self.events if e["stream"] == "client" and e.get("parsed", {}).get("method")]
        self.assertEqual(["initialize", "initialized", "thread/start", "turn/start"],
                         [m["method"] for m in calls])

    def test_app_server_turn_completed_status_is_authoritative(self) -> None:
        for status, expected in (("failed", "failed"), (None, "uncertain")):
            with self.subTest(status=status):
                status_source = "" if status is None else f',"status":{json.dumps(status)}'
                body = f'''
import json, sys
def send(x): print(json.dumps(x), flush=True)
for line in sys.stdin:
    m=json.loads(line); method=m.get("method")
    if method == "initialize": send({{"id":m["id"],"result":{{}}}})
    elif method == "thread/start": send({{"id":m["id"],"result":{{"threadId":"t"}}}})
    elif method == "turn/start":
        send({{"id":m["id"],"result":{{"turnId":"u"}}}})
        send({{"method":"turn/completed","params":{{"threadId":"t","turn":{{"id":"u"{status_source}}}}}}})
        break
'''
                result = run(self.spec("codex-app-server", self.fixture(body)), "prompt", None,
                             self.emit, threading.Event())
                self.assertEqual(expected, result["state"])

    def test_app_server_resume_uses_existing_thread_and_captures_stderr(self) -> None:
        command = self.fixture(r'''
import json, sys
print("fixture server diagnostic", file=sys.stderr, flush=True)
def send(x): print(json.dumps(x), flush=True)
for line in sys.stdin:
    m=json.loads(line); method=m.get("method")
    if method == "initialize": send({"id":m["id"],"result":{}})
    elif method == "thread/resume":
        assert m["params"]["threadId"] == "thread-old"
        send({"id":m["id"],"result":{"threadId":"thread-old"}})
    elif method == "turn/start": send({"id":m["id"],"result":{"turnId":"turn-resumed"}})
    elif method == "turn/interrupt": send({"method":"turn/interrupted","params":{"threadId":"thread-old","turn":{"id":"turn-resumed","status":"interrupted"}}})
''')
        cancel = threading.Event()
        def capture(event: dict) -> None:
            self.events.append(event)
            parsed = event.get("parsed")
            if isinstance(parsed, dict) and parsed.get("method") == "turn/start":
                cancel.set()
        result = run(self.spec("codex-app-server", command), "continue", "thread-old",
                     capture, cancel)
        self.assertEqual("cancelled", result["state"], (result, self.events))
        self.assertEqual("thread-old", result["nativeSessionId"])
        self.assertEqual("turn-resumed", result["nativeTurnId"])
        self.assertTrue(any(e["stream"] == "stderr" and "fixture server diagnostic" in e["raw"]
                            for e in self.events))

    def test_app_server_resume_mismatch_stops_before_turn_start(self) -> None:
        command = self.fixture(r'''
import json, sys
def send(x): print(json.dumps(x), flush=True)
for line in sys.stdin:
    m=json.loads(line); method=m.get("method")
    if method == "initialize": send({"id":m["id"],"result":{}})
    elif method == "thread/resume":
        send({"id":m["id"],"result":{"threadId":"different-thread"}})
        break
''')
        result = run(self.spec("codex-app-server", command), "continue", "requested-thread",
                     self.emit, threading.Event())
        self.assertEqual("uncertain", result["state"])
        self.assertEqual("requested-thread", result["nativeSessionId"])
        self.assertIsNone(result["nativeTurnId"])
        self.assertFalse(any(e.get("parsed", {}).get("method") == "turn/start" for e in self.events))

    def test_app_server_approval_is_never_accepted(self) -> None:
        command = self.fixture(r'''
import json, sys
def send(x): print(json.dumps(x), flush=True)
for line in sys.stdin:
    m=json.loads(line); method=m.get("method")
    if method == "initialize": send({"id":m["id"],"result":{}})
    elif method == "thread/start": send({"id":m["id"],"result":{"threadId":"t"}})
    elif method == "turn/start":
        send({"id":m["id"],"result":{"turnId":"u"}})
        send({"id":"approval-1","method":"item/commandExecution/requestApproval","params":{"threadId":"t","turn":{"id":"u"}}})
''')
        result = run(self.spec("codex-app-server", command), "prompt", None,
                     self.emit, threading.Event())
        self.assertEqual("needs_input", result["state"])
        self.assertIn("no approval was granted", result["detail"])
        self.assertFalse(any(e.get("parsed", {}).get("result") == "accept" for e in self.events))

    def test_app_server_deadline_without_terminal_is_uncertain(self) -> None:
        command = self.fixture(r'''
import json, sys, time
def send(x): print(json.dumps(x), flush=True)
for line in sys.stdin:
    m=json.loads(line); method=m.get("method")
    if method == "initialize": send({"id":m["id"],"result":{}})
    elif method == "thread/start": send({"id":m["id"],"result":{"threadId":"t"}})
    elif method == "turn/start":
        send({"id":m["id"],"result":{"turnId":"u"}})
        time.sleep(2)
''')
        spec = self.spec("codex-app-server", command, timeoutSeconds=0.35)
        result = run(spec, "prompt", None, self.emit, threading.Event())
        self.assertEqual("uncertain", result["state"])
        self.assertEqual("t", result["nativeSessionId"])
        self.assertEqual("u", result["nativeTurnId"])
        self.assertIsNone(result["usage"])

    def test_app_server_cancellation_targets_exact_turn_and_waits_for_interrupt(self) -> None:
        command = self.fixture(r'''
import json, sys
def send(x): print(json.dumps(x), flush=True)
thread = None; turn = None
for line in sys.stdin:
    m=json.loads(line); method=m.get("method")
    if method == "initialize": send({"id":m["id"],"result":{}})
    elif method == "thread/start": thread="t-own"; send({"id":m["id"],"result":{"threadId":thread}})
    elif method == "turn/start": turn="u-own"; send({"id":m["id"],"result":{"turnId":turn}})
    elif method == "turn/interrupt":
        assert m["params"] == {"threadId":"t-own","turnId":"u-own"}
        send({"method":"turn/interrupted","params":{"threadId":thread,"turn":{"id":turn,"status":"interrupted"}}})
''')
        cancel = threading.Event()
        def capture(event: dict) -> None:
            self.events.append(event)
            parsed = event.get("parsed")
            if isinstance(parsed, dict) and parsed.get("method") == "turn/start":
                cancel.set()
        result = run(self.spec("codex-app-server", command), "prompt", None,
                     capture, cancel)
        self.assertEqual("cancelled", result["state"], (result, self.events))
        self.assertEqual("t-own", result["nativeSessionId"])
        self.assertEqual("u-own", result["nativeTurnId"])
        interrupt = [e["parsed"] for e in self.events if e["stream"] == "client"
                     and e.get("parsed", {}).get("method") == "turn/interrupt"]
        self.assertEqual({"threadId": "t-own", "turnId": "u-own"}, interrupt[0]["params"])

    def test_rejects_identity_and_instruction_overrides(self) -> None:
        command = self.fixture("raise SystemExit('should not start')")
        with self.assertRaises(BackendSpecError):
            run(self.spec("codex-app-server", command, threadOptions={"threadId": "forged"}),
                "prompt", None, self.emit, threading.Event())
        with self.assertRaises(BackendSpecError):
            run(self.spec("codex-app-server", command, turnOptions={"developerInstructions": "x"}),
                "prompt", None, self.emit, threading.Event())
        with self.assertRaises(BackendSpecError):
            run(self.spec("codex-cli", ["codex"]), "prompt", None,
                self.emit, threading.Event())
        with self.assertRaises(BackendSpecError):
            run(self.spec("codex-cli", command, effort='high"; arbitrary=true'), "prompt", None,
                self.emit, threading.Event())

    def test_runtime_options_are_closed_typed_and_model_paired(self) -> None:
        self.assertEqual(RUNTIME_OPTIONS, validate_runtime_options(RUNTIME_OPTIONS, "fixture-model", "high"))
        invalid = [
            {**RUNTIME_OPTIONS, "permissionBypass": True},
            {**RUNTIME_OPTIONS, "memoryEnabled": "false"},
            {**RUNTIME_OPTIONS, "sandbox": "danger-full-access"},
            {**RUNTIME_OPTIONS, "nativeHelperModel": "another-model"},
            {**RUNTIME_OPTIONS, "nativeHelperEffort": "low"},
        ]
        for options in invalid:
            with self.subTest(options=options), self.assertRaises(BackendSpecError):
                validate_runtime_options(options, "fixture-model", "high")


if __name__ == "__main__":
    unittest.main()
