"""R7 pure producer/receipt/translator boundary on isolated historical copies."""
import importlib.util
import json
import unittest

import native_fixture_budget as budget
import government_native_profile as profiles
import test_government_check_receipts as receipt_tests
from test_government_check_receipts import PACKAGE, encoded

spec = importlib.util.spec_from_file_location("r7_receipt_driver", PACKAGE / "run-native-integration-r5.py")
driver = importlib.util.module_from_spec(spec)
spec.loader.exec_module(driver)


class NativeR7ReceiptTransitionTests(receipt_tests.GovernmentResumeReceiptTransitionTests):
    profile_name = "r7"

    def configure_test_request(self, request, base):
        selected = profiles.R7
        original = {key: request["nativeFixtureGrant"][key] for key in ("path", "sha256")}
        old_marker = request.pop("nativeFixtureR6Grant")
        old_paths = {original["path"], old_marker["path"],
                     str(base.parent / profiles.R6.snapshot_path.name)}
        other_inputs = [item for item in request["releasedInputs"] if item["path"] not in old_paths]
        request.update(driver.fixture_grant_bindings(
            original, driver.binding(selected.envelope_path), driver.binding(selected.snapshot_path), "r7"))
        request["releasedInputs"].extend(other_inputs)
        request["dispatchId"] = selected.dispatch_id
        request["trialId"] = selected.key
        request["task"]["id"] = selected.task_id
        backlog_path = base / "released/backlog.json"
        backlog = json.loads(backlog_path.read_bytes())
        backlog["jobs"][0]["id"] = selected.task_id
        backlog_path.write_bytes(encoded(backlog))
        auth_path = base / "released/role-auth.json"
        auth = json.loads(auth_path.read_bytes())
        auth.update(dispatchId=selected.dispatch_id, trialId=selected.key, taskId=selected.task_id)
        auth_path.write_bytes(encoded(auth))

    def test_real_resume_pre_reservation_consumer_accepts_relocated_r6_receipts(self):
        instance, row, _, _, _ = self.relocated_gate()
        request = instance.profile_request
        admitted = budget.validate_r7_grant_binding(
            request, request["nativeFixtureGrant"]["path"], request["nativeFixtureGrant"]["sha256"])
        self.assertEqual(admitted["profileName"], "r7")
        self.assertNotIn("nativeFixtureR6Grant", request)
        result = instance._profile_validate_queue_success(row)
        self.assertEqual(result["status"], "completed")
        self.assertEqual(result["government"]["nativeJobStates"][0]["id"], profiles.R7.task_id)


if __name__ == "__main__":
    unittest.main()
