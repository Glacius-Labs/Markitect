import base64
import argparse
import hashlib
import io
import json
import subprocess
import tempfile
import unittest
from unittest.mock import patch
from pathlib import Path

import runner


def invocation(role: str = "executor") -> dict:
    return {
        "apiVersion": "markitect.example.org/agent-execution/v1alpha1",
        "runId": "a" * 32,
        "nonce": "b" * 32,
        "inputDigest": "sha256:" + "c" * 64,
        "request": {
            "role": role,
            "sourceRevision": "d" * 40,
            "modelDigest": "sha256:" + "e" * 64,
            "modulePin": "module@sha256:" + "f" * 64,
            "projectionId": "projection/example",
            "scopeIds": ["scope/example"],
            "policyIds": [],
            "context": {"numericProperty": 42},
            "artifacts": [],
        },
    }


class CodexRunnerTests(unittest.TestCase):
    def test_strict_json_rejects_duplicate_keys_at_any_depth(self) -> None:
        with self.assertRaises(runner.AdapterError):
            runner.strict_loads('{"outer":{"x":1,"x":2}}')

    def test_invocation_rejects_unknown_fields(self) -> None:
        value = invocation()
        value["request"]["transcript"] = "executor private conversation"
        with self.assertRaises(runner.AdapterError):
            runner.validate_invocation(value)

    def test_artifact_bytes_are_verified(self) -> None:
        value = invocation()
        data = b"content"
        value["request"]["artifacts"] = [{
            "path": "src/file.cs",
            "mode": "0644",
            "digest": "sha256:" + hashlib.sha256(data).hexdigest(),
            "content": base64.b64encode(data).decode("ascii"),
        }]
        self.assertIs(runner.validate_invocation(value), value)
        value["request"]["artifacts"][0]["digest"] = "sha256:" + "0" * 64
        with self.assertRaises(runner.AdapterError):
            runner.validate_invocation(value)

    def test_empty_artifact_bytes_and_digest_prefix_are_closed(self) -> None:
        value = invocation()
        value["request"]["artifacts"] = [{
            "path": "empty.txt",
            "mode": "0644",
            "digest": "sha256:" + hashlib.sha256(b"").hexdigest(),
            "content": "",
        }]
        self.assertIs(runner.validate_invocation(value), value)
        value["request"]["artifacts"][0]["digest"] = hashlib.sha256(b"").hexdigest()
        with self.assertRaises(runner.AdapterError):
            runner.validate_invocation(value)

    def test_role_instructions_do_not_reuse_executor_transcript(self) -> None:
        verifier = runner.role_instructions("verifier")
        self.assertIn("fresh process", verifier)
        self.assertIn("No executor transcript", runner.make_prompt(invocation("verifier")))
        self.assertNotIn("executor private", verifier)

    def test_explicit_model_options_become_literal_codex_config_arguments(self) -> None:
        self.assertEqual(
            runner.model_config_args({"model_reasoning_effort": "high"}),
            ["--config", 'model_reasoning_effort="high"'],
        )

    def test_event_log_records_provider_usage_and_tool_count(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            log_path = Path(directory) / "events.jsonl"
            collector = runner.EventCollector(log_path)
            collector.record_line(b'{"type":"item.started","item":{"type":"command_execution"}}\n')
            collector.record_line(b'{"type":"turn.completed","usage":{"input_tokens":11,"output_tokens":5,"cached_input_tokens":3}}\n')
            collector.close()
            self.assertEqual(
                collector.telemetry(),
                {"source": "provider-reported", "inputTokens": 11, "outputTokens": 5, "cachedTokens": 3, "toolCalls": 1},
            )
            self.assertTrue(log_path.exists())

    def test_codex_launch_uses_read_only_ephemeral_flags_and_private_telemetry(self) -> None:
        value = invocation()
        captured = {}

        class FakeProcess:
            def __init__(self, argv, **kwargs):
                captured["argv"] = argv
                self.stdin = io.BytesIO()
                self.stdout = io.BytesIO(
                    b'{"type":"item.started","item":{"type":"command_execution"}}\n'
                    b'{"type":"turn.completed","usage":{"input_tokens":7,"output_tokens":4}}\n'
                )
                self.stderr = io.BytesIO(b"private model catalog detail")
                response_path = Path(argv[argv.index("--output-last-message") + 1])
                response_path.write_text(json.dumps({
                    "apiVersion": value["apiVersion"],
                    "runId": value["runId"],
                    "nonce": value["nonce"],
                    "role": "executor",
                    "inputDigest": value["inputDigest"],
                    "outcome": "proposed",
                    "candidateFiles": [{"path":"candidate.txt","mode":"0644","content":"candidate"}],
                    "evidenceRefs": [],
                    "verifierObservations": [],
                    "uncertainty": [],
                    "usage": {"source":"model-invented","inputTokens":999},
                }), encoding="utf-8")

            def wait(self, timeout=None):
                return 0

            def terminate(self):
                return None

            def kill(self):
                return None

        with tempfile.TemporaryDirectory() as directory:
            cwd = Path(directory)
            log_path = cwd / "events.jsonl"
            args = argparse.Namespace(
                codex_executable="node.exe",
                codex_script="codex.js",
                codex_version="0.130.0",
                model="gpt-5.5",
                timeout_seconds=10,
            )
            with patch.object(runner, "resolve_codex", return_value=["node.exe", "codex.js"]), \
                 patch.object(runner, "check_version"), \
                 patch.object(runner.subprocess, "Popen", side_effect=FakeProcess):
                response = runner.launch_codex(value, args, {"model_reasoning_effort":"high"}, cwd, log_path)
            argv = captured["argv"]
            self.assertIn("--ignore-user-config", argv)
            self.assertIn("--ephemeral", argv)
            self.assertIn("--json", argv)
            self.assertIn("--sandbox", argv)
            self.assertIn("read-only", argv)
            self.assertIn("--disable", argv)
            self.assertIn("plugins", argv)
            self.assertEqual(argv[-1], "-")
            self.assertLess(argv.index("--config"), len(argv) - 1)
            self.assertNotIn("--add-dir", argv)
            self.assertEqual(response["usage"], {
                "source":"provider-reported", "inputTokens":7, "outputTokens":4, "toolCalls":1,
            })
            private_log = log_path.read_text(encoding="utf-8")
            self.assertIn('"type":"provider.stderr"', private_log)
            self.assertNotIn("private model catalog detail", private_log)
    def test_incomplete_wrapper_timeout_echoes_bound_invocation(self) -> None:
        response = runner.incomplete_response(invocation("infer"), "timeout", type("Collector", (), {"telemetry": lambda self: None})())
        self.assertEqual(response["outcome"], "incomplete")
        self.assertEqual(response["role"], "infer")
        self.assertEqual(response["inputDigest"], invocation("infer")["inputDigest"])


if __name__ == "__main__":
    unittest.main()
