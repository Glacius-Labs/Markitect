"""Pure tests for classified-client.py; never launches a subprocess or app-server."""
import hashlib
import json
from pathlib import Path
import runpy
import tempfile
import unittest
import zipfile

ROOT = Path(__file__).resolve().parent
MODULE = runpy.run_path(str(ROOT / "classified-client.py"))
classify = MODULE["classify_server_frame"]
load_enums = MODULE["load_method_enums"]
validate_response = MODULE["validate_response_envelope"]
validate_grant = MODULE["validate_authorization_grant"]
validate_history = MODULE["validate_prior_consumption"]
validate_authref = MODULE["validate_authorization_reference"]
ENUMS = load_enums()


def resolve_refs(node, definitions):
    if isinstance(node, dict):
        if "$ref" in node and node["$ref"].startswith("#/definitions/"):
            return resolve_refs(definitions[node["$ref"].split("/")[-1]], definitions)
        return {k: resolve_refs(v, definitions) for k, v in node.items()}
    if isinstance(node, list):
        return [resolve_refs(value, definitions) for value in node]
    return node


class ClassificationTests(unittest.TestCase):
    def test_frozen_maps_match_exact_archived_protocol_union(self):
        source = ENUMS["source"]
        archive = (ROOT / source["archive"]).resolve()
        data = archive.read_bytes()
        self.assertEqual(hashlib.sha256(data).hexdigest(), source["archiveSha256"])
        with zipfile.ZipFile(archive) as schema_zip:
            for key, path in (("notificationSchemaSha256", "ServerNotification.json"),
                              ("requestSchemaSha256", "ServerRequest.json"),
                              ("requestIdSchemaSha256", "RequestId.json")):
                self.assertEqual(hashlib.sha256(schema_zip.read(path)).hexdigest(), source[key])
            for filename, kind in (("ServerNotification.json", "ServerNotification"),
                                   ("ServerRequest.json", "ServerRequest")):
                schema = json.loads(schema_zip.read(filename))
                extracted = {}
                for member in schema["oneOf"]:
                    method = member["properties"]["method"]["enum"][0]
                    extracted[method] = {"class": member["title"], "required": member.get("required", [])}
                    if kind == "ServerNotification" and method in ENUMS["discardableNotifications"]:
                        extracted[method]["paramsSchema"] = resolve_refs(member["properties"]["params"], schema["definitions"])
                self.assertEqual(extracted, ENUMS["methods"][kind])
                if kind == "ServerNotification":
                    self.assertEqual(schema.get("properties", {}), ENUMS["notificationEnvelopeOptionalProperties"])
        self.assertEqual(ENUMS["methodCounts"], {k: len(v) for k, v in ENUMS["methods"].items()})
        self.assertEqual(len(ENUMS["methods"]["ServerNotification"]), 83)
        self.assertEqual(len(ENUMS["methods"]["ServerRequest"]), 11)

    def test_authorization_copy_is_bound_to_original_coordinator_grant(self):
        grant_path = ROOT / "authorization-grant.json"
        if not grant_path.exists():
            self.skipTest("root has not staged the local authorization grant yet")
        binding = {"path": str(grant_path), "sha256": hashlib.sha256(grant_path.read_bytes()).hexdigest()}
        self.assertEqual(validate_grant({"authorizationGrant": binding})[0], binding["sha256"])
        doc = json.loads(grant_path.read_bytes())
        doc["grant"]["maxAdditionalAppServerSessions"] = 2
        with tempfile.TemporaryDirectory() as temp:
            altered = Path(temp) / "altered-grant.json"
            altered.write_text(json.dumps(doc), encoding="utf-8")
            altered_binding = {"path": str(altered), "sha256": hashlib.sha256(altered.read_bytes()).hexdigest()}
            with self.assertRaises(RuntimeError):
                validate_grant({"authorizationGrant": altered_binding})

    def test_prior_consumption_must_match_authorized_and_reserved_history(self):
        expected = MODULE["EXPECTED_PRIOR_CONSUMPTION"]
        authorization = {"priorConsumption": dict(expected)}
        reservation = {"priorConsumption": dict(expected)}
        validate_history(authorization, reservation)
        for altered in ({**expected, "actorStarts": 6}, {**expected, "historicalActorTokenTotal": 0}):
            with self.subTest(altered=altered):
                with self.assertRaises(RuntimeError):
                    validate_history({"priorConsumption": altered}, reservation)
                with self.assertRaises(RuntimeError):
                    validate_history(authorization, {"priorConsumption": altered})

    def test_authorization_reference_policy_matches_the_fixed_grant_boundary(self):
        expected = {
            "allowedInboundNotifications": MODULE["EXPECTED_ALLOWED_INBOUND"],
            "warningPolicy": MODULE["EXPECTED_WARNING_POLICY"],
            "allServerRequests": MODULE["EXPECTED_ALL_SERVER_REQUESTS_POLICY"],
        }
        grant = {"allowedInboundNotifications": MODULE["EXPECTED_ALLOWED_INBOUND"],
                 "warningResultRule": "Non-deprecation allowed warnings make any policy result provisional and unusable as effective exec-policy/access proof.",
                 "abortOn": "Every server request; no reply."}
        validate_authref(expected, grant)
        for changed in (
            {**expected, "allowedInboundNotifications": ["warning"]},
            {**expected, "warningPolicy": "interpret warning text"},
            {**expected, "allServerRequests": "reply to currentTime/read"},
        ):
            with self.subTest(changed=changed):
                with self.assertRaises(RuntimeError):
                    validate_authref(changed, grant)

    def test_warning_is_discarded_and_only_method_class_is_retained(self):
        result = classify({"method": "warning", "params": {"message": "private warning text", "threadId": None}}, ENUMS, 0)
        self.assertEqual(result["action"], "discard-notification")
        self.assertTrue(result["provisional"])
        self.assertEqual(result["event"], {"method": "warning", "class": "WarningNotification"})
        self.assertNotIn("private warning text", json.dumps(result))
        self.assertEqual(result["notificationCount"], 1)

    def test_optional_notification_timestamp_obeys_frozen_signed_int64_shape(self):
        valid = classify({"method": "warning", "emittedAtMs": -1, "params": {"message": "x"}}, ENUMS, 0)
        self.assertEqual(valid["action"], "discard-notification")
        for timestamp in (True, 1.5, -9223372036854775809, 9223372036854775808):
            with self.subTest(timestamp=timestamp):
                result = classify({"method": "warning", "emittedAtMs": timestamp, "params": {"message": "x"}}, ENUMS, 0)
                self.assertEqual(result["reason"], "malformed-notification-envelope")

    def test_config_warning_validates_nested_range_but_does_not_interpret_it(self):
        result = classify({"method": "configWarning", "params": {
            "summary": "uninterpreted", "details": None, "path": None,
            "range": {"start": {"line": 1, "column": 2}, "end": {"line": 3, "column": 4}},
        }}, ENUMS, 0)
        self.assertEqual(result["action"], "discard-notification")
        self.assertTrue(result["provisional"])
        self.assertEqual(result["event"]["method"], "configWarning")
        self.assertNotIn("uninterpreted", json.dumps(result))

    def test_deprecation_notice_is_discardable_without_making_config_provisional(self):
        result = classify({"method": "deprecationNotice", "params": {"summary": "ignored", "details": None}}, ENUMS, 0)
        self.assertEqual(result["action"], "discard-notification")
        self.assertFalse(result["provisional"])
        self.assertEqual(result["event"]["class"], "DeprecationNoticeNotification")

    def test_world_writable_notice_schema_is_checked_and_marks_provisional(self):
        valid = {"method": "windows/worldWritableWarning", "params": {"extraCount": 0, "failedScan": False, "samplePaths": ["C:\\safe"]}}
        result = classify(valid, ENUMS, 0)
        self.assertEqual(result["action"], "discard-notification")
        self.assertTrue(result["provisional"])
        invalid = {"method": "windows/worldWritableWarning", "params": {"extraCount": -1, "failedScan": False, "samplePaths": []}}
        stopped = classify(invalid, ENUMS, 0)
        self.assertEqual(stopped["reason"], "malformed-discardable-notification")
        self.assertTrue(stopped["provisional"])

    def test_malformed_exception_notifications_stop_without_payload_retention(self):
        cases = [
            {"method": "warning", "params": {}},
            {"method": "configWarning", "params": {"summary": 7}},
            {"method": "deprecationNotice", "params": {"summary": "ok", "details": 9}},
            {"method": "windows/worldWritableWarning", "params": {"extraCount": 0, "failedScan": False}},
        ]
        for frame in cases:
            with self.subTest(method=frame["method"]):
                result = classify(frame, ENUMS, 0)
                self.assertEqual(result["action"], "stop")
                self.assertEqual(result["reason"], "malformed-discardable-notification")
                self.assertNotIn("params", json.dumps(result))

    def test_all_server_requests_are_classified_and_left_unanswered(self):
        for method, metadata in ENUMS["methods"]["ServerRequest"].items():
            with self.subTest(method=method):
                result = classify({"id": 7, "method": method, "params": {}}, ENUMS, 0)
                self.assertEqual(result["action"], "stop")
                self.assertEqual(result["reason"], "unanswered-server-request")
                self.assertEqual(result["event"], {"method": method, "class": metadata["class"]})
                self.assertEqual(result["notificationCount"], 0)

    def test_auth_approval_request_and_time_request_stop_without_response_path(self):
        for method in ("account/chatgptAuthTokens/refresh", "item/permissions/requestApproval", "currentTime/read"):
            with self.subTest(method=method):
                result = classify({"id": "r1", "method": method, "params": {}}, ENUMS, 0)
                self.assertEqual(result["reason"], "unanswered-server-request")
                self.assertEqual(result["event"]["method"], method)

    def test_request_id_and_frame_kind_must_match_the_union(self):
        for identifier in (True, 1.5, None, 9223372036854775808, -9223372036854775809):
            with self.subTest(identifier=identifier):
                self.assertEqual(classify({"id": identifier, "method": "currentTime/read"}, ENUMS, 0)["reason"], "malformed-request-id")
        request_without_id = classify({"method": "currentTime/read", "params": {}}, ENUMS, 0)
        self.assertEqual(request_without_id["reason"], "unexpected-or-malformed-notification")
        notification_with_id = classify({"id": 1, "method": "warning", "params": {"message": "x"}}, ENUMS, 0)
        self.assertEqual(notification_with_id["reason"], "unexpected-or-malformed-request")

    def test_unknown_and_non_discardable_notifications_stop(self):
        unknown = classify({"method": "unknown/method", "params": {"secret": "x"}}, ENUMS, 0)
        self.assertEqual(unknown["reason"], "unexpected-or-malformed-notification")
        self.assertIsNone(unknown["event"])
        ordinary = next(k for k in ENUMS["methods"]["ServerNotification"] if k not in {"warning", "configWarning", "deprecationNotice", "windows/worldWritableWarning"})
        result = classify({"method": ordinary, "params": {}}, ENUMS, 0)
        self.assertEqual(result["reason"], "non-discardable-notification")
        self.assertEqual(result["event"]["method"], ordinary)

    def test_notification_cap_includes_notifications_seen_around_rpc_boundaries(self):
        frame = {"method": "deprecationNotice", "params": {"summary": "x"}}
        for count in range(15):
            self.assertEqual(classify(frame, ENUMS, count)["action"], "discard-notification")
        sixteenth = classify(frame, ENUMS, 15)
        self.assertEqual(sixteenth["action"], "discard-notification")
        self.assertEqual(sixteenth["notificationCount"], 16)
        seventeenth = classify(frame, ENUMS, sixteenth["notificationCount"])
        self.assertEqual(seventeenth["reason"], "notification-limit")
        self.assertEqual(seventeenth["notificationCount"], 17)

    def test_malformed_frame_has_no_retained_payload_or_identifier(self):
        result = classify({"id": "secret-id", "params": {"secret": "secret-value"}}, ENUMS, 0)
        self.assertEqual(result["reason"], "malformed-server-frame")
        serialized = json.dumps(result)
        self.assertNotIn("secret-id", serialized)
        self.assertNotIn("secret-value", serialized)

    def test_response_envelope_rejects_mixed_frames_bool_ids_and_result_error_pairs(self):
        self.assertEqual(validate_response({"id": 1, "result": {"safe": True}}, 1), {"safe": True})
        for frame, identifier in (
            ({"id": True, "result": {}}, 1),
            ({"id": 1, "result": {}, "error": {"code": -1}}, 1),
            ({"id": 1}, 1),
            ({"id": 1, "method": "currentTime/read", "result": {}}, 1),
        ):
            with self.subTest(frame=frame):
                with self.assertRaises(ValueError):
                    validate_response(frame, identifier)


if __name__ == "__main__":
    unittest.main()
