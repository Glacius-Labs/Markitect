"""Six bounded synthetic cases for native CWD binding and strict thread gating."""
import copy
import json
from pathlib import Path
import unittest
from unittest.mock import patch

import client


HERE = Path(__file__).resolve().parent
PROFILE = json.loads((HERE / "profile.json").read_text(encoding="utf-8"))
AUTHORITY = json.loads((HERE / "authorization-grant.json").read_text(encoding="utf-8"))
CONTRACT = json.loads((HERE / "schema-contract.json").read_text(encoding="utf-8"))
PROTOCOL_MODULE = client.load_module(client.PROTOCOL, client.PROTOCOL_SHA, "native_cwd_test_protocol")
PROTOCOL = PROTOCOL_MODULE.Protocol(client.SCHEMA, CONTRACT)
GATE_MODULE = client.load_module(HERE / "thread_gate.py", client.sha((HERE / "thread_gate.py").read_bytes()),
                                 "native_cwd_test_gate")

SYNTHETIC_PARAMS = {
    "cwd": r"C:\Synthetic\AgentRoot",
    "model": "synthetic-model-alpha",
    "modelProvider": "synthetic-provider-alpha",
    "approvalPolicy": "never",
    "ephemeral": True,
}


def synthetic_profile():
    return {"threadStart": {"params": copy.deepcopy(SYNTHETIC_PARAMS)}}


def synthetic_response(profile, **changes):
    params = profile["threadStart"]["params"]
    value = {
        "approvalPolicy": params["approvalPolicy"],
        "approvalsReviewer": "user",
        "cwd": params["cwd"],
        "model": params["model"],
        "modelProvider": params["modelProvider"],
        "sandbox": {"type": "readOnly", "networkAccess": False},
        "activePermissionProfile": {"id": ":read-only", "extends": None},
        "thread": {
            "cliVersion": "offline-synthetic",
            "createdAt": 1,
            "cwd": params["cwd"],
            "ephemeral": True,
            "id": "synthetic-native-cwd-thread",
            "model": params["model"],
            "modelProvider": params["modelProvider"],
            "preview": "",
            "projectId": None,
            "sessionId": "synthetic-native-cwd-session",
            "source": "appServer",
            "status": {"type": "active", "activeFlags": []},
            "turns": [],
            "updatedAt": 1,
            "parentThreadId": None,
            "forkedFromId": None,
        },
    }
    value.update(changes)
    return value


def expected_native_identity():
    contract = client.CWD_IDENTITY
    return {
        "volumeSerial": contract["volumeSerial"],
        "fileIndex": contract["fileIndex"],
        "attributes": contract["expectedLeafAttributes"],
    }


