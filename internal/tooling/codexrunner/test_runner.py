import base64
import argparse
import hashlib
import io
import json
import time
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

    def test_prompt_states_closed_wire_response_contract(self) -> None:
        prompt = runner.make_prompt(invocation("verifier"))
        for instruction in (
            "Copy apiVersion, runId, nonce, and inputDigest exactly from the invocation envelope into the response; copy role exactly from invocation.request.role",
            "Always include candidateFiles, evidenceRefs, verifierObservations, and uncertainty as arrays",
            "exact strings supplied in request.scopeIds, request.policyIds, or request.artifacts[].path",
            "Do not use digests, hashes, labels, paraphrases, or derived values as evidence references",
            "Do not duplicate references; list them in lexicographic order",
            "exactly one verifierObservations entry for each supplied scopeIds and policyIds value",
            "using that exact value as subject",
            "incomplete or escalated",
        ):
            with self.subTest(instruction=instruction):
                self.assertIn(instruction, prompt)

    def test_verifier_prompt_requires_complete_sorted_unique_request_refs(self) -> None:
        value = invocation("verifier")
        value["request"]["scopeIds"] = ["scope/shared", "scope/z"]
        value["request"]["policyIds"] = ["policy/a", "scope/shared"]
        value["request"]["artifacts"] = [
            {"path": "source/file.cs", "mode": "0644", "digest": "unused", "content": ""},
            {"path": "checks/fixed-input.yaml", "mode": "0644", "digest": "unused", "content": ""},
        ]

        prompt = runner.make_prompt(value)

        expected_refs = [
            "checks/fixed-input.yaml",
            "policy/a",
            "scope/shared",
            "scope/z",
            "source/file.cs",
        ]
        encoded_refs = json.dumps(expected_refs, ensure_ascii=False, separators=(",", ":"))
        self.assertIn("evidenceRefs must equal the complete sorted unique union", prompt)
        self.assertIn("including fixed-check input artifact paths", prompt)
        self.assertIn("The exact required list is " + encoded_refs, prompt)
        self.assertIn("Listing a reference is protocol bookkeeping", prompt)

    def test_typed_observation_subjects_preserve_separate_evidence_references(self) -> None:
        value = invocation("verifier")
        entries = [
            {"subject": "artifact:src/file.cs", "kind": "artifact", "id": "src/file.cs"},
            {"subject": "check:opaque-tuple", "kind": "check", "id": "behavior", "version": "v1", "digest": "sha256:" + "a" * 64},
            {"subject": "scope:scope/example", "kind": "scope", "id": "scope/example"},
        ]
        value["request"]["context"]["requiredObservationSubjects"] = entries
        prompt = runner.make_prompt(value)
        self.assertIn('The exact required observation subjects are ["artifact:src/file.cs","check:opaque-tuple","scope:scope/example"]', prompt)
        self.assertIn("Overall passed requires exactly one passed observation per subject", prompt)
        self.assertIn("Failed, incomplete, or escalated may retain a partial set", prompt)
        self.assertIn("Never place these typed observation identities in evidenceRefs", prompt)
        self.assertIn('The exact required list is ["scope/example"]', prompt)
        self.assertNotIn("using that exact value as subject", prompt)

    def test_invalid_typed_subjects_refuse_before_provider_or_workspace_writes(self) -> None:
        entry = {"subject": "scope:scope/example", "kind": "scope", "id": "scope/example"}
        invalid = [None, [], [entry, entry], [entry] * 129, [{"subject": "scope:x"}],
                   [{**entry, "subject": "x" * 4097}], [{**entry, "unexpected": "x"}]]
        for entries in invalid:
            with self.subTest(entries=entries), tempfile.TemporaryDirectory() as directory:
                value = invocation("verifier")
                value["request"]["context"]["requiredObservationSubjects"] = entries
                cwd = Path(directory)
                with patch.object(runner, "resolve_codex") as resolve, patch.object(runner.subprocess, "Popen") as spawn:
                    with self.assertRaises(runner.AdapterError):
                        runner.launch_codex(value, argparse.Namespace(), {}, cwd, cwd / "events.jsonl")
                    resolve.assert_not_called()
                    spawn.assert_not_called()
                    self.assertEqual(list(cwd.iterdir()), [])

    def test_executor_and_inference_may_cite_relevant_subsets(self) -> None:
        for role in ("executor", "infer"):
            with self.subTest(role=role):
                prompt = runner.make_prompt(invocation(role))
                self.assertIn("include only relevant exact references", prompt)
                self.assertNotIn("evidenceRefs must equal the complete sorted unique union", prompt)

    def test_role_instructions_list_role_specific_outcomes(self) -> None:
        expected = {
            "executor": ("proposed, failed, incomplete, or escalated", "proposed requires at least one candidate file"),
            "verifier": ("passed, failed, incomplete, or escalated", "passed and failed require concrete verifier observations"),
            "infer": ("proposed, failed, incomplete, or escalated", "proposed requires a JSON object candidate"),
        }
        for role, phrases in expected.items():
            with self.subTest(role=role):
                instructions = runner.role_instructions(role)
                for phrase in phrases:
                    self.assertIn(phrase, instructions)

    def test_explicit_model_options_become_literal_codex_config_arguments(self) -> None:
        self.assertEqual(
            runner.model_config_args({"model_reasoning_effort": "high"}),
            ["--config", 'model_reasoning_effort="high"'],
        )

    def test_inference_candidate_uses_closed_string_transport_and_is_parsed(self) -> None:
        self.assertEqual(runner.RESPONSE_SCHEMA["properties"]["candidateJson"]["type"], ["string", "null"])
        self.assertIn("candidateJson", runner.RESPONSE_SCHEMA["required"])
        response = runner.normalize_codex_response({"candidateJson": '{"proposal":{"value":1}}'})
        self.assertEqual(response["candidateJson"], {"proposal": {"value": 1}})
        self.assertNotIn("candidateJson", runner.normalize_codex_response({"candidateJson": None}))
        with self.assertRaises(runner.AdapterError):
            runner.normalize_codex_response({"candidateJson": "[]"})

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
        value["request"]["context"] = {"privatePromptSentinel": "DO_NOT_LOG_PROMPT_CONTENT"}
        captured = {}

        class CapturingStdin(io.BytesIO):
            def __init__(self):
                super().__init__()
                self.submitted = bytearray()
                self.closed_after_flush = False
                self.flushed = False

            def write(self, data):
                self.submitted.extend(data)
                return super().write(data)

            def flush(self):
                self.flushed = True
                return super().flush()

            def close(self):
                self.closed_after_flush = self.flushed
                return super().close()

        class FakeProcess:
            def __init__(self, argv, **kwargs):
                captured["argv"] = argv
                self.stdin = CapturingStdin()
                captured["stdin"] = self.stdin
                self.stdout = io.BytesIO(
                    b'{"type":"thread.started","thread_id":"fresh-thread"}\n'
                    b'{"type":"turn.started"}\n'
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
                    "candidateJson": None,
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
            self.assertNotIn("candidateJson", response)
            self.assertEqual(response["usage"], {
                "source":"provider-reported", "inputTokens":7, "outputTokens":4, "toolCalls":1,
            })
            private_log = log_path.read_text(encoding="utf-8")
            self.assertIn('"type":"provider.stderr"', private_log)
            self.assertNotIn("private model catalog detail", private_log)
            self.assertNotIn("DO_NOT_LOG_PROMPT_CONTENT", private_log)
            self.assertNotIn(value["nonce"], private_log)
            self.assertEqual(bytes(captured["stdin"].submitted), runner.make_prompt(value).encode("utf-8"))
            self.assertTrue(captured["stdin"].closed_after_flush)
            prompt_events = [json.loads(line) for line in private_log.splitlines() if '"type":"adapter.prompt-submitted"' in line]
            self.assertEqual(prompt_events, [{
                "type": "adapter.prompt-submitted",
                "runId": value["runId"],
                "inputDigest": value["inputDigest"],
                "promptSha256": "sha256:" + hashlib.sha256(runner.make_prompt(value).encode("utf-8")).hexdigest(),
                "promptBytes": len(runner.make_prompt(value).encode("utf-8")),
            }])
            self.assertIn('"type":"thread.started"', private_log)
            self.assertIn('"type":"turn.started"', private_log)

    def test_codex_launch_does_not_mark_broken_or_partial_prompt_delivery(self) -> None:
        value = invocation()
        processes = []

        class FakeStdin(io.BytesIO):
            def __init__(self, behavior: str):
                super().__init__()
                self.behavior = behavior

            def write(self, data):
                if self.behavior == "broken":
                    raise BrokenPipeError("closed child input")
                if self.behavior == "partial":
                    super().write(data[:-1])
                    return len(data) - 1
                return super().write(data)

            def flush(self):
                if self.behavior == "flush-error":
                    raise OSError("flush failed")
                return super().flush()

            def close(self):
                if self.behavior == "close-error":
                    super().close()
                    raise OSError("close failed")
                return super().close()

        for behavior in ("broken", "partial", "flush-error", "close-error"):
            with self.subTest(behavior=behavior), tempfile.TemporaryDirectory() as directory:
                cwd = Path(directory)
                log_path = cwd / "events.jsonl"
                args = argparse.Namespace(
                    codex_executable="codex.exe",
                    codex_script=None,
                    codex_version="0.130.0",
                    model="gpt-5.5",
                    timeout_seconds=10,
                )

                class FakeProcess:
                    def __init__(self, argv, **kwargs):
                        self.stdin = FakeStdin(behavior)
                        self.stdout = io.BytesIO()
                        self.stderr = io.BytesIO()
                        response_path = Path(argv[argv.index("--output-last-message") + 1])
                        response_path.write_text("{}", encoding="utf-8")
                        self.terminated = False
                        processes.append(self)

                    def wait(self, timeout=None):
                        return 0

                    def terminate(self):
                        self.terminated = True

                    def kill(self):
                        return None

                with patch.object(runner, "resolve_codex", return_value=["codex.exe"]), \
                     patch.object(runner, "check_version"), \
                     patch.object(runner.subprocess, "Popen", side_effect=FakeProcess):
                    with self.assertRaisesRegex(runner.AdapterError, "prompt could not be submitted"):
                        runner.launch_codex(value, args, {}, cwd, log_path)

                self.assertTrue(processes[-1].terminated)
                log = log_path.read_text(encoding="utf-8")
                self.assertNotIn("adapter.prompt-submitted", log)
                self.assertNotIn(value["nonce"], log)

    def test_codex_timeout_bounds_large_prompt_to_child_that_never_reads_stdin(self) -> None:
        value = invocation()
        value["request"]["context"] = {"largeBoundedPrompt": "x" * (1024 * 1024)}
        with tempfile.TemporaryDirectory() as directory:
            cwd = Path(directory)
            script = cwd / "nonreading-codex.py"
            script.write_text("import time\ntime.sleep(30)\n", encoding="utf-8")
            args = argparse.Namespace(
                codex_executable=runner.sys.executable,
                codex_script=str(script),
                codex_version="0.130.0",
                model="gpt-5.5",
                timeout_seconds=1,
            )
            started = time.monotonic()
            with patch.object(runner, "resolve_codex", return_value=[runner.sys.executable, str(script)]), \
                 patch.object(runner, "check_version"):
                response = runner.launch_codex(value, args, {}, cwd, cwd / "events.jsonl")
            elapsed = time.monotonic() - started

            self.assertEqual(response["outcome"], "incomplete")
            self.assertLess(elapsed, 4, f"prompt delivery exceeded the configured timeout: {elapsed:.2f}s")
            private_log = (cwd / "events.jsonl").read_text(encoding="utf-8")
            self.assertNotIn("adapter.prompt-submitted", private_log)

    def test_incomplete_wrapper_timeout_echoes_bound_invocation(self) -> None:
        response = runner.incomplete_response(invocation("infer"), "timeout", type("Collector", (), {"telemetry": lambda self: None})())
        self.assertEqual(response["outcome"], "incomplete")
        self.assertEqual(response["role"], "infer")
        self.assertEqual(response["inputDigest"], invocation("infer")["inputDigest"])


if __name__ == "__main__":
    unittest.main()
