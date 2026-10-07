"""Synthetic unit coverage for retained early native-wrapper diagnostics."""
import atexit
from contextlib import closing
import hashlib
import io
import json
from pathlib import Path
import sqlite3
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).parent))
import government_roles


class _BinaryStream:
    """Small TextIO-like target for Tee; no terminal or process is involved."""
    def __init__(self):
        self.buffer = io.BytesIO()

    def flush(self):
        pass


class NativeRoleDiagnosticsTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="native-role-diagnostics-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()
        self._write_inputs()
        self.original_stdout, self.original_stderr = sys.stdout, sys.stderr
        self.addCleanup(self._restore_streams)

    def _restore_streams(self):
        sys.stdout, sys.stderr = self.original_stdout, self.original_stderr

    @staticmethod
    def _sha(raw):
        return hashlib.sha256(raw).hexdigest()

    def _write_inputs(self):
        grant = {
            "key": "native-s1-corrected-integration-20261008-r2",
            "limits": {
                arm: {"maxAdditionalRoleInvocationsIncludingFailedWrapperStarts": 6}
                for arm in ("government", "classic")
            },
            "realActorCalls": 0,
            "providerCalls": 0,
            "metadataAppServerTrees": 0,
            "studyCells": 0,
        }
        snapshot_raw = json.dumps({"threads": [{"name": "Scientist", "evidence": {
            "correctedNativeIntegrationGrant": grant,
        }}]}, sort_keys=True).encode()
        self.snapshot_path = self.root / "coordination-snapshot.json"
        self.snapshot_path.write_bytes(snapshot_raw)
        correction = {
            "sourceThreadId": "01a11367-a781-7683-a20f-46e12614dcb4",
            "sourceJsonPointer": "threads[name=Scientist].evidence.correctedNativeIntegrationGrant",
            "sourceCoordinationPath": str(self.snapshot_path),
            "sourceCoordinationSha256": self._sha(snapshot_raw),
            "grant": grant,
        }
        self.correction_path = self.root / "correction.json"
        self.correction_path.write_text(json.dumps(correction, sort_keys=True), encoding="utf-8")
        correction_raw = self.correction_path.read_bytes()
        self.config = {
            "arm": "government",
            "correction": {
                "path": str(self.correction_path),
                "sha256": self._sha(correction_raw),
                "sourceKey": "native-s1-corrected-integration-20261008-r2",
            },
            "requestPath": str(self.root / "request.json"),
        }
        self.config_path = self.root / "diagnostics.json"
        self._save_config()

    def _save_config(self):
        self.config_raw = json.dumps(self.config, sort_keys=True).encode()
        self.config_path.write_bytes(self.config_raw)
        self.config_sha = self._sha(self.config_raw)

    def _argv(self, *, config_path=None, config_sha=None):
        return [
            str(Path(government_roles.__file__).resolve()),
            "--diagnostics-config", str(config_path or self.config_path),
            "--diagnostics-sha256", config_sha or self.config_sha,
        ]

    def _start(self, *, argv=None):
        out, err = _BinaryStream(), _BinaryStream()
        registered = []
        sys.stdout, sys.stderr = out, err
        try:
            with patch.object(atexit, "register", side_effect=registered.append):
                state = government_roles.start_native_diagnostics(argv or self._argv())
                tee_out, tee_err = sys.stdout, sys.stderr
        finally:
            self._restore_streams()
        return state, registered, (out, err, tee_out, tee_err)

    def _assert_start_rejected(self):
        try:
            state, registered, streams = self._start()
        except ValueError:
            return
        self._finish(state, registered, streams)
        self.fail("invalid diagnostic configuration was accepted")

    @staticmethod
    def _finish(state, registered, streams):
        try:
            if len(registered) != 1 or registered[0] != state["finish"]:
                raise AssertionError("finish must be registered for process exit")
            state["finish"]()
        finally:
            streams[2].file.close()
            streams[3].file.close()

    def test_retains_early_raw_stderr_and_digest_receipt(self):
        state, registered, streams = self._start()
        message = b"Traceback: bootstrap import failed before controller setup\r\n"
        streams[3].write(message.decode("utf-8"))
        call = Path(state["callDirectory"])
        start = json.loads((call / "start.json").read_bytes())
        self.assertEqual(start["arm"], "government")
        self.assertIsNone(start["providerUsage"])
        self.assertEqual(start["correction"]["sourceKey"], "native-s1-corrected-integration-20261008-r2")
        self.assertEqual((call / "stderr.log").read_bytes(), message)

        self._finish(state, registered, streams)
        finished = json.loads((call / "finished.json").read_bytes())
        stderr_receipt = next(row for row in finished["receipts"] if row["path"].endswith("stderr.log"))
        self.assertEqual(stderr_receipt["sha256"], self._sha(message))
        self.assertEqual(stderr_receipt["sha256"], self._sha((call / "stderr.log").read_bytes()))
        self.assertTrue((call / "start.json").is_file())

    def test_changed_configuration_digest_is_rejected_before_reserving_a_start(self):
        with self.assertRaisesRegex(ValueError, "configuration digest mismatch"):
            government_roles.start_native_diagnostics(self._argv(config_sha="0" * 64))
        self.assertFalse((self.root / "wrapper-diagnostics").exists())

    def test_correction_source_key_and_configuration_shape_are_bound(self):
        changed = dict(self.config)
        changed["correction"] = dict(self.config["correction"], sourceKey="other")
        self.config = changed
        self._save_config()
        self._assert_start_rejected()
        self.assertFalse((self.root / "wrapper-diagnostics").exists())

        self.config = dict(self.config, unexpected="field")
        self._save_config()
        self._assert_start_rejected()
        self.assertFalse((self.root / "wrapper-diagnostics").exists())

    def test_six_wrapper_starts_are_retained_and_seventh_is_rejected(self):
        states = []
        for _ in range(6):
            state, registered, streams = self._start()
            states.append((state, registered, streams))
            self._finish(state, registered, streams)
        with closing(sqlite3.connect(self.root / "wrapper-diagnostics" / "starts.sqlite")) as db:
            self.assertEqual(db.execute("SELECT COUNT(*) FROM starts").fetchone()[0], 6)
        self.assertEqual(len(list((self.root / "wrapper-diagnostics").glob("wrapper-*/start.json"))), 6)

        with self.assertRaisesRegex(ValueError, "cap exhausted"):
            government_roles.start_native_diagnostics(self._argv())
        with closing(sqlite3.connect(self.root / "wrapper-diagnostics" / "starts.sqlite")) as db:
            self.assertEqual(db.execute("SELECT COUNT(*) FROM starts").fetchone()[0], 6)


if __name__ == "__main__":
    unittest.main()
