"""Offline contract tests for the bounded common-runner read-tool packet.

These exercise the actual collector and transport helpers with synthetic
protocol values. They never start an app-server, Actor, or subprocess.
"""
from __future__ import annotations

import copy
import importlib.util
import json
from pathlib import Path
import unittest


HERE = Path(__file__).resolve().parent
PACKAGE = HERE.parents[1]


def load_file(name: str, path: Path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


client = load_file("read_tool_client_tested", HERE / "client.py")
collector_module = load_file("read_tool_collector_tested", HERE / "collector.py")
protocol_module = load_file("read_tool_protocol_tested", HERE / "protocol.py")
PROFILE = json.loads((HERE / "profile.json").read_text(encoding="utf-8"))
EXPECTED = json.loads((HERE / "expected.json").read_text(encoding="utf-8"))
CONTRACT = json.loads((HERE / "schema-contract.json").read_text(encoding="utf-8"))
ARCHIVE = PACKAGE / "evidence/policy-compatibility/run-1/protocol-schema.zip"


def make_collector(*, profile=None, validate=None):
    return collector_module.Collector(
        copy.deepcopy(profile if profile is not None else PROFILE),
        copy.deepcopy(EXPECTED),
        validate if validate is not None else (lambda _member, _value: True),
    )


def thread_response(*, permission=True, sandbox=None):
    response = {
        "cwd": PROFILE["cwd"], "model": "gpt-6.1-sol", "modelProvider": "openai",
        "approvalPolicy": "never",
        "thread": {"id": "thread-r1", "cwd": PROFILE["cwd"], "model": "gpt-6.1-sol",
                   "modelProvider": "openai", "ephemeral": True,
                   "parentThreadId": None, "forkedFromId": None},
        "sandbox": sandbox if sandbox is not None else {"type": "readOnly", "networkAccess": False},
    }
    if permission is True:
        response["activePermissionProfile"] = {"id": ":read-only", "extends": None}
    elif permission is not False:
        response["activePermissionProfile"] = permission
    return response


def user_item():
    return {"id": "user-1", "type": "userMessage",
            "content": [{"type": "text", "text": PROFILE["turnPrompt"]}]}


def command_item(*, command=None, cwd=None, status="completed", exit_code=0, output=None, item_id="cmd-1"):
    return {"id": item_id, "type": "commandExecution",
            "command": command if command is not None else PROFILE["command"],
            "cwd": cwd if cwd is not None else PROFILE["cwd"], "status": status,
            "exitCode": exit_code,
            "aggregatedOutput": output if output is not None else EXPECTED["sentinel"],
            "source": "agent", "commandActions": []}


def final_item(text=None, *, phase="final", item_id="answer-1"):
    return {"id": item_id, "type": "agentMessage",
            "text": text if text is not None else json.dumps({"sentinel": EXPECTED["sentinel"]}),
            "phase": phase}


def response_turn(*, turn_id="turn-r1", status="inProgress", items=None):
    return {"turn": {"id": turn_id, "status": status,
                      "items": items if items is not None else [user_item()]}}


def notify(c, method, params):
    c.accept_notification(method, params)


def establish(c, *, turn_status="inProgress"):
    c.accept_response("thread/start", thread_response())
    notify(c, "thread/started", {"thread": {"id": "thread-r1", "cwd": PROFILE["cwd"],
          "model": "gpt-6.1-sol", "modelProvider": "openai", "ephemeral": True,
          "parentThreadId": None, "forkedFromId": None}})
    c.accept_response("turn/start", response_turn(status=turn_status))
    notify(c, "turn/started", {"threadId": "thread-r1", "turn": {
          "id": "turn-r1", "status": "inProgress", "items": []}})


def command_complete(c, item=None, *, timestamp=1):
    notify(c, "item/completed", {"threadId": "thread-r1", "turnId": "turn-r1",
          "completedAtMs": timestamp, "item": item if item is not None else command_item()})


def terminal(c, *, items=None):
    notify(c, "turn/completed", {"threadId": "thread-r1", "turn": {
          "id": "turn-r1", "status": "completed",
          "items": items if items is not None else [final_item()]}})


def usage_notification(*, input_tokens=10, output_tokens=5, cached=0, reasoning=0,
                       cache_write=0, total=None, context=100000):
    return {"threadId": "thread-r1", "turnId": "turn-r1", "tokenUsage": {
        "modelContextWindow": context,
        "total": {"inputTokens": input_tokens, "outputTokens": output_tokens,
                  "cachedInputTokens": cached, "reasoningOutputTokens": reasoning,
                  "cacheWriteInputTokens": cache_write,
                  "totalTokens": total if total is not None else input_tokens + output_tokens},
    }}


def schema_thread():
    return {"cliVersion": "offline-test", "createdAt": 1, "cwd": PROFILE["cwd"],
            "ephemeral": True, "id": "thread-r1", "model": "gpt-6.1-sol",
            "modelProvider": "openai", "preview": "", "projectId": None,
            "sessionId": "session-r1", "source": "appServer",
            "status": {"type": "active", "activeFlags": []}, "turns": [],
            "updatedAt": 1, "parentThreadId": None, "forkedFromId": None}


def schema_thread_response():
    return {"approvalPolicy": "never", "approvalsReviewer": "user",
            "cwd": PROFILE["cwd"], "model": "gpt-6.1-sol", "modelProvider": "openai",
            "sandbox": {"type": "readOnly", "networkAccess": False},
            "activePermissionProfile": {"id": ":read-only", "extends": None},
            "thread": schema_thread()}


def schema_command(*, status="completed", output=None, exit_code=0):
    return {"id": "cmd-1", "type": "commandExecution", "command": PROFILE["command"],
            "commandActions": [], "cwd": PROFILE["cwd"], "status": status,
            "source": "agent", "exitCode": exit_code, "aggregatedOutput": output}


def schema_usage_notification(*, include_last=True):
    def breakdown():
        return {"inputTokens": 8, "outputTokens": 4, "cachedInputTokens": 1,
                "reasoningOutputTokens": 2, "totalTokens": 12, "cacheWriteInputTokens": 0}
    usage = {"total": breakdown(), "modelContextWindow": 100000}
    if include_last:
        usage["last"] = breakdown()
    return {"threadId": "thread-r1", "turnId": "turn-r1", "tokenUsage": usage}


def establish_actual(c, *, turn_error=None):
    c.accept_response("thread/start", schema_thread_response())
    notify(c, "thread/started", {"thread": schema_thread()})
    turn = {"id": "turn-r1", "status": "inProgress", "items": [user_item()]}
    if turn_error is not None:
        turn["error"] = turn_error
    c.accept_response("turn/start", {"turn": turn})
    notify(c, "turn/started", {"threadId": "thread-r1", "turn": {
          "id": "turn-r1", "status": "inProgress", "items": []}})


def complete_actual(c, *, command_output=None):
    notify(c, "turn/completed", {"threadId": "thread-r1", "turn": {
          "id": "turn-r1", "status": "completed", "items": [
              {"id": "answer-1", "type": "agentMessage",
               "text": json.dumps({"sentinel": EXPECTED["sentinel"]}),
               "phase": "final_answer"},
          ]}})


class CollectorLifecycleTests(unittest.TestCase):
    def test_full_success_requires_correlated_thread_command_and_completed_turn(self):
        c = make_collector()
        establish(c)
        command_complete(c)
        terminal(c)
        receipt = c.finish()
        self.assertEqual(receipt["status"], "complete")
        self.assertEqual(receipt["safePublicOutput"], EXPECTED["sentinel"])
        self.assertEqual(receipt["finalSentinel"], EXPECTED["sentinel"])
        self.assertEqual(receipt["itemIds"]["commandExecution"], "cmd-1")

    def test_permission_profile_and_effective_readonly_sandbox_are_both_required(self):
        cases = [
            (None, {"type": "workspaceWrite", "networkAccess": False}),
            (None, {"type": "readOnly", "networkAccess": True}),
            ({"id": ":workspace-write", "extends": None}, {"type": "readOnly", "networkAccess": False}),
            ({"id": ":read-only", "extends": "workspace"}, {"type": "readOnly", "networkAccess": False}),
        ]
        for profile, sandbox in cases:
            with self.subTest(profile=profile, sandbox=sandbox):
                c = make_collector()
                c.accept_response("thread/start", thread_response(permission=profile, sandbox=sandbox))
                self.assertEqual(c.failure_reason, "thread-response-permission-provenance-unknown"
                                 if profile is None else "thread-response-permission-profile-mismatch")

    def test_explicit_readonly_profile_and_null_or_absent_parent_are_allowed(self):
        for profile in ({"id": ":read-only", "extends": None},
                        {"id": ":read-only", "extends": ""}):
            c = make_collector()
            c.accept_response("thread/start", thread_response(permission=profile))
            self.assertIsNone(c.failure_reason)
        c = make_collector()
        c.accept_response("thread/start", thread_response(permission=False,
            sandbox={"type": "readOnly", "networkAccess": False}))
        self.assertIsNone(c.failure_reason)
        c = make_collector()
        c.accept_response("thread/start", thread_response(permission=None,
            sandbox={"type": "readOnly", "networkAccess": False}))
        self.assertIsNone(c.failure_reason)

    def test_early_turn_event_must_match_start_response_id(self):
        c = make_collector()
        notify(c, "turn/started", {"threadId": "thread-r1", "turn": {
            "id": "early-turn", "status": "inProgress", "items": []}})
        c.accept_response("thread/start", thread_response())
        c.accept_response("turn/start", response_turn(turn_id="different-turn"))
        self.assertEqual(c.failure_reason, "turn-response-id-conflicts-with-early-event")

    def test_start_response_completed_is_not_terminal_proof(self):
        c = make_collector()
        establish(c, turn_status="completed")
        self.assertFalse(c.terminal)
        self.assertEqual(c.finish()["status"], "failed")

    def test_user_final_answer_without_completed_native_command_is_insufficient(self):
        c = make_collector()
        establish(c)
        terminal(c)
        receipt = c.finish()
        self.assertEqual(receipt["status"], "failed")
        self.assertNotEqual(receipt["safePublicOutput"], EXPECTED["sentinel"])

    def test_successful_command_without_final_sentinel_is_insufficient(self):
        c = make_collector()
        establish(c)
        command_complete(c)
        terminal(c, items=[])
        self.assertEqual(c.finish()["status"], "failed")

    def test_command_must_match_exact_frozen_command_cwd_and_zero_exit(self):
        variants = [
            command_item(command=PROFILE["command"] + "; whoami"),
            command_item(cwd=PROFILE["cwd"] + "/other"),
            command_item(exit_code=1),
            command_item(status="declined", exit_code=None),
            command_item(output="wrong-output"),
        ]
        for item in variants:
            with self.subTest(item=item):
                c = make_collector()
                establish(c)
                command_complete(c, item)
                self.assertIsNotNone(c.failure_reason)
                self.assertEqual(c.finish()["commandExecutionItemCount"], 1)

    def test_second_native_command_is_rejected(self):
        c = make_collector()
        establish(c)
        command_complete(c)
        command_complete(c, command_item(item_id="cmd-2"), timestamp=2)
        self.assertEqual(c.failure_reason, "multiple-command-items")
        self.assertEqual(c.finish()["commandExecutionItemCount"], 2)

    def test_second_native_command_is_rejected_at_first_observation(self):
        c = make_collector()
        establish(c)
        for item_id, timestamp in (("cmd-1", 1), ("cmd-2", 2)):
            item = command_item(item_id=item_id, status="inProgress", exit_code=None, output=None)
            notify(c, "item/started", {"threadId": "thread-r1", "turnId": "turn-r1",
                  "startedAtMs": timestamp, "item": item})
            if c.failure_reason:
                break
        self.assertEqual(c.failure_reason, "multiple-command-items")
        self.assertEqual(c.finish()["commandExecutionItemCount"], 2)

    def test_second_native_command_in_terminal_snapshot_is_rejected(self):
        c = make_collector()
        establish(c)
        terminal(c, items=[command_item(), command_item(item_id="cmd-2")])
        self.assertEqual(c.failure_reason, "multiple-command-items")
        self.assertEqual(c.finish()["commandExecutionItemCount"], 2)

    def test_only_completed_turn_notification_closes_collection(self):
        c = make_collector()
        establish(c)
        command_complete(c)
        terminal(c)
        self.assertTrue(c.terminal)
        notify(c, "item/started", {"threadId": "thread-r1", "turnId": "turn-r1",
              "startedAtMs": 2, "item": {"id": "late", "type": "reasoning"}})
        self.assertEqual(c.failure_reason, "notification-after-terminal-turn")

    def test_missing_tool_or_wrong_correlated_ids_fail_closed(self):
        c = make_collector()
        establish(c)
        notify(c, "item/completed", {"threadId": "other-thread", "turnId": "turn-r1",
              "completedAtMs": 1, "item": command_item()})
        self.assertEqual(c.failure_reason, "thread-id-mismatch")

        c = make_collector()
        establish(c)
        notify(c, "item/completed", {"threadId": "thread-r1", "turnId": "other-turn",
              "completedAtMs": 1, "item": command_item()})
        self.assertEqual(c.failure_reason, "turn-id-mismatch")

    def test_unknown_notifications_and_notification_limit_stop(self):
        c = make_collector()
        c.accept_notification("server/request", {})
        self.assertEqual(c.failure_reason, "unknown-notification")

        small = copy.deepcopy(PROFILE)
        small["limits"]["maxInboundNotifications"] = 1
        c = make_collector(profile=small)
        notify(c, "thread/started", {"thread": {"id": "thread-r1", "cwd": PROFILE["cwd"],
              "model": "gpt-6.1-sol", "modelProvider": "openai", "ephemeral": True,
              "parentThreadId": None, "forkedFromId": None}})
        notify(c, "thread/started", {"thread": {"id": "thread-r1", "cwd": PROFILE["cwd"],
              "model": "gpt-6.1-sol", "modelProvider": "openai", "ephemeral": True,
              "parentThreadId": None, "forkedFromId": None}})
        self.assertEqual(c.failure_reason, "notification-limit")


class UsageAndBoundaryTests(unittest.TestCase):
    def test_latest_cumulative_usage_snapshots_are_not_added(self):
        c = make_collector()
        establish(c)
        notify(c, "thread/tokenUsage/updated", usage_notification(input_tokens=20000, output_tokens=10000))
        notify(c, "thread/tokenUsage/updated", usage_notification(input_tokens=26000, output_tokens=10000,
              cached=20000, reasoning=5000, cache_write=100))
        self.assertIsNone(c.failure_reason)
        latest = c.finish()
        self.assertEqual(latest["usage"]["inputTokens"], 26000)
        self.assertIsNone(latest["usageOvershoot"])

    def test_threshold_uses_input_plus_output_and_records_overshoot(self):
        c = make_collector()
        establish(c)
        notify(c, "thread/tokenUsage/updated", usage_notification(input_tokens=50000, output_tokens=0))
        self.assertIsNone(c.failure_reason)
        notify(c, "thread/tokenUsage/updated", usage_notification(input_tokens=50001, output_tokens=0))
        self.assertEqual(c.failure_reason, "token-observation-threshold")
        self.assertEqual(c.finish()["usageOvershoot"], 1)

    def test_missing_usage_remains_unknown_in_safe_receipt(self):
        c = make_collector()
        establish(c)
        receipt = c.finish()
        self.assertIsNone(receipt["usage"])
        self.assertFalse(receipt["usageSeen"])

    def test_usage_bool_negative_and_int64_overflow_are_rejected(self):
        invalid = [
            usage_notification(input_tokens=True),
            usage_notification(input_tokens=-1),
            usage_notification(input_tokens=1 << 63),
            usage_notification(context=True),
            usage_notification(context=1 << 63),
        ]
        for params in invalid:
            with self.subTest(params=params):
                c = make_collector()
                establish(c)
                notify(c, "thread/tokenUsage/updated", params)
                self.assertIsNotNone(c.failure_reason)

    def test_timestamp_and_reasoning_indices_reject_bool_and_int64_overflow(self):
        for timestamp in (True, 1 << 63):
            c = make_collector()
            establish(c)
            notify(c, "item/completed", {"threadId": "thread-r1", "turnId": "turn-r1",
                  "completedAtMs": timestamp, "item": command_item()})
            self.assertEqual(c.failure_reason, "invalid-item-timestamp")

        for index in (False, 1 << 63):
            c = make_collector()
            establish(c)
            notify(c, "item/started", {"threadId": "thread-r1", "turnId": "turn-r1",
                  "startedAtMs": 1, "item": {"id": "reason-1", "type": "reasoning"}})
            notify(c, "item/reasoning/textDelta", {"threadId": "thread-r1", "turnId": "turn-r1",
                  "itemId": "reason-1", "contentIndex": index, "delta": "x"})
            self.assertEqual(c.failure_reason, "invalid-reasoning-index")


class TransportAndSchemaTests(unittest.TestCase):
    def test_duplicate_json_keys_and_nonstandard_numbers_are_rejected(self):
        for raw in (b'{"method":"x","method":"y"}', b'{"n":NaN}', b'{"n":1e999}'):
            with self.subTest(raw=raw), self.assertRaises(ValueError):
                client.strict_json(raw)

    def test_protocol_constructs_from_pinned_archive_and_rejects_bad_member_values(self):
        protocol = protocol_module.Protocol(ARCHIVE, CONTRACT)
        with self.assertRaises(ValueError):
            protocol.validate("v2/ThreadStartResponse.json", {"thread": []})
        with self.assertRaises(ValueError):
            protocol.validate("v2/NotFrozen.json", {})
        with self.assertRaises(ValueError):
            protocol.validate("v2/TurnStartResponse.json", {"turn": {"id": 1,
                "status": "inProgress", "items": []}})

    def test_interrupt_requires_current_validated_owned_ids_and_only_one_send(self):
        ids = {"threadId": "thread-r1", "turnId": "turn-r1"}
        expected = {"method": "turn/interrupt", "id": 5, "params": ids}
        self.assertEqual(client.interrupt_request(ids, True, False), expected)
        for active, validated, sent in (
            ({}, True, False), (ids, False, False), (ids, True, True),
            ({"threadId": "thread-r1"}, True, False),
            ({"threadId": "thread-r1", "turnId": "x" * 129}, True, False),
            ({"threadId": "thread-r1", "turnId": True}, True, False),
        ):
            with self.subTest(active=active, validated=validated, sent=sent):
                self.assertIsNone(client.interrupt_request(active, validated, sent))

    def test_route_notification_rejects_server_requests_bad_envelopes_and_unknowns(self):
        protocol = object()
        collector = make_collector()
        enums = {}
        for frame in (
            {"method": "thread/started", "params": {}, "id": 9},
            {"method": "thread/started", "params": {}, "emittedAtMs": True},
            {"method": "unknown/event", "params": {}},
        ):
            with self.subTest(frame=frame), self.assertRaises(ValueError):
                client.route_notification(frame, protocol, collector, enums, metadata_only=False)

    def test_disabled_remote_status_is_reduced_and_does_not_reach_collector(self):
        prior = client.load_module(client.PRIOR, client.PRIOR_SHA, "read_tool_prior_test")
        enums = prior.load_method_enums(HERE / "frozen-method-enums.json")
        collector = make_collector()
        params = {"status": "disabled", "installationId": "install-id",
                  "serverName": "server-name", "environmentId": "environment-id"}
        frame = {"method": "remoteControl/status/changed", "params": params}
        self.assertEqual(client.route_notification(frame, object(), collector, enums, metadata_only=True),
                         {"method": "remoteControl/status/changed", "status": "disabled"})
        self.assertEqual(collector._notification_count, 0)

    def test_raw_notification_requires_frozen_protocol_validation_and_safe_reduction(self):
        class ProtocolStub:
            def validate(self, _member, _value):
                raise ValueError("bad schema")

        c = make_collector(validate=ProtocolStub().validate)
        frame = {"method": "thread/started", "params": {"thread": {}}}
        with self.assertRaises(ValueError):
            client.route_notification(frame, ProtocolStub(), c, {}, metadata_only=False)

    def test_lifecycle_notifications_are_blocked_during_metadata_handshake(self):
        c = make_collector()
        frame = {"method": "thread/started", "params": {"thread": {"id": "thread-r1"}}}
        with self.assertRaises(ValueError):
            client.route_notification(frame, object(), c, {}, metadata_only=True)


class ActualSchemaIntegrationTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.protocol = protocol_module.Protocol(ARCHIVE, CONTRACT)

    def actual_collector(self):
        return make_collector(validate=self.protocol.validate)

    def test_full_schema_valid_lifecycle_with_usage_and_trailing_newline_delta(self):
        c = self.actual_collector()
        establish_actual(c)
        notify(c, "thread/tokenUsage/updated", schema_usage_notification())
        notify(c, "item/started", {"threadId": "thread-r1", "turnId": "turn-r1",
              "startedAtMs": 1, "item": schema_command(status="inProgress", output=None, exit_code=None)})
        sentinel = EXPECTED["sentinel"]
        notify(c, "item/commandExecution/outputDelta", {"threadId": "thread-r1", "turnId": "turn-r1",
              "itemId": "cmd-1", "delta": sentinel[:12]})
        notify(c, "item/commandExecution/outputDelta", {"threadId": "thread-r1", "turnId": "turn-r1",
              "itemId": "cmd-1", "delta": sentinel[12:] + "\r\n"})
        notify(c, "item/completed", {"threadId": "thread-r1", "turnId": "turn-r1",
              "completedAtMs": 2, "item": schema_command(output=sentinel + "\r\n")})
        complete_actual(c)
        receipt = c.finish()
        self.assertEqual(receipt["status"], "complete")
        self.assertTrue(receipt["commandCompletionEventObserved"])
        self.assertEqual(receipt["usage"]["inputTokens"], 8)
        self.assertEqual(receipt["safePublicOutput"], sentinel)

    def test_actual_schema_rejects_missing_required_usage_last_breakdown(self):
        c = self.actual_collector()
        establish_actual(c)
        notify(c, "thread/tokenUsage/updated", schema_usage_notification(include_last=False))
        self.assertEqual(c.failure_reason, "schema-rejected")

    def test_terminal_command_snapshot_alone_is_not_an_item_completion_event(self):
        c = self.actual_collector()
        establish_actual(c)
        notify(c, "thread/tokenUsage/updated", schema_usage_notification())
        # The completed turn snapshot reports a successful command, but no
        # item/completed event ever confirms that native command lifecycle.
        notify(c, "turn/completed", {"threadId": "thread-r1", "turn": {
              "id": "turn-r1", "status": "completed", "items": [
                  schema_command(output=EXPECTED["sentinel"]),
                  {"id": "answer-1", "type": "agentMessage",
                   "text": json.dumps({"sentinel": EXPECTED["sentinel"]}), "phase": "final_answer"},
              ]}})
        receipt = c.finish()
        self.assertEqual(receipt["status"], "failed")
        self.assertEqual(receipt["stopReason"], "command-completion-event-missing")
        self.assertFalse(receipt["commandCompletionEventObserved"])

    def test_turn_error_prevents_actual_schema_lifecycle_admission(self):
        c = self.actual_collector()
        establish_actual(c, turn_error={"message": "synthetic failure"})
        self.assertEqual(c.failure_reason, "turn-response-has-error")

    def test_actual_protocol_schema_rejects_missing_thread_required_field(self):
        c = self.actual_collector()
        response = schema_thread_response()
        del response["approvalsReviewer"]
        c.accept_response("thread/start", response)
        self.assertEqual(c.failure_reason, "schema-rejected")


if __name__ == "__main__":
    unittest.main(verbosity=2)
