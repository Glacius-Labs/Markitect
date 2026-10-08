"""Hypothetical typed Startupstatus classification. No admission or I/O APIs."""
import copy
import json
import re

MEMBER = 'v2/McpServerStatusUpdatedNotification.json'
METHOD = 'mcpServer/startupStatus/updated'
REFUSAL_VERSION = 1
REFUSALS_V1 = frozenset(('schema-invalid', 'validator-failure', 'local-shape',
    'local-size', 'missing-thread-id', 'unresolved-thread', 'foreign-thread',
    'invalid-scope-context', 'terminal-status', 'error-present',
    'failure-reason-present', 'observation-limit'))
KEYS = frozenset(('name', 'status', 'error', 'failureReason', 'threadId'))
STATUSES = ('starting', 'ready', 'failed', 'cancelled', 'unknown')
SCOPES = ('app', 'own-thread', 'foreign-thread', 'missing', 'unresolved-thread', 'invalid')
MAX_OBSERVATIONS = 2048
MAX_PAYLOAD_BYTES = 8192


def valid_id(value):
    return type(value) is str and re.fullmatch(r'[A-Za-z0-9:_-]{1,128}', value) is not None


def row(status='unknown', scope='invalid', reason='schema-invalid'):
    return {'refusalVersion': REFUSAL_VERSION,
            'hypotheticalDisposition': 'observation-only' if reason is None else 'terminal-reject',
            'status': status, 'scope': scope, 'refusalCode': reason}


class StartupObserver:
    def __init__(self, protocol):
        self._protocol = protocol
        self._observed = 0
        self._limit = False
        self._statuses = dict.fromkeys(STATUSES, 0)
        self._scopes = dict.fromkeys(SCOPES, 0)
        self._dispositions = dict.fromkeys(('observation-only', 'terminal-reject'), 0)
        self._first = None

    def _classify(self, params, own_thread_id):
        try:
            self._protocol.validate(MEMBER, params)
        except ValueError:
            return row()
        except Exception:
            return row(reason='validator-failure')
        # Schema permits extras; this is an expressly additional local rule.
        if type(params) is not dict or not set(params) <= KEYS:
            return row(reason='local-shape')
        caps = {'name': (1, 256), 'status': (1, 16), 'error': (0, 1024),
                'failureReason': (0, 32), 'threadId': (1, 128)}
        for key, (minimum, maximum) in caps.items():
            value = params.get(key)
            if value is not None and (type(value) is not str or not minimum <= len(value) <= maximum):
                return row(reason='local-size')
        # Bounded primitives only; transient encoding is never retained/output.
        if len(json.dumps(params, ensure_ascii=True, separators=(',', ':'), allow_nan=False)) > MAX_PAYLOAD_BYTES:
            return row(reason='local-size')
        status = params['status']  # Real schema has already checked the enum.
        if 'threadId' not in params:
            return row(status, 'missing', 'missing-thread-id')
        identifier = params['threadId']
        if identifier is None:
            scope = 'app'
        elif not valid_id(identifier):
            return row(status, 'invalid', 'local-size')
        elif own_thread_id is None:
            return row(status, 'unresolved-thread', 'unresolved-thread')
        elif not valid_id(own_thread_id):
            return row(status, 'invalid', 'invalid-scope-context')
        elif identifier != own_thread_id:
            return row(status, 'foreign-thread', 'foreign-thread')
        else:
            scope = 'own-thread'
        if status in ('failed', 'cancelled'):
            return row(status, scope, 'terminal-status')
        if params.get('error') is not None:
            return row(status, scope, 'error-present')
        if params.get('failureReason') is not None:
            return row(status, scope, 'failure-reason-present')
        return row(status, scope, None)

    def observe(self, params, own_thread_id=None):
        if self._observed >= MAX_OBSERVATIONS:
            self._limit = True
            return row(reason='observation-limit')
        classification = self._classify(params, own_thread_id)
        self._observed += 1
        self._statuses[classification['status']] += 1
        self._scopes[classification['scope']] += 1
        self._dispositions[classification['hypotheticalDisposition']] += 1
        if self._first is None:
            self._first = copy.deepcopy(classification)
        return copy.deepcopy(classification)

    def snapshot(self):
        return copy.deepcopy({'version': 1, 'mode': 'hypothetical-only',
            'methodAdmitted': False, 'observed': self._observed, 'limitReached': self._limit,
            'statusCounts': self._statuses, 'scopeCounts': self._scopes,
            'dispositionCounts': self._dispositions, 'firstClassification': self._first})
