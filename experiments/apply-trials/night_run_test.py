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
    def test_prelaunch_failures_report_committed_or_uncertain_reservation(self):
        for phase in ('partial-write', 'registry', 'output', 'launch'):
            with self.subTest(phase=phase), tempfile.TemporaryDirectory() as folder:
                root = Path(folder)
                manifest, path = self.dispatch_manifest(root, 'prelaunch')
                if phase == 'registry':
                    (root / 'process-groups-prelaunch.jsonl').touch()
                if phase == 'output':
                    (root / 'supervisor-child-prelaunch.jsonl').touch()
                clock = {'utc': 1792620000000, 'ticks': 0, 'frequency': 1000}
                sync = runner.sync_dir
                with patch.object(runner, 'validate', return_value=manifest), \
                        patch.object(runner, 'host_clock', return_value=clock), \
                        patch.object(runner, 'sync_dir', side_effect=OSError('private payload') if phase == 'partial-write' else sync), \
                        patch.object(runner.subprocess, 'Popen', side_effect=OSError('private payload')) as launch:
                    with self.assertRaises(OSError) as failure:
                        runner.supervise(path, 'gate', '1')
                    diagnostic = runner.supervisor_failure(failure.exception)
                    if phase != 'launch':
                        launch.assert_not_called()
                self.assertNotIn('private payload', json.dumps(diagnostic))
                self.assertEqual(diagnostic['status'], 'reservation-write-uncertain' if phase == 'partial-write' else 'reserved-no-child')
                self.assertEqual(diagnostic['reservedSeconds'], 2700)
                self.assertTrue(diagnostic['recoveryRequired'])
                self.assertTrue((root / 'ACTIVE-SUPERVISOR').exists())
                rows = (root / 'attempts.jsonl').read_text().splitlines()
                self.assertEqual(len(rows), 0 if phase == 'partial-write' else 1)

    def test_child_bootstrap_diagnostics_do_not_echo_environment_payload(self):
        for anchor in (None, 'private-invalid-json'):
            env = dict(runner.os.environ)
            env.pop('MARKITECT_NIGHT_PROCESS_REGISTRY', None)
            if anchor is not None:
                env.update(MARKITECT_NIGHT_PROCESS_REGISTRY='unused', MARKITECT_NIGHT_STARTED_HOST=anchor)
            done = subprocess.run([sys.executable, '-B', runner.__file__, '--bounded-child'],
                                  env=env, capture_output=True, timeout=5)
            self.assertEqual(done.returncode, 3)
            diagnostic = json.loads(done.stdout)
            self.assertEqual(diagnostic['phase'], 'child preparation')
            self.assertNotIn('private-invalid-json', done.stdout.decode() + done.stderr.decode())
            self.assertEqual(done.stderr, b'')

    def test_signal_during_child_preflight_retains_reserved_attempt_identity(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            manifest, path = self.dispatch_manifest(root, 'signal-preflight')
            clock = {'utc': 1792620000000, 'ticks': 0, 'frequency': 1000}
            runner.reserve_dispatch(root, manifest['packets'][0], manifest, path, clock)
            def interrupted_clock():
                runner.interrupt(signal.SIGTERM, None)
            with patch.object(runner, 'validate', return_value=manifest), \
                    patch.object(runner, 'host_clock', side_effect=interrupted_clock), \
                    patch.object(runner, 'process_started_host', clock, create=True), \
                    patch.dict(runner.INTERRUPT, seen=False, finalizing=False, committed=False, attempt=None), \
                    patch.object(runner.sys, 'argv', ['night_run.py', path, 'gate', '1']):
                with self.assertRaises(InterruptedError):
                    runner.main()
            self.assertEqual(json.loads((root / 'STOP-SERIES').read_text())['attempt'], 'signal-preflight')
            self.assertEqual(len((root / 'attempts.jsonl').read_text().splitlines()), 1)

    def test_signal_during_ledger_validation_has_authorized_attempt_identity(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            manifest, path = self.dispatch_manifest(root, 'signal-ledger')
            clock = {'utc': 1792620000000, 'ticks': 0, 'frequency': 1000}
            runner.reserve_dispatch(root, manifest['packets'][0], manifest, path, clock)
            with patch.object(runner, 'validate', return_value=manifest), \
                    patch.object(runner, 'reserved_dispatch', side_effect=lambda *args: runner.interrupt(signal.SIGTERM, None)), \
                    patch.dict(runner.INTERRUPT, seen=False, finalizing=False, committed=False, attempt=None, stopPath=None), \
                    patch.object(runner.sys, 'argv', ['night_run.py', path, 'gate', '1']):
                with self.assertRaises(InterruptedError):
                    runner.main()
            self.assertEqual(json.loads((root / 'STOP-SERIES').read_text())['attempt'], 'signal-ledger')

    def test_interrupt_immediately_after_launch_forces_child_cessation(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            script = root / 'sleep.py'
            script.write_text('import time\ntime.sleep(60)\n')
            manifest, path = self.dispatch_manifest(root, 'launch-interrupted')
            clock = {'utc': 1792620000000, 'ticks': 0, 'frequency': 1000}
            class InterruptedState(dict):
                def update(self, *args, **kwargs):
                    if kwargs.get('childLaunched'):
                        raise KeyboardInterrupt()
                    return super().update(*args, **kwargs)
            with patch.object(runner, '__file__', str(script)), \
                    patch.object(runner, 'SUPERVISION', InterruptedState()), \
                    patch.object(runner, 'validate', return_value=manifest), \
                    patch.object(runner, 'host_clock', return_value=clock), \
                    patch.object(runner.subprocess, 'run', return_value=subprocess.CompletedProcess([], 0, b'', b'')):
                self.assertEqual(runner.supervise(path, 'gate', '1'), 4)
            result = json.loads((root / 'forced-cleanup-launch-interrupted.json').read_text())
            self.assertNotIn('runner-stop-unknown', result['errors'])
            self.assertEqual(result['runningAfter'], [])
            self.assertTrue((root / 'ACTIVE-SUPERVISOR').exists())
            self.assertTrue((root / 'STOP-SERIES').exists())

    def test_second_interrupt_waits_for_owned_docker_cleanup_and_evidence(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            registry = root / 'groups'
            registry.touch()
            child = subprocess.Popen([sys.executable, '-c', 'import time; time.sleep(60)'], start_new_session=True)
            original_handler = signal.signal(signal.SIGINT, signal.default_int_handler)
            try:
                def queue_interrupt(*args):
                    runner.os.kill(runner.os.getpid(), signal.SIGINT)
                replies = [subprocess.CompletedProcess([], 0, data, b'') for data in (b'owned-container\n', b'', b'')]
                with patch.object(runner, 'signal_tracked', side_effect=queue_interrupt), \
                        patch.object(runner.subprocess, 'run', side_effect=replies) as docker:
                    with self.assertRaises(KeyboardInterrupt):
                        runner.force_supervised_stop(child, registry, root,
                                                     {'name': 'twice', 'store': 'owned-twice'}, 'first interrupt')
                self.assertIsNotNone(child.poll())
                self.assertEqual(docker.call_args_list[1].args[0], ['/usr/bin/docker', 'kill', 'owned-container'])
                result = json.loads((root / 'forced-cleanup-twice.json').read_text())
                self.assertEqual(result['runningAfter'], [])
                self.assertEqual(result['errors'], [])
                self.assertTrue((root / 'STOP-SERIES').exists())
            finally:
                signal.signal(signal.SIGINT, original_handler)
                if child.poll() is None:
                    runner.os.killpg(child.pid, signal.SIGKILL)
                child.wait()

    def dispatch_manifest(self, root, name, start_number=1, previous=()):
        (root / 'attempts.jsonl').write_text(''.join(json.dumps(row) + '\n' for row in previous))
        manifest = {'nightRoot': str(root), 'group': 'control', 'packets': [
            {'name': name, 'store': 'owned-' + name, 'scenario': 'NIGHT-VN',
             'mode': 'verify', 'startNumber': start_number}]}
        path = root / 'manifest.json'
        path.write_text(json.dumps(manifest))
        return manifest, str(path)

    def test_safe_guard_diagnostic_never_echoes_arbitrary_exception_payload(self):
        hidden = 'PRIVATE-CREDENTIAL-PAYLOAD'
        diagnostic = runner.failure_diagnostic(ValueError(hidden), 'controlled phase')
        self.assertNotIn(hidden, json.dumps(diagnostic))
        try:
            runner.require(False, 'global start number differs')
        except runner.GuardRefusal as error:
            diagnostic = runner.failure_diagnostic(error, 'controlled guard')
        self.assertEqual(diagnostic['failedGuard']['reason'], 'global start number differs')
        self.assertEqual(diagnostic['failedGuard']['source'], 'night_run.py')
        self.assertIsInstance(diagnostic['failedGuard']['line'], int)

    def test_first_position_of_later_group_consumes_global_reservation_once(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            previous = [{'name': 'earlier-%d' % i, 'status': 'closed', 'chargedSeconds': 2700} for i in (1, 2)]
            manifest, path = self.dispatch_manifest(root, 'later-first', 3, previous)
            packet = manifest['packets'][0]  # local position one, global start three
            clock = {'utc': 1792620000000, 'ticks': 0, 'frequency': 1000}
            reservation = runner.reserve_dispatch(root, packet, manifest, path, clock)
            before = (root / 'attempts.jsonl').read_bytes()
            records = [json.loads(line) for line in before.splitlines()]
            prior, consumed = runner.reserved_dispatch(root, packet, path, records)
            self.assertEqual(prior, previous)
            self.assertEqual(consumed, reservation)
            self.assertEqual(sum(row['chargedSeconds'] for row in records), 8100)
            self.assertEqual((root / 'attempts.jsonl').read_bytes(), before)
            with self.assertRaises(runner.GuardRefusal):
                runner.reserved_dispatch(root, dict(packet, startNumber=1), path, records)
            changed = root / 'different-manifest.json'
            changed.write_text('changed source-bound release')
            with self.assertRaises(runner.GuardRefusal):
                runner.reserved_dispatch(root, packet, str(changed), records)

    def test_child_prestart_refusal_keeps_full_charge_and_blocks_repeat(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            script = root / 'refuse.py'
            script.write_text('import json,sys\nfrom pathlib import Path\n'
                              'r=Path(__file__).parent\n'
                              'rows=[json.loads(x) for x in (r/"attempts.jsonl").read_text().splitlines()]\n'
                              'assert len(rows)==3 and rows[-1]["status"]=="running"\n'
                              'assert sum(x["chargedSeconds"] for x in rows)==8100\n'
                              'print(json.dumps({"phase":"controlled prestart guard","generationRequests":0}),flush=True)\n'
                              'sys.exit(3)\n')
            previous = [{'name': 'earlier-%d' % i, 'status': 'closed', 'chargedSeconds': 2700} for i in (1, 2)]
            manifest, path = self.dispatch_manifest(root, 'refused', 3, previous)
            clock = {'utc': 1792620000000, 'ticks': 0, 'frequency': 1000}
            with patch.object(runner, '__file__', str(script)), \
                    patch.object(runner, 'validate', return_value=manifest), \
                    patch.object(runner, 'host_clock', return_value=clock), \
                    patch.object(runner.subprocess, 'run', return_value=subprocess.CompletedProcess([], 0, b'', b'')):
                self.assertEqual(runner.supervise(path, 'gate', '1'), 4)
                before = (root / 'attempts.jsonl').read_bytes()
                with patch.object(runner.subprocess, 'Popen') as launch, self.assertRaises(runner.GuardRefusal):
                    runner.supervise(path, 'gate', '1')
                launch.assert_not_called()
            rows = [json.loads(line) for line in before.splitlines()]
            self.assertEqual(rows[:2], previous)
            self.assertEqual(rows[-1]['chargedSeconds'], 2700)
            self.assertEqual((root / 'attempts.jsonl').read_bytes(), before)
            self.assertTrue((root / 'ACTIVE-ATTEMPT').exists())
            self.assertTrue((root / 'ACTIVE-SUPERVISOR').exists())
            self.assertTrue((root / 'STOP-SERIES').exists())
            self.assertIn('controlled prestart guard', (root / 'supervisor-child-refused.jsonl').read_text())
            self.assertFalse((root / 'attempts/refused').exists())

    def test_real_main_uses_local_position_and_global_charge_before_live_guard(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            previous = [{'name': 'earlier-%d' % i, 'status': 'closed', 'chargedSeconds': 2700} for i in (1, 2)]
            manifest, path = self.dispatch_manifest(root, 'later-main', 3, previous)
            credential = root / 'preserve-existing-credential'
            credential.write_text('not a real credential')
            manifest['provider'] = {'credentialPath': str(credential)}
            Path(path).write_text(json.dumps(manifest))
            clock = {'utc': 1792620000000, 'ticks': 0, 'frequency': 1000}
            runner.reserve_dispatch(root, manifest['packets'][0], manifest, path, clock)
            before = (root / 'attempts.jsonl').read_bytes()
            with patch.object(runner, 'validate', return_value=manifest), \
                    patch.object(runner, 'host_clock', return_value=clock), \
                    patch.object(runner, 'process_started_host', clock, create=True), \
                    patch.dict(runner.INTERRUPT, dict(runner.INTERRUPT)), \
                    patch.object(runner.sys, 'argv', ['night_run.py', path, 'gate', '1']), \
                    patch.object(runner.subprocess, 'run') as docker, \
                    patch.object(runner.subprocess, 'Popen') as launch:
                with self.assertRaises(runner.GuardRefusal) as failure:
                    runner.main()
                self.assertEqual(failure.exception.diagnostic['reason'], 'unexpected credential file is preserved')
                docker.assert_not_called()
                launch.assert_not_called()
            self.assertEqual((root / 'attempts.jsonl').read_bytes(), before)
            self.assertEqual(credential.read_text(), 'not a real credential')
            self.assertFalse((root / 'attempts/later-main').exists())

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
            manifest, path = self.dispatch_manifest(root, 'test')
            clock = {'utc': 1792620000000, 'ticks': 100, 'frequency': 1000}
            samples = [clock, OSError('injected host bridge failure')] if failure == 'clock' else [clock]
            with patch.object(runner, '__file__', str(script)), \
                    patch.object(runner, 'validate', return_value=manifest), \
                    patch.object(runner, 'host_clock', side_effect=samples), \
                    patch.object(runner.subprocess, 'run', return_value=subprocess.CompletedProcess([], 0, b'', b'')):
                if failure == 'registry':
                    with patch.object(runner, 'track_process', side_effect=OSError('injected fsync failure')):
                        self.assertEqual(runner.supervise(path, 'gate', '1'), 4)
                else:
                    self.assertEqual(runner.supervise(path, 'gate', '1'), 4)
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
            manifest, path = self.dispatch_manifest(root, 'deadline')
            base = {'utc': 1792620000000, 'ticks': 0, 'frequency': 1000}
            samples = [base, dict(base, ticks=2670000), dict(base, ticks=2686000)]
            with patch.object(runner, '__file__', str(script)), \
                    patch.object(runner, 'validate', return_value=manifest), \
                    patch.object(runner, 'host_clock', side_effect=samples), \
                    patch.object(runner.subprocess, 'run', return_value=subprocess.CompletedProcess([], 0, b'', b'')):
                self.assertEqual(runner.supervise(path, 'gate', '1'), 4)
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
                              'import json\n'
                              'r=Path(__file__).parent\n'
                              'p=r/"attempts.jsonl"\n'
                              'rows=[json.loads(x) for x in p.read_text().splitlines()]\n'
                              'rows[-1]["status"]="closed"\n'
                              'p.write_text("".join(json.dumps(x)+"\\n" for x in rows))\n'
                              '(r/"ACTIVE-ATTEMPT").unlink()\n')
            manifest, path = self.dispatch_manifest(root, 'clean')
            clock = {'utc': 1792620000000, 'ticks': 0, 'frequency': 1000}
            with patch.object(runner, '__file__', str(script)), \
                    patch.object(runner, 'validate', return_value=manifest), \
                    patch.object(runner, 'host_clock', return_value=clock):
                self.assertEqual(runner.supervise(path, 'gate', '1'), 0)
            self.assertFalse((root / 'ACTIVE-SUPERVISOR').exists())
            self.assertFalse((root / 'STOP-SERIES').exists())

    def test_changed_supervisor_marker_is_preserved_after_clean_child_exit(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            script = root / 'changed-marker.py'
            script.write_text('from pathlib import Path\nimport json\n'
                              'r=Path(__file__).parent\np=r/"attempts.jsonl"\n'
                              'row=json.loads(p.read_text()); row["status"]="closed"\n'
                              'p.write_text(json.dumps(row)+"\\n")\n'
                              '(r/"ACTIVE-ATTEMPT").unlink()\n'
                              '(r/"ACTIVE-SUPERVISOR").write_text(json.dumps({"attempt":"changed-owner"}))\n')
            manifest, path = self.dispatch_manifest(root, 'changed')
            clock = {'utc': 1792620000000, 'ticks': 0, 'frequency': 1000}
            with patch.object(runner, '__file__', str(script)), \
                    patch.object(runner, 'validate', return_value=manifest), \
                    patch.object(runner, 'host_clock', return_value=clock), \
                    patch.object(runner.subprocess, 'run', return_value=subprocess.CompletedProcess([], 0, b'', b'')):
                self.assertEqual(runner.supervise(path, 'gate', '1'), 4)
            self.assertEqual(json.loads((root / 'ACTIVE-SUPERVISOR').read_text())['attempt'], 'changed-owner')
            self.assertTrue((root / 'STOP-SERIES').exists())

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
