"""One authorized, charged CCF-20261021-v1 Apply attempt (Linux operator wrapper). No automatic retries or model fallback.

Usage: python3 -B ccf_run.py MANIFEST COORDINATOR_GATE ATTEMPT_NUMBER
The final manifest and gate are deliberately absent during preparation.
"""
import datetime as dt
import fcntl
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import time

INTERRUPT = {'seen': False, 'finalizing': False, 'committed': False,
             'stopPath': None, 'attempt': None, 'markerWriteFailed': False}

def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

def require(condition, reason='bound precondition refused'):
    if not condition:
        raise ValueError(reason)

def validate(manifest_path, gate_path):
    m = json.loads(Path(manifest_path).read_text())
    g = json.loads(Path(gate_path).read_text())
    require(m['series'] == 'CCF-20261021-v1')
    require(m['status'] == 'frozen' and g['startAuthorized'] is True)
    require(g['manifestSha256'] == sha(manifest_path) and g['sourceCommit'] == m['sourceCommit'])
    require(g['cReady'] is True and g['dockerSlotApproved'] is True and g['codexQuotaReserveApproved'] is True)
    require(g['coordinatorMessage'] and g['cEvidenceSha256'] and g['independentReviewEvidence'])
    require(m['provider']['name'] == 'codex-app-server' and m['provider']['model'] == 'gpt-6.1-sol')
    require(m['provider']['effort'] == 'high' and m['provider']['credentialMode'] == 'dedicated-runtime-login')
    require(m['provider']['credentialSourceVolume'] == 'mk-codex-runtime-auth')
    require(m['provider']['minimumRemainingQuotaPercent'] == 25)
    require([p['scenario'] for p in m['packets']] == ['CCF-P', 'CCF-R', 'CCF-G'])
    require(m['ceilings']['starts'] == 3 and m['ceilings']['attemptSeconds'] == 2700)
    require(m['ceilings']['summedSeconds'] == 8100 and m['ceilings']['retries'] == 0)
    require(m['ceilings']['workDeadlineFromAttemptStartSeconds'] == 1740)
    require(m['ceilings']['providerDeadlineFromAttemptStartSeconds'] == 2460)
    require(m['ceilings']['outerProcessDeadlineFromAttemptStartSeconds'] == 2490)
    require(m['ceilings']['providerDeadlineFromAttemptStartSeconds'] - m['ceilings']['workDeadlineFromAttemptStartSeconds'] >= 720)
    require(m['ceilings']['attemptSeconds'] - m['ceilings']['outerProcessDeadlineFromAttemptStartSeconds'] >= 180)
    require(m['ceilings']['writerStopCleanupReserveSeconds'] >= 180)
    require(m['ceilings']['wrapperSoftSeconds'] == 2670 and m['ceilings']['wrapperKillGraceSeconds'] == 15)
    require(m['ceilings']['wrapperSoftSeconds'] - m['ceilings']['outerProcessDeadlineFromAttemptStartSeconds'] >= 180)
    require(m['ceilings']['wrapperSoftSeconds'] + m['ceilings']['wrapperKillGraceSeconds'] < m['ceilings']['attemptSeconds'])
    require(sha(Path(m['build']) / 'markitect') == m['cliSha256'])
    for p in m['packets']:
        require(sha(p['profile']) == p['profileSha256'])
        for relative, digest in p['packet']['files'].items():
            require(sha(Path(p['repo']) / relative) == digest)
    for relative, digest in m['sourceFiles'].items():
        require(sha(Path(m['source']) / relative) == digest)
    require(m['limits'] == {'turnSeconds': 240, 'turns': 240, 'passes': 8, 'probesPerTurn': 12})
    require(sha(__file__) == m['runnerSha256'])
    return m

def sync_dir(path):
    fd = os.open(path, os.O_RDONLY | os.O_DIRECTORY)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)

def write(path, value):
    path = Path(path)
    with path.open('w') as f:
        f.write(json.dumps(value, indent=2, sort_keys=True) + '\n')
        f.flush()
        os.fsync(f.fileno())
    sync_dir(path.parent)

