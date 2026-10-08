"""Bounded first-rejection metadata, version 1. No I/O or caller-supplied catalog."""
import copy
import json

VERSION = 1
UNDECODED = object()
PUBLIC_METHODS = frozenset(('account/chatgptAuthTokens/refresh', 'account/gatewayOAuth/changed', 'account/login/completed', 'account/rateLimits/updated', 'account/updated', 'app/list/updated', 'applyPatchApproval', 'attestation/generate', 'autoApprovalReview/strictReviewRequired', 'command/exec/outputDelta', 'configWarning', 'currentTime/read', 'deprecationNotice', 'error', 'execCommandApproval', 'externalAgentConfig/import/completed', 'externalAgentConfig/import/progress', 'fs/changed', 'fuzzyFileSearch/sessionCompleted', 'fuzzyFileSearch/sessionUpdated', 'guardianWarning', 'hook/completed', 'hook/started', 'item/agentMessage/delta', 'item/autoApprovalReview/completed', 'item/autoApprovalReview/started', 'item/commandExecution/outputDelta', 'item/commandExecution/requestApproval', 'item/commandExecution/terminalInteraction', 'item/completed', 'item/fileChange/outputDelta', 'item/fileChange/patchUpdated', 'item/fileChange/requestApproval', 'item/mcpToolCall/progress', 'item/permissions/requestApproval', 'item/plan/delta', 'item/reasoning/summaryPartAdded', 'item/reasoning/summaryTextDelta', 'item/reasoning/textDelta', 'item/started', 'item/tool/call', 'item/tool/requestUserInput', 'mcpServer/elicitation/request', 'mcpServer/event/stream/notification', 'mcpServer/oauthLogin/completed', 'mcpServer/startupStatus/updated', 'model/rerouted', 'model/safetyBuffering/updated', 'model/verification', 'modelProvider/authRecoveryCompleted', 'modelProvider/authRecoveryStarted', 'process/exited', 'process/outputDelta', 'project/changed', 'remoteControl/status/changed', 'serverRequest/resolved', 'skills/changed', 'thread/archived', 'thread/attachment/updated', 'thread/closed', 'thread/compacted', 'thread/deleted', 'thread/environment/connected', 'thread/environment/disconnected', 'thread/goal/cleared', 'thread/goal/updated', 'thread/name/updated', 'thread/project/updated', 'thread/queue/changed', 'thread/realtime/closed', 'thread/realtime/error', 'thread/realtime/item/completed', 'thread/realtime/item/started', 'thread/realtime/item/transcript/delta', 'thread/realtime/itemAdded', 'thread/realtime/outputAudio/delta', 'thread/realtime/sdp', 'thread/realtime/started', 'thread/realtime/transcript/delta', 'thread/realtime/transcript/done', 'thread/reverted', 'thread/settings/updated', 'thread/started', 'thread/status/changed', 'thread/tokenUsage/updated', 'thread/unarchived', 'turn/completed', 'turn/diff/updated', 'turn/moderationMetadata', 'turn/plan/updated', 'turn/started', 'warning', 'windows/worldWritableWarning', 'windowsSandbox/setupCompleted'))
PHASES = frozenset(('metadata', 'thread', 'cleanup'))
STAGES = frozenset(('frame-bounds', 'json-decode', 'notification-envelope',
    'notification-timestamp', 'remote-status', 'thread-lifecycle',
    'notification-provisional', 'notification-gate', 'response-envelope',
    'response-provisional', 'response-schema', 'thread-response-gate'))
REFUSAL_CODES_V1 = frozenset((
    'stdout-closed-or-frame-limit', 'notification-limit',
    'server-request-or-notification-envelope', 'notification-timestamp',
    'remote-control-not-disabled', 'lifecycle-before-thread-request',
    'unsolicited-response', 'warning-or-provisional-result',
    'forbidden-or-unknown-notification', 'thread-status-not-allowed',
    'early-thread-closure-or-pending-limit', 'thread-id-invalid',
    'thread-identity-mismatch', 'thread-model-or-parent-mismatch',
    'unexpected-existing-turn', 'thread-notification-id-mismatch',
    'duplicate-or-late-thread-started', 'duplicate-thread-closed',
    'unexpected-active-status-without-turn', 'duplicate-json-key',
    'invalid-json-number', 'protocol-schema-invalid', 'schema-resource-limit',
    'malformed-response-envelope', 'unexpected-response-id-or-shape',
    'rpc-error', 'unexpected-result-shape',
    'thread-response-identity-or-policy-mismatch'))
EXCEPTION_CLASSES = {ValueError: 'ValueError', json.JSONDecodeError: 'JSONDecodeError',
                     TypeError: 'TypeError', KeyError: 'KeyError'}


def envelope(frame):
    if frame is UNDECODED:
        return 'undecoded'
    if type(frame) is not dict:
        return 'nonobject'
    if ('method' in frame and ('result' in frame or 'error' in frame)) or ('result' in frame and 'error' in frame):
        return 'mixed'
    if 'method' in frame:
        return 'server-request' if 'id' in frame else 'notification'
    if 'result' in frame:
        return 'result-response'
    if 'error' in frame:
        return 'error-response'
    return 'unknown-object'


class FirstProtocolFailure:
    def __init__(self):
        self._first = None

    def record(self, frame, phase, stage, exc):
        if self._first is not None:
            return
        method = {'category': 'absent', 'name': None}
        if type(frame) is dict and 'method' in frame:
            value = frame['method']
            if type(value) is not str:
                method['category'] = 'nonstring'
            elif value in PUBLIC_METHODS:
                method = {'category': 'listed', 'name': value}
            else:
                method['category'] = 'unlisted'
        code = 'unknown'
        if type(exc) is ValueError and len(exc.args) == 1 and type(exc.args[0]) is str and exc.args[0] in REFUSAL_CODES_V1:
            code = exc.args[0]
        self._first = {
            'version': VERSION,
            'phase': phase if type(phase) is str and phase in PHASES else 'unknown',
            'stage': stage if type(stage) is str and stage in STAGES else 'unknown',
            'envelopeCategory': envelope(frame), 'method': method,
            'exceptionClass': EXCEPTION_CLASSES.get(type(exc), 'unknown'),
            'refusalCode': code,
        }

    def snapshot(self):
        return copy.deepcopy(self._first)
