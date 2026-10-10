import argparse
import base64
import hashlib
import io
import json
import os
import stat
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import native_work
import runner


def item(path: str, content: bytes, mode: str = "0644", raw_digest: bool = False) -> dict:
    digest = hashlib.sha256(content).hexdigest()
    return {
        "path": path,
        "mode": mode,
        "digest": digest if raw_digest else "sha256:" + digest,
        "content": base64.b64encode(content).decode("ascii"),
    }


def task_invocation() -> dict:
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
            "scopeIds": ["manager-1", "src/main.py"],
            "policyIds": [],
            "context": {
                "kind": "projectrun-task/v1",
                "managerId": "manager-1",
                "phase": "work",
                "ownTask": "Implement the bounded change.",
                "globalGoal": "Complete the broader workflow.",
                "phaseGuidance": "Make the assigned change.",
                "allowedWritePaths": ["src/"],
                "excludedWritePaths": ["src/generated/"],
                "foreignOwnership": [{"path": "src/foreign/", "owner": "manager-other"}],
                "responseSchema": {
                    "type": "object",
                    "additionalProperties": False,
                    "required": ["status", "summary"],
                    "properties": {
                        "status": {"type": "string", "enum": ["complete", "partial"]},
                        "summary": {"type": "string", "minLength": 1},
                    },
                },
                "nativeWorkspace": {
                    "apiVersion": native_work.WORKSPACE_API,
                    "instructions": [item("AGENTS.md", b"Use the project's normal edit and test workflow.\n", "100644", raw_digest=True)],
                },
            },
            "artifacts": [item("src/main.py", b"def value():\n    return 1\n")],
        },
    }


def normalized_response(invocation: dict, candidate_files: list[dict]) -> dict:
    aliases = runner.evidence_ref_aliases(invocation)
    return {
        "apiVersion": invocation["apiVersion"],
        "runId": invocation["runId"],
        "nonce": invocation["nonce"],
        "role": "executor",
        "inputDigest": invocation["inputDigest"],
        "outcome": "proposed",
        "candidateFiles": candidate_files,
        "candidateJson": None,
        "reportJson": json.dumps({"status": "complete", "summary": "Implemented and checked."}),
        "evidenceRefs": [],
        "verifierObservations": [],
        "uncertainty": [],
    }


def prepare(root: Path, value: dict | None = None) -> native_work.PreparedWorkspace:
    return native_work.prepare(value or task_invocation(), root)


