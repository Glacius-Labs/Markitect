"""Offline receipt identity regression using sealed R6 bytes; no run admission."""
import copy
import hashlib
import json
from pathlib import Path
import shutil
import tempfile
import unittest
from unittest.mock import patch

import government_native_profile as profiles
import native_fixture_budget as budget

PACKAGE = Path(__file__).parents[1]
SEALED = PACKAGE / "evidence/government-released-binding-native-20261008-r6"
ARCHIVE = SEALED / "raw-external"
MANIFEST_SHA = "84e4633ce9dd8cb97a04d0fbd6c4abb28e7b1b9678f1bb968138c5cf5247fc2e"


def encoded(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":")).encode() + b"\n"


def sealed_json(path):
    manifest_raw = (SEALED / "manifest.json").read_bytes()
    if hashlib.sha256(manifest_raw).hexdigest() != MANIFEST_SHA:
        raise AssertionError("sealed R6 manifest changed")
    entries = {item["path"]: item["sha256"] for item in json.loads(manifest_raw)["entries"]}
    raw = path.read_bytes()
    if hashlib.sha256(raw).hexdigest() != entries[path.relative_to(PACKAGE).as_posix()]:
        raise AssertionError(f"sealed R6 input changed: {path}")
    return json.loads(raw)


class SealedCheckInputs(unittest.TestCase):
    def setUp(self):
        self.addCleanup(patch.stopall)
        patch("subprocess.Popen", side_effect=AssertionError("offline test forbids processes")).start()
        patch("sqlite3.connect", side_effect=AssertionError("offline test forbids ledger access")).start()
        self.definitions = sealed_json(ARCHIVE / "government/released/runtime.json")["checks"]
        reports = list((ARCHIVE / "government/results/run-state").glob("*/report.json"))
        self.assertEqual(len(reports), 1)
        self.results = sealed_json(reports[0])["checks"]

    def consume(self, definitions=None, results=None):
        return budget._validate_r4_fresh_check_receipts(
            self.definitions if definitions is None else definitions,
            self.results if results is None else results)


class GovernmentCheckReceiptTests(SealedCheckInputs):
    def test_exact_sealed_configuration_and_native_gate_result_match(self):
        self.assertNotIn("tool", self.definitions[0])
        self.assertEqual(self.results[0]["Tool"], self.definitions[0]["run"][0])
        self.assertTrue(self.consume())

    def test_exact_check_set_allows_reordering(self):
        definitions = self.definitions + [dict(self.definitions[0], name="second")]
        results = [dict(self.results[0], Name="second")] + self.results
        self.assertTrue(self.consume(definitions, results))

    def test_wrong_name_or_executable_token_is_rejected(self):
        for field, value in (("Name", "other"), ("Tool", "python")):
            with self.subTest(field=field):
                results = copy.deepcopy(self.results)
                results[0][field] = value
                with self.assertRaises(ValueError):
                    self.consume(results=results)

    def test_missing_additional_and_duplicate_checks_are_rejected(self):
        variants = [([], self.results), (self.definitions, []),
                    (self.definitions, self.results * 2),
                    (self.definitions * 2, self.results * 2),
                    (self.definitions + [dict(self.definitions[0], name="second")], self.results * 2)]
        for definitions, results in variants:
            with self.subTest(definitions=definitions, results=results):
                with self.assertRaises(ValueError):
                    self.consume(definitions, results)

    def test_invalid_configuration_shapes_and_commands_are_rejected(self):
        for definitions in (None, {}, "checks", [], [None], [{}],
                            [{"name": [], "run": ["go"]}],
                            [{"name": "", "run": ["go"]}],
                            [{"name": "inventory-overflow", "tool": "go"}]):
            with self.subTest(definitions=definitions), self.assertRaises(ValueError):
                budget._validate_r4_fresh_check_receipts(definitions, self.results)
        for command in (None, "go test", (), [], [None], [""], ["go test"],
                        ["/usr/bin/go"], ["C:\\go.exe"], ["go\u00a0"],
                        ["go\x00"], ["go", 1], ["go", "test\x00"]):
            with self.subTest(command=command), self.assertRaises(ValueError):
                self.consume([dict(self.definitions[0], run=command)])

    def test_invalid_native_identity_shapes_are_rejected(self):
        for results in (None, {}, "checks", [None], [{}],
                        [dict(self.results[0], Name=[])],
                        [dict(self.results[0], Tool={})],
                        [{"name": "inventory-overflow", "tool": "go"}]):
            with self.subTest(results=results), self.assertRaises(ValueError):
                budget._validate_r4_fresh_check_receipts(self.definitions, results)

    def test_exit_receipt_must_be_integer_zero(self):
        for value in (1, -1, True, False, None, 0.0, "0"):
            with self.subTest(value=value), self.assertRaises(ValueError):
                self.consume(results=[dict(self.results[0], ExitCode=value)])

    def test_invalid_or_exceeded_time_receipts_are_rejected(self):
        for field, values in (("Milliseconds", (-1, True, None, 1.0, "1", 30001)),
                              ("TimeoutMilliseconds", (0, -1, True, None, 30000.0, "30000", 6269))):
            for value in values:
                with self.subTest(field=field, value=value), self.assertRaises(ValueError):
                    self.consume(results=[dict(self.results[0], **{field: value})])

    def test_zero_duration_and_exact_timeout_remain_valid(self):
        for duration in (0, 30000):
            with self.subTest(duration=duration):
                self.assertTrue(self.consume(results=[dict(self.results[0], Milliseconds=duration)]))


