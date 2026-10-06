import json
import tempfile
import unittest
from pathlib import Path

from run import (
    ProofError,
    assurance_node_count,
    new_attempt_paths,
    proposal_metrics,
    require_digest,
    runtime_file_fact,
    run_metrics,
    safe_text_facts,
    sha,
    strict_bytes,
    summarize_receipt,
    validate_build_receipt,
    write_new_bytes,
)


class EvidenceSafetyTests(unittest.TestCase):
    def test_partial_cli_report_null_collections_are_empty(self):
        proposed = proposal_metrics({
            "status": "blocked",
            "plan": {
                "proposals": None,
                "observedPaths": None,
                "unknownArtifacts": None,
            },
        })
        self.assertEqual(proposed["proposals"], [])
        self.assertEqual(proposed["observedPaths"], [])
        self.assertEqual(proposed["unknownArtifacts"], [])

        verified = run_metrics("controller-verify", {
            "status": "blocked",
            "results": None,
            "verifierRuns": None,
            "assurance": None,
            "limits": None,
        })
        self.assertEqual(verified["results"], [])
        self.assertEqual(verified["verifierRuns"], [])
        self.assertIsNone(verified["assuranceNodeCount"])
        self.assertIsNone(verified["assuranceDigest"])
        self.assertEqual(verified["limits"], [])

    def test_wrong_collection_shape_fails_closed(self):
        with self.assertRaises(ProofError):
            proposal_metrics({"plan": {"proposals": {"not": "an array"}}})

    def test_assurance_count_uses_documented_evaluation_nodes_shape(self):
        def count(assurance):
            return run_metrics("controller-verify", {"assurance": assurance})["assuranceNodeCount"]

        self.assertEqual(count({"Evaluation": {"Nodes": [{"NodeID": "root"}]}}), 1)
        self.assertEqual(count({"Evaluation": {"Nodes": []}}), 0)
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
