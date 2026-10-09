"""Offline protocol fixtures for the Conventional MCP lifecycle facade."""

from __future__ import annotations

import io
import json
import sys
import unittest
from pathlib import Path

PLAYGROUND = Path(__file__).resolve().parents[1]
if str(PLAYGROUND) not in sys.path:
    sys.path.insert(0, str(PLAYGROUND))

from conventional.mcp import SUPPORTED_PROTOCOL_VERSIONS, serve  # noqa: E402


class FakeService:
    def __init__(self) -> None:
        self.calls: list[tuple[object, ...]] = []
        self.fail = False

    def _record(self, *call: object) -> dict[str, object]:
        self.calls.append(call)
        if self.fail:
            raise ValueError("run handle is unavailable")
        return {"run_id": "run-1", "state": "accepted"}

    def start(self, prompt: str) -> dict[str, object]:
        return self._record("start", prompt)

    def resume(self, run_id: str, prompt: str) -> dict[str, object]:
        return self._record("resume", run_id, prompt)

    def status(self, run_id: str) -> dict[str, object]:
        return self._record("status", run_id)

    def cancel(self, run_id: str) -> dict[str, object]:
        return self._record("cancel", run_id)


def request(method: str, request_id: object, params: dict[str, object] | None = None) -> dict[str, object]:
    message: dict[str, object] = {"jsonrpc": "2.0", "id": request_id, "method": method}
    if params is not None:
        message["params"] = params
    return message


def initialized_sequence(version: str = "2025-11-25") -> list[dict[str, object]]:
    return [
        request(
            "initialize",
            "init",
            {"protocolVersion": version, "capabilities": {}, "clientInfo": {"name": "fixture", "version": "1"}},
        ),
        {"jsonrpc": "2.0", "method": "notifications/initialized"},
    ]


def run_messages(service: FakeService, messages: list[object]) -> list[dict[str, object]]:
    incoming = "".join(json.dumps(message) + "\n" for message in messages)
    outgoing = io.StringIO()
    serve(service, io.StringIO(incoming), outgoing)
    return [json.loads(line) for line in outgoing.getvalue().splitlines()]


