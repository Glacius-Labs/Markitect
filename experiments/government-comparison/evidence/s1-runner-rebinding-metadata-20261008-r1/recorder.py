"""One finite version/schema metadata recording. No import-time execution or raw logs."""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import threading
import time

ROOT = Path(__file__).resolve().parent
REPO = ROOT.parents[3]
AUTH = json.loads((ROOT / 'authorization.json').read_bytes())
G = AUTH['grant']
OUT = Path(G['externalDirectory'])
PREP = json.loads((ROOT / 'preparation.json').read_bytes())
PROCESS = REPO / 'experiments/government-comparison/runtime/process.py'


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def require(ok, reason):
    if not ok:
        raise ValueError(reason)


def save(name, value):
    with (OUT / name).open('x', encoding='utf-8') as f:
        json.dump(value, f, indent=2); f.write('\n'); f.flush(); os.fsync(f.fileno())


def authority():
    live = json.loads(Path(AUTH['sourcePath']).read_bytes())
    actual = next(t for t in live['threads'] if t['name'] == 'Scientist')['evidence']['runnerRebindingMetadataGrant']
    require(actual == G and live['fullSuiteSlot'] == AUTH['slot'], 'live-authority-or-slot-changed')


def cwd_guard():
    p = Path(G['cwd']); entries = list(p.rglob('*'))
    require(not p.is_symlink() and not p.is_junction() and all(x.is_file() and not x.is_symlink() and not x.is_junction() for x in entries), 'public-cwd-entry-changed')
    require({x.relative_to(p).as_posix(): sha(x) for x in entries} == PREP['expectedFiles'], 'public-cwd-bytes-changed')


def tree():
    p = OUT / 'schemas'; count = size = 0
    if not p.exists():
        return {'files': 0, 'bytes': 0}
    require(p.is_dir() and not p.is_symlink() and not p.is_junction(), 'schema-root-invalid')
    for x in p.rglob('*'):
        require(not x.is_symlink() and not x.is_junction(), 'schema-link-found')
        if x.is_file():
            require(x.suffix == '.json', 'unexpected-schema-file-type')
            count += 1; size += x.stat().st_size
        else:
            require(x.is_dir(), 'unexpected-schema-entry')
    return {'files': count, 'bytes': size}


def source_guard():
    b = json.loads((ROOT / 'binding.json').read_bytes())
    for rel, digest in b['sourceFiles'].items():
        require(sha(REPO / rel) == digest, 'source-pin-changed')
    require(sha(sys.executable) == b['interpreterSha256'], 'interpreter-pin-changed')
    require((OUT / 'preparation.json').read_bytes() == (ROOT / 'preparation.json').read_bytes(), 'initial-state-binding-changed')
    return b


