"""Six offline synthetic tests for bounded thread/start response comparisons."""
import copy
import hashlib
import importlib.util
import json
import sys
import unittest
from pathlib import Path


HERE = Path(__file__).resolve().parent
REPO = HERE.parents[3]
PRIOR = REPO / "experiments/government-comparison/evidence/s1-stderr-observation-diagnostic-20261008-r2"
SCHEMA = REPO / "experiments/government-comparison/evidence/s1-runner-rebinding-metadata-20261008-r1/generated-schemas.zip"
PROTOCOL_PATH = REPO / "experiments/government-comparison/evidence/s1-common-runner-read-tool-20261008-r1/protocol.py"
SCHEMA_CONTRACT = json.loads((PRIOR / "schema-contract.json").read_text(encoding="utf-8"))
SCHEMA_SHA = "ebd087657febf63846904d83a75bc226a5412458418d20abe024ed267febb822"
require_pin = hashlib.sha256(PROTOCOL_PATH.read_bytes()).hexdigest() == SCHEMA_SHA
if not require_pin:
    raise RuntimeError("pinned local protocol source changed")
protocol_spec = importlib.util.spec_from_file_location("thread_response_fields_protocol", PROTOCOL_PATH)
protocol_module = importlib.util.module_from_spec(protocol_spec)
protocol_spec.loader.exec_module(protocol_module)

sys.path.insert(0, str(HERE))
from thread_gate import ThreadGate  # noqa: E402
from consumer import ThreadResponseConsumer  # noqa: E402


EXPECTED = {
    "cwd": "C:/Synthetic/AgentRoot",
    "model": "synthetic-model-alpha",
    "modelProvider": "synthetic-provider-alpha",
    "approvalPolicy": "never",
}
PROFILE = {"threadStart": {"params": {
    **EXPECTED, "permissions": ":read-only", "ephemeral": True,
}}}
PROTOCOL = protocol_module.Protocol(SCHEMA, SCHEMA_CONTRACT)
FIELDS = ("cwd", "model", "modelProvider", "approvalPolicy")


def response(**changes):
    result = {
        "approvalPolicy": EXPECTED["approvalPolicy"],
        "approvalsReviewer": "user",
        "cwd": EXPECTED["cwd"],
        "model": EXPECTED["model"],
        "modelProvider": EXPECTED["modelProvider"],
        "sandbox": {"type": "readOnly", "networkAccess": False},
        "activePermissionProfile": {"id": ":read-only", "extends": None},
        "thread": {
            "cliVersion": "offline-synthetic",
            "createdAt": 1,
            "cwd": EXPECTED["cwd"],
            "ephemeral": True,
            "id": "synthetic-thread-01",
            "model": EXPECTED["model"],
            "modelProvider": EXPECTED["modelProvider"],
            "preview": "",
            "projectId": None,
            "sessionId": "synthetic-session-01",
            "source": "appServer",
            "status": {"type": "active", "activeFlags": []},
            "turns": [],
            "updatedAt": 1,
            "parentThreadId": None,
            "forkedFromId": None,
        },
    }
    result.update(changes)
    return result


def consumer():
    return ThreadResponseConsumer(copy.deepcopy(PROFILE), PROTOCOL, SCHEMA_CONTRACT)


def row(field, equal, expected_type="string", actual_type="string"):
    return {"field": field, "expectedType": expected_type,
            "actualType": actual_type, "equal": equal}


