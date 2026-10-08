"""Offline-only wiring: classify beside the unchanged rejecting ThreadGate."""
import hashlib
import importlib.util
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parent
REPO = ROOT.parents[3]
ARCHIVE = REPO / 'experiments/government-comparison/evidence/s1-runner-rebinding-metadata-20261008-r1/generated-schemas.zip'
PROTOCOL = REPO / 'experiments/government-comparison/evidence/s1-common-runner-read-tool-20261008-r1/protocol.py'
PROTOCOL_SHA = 'ebd087657febf63846904d83a75bc226a5412458418d20abe024ed267febb822'
GATE_SHA = 'd62ae5d946af927d09538628905cf05a242ede47267e693b571736b79c50c7de'
FAILURE_SHA = '776365db287479a23be67a4e22ac083f54ce881fa1452337d0e281529adcb38f'


def require(ok, reason):
    if not ok:
        raise ValueError(reason)


def load(path, expected, name):
    require(hashlib.sha256(path.read_bytes()).hexdigest() == expected, 'source-pin-mismatch')
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def load_fixture():
    import startup_observer
    contract = json.loads((ROOT / 'observer-schema-contract.json').read_text(encoding='utf-8'))
    admission = json.loads((ROOT / 'admission-contract.json').read_text(encoding='utf-8'))
    require(startup_observer.METHOD not in admission['notifications'], 'offline-admission-changed')
    protocol = load(PROTOCOL, PROTOCOL_SHA, 'offline_startup_protocol').Protocol(ARCHIVE, contract)
    gate = load(ROOT / 'thread_gate.py', GATE_SHA, 'offline_startup_gate')
    failure = load(ROOT / 'protocol_failure.py', FAILURE_SHA, 'offline_startup_failure')
    profile = json.loads((ROOT / 'profile.json').read_text(encoding='utf-8'))
    return protocol, gate, failure, profile, admission


class OfflineProbe:
    def __init__(self, profile=None):
        import startup_observer
        protocol, gate, failure, default_profile, admission = load_fixture()
        self.gate = gate.ThreadGate(default_profile if profile is None else profile, protocol, admission)
        self.observer = startup_observer.StartupObserver(protocol)
        self._failure_module = failure
        self._failure = failure.FirstProtocolFailure()
        self._notifications = 0
        self._drain_reason = None

    def accept_response(self, response):
        try:
            self.gate.accept_response(response)
        except BaseException as exc:
            self._failure.record({'result': None}, 'thread', 'thread-response-gate', exc)
            raise

    def notification(self, frame, phase='thread'):
        import startup_observer
        stage = 'notification-envelope'
        try:
            self._notifications += 1
            require(self._notifications <= 2048, 'notification-limit')
            require(type(frame) is dict and set(frame) <= {'method', 'params', 'emittedAtMs'}
                    and type(frame.get('method')) is str and 'params' in frame,
                    'server-request-or-notification-envelope')
            stage = 'notification-timestamp'
            if 'emittedAtMs' in frame:
                require(type(frame['emittedAtMs']) is int and -(2**63) <= frame['emittedAtMs'] < 2**63,
                        'notification-timestamp')
            if frame['method'] == startup_observer.METHOD:
                self.observer.observe(frame['params'], self.gate.thread_id)
            stage = 'thread-lifecycle'
            require(phase in ('thread', 'cleanup'), 'lifecycle-before-thread-request')
            stage = 'notification-gate'
            # Classification NEVER bypasses or changes actual admission.
            self.gate.accept_notification(frame['method'], frame['params'])
        except BaseException as exc:
            self._failure.record(frame, phase, stage, exc)
            raise

    def drain(self, frames):
        require(type(frames) is list and len(frames) <= 64, 'notification-limit')
        for frame in frames:
            try:
                self.notification(frame, 'cleanup')
            except BaseException:
                self._drain_reason = self._drain_reason or 'unexpected-final-output'

    def finish(self):
        first = self._failure.snapshot()
        return {'mode': 'offline-only', 'status': 'stopped' if first else 'no-runtime-executed',
            'stopReason': self._drain_reason or (first['refusalCode'] if first else None),
            'startupObservation': self.observer.snapshot(), 'firstProtocolFailure': first,
            'threadResponseValidated': self.gate.validated,
            'turnsRequested': 0, 'toolsRequested': 0}


if __name__ == '__main__':
    raise SystemExit(2)
