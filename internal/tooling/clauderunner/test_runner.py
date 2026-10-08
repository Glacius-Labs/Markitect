import argparse
import io
import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import runner


def invocation() -> dict:
    return {
        "apiVersion": "markitect.example.org/agent-execution/v1alpha1",
        "runId": "a" * 32,
        "nonce": "b" * 32,
        "inputDigest": "sha256:" + "c" * 64,
        "request": {
            "role": "executor",
            "sourceRevision": "d" * 40,
            "modelDigest": "sha256:" + "e" * 64,
            "modulePin": "module@sha256:" + "f" * 64,
            "projectionId": "projection/example",
            "scopeIds": ["scope/example"],
            "policyIds": [],
            "context": {"privatePromptSentinel": "DO_NOT_LOG_PROMPT_CONTENT"},
            "artifacts": [],
        },
    }


class ClaudeRunnerTests(unittest.TestCase):
    def test_strict_json_rejects_duplicate_keys(self) -> None:
        with self.assertRaises(runner.AdapterError):
            runner.strict_loads('{"outer":{"x":1,"x":2}}')

    def test_invocation_rejects_unknown_request_fields(self) -> None:
        value = invocation()
        value["request"]["transcript"] = "private"
        with self.assertRaises(runner.AdapterError):
            runner.validate_invocation(value)

    def test_claude_response_requires_structured_output_and_parses_candidate_json(self) -> None:
        response = {
            "apiVersion": "markitect.example.org/agent-execution/v1alpha1",
            "runId": "a" * 32,
            "nonce": "b" * 32,
            "role": "infer",
            "inputDigest": "sha256:" + "c" * 64,
            "outcome": "proposed",
            "candidateFiles": [],
            "candidateJson": '{"proposal":{"value":1}}',
            "evidenceRefs": [],
            "verifierObservations": [],
            "uncertainty": ["inferred"],
        }
        normalized = runner.normalize_claude_response({"structured_output": response})
        self.assertEqual(normalized["candidateJson"], {"proposal": {"value": 1}})
        with self.assertRaises(runner.AdapterError):
            runner.normalize_claude_response({"result": "unstructured"})

    def test_version_requires_restricted_mode_minimum_and_exact_match(self) -> None:
        with self.assertRaises(runner.AdapterError):
            runner.check_version(["claude"], "2.1.247")
        with patch.object(runner.subprocess, "run", return_value=type("R", (), {"returncode": 0, "stdout": b"2.1.2480", "stderr": b""})()):
            with self.assertRaises(runner.AdapterError):
                runner.check_version(["claude"], "2.1.248")

    def test_launch_uses_closed_no_tools_flags_and_private_output_log(self) -> None:
        value = invocation()
        captured = {}
        structured = {
            "apiVersion": value["apiVersion"],
            "runId": value["runId"],
            "nonce": value["nonce"],
            "role": "executor",
            "inputDigest": value["inputDigest"],
            "outcome": "proposed",
            "candidateFiles": [{"path": "candidate.txt", "mode": "0644", "content": "candidate"}],
            "candidateJson": None,
            "evidenceRefs": ["scope/example"],
            "verifierObservations": [],
            "uncertainty": [],
        }

        class FakeProcess:
            def __init__(self, argv, **kwargs):
                captured["argv"] = argv
                captured["kwargs"] = kwargs
                self.stdin = io.BytesIO()
                self.stdout = io.BytesIO(json.dumps({"structured_output": structured}).encode("utf-8"))
                self.stderr = io.BytesIO(b"private provider diagnostic")

            def wait(self, timeout=None):
                return 0

            def terminate(self):
                return None

            def kill(self):
                return None

        with tempfile.TemporaryDirectory() as directory:
            cwd = Path(directory)
            log_path = cwd / "private.jsonl"
            args = argparse.Namespace(
                claude_executable="C:/managed/claude.exe",
                claude_version="2.1.248",
                model="sonnet",
                timeout_seconds=15,
            )
            with patch.object(runner, "resolve_claude", return_value=[args.claude_executable]), \
                 patch.object(runner, "check_version"), \
                 patch.object(runner.subprocess, "Popen", FakeProcess):
                result = runner.launch_claude(value, args, {"effort": "high"}, cwd, log_path)
            argv = captured["argv"]
            self.assertIn("--restricted", argv)
            self.assertIn("--permission-mode", argv)
            self.assertIn("dontAsk", argv)
            self.assertEqual(argv[argv.index("--tools") + 1], "")
            self.assertIn("mcp__*", argv)
            self.assertIn("--json-schema", argv)
            self.assertEqual(argv[argv.index("--effort") + 1], "high")
            self.assertFalse(captured["kwargs"]["shell"])
            self.assertEqual(result["candidateFiles"][0]["path"], "candidate.txt")
            log = log_path.read_text(encoding="utf-8")
            self.assertIn("adapter.prompt-submitted", log)
            self.assertIn("provider.stderr", log)
            self.assertNotIn("DO_NOT_LOG_PROMPT_CONTENT", log)
            self.assertNotIn(value["nonce"], log)


if __name__ == "__main__":
    unittest.main()
