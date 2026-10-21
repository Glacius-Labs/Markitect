"""One source-released NIGHT-20261021 attempt. No automatic retry or model fallback.

Usage: python3 -B night_run.py GROUP_MANIFEST COORDINATOR_GATE GROUP_POSITION
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

HOST_CLOCK = '/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe'
ABSOLUTE_START_CUTOFF = dt.datetime(2026, 10, 22, 4, 0, tzinfo=dt.timezone.utc)
ABSOLUTE_ATTEMPT_STOP = dt.datetime(2026, 10, 22, 4, 45, tzinfo=dt.timezone.utc)

def host_clock():
    """Independent Windows wall/Stopwatch sample; no credentials or generation."""
    command = ('[Console]::WriteLine(([ordered]@{utc=[DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds();'
               'ticks=[Diagnostics.Stopwatch]::GetTimestamp();frequency=[Diagnostics.Stopwatch]::Frequency}'
               '|ConvertTo-Json -Compress))')
    done = subprocess.run([HOST_CLOCK, '-NoLogo', '-NoProfile', '-NonInteractive', '-Command', command],
                          capture_output=True, check=True, timeout=5)
    value = json.loads(done.stdout.decode('utf-8-sig'))
    require(value['frequency'] > 0)
    return value

def host_utc(sample):
    return dt.datetime.fromtimestamp(sample['utc'] / 1000, dt.timezone.utc)

def host_elapsed(sample=None):
    sample = sample or host_clock()
    require(sample['frequency'] == process_started_host['frequency'])
    return (sample['ticks'] - process_started_host['ticks']) / sample['frequency']


def validate_profile(path, packet, manifest):
    # JSON is a Runtime YAML subset. Bind the actual input without inventing
    # another permissive YAML decoder or trusting manifest-side declarations.
    profile = json.loads(Path(path).read_text(encoding='utf-8'))
    require(profile['apiVersion'] == 'markitect.runtime/v1' and profile['agentProfiles'])
    for agent in profile['agentProfiles'].values():
        require(set(agent) == {'codexAppServer'}, 'every Agent role must use Codex App Server')
        settings = agent['codexAppServer']
        session = settings['session']
        require(settings['imageID'] == manifest['images']['agent']
                and settings['engineExecutable'] == '/usr/bin/docker'
                and settings['credentialRef'] == 'dedicated-codex-runtime', 'actual provider image/credential binding differs')
        require(session['model'] == manifest['provider']['model'] and session['effort'] == manifest['provider']['effort'],
                'actual session model/effort differs from the authorized manifest')
        require(session['store'] == packet['store'], 'actual Agent store differs from owned cleanup scope')
    require(profile['checkProfiles'])
    for check in profile['checkProfiles'].values():
        require(set(check) == {'container'} and check['container']['imageID'] == manifest['images']['oracle']
                and check['container']['engineExecutable'] == '/usr/bin/docker',
                'actual check profile image differs from frozen binding')

def validate(manifest_path, gate_path, files=True):
    m = json.loads(Path(manifest_path).read_text())
    g = json.loads(Path(gate_path).read_text())
    require(m['series'] == 'NIGHT-20261021' and m['group'])
    require(m['status'] == 'frozen' and g['startAuthorized'] is True)
    require(g['manifestSha256'] == sha(manifest_path) and g['sourceCommit'] == m['sourceCommit'])
    require(g['cReady'] is True and g['dockerSlotApproved'] is True and g['codexQuotaReserveApproved'] is True)
    require(g['coordinatorMessage'] and g['cEvidenceSha256'] and g['independentReviewEvidence'])
    require(g['authorizedAttempts'] == [p['name'] for p in m['packets']])
    require(m['absoluteStartCutoffUtc'] == '2026-10-22T04:00:00Z')
    require(m['absoluteAttemptStopUtc'] == '2026-10-22T04:45:00Z')
    require(m['writersStoppedByUtc'] == '2026-10-22T04:55:00Z')
    require(m['provider']['name'] == 'codex-app-server' and m['provider']['model'] == 'gpt-6.1-sol')
    require(m['provider']['effort'] == 'high' and m['provider']['credentialMode'] == 'dedicated-runtime-login')
    require(m['provider']['credentialSourceVolume'] == 'mk-codex-runtime-auth')
    require(m['provider']['minimumRemainingQuotaPercent'] == 25)
    require(1 <= len(m['packets']) <= 6)
    require(all(p['scenario'] in ('NIGHT-VN', 'NIGHT-VS', 'NIGHT-P', 'NIGHT-R') for p in m['packets']))
    require(all(p['mode'] in ('verify', 'apply') for p in m['packets']))
    require(m['ceilings']['starts'] == 6 and m['ceilings']['attemptSeconds'] == 2700)
    require(m['ceilings']['summedSeconds'] == 16200 and m['ceilings']['retries'] == 0)
    require(m['ceilings']['workDeadlineFromAttemptStartSeconds'] == 1740)
    require(m['ceilings']['providerDeadlineFromAttemptStartSeconds'] == 2460)
    require(m['ceilings']['outerProcessDeadlineFromAttemptStartSeconds'] == 2490)
    require(m['ceilings']['providerDeadlineFromAttemptStartSeconds'] - m['ceilings']['workDeadlineFromAttemptStartSeconds'] >= 720)
    require(m['ceilings']['attemptSeconds'] - m['ceilings']['outerProcessDeadlineFromAttemptStartSeconds'] >= 180)
    require(m['ceilings']['writerStopCleanupReserveSeconds'] >= 180)
    require(m['ceilings']['wrapperSoftSeconds'] == 2670 and m['ceilings']['wrapperKillGraceSeconds'] == 15)
    require(m['ceilings']['wrapperSoftSeconds'] - m['ceilings']['outerProcessDeadlineFromAttemptStartSeconds'] >= 180)
    require(m['ceilings']['wrapperSoftSeconds'] + m['ceilings']['wrapperKillGraceSeconds'] < m['ceilings']['attemptSeconds'])
    if files:
        require(sha(Path(m['build']) / 'markitect') == m['cliSha256'])
        require(sha(Path(m['build']) / 'verify-harness') == m['verifyHarnessSha256'])
        require(sha(Path(m['nightRoot']) / 'authority.json') == m['authoritySha256'])
        for p in m['packets']:
            require(sha(p['profile']) == p['profileSha256'])
            validate_profile(p['profile'], p, m)
            for relative, digest in p['packet']['files'].items():
                require(sha(Path(p['repo']) / relative) == digest)
        for relative, digest in m['sourceFiles'].items():
            require(sha(Path(m['source']) / relative) == digest)
    require(m['limits'] == {'turnSeconds': 300, 'turns': 240, 'passes': 8, 'probesPerTurn': 12})
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


def recovery(root, name, reason):
    """A durable override, never a claim that earlier result snapshots stand."""
    marker = Path(root) / 'STOP-SERIES'
    try:
        with marker.open('x') as stream:
            json.dump({'status': 'unsafe', 'attempt': name, 'recoveryRequired': True,
                       'reason': reason, 'overrides': 'earlier result.json and ledger row for this attempt'}, stream)
            stream.write('\n')
            stream.flush()
            os.fsync(stream.fileno())
        sync_dir(marker.parent)
    except FileExistsError:
        pass
    except OSError:
        # A persistence failure must never prevent forced cessation. The
        # unfinished ACTIVE/full reservation independently blocks new starts.
        return False
    return True


def process_identity(pid):
    """Kernel start identity prevents killing a reused PID or foreign group."""
    try:
        fields = Path('/proc/%d/stat' % pid).read_text().rsplit(')', 1)[1].split()
        return {'pid': pid, 'group': int(fields[2]), 'start': fields[19]}
    except (FileNotFoundError, ProcessLookupError):
        return None


def track_process(path, pid):
    identity = process_identity(pid)
    require(identity is not None and identity['group'] == pid, 'owned process group not established')
    with Path(path).open('a') as stream:
        fcntl.flock(stream, fcntl.LOCK_EX)
        stream.write(json.dumps(identity) + '\n')
        stream.flush()
        os.fsync(stream.fileno())


def signal_tracked(path, signum):
    with Path(path).open() as stream:
        fcntl.flock(stream, fcntl.LOCK_SH)
        lines = stream.read().splitlines()
    invalid = False
    for line in lines:
        try:
            identity = json.loads(line)
            require(set(identity) == {'pid', 'group', 'start'} and type(identity['pid']) is int
                    and type(identity['group']) is int and type(identity['start']) is str
                    and identity['pid'] > 1 and identity['pid'] == identity['group'])
        except (ValueError, TypeError, KeyError):
            invalid = True
            continue
        if process_identity(identity['pid']) == identity:
            try:
                os.killpg(identity['group'], signum)
            except ProcessLookupError:
                pass
    if invalid:
        raise ValueError('invalid process registry record; valid owned groups were still signaled')


def force_supervised_stop(child, registry, root, packet, reason, stop_sample=None):
    """Cessation continues even if recovery or process-registry writes failed."""
    errors = []
    marker = recovery(root, packet['name'], reason + '; cleanup not yet proved')
    def cease(signum):
        try:
            signal_tracked(registry, signum)
        except Exception as error:
            errors.append(type(error).__name__)
        # Popen still owns this unreaped identity even if its registry failed.
        if child.poll() is None:
            try:
                os.killpg(child.pid, signum)
            except ProcessLookupError:
                pass
    cease(signal.SIGTERM)
    try:
        # Reuse the supervisor's last clock sample. Clock/registry failures
        # receive no grace: do not block on another uncertain clock query.
        while stop_sample is not None and child.poll() is None:
            sample = host_clock()
            require(sample['frequency'] == stop_sample['frequency'] and sample['ticks'] >= stop_sample['ticks'])
            if (sample['ticks'] - stop_sample['ticks']) / sample['frequency'] >= 15:
                break
            time.sleep(0.1)
    except Exception as error:
        errors.append(type(error).__name__)  # clock uncertainty escalates now
    finally:
        cease(signal.SIGKILL)
    # Stop only the exact reserved store; never credit emergency cessation as
    # normal Turn stop. Retain ACTIVE/full reservation and every volume.
    remaining = None
    docker = '/usr/bin/docker'
    try:
        found = subprocess.run([docker, 'ps', '-q', '--filter', 'label=markitect.store=' + packet['store']],
                               capture_output=True, check=True, timeout=2).stdout.decode().split()
        if found:
            subprocess.run([docker, 'kill', *found], capture_output=True, check=True, timeout=3)
        remaining = subprocess.run([docker, 'ps', '-q', '--filter', 'label=markitect.store=' + packet['store']],
                                   capture_output=True, check=True, timeout=2).stdout.decode().split()
    except Exception as error:
        errors.append(type(error).__name__)
    try:
        child.wait(timeout=1)
    except subprocess.TimeoutExpired:
        errors.append('runner-stop-unknown')
    try:
        write(root / ('forced-cleanup-' + packet['name'] + '.json'),
              {'forced': True, 'runningAfter': remaining, 'normalTurnStopsProved': False,
               'recoveryMarkerDurable': marker, 'errors': errors})
    except OSError:
        pass  # unresolved ACTIVE/full charge remains authoritative
    return 4


def supervise(manifest_path, gate_path, position):
    """Independent process supervision for the observed guest-clock drift.

    Requires responsive WSL scheduling, guest timers and Windows interop. It does not promise
    enforcement through suspension of the entire subsystem or host bridge.
    """
    started = host_clock()
    m = validate(manifest_path, gate_path, files=False)
    require(1 <= int(position) <= len(m['packets']))
    packet = m['packets'][int(position) - 1]
    root = Path(m['nightRoot'])
    # Independent durable ownership survives a child closing ACTIVE while the
    # supervisor detects a deadline or loses storage. Forced paths never remove
    # this pre-existing blocker, even if a new STOP write is impossible.
    supervisor_active = root / 'ACTIVE-SUPERVISOR'
    with supervisor_active.open('x') as stream:
        json.dump({'attempt': packet['name'], 'startedHostClock': started}, stream)
        stream.flush()
        os.fsync(stream.fileno())
    sync_dir(root)
    registry = root / ('process-groups-' + packet['name'] + '.jsonl')
    with registry.open('x') as stream:
        stream.flush()
        os.fsync(stream.fileno())
    sync_dir(root)
    environment = dict(os.environ, MARKITECT_NIGHT_PROCESS_REGISTRY=str(registry),
                       MARKITECT_NIGHT_STARTED_HOST=json.dumps(started))
    child = subprocess.Popen([sys.executable, '-B', __file__, '--bounded-child',
                              manifest_path, gate_path, position], start_new_session=True, env=environment)
    try:
        track_process(registry, child.pid)
        while child.poll() is None:
            sample = host_clock()
            require(sample['frequency'] == started['frequency'] and sample['ticks'] >= started['ticks'],
                    'independent host clock changed identity or moved backward')
            elapsed = (sample['ticks'] - started['ticks']) / sample['frequency']
            if elapsed >= 2670 or host_utc(sample) >= ABSOLUTE_ATTEMPT_STOP - dt.timedelta(seconds=30):
                return force_supervised_stop(child, registry, root, packet, 'independent host-clock supervisor deadline', sample)
            time.sleep(0.1)
    except BaseException as error:
        return force_supervised_stop(child, registry, root, packet,
                                     'supervisor failure ' + type(error).__name__)
    if child.returncode == 0 and not (root / 'ACTIVE-ATTEMPT').exists() and not (root / 'STOP-SERIES').exists():
        records = [json.loads(line) for line in (root / 'attempts.jsonl').read_text().splitlines()]
        require(records and records[-1]['name'] == packet['name'] and records[-1]['status'] == 'closed')
        supervisor_active.unlink()
        sync_dir(root)
    return child.returncode

def main():
    manifest_path, gate_path, position = sys.argv[1:]
    # Guard precedes Docker, credentials, ledger reservation and the binary.
    m = validate(manifest_path, gate_path, files=False)
    position = int(position)
    require(1 <= position <= len(m['packets']))
    root = Path(m['nightRoot']).resolve()
    INTERRUPT['stopPath'] = root / 'STOP-SERIES'
    require(not INTERRUPT['stopPath'].exists(), 'series stop marker is preserved')
    active = root / 'ACTIVE-ATTEMPT'
    require(not active.exists(), 'unfinished durable reservation is preserved')
    ledger = root / 'attempts.jsonl'
    lock = (root / 'series.lock').open('a')
    fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
    previous = [json.loads(line) for line in ledger.read_text().splitlines()] if ledger.exists() else []
    p = m['packets'][position - 1]
    require(json.loads((root / 'ACTIVE-SUPERVISOR').read_text())['attempt'] == p['name'],
            'independent supervisor reservation does not match this attempt')
    require(len(previous) == p['startNumber'] - 1 and len(previous) < 6 and all(r['status'] == 'closed' for r in previous))
    require(sum(r['chargedSeconds'] for r in previous) + 2700 <= 16200)
    require(host_utc(host_clock()) < ABSOLUTE_START_CUTOFF, 'absolute no-new-start cutoff reached')
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
    reservation = {'name': p['name'], 'scenario': p['scenario'], 'mode': p['mode'], 'group': m['group'], 'startedUtc': started.isoformat(),
                   'startedHostClock': process_started_host,
                   'reservedAtUtc': dt.datetime.now(dt.timezone.utc).isoformat(),
                   'reservationOffsetSeconds': round(time.monotonic() - start, 3),
                   'status': 'running', 'chargedSeconds': 2700, 'manifestSha256': sha(manifest_path)}
    credential_created = False
    unsafe = False
    result = {'classification': 'technical-incomplete', 'name': p['name']}
    def remaining(bound):
        host = host_clock()
        seconds = min(bound - (time.monotonic() - start), bound - host_elapsed(host),
                      (ABSOLUTE_ATTEMPT_STOP - host_utc(host)).total_seconds())
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
                track_process(os.environ['MARKITECT_NIGHT_PROCESS_REGISTRY'], peer.pid)
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
        # Hash/profile/packet failures after an authorized dispatch consume the
        # durable start too. Invalid/closed releases still refuse before effect.
        attempt.mkdir(parents=True)
        (attempt / 'out').mkdir()
        validate(manifest_path, gate_path)
        credential_path.parent.mkdir(mode=0o700, exist_ok=True)
        source = Path(m['source'])
        require(run(['git', '-C', str(source), 'rev-parse', 'HEAD'], capture_output=True, check=True).stdout.decode().strip() == m['sourceCommit'])
        require(not run(['git', '-C', str(source), 'status', '--porcelain'], capture_output=True, check=True).stdout.strip())
        repo = attempt / 'repo'
        run(['git', 'clone', '--quiet', '--no-hardlinks', p['repo'], str(repo)], check=True, capture_output=True)
        run(['git', '-C', str(repo), 'remote', 'remove', 'origin'], check=True, capture_output=True)
        require(run(['git', '-C', str(repo), 'rev-parse', 'HEAD'], capture_output=True, check=True).stdout.decode().strip() == p['packet']['head'])
        shutil.copy2(Path(m['build']) / 'markitect', attempt / 'markitect')
        if p['mode'] == 'verify':
            shutil.copy2(Path(m['build']) / 'verify-harness', attempt / 'verify-harness')
        (attempt / 'host').mkdir()
        shutil.copy2(p['profile'], attempt / 'host/host.yaml')
        require(sha(attempt / 'host/host.yaml') == p['profileSha256'], 'copied actual Runtime profile differs')
        validate_profile(attempt / 'host/host.yaml', p, m)
        (attempt / 'binding.txt').write_text('runtime_image=' + m['images']['runtime'] + '\n')
        deadline = (started + dt.timedelta(seconds=m['ceilings']['providerDeadlineFromAttemptStartSeconds'])).strftime('%Y-%m-%dT%H:%M:%SZ')
        run([sys.executable, str(source / 'experiments/recursive-apply-demo/ra_runconfig.py'), str(attempt),
             'run', m['sourceCommit'], deadline, 'Apply the committed model.'], check=True, capture_output=True)
        config_path = attempt / 'run-config-run.json'
        config = json.loads(config_path.read_text())
        config['workDeadline'] = (started + dt.timedelta(seconds=m['ceilings']['workDeadlineFromAttemptStartSeconds'])).strftime('%Y-%m-%dT%H:%M:%SZ')
        config['repositoryIdentity'] = 'project-' + p['name'].lower()
        # The shared helper's legacy YAML line scan cannot read JSON stores.
        # Bind the actual workspace helper to the same owned cleanup namespace.
        config['workspaceHelper']['labels'] = {'markitect.store': p['store']}
        require(config['workspaceHelper']['image'] == m['images']['runtime']
                and config['engineExecutable'] == '/usr/bin/docker', 'actual workspace Runtime binding differs')
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
        require(not quota.get('spendControlReached', False))
        require(all(100 - window['usedPercent'] >= m['provider']['minimumRemainingQuotaPercent']
                    for window in (quota.get('primary'), quota.get('secondary')) if window is not None))
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
        environment = os.environ.copy()
        if p['mode'] == 'verify':
            environment['MARKITECT_VERIFY_EVALUATION'] = 'run-verify-evaluation'
            environment['MARKITECT_VERIFY_EVALUATION_ARGS'] = json.dumps(args[1:])
            args = [str(attempt / 'verify-harness'), '-test.run=^TestVerifyEvaluationHarness$', '-test.v', '-test.timeout=42m']
        raw_path = attempt / 'out/provider-output.txt'
        require(host_utc(host_clock()) < ABSOLUTE_START_CUTOFF, 'absolute cutoff reached before productive dispatch')
        with raw_path.open('wb') as stdout, (attempt / 'out/apply.stderr').open('wb') as stderr:
            done = run(['/usr/bin/timeout', '--signal=TERM', '--kill-after=20s', str(int(remaining(2490))), *args],
                       stdout=stdout, stderr=stderr, timeout=2540, env=environment)
        result['applyExit'] = done.returncode
        unsafe = done.returncode in (124, 137)
        first_cleanup = cleanup()
        state = attempt / 'state-run'
        run_id = (state / 'current-run').read_text().strip()
        evidence = state / 'runs' / run_id
        # The existing real Verify harness prints Go test framing around its
        # normal product JSON report. Use the existing external report parser.
        spec = importlib.util.spec_from_file_location('night_evaluate', source / 'experiments/apply-trials/evaluate.py')
        evaluate_module = importlib.util.module_from_spec(spec)
        sys.path.insert(0, str(source / 'experiments/apply-trials'))
        spec.loader.exec_module(evaluate_module)
        printed = evaluate_module.load_report(raw_path)
        write(attempt / 'out/apply.json', printed)
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
            host = host_clock()
            result['endedHostClock'] = host
            result['hostElapsedSeconds'] = round(host_elapsed(host), 3)
        except Exception:
            unsafe = True
            result['hostClockUnknown'] = True
            result['unsafeOrUnknownCleanup'] = True
            result['recoveryRequired'] = True
            result['classification'] = 'technical-incomplete'
        try:
            write(attempt / 'result.json', result)
        except Exception as error:
            unsafe = True
            result['resultWriteErrorType'] = type(error).__name__
            result['unsafeOrUnknownCleanup'] = True
            result['recoveryRequired'] = True
            result['classification'] = 'technical-incomplete'
            recovery(root, p['name'], 'result persistence failed; earlier snapshots do not establish completion')
        records = [*previous, {**reservation, 'status': 'unsafe' if unsafe else 'closed',
                              # Charge the full reservation, never a faster guest
                              # clock or an underestimate of final persistence.
                              # Actual observed durations remain separate above.
                              'chargedSeconds': 2700,
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
            recovery(root, p['name'], 'ledger persistence failed; earlier snapshots do not establish completion')
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
                result['unsafeOrUnknownCleanup'] = True
                result['classification'] = 'technical-incomplete'
                result['reservationCloseErrorType'] = type(error).__name__
                recovery(root, p['name'], 'ACTIVE close failed; earlier closed snapshots require recovery')
            finally:
                signal.pthread_sigmask(signal.SIG_SETMASK, blocked)
        print(json.dumps(result))
    return 4 if unsafe else 0

if __name__ == '__main__':
    # Bound the entire process, including setup and finalization, to 2,700 s.
    # The 1,740 s work cutoff and 2,460 s provider deadline leaves positive writer-stop/cleanup reserve.
    if sys.argv[1:2] != ['--bounded-child']:
        sys.exit(supervise(*sys.argv[1:]))
    del sys.argv[1]
    require(os.environ.get('MARKITECT_NIGHT_PROCESS_REGISTRY'), 'independent supervisor is required')
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
    process_started_host = json.loads(os.environ['MARKITECT_NIGHT_STARTED_HOST'])
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