class ThreadResponseFieldGateTests(unittest.TestCase):
    def test_each_single_field_mismatch_reports_four_fixed_rows_and_stops(self):
        variants = {
            "cwd": {"cwd": "C:/Synthetic/OtherRoot"},
            "model": {"model": "synthetic-model-beta"},
            "modelProvider": {"modelProvider": "synthetic-provider-beta"},
            "approvalPolicy": {"approvalPolicy": "on-request"},
        }
        for changed_field, override in variants.items():
            with self.subTest(field=changed_field):
                instance = consumer()
                with self.assertRaisesRegex(ValueError, "thread-response-identity-or-policy-mismatch"):
                    instance.accept_response(response(**override))
                diagnostic = instance.diagnostic()
                self.assertEqual(set(diagnostic), {"comparisons", "cwdLexicallyEquivalent"})
                self.assertEqual([item["field"] for item in diagnostic["comparisons"]], list(FIELDS))
                self.assertEqual(
                    diagnostic["comparisons"],
                    [row(field, field != changed_field) for field in FIELDS],
                )
                self.assertIs(type(diagnostic["cwdLexicallyEquivalent"]), bool)
                self.assertNotIn("Synthetic/OtherRoot", json.dumps(diagnostic))
                with self.assertRaisesRegex(ValueError, "response-consumer-terminal"):
                    instance.accept_response(response())

    def test_multiple_mismatches_include_schema_supported_object_policy_and_types(self):
        object_policy = {"granular": {
            "mcp_elicitations": False, "rules": False,
            "sandbox_approval": False,
        }}
        instance = consumer()
        with self.assertRaisesRegex(ValueError, "thread-response-identity-or-policy-mismatch"):
            instance.accept_response(response(
                cwd="C:/Synthetic/AlternateRoot",
                model="synthetic-model-beta",
                approvalPolicy=object_policy,
            ))
        diagnostic = instance.diagnostic()
        self.assertEqual(diagnostic["comparisons"], [
            row("cwd", False), row("model", False), row("modelProvider", True),
            row("approvalPolicy", False, "string", "object"),
        ])
        serialized = json.dumps(diagnostic, sort_keys=True)
        for private_value in ("AlternateRoot", "synthetic-model-beta", "mcp_elicitations"):
            self.assertNotIn(private_value, serialized)

    def test_windows_cwd_spelling_is_metadata_only_and_other_paths_remain_unequal(self):
        examples = (
            (r"C:\Synthetic\AgentRoot", "c:/synthetic/agentroot", True),
            (r"\\SyntheticHost\Share\Root", "//synthetichost/share/root", True),
            ("C:/Synthetic/AgentRoot", "C:/Synthetic/AgentRoot/child", False),
        )
        for expected_cwd, actual_cwd, lexical in examples:
            with self.subTest(lexical=lexical):
                profile = copy.deepcopy(PROFILE)
                profile["threadStart"]["params"]["cwd"] = expected_cwd
                instance = ThreadResponseConsumer(profile, PROTOCOL, SCHEMA_CONTRACT)
                with self.assertRaisesRegex(ValueError, "thread-response-identity-or-policy-mismatch"):
                    instance.accept_response(response(cwd=actual_cwd))
                diagnostic = instance.diagnostic()
                self.assertFalse(diagnostic["comparisons"][0]["equal"])
                self.assertIs(diagnostic["cwdLexicallyEquivalent"], lexical)
                text = json.dumps(diagnostic)
                self.assertNotIn("AgentRoot", text)
                self.assertNotIn("SyntheticHost", text)
                self.assertNotIn("child", text)

    def test_schema_failure_has_no_comparison_and_consumer_stays_terminal(self):
        instance = consumer()
        invalid = response()
        del invalid["sandbox"]
        with self.assertRaises(ValueError):
            instance.accept_response(invalid)
        self.assertIsNone(instance.diagnostic())
        with self.assertRaisesRegex(ValueError, "response-consumer-terminal"):
            instance.accept_response(response())
        self.assertIsNone(instance.diagnostic())

    def test_valid_baseline_and_existing_sandbox_parent_turn_boundaries(self):
        valid = consumer()
        valid.accept_response(response())
        self.assertEqual(valid.diagnostic(), {
            "comparisons": [row(field, True) for field in FIELDS],
            "cwdLexicallyEquivalent": True,
        })

        invalid_cases = (
            response(sandbox={"type": "readOnly", "networkAccess": True}),
            response(thread={**response()["thread"], "parentThreadId": "synthetic-parent"}),
            response(thread={**response()["thread"], "forkedFromId": "synthetic-source"}),
            response(thread={**response()["thread"], "turns": [{
                "id": "synthetic-existing-turn", "status": "completed", "items": [],
            }]}),
            response(thread={**response()["thread"], "ephemeral": False}),
        )
        reasons = (
            "thread-response-effective-sandbox-mismatch",
            "thread-model-or-parent-mismatch",
            "thread-model-or-parent-mismatch",
            "unexpected-existing-turn",
            "thread-identity-mismatch",
        )
        for bad_response, reason in zip(invalid_cases, reasons):
            with self.subTest(reason=reason):
                instance = consumer()
                with self.assertRaisesRegex(ValueError, reason):
                    instance.accept_response(bad_response)
                self.assertEqual(
                    [item["equal"] for item in instance.diagnostic()["comparisons"]],
                    [True, True, True, True],
                )

    def test_failure_keeps_copy_isolated_metadata_and_unknown_notifications_stop(self):
        instance = consumer()
        with self.assertRaisesRegex(ValueError, "thread-response-identity-or-policy-mismatch"):
            instance.accept_response(response(model="synthetic-model-beta"))
        snapshot = instance.diagnostic()
        snapshot["comparisons"][0]["field"] = "tampered"
        self.assertEqual(instance.diagnostic()["comparisons"][0]["field"], "cwd")
        self.assertEqual(set(instance.diagnostic()), {"comparisons", "cwdLexicallyEquivalent"})
        self.assertEqual(len(instance.diagnostic()["comparisons"]), 4)
        with self.assertRaisesRegex(ValueError, "response-consumer-terminal"):
            instance.accept_response(response())

        gate = ThreadGate(copy.deepcopy(PROFILE), PROTOCOL, SCHEMA_CONTRACT)
        with self.assertRaisesRegex(ValueError, "forbidden-or-unknown-notification"):
            gate.accept_notification("thread/prediction/updated", {"threadId": "synthetic-thread-01"})
        self.assertIsNone(gate.finish())


if __name__ == "__main__":
    unittest.main()
