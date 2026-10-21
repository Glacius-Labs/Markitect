"""Linux operator controls: real owned processes, no Docker or provider starts.

    python3 -B -m unittest discover -s experiments/apply-trials -p night_run_test.py
"""
import json
import copy
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

import night_run as runner


class SupervisorControls(unittest.TestCase):
    def test_actual_profiles_match_authorized_provider_model_effort_image_and_store(self):
        manifest = {'provider': {'model': 'gpt-6.1-sol', 'effort': 'high'},
                    'images': {'agent': 'agent-image', 'oracle': 'check-image'}}
        packet = {'store': 'owned'}
        profile = {'apiVersion': 'markitect.runtime/v1', 'agentProfiles': {'all-roles': {'codexAppServer': {
            'imageID': 'agent-image', 'engineExecutable': '/usr/bin/docker', 'credentialRef': 'dedicated-codex-runtime',
            'session': {'model': 'gpt-6.1-sol', 'effort': 'high', 'store': 'owned'}}}},
            'checkProfiles': {'tests': {'container': {'imageID': 'check-image', 'engineExecutable': '/usr/bin/docker'}}}}
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / 'profile.json'
            path.write_text(json.dumps(profile))
            runner.validate_profile(path, packet, manifest)
            for key, value in (('model', 'another-model'), ('effort', 'low'), ('store', 'foreign')):
                bad = copy.deepcopy(profile)
                bad['agentProfiles']['all-roles']['codexAppServer']['session'][key] = value
                path.write_text(json.dumps(bad))
                with self.subTest(session=key), self.assertRaises(ValueError):
                    runner.validate_profile(path, packet, manifest)
            for key, value in (('imageID', 'another-image'), ('credentialRef', 'desktop-auth'), ('engineExecutable', '/other/engine')):
                bad = copy.deepcopy(profile)
                bad['agentProfiles']['all-roles']['codexAppServer'][key] = value
                path.write_text(json.dumps(bad))
                with self.subTest(provider=key), self.assertRaises(ValueError):
                    runner.validate_profile(path, packet, manifest)
            for field in ('agentProfiles', 'checkProfiles'):
                bad = copy.deepcopy(profile)
                bad[field]['extra'] = {'claudeCode': {}} if field == 'agentProfiles' else {'container': {'imageID': 'wrong-check-image'}}
                path.write_text(json.dumps(bad))
                with self.subTest(extra=field), self.assertRaises(ValueError):
                    runner.validate_profile(path, packet, manifest)

    def test_reused_identity_never_receives_a_signal(self):
        with tempfile.TemporaryDirectory() as folder:
            registry = Path(folder) / 'groups'
            registry.write_text(json.dumps({'pid': 100, 'group': 100, 'start': 'old'}) + '\n')
            with patch.object(runner, 'process_identity', return_value={'pid': 100, 'group': 100, 'start': 'new'}), \
                    patch.object(runner.os, 'killpg') as kill:
                runner.signal_tracked(registry, signal.SIGKILL)
                kill.assert_not_called()

    def test_marker_write_failure_cannot_prevent_cessation(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            registry = root / 'groups'
            registry.touch()
            child = subprocess.Popen([sys.executable, '-c', 'import time; time.sleep(60)'], start_new_session=True)
            try:
                runner.track_process(registry, child.pid)
                with patch.object(runner, 'recovery', return_value=False), \
                        patch.object(runner.subprocess, 'run', return_value=subprocess.CompletedProcess([], 0, b'', b'')) as docker:
                    status = runner.force_supervised_stop(child, registry, root, {'name': 'test', 'store': 'owned-test'}, 'injected write failure')
                self.assertEqual(status, 4)
                self.assertIsNotNone(child.poll())
                evidence = json.loads((root / 'forced-cleanup-test.json').read_text())
                self.assertFalse(evidence['recoveryMarkerDurable'])
                self.assertFalse(evidence['normalTurnStopsProved'])
                for call in docker.call_args_list:
                    self.assertIn('label=markitect.store=owned-test', call.args[0])
            finally:
                if child.poll() is None:
                    runner.os.killpg(child.pid, signal.SIGKILL)
                child.wait()

    def supervise_failure(self, failure):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            script = root / 'sleep.py'
            script.write_text('import time\ntime.sleep(60)\n')
            manifest = {'nightRoot': folder, 'packets': [{'name': 'test', 'store': 'owned-test'}]}
            clock = {'utc': 1792620000000, 'ticks': 100, 'frequency': 1000}
            samples = [clock, OSError('injected host bridge failure')] if failure == 'clock' else [clock]
            with patch.object(runner, '__file__', str(script)), \
                    patch.object(runner, 'validate', return_value=manifest), \
                    patch.object(runner, 'host_clock', side_effect=samples), \
                    patch.object(runner.subprocess, 'run', return_value=subprocess.CompletedProcess([], 0, b'', b'')):
                if failure == 'registry':
                    with patch.object(runner, 'track_process', side_effect=OSError('injected fsync failure')):
                        self.assertEqual(runner.supervise('manifest', 'gate', '1'), 4)
                else:
                    self.assertEqual(runner.supervise('manifest', 'gate', '1'), 4)
            marker = json.loads((root / 'STOP-SERIES').read_text())
            evidence = json.loads((root / 'forced-cleanup-test.json').read_text())
            self.assertTrue(marker['recoveryRequired'])
            self.assertTrue((root / 'ACTIVE-SUPERVISOR').exists())
            self.assertFalse(evidence['normalTurnStopsProved'])
            self.assertNotIn('runner-stop-unknown', evidence['errors'])

    def test_registry_failure_after_launch_stops_the_child(self):
        self.supervise_failure('registry')

    def test_clock_failure_stops_without_another_clock_query(self):
        self.supervise_failure('clock')

    def test_host_deadline_forces_stop_with_guest_process_still_live(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            script = root / 'sleep.py'
            script.write_text('import time\ntime.sleep(60)\n')
            manifest = {'nightRoot': folder, 'packets': [{'name': 'deadline', 'store': 'owned-deadline'}]}
            base = {'utc': 1792620000000, 'ticks': 0, 'frequency': 1000}
            samples = [base, dict(base, ticks=2670000), dict(base, ticks=2686000)]
            with patch.object(runner, '__file__', str(script)), \
                    patch.object(runner, 'validate', return_value=manifest), \
                    patch.object(runner, 'host_clock', side_effect=samples), \
                    patch.object(runner.subprocess, 'run', return_value=subprocess.CompletedProcess([], 0, b'', b'')):
                self.assertEqual(runner.supervise('manifest', 'gate', '1'), 4)
            marker = json.loads((root / 'STOP-SERIES').read_text())
            self.assertIn('deadline', marker['reason'])
            evidence = json.loads((root / 'forced-cleanup-deadline.json').read_text())
            self.assertEqual(evidence['runningAfter'], [])
            self.assertFalse(evidence['normalTurnStopsProved'])

    def test_clean_child_releases_supervisor_only_after_closed_ledger(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            script = root / 'closed.py'
            script.write_text('from pathlib import Path\n'
                              'Path(__file__).with_name("attempts.jsonl").write_text(\'{"name":"clean","status":"closed"}\\n\')\n')
            manifest = {'nightRoot': folder, 'packets': [{'name': 'clean', 'store': 'owned-clean'}]}
            clock = {'utc': 1792620000000, 'ticks': 0, 'frequency': 1000}
            with patch.object(runner, '__file__', str(script)), \
                    patch.object(runner, 'validate', return_value=manifest), \
                    patch.object(runner, 'host_clock', return_value=clock):
                self.assertEqual(runner.supervise('manifest', 'gate', '1'), 0)
            self.assertFalse((root / 'ACTIVE-SUPERVISOR').exists())
            self.assertFalse((root / 'STOP-SERIES').exists())

    def test_corrupt_registry_line_does_not_hide_valid_owned_group(self):
        with tempfile.TemporaryDirectory() as folder:
            registry = Path(folder) / 'groups'
            identity = {'pid': 100, 'group': 100, 'start': 'owned'}
            registry.write_text('not json\n' + json.dumps(identity) + '\n')
            with patch.object(runner, 'process_identity', return_value=identity), \
                    patch.object(runner.os, 'killpg') as kill:
                with self.assertRaises(ValueError):
                    runner.signal_tracked(registry, signal.SIGKILL)
                kill.assert_called_once_with(100, signal.SIGKILL)


if __name__ == '__main__':
    unittest.main()
