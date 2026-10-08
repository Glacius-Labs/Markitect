"""Focused offline tests for bounded, privacy-preserving stderr diagnostics."""
from __future__ import annotations

import importlib.util
import json
from pathlib import Path
import unittest


HERE = Path(__file__).resolve().parent


def load_file(name: str, path: Path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


collector_module = load_file("thread_diagnostic_stderr_collector", HERE / "stderr_collector.py")
StderrCollector = collector_module.StderrCollector
client_module = load_file("thread_diagnostic_client_helpers", HERE / "client.py")
CONTRACT = json.loads((HERE / "diagnostic-contract.json").read_text(encoding="utf-8"))
SCHEMA_CONTRACT = json.loads((HERE / "schema-contract.json").read_text(encoding="utf-8"))
PROFILE = json.loads((HERE / "profile.json").read_text(encoding="utf-8"))


def schema_thread(thread_id="thread-diagnostic-r1", turns=None):
    return {"cliVersion": "offline-test", "createdAt": 1, "cwd": PROFILE["cwd"],
            "ephemeral": True, "id": thread_id, "model": "gpt-6.1-sol",
            "modelProvider": "openai", "preview": "", "projectId": None,
            "sessionId": "session-diagnostic-r1", "source": "appServer",
            "status": {"type": "active", "activeFlags": []},
            "turns": [] if turns is None else turns, "updatedAt": 1,
            "parentThreadId": None, "forkedFromId": None}


def schema_thread_response(thread_id="thread-diagnostic-r1", turns=None):
    return {"approvalPolicy": "never", "approvalsReviewer": "user",
            "cwd": PROFILE["cwd"], "model": "gpt-6.1-sol", "modelProvider": "openai",
            "sandbox": {"type": "readOnly", "networkAccess": False},
            "activePermissionProfile": {"id": ":read-only", "extends": None},
            "thread": schema_thread(thread_id, turns)}


def make_gate():
    protocol_module = client_module.load_module(client_module.PROTOCOL,
        client_module.PROTOCOL_SHA, "thread_diagnostic_pinned_protocol_for_test")
    protocol = protocol_module.Protocol(client_module.SCHEMA, SCHEMA_CONTRACT)
    gate_module = load_file("thread_diagnostic_thread_gate", HERE / "thread_gate.py")
    return gate_module.ThreadGate(PROFILE, protocol, SCHEMA_CONTRACT)


class StderrCaptureTests(unittest.TestCase):
    def make_collector(self, byte_limit=None, line_limit=None):
        return StderrCollector(byte_limit or CONTRACT["capture"]["maxBytes"],
                               line_limit or CONTRACT["capture"]["maxLines"])

    def test_split_chunks_reassemble_one_known_structural_line_without_retaining_text(self):
        c = self.make_collector()
        prefix = b"2026-10-08T06:00:00.123Z WARN codex_app_server::startup "
        message = b"startup diagnostic SECRET /private/path https://private.invalid id=account-847\n"
        c.feed(prefix[:17])
        self.assertTrue(c.triggered)
        self.assertTrue(c.triggered)  # first nonempty warning remains terminal
        c.feed(prefix[17:] + message[:11])
        c.feed(message[11:])
        result = c.finish()
        self.assertEqual(result["observedBytes"], len(prefix + message))
        self.assertEqual(result["capturedBytes"], len(prefix + message))
        self.assertEqual(result["completeLines"], 1)
        self.assertEqual(result["classifiedLines"], [{
            "line": 1, "code": "stderr.structured.unclassified",
            "severity": "warning", "component": "codex_app_server",
        }])
        self.assertFalse(result["rawTextRetained"])
        serialized = json.dumps(result)
        for secret in ("SECRET", "/private/path", "https://private.invalid", "account-847", "startup diagnostic"):
            self.assertNotIn(secret, serialized)

    def test_unknown_and_malformed_lines_are_unclassified_and_redacted(self):
        c = self.make_collector()
        c.feed(b"VERBOSE codex_unknown::component private-token-xyz\n")
        c.feed(b"2026-99-44T32:00:00Z INFO codex_core::clock invalid-date-secret\n")
        result = c.finish()
        self.assertEqual([line["code"] for line in result["classifiedLines"]],
                         ["stderr.unclassified", "stderr.unclassified"])
        self.assertTrue(all(line["component"] == "unknown" for line in result["classifiedLines"]))
        self.assertNotIn("private-token-xyz", json.dumps(result))
        self.assertNotIn("invalid-date-secret", json.dumps(result))

    def test_capture_stops_at_exact_16kib_ceiling_and_marks_discarded_fragment(self):
        c = self.make_collector()
        c.feed(b"x" * (16_384 + 19))
        result = c.finish()
        self.assertTrue(result["triggered"])
        self.assertEqual(result["observedBytes"], 16_384 + 19)
        self.assertEqual(result["capturedBytes"], 16_384)
        self.assertEqual(result["discardedBytes"], 19)
        self.assertTrue(result["captureCapped"])
        self.assertTrue(result["truncated"])
        self.assertTrue(result["unterminatedFragment"])
        self.assertEqual(result["classifiedLines"], [])
        self.assertFalse(result["rawTextRetained"])

    def test_only_eight_complete_lines_are_classified_and_later_bytes_are_discarded(self):
        c = self.make_collector()
        line = b"nonsense\n"
        c.feed(line * 10)
        result = c.finish()
        self.assertEqual(result["completeLines"], 8)
        self.assertEqual(len(result["classifiedLines"]), 8)
        self.assertTrue(result["lineCapReached"])
        self.assertTrue(result["truncated"])
        self.assertEqual(result["capturedBytes"], len(line) * 8)
        self.assertEqual(result["discardedBytes"], len(line) * 2)
        self.assertEqual(result["classificationCounts"]["stderr.unclassified"], 8)

    def test_invalid_utf8_and_unterminated_partial_utf8_are_never_exposed(self):
        invalid = self.make_collector()
        invalid.feed(b"INFO codex_core::logger bad-\xff-secret\n")
        result = invalid.finish()
        self.assertTrue(result["utf8Invalid"])
        self.assertEqual(result["classifiedLines"][0]["code"], "stderr.invalid_utf8")
        self.assertNotIn("bad-", json.dumps(result))

        partial = self.make_collector()
        partial.feed(b"WARN codex_cli::start private-\xe2\x82")
        result = partial.finish()
        self.assertTrue(result["unterminatedFragment"])
        self.assertFalse(result["utf8Invalid"])
        self.assertEqual(result["classifiedLines"], [])
        self.assertNotIn("private-", json.dumps(result))

    def test_valid_multibyte_utf8_split_across_feeds_is_classified_only_when_complete(self):
        c = self.make_collector()
        c.feed(b"INFO codex_core::logger caf\xc3")
        self.assertTrue(c.triggered)
        c.feed(b"\xa9\n")
        result = c.finish()
        self.assertEqual(result["classifiedLines"], [{
            "line": 1, "code": "stderr.structured.unclassified",
            "severity": "info", "component": "codex_core",
        }])
        self.assertFalse(result["utf8Invalid"])

    def test_empty_stderr_does_not_trigger_and_feed_after_finish_is_rejected(self):
        c = self.make_collector()
        c.feed(b"")
        self.assertFalse(c.triggered)
        result = c.finish()
        self.assertFalse(result["triggered"])
        self.assertEqual(result["observedBytes"], 0)
        with self.assertRaises(RuntimeError):
            c.feed(b"late cleanup bytes")

    def test_fixed_capture_bounds_cannot_be_loosened(self):
        for byte_limit, line_limit in ((16_385, 8), (16_384, 9), (0, 8), (16_384, 0)):
            with self.subTest(byte_limit=byte_limit, line_limit=line_limit), self.assertRaises(ValueError):
                StderrCollector(byte_limit, line_limit)


class TerminalAdmissionTests(unittest.TestCase):
    def test_first_warning_stderr_triggers_stop_before_next_rpc(self):
        c = StderrCollector(CONTRACT["capture"]["maxBytes"], CONTRACT["capture"]["maxLines"])
        reason = client_module.consume_stderr(c, b"WARN codex_cli::startup harmless-looking-warning\n")
        self.assertEqual(reason, "server-stderr-activity")
        self.assertTrue(c.triggered)
        with self.assertRaisesRegex(ValueError, "server-stderr-activity"):
            client_module.admit_rpc("initialized", ["initialize"], c.triggered, False)

    def test_cleanup_fragment_feed_stays_terminal_and_cannot_resume_rpc_sequence(self):
        c = StderrCollector(CONTRACT["capture"]["maxBytes"], CONTRACT["capture"]["maxLines"])
        sent = ["initialize", "initialized", "config/read", "configRequirements/read"]
        self.assertEqual(client_module.consume_stderr(c, b"ERROR codex_core::startup fragment"),
                         "server-stderr-activity")
        self.assertEqual(client_module.consume_stderr(c, b" continued-in-cleanup\n", cleanup=True),
                         "server-stderr-activity")
        with self.assertRaisesRegex(ValueError, "server-stderr-activity"):
            client_module.admit_rpc("thread/start", sent, c.triggered, False)
        self.assertEqual(sent, ["initialize", "initialized", "config/read", "configRequirements/read"])
        result = c.finish()
        self.assertEqual(result["completeLines"], 1)
        self.assertNotIn("continued-in-cleanup", json.dumps(result))

    def test_second_stderr_chunk_before_cleanup_is_rejected_without_capture(self):
        c = StderrCollector(CONTRACT["capture"]["maxBytes"], CONTRACT["capture"]["maxLines"])
        first = b"ERROR codex_core::startup first-fragment"
        self.assertEqual(client_module.consume_stderr(c, first), "server-stderr-activity")
        with self.assertRaisesRegex(ValueError, "stderr-fragment-outside-cleanup"):
            client_module.consume_stderr(c, b"outside-cleanup-secret")
        result = c.finish()
        self.assertEqual(result["observedBytes"], len(first))
        self.assertNotIn("outside-cleanup-secret", json.dumps(result))

    def test_rpc_gate_rejects_every_call_after_thread_response_and_never_admits_turn(self):
        sent = ["initialize", "initialized", "config/read", "configRequirements/read", "thread/start"]
        with self.assertRaisesRegex(ValueError, "rpc-after-thread-terminal"):
            client_module.admit_rpc("turn/start", sent, False, True)
        with self.assertRaisesRegex(ValueError, "forbidden-or-out-of-order-client-method"):
            client_module.admit_rpc("turn/start", sent, False, False)


class ThreadGateTests(unittest.TestCase):
    def test_valid_schema_bound_thread_response_finishes_before_any_turn(self):
        gate = make_gate()
        gate.accept_response(schema_thread_response())
        result = gate.finish()
        self.assertTrue(result["responseValidated"])
        self.assertEqual(result["threadId"], "thread-diagnostic-r1")
        self.assertEqual(result["turnsRequested"], 0)
        self.assertEqual(result["toolsRequested"], 0)
        with self.assertRaisesRegex(ValueError, "forbidden-or-unknown-notification"):
            gate.accept_notification("turn/started", {})

    def test_conflicting_early_thread_id_is_rejected_after_valid_response(self):
        gate = make_gate()
        gate.accept_notification("thread/started", {"thread": schema_thread("other-thread")})
        with self.assertRaisesRegex(ValueError, "thread-notification-id-mismatch"):
            gate.accept_response(schema_thread_response())

    def test_existing_turn_in_thread_response_is_rejected(self):
        gate = make_gate()
        turns = [{"id": "existing-turn", "status": "completed", "items": []}]
        with self.assertRaisesRegex(ValueError, "unexpected-existing-turn"):
            gate.accept_response(schema_thread_response(turns=turns))


if __name__ == "__main__":
    unittest.main(verbosity=2)
