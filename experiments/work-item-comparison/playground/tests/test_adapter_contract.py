"""Offline tests for the explicit, hash-pinned Playground adapter boundary."""
import hashlib
import importlib
import importlib.util
import io
import json
import os
import py_compile
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

PLAYGROUND = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(PLAYGROUND))

from adapters.contract import AdapterError, deliver_observations, validate_descriptor, validate_result
from adapters.loader import load_adapter


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


class AdapterContractTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        root = Path(self.temp.name)
        self.module_name = "fake_adapter_" + root.name.replace("-", "_")
        self.config = root / "config.json"
        self.config.write_text("{}\n", encoding="utf-8")
        self.module = root / (self.module_name + ".py")
        self.module.write_text('''
import hashlib
from pathlib import Path

class FakeAdapter:
    def __init__(self, config_path):
        self.config_path = config_path
        self.pins = None
        self.states = ["running", "completed"]
        self.closed = 0

    def bind_manifest_source_pins(self, pins):
        self.pins = pins

    def describe(self):
        return {"schema": 1, "id": "fake", "method": "Fake", "version": "1",
                "sourcePins": self.pins,
                "capabilities": {"ensure_runtime": True, "start": True, "resume": True,
                                  "status": True, "cancel": True, "close": True},
                "runtime": {"kind": "fixture", "owner": "none", "scope": "offline"}}

    def setup(self, context):
        return {"schema": 1, "state": "ready", "context": context}

    def ensure_runtime(self):
        return {"schema": 1, "state": "ready"}

    def start(self, prompt):
        return {"schema": 1, "state": "accepted", "runId": "fixture-run",
                "taskAssessment": "NOT RUN"}

    def resume(self, run_id, prompt):
        return {"schema": 1, "state": "accepted", "runId": run_id}

    def status(self, run_id):
        state = self.states.pop(0) if self.states else "completed"
        return {"schema": 1, "state": state, "runId": run_id}

    def cancel(self, run_id):
        return {"schema": 1, "state": "cancelled", "runId": run_id}

    def close(self):
        self.closed += 1
        return {"schema": 1, "state": "closed"}
''', encoding="utf-8")
        self.manifest = root / "adapter.json"
        self.write_manifest(digest(self.module))

    def write_manifest(self, module_hash):
        self.manifest.write_text(json.dumps({
            "schema": 1,
            "factory": self.module_name + ":FakeAdapter",
            "configPath": str(self.config.resolve()),
            "sourcePins": {str(self.module.resolve()): module_hash},
        }), encoding="utf-8")

    def test_loader_verifies_pins_before_import_and_binds_manifest(self):
        adapter = load_adapter(self.manifest)
        self.assertEqual(adapter.config_path, self.config.resolve())
        self.assertEqual(adapter.describe()["sourcePins"], {str(self.module.resolve()): digest(self.module)})

    def test_changed_source_is_rejected_before_factory_import(self):
        self.module.write_text(self.module.read_text(encoding="utf-8") + "\n# modified\n", encoding="utf-8")
        with self.assertRaisesRegex(AdapterError, "pin mismatch"):
            load_adapter(self.manifest)

    def test_cached_factory_is_rejected_after_its_source_changes(self):
        load_adapter(self.manifest)
        self.module.write_text(self.module.read_text(encoding="utf-8") + "\n# newer source\n", encoding="utf-8")
        self.write_manifest(digest(self.module))
        with self.assertRaisesRegex(AdapterError, "cached factory module source changed"):
            load_adapter(self.manifest)

    def test_stale_timestamp_pyc_cannot_override_new_hash_pinned_source(self):
        old_source = self.module.read_bytes()
        old_marker = b'"method": "Fake"'
        new_marker = b'"method": "Faux"'
        self.assertIn(old_marker, old_source)
        new_source = old_source.replace(old_marker, new_marker, 1)
        self.assertEqual(len(old_source), len(new_source))
        fixed_ns = 1_700_000_000_000_000_000
        self.module.write_bytes(old_source)
        os.utime(self.module, ns=(fixed_ns, fixed_ns))
        py_compile.compile(str(self.module), doraise=True,
                           invalidation_mode=py_compile.PycInvalidationMode.TIMESTAMP)
        pyc_path = Path(importlib.util.cache_from_source(str(self.module)))
        self.assertTrue(pyc_path.is_file())

        self.module.write_bytes(new_source)
        os.utime(self.module, ns=(fixed_ns, fixed_ns))
        self.assertEqual(self.module.stat().st_size, len(old_source))
        self.write_manifest(digest(self.module))

        adapter = load_adapter(self.manifest)
        self.assertEqual(adapter.describe()["method"], "Faux")

    def test_manifest_requires_factory_source_pin(self):
        self.manifest.write_text(json.dumps({
            "schema": 1, "factory": self.module_name + ":FakeAdapter",
            "configPath": str(self.config.resolve()), "sourcePins": {str(self.config.resolve()): digest(self.config)},
        }), encoding="utf-8")
        with self.assertRaisesRegex(AdapterError, "factory module source"):
            load_adapter(self.manifest)

    def test_descriptor_and_result_contracts(self):
        adapter = load_adapter(self.manifest)
        self.assertEqual(validate_descriptor(adapter.describe())["method"], "Fake")
        self.assertEqual(validate_result("setup", adapter.setup({"case": "x"}))["state"], "ready")
        with self.assertRaisesRegex(AdapterError, "runId"):
            validate_result("start", {"schema": 1, "state": "accepted"})
        self.assertEqual(validate_result("start", {"schema": 1, "state": "blocked"})["state"], "blocked")
        for state in ("completed", "cancelled"):
            self.assertEqual(validate_result("resume", {"schema": 1, "state": state,
                                                          "runId": "fixture-run"})["state"], state)
        with self.assertRaisesRegex(AdapterError, "observation"):
            validate_result("status", {"schema": 1, "state": "completed", "runId": "x",
                                        "observations": [{"schema": 1, "type": "unknown"}]})

    def test_observer_failure_does_not_change_lifecycle_result(self):
        result = {"schema": 1, "state": "completed", "runId": "x", "taskAssessment": "NOT RUN",
                  "observations": [{"schema": 1, "type": "attempt.finished",
                                    "observedAt": "2026-10-09T12:00:00Z", "data": {"details": {"x": 1}}}]}

        def mutate_then_fail(event):
            event["data"]["details"]["x"] = 2
            raise RuntimeError("sink")

        enriched = deliver_observations(result, mutate_then_fail)
        self.assertEqual(enriched["state"], "completed")
        self.assertEqual(enriched["observerFailures"], [{"eventType": "attempt.finished", "error": "RuntimeError"}])
        self.assertEqual(result["observations"][0]["data"]["details"]["x"], 1)
        self.assertNotIn("observerFailures", result)

    def test_cli_emits_handle_before_polling_to_terminal(self):
        cli = importlib.import_module("playground_adapter")
        output = io.StringIO()
        with patch("sys.stdout", output), patch("sys.stderr", io.StringIO()), patch("time.sleep"):
            code = cli.main(["--adapter", str(self.manifest), "--config", str(self.config), "start", "do work"])
        rows = [json.loads(line) for line in output.getvalue().splitlines()]
        self.assertEqual(code, 0)
        self.assertEqual([row["state"] for row in rows], ["accepted", "running", "completed"])
        self.assertEqual(rows[0]["runId"], "fixture-run")

    def test_cli_uncertain_is_terminal_and_never_replays(self):
        self.module.write_text(self.module.read_text(encoding="utf-8").replace(
            'self.states = ["running", "completed"]', 'self.states = ["uncertain"]'), encoding="utf-8")
        self.write_manifest(digest(self.module))
        cli = importlib.import_module("playground_adapter")
        output = io.StringIO()
        with patch("sys.stdout", output), patch("sys.stderr", io.StringIO()), patch("time.sleep") as sleep:
            code = cli.main(["--adapter", str(self.manifest), "--config", str(self.config), "start", "do work"])
        rows = [json.loads(line) for line in output.getvalue().splitlines()]
        self.assertEqual(code, 1)
        self.assertEqual([row["state"] for row in rows], ["accepted", "uncertain"])
        sleep.assert_not_called()


if __name__ == "__main__":
    unittest.main()
