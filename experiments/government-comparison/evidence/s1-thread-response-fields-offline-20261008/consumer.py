"""Offline-only one-response consumer; no process, transport, file or logging API."""

from thread_gate import ThreadGate, require


class ThreadResponseConsumer:
    def __init__(self, profile, protocol, contract):
        self._gate = ThreadGate(profile, protocol, contract)
        self._closed = False

    def accept_response(self, result):
        require(not self._closed, "response-consumer-terminal")
        self._closed = True
        # Existing schema/identity/sandbox/parent/turn checks raise unchanged.
        # The gate retains metadata before its unchanged equality stop.
        self._gate.accept_response(result)

    def diagnostic(self):
        return self._gate.finish()
