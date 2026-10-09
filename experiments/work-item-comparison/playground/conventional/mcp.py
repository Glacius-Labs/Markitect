"""MCP stdio facade for the Conventional Playground lifecycle service.

This module owns only the MCP wire protocol. The injected service owns run
creation, continuation, status, and cancellation. In particular, a successful
``start`` response means a handle was accepted; it does not mean the coding
work completed or passed assessment.
"""

from __future__ import annotations

import json
import math
import sys
from typing import Any, TextIO


SUPPORTED_PROTOCOL_VERSIONS = ("2025-11-25", "2025-06-18", "2024-11-05")
SERVER_INFO = {"name": "markitect-conventional-playground", "version": "0.1.0"}

_TOOLS: tuple[dict[str, Any], ...] = (
    {
        "name": "conventional_start",
        "description": (
            "Start one Conventional Playground run with the supplied work request. "
            "Returns a run handle when accepted; inspect status for later progress. "
            "A returned handle does not mean the work is complete or assessed."
        ),
        "inputSchema": {
            "type": "object",
            "properties": {"prompt": {"type": "string", "minLength": 1}},
            "required": ["prompt"],
            "additionalProperties": False,
        },
        "annotations": {
            "readOnlyHint": False,
            "idempotentHint": False,
            "openWorldHint": True,
        },
    },
    {
        "name": "conventional_resume",
        "description": (
            "Continue a Conventional Playground run identified by run_id with a "
            "new work request. Returns the continuation result or current run handle."
        ),
        "inputSchema": {
            "type": "object",
            "properties": {
                "run_id": {"type": "string", "minLength": 1},
                "prompt": {"type": "string", "minLength": 1},
            },
            "required": ["run_id", "prompt"],
            "additionalProperties": False,
        },
        "annotations": {
            "readOnlyHint": False,
            "idempotentHint": False,
            "openWorldHint": True,
        },
    },
    {
        "name": "conventional_status",
        "description": "Read the current lifecycle status for a Conventional run handle.",
        "inputSchema": {
            "type": "object",
            "properties": {"run_id": {"type": "string", "minLength": 1}},
            "required": ["run_id"],
            "additionalProperties": False,
        },
        "annotations": {
            "readOnlyHint": True,
            "destructiveHint": False,
            "idempotentHint": True,
            "openWorldHint": False,
        },
    },
    {
        "name": "conventional_cancel",
        "description": "Request cancellation of a Conventional run identified by run_id.",
        "inputSchema": {
            "type": "object",
            "properties": {"run_id": {"type": "string", "minLength": 1}},
            "required": ["run_id"],
            "additionalProperties": False,
        },
        "annotations": {
            "readOnlyHint": False,
            "destructiveHint": False,
            "idempotentHint": True,
            "openWorldHint": False,
        },
    },
)

_TOOL_BY_NAME = {tool["name"]: tool for tool in _TOOLS}


class _RpcFault(Exception):
    def __init__(self, code: int, message: str, data: Any = None) -> None:
        super().__init__(message)
        self.code = code
        self.message = message
        self.data = data


def _valid_id(value: Any) -> bool:
    if isinstance(value, bool) or value is None:
        return False
    if isinstance(value, str):
        return True
    if isinstance(value, int):
        return True
    return isinstance(value, float) and math.isfinite(value)


def _error(request_id: Any, code: int, message: str, data: Any = None) -> dict[str, Any]:
    error: dict[str, Any] = {"code": code, "message": message}
    if data is not None:
        error["data"] = data
    return {"jsonrpc": "2.0", "id": request_id, "error": error}


def _response(request_id: Any, result: Any) -> dict[str, Any]:
    return {"jsonrpc": "2.0", "id": request_id, "result": result}


def _validate_envelope(message: Any) -> tuple[str, Any, dict[str, Any], bool]:
    if not isinstance(message, dict):
        raise _RpcFault(-32600, "Invalid Request")
    if set(message) - {"jsonrpc", "method", "id", "params"}:
        raise _RpcFault(-32600, "Invalid Request")
    if message.get("jsonrpc") != "2.0" or not isinstance(message.get("method"), str):
        raise _RpcFault(-32600, "Invalid Request")
    is_notification = "id" not in message
    request_id = message.get("id")
    if not is_notification and not _valid_id(request_id):
        raise _RpcFault(-32600, "Invalid Request")
    params = message.get("params", {})
    if not isinstance(params, dict):
        raise _RpcFault(-32600, "Invalid Request", "params must be an object")
    return message["method"], request_id, params, is_notification


def _require_exact_keys(params: dict[str, Any], expected: set[str]) -> None:
    if set(params) != expected:
        raise _RpcFault(-32602, "Invalid params")


def _require_text(params: dict[str, Any], name: str) -> str:
    value = params.get(name)
    if not isinstance(value, str) or not value.strip():
        raise _RpcFault(-32602, f"Invalid params: {name} must be a non-empty string")
    return value


def _tool_result(value: Any) -> dict[str, Any]:
    if not isinstance(value, dict):
        raise TypeError("Lifecycle service must return a JSON object")
    # Round-trip once to reject values JSON cannot represent and to detach any
    # custom mappings or mutable service-owned objects from the protocol layer.
    structured = json.loads(json.dumps(value, ensure_ascii=False, allow_nan=False))
    return {
        "content": [{"type": "text", "text": json.dumps(structured, ensure_ascii=False)}],
        "structuredContent": structured,
        "isError": False,
    }