class NativeWorkspaceTests(unittest.TestCase):
    def test_materializes_verified_inputs_and_instruction_at_real_paths(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary) / "candidate"
            prepared = prepare(root)
            self.assertEqual((root / "AGENTS.md").read_bytes(), b"Use the project's normal edit and test workflow.\n")
            self.assertTrue((root / "src" / "main.py").is_file())
            self.assertTrue((root / ".markitect" / "manager-context.json").is_file())
            context_bytes = (root / ".markitect" / "manager-context.json").read_bytes()
            self.assertNotIn(b"responseSchema", context_bytes)
            self.assertEqual(prepared.instructions, frozenset({"AGENTS.md"}))
            self.assertTrue(prepared.base_digest.startswith("sha256:"))

    def test_harvest_uses_real_disk_delta_and_requires_matching_model_claim(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary) / "candidate"
            prepared = prepare(root)
            changed = b"def value():\n    return 2\n"
            (root / "src" / "main.py").write_bytes(changed)
            declared = [{"path": "src/main.py", "mode": "0644", "content": changed.decode()}]
            files, final_digest, delta_digest, paths = native_work.harvest(prepared, declared, tool_calls=3)
            self.assertEqual(files, declared)
            self.assertEqual(paths, ["src/main.py"])
            self.assertNotEqual(final_digest, prepared.base_digest)
            self.assertTrue(delta_digest.startswith("sha256:"))
            with self.assertRaisesRegex(native_work.SafeDeltaRejected, "disagrees"):
                native_work.harvest(prepared, [], tool_calls=3)

    def test_scope_exclusion_longest_foreign_owner_control_plane_and_deletion_block(self) -> None:
        cases = [
            ("src/generated/new.py", "explicitly excluded"),
            ("src/foreign/new.py", "foreign Manager"),
            ("docs/new.md", "outside allowedWritePaths"),
            (".git/config", "control-plane"),
            (".markitect/areas/model.yaml", "control-plane"),
        ]
        for path, message in cases:
            with self.subTest(path=path), tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary) / "candidate"
                prepared = prepare(root)
                target = root.joinpath(*path.split("/"))
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_text("candidate", encoding="utf-8")
                with self.assertRaisesRegex(native_work.SafeDeltaRejected, message):
                    native_work.harvest(prepared, [], tool_calls=1)

        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary) / "candidate"
            prepared = prepare(root)
            (root / "src" / "main.py").unlink()
            with self.assertRaisesRegex(native_work.SafeDeltaRejected, "deleted an existing input") as caught:
                native_work.harvest(prepared, [], tool_calls=0)
            self.assertIn("src/main.py", caught.exception.native_work["changedPaths"])

    def test_instruction_context_are_immutable_and_non_utf8_is_blocked(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary) / "candidate"
            prepared = prepare(root)
            (root / "AGENTS.md").write_text("changed", encoding="utf-8")
            with self.assertRaisesRegex(native_work.SafeDeltaRejected, "immutable instructions"):
                native_work.harvest(prepared, [], tool_calls=1)
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary) / "candidate"
            prepared = prepare(root)
            (root / "src" / "binary.dat").write_bytes(b"\xff\x00")
            with self.assertRaisesRegex(native_work.SafeDeltaRejected, "non-UTF-8"):
                native_work.harvest(prepared, [], tool_calls=1)

    def test_rejects_unsafe_paths_digest_collisions_and_links(self) -> None:
        for path in ("../escape", "C:/escape", "a\\b", ".markitect/manager-context.json"):
            with self.subTest(path=path), tempfile.TemporaryDirectory() as temporary:
                value = task_invocation()
                value["request"]["context"]["nativeWorkspace"]["instructions"] = [item(path, b"x", "100644", raw_digest=True)]
                with self.assertRaises(native_work.NativeWorkError):
                    prepare(Path(temporary) / "candidate", value)
        with tempfile.TemporaryDirectory() as temporary:
            value = task_invocation()
            value["request"]["context"]["nativeWorkspace"]["instructions"].append(item("agents.md", b"alias", "100644", raw_digest=True))
            with self.assertRaisesRegex(native_work.NativeWorkError, "case-alias"):
                prepare(Path(temporary) / "candidate", value)
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary) / "candidate"
            prepared = prepare(root)
            os.link(root / "src" / "main.py", root / "src" / "linked.py")
            with self.assertRaisesRegex(native_work.NativeWorkError, "hard link"):
                native_work.harvest(prepared, [], tool_calls=1)
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary) / "candidate"
            prepared = prepare(root)
            try:
                (root / "src" / "link.py").symlink_to(root / "src" / "main.py")
            except OSError:
                self.skipTest("Host filesystem does not permit creating a test symlink")
            with self.assertRaisesRegex(native_work.NativeWorkError, "link or non-regular"):
                native_work.harvest(prepared, [], tool_calls=1)

    def test_delta_limits_and_mode_changes_are_observed(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary) / "candidate"
            prepared = prepare(root)
            original_reader = native_work._read_candidate
            final = original_reader(prepared)
            original = final["src/main.py"]
            final["src/main.py"] = native_work.FileRecord(original.path, "0755", original.content, original.digest)
            expected = [{"path": "src/main.py", "mode": "0755", "content": original.content.decode("utf-8")}]
            with patch.object(native_work, "_read_candidate", return_value=final):
                files, _, _, paths = native_work.harvest(prepared, expected, tool_calls=0)
            self.assertEqual(files, expected)
            self.assertEqual(paths, ["src/main.py"])
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary) / "candidate"
            prepared = prepare(root)
            (root / "src" / "large.bin").write_bytes(b"x" * (native_work.MAX_BYTES + 1))
            with self.assertRaisesRegex(native_work.NativeWorkError, "scan size bound"):
                native_work.harvest(prepared, [], tool_calls=0)

    def test_posix_mode_classifier_normalizes_regular_files_like_git(self) -> None:
        cases = (
            (0o644, "0644"), (0o755, "0755"), (0o640, "0644"), (0o700, "0755"),
            (0o4755, "0755"), (0o664, "0644"), (0o775, "0755"), (0o600, "0644"),
        )
        for mode_bits, expected in cases:
            with self.subTest(mode=f"{mode_bits:04o}"):
                self.assertEqual(native_work._posix_mode(mode_bits, None), expected)
        private = native_work.FileRecord("src/main.py", "0600", b"", native_work._digest(b""))
        self.assertEqual(native_work._posix_mode(0o600, private), "0600")
        self.assertEqual(native_work._posix_mode(0o640, private), "0644")

    @unittest.skipUnless(hasattr(os, "mkfifo"), "POSIX FIFOs only")
    def test_non_regular_candidate_file_is_still_rejected(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary) / "candidate"
            prepared = prepare(root)
            os.mkfifo(root / "src" / "pipe")
            with self.assertRaisesRegex(native_work.NativeWorkError, "link or non-regular file: src/pipe"):
                native_work.harvest(prepared, [], tool_calls=1)

    def test_unchanged_private_input_is_allowed_but_private_delta_is_rejected(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            value = task_invocation()
            value["request"]["artifacts"][0] = item("src/main.py", b"def value():\n    return 1\n", "0600")
            root = Path(temporary) / "candidate"
            prepared = prepare(root, value)
            self.assertEqual(native_work.harvest(prepared, [], tool_calls=0)[0], [])
            target = root / "src" / "main.py"
            target.write_text("def value():\n    return 2\n", encoding="utf-8")
            target.chmod(0o600)
            with self.assertRaisesRegex(native_work.SafeDeltaRejected, "unsupported repository mode 0600") as caught:
                native_work.harvest(prepared, [], tool_calls=1)
            self.assertEqual(caught.exception.native_work["changedPaths"], ["src/main.py"])

        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary) / "candidate"
            prepared = prepare(root)
            target = root / "src" / "private.py"
            target.write_text("secret = True\n", encoding="utf-8")
            private_content = b"secret = True\n"
            final = native_work._read_candidate(prepared)
            final["src/private.py"] = native_work.FileRecord(
                "src/private.py", "0600", private_content, native_work._digest(private_content)
            )
            with self.assertRaisesRegex(native_work.SafeDeltaRejected, "unsupported repository mode 0600"):
                with patch.object(native_work, "_read_candidate", return_value=final):
                    native_work.harvest(prepared, [], tool_calls=1)

    @unittest.skipUnless(os.name == "posix", "POSIX file modes only")
    def test_chmod_only_change_is_a_delta_only_when_the_git_mode_changes(self) -> None:
        content = "def value():\n    return 1\n"
        for chmod, delta_mode in ((0o664, None), (0o640, None), (0o600, None), (0o775, "0755"), (0o700, "0755")):
            with self.subTest(mode=f"{chmod:04o}"), tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary) / "candidate"
                prepared = prepare(root)
                (root / "src" / "main.py").chmod(chmod)
                expected = [] if delta_mode is None else [{"path": "src/main.py", "mode": delta_mode, "content": content}]
                files, _, _, paths = native_work.harvest(prepared, expected, tool_calls=0)
                self.assertEqual(files, expected)
                self.assertEqual(paths, [entry["path"] for entry in expected])


