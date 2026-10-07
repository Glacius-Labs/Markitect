import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch
import run as driver

from run import (
    CODEX_RUNNER_DIGEST,
    PROTOCOL,
    ProofError,
    array_value,
    assurance_node_count,
    assurance_nodes_availability,
    bounded_runner_args,
    new_attempt_paths,
    proposal_metrics,
    require_digest,
    require_first_script,
    require_runner_runtime_file,
    runner_flag_value,
    runtime_file_fact,
    run_metrics,
    safe_text_facts,
    sha,
    strict_bytes,
    summarize_receipt,
    validate_build_receipt,
    build_receipt_fields,
    write_new_bytes,
)


class EvidenceSafetyTests(unittest.TestCase):
    def test_current_protocol_freezes_actual_adapter_bytes(self):
        root = Path(__file__).resolve().parents[2]
        self.assertEqual(PROTOCOL, "operating-model-proof/v9")
        self.assertEqual(sha((root / "internal/tooling/codexrunner/runner.py").read_bytes()), CODEX_RUNNER_DIGEST)

    def test_partial_cli_report_preserves_array_availability(self):
        proposed = proposal_metrics({
            "status": "blocked",
            "plan": {
                "proposals": None,
                "observedPaths": None,
                "unknownArtifacts": None,
                "evidenceRefreshRequired": [],
            },
        })
        self.assertEqual(proposed["proposals"], [])
        self.assertEqual(proposed["proposalsAvailability"], "null")
        self.assertEqual(proposed["observedPaths"], [])
        self.assertEqual(proposed["observedPathsAvailability"], "null")
        self.assertEqual(proposed["unknownArtifacts"], [])
        self.assertEqual(proposed["unknownArtifactsAvailability"], "null")
        self.assertEqual(proposed["evidenceRefreshRequired"], [])
        self.assertEqual(proposed["evidenceRefreshRequiredAvailability"], "present")
        self.assertEqual(proposed["unobservedProjectionsAvailability"], "missing")

        verified = run_metrics("controller-verify", {
            "status": "blocked",
            "results": [],
            "verifierRuns": None,
            "assurance": None,
            "limits": None,
        })
        self.assertEqual(verified["results"], [])
        self.assertEqual(verified["resultsAvailability"], "present")
        self.assertEqual(verified["verifierRuns"], [])
        self.assertEqual(verified["verifierRunsAvailability"], "null")
        self.assertIsNone(verified["assuranceNodeCount"])
        self.assertIsNone(verified["assuranceDigest"])
        self.assertEqual(verified["limits"], [])
        self.assertEqual(verified["limitsAvailability"], "null")

    def test_array_availability_distinguishes_missing_null_and_empty(self):
        self.assertEqual(array_value({}, "items", "items"), ([], "missing"))
        self.assertEqual(array_value({"items": None}, "items", "items"), ([], "null"))
        self.assertEqual(array_value({"items": []}, "items", "items"), ([], "present"))

    def test_codex_runner_must_be_first_argument_and_runtime_bound(self):
        runner = Path(tempfile.gettempdir()) / "markitect" / "runner.py"
        args = [str(runner), "--codex-executable", "C:/codex.exe", "--codex-version", "0.130.0", "--model", "gpt-5.5"]
        self.assertEqual(bounded_runner_args(args, "executor"), args)
        require_first_script(args, runner, "executor")
        self.assertEqual(runner_flag_value(args, "--codex-executable", "executor"), "C:/codex.exe")
        require_runner_runtime_file([{"path": str(runner), "digest": "sha256:" + "a" * 64}], runner,
                                    "sha256:" + "a" * 64, "executor")
        substituted = ["--something", str(runner), *args[1:]]
        with self.assertRaises(ProofError):
            require_first_script(substituted, runner, "executor")
        with self.assertRaises(ProofError):
            require_runner_runtime_file([{"path": str(runner), "digest": "sha256:" + "b" * 64}], runner,
                                        "sha256:" + "a" * 64, "executor")

    def test_runtime_inspection_binds_the_executed_script_not_a_later_argument(self):
        wrapper_bytes = (driver.ROOT / "internal/tooling/codexrunner/runner.py").read_bytes()
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / "source"
            fixture = root / "fixture"
            fixture.mkdir()
            wrapper = source / "internal/tooling/codexrunner/runner.py"
            wrapper.parent.mkdir(parents=True)
            wrapper.write_bytes(wrapper_bytes)
            other = root / "other.py"
            other.write_bytes(b"unrelated script")
            python = root / "python.exe"
            python.write_bytes(b"offline test executable")
            codex = root / "codex.exe"
            codex.write_bytes(b"offline test provider; never executed")
            runner = {
                "command": str(python),
                "args": [str(wrapper), "--codex-executable", str(codex),
                         "--codex-version", "0.130.0", "--model", "gpt-5.5"],
                "model": "gpt-5.5", "providerVersion": "0.130.0",
                "modelOptions": {"model_reasoning_effort": "high"},
                "runtimeFiles": [{"path": str(wrapper), "digest": CODEX_RUNNER_DIGEST}],
            }
            config = {
                "apiVersion": "markitect.canonical/controller/v1alpha1",
                "recordStore": str(root / "store"), "privateLogs": str(root / "logs"),
                "executor": dict(runner), "verifier": dict(runner),
            }
            runtime = root / "runtime.json"
            with patch.object(driver, "ROOT", source), patch.object(driver.subprocess, "run") as process:
                process.return_value.stdout = b"Python 3.13.3\n"
                runtime.write_text(json.dumps(config), encoding="utf-8")
                result = driver.inspect_runtime(runtime, fixture)
                self.assertEqual(result["runtimeConfigDigest"], sha(runtime.read_bytes()))
                self.assertEqual([call.args[0] for call in process.call_args_list],
                                 [[str(python), "--version"], [str(python), "--version"]])
                # The old anywhere-in-args guard accepted this valid later mention.
                config["executor"]["args"] = [str(other), *runner["args"]]
                runtime.write_text(json.dumps(config), encoding="utf-8")
                with self.assertRaisesRegex(ProofError, "first script argument"):
                    driver.inspect_runtime(runtime, fixture)
                config["executor"]["args"] = runner["args"]
                config["executor"]["runtimeFiles"] = []
                runtime.write_text(json.dumps(config), encoding="utf-8")
                with self.assertRaisesRegex(ProofError, "exact repository Codex adapter"):
                    driver.inspect_runtime(runtime, fixture)
                self.assertTrue(all(call.args[0] == [str(python), "--version"]
                                    for call in process.call_args_list))

    def test_runner_flag_values_are_bounded_and_never_index_missing_values(self):
        for args in (["--model"], ["--model", "--codex-version"], ["--model", "x" * 513], ["--model", "x", "--model", "y"]):
            with self.subTest(args=args), self.assertRaises(ProofError):
                runner_flag_value(list(args), "--model", "executor")
        with self.assertRaises(ProofError):
            bounded_runner_args(["runner", 3], "executor")

    def test_apply_summarizes_exact_artifacts_and_source_evidence_revision(self):
        digest = "sha256:" + "d" * 64
        applied = run_metrics("controller-apply", {
            "status": "materialized-unverified",
            "runDigest": "sha256:" + "e" * 64,
            "evidenceRevision": "a" * 40,
            "records": [{
                "id": "sha256:" + "f" * 64,
                "projectionId": "projection",
                "artifacts": [{"path": "src/file.cs", "mode": "100644", "digest": digest, "change": "created"}],
            }],
            "written": [],
            "evidencePaths": None,
            "evidenceRefreshRequired": [],
        })
        self.assertEqual(applied["evidenceRevision"], "a" * 40)
        self.assertEqual(applied["records"][0]["artifacts"], [{"path": "src/file.cs", "mode": "100644", "digest": digest}])
        self.assertEqual(applied["records"][0]["artifactsAvailability"], "present")
        self.assertEqual(applied["writtenAvailability"], "present")
        self.assertEqual(applied["evidencePathsAvailability"], "null")
        self.assertEqual(applied["evidenceRefreshRequiredAvailability"], "present")

    def test_apply_keeps_record_artifact_array_availability(self):
        for record, expected in (({"artifacts": []}, "present"), ({"artifacts": None}, "null"), ({}, "missing")):
            with self.subTest(expected=expected):
                result = run_metrics("controller-apply", {"records": [record]})
                self.assertEqual(result["records"][0]["artifacts"], [])
                self.assertEqual(result["records"][0]["artifactsAvailability"], expected)

    def test_wrong_collection_shape_fails_closed(self):
        with self.assertRaises(ProofError):
            proposal_metrics({"plan": {"proposals": {"not": "an array"}}})

    def test_assurance_count_uses_documented_evaluation_nodes_shape(self):
        def count(assurance):
            return run_metrics("controller-verify", {"assurance": assurance})["assuranceNodeCount"]

        self.assertEqual(count({"Evaluation": {"Nodes": [{"NodeID": "root"}]}}), 1)
        self.assertEqual(count({"Evaluation": {"Nodes": []}}), 0)
        self.assertEqual(assurance_nodes_availability({"assurance": {"Evaluation": {"Nodes": []}}}), "present")
        self.assertEqual(assurance_nodes_availability({"assurance": {"Evaluation": {"Nodes": None}}}), "null")
        self.assertEqual(assurance_nodes_availability({}), "missing")
        self.assertEqual(assurance_nodes_availability({"assurance": None}), "null")
        self.assertIsNone(count({"Evaluation": {"Nodes": None}}))
        self.assertIsNone(count(None))
        self.assertIsNone(count({"unknown": {"nodes": [{"NodeID": "root"}]}}))
        with self.assertRaises(ProofError):
            count({"Evaluation": {"Nodes": "not-an-array"}})
        with self.assertRaises(ProofError):
            count([{"NodeID": "root"}])

    def test_runtime_wrapper_digest_must_match_frozen_bytes(self):
        expected = "sha256:" + "a" * 64
        require_digest(expected, expected, "Codex runner wrapper")
        with self.assertRaises(ProofError):
            require_digest("sha256:" + "b" * 64, expected, "Codex runner wrapper")

    def test_cli_report_rejects_duplicate_nested_keys(self):
        with self.assertRaises(ProofError):
            strict_bytes(b'{"status":"passed","nested":{"value":1,"value":2}}')

    def test_usage_fields_survive_receipt_sanitization(self):
        result = summarize_receipt({
            "runId": "actual-run",
            "privatePrompt": "must stay out",
            "usage": {
                "source": "codex",
                "inputTokens": 41,
                "outputTokens": 9,
                "cachedTokens": 3,
                "toolCalls": 2,
                "providerPayload": "must stay out",
            },
        })
        self.assertEqual(result["usage"], {
            "source": "codex",
            "inputTokens": 41,
            "outputTokens": 9,
            "cachedTokens": 3,
            "toolCalls": 2,
        })
        self.assertNotIn("privatePrompt", result)
        self.assertNotIn("providerPayload", result["usage"])

    def test_retry_uses_new_immutable_capture_paths(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            first_id, first_out, first_err = new_attempt_paths(root, "execute")
            second_id, second_out, second_err = new_attempt_paths(root, "execute")
            self.assertNotEqual(first_id, second_id)
            self.assertNotEqual(first_out, second_out)
            write_new_bytes(first_out, b"first candidate report")
            write_new_bytes(first_err, b"first error")
            write_new_bytes(second_out, b"corrected candidate report")
            write_new_bytes(second_err, b"corrected error")
            with self.assertRaises(FileExistsError):
                write_new_bytes(first_out, b"overwrite attempt")
            self.assertEqual(first_out.read_bytes(), b"first candidate report")
            self.assertEqual(second_out.read_bytes(), b"corrected candidate report")

    def test_build_receipt_binds_source_command_exit_and_binary(self):
        source_sha = "a" * 40
        binary_digest = "sha256:" + "b" * 64
        receipt = {
            "sourceSha": source_sha,
            "buildCommand": ["go", "build", "-o", "markitect.exe", "./cmd/markitect"],
            "exitCode": 0,
            "binaryDigest": binary_digest,
        }
        facts = validate_build_receipt(receipt, source_sha, binary_digest)
        self.assertEqual(facts["sourceSha"], source_sha)
        self.assertEqual(facts["binaryDigest"], binary_digest)
        self.assertTrue(facts["buildCommandDigest"].startswith("sha256:"))
        with self.assertRaises(ProofError):
            validate_build_receipt(receipt, source_sha, "sha256:" + "c" * 64)
        failed = dict(receipt, exitCode=1)
        with self.assertRaises(ProofError):
            validate_build_receipt(failed, source_sha, binary_digest)

    def test_captured_build_receipt_is_bound_without_rewriting(self):
        source = "a" * 40
        digest = "sha256:" + "b" * 64
        receipt = {"sourceCommit": source, "command": ["/fixed/go", "build", "./cmd/markitect"],
                   "exitCode": 0, "binaryDigest": digest, "sourceWorktreeCleanBeforeAndAfter": True}
        original = json.dumps(receipt, sort_keys=True)
        facts = validate_build_receipt(receipt, source, digest)
        self.assertEqual(facts["receiptFieldSchema"], "sourceCommit/command")
        self.assertEqual(facts["sourceSha"], source)
        self.assertEqual(json.dumps(receipt, sort_keys=True), original)
        with self.assertRaises(ProofError):
            validate_build_receipt(receipt, "c" * 40, digest)
        with self.assertRaises(ProofError):
            validate_build_receipt(dict(receipt, exitCode=True), source, digest)

    def test_build_receipt_rejects_mixed_missing_or_malformed_identity(self):
        source = "a" * 40
        for receipt in [{}, {"sourceCommit": source}, {"command": ["go", "build"]},
                        {"sourceCommit": "main", "command": ["go", "build"]},
                        {"sourceCommit": source, "command": []},
                        {"sourceSha": source, "buildCommand": ["go", "build"],
                         "sourceCommit": source, "command": ["go", "build"]},
                        {"sourceSha": source, "command": ["go", "build"]}]:
            with self.subTest(receipt=receipt), self.assertRaises(ProofError):
                build_receipt_fields(receipt)

    def test_runtime_file_binding_does_not_publish_absolute_path(self):
        path = Path(tempfile.gettempdir()) / "private-runtime" / "codex.exe"
        fact = runtime_file_fact(path, "sha256:" + "c" * 64, 123)
        serialized = json.dumps(fact)
        self.assertNotIn(str(path), serialized)
        self.assertEqual(fact["pathDigest"], sha(str(path.resolve()).encode()))
        self.assertEqual(fact["bytes"], 123)

    def test_verifier_details_are_retained_as_digest_only(self):
        facts = safe_text_facts([{
            "subject": "Checkout composition",
            "outcome": "failed",
            "detail": "private explanation text",
        }])
        self.assertEqual(facts[0]["subject"], "Checkout composition")
        self.assertEqual(facts[0]["outcome"], "failed")
        self.assertEqual(facts[0]["detailDigest"], sha(b"private explanation text"))
        self.assertNotIn("detail", facts[0])
        self.assertNotIn("private explanation text", json.dumps(facts))


if __name__ == "__main__":
    unittest.main()
