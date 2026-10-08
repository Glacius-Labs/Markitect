"""Offline account-notice classification; no authentication or I/O operations."""
import copy
import json

MEMBER = 'v2/AccountUpdatedNotification.json'
METHOD = 'account/updated'
MAX_OBSERVATIONS = 8
MAX_INPUT_BYTES = 4096
MAX_PROJECTION_BYTES = 2048
REFUSALS = ('schema-invalid', 'validator-failure', 'local-size', 'local-shape',
            'mode-not-explicitly-compatible', 'observation-limit', 'terminal-latched')


def row(reason=None):
    return {'version': 1, 'scope': 'connection-account',
            'hypotheticalDisposition': 'observation-only' if reason is None else 'terminal-reject',
            'refusalCode': reason}


class AccountObserver:
    def __init__(self, protocol):
        self._protocol = protocol
        self._observed = 0
        self._limit = False
        self._terminal = False
        self._dispositions = dict.fromkeys(('observation-only', 'terminal-reject'), 0)
        self._reasons = dict.fromkeys(REFUSALS, 0)
        self._first = None

    def _classify(self, params, serialized_input_bytes):
        # The archive schema is authoritative for types and enum membership.
        try:
            self._protocol.validate(MEMBER, params)
        except ValueError:
            return row('schema-invalid')
        except Exception:
            return row('validator-failure')
        # Encoding is transient and never logged, returned or fingerprinted.
        try:
            size = len(json.dumps(params, ensure_ascii=True, separators=(',', ':'),
                                  allow_nan=False).encode('ascii'))
        except (ValueError, TypeError, OverflowError, RecursionError):
            return row('local-size')
        if size > MAX_INPUT_BYTES or (serialized_input_bytes is not None and
                (type(serialized_input_bytes) is not int or
                 not 0 <= serialized_input_bytes <= MAX_INPUT_BYTES)):
            return row('local-size')
        # The schema permits missing/null fields and extras. These are stricter
        # local rules, not alleged schema requirements or account assertions.
        if type(params) is not dict or not set(params) <= {'authMode', 'planType'}:
            return row('local-shape')
        if params.get('authMode') != 'chatgpt':
            return row('mode-not-explicitly-compatible')
        return row()

    def observe(self, params, serialized_input_bytes=None):
        if self._observed >= MAX_OBSERVATIONS:
            self._limit = True
            self._terminal = True
            return row('observation-limit')
        if self._terminal:
            return row('terminal-latched')
        classification = self._classify(params, serialized_input_bytes)
        self._observed += 1
        self._dispositions[classification['hypotheticalDisposition']] += 1
        reason = classification['refusalCode']
        if reason is not None:
            self._reasons[reason] += 1
            self._terminal = True
        if self._first is None:
            self._first = copy.deepcopy(classification)
        return classification

    def snapshot(self):
        projection = {'version': 1, 'scope': 'connection-account',
                      'observed': self._observed, 'limitReached': self._limit,
                      'terminal': self._terminal,
                      'dispositionCounts': self._dispositions,
                      'refusalCounts': self._reasons, 'firstClassification': self._first,
                      'telemetryOnly': True, 'ownThreadOrAccountIdentityEstablished': False}
        result = copy.deepcopy(projection)
        if len(json.dumps(result, ensure_ascii=True).encode('ascii')) > MAX_PROJECTION_BYTES:
            raise ValueError('fixed-projection-limit')
        return result