class NativeRunnerTests(unittest.TestCase):
    def test_prompt_and_cli_enable_native_file_shell_and_test_work_without_helper_launch(self) -> None:
        value = task_invocation()
        prompt = runner.make_prompt(value)
        self.assertIn("fresh scoped candidate workspace", prompt)
        self.assertIn("available shell", prompt)
        self.assertIn("run relevant tests", prompt)
        self.assertIn("Native helper agents are disabled", prompt)
        self.assertNotIn("do not invoke tools", prompt)
        self.assertIn("source repository", prompt)

        args = argparse.Namespace(
            native_helper_limit=0, codex_profile="luna-high",
            model="gpt-6-luna", codex_executable="codex", codex_script="", codex_version="0.162.0",
            timeout_seconds=20,
        )
        candidate = b"def value():\n    return 2\n"
        response_path = Path("unused")
        response = normalized_response(value, [{"path": "src/main.py", "mode": "0644", "content": candidate.decode()}])
        last_argv: list[str] = []
        read_instruction = ""

        class FakeProcess:
            def __init__(self, argv, cwd, **kwargs):
                nonlocal last_argv, read_instruction
                last_argv = argv
                self.cwd = Path(cwd)
                read_instruction = (self.cwd / "AGENTS.md").read_text(encoding="utf-8")
                (self.cwd / "src" / "main.py").write_bytes(candidate)
                output_path = Path(argv[argv.index("--output-last-message") + 1])
                output_path.write_text(json.dumps(response), encoding="utf-8")
                self.stdin = io.BytesIO()
                self.stdout = io.BytesIO(b'{"type":"item.started","item":{"type":"command_execution"}}\n{"type":"turn.completed","usage":{"input_tokens":12,"output_tokens":4}}\n')
                self.stderr = io.BytesIO()

            def wait(self, timeout=None):
                return 0

            def terminate(self):
                return None

            def kill(self):
                return None

        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            log = root / "events.jsonl"
            with patch.object(runner, "resolve_codex", return_value=["codex"]), patch.object(runner, "check_version"), patch.object(runner.subprocess, "Popen", FakeProcess):
                result = runner.launch_codex(value, args, {"model_reasoning_effort": "high"}, root, log)
            self.assertNotIn("--sandbox", last_argv)
            self.assertIn("--profile", last_argv)
            self.assertIn("luna-high", last_argv)
            self.assertNotIn("--ephemeral", last_argv)
            for disabled in ("plugins", "shell_tool", "unified_exec"):
                self.assertNotIn(disabled, last_argv)
            self.assertIn("multi_agent", last_argv)
            self.assertIn("normal edit and test workflow", read_instruction)
            self.assertEqual(result["candidateFiles"], [{"path": "src/main.py", "mode": "0644", "content": candidate.decode()}])
            self.assertEqual(result["nativeWork"]["toolCalls"], 1)
            self.assertEqual(result["nativeWork"]["helperStarts"], 0)
            self.assertEqual(result["nativeWork"]["helperAccounting"], "disabled")
            self.assertNotIn("toolCalls", result["usage"])

    @unittest.skipUnless(os.name == "posix", "POSIX file modes only")
    def test_new_file_modes_are_proposed_as_git_modes_under_any_umask(self) -> None:
        value = task_invocation()
        content = "def added():\n    return 3\n"
        args = argparse.Namespace(
            native_helper_limit=0, codex_profile="luna-high",
            model="gpt-6-luna", codex_executable="codex", codex_script="", codex_version="0.162.0",
            timeout_seconds=20,
        )
        # umask 002, common on desktop Linux, gives new files 0664 and new
        # scripts 0775; umask 077 gives private 0600 files.
        cases = ((0o022, 0o666, 0o644, "0644"), (0o002, 0o666, 0o664, "0644"), (0o002, 0o777, 0o775, "0755"), (0o077, 0o666, 0o600, "0644"))
        for umask, requested_mode, observed_mode, git_mode in cases:
            observed: list[int] = []
            response = normalized_response(value, [{"path": "src/added.py", "mode": git_mode, "content": content}])

            class FakeProcess:
                def __init__(self, argv, cwd, **kwargs):
                    previous = os.umask(umask)
                    try:
                        target = Path(cwd) / "src" / "added.py"
                        with os.fdopen(os.open(target, os.O_WRONLY | os.O_CREAT | os.O_EXCL, requested_mode), "w", encoding="utf-8") as stream:
                            stream.write(content)
                    finally:
                        os.umask(previous)
                    observed.append(stat.S_IMODE(target.stat().st_mode))
                    Path(argv[argv.index("--output-last-message") + 1]).write_text(json.dumps(response), encoding="utf-8")
                    self.stdin = io.BytesIO()
                    self.stdout = io.BytesIO(b'{"type":"turn.completed","usage":{"input_tokens":1,"output_tokens":1}}\n')
                    self.stderr = io.BytesIO()

                def wait(self, timeout=None):
                    return 0

                def terminate(self):
                    return None

                def kill(self):
                    return None

            with self.subTest(umask=f"{umask:03o}", mode=f"{observed_mode:04o}"), tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary)
                log = root / "events.jsonl"
                with patch.object(runner, "resolve_codex", return_value=["codex"]), patch.object(runner, "check_version"), patch.object(runner.subprocess, "Popen", FakeProcess):
                    result = runner.launch_codex(value, args, {"model_reasoning_effort": "high"}, root, log)
                self.assertEqual(observed, [observed_mode])
                self.assertEqual(result["outcome"], "proposed")
                self.assertEqual(result["candidateFiles"], [{"path": "src/added.py", "mode": git_mode, "content": content}])
                self.assertEqual(result["nativeWork"]["changedPaths"], ["src/added.py"])
                # The Host's agentexec nativeDeltaDigest encoding of the proposed candidateFiles.
                go_delta = '[{"digest":"sha256:%s","mode":"%s","path":"src/added.py"}]' % (hashlib.sha256(content.encode()).hexdigest(), git_mode)
                self.assertEqual(result["nativeWork"]["deltaDigest"], "sha256:" + hashlib.sha256(go_delta.encode()).hexdigest())

    def test_helpers_fail_before_provider_and_native_selection_comes_from_task_context(self) -> None:
        value = task_invocation()
        args = argparse.Namespace(
            native_helper_limit=1, codex_profile="luna-high",
            model="gpt-6-luna", codex_executable="codex", codex_script="", codex_version="0.162.0",
            timeout_seconds=20,
        )
        with tempfile.TemporaryDirectory() as temporary:
            with patch.object(runner.subprocess, "Popen") as spawn:
                with self.assertRaisesRegex(runner.AdapterError, "lifecycle and resource accounting"):
                    runner.launch_codex(value, args, {"model_reasoning_effort": "high"}, Path(temporary), Path(temporary) / "log")
                spawn.assert_not_called()
        self.assertTrue(runner.is_projectrun_manager_task(value))

    def test_task_context_with_missing_inconsistent_or_unsupported_manifest_fails_before_launch(self) -> None:
        args = argparse.Namespace(
            native_helper_limit=0, codex_profile="luna-high", model="gpt-6-luna",
            codex_executable="codex", codex_script="", codex_version="0.162.0", timeout_seconds=20,
        )
        cases = [
            ("missing manifest", lambda context: context.pop("nativeWorkspace"), "requires a valid Host nativeWorkspace"),
            ("wrong api", lambda context: context["nativeWorkspace"].update(apiVersion="wrong/v1"), "unsupported or malformed"),
            ("unsupported phase", lambda context: context.update(phase="verify"), "requires a work, integrate, or repair phase"),
            ("inconsistent task context", lambda context: context.update(kind="projectrun-review/v1"), "supported only for projectrun Manager executor tasks"),
        ]
        for label, mutate, message in cases:
            with self.subTest(label=label), tempfile.TemporaryDirectory() as temporary:
                value = task_invocation()
                mutate(value["request"]["context"])
                with patch.object(runner, "resolve_codex") as resolve, patch.object(runner.subprocess, "Popen") as spawn:
                    with self.assertRaisesRegex(runner.AdapterError, message):
                        runner.launch_codex(value, args, {"model_reasoning_effort": "high"}, Path(temporary), Path(temporary) / "log")
                    resolve.assert_not_called()
                    spawn.assert_not_called()

    def test_inference_context_keeps_the_read_only_cli_route(self) -> None:
        value = task_invocation()
        value["request"]["role"] = "infer"
        value["request"]["context"] = {"prompt": "Infer the closed candidate."}
        value["request"]["artifacts"] = []
        response = {
            "apiVersion": value["apiVersion"], "runId": value["runId"], "nonce": value["nonce"],
            "role": "infer", "inputDigest": value["inputDigest"], "outcome": "proposed",
            "candidateFiles": [], "candidateJson": "{}", "reportJson": None,
            "evidenceRefs": [], "verifierObservations": [], "uncertainty": [],
        }
        argv_seen: list[str] = []

        class FakeProcess:
            def __init__(self, argv, cwd, **kwargs):
                nonlocal argv_seen
                argv_seen = argv
                output_path = Path(argv[argv.index("--output-last-message") + 1])
                output_path.write_text(json.dumps(response), encoding="utf-8")
                self.stdin = io.BytesIO()
                self.stdout = io.BytesIO(b"")
                self.stderr = io.BytesIO()

            def wait(self, timeout=None):
                return 0

            def terminate(self):
                return None

            def kill(self):
                return None

        args = argparse.Namespace(
            native_helper_limit=0, model="gpt-6-luna", codex_executable="codex", codex_script="",
            codex_version="0.162.0", timeout_seconds=20,
        )
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            with patch.object(runner, "resolve_codex", return_value=["codex"]), patch.object(runner, "check_version"), patch.object(runner.subprocess, "Popen", FakeProcess):
                result = runner.launch_codex(value, args, {"model_reasoning_effort": "high"}, root, root / "events.jsonl")
        self.assertIn("read-only", argv_seen)
        self.assertIn("--ephemeral", argv_seen)
        self.assertNotIn("--profile", argv_seen)
        self.assertNotIn("nativeWork", result)
        self.assertEqual(result["candidateJson"], {})

    def test_native_workspace_marker_on_non_manager_roles_fails_before_cli_resolution(self) -> None:
        args = argparse.Namespace(
            native_helper_limit=0, codex_profile="luna-high", model="gpt-6-luna",
            codex_executable="codex", codex_script="", codex_version="0.162.0", timeout_seconds=20,
        )
        for role, kind in (("verifier", "projectrun-review/v1"), ("infer", "inference/v1")):
            with self.subTest(role=role), tempfile.TemporaryDirectory() as temporary:
                value = task_invocation()
                value["request"]["role"] = role
                value["request"]["context"]["kind"] = kind
                with patch.object(runner, "resolve_codex") as resolve, patch.object(runner.subprocess, "Popen") as spawn:
                    with self.assertRaisesRegex(runner.AdapterError, "only for projectrun Manager executor tasks"):
                        runner.launch_codex(value, args, {"model_reasoning_effort": "high"}, Path(temporary), Path(temporary) / "log")
                    resolve.assert_not_called()
                    spawn.assert_not_called()


if __name__ == "__main__":
    unittest.main()