def call(index, overall):
    authority(); cwd_guard(); source_guard()
    require(sha(G['executable']) == G['executableSha256'], 'binary-prehash-mismatch')
    require(time.monotonic() < overall + 40, 'overall-deadline-before-invocation')
    expected = {'preparation.json', 'execution-once.json', 'job-assigned.json'}
    if index == 1: expected |= {'invocation-1.json', 'result-1.json'}
    require({x.name for x in OUT.iterdir()} == expected, 'unexpected-output-root-content')
    if index == 1:
        require(not (OUT / 'schemas').exists(), 'unexpected-schema-output-before-start')
    stage = time.monotonic(); seconds = G['versionSeconds'] if index == 0 else G['schemaSeconds']
    save(f'invocation-{index+1}.json', {'key': G['key'], 'index': index+1, 'argv': G['argv'][index], 'beforeSha256': G['executableSha256'], 'recordedAtUnix': time.time(), 'startAttemptReserved': True, 'metadataOnly': True})
    data = bytearray(); counts = {'stdout': 0, 'stderr': 0}; hit = threading.Event(); lock = threading.Lock(); native = None; code = None; reason = None; threads = []; max_tree = {'files': 0, 'bytes': 0}; observed_version = None; after = None

    def pump(stream, kind):
        while True:
            raw = stream.read1(1024)
            if not raw:
                return
            with lock:
                counts[kind] += len(raw)
                if kind == 'stderr' or counts[kind] > G['maxStdoutBytes'] or (index == 0 and counts['stdout'] > 256):
                    hit.set()
                if kind == 'stdout' and index == 0:
                    data.extend(raw[:max(0, 256 - len(data))])
            # Raw stderr is immediately discarded; any observed byte is terminal.
            if kind == 'stderr' or counts[kind] > G['maxStdoutBytes'] or (index == 0 and counts['stdout'] > 256):
                return

    try:
        native = subprocess.Popen(G['argv'][index], cwd=G['cwd'], stdin=subprocess.DEVNULL, stdout=subprocess.PIPE, stderr=subprocess.PIPE, shell=False)
        for stream, kind in [(native.stdout, 'stdout'), (native.stderr, 'stderr')]:
            t = threading.Thread(target=pump, args=(stream, kind), daemon=True); threads.append(t); t.start()
        deadline = min(stage + seconds, overall + 40)
        while True:
            current = tree(); max_tree = {k: max(max_tree[k], current[k]) for k in current}
            if hit.is_set(): reason = 'output-stop-signal'; break
            if current['files'] > G['maxSchemaFiles'] or current['bytes'] > G['maxSchemaBytes']: reason = 'schema-output-limit'; break
            if time.monotonic() >= deadline: reason = 'metadata-deadline'; break
            if native.poll() is not None: break
            time.sleep(.01)
        cleanup = time.monotonic(); cleanup_deadline = min(cleanup + 5, overall + 45)
        if reason and native.poll() is None:
            native.kill()
        code = native.wait(timeout=max(.001, cleanup_deadline-time.monotonic()))
        for t in threads:
            t.join(timeout=max(0, min(1, cleanup_deadline-time.monotonic())))
        require(not any(t.is_alive() for t in threads), 'pipe-drain-incomplete')
        if counts['stderr']: reason = 'nonempty-stderr'
        elif counts['stdout'] > G['maxStdoutBytes']: reason = 'stdout-limit'
        elif index == 0 and counts['stdout'] > 256: reason = 'version-line-limit'
        elif code != 0: reason = reason or 'nonzero-exit'
        final_tree = tree(); max_tree = {k: max(max_tree[k], final_tree[k]) for k in final_tree}
        if final_tree['files'] > G['maxSchemaFiles'] or final_tree['bytes'] > G['maxSchemaBytes']: reason = reason or 'schema-output-limit'
        if index == 0 and final_tree['files']: reason = reason or 'unexpected-version-output-tree'
        if not reason and index == 0:
            match = re.fullmatch(rb'codex-cli ([0-9]{1,4}\.[0-9]{1,4}\.[0-9]{1,4}(?:[-+][A-Za-z0-9.-]{1,64})?)\r?\n?', bytes(data))
            if match: observed_version = 'codex-cli ' + match[1].decode('ascii')
            else: reason = 'invalid-public-version-line'
        if not reason and index == 1 and not final_tree['files']: reason = 'schema-output-missing'
        after = sha(G['executable'])
        if after != G['executableSha256']: reason = reason or 'binary-posthash-mismatch'
        cwd_guard()
    except ValueError as exc:
        reason = str(exc) if re.fullmatch('[a-z-]{1,100}', str(exc)) else 'invalid-metadata-state'
        if 'cleanup' not in locals(): cleanup = time.monotonic(); cleanup_deadline = min(cleanup+5, overall+45)
    except BaseException:
        reason = 'local-metadata-recorder-failure'
        if 'cleanup' not in locals(): cleanup = time.monotonic(); cleanup_deadline = min(cleanup+5, overall+45)
    finally:
        data.clear()
        if native and native.poll() is None:
            native.kill()
            try: code = native.wait(timeout=max(.001, cleanup_deadline-time.monotonic()))
            except BaseException: reason = 'native-cleanup-incomplete'
        result = {'key': G['key'], 'index': index+1, 'argv': G['argv'][index], 'nativeProcessStarted': native is not None, 'nativePid': native.pid if native else None, 'nativeReturnCode': code, 'stopReason': reason, 'beforeSha256': G['executableSha256'], 'afterSha256': after, 'versionLine': observed_version if not reason else None, 'wallSeconds': time.monotonic()-stage, 'phaseLimitSeconds': seconds, 'cleanupSeconds': time.monotonic()-cleanup, 'observedBytes': counts, 'versionCaptureBytesMaximum': 256, 'stdoutBytesOvershoot': max(0,counts['stdout']-G['maxStdoutBytes']), 'stderrBytesOvershoot': max(0,counts['stderr']-G['maxStderrBytes']), 'stderrRawPersisted': False, 'stdoutRawPersisted': False, 'schemaTreeMaximumObserved': max_tree, 'schemaBytesOvershoot': max(0,max_tree['bytes']-G['maxSchemaBytes']), 'schemaFilesOvershoot': max(0,max_tree['files']-G['maxSchemaFiles']), 'monitorIsHardDiskQuota': False, 'nativeInternalActivity': None, 'automaticRetries': 0}
        save(f'result-{index+1}.json', result)
    return result


