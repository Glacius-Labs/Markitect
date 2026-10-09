"""Provider-free tests for the final-only writable scratch assessor."""

from datetime import datetime, timedelta, timezone
import hashlib
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import final_scratch_assessor as assessor


def _manifest(root: Path) -> dict[str, str]:
    return {path.relative_to(root).as_posix(): hashlib.sha256(path.read_bytes()).hexdigest()
            for path in sorted(root.rglob("*")) if path.is_file() and ".git" not in path.parts}


class FinalScratchAssessorTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        self.candidate = self.root / "frozen-candidate"
        self.candidate.mkdir()
        (self.candidate / "README.md").write_text("frozen candidate\n", encoding="utf-8")
        (self.candidate / "tests").mkdir()
        (self.candidate / "tests" / "test_app.py").write_text("assert True\n", encoding="utf-8")
        self.public = self.root / "initial-public"
        self.public.mkdir()
        (self.public / "requirements.md").write_text("frozen public requirements\n", encoding="utf-8")
        self.audit = self.root / "audit"
        self.audit.mkdir()
        self.candidate_manifest = _manifest(self.candidate)
        self.public_manifest = _manifest(self.public)
        self.prepared = assessor.prepare_scratch(
            self.candidate, self.audit, self.public,
            self.candidate_manifest, self.public_manifest)
        self.choice = {"case": "readinglog", "config": {
            "backend": "codex-app-server", "model": "gpt-6-luna", "effort": "high"}}
        self.plan = {"id": "fixture-supplement", "freshFinalSeconds": 5400}

    def tearDown(self):
        self.temp.cleanup()

    def _assess(self, fake_native):
        with patch("conventional.backends.run", side_effect=fake_native) as native:
            result = assessor.assess_scratch(
                self.plan, self.choice, Path(sys.executable), self.candidate, self.audit,
                datetime.now(timezone.utc) + timedelta(minutes=5), self.public,
                self.candidate_manifest, self.public_manifest)
        return result, native

    def test_setup_creates_exact_execution_copy_and_read_only_requirements_sibling(self):
        copy = self.prepared["executionCandidate"]
        requirements = self.prepared["requirements"]
        self.assertEqual(_manifest(copy), self.candidate_manifest)
        self.assertEqual(_manifest(requirements), self.public_manifest)
        self.assertEqual(copy.parent, requirements.parent)
        self.assertTrue((copy / ".scratch").is_dir())
        self.assertTrue((copy / ".assessment-output").is_dir())
        binding = json.loads(self.prepared["bindingPath"].read_text(encoding="utf-8"))
        self.assertEqual(binding["writableRoot"], str(copy.resolve()))
        self.assertEqual(binding["writableRoots"], [])

    def test_assessment_uses_bounded_appserver_and_accepts_only_runtime_artifacts(self):
        def native(spec, prompt, resume, emit, cancel):
            self.assertEqual(os.environ.get("PYTHONDONTWRITEBYTECODE"), "1")
            self.assertEqual(spec["backend"], "codex-app-server")
            self.assertEqual(spec["cwd"], str(self.prepared["executionCandidate"].resolve()))
            self.assertEqual(spec["runtimeOptions"], {
                "sandbox": "workspace-write", "approvalPolicy": "never", "memoryEnabled": False,
                "nativeHelperModel": "gpt-6-luna", "nativeHelperEffort": "high",
                "nativeMaxConcurrentAgents": 1, "scopedGitApproval": False,
                "allowLoginShell": False, "windowsSandbox": "mxc"})
            self.assertIn(str(self.prepared["requirements"].resolve()), prompt)
            self.assertIsNone(resume)
            self.assertIn("24 tests", prompt)
            self.assertIn(".assessment-output/final-assessment.md", prompt)
            emit({"kind": "fixture-terminal", "state": "completed"})
            (self.prepared["executionCandidate"] / ".scratch" / "test.db").write_bytes(b"db")
            (self.prepared["executionCandidate"] / ".assessment-output" / "final-assessment.md").write_text(
                "Tests: 24 passed. Public cases: pass.\n", encoding="utf-8")
            return {"state": "completed", "detail": "terminal reached"}

        with patch.dict(os.environ, {"PYTHONDONTWRITEBYTECODE": "parent-value"}):
            result, native_call = self._assess(native)
            self.assertEqual(os.environ.get("PYTHONDONTWRITEBYTECODE"), "parent-value")
        self.assertEqual(native_call.call_count, 1)
        self.assertEqual(result["state"], "completed")
        self.assertEqual(result["scientificJudgment"], "PENDING INDEPENDENT REVIEW OF PRESERVED REPORT")
        self.assertTrue(result["candidateFilesUnchanged"])
        self.assertTrue(result["publicRequirementsUnchanged"])
        self.assertTrue(result["executionRequirementsUnchanged"])
        self.assertTrue(result["preparationReceiptUnchanged"])
        self.assertTrue(result["noHelpersObserved"])
        self.assertTrue(result["executionArtifactBaselinePreserved"])
        self.assertEqual(result["nativeEnvironmentPolicy"]["PYTHONDONTWRITEBYTECODE"], "1")
        self.assertEqual(result["nativeEnvironmentPolicy"]["priorValue"], "not captured")
        self.assertEqual(set(result["allowedArtifacts"]), {
            ".scratch/test.db", ".assessment-output/final-assessment.md"})
        report = Path(result["scientificReportPath"])
        self.assertEqual(report.read_text(encoding="utf-8"), "Tests: 24 passed. Public cases: pass.\n")
        self.assertEqual(_manifest(self.candidate), self.candidate_manifest)
        self.assertEqual(_manifest(self.public), self.public_manifest)

    def test_edit_to_execution_copy_semantic_file_fails_assessment(self):
        def mutate(spec, prompt, resume, emit, cancel):
            (Path(spec["cwd"]) / "README.md").write_text("modified\n", encoding="utf-8")
            report = Path(spec["cwd"]) / ".assessment-output" / "final-assessment.md"
            report.write_text("report\n", encoding="utf-8")
            return {"state": "completed"}

        result, _ = self._assess(mutate)
        self.assertEqual(result["nativeState"], "completed")
        self.assertEqual(result["state"], "failed")
        self.assertFalse(result["boundaryChecks"]["executionSemanticFilesUnchanged"])

    def test_new_product_root_file_fails_but_artifacts_are_permitted(self):
        def add_file(spec, prompt, resume, emit, cancel):
            cwd = Path(spec["cwd"])
            (cwd / ".scratch" / "allowed.tmp").write_text("runtime", encoding="utf-8")
            (cwd / ".assessment-output" / "final-assessment.md").write_text("report", encoding="utf-8")
            (cwd / "new_product.py").write_text("print('not allowed')", encoding="utf-8")
            return {"state": "completed"}

        result, _ = self._assess(add_file)
        self.assertEqual(result["state"], "failed")
        self.assertFalse(result["boundaryChecks"]["executionSemanticFilesUnchanged"])
        self.assertEqual(set(result["allowedArtifacts"]), {
            ".scratch/allowed.tmp", ".assessment-output/final-assessment.md"})

    def test_new_git_metadata_is_rejected_even_though_it_is_not_a_semantic_manifest_entry(self):
        def initialize_git(spec, prompt, resume, emit, cancel):
            git = Path(spec["cwd"]) / ".git"
            git.mkdir()
            (git / "config").write_text("[core]\n", encoding="utf-8")
            (Path(spec["cwd"]) / ".assessment-output" / "final-assessment.md").write_text("report", encoding="utf-8")
            return {"state": "completed"}

        result, _ = self._assess(initialize_git)
        self.assertEqual(result["state"], "failed")
        self.assertFalse(result["boundaryChecks"]["executionGitMetadataAbsent"])

    def test_expected_input_manifest_mismatch_is_rejected_before_native_start(self):
        wrong = dict(self.public_manifest)
        wrong["requirements.md"] = "0" * 64
        with patch("conventional.backends.run") as native:
            with self.assertRaisesRegex(ValueError, "expected frozen inputs"):
                assessor.assess_scratch(
                    self.plan, self.choice, Path(sys.executable), self.candidate, self.audit,
                    datetime.now(timezone.utc) + timedelta(minutes=5), self.public,
                    self.candidate_manifest, wrong)
        native.assert_not_called()

    def test_source_input_change_after_prepare_is_rejected_before_native_start(self):
        (self.public / "requirements.md").write_text("changed after preparation\n", encoding="utf-8")
        with patch("conventional.backends.run") as native:
            with self.assertRaisesRegex(ValueError, "source input changed"):
                assessor.assess_scratch(
                    self.plan, self.choice, Path(sys.executable), self.candidate, self.audit,
                    datetime.now(timezone.utc) + timedelta(minutes=5), self.public,
                    self.candidate_manifest, self.public_manifest)
        native.assert_not_called()

    def test_mutated_requirements_copy_fails_even_when_native_turn_completes(self):
        def mutate_requirements(spec, prompt, resume, emit, cancel):
            (self.prepared["requirements"] / "requirements.md").write_text("changed\n", encoding="utf-8")
            (Path(spec["cwd"]) / ".assessment-output" / "final-assessment.md").write_text("report", encoding="utf-8")
            return {"state": "completed"}

        result, _ = self._assess(mutate_requirements)
        self.assertEqual(result["nativeState"], "completed")
        self.assertEqual(result["state"], "failed")
        self.assertFalse(result["executionRequirementsUnchanged"])

    def test_preparation_receipt_mutation_fails_even_when_native_turn_completes(self):
        def mutate_receipt(spec, prompt, resume, emit, cancel):
            self.prepared["bindingPath"].write_text("{}\n", encoding="utf-8")
            (Path(spec["cwd"]) / ".assessment-output" / "final-assessment.md").write_text("report", encoding="utf-8")
            return {"state": "completed"}

        result, _ = self._assess(mutate_receipt)
        self.assertEqual(result["nativeState"], "completed")
        self.assertEqual(result["state"], "failed")
        self.assertFalse(result["preparationReceiptUnchanged"])

    def test_helper_activity_fails_without_replaying_native_assessment(self):
        def helper_activity(spec, prompt, resume, emit, cancel):
            emit({"source": "codex-app-server", "stream": "stdout", "parsed": {
                "method": "codex/event", "params": {"event": {
                    "type": "subAgentActivity", "kind": "started"}}}})
            (Path(spec["cwd"]) / ".assessment-output" / "final-assessment.md").write_text("report", encoding="utf-8")
            return {"state": "completed"}

        result, native = self._assess(helper_activity)
        self.assertEqual(native.call_count, 1)
        self.assertEqual(result["nativeState"], "completed")
        self.assertEqual(result["state"], "failed")
        self.assertFalse(result["noHelpersObserved"])

    def test_bytecode_environment_is_restored_after_backend_exception(self):
        def fail_native(spec, prompt, resume, emit, cancel):
            self.assertEqual(os.environ.get("PYTHONDONTWRITEBYTECODE"), "1")
            raise RuntimeError("fixture backend failure")

        with patch.dict(os.environ):
            os.environ.pop("PYTHONDONTWRITEBYTECODE", None)
            result, native = self._assess(fail_native)
            self.assertNotIn("PYTHONDONTWRITEBYTECODE", os.environ)
        self.assertEqual(native.call_count, 1)
        self.assertEqual(result["state"], "uncertain")
        self.assertEqual(result["nativeEnvironmentPolicy"]["PYTHONDONTWRITEBYTECODE"], "1")

    def test_deleting_preexisting_artifact_is_rejected(self):
        probe = self.prepared["executionCandidate"] / ".scratch" / "probe.txt"
        probe.write_text("preflight", encoding="utf-8")

        def delete_probe(spec, prompt, resume, emit, cancel):
            probe.unlink()
            (Path(spec["cwd"]) / ".assessment-output" / "final-assessment.md").write_text("report", encoding="utf-8")
            return {"state": "completed"}

        result, _ = self._assess(delete_probe)
        self.assertEqual(result["state"], "failed")
        self.assertFalse(result["boundaryChecks"]["executionArtifactBaselinePreserved"])


if __name__ == "__main__":
    unittest.main()