def _tool_error(exc: Exception) -> dict[str, Any]:
    message = str(exc).strip() or exc.__class__.__name__
    return {
        "content": [{"type": "text", "text": message}],
        "isError": True,
    }


def _call_tool(service: Any, params: dict[str, Any]) -> dict[str, Any]:
    name = params.get("name")
    if not isinstance(name, str) or name not in _TOOL_BY_NAME:
        raise _RpcFault(-32602, "Unknown tool" if isinstance(name, str) else "Invalid params")
    if set(params) - {"name", "arguments", "_meta"} or not {"name", "arguments"}.issubset(params):
        raise _RpcFault(-32602, "Invalid params")
    if "_meta" in params and not isinstance(params["_meta"], dict):
        raise _RpcFault(-32602, "Invalid params: _meta must be an object")
    arguments = params["arguments"]
    if not isinstance(arguments, dict):
        raise _RpcFault(-32602, "Invalid params: arguments must be an object")

    if name == "conventional_start":
        _require_exact_keys(arguments, {"prompt"})
        prompt = _require_text(arguments, "prompt")
        method = lambda: service.start(prompt)
    elif name == "conventional_resume":
        _require_exact_keys(arguments, {"run_id", "prompt"})
        run_id = _require_text(arguments, "run_id")
        prompt = _require_text(arguments, "prompt")
        method = lambda: service.resume(run_id, prompt)
    elif name == "conventional_status":
        _require_exact_keys(arguments, {"run_id"})
        run_id = _require_text(arguments, "run_id")
        method = lambda: service.status(run_id)
    else:
        _require_exact_keys(arguments, {"run_id"})
        run_id = _require_text(arguments, "run_id")
        method = lambda: service.cancel(run_id)

    try:
        return _tool_result(method())
    except Exception as exc:  # lifecycle failures are actionable tool results
        return _tool_error(exc)


class _Protocol:
    def __init__(self, service: Any) -> None:
        self.service = service
        self.initialized = False
        self.negotiated_version: str | None = None

    def handle(self, message: Any) -> dict[str, Any] | None:
        try:
            method, request_id, params, notification = _validate_envelope(message)
        except _RpcFault as fault:
            # Invalid objects without a usable id still receive id:null per
            # JSON-RPC. Invalid notification-shaped envelopes are malformed
            # requests rather than valid notifications.
            return _error(None, fault.code, fault.message, fault.data)

        if notification:
            if method == "notifications/initialized" and self.negotiated_version:
                self.initialized = True
            # Cancellation of a JSON-RPC request is not a run cancellation.
            # A run has an explicit conventional_cancel tool and durable handle.
            return None

        assert request_id is not None
        try:
            if method == "ping":
                return _response(request_id, {})
            if method == "initialize":
                return self._initialize(request_id, params)
            if not self.negotiated_version or not self.initialized:
                raise _RpcFault(-32002, "Server not initialized")
            if method == "tools/list":
                if set(params) - {"cursor"}:
                    raise _RpcFault(-32602, "Invalid params")
                if "cursor" in params:
                    raise _RpcFault(-32602, "Pagination is not supported")
                tools = list(_TOOLS)
                if self.negotiated_version == "2024-11-05":
                    tools = [{key: value for key, value in tool.items() if key != "annotations"} for tool in tools]
                return _response(request_id, {"tools": tools})
            if method == "tools/call":
                result = _call_tool(self.service, params)
                if self.negotiated_version == "2024-11-05":
                    result.pop("structuredContent", None)
                return _response(request_id, result)
            raise _RpcFault(-32601, "Method not found")
        except _RpcFault as fault:
            return _error(request_id, fault.code, fault.message, fault.data)
        except Exception as exc:
            # Never put service exceptions on stdout or crash the stdio loop.
            return _error(request_id, -32603, "Internal error", str(exc))

    def _initialize(self, request_id: Any, params: dict[str, Any]) -> dict[str, Any]:
        if self.negotiated_version is not None:
            raise _RpcFault(-32600, "Already initialized")
        if set(params) - {"protocolVersion", "capabilities", "clientInfo"}:
            raise _RpcFault(-32602, "Invalid initialize params")
        version = params.get("protocolVersion")
        capabilities = params.get("capabilities")
        client_info = params.get("clientInfo")
        if (
            not isinstance(version, str)
            or not isinstance(capabilities, dict)
            or not isinstance(client_info, dict)
            or not isinstance(client_info.get("name"), str)
            or not isinstance(client_info.get("version"), str)
        ):
            raise _RpcFault(-32602, "Invalid initialize params")
        selected = version if version in SUPPORTED_PROTOCOL_VERSIONS else SUPPORTED_PROTOCOL_VERSIONS[0]
        self.negotiated_version = selected
        return _response(
            request_id,
            {
                "protocolVersion": selected,
                "capabilities": {"tools": {}},
                "serverInfo": dict(SERVER_INFO),
            },
        )


def serve(
    service: Any,
    input_stream: TextIO = sys.stdin,
    output_stream: TextIO = sys.stdout,
) -> None:
    """Serve newline-delimited MCP JSON-RPC on stdio, writing protocol only to stdout."""
    protocol = _Protocol(service)
    for line in input_stream:
        if not line.strip():
            continue
        try:
            message = json.loads(line)
        except (json.JSONDecodeError, UnicodeDecodeError):
            response = _error(None, -32700, "Parse error")
        else:
            response = protocol.handle(message)
        if response is not None:
            output_stream.write(json.dumps(response, ensure_ascii=False, allow_nan=False) + "\n")
            output_stream.flush()
