"""Linux operator controls: real owned processes, no Docker or provider starts.

    python3 -B -m unittest discover -s experiments/apply-trials -p night_run_test.py
"""
import json
import copy
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import time
import unittest
from unittest.mock import patch

import night_run as runner


class SupervisorControls(unittest.TestCase):
    def sol3_manifest(self, root):
        runner.write(root / 'authority.json', {'series': 'SOL3-20261022', 'seriesRoot': str(root),
                                             'starts': 3, 'summedSeconds': 8100})
        m = {'series': 'SOL3-20261022', 'seriesRoot': str(root), 'group': 'SOL3-P-R-W', 'status': 'frozen',
             'sourceCommit': 'frozen-source', 'runnerSha256': runner.sha(runner.__file__),
             'authoritySha256': runner.sha(root / 'authority.json'),
             'provider': {'name': 'codex-app-server', 'model': 'gpt-6-luna', 'effort': 'high',
                          'credentialMode': 'dedicated-runtime-login', 'credentialSourceVolume': 'mk-codex-runtime-auth',
                          'minimumRemainingQuotaPercent': 10},
             'ceilings': {'starts': 3, 'attemptSeconds': 2700, 'summedSeconds': 8100, 'retries': 0,
                          'workDeadlineFromAttemptStartSeconds': 1740, 'providerDeadlineFromAttemptStartSeconds': 2460,
                          'outerProcessDeadlineFromAttemptStartSeconds': 2490, 'writerStopCleanupReserveSeconds': 180,
                          'wrapperSoftSeconds': 2670, 'wrapperKillGraceSeconds': 15},
             'limits': {'turnSeconds': 300, 'turns': 240, 'passes': 8, 'probesPerTurn': 12},
             'packets': [{'name': name, 'scenario': name, 'store': 'owned-' + name.lower(), 'mode': 'apply',
                          'startNumber': i} for i, name in enumerate(('SOL3-P', 'SOL3-R', 'SOL3-W'), 1)]}
        path, gate = root / 'manifest.json', root / 'gate.json'
        runner.write(path, m)
        g = {'startAuthorized': True, 'manifestSha256': runner.sha(path), 'sourceCommit': m['sourceCommit'],
             'cReady': True, 'dockerSlotApproved': True, 'codexQuotaReserveApproved': True,
             'coordinatorMessage': 'source-bound independent release', 'cEvidenceSha256': 'review-digest',
             'independentReviewEvidence': 'reviewed-head', 'authorizedAttempts': [p['name'] for p in m['packets']],
             'series': m['series'], 'seriesRoot': str(root), 'operatorReleaseUtc': '2026-10-22T10:00:00Z',
             'absoluteStartCutoffUtc': '2026-10-22T11:30:00Z', 'absoluteAttemptStopUtc': '2026-10-22T12:15:00Z',
             'writersStoppedByUtc': '2026-10-22T12:15:00Z'}
        runner.write(gate, g)
        (root / 'attempts.jsonl').touch()
        return path, gate, m, g

    def test_sol3_release_binds_fresh_root_and_exact_finite_utc_grant(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            path, gate, m, g = self.sol3_manifest(root)
            validated = runner.validate(path, gate, files=False)
            starts, seconds, release, cutoff, end = runner.series_bounds(validated)
            self.assertEqual((starts, seconds), (3, 8100))
            self.assertEqual((end - release).total_seconds(), 8100)
            self.assertEqual((end - cutoff).total_seconds(), 2700)
            self.assertEqual(runner.series_root(validated), root)
            for key, bad in (('series', 'NIGHT-20261021'), ('seriesRoot', str(root / 'other')),
                             ('absoluteAttemptStopUtc', '2026-10-22T12:15:01Z')):
                runner.write(gate, dict(g, **{key: bad}))
                with self.subTest(key=key), self.assertRaises(runner.GuardRefusal):
                    runner.validate(path, gate, files=False)
            runner.write(gate, g)
            for key, bad in (('starts', 4), ('summedSeconds', 10800),
                             ('providerDeadlineFromAttemptStartSeconds', 2400), ('writerStopCleanupReserveSeconds', 179)):
                altered = copy.deepcopy(m)
                altered['ceilings'][key] = bad
                runner.write(path, altered)
                runner.write(gate, dict(g, manifestSha256=runner.sha(path)))
                with self.subTest(ceiling=key), self.assertRaises(runner.GuardRefusal):
                    runner.validate(path, gate, files=False)

    def test_sol3_append_only_full_charges_refuse_fourth_start_and_changed_release(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            path, gate, _, _ = self.sol3_manifest(root)
            m = runner.validate(path, gate, files=False)
            clock = {'utc': 1792663200000, 'ticks': 1, 'frequency': 1000}  # 10:00 UTC
            ledger = root / 'attempts.jsonl'
            for packet in m['packets']:
                before = ledger.read_bytes()
                reservation = runner.reserve_dispatch(root, packet, m, path, clock)
                reserved = ledger.read_bytes()
                self.assertTrue(reserved.startswith(before))
                records = runner.read_ledger(ledger, m)
                runner.reserved_dispatch(root, packet, path, records, m)
                runner.close_ledger(ledger, records[:-1], dict(reservation, status='closed', result={}), m)
                self.assertTrue(ledger.read_bytes().startswith(reserved), 'closing rewrote a reservation')
                (root / 'ACTIVE-ATTEMPT').unlink()
                (root / 'ACTIVE-SUPERVISOR').unlink()
                if packet['startNumber'] == 1:
                    with self.assertRaises(runner.GuardRefusal):
                        runner.reserve_dispatch(root, m['packets'][1], dict(m, gateSha256='different-release'), path, clock)
            self.assertEqual(len(ledger.read_text().splitlines()), 6)
            self.assertEqual(sum(r['chargedSeconds'] for r in runner.read_ledger(ledger, m)), 8100)
            with self.assertRaises(runner.GuardRefusal):
                runner.reserve_dispatch(root, dict(m['packets'][2], name='fourth', startNumber=4), m, path, clock)

    def test_actual_quota_reserve_boundary_and_account_limits(self):
        quota = {'primary': {'usedPercent': 90}, 'secondary': None, 'rateLimitReachedType': None}
        metadata = {'quota': {'rateLimitsByLimitId': {'codex': quota}, 'rateLimits': None}}
        runner.validate_quota(metadata, 10)
        for changed in ({'primary': {'usedPercent': 91}}, {'secondary': {'usedPercent': 91}},
                        {'rateLimitReachedType': 'limit'}, {'spendControlReached': True}):
            with self.subTest(change=changed), self.assertRaises(runner.GuardRefusal):
                runner.validate_quota({'quota': {'rateLimitsByLimitId': {'codex': dict(quota, **changed)},
                                                'rateLimits': None}}, 10)
        with self.assertRaises(runner.GuardRefusal):
            runner.validate_quota(metadata, 25)

    def test_only_sol3_authorizes_ten_percent_reserve(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            path, gate, m, g = self.sol3_manifest(root)
            runner.validate(path, gate, files=False)
            for invalid in (9, 25):
                altered = copy.deepcopy(m)
                altered['provider']['minimumRemainingQuotaPercent'] = invalid
                runner.write(path, altered)
                runner.write(gate, dict(g, manifestSha256=runner.sha(path)))
                with self.subTest(reserve=invalid), self.assertRaises(runner.GuardRefusal):
                    runner.validate(path, gate, files=False)
            m.update(series='NIGHT-20261021', nightRoot=str(root),
                     absoluteStartCutoffUtc='2026-10-22T04:00:00Z',
                     absoluteAttemptStopUtc='2026-10-22T04:45:00Z', writersStoppedByUtc='2026-10-22T04:55:00Z')
            m['provider'].update(model='gpt-6.1-sol', minimumRemainingQuotaPercent=25)
            m['ceilings'].update(starts=6, summedSeconds=16200)
            for packet in m['packets']:
                packet['scenario'] = 'NIGHT-P'
            runner.write(path, m)
            runner.write(gate, dict(g, manifestSha256=runner.sha(path)))
            runner.validate(path, gate, files=False)
            m['provider']['minimumRemainingQuotaPercent'] = 10
            runner.write(path, m)
            runner.write(gate, dict(g, manifestSha256=runner.sha(path)))
            with self.assertRaises(runner.GuardRefusal):
                runner.validate(path, gate, files=False)

    def test_sol3_refuses_sol_or_an_unbound_effort_after_luna_owner_change(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            path, gate, m, g = self.sol3_manifest(root)
            for key, value in (('model', 'gpt-6.1-sol'), ('model', 'gpt-6-astra'), ('effort', 'medium')):
                altered = copy.deepcopy(m)
                altered['provider'][key] = value
                runner.write(path, altered)
                runner.write(gate, dict(g, manifestSha256=runner.sha(path)))
                with self.subTest(key=key, value=value), self.assertRaises(runner.GuardRefusal):
                    runner.validate(path, gate, files=False)

    def test_sol3_before_release_or_without_full_remaining_attempt_is_refused(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            path, gate, _, _ = self.sol3_manifest(root)
            m = runner.validate(path, gate, files=False)
            _, _, release, cutoff, _ = runner.series_bounds(m)
            for instant in (release.timestamp() - 1, cutoff.timestamp(), cutoff.timestamp() + 1):
                with self.subTest(utc=instant), self.assertRaises(runner.GuardRefusal):
                    runner.reserve_dispatch(root, m['packets'][0], m, path,
                                            {'utc': instant * 1000, 'ticks': 1, 'frequency': 1000})
            self.assertEqual((root / 'attempts.jsonl').read_bytes(), b'')

    def test_sol3_unsafe_or_partial_reservation_never_allows_a_following_start(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            path, gate, _, _ = self.sol3_manifest(root)
            m = runner.validate(path, gate, files=False)
            clock = {'utc': 1792663200000, 'ticks': 1, 'frequency': 1000}
            first = runner.reserve_dispatch(root, m['packets'][0], m, path, clock)
            ledger = root / 'attempts.jsonl'
            for status in (None, 'unsafe'):
                if status:
                    runner.close_ledger(ledger, [], dict(first, status=status, result={}), m)
                for marker in ('ACTIVE-ATTEMPT', 'ACTIVE-SUPERVISOR'):
                    (root / marker).unlink(missing_ok=True)  # even lost markers cannot reset the charge
                with self.assertRaises(runner.GuardRefusal):
                    runner.reserve_dispatch(root, m['packets'][1], m, path, clock)

    def test_historical_night_bounds_remain_expired_and_unchanged(self):
        self.assertEqual(runner.series_bounds({'series': 'NIGHT-20261021'})[:2], (6, 16200))
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            m, path = self.dispatch_manifest(root, 'old-night')
            with self.assertRaises(runner.GuardRefusal):
                runner.reserve_dispatch(root, m['packets'][0], m, path,
                                        {'utc': 1792663200000, 'ticks': 1, 'frequency': 1000})
            self.assertEqual((root / 'attempts.jsonl').read_bytes(), b'')

    def assert_process_stopped(self, identity):
        for _ in range(500):
            if runner.process_identity(identity['pid']) != identity:
                return
            fields = Path('/proc/%d/stat' % identity['pid']).read_text().rsplit(')', 1)[1].split()
            if fields[0] == 'Z':
                return  # exited orphan awaiting the system reaper, not a writer
            time.sleep(0.01)
        self.fail('owned process still live: %s' % identity)

    def test_direct_command_timeout_stops_term_resistant_grandchild_and_preserves_foreign_process(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            ready = root / 'grandchild.json'
            registry = root / 'groups'
            registry.touch()
            module_dir = str(Path(runner.__file__).resolve().parent)
            grandchild = ('import sys,signal,time,json\n'
                          'sys.path.insert(0, ' + repr(module_dir) + ')\n'
                          'import night_run as runner\n'
                          'signal.signal(signal.SIGTERM, signal.SIG_IGN)\n'
                          'open(' + repr(str(ready)) + ', "w").write(json.dumps(runner.process_identity(runner.os.getpid())))\n'
                          'time.sleep(60)\n')
            parent = ('import subprocess,sys,time\n'
                      'subprocess.Popen([sys.executable,"-B","-c",' + repr(grandchild) + '])\n'
                      'time.sleep(60)\n')
            foreign = subprocess.Popen([sys.executable, '-c', 'import time; time.sleep(60)'], start_new_session=True)
            timing = runner.AttemptTiming(time.monotonic(), 'execution')
            try:
                with patch.object(runner, 'host_clock', side_effect=AssertionError('no bridge in command/stop')):
                    with self.assertRaises(subprocess.TimeoutExpired):
                        runner.run_tracked([sys.executable, '-B', '-c', parent], registry, 0.5, timing,
                                           capture_output=True)
                self.assertTrue(ready.exists(), 'timeout did not exercise a live grandchild')
                self.assert_process_stopped(json.loads(ready.read_text()))
                self.assert_process_stopped(json.loads(registry.read_text()))
                self.assertIsNone(foreign.poll())
                self.assertGreater(timing.seconds['stop'], 0)
            finally:
                foreign.terminate()
                foreign.wait(timeout=5)
                if ready.exists():
                    identity = json.loads(ready.read_text())
                    if runner.process_identity(identity['pid']) == identity:
                        os.kill(identity['pid'], signal.SIGKILL)

    def test_direct_command_success_and_nonzero_are_not_timeout_success(self):
        with tempfile.TemporaryDirectory() as folder:
            registry = Path(folder) / 'groups'
            timing = runner.AttemptTiming(time.monotonic())
            for code in (0, 7):
                done = runner.run_tracked([sys.executable, '-c', 'print("done"); exit(%d)' % code],
                                          registry, 5, timing, capture_output=True)
                self.assertEqual(done.returncode, code)
                self.assertEqual(done.stdout, b'done\n')

    def test_local_command_cap_counts_setup_without_clock_queries(self):
        with patch.object(runner.time, 'monotonic', return_value=150), \
                patch.object(runner, 'host_clock', side_effect=AssertionError('supervisor owns polling')):
            self.assertEqual(runner.remaining_seconds(100, 60), 10)
            with self.assertRaises(TimeoutError):
                runner.remaining_seconds(100, 50)

    def test_phase_measurements_are_exclusive_monotonic_intervals(self):
        timing = runner.AttemptTiming(100)
        with patch.object(runner.time, 'monotonic', side_effect=[105, 115, 118, 120, 122]):
            timing.enter('execution')
            timing.enter('stop')
            timing.enter('cleanup')
            timing.enter('finalization')
            measured = timing.snapshot()['seconds']
        self.assertEqual(measured, dict(setup=5, execution=10, stop=3, cleanup=2, evaluation=0, finalization=2))

    def test_absolute_start_cutoff_uses_utc_even_with_foreign_host_timezone(self):
        with tempfile.TemporaryDirectory() as folder, patch.dict(os.environ, TZ='Pacific/Honolulu'):
            root = Path(folder)
            manifest, path = self.dispatch_manifest(root, 'utc')
            clock = {'utc': runner.ABSOLUTE_START_CUTOFF.timestamp() * 1000, 'ticks': 1, 'frequency': 1000}
            with self.assertRaises(runner.GuardRefusal):
                runner.reserve_dispatch(root, manifest['packets'][0], manifest, path, clock)
            self.assertFalse((root / 'ACTIVE-SUPERVISOR').exists())
            self.assertEqual((root / 'attempts.jsonl').read_text(), '')

    def test_child_alarm_deducts_supervisor_setup_time(self):
        started = {'utc': 1792620000000, 'ticks': 1, 'frequency': 1000}
        with patch.dict(os.environ, MARKITECT_NIGHT_PROCESS_REGISTRY='unused',
                        MARKITECT_NIGHT_STARTED_MONOTONIC='100', MARKITECT_NIGHT_STARTED_HOST=json.dumps(started)), \
                patch.object(runner, 'arm_supervisor_liveness_guard'), \
                patch.object(runner.time, 'monotonic', return_value=150), \
                patch.object(runner.signal, 'signal'), patch.object(runner.signal, 'setitimer') as timer, \
                patch.object(runner, 'main', return_value=0):
            self.assertEqual(runner.bounded_child(), 0)
        timer.assert_called_once_with(signal.ITIMER_REAL, 2620)

    def test_abrupt_supervisor_death_stops_the_exact_bounded_child(self):
        if not sys.platform.startswith('linux'):
            self.skipTest('NIGHT runner lifecycle is Linux-only')
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            manifest, path = self.dispatch_manifest(root, 'parent-abort')
            child_script = root / 'bounded-child.py'
            ready = root / 'child-ready'
            module_dir = str(Path(runner.__file__).resolve().parent)
            child_script.write_text(
                'import sys, time\n'
                'sys.path.insert(0, ' + repr(module_dir) + ')\n'
                'import night_run as runner\n'
                'def work():\n'
                '    open(' + repr(str(ready)) + ', "w").close()\n'
                '    runner.run_tracked([sys.executable, "-c", "import time; time.sleep(60)"],\n'
                '        runner.os.environ["MARKITECT_NIGHT_PROCESS_REGISTRY"], 60, runner.AttemptTiming(time.monotonic()))\n'
                'runner.main = work\n'
                'try: runner.bounded_child()\n'
                'except InterruptedError: sys.exit(0)\n')
            parent_script = root / 'supervisor.py'
            parent_script.write_text(
                'import json, sys\n'
                'from pathlib import Path\n'
                'sys.path.insert(0, ' + repr(module_dir) + ')\n'
                'import night_run as runner\n'
                'root = Path(' + repr(str(root)) + ')\n'
                'manifest = json.loads(Path(' + repr(path) + ').read_text())\n'
                'runner.__file__ = ' + repr(str(child_script)) + '\n'
                'runner.validate = lambda *args, **kwargs: manifest\n'
                'runner.host_clock = lambda: {"utc": 1792620000000, "ticks": 0, "frequency": 1000}\n'
                'runner.supervise(' + repr(path) + ', "gate", "1")\n')
            env = dict(os.environ, PYTHONPATH=module_dir)
            parent = subprocess.Popen([sys.executable, '-B', str(parent_script)], cwd=root, env=env,
                                      stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            child_identity = None
            try:
                registry = root / 'process-groups-parent-abort.jsonl'
                for _ in range(500):
                    if child_identity is None and registry.exists() and registry.stat().st_size:
                        child_identity = json.loads(registry.read_text().splitlines()[0])
                    if ready.exists() and child_identity is not None and len(registry.read_text().splitlines()) == 2:
                        break
                    if parent.poll() is not None:
                        self.fail('supervisor exited before child reached its bounded work phase')
                    time.sleep(0.01)
                self.assertIsNotNone(child_identity, 'bounded child did not become ready')
                marker = json.loads((root / 'ACTIVE-SUPERVISOR').read_text())
                self.assertEqual(marker['supervisorIdentity']['pid'], parent.pid)
                os.kill(parent.pid, signal.SIGKILL)
                parent.wait(timeout=5)
                pid = child_identity['pid']
                stopped = False
                for _ in range(500):
                    current = runner.process_identity(pid)
                    if current != child_identity:
                        stopped = True
                    else:
                        fields = Path('/proc/%d/stat' % pid).read_text().rsplit(')', 1)[1].split()
                        stopped = fields[0] == 'Z'
                    if stopped:
                        break
                    time.sleep(0.01)
                self.assertTrue(stopped, 'bounded child survived abrupt supervisor death')
                command_identity = json.loads(registry.read_text().splitlines()[1])
                self.assert_process_stopped(command_identity)
                self.assertFalse((root / 'STOP-SERIES').exists())
                self.assertTrue((root / 'ACTIVE-ATTEMPT').exists(), 'unclosed reservation must remain blocking')
            finally:
                if parent.poll() is None:
                    parent.kill()
                    parent.wait(timeout=5)
                if child_identity is not None:
                    try:
                        pidfd = os.pidfd_open(child_identity['pid'])
                    except ProcessLookupError:
                        pidfd = None
                    if pidfd is not None:
                        try:
                            if runner.process_identity(child_identity['pid']) == child_identity:
                                signal.pidfd_send_signal(pidfd, signal.SIGKILL)
                        except ProcessLookupError:
                            pass
                        finally:
                            os.close(pidfd)

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

    def test_clock_query_keeps_five_second_bound_and_safe_origin_diagnostic(self):
        error = subprocess.TimeoutExpired([runner.HOST_CLOCK, 'PRIVATE-PAYLOAD'], 5)
        with patch.object(runner.subprocess, 'run', side_effect=error) as query:
            with self.assertRaises(subprocess.TimeoutExpired):
                runner.host_clock()
        self.assertEqual(query.call_args.kwargs['timeout'], 5)
        diagnostic = runner.failure_diagnostic(error, 'controlled clock query')
        self.assertEqual(diagnostic['failedProbe']['configuredTimeoutSeconds'], 5)
        self.assertNotIn('PRIVATE-PAYLOAD', json.dumps(diagnostic))

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
            error = (subprocess.TimeoutExpired([runner.HOST_CLOCK, 'PRIVATE-PAYLOAD'], 5)
                     if failure == 'timeout' else OSError('injected host bridge failure'))
            samples = [clock, error] if failure in ('clock', 'timeout') else [clock]
            with patch.object(runner, '__file__', str(script)), \
                    patch.object(runner, 'validate', return_value=manifest), \
                    patch.object(runner, 'host_clock', side_effect=samples) as clock_query, \
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
            rows = [json.loads(x) for x in (root / 'attempts.jsonl').read_text().splitlines()]
            self.assertEqual(len(rows), 1)
            self.assertEqual(rows[0]['chargedSeconds'], 2700)
            self.assertTrue((root / 'ACTIVE-ATTEMPT').exists())
            if failure == 'timeout':
                self.assertEqual(clock_query.call_count, 2)  # one initial sample, one failed poll; no retry
                self.assertIn('host-clock probe', marker['reason'])
                self.assertNotIn('PRIVATE-PAYLOAD', json.dumps(marker))

    def test_registry_failure_after_launch_stops_the_child(self):
        self.supervise_failure('registry')

    def test_clock_failure_stops_without_another_clock_query(self):
        self.supervise_failure('clock')

    def test_clock_probe_timeout_stops_without_retry_and_retains_full_charge(self):
        self.supervise_failure('timeout')

    def test_unknown_container_cessation_retains_charge_and_blocks_another_start(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            script = root / 'refuse.py'
            script.write_text('raise SystemExit(3)\n')
            manifest, path = self.dispatch_manifest(root, 'unknown-stop')
            clock = {'utc': 1792620000000, 'ticks': 0, 'frequency': 1000}
            with patch.object(runner, '__file__', str(script)), \
                    patch.object(runner, 'validate', return_value=manifest), \
                    patch.object(runner, 'host_clock', return_value=clock), \
                    patch.object(runner.subprocess, 'run', side_effect=subprocess.TimeoutExpired('owned Docker query', 2)):
                self.assertEqual(runner.supervise(path, 'gate', '1'), 4)
                before = (root / 'attempts.jsonl').read_bytes()
                with patch.object(runner.subprocess, 'Popen') as launch, self.assertRaises(runner.GuardRefusal):
                    runner.supervise(path, 'gate', '1')
                launch.assert_not_called()
            evidence = json.loads((root / 'forced-cleanup-unknown-stop.json').read_text())
            self.assertIsNone(evidence['runningAfter'])
            self.assertFalse(evidence['normalTurnStopsProved'])
            self.assertIn('TimeoutExpired', evidence['errors'])
            self.assertEqual((root / 'attempts.jsonl').read_bytes(), before)
            self.assertEqual(json.loads(before)['chargedSeconds'], 2700)
            self.assertTrue((root / 'ACTIVE-SUPERVISOR').exists())
            self.assertTrue((root / 'ACTIVE-ATTEMPT').exists())
            self.assertTrue((root / 'STOP-SERIES').exists())

    def test_host_deadline_forces_stop_with_guest_process_still_live(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            script = root / 'sleep.py'
            script.write_text('import time\ntime.sleep(60)\n')
            manifest, path = self.dispatch_manifest(root, 'deadline')
            base = {'utc': 1792620000000, 'ticks': 0, 'frequency': 1000}
            # Poll just before the unchanged deadline, then 0.9 s after it.
            samples = [base, dict(base, ticks=2669900), dict(base, ticks=2670900)]
            with patch.object(runner, '__file__', str(script)), \
                    patch.object(runner, 'validate', return_value=manifest), \
                    patch.object(runner, 'host_clock', side_effect=samples) as clock_query, \
                    patch.object(runner.subprocess, 'run', return_value=subprocess.CompletedProcess([], 0, b'', b'')), \
                    patch.object(runner.time, 'sleep') as sleep:
                self.assertEqual(runner.supervise(path, 'gate', '1'), 4)
            self.assertEqual(clock_query.call_count, 3, 'stop/cleanup must not restart the host bridge')
            # Popen.wait also uses this shared time module for subsecond reaping.
            self.assertEqual(sleep.call_args_list[0].args, (1,))
            self.assertEqual(sum(call.args == (1,) for call in sleep.call_args_list), 1)
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