class GovernmentResumeReceiptTransitionTests(SealedCheckInputs):
    """Call the actual pre-reservation helper and translator on relocated copies."""
    profile_name = "r6"

    def configure_test_request(self, request, base):
        """Successor tests may rebind only their isolated fixture metadata."""

    def relocated_gate(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        external = Path(temp.name).resolve() / "external"
        shutil.copytree(ARCHIVE, external)
        old_root = str(Path(sealed_json(ARCHIVE / "government/released/request.json")
                            ["actorRepository"]).parents[1])

        def relocate(value):
            if isinstance(value, str):
                return value.replace(old_root, str(external))
            if isinstance(value, list):
                return [relocate(item) for item in value]
            if isinstance(value, dict):
                return {key: relocate(item) for key, item in value.items()}
            return value

        for path in external.rglob("*.json"):
            path.write_bytes(encoded(relocate(json.loads(path.read_bytes()))))
        base = external / "government"
        request_path = base / "released/request.json"
        request = json.loads(request_path.read_bytes())
        self.configure_test_request(request, base)
        auth_path = base / "released/role-auth.json"
        runtime_path = base / "released/runtime.json"
        runtime = json.loads(runtime_path.read_bytes())
        auth_sha = budget.sha(auth_path)
        for role in [runtime["executor"], runtime["verifier"], runtime["ressorts"][0]["runner"]]:
            role["args"][role["args"].index("--authorization-sha256") + 1] = auth_sha
            for item in role["runtimeFiles"]:
                if item["path"] == str(auth_path):
                    item["digest"] = "sha256:" + auth_sha
        runtime_path.write_bytes(encoded(runtime))
        for binding in request["releasedInputs"] + list(request["product"]["government"].values()):
            if isinstance(binding, dict) and binding.get("path") in {
                    str(runtime_path), str(base / "released/backlog.json"), str(auth_path)}:
                binding["sha256"] = budget.sha(binding["path"])
        request_path.write_bytes(encoded(request))
        queue_path = Path(request["evidenceDirectory"]) / "process/stdout.log"
        queue = relocate(json.loads(queue_path.read_bytes()))
        queue["jobs"][0]["id"] = request["task"]["id"]
        report_path = Path(queue["jobs"][0]["reportPath"])

        def refresh_queue():
            queue["backlogDigest"] = budget.sha(base / "released/backlog.json")
            queue["jobs"][0]["reportDigest"] = budget.sha(report_path)
            raw = encoded(queue)
            queue_path.write_bytes(raw)
            (Path(queue["queueDirectory"]) /
             f"queue-report-{queue['journalSequence']:08d}.json").write_bytes(raw)

        refresh_queue()
        row = next(item for item in sealed_json(SEALED / "terminal-native-ledger.json")["tables"]["starts"]
                   if item["label"] == "government-native-released-binding-r6/queue")
        argv = relocate(json.loads(row["argv"]))
        receipt = relocate(json.loads(row["receipt"]))
        selected = profiles.profile(self.profile_name)
        queue_row = (row["product"], f"{selected.dispatch_id}/queue", json.dumps(argv), row["claimed"],
                     row["finished"], row["reserved_seconds"], encoded(receipt))
        instance = object.__new__(budget.FixtureBudget)
        instance.profile = selected
        instance.profile_request = request
        return instance, queue_row, report_path, runtime_path, refresh_queue

    def test_real_resume_pre_reservation_consumer_accepts_relocated_r6_receipts(self):
        instance, row, _, _, _ = self.relocated_gate()
        result = instance._profile_validate_queue_success(row)
        self.assertEqual(result["status"], "completed")
        self.assertEqual(result["government"]["queueStatus"], "complete")

    def test_real_resume_consumer_rejects_wrong_native_tool_after_translation(self):
        instance, row, report_path, _, refresh = self.relocated_gate()
        report = json.loads(report_path.read_bytes())
        report["checks"][0]["Tool"] = "python"
        report_path.write_bytes(encoded(report))
        refresh()
        with self.assertRaisesRegex(ValueError, "exact configured check names/tools"):
            instance._profile_validate_queue_success(row)

    def test_resume_consumer_keeps_runtime_and_report_digest_guards(self):
        for target in ("runtime", "report"):
            with self.subTest(target=target):
                instance, row, report_path, runtime_path, _ = self.relocated_gate()
                path = runtime_path if target == "runtime" else report_path
                path.write_bytes(path.read_bytes() + b" ")
                with self.assertRaisesRegex(ValueError, "binding is missing or changed|digest mismatch"):
                    instance._profile_validate_queue_success(row)


if __name__ == "__main__":
    unittest.main()