class NativeCwdDiagnosticTests(unittest.TestCase):
    def test_profile_native_spellings_identity_and_exact_public_inputs_are_consistent(self):
        profile = copy.deepcopy(PROFILE)
        grant = copy.deepcopy(AUTHORITY["grant"])
        client.validate_profile(profile, grant)
        with patch.object(client, "native_identity", return_value=expected_native_identity()) as native_id:
            client.validate_cwd(profile)
        self.assertEqual(native_id.call_count, 2)
        self.assertEqual(profile["cwd"], client.CWD_IDENTITY["nativeSpelling"])
        self.assertEqual(profile["expectedFiles"], client.CWD_IDENTITY["exactRelativeFiles"])

    def test_wrong_or_mixed_cwd_spellings_are_rejected_at_profile_and_capture_boundaries(self):
        wrong_profile = copy.deepcopy(PROFILE)
        wrong_profile["cwd"] = client.CWD_IDENTITY["previousSpelling"]
        with self.assertRaisesRegex(ValueError, "cwd-input-spelling-mismatch"):
            client.validate_cwd(wrong_profile)

        mixed_profile = copy.deepcopy(PROFILE)
        mixed_profile["rpc"][2]["params"]["cwd"] = client.CWD_IDENTITY["previousSpelling"]
        with self.assertRaisesRegex(ValueError, "argv-or-rpc-mismatch"):
            client.validate_profile(mixed_profile, AUTHORITY["grant"])

        mismatched_thread = copy.deepcopy(PROFILE)
        mismatched_thread["threadStart"]["params"]["cwd"] = client.CWD_IDENTITY["previousSpelling"]
        with self.assertRaisesRegex(ValueError, "thread-or-input-mismatch"):
            client.validate_profile(mismatched_thread, AUTHORITY["grant"])

    def test_native_file_identity_and_pinned_metadata_mismatches_fail_closed(self):
        profile = copy.deepcopy(PROFILE)
        wrong_id = expected_native_identity()
        wrong_id["fileIndex"] += 1
        with patch.object(client, "native_identity", return_value=wrong_id):
            with self.assertRaisesRegex(ValueError, "cwd-file-id-mismatch"):
                client.validate_cwd(profile)

        wrong_inventory = copy.deepcopy(profile)
        wrong_inventory["expectedFiles"]["extra.txt"] = "0" * 64
        with self.assertRaisesRegex(ValueError, "cwd-expected-files-mismatch"):
            client.validate_cwd(wrong_inventory)

        changed_contract = copy.deepcopy(AUTHORITY["grant"])
        changed_contract["cwdIdentityContract"]["volumeSerial"] += 1
        with self.assertRaisesRegex(ValueError, "cwd-identity-contract-mismatch"):
            client.validate_profile(profile, changed_contract)

    def test_top_level_and_nested_response_cwd_mismatches_remain_strict(self):
        profile = synthetic_profile()
        top_gate = GATE_MODULE.ThreadGate(profile, PROTOCOL, CONTRACT)
        with self.assertRaisesRegex(ValueError, "thread-response-identity-or-policy-mismatch"):
            top_gate.accept_response(synthetic_response(profile, cwd=r"C:\Synthetic\OtherRoot"))
        self.assertFalse(top_gate.finish()["responseValidated"])

        nested_gate = GATE_MODULE.ThreadGate(profile, PROTOCOL, CONTRACT)
        bad_thread = synthetic_response(profile)["thread"]
        bad_thread["cwd"] = r"C:\Synthetic\OtherRoot"
        with self.assertRaisesRegex(ValueError, "thread-identity-mismatch"):
            nested_gate.accept_response(synthetic_response(profile, thread=bad_thread))
        self.assertFalse(nested_gate.finish()["responseValidated"])

    def test_full_valid_synthetic_thread_boundary_still_ends_before_turns(self):
        profile = synthetic_profile()
        gate = GATE_MODULE.ThreadGate(profile, PROTOCOL, CONTRACT)
        gate.accept_response(synthetic_response(profile))
        boundary = gate.finish()
        self.assertEqual(boundary["threadId"], "synthetic-native-cwd-thread")
        self.assertTrue(boundary["responseValidated"])
        self.assertEqual(boundary["turnsRequested"], 0)
        self.assertEqual(boundary["toolsRequested"], 0)
        with self.assertRaisesRegex(ValueError, "rpc-after-thread-terminal"):
            client.admit_rpc("turn/start", client.METHODS, False, gate.validated)

    def test_fresh_authority_and_freeze_binding_preserve_native_identity_contract(self):
        grant = copy.deepcopy(AUTHORITY["grant"])
        slot = {
            "owner": "Scientist", "key": client.KEY, "status": client.SLOT_STATUS,
            "grantIssuedUtc": client.ISSUED, "assignedUtc": client.ISSUED,
        }
        self.assertEqual(client.validate_live_authority(grant, slot, copy.deepcopy(grant)), client.ISSUED)

        request = {
            "key": client.KEY, "grantIssuedUtc": client.ISSUED,
            "sourceCommit": "a" * 40, "slotAssignedUtc": client.ISSUED,
        }
        freeze = {**request, "frozenAtUtc": client.ISSUED}
        client.validate_binding_identity(request, freeze, client.ISSUED)
        with self.assertRaisesRegex(ValueError, "slot-freeze-binding-mismatch"):
            client.validate_binding_identity(
                request, {**freeze, "slotAssignedUtc": "2026-10-08T18:24:12Z"}, client.ISSUED,
            )
        changed = copy.deepcopy(grant)
        changed["cwdIdentityContract"]["fileIndex"] += 1
        with self.assertRaisesRegex(ValueError, "live-grant-mismatch"):
            client.validate_live_authority(changed, slot, grant)


if __name__ == "__main__":
    unittest.main(verbosity=2)
