"""Synthetic coverage for R3 early wrapper diagnostics; no native process is started."""
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
import native_fixture_budget


R3_KEY = "native-s1-contract-corrected-integration-20261008-r3"


class _BinaryStream:
    def __init__(self):
        self.buffer = io.BytesIO()

    def flush(self):
        pass


class NativeR3DiagnosticsTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="native-r3-wrapper-diagnostics-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()
        self.original_stdout, self.original_stderr = sys.stdout, sys.stderr
        self.addCleanup(self._restore_streams)
        self.request_path = self.root / "request.json"
        self.grant_path = self.root / "released-r3-grant.json"
        self.grant_path.write_text("synthetic placeholder; validator is patched", encoding="utf-8")
        self.request = {
            "arm": "government",
            "mode": "mechanical",
            "nativeFixtureGrant": {
                "path": str(self.root / "original-r1-grant.json"),
                "sha256": "1" * 64,
                "sourceKey": "native-s1-integration-fixtures-20261008",
            },
            "nativeFixtureR3Grant": {
                "path": str(self.grant_path),
                "sha256": "2" * 64,
                "sourceKey": R3_KEY,
            },
        }
        self._save_request()
        self.config = {
            "arm": "government",
            "correction": dict(self.request["nativeFixtureR3Grant"]),
            "requestPath": str(self.request_path),
        }
        self.config_path = self.root / "diagnostics.json"
        self._save_config()
        self.validated = {
            "grantKey": R3_KEY,
            "maxRoleStarts": 6,
            "maxDeterministicDelegates": 6,
            "maxNativeStarts": 2,
            "maxReservedSessionSeconds": 300,
        }

    def _restore_streams(self):
        sys.stdout, sys.stderr = self.original_stdout, self.original_stderr

    @staticmethod
    def _sha(raw):
        return hashlib.sha256(raw).hexdigest()

    def _save_request(self):
        self.request_path.write_text(json.dumps(self.request, sort_keys=True), encoding="utf-8")

    def _save_config(self):
        raw = json.dumps(self.config, sort_keys=True).encode("utf-8")
        self.config_path.write_bytes(raw)
        self.config_sha = self._sha(raw)

    def _argv(self):
        return [str(Path(government_roles.__file__).resolve()),
                "--diagnostics-config", str(self.config_path),
                "--diagnostics-sha256", self.config_sha]

    def _start(self):
        out, err = _BinaryStream(), _BinaryStream()
        registered = []
        sys.stdout, sys.stderr = out, err
        try:
            with patch.object(atexit, "register", side_effect=registered.append):
                state = government_roles.start_native_diagnostics(self._argv())
                tee_out, tee_err = sys.stdout, sys.stderr
        finally:
            self._restore_streams()
        return state, registered, (out, err, tee_out, tee_err)

    @staticmethod
    def _finish(state, registered, streams):
        try:
            if registered != [state["finish"]]:
                raise AssertionError("the raw diagnostic receipt must be registered once")
            state["finish"]()
        finally:
            streams[2].file.close()
            streams[3].file.close()

    def test_exact_r3_grant_caps_six_and_retains_raw_stream_receipts(self):
        with patch("native_fixture_budget.validate_r3_grant_binding", return_value=self.validated) as validate:
            for index in range(6):
                state, registered, streams = self._start()
                self.assertEqual(state["config"], self.config)
                validate.assert_called_with(self.request, self.request["nativeFixtureGrant"]["path"],
                                            self.request["nativeFixtureGrant"]["sha256"])
                error = f"synthetic wrapper bootstrap failure {index}\r\n".encode()
                streams[3].write(error.decode("utf-8"))
                call_dir = Path(state["callDirectory"])
                self.assertEqual((call_dir / "stderr.log").read_bytes(), error)
                self._finish(state, registered, streams)
                finished = json.loads((call_dir / "finished.json").read_bytes())
                receipt = next(item for item in finished["receipts"]
                               if item["path"].endswith("stderr.log"))
                self.assertEqual(receipt["sha256"], self._sha(error))

        starts = self.root / "wrapper-diagnostics" / "starts.sqlite"
        with closing(sqlite3.connect(starts)) as db:
            self.assertEqual(db.execute("SELECT COUNT(*) FROM starts").fetchone()[0], 6)
        self.assertEqual(len(list((self.root / "wrapper-diagnostics").glob("wrapper-*/start.json"))), 6)
        with patch("native_fixture_budget.validate_r3_grant_binding", return_value=self.validated):
            with self.assertRaisesRegex(ValueError, "cap exhausted"):
                government_roles.start_native_diagnostics(self._argv())
        with closing(sqlite3.connect(starts)) as db:
            self.assertEqual(db.execute("SELECT COUNT(*) FROM starts").fetchone()[0], 6)

    def test_r3_request_binding_must_match_and_cannot_reuse_r2_correction(self):
        self.request["nativeFixtureR3Grant"]["sha256"] = "3" * 64
        self._save_request()
        with self.assertRaisesRegex(ValueError, "exact R3 Request"):
            self._start()
        self.assertFalse((self.root / "wrapper-diagnostics").exists())

        self.request["nativeFixtureR3Grant"] = dict(self.config["correction"])
        self.request["nativeFixtureCorrection"] = {"sourceKey": "native-s1-corrected-integration-20261008-r2"}
        self._save_request()
        with self.assertRaisesRegex(ValueError, "exact R3 Request"):
            self._start()
        self.assertFalse((self.root / "wrapper-diagnostics").exists())

    def test_r3_diagnostics_reject_mismatched_validated_bounds_before_start_record(self):
        invalid = dict(self.validated, maxRoleStarts=7)
        with patch("native_fixture_budget.validate_r3_grant_binding", return_value=invalid):
            with self.assertRaisesRegex(ValueError, "exact R3 role allocation"):
                self._start()
        self.assertFalse((self.root / "wrapper-diagnostics").exists())

    def test_direct_run_role_checks_live_gate_before_reservation_or_delegate(self):
        request_raw = b"synthetic request bytes"
        invocation_raw = b"synthetic native Invocation bytes"
        request = {
            "arm": "government", "dispatchId": "dispatch-r3-test", "trialId": "trial-r3-test",
            "task": {"id": "task-r3-test"}, "actorRepository": str(self.root / "actor"),
            "evidenceDirectory": str(self.root / "trial-evidence"), "product": {},
            "nativeFixtureGrant": {"sourceKey": "original"},
            "nativeFixtureR3Grant": {"sourceKey": R3_KEY},
        }
        validated = dict(self.validated, grant={"key": R3_KEY})
        fixture_bounds = {"maxRoleStarts": 6, "maxRoleParallel": 2,
                          "maxRoleProcessSeconds": 300, "r3Grant": validated}
        role_auth = {"expiresAt": 9999999999, "roleEvidenceDirectory": str(self.root / "role-evidence"),
                     "requestPath": str(self.request_path)}
        role_slot = {"slotId": "government/test", "phase": "execute", "responseRole": "executor",
                     "delegate": {"command": str(self.root / "delegate.exe"),
                                  "argv": [str(self.root / "delegate.exe")], "timeoutSeconds": 38,
                                  "maxStdoutBytes": 1024, "maxStderrBytes": 1024}}

        class FakeLedger:
            def dispatch_record(self, _dispatch_id):
                return {"phase": "launching", "result": None, "request": request_raw,
                        "execution_sha": "expected-execution", "attempt": 1}

        class FakeAuthority:
            grant = {"retrospectiveTokenThreshold": 10000}
            ledger_path = self.root / "trial.sqlite"
            paths = []

            def validate(self, _request_raw):
                return request, {}

            def ledger(self, *_args, **_kwargs):
                return FakeLedger()

        reserve = unittest.mock.Mock()
        bounded = unittest.mock.Mock()
        with (patch.object(government_roles, "parse_invocation",
                           return_value={"runId": "run", "nonce": "nonce",
                                         "inputDigest": "input", "request": {"role": "executor"}}),
              patch.object(government_roles.native_controller, "validate_native_fixture_grant",
                           return_value=fixture_bounds),
              patch.object(government_roles, "_role_auth", return_value=(role_auth, role_slot, {})),
              patch.object(government_roles, "_execution_sha", return_value="expected-execution"),
              patch.object(government_roles, "_effective_timeout", return_value=38),
              patch.object(government_roles, "_reserve", reserve),
              patch.object(government_roles, "bounded", bounded),
              patch.object(native_fixture_budget, "validate_r3_entry_gate",
                           side_effect=ValueError("live R3 slot is not assigned")) as gate):
            with self.assertRaisesRegex(ValueError, "live R3 slot"):
                government_roles.run_role(
                    invocation_raw, b"synthetic auth", "auth-sha", str(self.root / "authorization.json"),
                    request_raw, FakeAuthority())
        gate.assert_called_once_with(validated)
        reserve.assert_not_called()
        bounded.assert_not_called()


if __name__ == "__main__":
    unittest.main()
