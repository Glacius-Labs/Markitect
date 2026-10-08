from __future__ import annotations
import copy
import importlib.util
import json
from pathlib import Path
import unittest

HERE = Path(__file__).resolve().parent

def load(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module

client = load('r2_consumer_client_under_test', HERE / 'client.py')
profile = json.loads((HERE / 'profile.json').read_text(encoding='utf-8'))
contract = json.loads((HERE / 'schema-contract.json').read_text(encoding='utf-8'))
protocol_module = client.load_module(client.PROTOCOL, client.PROTOCOL_SHA, 'r2_consumer_protocol')
protocol = protocol_module.Protocol(client.SCHEMA, contract)
gate_module = load('r2_consumer_thread_gate', HERE / 'thread_gate.py')


def thread(thread_id='thread-r2-test', turns=None, **changes):
    value = {
        'cliVersion': 'offline-test', 'createdAt': 1, 'cwd': profile['cwd'],
        'ephemeral': True, 'id': thread_id, 'model': 'gpt-6.1-sol',
        'modelProvider': 'openai', 'preview': '', 'projectId': None,
        'sessionId': 'session-r2-test', 'source': 'appServer',
        'status': {'type': 'active', 'activeFlags': []},
        'turns': [] if turns is None else turns, 'updatedAt': 1,
        'parentThreadId': None, 'forkedFromId': None,
    }
    value.update(changes)
    return value


def response(**changes):
    value = {
        'approvalPolicy': 'never', 'approvalsReviewer': 'user',
        'cwd': profile['cwd'], 'model': 'gpt-6.1-sol', 'modelProvider': 'openai',
        'sandbox': {'type': 'readOnly', 'networkAccess': False},
        'activePermissionProfile': {'id': ':read-only', 'extends': None},
        'thread': thread(),
    }
    value.update(changes)
    return value


def new_gate():
    return gate_module.ThreadGate(profile, protocol, contract)


class NewSchemaConsumerCases(unittest.TestCase):
    def test_new_schema_valid_identity_and_readonly_thread_response_passes_before_turn(self):
        gate = new_gate()
        value = response()
        protocol.validate(contract['responseSchemas']['thread/start'], value)
        gate.accept_response(value)
        observed = gate.finish()
        self.assertTrue(observed['responseValidated'])
        self.assertEqual(observed['threadId'], 'thread-r2-test')
        self.assertEqual(observed['turnsRequested'], 0)
        self.assertEqual(observed['toolsRequested'], 0)

    def test_each_identity_sandbox_and_thread_state_violation_stays_negative(self):
        turns = [{'id': 'existing', 'status': 'completed', 'items': []}]
        cases = {
            'model': response(model='gpt-6-astra'),
            'provider': response(modelProvider='other-provider'),
            'cwd': response(cwd='C:/other'),
            'approval': response(approvalPolicy='on-request'),
            'network': response(sandbox={'type': 'readOnly', 'networkAccess': True}),
            'write-sandbox': response(sandbox={'type': 'workspaceWrite', 'networkAccess': False}),
            'ephemeral': response(thread=thread(ephemeral=False)),
            'parent': response(thread=thread(parentThreadId='parent-thread')),
            'fork': response(thread=thread(forkedFromId='source-thread')),
            'thread-model': response(thread=thread(model='gpt-6-astra')),
            'existing-turn': response(thread=thread(turns=turns)),
        }
        for label, value in cases.items():
            with self.subTest(case=label):
                gate = new_gate()
                with self.assertRaises(ValueError):
                    gate.accept_response(value)
                self.assertFalse(gate.finish()['responseValidated'])

    def test_thread_started_notification_requires_response_id_correlation(self):
        gate = new_gate()
        gate.accept_response(response())
        gate.accept_notification('thread/started', {'thread': thread()})
        self.assertTrue(gate.finish()['threadStartedNotificationCorrelated'])
        mismatch = new_gate()
        mismatch.accept_response(response())
        with self.assertRaisesRegex(ValueError, 'thread-notification-id-mismatch'):
            mismatch.accept_notification('thread/started', {'thread': thread(thread_id='other-thread')})
        self.assertTrue(mismatch.finish()['responseValidated'])
        self.assertFalse(mismatch.finish()['threadStartedNotificationCorrelated'])

    def test_new_prediction_notification_is_schema_valid_but_terminal_to_narrow_gate(self):
        frame = {
            'method': 'thread/prediction/updated',
            'params': {'threadId': 'thread-r2-test', 'sourceTurnId': 'turn-1',
                       'result': {'type': 'completed', 'text': 'prediction'}},
        }
        protocol.validate('ServerNotification.json', frame)
        gate = new_gate()
        gate.accept_response(response())
        with self.assertRaisesRegex(ValueError, 'forbidden-or-unknown-notification'):
            gate.accept_notification(frame['method'], frame['params'])

    def test_broadened_error_info_shape_is_valid_but_error_notification_stays_terminal(self):
        frame = {
            'method': 'error',
            'params': {
                'error': {'message': 'offline fixture',
                          'codexErrorInfo': {'responseStreamDisconnected': {'httpStatusCode': 503}}},
                'threadId': 'thread-r2-test', 'turnId': 'turn-1', 'willRetry': False,
            },
        }
        protocol.validate('ServerNotification.json', frame)
        gate = new_gate()
        with self.assertRaisesRegex(ValueError, 'forbidden-or-unknown-notification'):
            gate.accept_notification(frame['method'], frame['params'])
        self.assertFalse(gate.finish()['responseValidated'])

    def test_rpc_error_envelope_never_becomes_a_thread_response(self):
        prior = client.load_module(client.PRIOR, client.PRIOR_SHA, 'r2_consumer_response_guard')
        rpc_error = {'jsonrpc': '2.0', 'id': 3,
                     'error': {'code': -32000, 'message': 'offline fixture',
                               'data': {'codexErrorInfo': {'responseStreamDisconnected': {'httpStatusCode': 503}}}}}
        with self.assertRaisesRegex(ValueError, 'rpc-error'):
            prior.validate_response_envelope(rpc_error, 3)
        gate = new_gate()
        self.assertFalse(gate.finish()['responseValidated'])

    def test_client_admission_rejects_new_methods_and_all_calls_after_thread_terminal(self):
        sent = ['initialize', 'initialized', 'config/read', 'configRequirements/read']
        for method in ('thread/prediction/updated', 'thread/updated', 'prediction/updated', 'turn/start'):
            with self.subTest(method=method):
                with self.assertRaisesRegex(ValueError, 'forbidden-or-out-of-order-client-method'):
                    client.admit_rpc(method, sent, False, False)
        sent.append('thread/start')
        with self.assertRaisesRegex(ValueError, 'rpc-after-thread-terminal'):
            client.admit_rpc('turn/start', sent, False, True)


if __name__ == '__main__':
    unittest.main(verbosity=2)