def main():
    manifest_path, gate_path, position = sys.argv[1:]
    # Guard precedes Docker, credentials, ledger reservation and the binary.
    m = validate(manifest_path, gate_path)
    position = int(position)
    require(1 <= position <= 3)
    root = Path(manifest_path).resolve().parent
    INTERRUPT['stopPath'] = root / 'STOP-SERIES'
    require(not INTERRUPT['stopPath'].exists(), 'series stop marker is preserved')
    active = root / 'ACTIVE-ATTEMPT'
    require(not active.exists(), 'unfinished durable reservation is preserved')
    ledger = root / 'attempts.jsonl'
    lock = (root / 'series.lock').open('a')
    fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
    previous = [json.loads(line) for line in ledger.read_text().splitlines()] if ledger.exists() else []
    require(len(previous) == position - 1 and all(r['status'] == 'closed' for r in previous))
    require(sum(r['chargedSeconds'] for r in previous) + 2700 <= 8100)
    p = m['packets'][position - 1]
    INTERRUPT['attempt'] = p['name']
    credential_path = Path(m['provider']['credentialPath'])
    require(not credential_path.exists(), 'unexpected credential file is preserved')
    attempt = root / 'attempts' / p['name']
    require(not attempt.exists())
    docker = '/usr/bin/docker'
    baseline = subprocess.run([docker, 'ps', '--no-trunc', '--format', '{{.ID}}|{{.Image}}|{{.Names}}'],
                              capture_output=True, check=True, timeout=15).stdout.decode().splitlines()
    require(sorted(baseline) == m['hostBaselineRunning'], 'unapproved Docker workload changed the reserved host slot')
    busy = subprocess.run([docker, 'ps', '-q', '--filter', 'label=markitect.store'],
                          capture_output=True, check=True, timeout=15)
    require(not busy.stdout.strip(), 'Docker host is busy')
    start = process_started
    started = process_started_utc
    reservation = {'name': p['name'], 'scenario': p['scenario'], 'startedUtc': started.isoformat(),
                   'reservedAtUtc': dt.datetime.now(dt.timezone.utc).isoformat(),
                   'reservationOffsetSeconds': round(time.monotonic() - start, 3),
                   'status': 'running', 'chargedSeconds': 2700, 'manifestSha256': sha(manifest_path)}
    credential_created = False
    unsafe = False
    result = {'classification': 'technical-incomplete', 'name': p['name']}
    def remaining(bound):
        seconds = bound - (time.monotonic() - start)
        if seconds < 1:
            raise TimeoutError('finite attempt budget exhausted')
        return seconds
    def run(args, **kwargs):
        limit = min(remaining(2680), kwargs.pop('timeout', 30))
        check, capture = kwargs.pop('check', False), kwargs.pop('capture_output', False)
        payload = kwargs.pop('input', None)
        if capture:
            kwargs['stdout'] = kwargs['stderr'] = subprocess.PIPE
        if payload is not None:
            kwargs['stdin'] = subprocess.PIPE
        if kwargs.pop('text', False) and isinstance(payload, str):
            payload = payload.encode()
        with subprocess.Popen(args, start_new_session=True, **kwargs) as peer:
            try:
                stdout, stderr = peer.communicate(payload, timeout=limit)
            except BaseException:
                if peer.poll() is None:
                    os.killpg(peer.pid, signal.SIGTERM)
                    try:
                        peer.communicate(timeout=5)
                    except subprocess.TimeoutExpired:
                        os.killpg(peer.pid, signal.SIGKILL)
                        peer.communicate(timeout=1)
                raise
            completed = subprocess.CompletedProcess(args, peer.returncode, stdout, stderr)
            if check and peer.returncode:
                raise subprocess.CalledProcessError(peer.returncode, args)
            return completed
    def cleanup():
        nonlocal unsafe
        found = run([docker, 'ps', '-q', '--filter', 'label=markitect.store=' + p['store']],
                    capture_output=True, check=True).stdout.decode().split()
        if found:
            # Emergency cessation is retained as unsafe, never as successful Turn stop proof.
            unsafe = True
            run([docker, 'stop', '--time', '10', *found], capture_output=True, check=True, timeout=20)
        after = run([docker, 'ps', '-q', '--filter', 'label=markitect.store=' + p['store']],
                    capture_output=True, check=True).stdout.strip()
        require(not after, 'owned containers still running')
        return {'runningBeforeEmergencyStop': len(found), 'runningAfter': 0,
                'emergencyStopUsed': bool(found), 'volumes': 'retained; no foreign resource removed'}
    try:
        # This reservation remains until cleanup and both final records are
        # durable. It also blocks a next start if a later recovery write fails.
        with active.open('x') as f:
            f.write(json.dumps(reservation) + '\n')
            f.flush()
            os.fsync(f.fileno())
        sync_dir(root)
        with ledger.open('a') as f:
            f.write(json.dumps(reservation) + '\n')
            f.flush()
            os.fsync(f.fileno())
        sync_dir(root)
        attempt.mkdir(parents=True)
        (attempt / 'out').mkdir()
        credential_path.parent.mkdir(mode=0o700, exist_ok=True)
        source = Path(m['source'])
        require(run(['git', '-C', str(source), 'rev-parse', 'HEAD'], capture_output=True, check=True).stdout.decode().strip() == m['sourceCommit'])
        require(not run(['git', '-C', str(source), 'status', '--porcelain'], capture_output=True, check=True).stdout.strip())
        repo = attempt / 'repo'
        run(['git', 'clone', '--quiet', '--no-hardlinks', p['repo'], str(repo)], check=True, capture_output=True)
        run(['git', '-C', str(repo), 'remote', 'remove', 'origin'], check=True, capture_output=True)
        require(run(['git', '-C', str(repo), 'rev-parse', 'HEAD'], capture_output=True, check=True).stdout.decode().strip() == p['packet']['head'])
        shutil.copy2(Path(m['build']) / 'markitect', attempt / 'markitect')
        (attempt / 'host').mkdir()
        shutil.copy2(p['profile'], attempt / 'host/host.yaml')
        (attempt / 'binding.txt').write_text('runtime_image=' + m['images']['runtime'] + '\n')
        deadline = (started + dt.timedelta(seconds=m['ceilings']['providerDeadlineFromAttemptStartSeconds'])).strftime('%Y-%m-%dT%H:%M:%SZ')
        run([sys.executable, str(source / 'experiments/recursive-apply-demo/ra_runconfig.py'), str(attempt),
             'run', m['sourceCommit'], deadline, 'Apply the committed model.'], check=True, capture_output=True)
        config_path = attempt / 'run-config-run.json'
        config = json.loads(config_path.read_text())
        config['workDeadline'] = (started + dt.timedelta(seconds=m['ceilings']['workDeadlineFromAttemptStartSeconds'])).strftime('%Y-%m-%dT%H:%M:%SZ')
        config['repositoryIdentity'] = 'project-' + p['name'].lower()
        config['credentials'] = {'dedicated-codex-runtime': {
            'provider': 'codex-app-server', 'mode': 'dedicated-runtime-login', 'path': str(credential_path)}}
        write(config_path, config)
        write(attempt / 'effective-binding.json', {'profileSha256': sha(attempt / 'host/host.yaml'),
              'runConfigSha256': sha(config_path), 'packetHead': p['packet']['head'],
              'deadlineUtc': deadline, 'workDeadlineUtc': config['workDeadline'], 'verifyReserveSeconds': 720, 'cleanupReserveSeconds': 180, 'manifestSha256': sha(manifest_path)})
        # Nongenerative metadata is part of this charged attempt, never a hidden
        # inference preflight. Its script contains only the four read-only RPCs.
        readiness = run([docker, 'run', '--rm', '--init', '--name', p['store'] + '-metadata',
            '--label', 'markitect.store=' + p['store'], '--read-only', '--user', '1000:1000',
            '--cap-drop', 'ALL', '--security-opt', 'no-new-privileges', '--cpus', '1',
            '--memory', '536870912', '--pids-limit', '128',
            '--tmpfs', '/tmp:rw,exec,nosuid,nodev,mode=1777,size=67108864',
            '--mount', 'type=volume,src=mk-codex-runtime-auth,dst=/auth,readonly',
            '--mount', 'type=bind,src=' + str(source / 'experiments/apply-trials/ccf_metadata.py') + ',dst=/metadata.py,readonly',
            '--entrypoint', '/opt/markitect/bin/python', m['images']['agent'], '-B', '/metadata.py'],
            capture_output=True, check=True, timeout=75)
        metadata = json.loads(readiness.stdout)
        require(metadata['generationRequests'] == 0 and metadata['accessValiditySeconds'] >= 2700)
        require(metadata['appServerStopped'] and metadata['temporaryCredentialRemoved'])
        require(metadata['loginStatusExit'] == 0 and metadata['loginReportsChatGPT'])
        require(metadata['account']['type'] == 'chatgpt' and not metadata['hasApiKey'] and not metadata['refreshTokenDelivered'])
        require(any(item['model'] == 'gpt-6.1-sol' and any(effort['reasoningEffort'] == 'high'
                    for effort in item['supportedReasoningEfforts']) for item in metadata['selected']))
        quota = (metadata['quota']['rateLimitsByLimitId'] or {}).get('codex') or metadata['quota']['rateLimits']
        require(quota is not None and quota['primary'] is not None and quota['rateLimitReachedType'] is None)
        require(100 - quota['primary']['usedPercent'] >= m['provider']['minimumRemainingQuotaPercent'])
        require(not run([docker, 'ps', '-aq', '--filter', 'name=^/' + p['store'] + '-metadata$'], capture_output=True, check=True).stdout.strip(),
                'metadata container cessation/removal unconfirmed')
        write(attempt / 'readiness.json', {'manifestSha256': sha(manifest_path),
            'metadataScriptSha256': sha(source / 'experiments/apply-trials/ccf_metadata.py'),
            'observedMonotonicSeconds': round(time.monotonic() - start, 3), 'metadata': metadata})
        result['readinessSha256'] = sha(attempt / 'readiness.json')
        require(json.loads((attempt / 'readiness.json').read_text())['metadata'] == metadata)
        # Use only the existing dedicated volume reader; never desktop auth or an API key.
        spec = importlib.util.spec_from_file_location('dedicated_login', source / 'experiments/runtime-baseline/codex_login.py')
        login_module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(login_module)
        login_module.IMAGE = m['images']['agent']
        def auth_docker(*args, **kwargs):
            require(args[0] == 'run')
            return run([docker, 'run', '--name', p['store'] + '-read-login', '--network', 'none',
                        '--label', 'markitect.store=' + p['store'], *args[1:]], **kwargs)
        login_module.docker = auth_docker
        raw = login_module.session_login()
        login = json.loads(raw)
        require(login['auth_mode'] == 'chatgpt' and login.get('OPENAI_API_KEY') is None)
        require(login['tokens']['refresh_token'] == '')
        fd = os.open(credential_path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        credential_created = True
        with os.fdopen(fd, 'w') as f:
            f.write(raw)
        live_baseline = run([docker, 'ps', '--no-trunc', '--format', '{{.ID}}|{{.Image}}|{{.Names}}'], capture_output=True, check=True).stdout.decode().splitlines()
        require(sorted(live_baseline) == m['hostBaselineRunning'], 'Docker host changed before provider start')
        args = [str(attempt / 'markitect'), 'apply', '--repo', str(repo), '--model', '.markitect/model.yaml',
                '--dependency', '.markitect/dependencies/core/model.yaml',
                '--dependency', '.markitect/dependencies/software/model.yaml',
                '--runtime-profile', str(attempt / 'host/host.yaml'), '--run-config', str(config_path)]
        with (attempt / 'out/apply.json').open('wb') as stdout, (attempt / 'out/apply.stderr').open('wb') as stderr:
            done = run(['/usr/bin/timeout', '--signal=TERM', '--kill-after=20s', str(int(remaining(2490))), *args],
                       stdout=stdout, stderr=stderr, timeout=2540)
        result['applyExit'] = done.returncode
        unsafe = done.returncode in (124, 137)
        first_cleanup = cleanup()
        state = attempt / 'state-run'
        run_id = (state / 'current-run').read_text().strip()
        evidence = state / 'runs' / run_id
        printed = json.loads((attempt / 'out/apply.json').read_text())
        persisted = json.loads((evidence / 'report.json').read_text())
        result['fileEqualsStdout'] = printed == persisted
        result['runId'] = run_id
        result['usage'] = (printed.get('run') or {}).get('usage')
        result['billingCostUSD'] = None
        result['billingCostKnown'] = False
        turns = (printed.get('run') or {}).get('turns', [])
        if not all(t.get('stopped') is True for t in turns):
            unsafe = True
        require(printed == persisted, 'stdout and report.json differ')
        command = [docker, 'run', '--rm', '--init', '--name', p['store'] + '-oracle',
                   '--label', 'markitect.store=' + p['store'], '--network', 'none', '--read-only',
                   '--user', '1000:1000', '--cap-drop', 'ALL', '--security-opt', 'no-new-privileges',
                   '--cpus', '2', '--memory', '2147483648', '--pids-limit', '128',
                   '--tmpfs', '/tmp:rw,exec,nosuid,nodev,mode=1777,size=536870912',
                   '--mount', 'type=bind,src=' + str(source / 'experiments') + ',dst=/kit,readonly',
                   '--mount', 'type=bind,src=' + str(attempt) + ',dst=' + str(attempt) + ',readonly',
                   '--entrypoint', '/opt/markitect/bin/python', m['images']['oracle'], '-B',
                   '/kit/apply-trials/evaluate.py', p['scenario'], '--arm', 'direct', '--repo', str(repo),
                   '--report', str(attempt / 'out/apply.json'), '--events', str(evidence / 'events.jsonl')]
        with (attempt / 'out/evaluate.json').open('wb') as out, (attempt / 'out/evaluate.stderr').open('wb') as err:
            scored = run(command, stdout=out, stderr=err, timeout=min(60, remaining(2670)))
        require(scored.returncode == 0)
        verdict = json.loads((attempt / 'out/evaluate.json').read_text())
        result['oracleVerdict'], result['oracleCategory'] = verdict['verdict'], verdict['category']
        result['processVerdict'] = verdict['process']['verdict']
        verdicts = [verdict['verdict'], verdict['process']['verdict']]
        result['classification'] = ('nonconforming' if 'not_met' in verdicts else
                                    'correct' if all(v == 'met' for v in verdicts) else 'technical-incomplete')
        for phase, extra in (('model-file', []),):
            with (attempt / 'out' / (phase + '.json')).open('wb') as out:
                checked = run([str(attempt / 'markitect'), 'check', '--repo', str(repo),
                               '--dependency', '.markitect/dependencies/core/model.yaml',
                               '--dependency', '.markitect/dependencies/software/model.yaml', *extra], stdout=out, timeout=20)
            result[phase + 'Exit'] = checked.returncode
        if result['model-fileExit'] == 1:
            result['classification'] = 'nonconforming'
        elif result['model-fileExit'] != 0 or unsafe:
            result['classification'] = 'technical-incomplete'
        result['initialCleanup'] = first_cleanup
    except Exception as error:
        result['errorType'] = type(error).__name__  # never print credential payloads or captured process output
        if isinstance(error, (subprocess.TimeoutExpired, InterruptedError)):
            unsafe = True
    finally:
        INTERRUPT['finalizing'] = True
        try:
            result['finalCleanup'] = cleanup()
        except Exception as error:
            unsafe = True
            result['cleanupErrorType'] = type(error).__name__
        try:
            if credential_created and credential_path.exists():
                credential_path.unlink()
        except Exception as error:
            unsafe = True
            result['credentialCleanupErrorType'] = type(error).__name__
        unsafe = unsafe or INTERRUPT['seen']
        result['unsafeOrUnknownCleanup'] = unsafe
        result['recoveryRequired'] = unsafe or INTERRUPT['markerWriteFailed']
        if unsafe:
            result['classification'] = 'technical-incomplete'
        result['elapsedSeconds'] = round(time.monotonic() - start, 3)
        result['endedUtc'] = dt.datetime.now(dt.timezone.utc).isoformat()
        try:
            write(attempt / 'result.json', result)
        except Exception as error:
            unsafe = True
            result['resultWriteErrorType'] = type(error).__name__
            result['unsafeOrUnknownCleanup'] = True
            result['classification'] = 'technical-incomplete'
        records = [*previous, {**reservation, 'status': 'unsafe' if unsafe else 'closed',
                              'chargedSeconds': max(1, int(time.monotonic() - start + 0.999)),
                              'result': result}]
        replacement = ledger.with_suffix('.new')
        try:
            with replacement.open('w') as f:
                f.write(''.join(json.dumps(r) + '\n' for r in records))
                f.flush()
                os.fsync(f.fileno())
            os.replace(replacement, ledger)
            sync_dir(root)
        except Exception as error:
            unsafe = True
            result['ledgerWriteErrorType'] = type(error).__name__
            result['recoveryRequired'] = True  # original charged running line remains fail-closed
            result['unsafeOrUnknownCleanup'] = True
            result['classification'] = 'technical-incomplete'
        if INTERRUPT['seen']:
            unsafe = True
            result['unsafeOrUnknownCleanup'] = True
            result['classification'] = 'technical-incomplete'
        if not unsafe:
            blocked = signal.pthread_sigmask(signal.SIG_BLOCK, {signal.SIGTERM, signal.SIGINT, signal.SIGALRM})
            try:
                if not INTERRUPT['seen']:
                    active.unlink()
                    # Removal need not survive power loss: a reappearing ACTIVE
                    # conservatively blocks starts; the charged ledger is durable.
                    INTERRUPT['committed'] = True
            except OSError as error:
                unsafe = True
                result['recoveryRequired'] = True
                result['reservationCloseErrorType'] = type(error).__name__
            finally:
                signal.pthread_sigmask(signal.SIG_SETMASK, blocked)
        print(json.dumps(result))
    return 4 if unsafe else 0

if __name__ == '__main__':
    # Bound the entire process, including setup and finalization, to 2,700 s.
    # The 1,740 s work cutoff and 2,460 s provider deadline leaves positive writer-stop/cleanup reserve.
    if sys.argv[1:2] != ['--bounded-child']:
        os.execv('/usr/bin/timeout', ['timeout', '--signal=TERM', '--kill-after=15s', '2670s',
                 sys.executable, '-B', __file__, '--bounded-child', *sys.argv[1:]])
    del sys.argv[1]
    def interrupt(signum, frame):
        if INTERRUPT['committed']:
            return  # cleanup and durable terminal evidence already completed
        first = not INTERRUPT['seen']
        INTERRUPT['seen'] = True
        if INTERRUPT['stopPath'] is not None:
            try:
                # This later recovery record overrides any earlier closed row
                # or result snapshot; it is never removed by this runner.
                with INTERRUPT['stopPath'].open('x') as marker:
                    json.dump({'status': 'unsafe', 'recoveryRequired': True,
                               'attempt': INTERRUPT.get('attempt'),
                               'reason': 'signal or overall deadline',
                               'atUtc': dt.datetime.now(dt.timezone.utc).isoformat(),
                               'overrides': 'earlier result.json and ledger row for this attempt'}, marker)
                    marker.write('\n')
                    marker.flush()
                    os.fsync(marker.fileno())
                sync_dir(INTERRUPT['stopPath'].parent)
            except OSError:
                INTERRUPT['markerWriteFailed'] = True
        if first and not INTERRUPT['finalizing']:
            raise InterruptedError('bounded runner interrupted')
    signal.signal(signal.SIGTERM, interrupt)
    signal.signal(signal.SIGINT, interrupt)
    process_started = time.monotonic()
    process_started_utc = dt.datetime.now(dt.timezone.utc)
    def deadline(signum, frame):
        # The child also enforces the cap; --bounded-child cannot waive it.
        signal.signal(signal.SIGALRM, lambda signum, frame: os._exit(124))
        signal.setitimer(signal.ITIMER_REAL, max(0.001, 2700 - (time.monotonic() - process_started)))
        interrupt(signum, frame)
    signal.signal(signal.SIGALRM, deadline)
    signal.setitimer(signal.ITIMER_REAL, 2670)
    try:
        sys.exit(main())
    except Exception as error:
        print(json.dumps({'status': 'refused-before-attempt', 'errorType': type(error).__name__}))
        sys.exit(3)