def worker():
    instruction = json.loads(sys.stdin.buffer.readline())
    require(instruction['go'] is True, 'missing-job-gate'); overall = instruction['startedMonotonic']
    b = source_guard(); require(sha(ROOT/'binding.json') == instruction['bindingSha256'], 'binding-file-changed')
    authority(); cwd_guard()
    results = []
    for index in range(2):
        result = call(index, overall); results.append(result)
        if result['stopReason']: break
    save('worker-result.json', {'key': G['key'], 'results': results, 'status': 'metadata-observed' if len(results)==2 and not results[-1]['stopReason'] else 'stopped', 'remainingAllocationClosed': True, 'rpcRequests': 0, 'actorReservations': 0, 'providerUsage': None})
    return 0 if len(results)==2 and not results[-1]['stopReason'] else 1


def main():
    b = source_guard(); authority(); cwd_guard()
    git = lambda *a: subprocess.check_output(['git', *a], cwd=REPO)
    require(not git('status','--porcelain').strip(), 'dirty-prestart-tree')
    for rel,digest in b['sourceFiles'].items():
        require(hashlib.sha256(git('show',b['sourceCommit']+':'+rel)).hexdigest()==digest, 'source-commit-pin-mismatch')
    require({x.name for x in OUT.iterdir()} == {'preparation.json'}, 'unexpected-external-content')
    require(sha(G['executable']) == G['executableSha256'], 'binary-prestart-mismatch')
    started = time.monotonic(); save('execution-once.json', {'key': G['key'], 'startedAtUnix': time.time(), 'sourceCommit': b['sourceCommit'], 'bindingCommit': git('rev-parse','HEAD').decode().strip(), 'bindingSha256': sha(ROOT/'binding.json'), 'maxCliInvocations': 2, 'actorReservations': 0})
    spec = importlib.util.spec_from_file_location('pinned_process_guard', PROCESS); module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
    job = child = None; reason = None; code = None; cleanup = started
    try:
        child = subprocess.Popen([sys.executable, str(Path(__file__).resolve()), '--worker'], cwd=G['cwd'], stdin=subprocess.PIPE, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, shell=False)
        job = module.WindowsJob(child)
        save('job-assigned.json', {'key': G['key'], 'workerPid': child.pid, 'controllerPid': os.getpid(), 'jobBeforeGo': True, 'bindingSha256': sha(ROOT/'binding.json')})
        child.stdin.write((json.dumps({'go':True,'startedMonotonic':started,'bindingSha256':sha(ROOT/'binding.json')})+'\n').encode()); child.stdin.close()
        while child.poll() is None:
            if time.monotonic() >= started+40: reason='overall-worker-deadline'; job.kill(); break
            time.sleep(.01)
        cleanup = time.monotonic(); code=child.wait(timeout=max(.001,min(5,started+45-time.monotonic())))
    except BaseException:
        reason='controller-or-job-failure'; cleanup=time.monotonic()
        if child and child.poll() is None:
            if job: job.kill()
            else: child.kill()
            try: code=child.wait(timeout=max(.001,min(5,started+45-time.monotonic())))
            except BaseException: reason='owned-cleanup-incomplete'
    finally:
        if job: job.close()
    receipt = {'key':G['key'],'workerReturnCode':code,'stopReason':reason,'controllerSeconds':time.monotonic()-started,'cleanupSeconds':time.monotonic()-cleanup,'jobKillOnClose':job is not None,'overallLimitSeconds':45,'remainingAllocationClosed':True,'automaticRetries':0}
    save('controller-receipt.json',receipt)
    # Outer bound includes receipt readback, not later schema analysis.
    readback=json.loads((OUT/'controller-receipt.json').read_bytes())
    result_path=OUT/'worker-result.json'; worker_result=json.loads(result_path.read_bytes()) if result_path.exists() else None
    elapsed=time.monotonic()-started
    save('outer-receipt.json',{'key':G['key'],'executionAndReadbackSeconds':elapsed,'within45Seconds':elapsed<=45,'controllerReceiptSha256':sha(OUT/'controller-receipt.json'),'workerResultSha256':sha(result_path) if worker_result else None,'status':worker_result['status'] if worker_result else 'terminal-result-missing','remainingAllocationClosed':True})
    print(json.dumps({'status':worker_result['status'] if worker_result else 'terminal-result-missing','executionAndReadbackSeconds':elapsed,'workerReturnCode':code,'remainingAllocationClosed':True}))
    return 0 if code==0 and reason is None and elapsed<=45 else 1


if __name__ == '__main__':
    try: status = worker() if sys.argv[1:]==['--worker'] else main() if not sys.argv[1:] else 2
    except BaseException:
        # No raw exception, stdout, stderr, config or credentials are surfaced.
        print('{"status":"preparation-or-binding-blocked","remainingAllocationClosed":true}')
        status=2
    raise SystemExit(status)