class ConventionalMcpTests(unittest.TestCase):
    def test_negotiates_each_supported_version_and_falls_back_for_unknown(self) -> None:
        for version in (*SUPPORTED_PROTOCOL_VERSIONS, "future-version"):
            with self.subTest(version=version):
                responses = run_messages(FakeService(), initialized_sequence(version))
                expected = version if version in SUPPORTED_PROTOCOL_VERSIONS else SUPPORTED_PROTOCOL_VERSIONS[0]
                self.assertEqual(responses[0]["result"]["protocolVersion"], expected)
                self.assertEqual(responses[0]["result"]["capabilities"], {"tools": {}})
                self.assertEqual(len(responses), 1, "initialized notification must not get a reply")

    def test_pre_init_ping_is_allowed_but_tools_wait_for_initialized(self) -> None:
        service = FakeService()
        messages = [
            request("ping", 1),
            request("tools/list", 2),
            *initialized_sequence(),
            request("tools/list", 3),
        ]
        responses = run_messages(service, messages)
        self.assertEqual(responses[0], {"jsonrpc": "2.0", "id": 1, "result": {}})
        self.assertEqual(responses[1]["error"]["code"], -32002)
        self.assertEqual([tool["name"] for tool in responses[3]["result"]["tools"]], [
            "conventional_start", "conventional_resume", "conventional_status", "conventional_cancel"
        ])
        self.assertTrue(responses[3]["result"]["tools"][2]["annotations"]["readOnlyHint"])
        self.assertFalse(responses[3]["result"]["tools"][3]["annotations"]["destructiveHint"])
        self.assertTrue(responses[3]["result"]["tools"][3]["annotations"]["idempotentHint"])

    def test_four_tools_delegate_exact_lifecycle_arguments_and_return_structured_json(self) -> None:
        service = FakeService()
        messages = initialized_sequence() + [
            request("tools/call", 10, {"name": "conventional_start", "arguments": {"prompt": "Implement this backlog"}}),
            request("tools/call", 11, {"name": "conventional_resume", "arguments": {"run_id": "run-1", "prompt": "Continue"}}),
            request("tools/call", 12, {"name": "conventional_status", "arguments": {"run_id": "run-1"}}),
            request("tools/call", 13, {"name": "conventional_cancel", "arguments": {"run_id": "run-1"}}),
        ]
        responses = run_messages(service, messages)
        self.assertEqual(service.calls, [
            ("start", "Implement this backlog"),
            ("resume", "run-1", "Continue"),
            ("status", "run-1"),
            ("cancel", "run-1"),
        ])
        for response in responses[1:]:
            result = response["result"]
            self.assertEqual(result["structuredContent"], {"run_id": "run-1", "state": "accepted"})
            self.assertEqual(json.loads(result["content"][0]["text"]), result["structuredContent"])
            self.assertFalse(result["isError"])

    def test_invalid_tool_arguments_are_protocol_errors_and_do_not_call_service(self) -> None:
        service = FakeService()
        messages = initialized_sequence() + [
            request("tools/call", "missing", {"name": "conventional_start", "arguments": {}}),
            request("tools/call", "extra", {"name": "conventional_status", "arguments": {"run_id": "r", "other": 1}}),
            request("tools/call", "badtype", {"name": "conventional_start", "arguments": {"prompt": 5}}),
        ]
        responses = run_messages(service, messages)
        self.assertEqual([response["error"]["code"] for response in responses[1:]], [-32602, -32602, -32602])
        self.assertEqual(service.calls, [])

    def test_call_metadata_is_accepted_but_never_forwarded_to_lifecycle_service(self) -> None:
        service = FakeService()
        messages = initialized_sequence() + [
            request(
                "tools/call",
                20,
                {
                    "name": "conventional_start",
                    "arguments": {"prompt": "Implement this backlog"},
                    "_meta": {"progressToken": "client-token"},
                },
            )
        ]
        responses = run_messages(service, messages)
        self.assertFalse(responses[1]["result"]["isError"])
        self.assertEqual(service.calls, [("start", "Implement this backlog")])
        tools = run_messages(FakeService(), initialized_sequence() + [request("tools/list", 21)])[1]["result"]["tools"]
        start, resume = tools[:2]
        self.assertTrue(start["annotations"]["openWorldHint"])
        self.assertTrue(resume["annotations"]["openWorldHint"])
        self.assertNotIn("destructiveHint", start["annotations"])
        self.assertNotIn("destructiveHint", resume["annotations"])

    def test_service_failure_is_a_tool_error_and_protocol_loop_continues(self) -> None:
        service = FakeService()
        service.fail = True
        messages = initialized_sequence() + [
            request("tools/call", 1, {"name": "conventional_start", "arguments": {"prompt": "Work"}}),
            request("ping", 2),
        ]
        responses = run_messages(service, messages)
        self.assertTrue(responses[1]["result"]["isError"])
        self.assertEqual(responses[1]["result"]["content"][0]["text"], "run handle is unavailable")
        self.assertEqual(responses[2]["result"], {})

    def test_request_cancellation_notification_does_not_cancel_lifecycle_run(self) -> None:
        service = FakeService()
        messages = initialized_sequence() + [
            {"jsonrpc": "2.0", "method": "notifications/cancelled", "params": {"requestId": 44}},
            request("tools/call", 2, {"name": "conventional_status", "arguments": {"run_id": "run-1"}}),
        ]
        responses = run_messages(service, messages)
        self.assertEqual(len(responses), 2)
        self.assertEqual(service.calls, [("status", "run-1")])

    def test_strict_request_id_shapes_and_unknown_methods(self) -> None:
        service = FakeService()
        messages = [
            {"jsonrpc": "2.0", "id": True, "method": "ping"},
            {"jsonrpc": "2.0", "id": None, "method": "ping"},
            {"jsonrpc": "2.0", "id": {}, "method": "ping"},
            request("not/a/method", "unknown"),
            {"jsonrpc": "2.0", "id": "extra", "method": "ping", "ignored": True},
        ]
        responses = run_messages(service, messages)
        self.assertEqual(responses[0]["error"]["code"], -32600)
        self.assertIsNone(responses[0]["id"])
        self.assertEqual(responses[1]["error"]["code"], -32600)
        self.assertEqual(responses[2]["error"]["code"], -32600)
        self.assertEqual(responses[3]["error"]["code"], -32002)
        self.assertEqual(responses[4]["error"]["code"], -32600)

    def test_malformed_json_gets_parse_error_and_server_continues(self) -> None:
        service = FakeService()
        incoming = "{not json}\n" + json.dumps(request("ping", 7)) + "\n"
        output = io.StringIO()
        serve(service, io.StringIO(incoming), output)
        responses = [json.loads(line) for line in output.getvalue().splitlines()]
        self.assertEqual(responses[0]["error"]["code"], -32700)
        self.assertEqual(responses[1]["result"], {})


if __name__ == "__main__":
    unittest.main()
