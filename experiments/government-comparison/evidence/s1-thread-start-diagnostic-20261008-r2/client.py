"""One-shot thread-start-only diagnostic client. Raw server bytes exist only in bounded memory.

Derived from the preserved classified client; no imports start processes.
The controller assigns a Windows Job to a gated Python worker before it can
launch the one app-server. Only that Job's owned process tree can be terminated.
"""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import queue
import re
import subprocess
import sys
import threading
import time
import copy
import math

ROOT = Path(__file__).resolve().parent
PACKAGE = ROOT.parents[1]
REPO = PACKAGE.parents[1]
KEY = "s1-thread-start-diagnostic-20261008-r2"
SOURCE_THREAD = "01a11367-a781-7683-a20f-46e12614dcb4"
ISSUED = "2026-10-08T15:48:46Z"
EXE = "C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/9691020b546a15b2/codex.exe"
EXE_SHA = "3553cd6e7df5a093d8cb8301cd8088a57e0971aba71ddbe0e67f7f44a15cdf68"
CWD = "C:/Users/Consiliari/Documents/Scientist-Probes/s1-common-runner-read-tool-20261008-r1/actor"
PRIOR = PACKAGE / "evidence/policy-read-disabled-status/run-1/classified-client.py"
PRIOR_SHA = "95c8486ad8962e14ad0c2da2989adcf154fa28f51d19c1fe983c532d1a885926"
PROCESS = PACKAGE / "runtime/process.py"
PROCESS_SHA = "d0cedb71d57be87095d9af119d9769fb0c3d311e1af073cba81f6337743c78ae"
SCHEMA = REPO / "experiments/government-comparison/evidence/s1-runner-rebinding-metadata-20261008-r1/generated-schemas.zip"
METHOD_ENUMS_SHA = "6553df9ac4a37d11402728992ed5e684ea379fbadeee790700867ae4542594cf"
FLAGS = {"apps", "goals", "hooks", "memories", "multi_agent", "plugins", "shell_tool", "unified_exec", "windows_sandbox_service", "network_proxy", "powershell_shell_version"}
ENUMS = {"approval_policy": {"never", "on-request", "untrusted", "on-failure"}, "approvals_reviewer": {"user", "auto_review"}, "sandbox_mode": {"read-only", "workspace-write", "danger-full-access"}, "windows": {"elevated", "unelevated"}, "model": {"gpt-6.1-sol"}, "model_provider": {"openai"}, "model_reasoning_effort": {"high"}}
LIMITS = dict(metadataHandshakeSeconds=33, threadStartSeconds=15, ownedProcessExecutionSeconds=50, maxCleanupSeconds=10, controllerWithCleanupSeconds=65, outerReceiptSeconds=80, maxQueuedFrames=64, maxSingleFrameBytes=1000000, maxRawBytes=8000000, maxInboundNotifications=2048, maxStderrCaptureBytes=16384, maxStderrDiagnosticLines=8)


def require(ok, reason):
    if not ok:
        raise ValueError(reason)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def load_module(path, expected, name):
    require(sha(path.read_bytes()) == expected, "source-pin-mismatch")
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def strict_json(raw):
    def pairs(items):
        out = {}
        for k, v in items:
            require(k not in out, "duplicate-json-key")
            out[k] = v
        return out
    return json.loads(raw, object_pairs_hook=pairs, parse_float=lambda x: finite_float(x), parse_constant=lambda _: (_ for _ in ()).throw(ValueError("invalid-json-number")))


def finite_float(value):
    number = float(value)
    require(math.isfinite(number), "invalid-json-number")
    return number


def field(obj, key, convert):
    if key not in obj:
        return {"state": "missing"}
    if obj[key] is None:
        return {"state": "null"}
    try:
        return {"state": "present", "value": convert(obj[key])}
    except (ValueError, TypeError, KeyError):
        return {"state": "invalid"}


def enum(value, allowed):
    require(type(value) is str and value in allowed, "invalid-enum")
    return value


def boolean(value):
    require(type(value) is bool, "invalid-boolean")
    return value


def profile_id(value):
    require(type(value) is str and re.fullmatch(r"[A-Za-z0-9:_-]{1,100}", value), "invalid-profile-id")
    return value if value == ":read-only" else "opaque-" + sha(value.encode())[:24]


def profile_map(value):
    require(type(value) is dict and len(value) <= 32, "invalid-profile-map")
    return {profile_id(k): boolean(v) for k, v in value.items()}


def profile_summary(value):
    require(type(value) is dict and len(value) <= 32, "invalid-profiles")
    require(all(type(v) is dict for v in value.values()), "invalid-profiles")
    return {profile_id(k): {"definitionPresent": True} for k in value}


ADAPTER = PACKAGE / "runtime/selected_feature_contract.py"
ADAPTER_SHA = "985ff1d8f8870de4ac697e030c2abeb543d1bc3e74f475968c5e1097be865550"


def adapter():
    return load_module(ADAPTER, ADAPTER_SHA, "accepted_selected_feature_contract")


def sanitize_config_read(result):
    return adapter().sanitize_config_read(result)


def sanitize_requirements(result):
    return adapter().sanitize_requirements(result)


def assess(config, requirements):
    return adapter().assess(config, requirements)


def assess_config(config):
    # Preliminary only. This placeholder can never satisfy final_assess.
    status, reason = assess(config, {"requirements": {"state": "null"}})
    return ("config-precheck-satisfied-awaiting-requirements", None) if status == "visible-metadata-precondition-satisfied" and reason is None else (status, reason)


def final_assess(observations, sent):
    expected = [("initialize", 0), ("initialized", None), ("config/read", 1), ("configRequirements/read", 2)]
    if [(x.get("method"), x.get("id")) for x in sent] != expected or [x.get("id") for x in observations] != [0, 1, 2]:
        return "insufficient", "actual-requirements-response-required"
    actual = observations[2]
    if actual.get("method") != "configRequirements/read" or actual.get("responseEnvelopeValidated") is not True:
        return "insufficient", "actual-requirements-response-required"
    status, reason = assess(observations[1]["sanitizedResult"], actual["sanitizedResult"])
    return ("reported-config-and-requirements-observed", None) if status == "visible-metadata-precondition-satisfied" and reason is None else (status, reason)


def decode_response(frame, identifier):
    prior = load_module(PRIOR, PRIOR_SHA, "preserved_response_guard")
    return sanitize(identifier, prior.validate_response_envelope(frame, identifier))


def classify(frame, enums, count):
    prior = load_module(PRIOR, PRIOR_SHA, "preserved_classifier")
    result = prior.classify_server_frame(frame, enums, count)
    if not isinstance(frame, dict) or set(frame) - {"method", "params", "emittedAtMs"} or "result" in frame or "error" in frame:
        return {**result, "action": "stop", "reason": "unexpected-frame-envelope"}
    if frame.get("method") != "remoteControl/status/changed":
        return {**result, "action": "stop", "reason": "warning-or-unexpected-method"}
    return result


def reject_provisional(value):
    if isinstance(value, dict):
        for k, v in value.items():
            if k.lower() in {"warning", "warnings", "configwarning", "provisional", "provisionalconfig"} and v not in (None, False, [], {}):
                raise ValueError("warning-or-provisional-result")
            reject_provisional(v)
    elif isinstance(value, list):
        for v in value:
            reject_provisional(v)


def sanitize(identifier, result):
    return adapter().sanitize(identifier, result)


R1 = PACKAGE / "evidence/s1-common-runner-read-tool-20261008-r1"
PROTOCOL = R1 / "protocol.py"
PROTOCOL_SHA = "ebd087657febf63846904d83a75bc226a5412458418d20abe024ed267febb822"
METHODS = ["initialize", "initialized", "config/read", "configRequirements/read", "thread/start"]


def validate_profile(profile, grant):
    previous = strict_json((R1/'profile.json').read_bytes())
    require(sha((R1/'profile.json').read_bytes())=='6a1d9c043758563d59b576c0a2c526e375f6bc819002768a535333b7cb1a128c','previous-profile-pin-mismatch')
    rpc=copy.deepcopy(previous['rpc']);rpc[0]['params']['clientInfo']['name']='scientist_thread_start_diagnostic_r2'
    argv=copy.deepcopy(previous['argv']);argv[0]=EXE
    require(profile['key']==grant['key']==KEY and grant['issuedUtc']==ISSUED and grant['sourceThreadId']==SOURCE_THREAD and grant['baseSha']=='8c1587b7a8ecdc58afcc8e77a5a6c27af626eeb7','grant-identity-mismatch')
    require(grant['ownerThreadId']=='01a1169c-3df6-7571-997d-48ab57365875' and grant['executable']==EXE and grant['executableSha256']==EXE_SHA,'grant-owner-or-binary-mismatch')
    require(set(profile)=={'key','argv','cwd','rpc','threadStart','expectedFiles','externalEvidence','limits'},'profile-extra-fields')
    require(profile['argv']==argv and profile['argv'].count('--config')==17 and profile['rpc']==rpc,'argv-or-rpc-mismatch')
    require(profile['cwd']==grant['cwd']==CWD and profile['externalEvidence']==grant['evidenceDirectory'] and profile['threadStart']==previous['threadStart'] and profile['expectedFiles']==previous['expectedFiles'],'thread-or-input-mismatch')
    require(profile['limits']==LIMITS and all(grant[k]==v for k,v in LIMITS.items()),'limit-mismatch')
    require(grant['clientMethods']==METHODS and grant['previousAppServerTrees']==6 and grant['cumulativeAppServerTreesMaximum']==7 and grant['previousActorReservations']==6,'grant-history-mismatch')
    require(all(grant[k]==1 for k in ['maxNewAppServerTrees','maxNewDiagnosticReservations','maxThreadStarts','maxParallel']),'grant-count-mismatch')
    require(all(grant[k]==0 for k in ['maxNewActorTaskReservations','maxTurnStarts','maxToolItems','maxCliMetadataCalls','retries','maxProducts','maxProductWrappers','maxDelegates','maxStudyCells','maxFullProductSuites']),'grant-forbidden-work')


def validate_cwd(profile):
    cwd=Path(profile['cwd']);require(cwd.is_dir() and not cwd.is_symlink() and not cwd.is_junction(),'cwd-invalid')
    paths=list(cwd.rglob('*'));require(all(p.is_file() and not p.is_symlink() and not p.is_junction() for p in paths),'unexpected-cwd-entry')
    inventory={p.relative_to(cwd).as_posix():sha(p.read_bytes()) for p in paths};require(inventory==profile['expectedFiles'],'cwd-content-mismatch');return inventory


def validate_live_authority(grant, slot, original):
    require(grant==original,'live-grant-mismatch')
    require(grant['status']=='Dispatched once; active preparation confirmed.','inactive-live-grant')
    require(slot['owner']=='Scientist' and slot['key']==KEY and slot['assignedUtc']==ISSUED and slot['status']=='Assigned to one prospective thread-start-only diagnostic.','inactive-live-slot')


def validate_pre_reservation():
    authority=strict_json((ROOT/'authorization-grant.json').read_bytes());profile=strict_json((ROOT/'profile.json').read_bytes());validate_profile(profile,authority['grant'])
    live=strict_json(Path(authority['sourceCoordinationPath']).read_bytes());grant=next(t for t in live['threads'] if t['name']=='Scientist')['evidence']['threadStartDiagnosticR2Grant'];validate_live_authority(grant,live['fullSuiteSlot'],authority['grant'])
    require(sha(Path(EXE).read_bytes())==EXE_SHA and sha(SCHEMA.read_bytes())==grant['schemaArchiveSha256'],'pre-reservation-binary-or-schema-pin-mismatch');validate_cwd(profile)


def load_request():
    request=strict_json((ROOT/'request.json').read_bytes());freeze_bytes=(ROOT/'freeze.json').read_bytes();freeze=strict_json(freeze_bytes);profile=strict_json((ROOT/'profile.json').read_bytes());authority=strict_json((ROOT/'authorization-grant.json').read_bytes())
    validate_profile(profile,authority['grant']);live=strict_json(Path(authority['sourceCoordinationPath']).read_bytes());grant=next(t for t in live['threads'] if t['name']=='Scientist')['evidence']['threadStartDiagnosticR2Grant'];validate_live_authority(grant,live['fullSuiteSlot'],authority['grant'])
    reservation_path=Path(profile['externalEvidence'])/'reservation.json';reservation=strict_json(reservation_path.read_bytes());git=lambda *a:subprocess.check_output(['git',*a],cwd=REPO);head=git('rev-parse','HEAD').decode().strip()
    require(not git('status','--porcelain').strip() and reservation['freezeCommit']==head,'unclean-or-wrong-freeze-commit')
    require(request['key']==freeze['key']==reservation['key']==KEY and request['sourceCommit']==freeze['sourceCommit']==reservation['sourceCommit'],'binding-identity-mismatch')
    require(request['profileSha256']==sha((ROOT/'profile.json').read_bytes()) and request['authorizationGrantSha256']==sha((ROOT/'authorization-grant.json').read_bytes()),'profile-or-grant-pin-mismatch')
    require(reservation['requestSha256']==freeze['requestSha256']==sha((ROOT/'request.json').read_bytes()) and reservation['freezeSha256']==sha(freeze_bytes),'request-freeze-reservation-mismatch')
    require(reservation['maxTrees']==reservation['maxDiagnosticReservations']==1 and reservation['maxActorTaskReservations']==0 and reservation['priorAppServerTrees']==reservation['priorActorReservations']==6,'reservation-count-mismatch')
    required={str((ROOT/n).relative_to(REPO).as_posix()) for n in ['client.py','run_once.py','stderr_collector.py','diagnostic-contract.json','thread_gate.py','schema-contract.json','profile.json','authorization-grant.json','test_client.py','focused-tests.log','preflight-review.md','offline-validation.json']}
    required|={p.relative_to(REPO).as_posix() for p in [PROTOCOL,PRIOR,PROCESS,ADAPTER,SCHEMA,R1/'profile.json']};require(required<=set(request['sourceFiles']),'required-source-pins-missing')
    for rel,digest in request['sourceFiles'].items():require(sha((REPO/rel).read_bytes())==sha(git('show',request['sourceCommit']+':'+rel))==digest,'source-pin-mismatch')
    for path,digest in freeze['files'].items():require(sha(Path(path).read_bytes())==digest,'freeze-input-mismatch')
    required_inputs={str(ROOT/'request.json'),EXE,request['interpreter']['path'],str(Path(CWD)/'README.md'),str(Path(CWD)/'probe-input.txt')};require(required_inputs<=set(freeze['files']),'required-freeze-inputs-missing')
    ledger_pins=strict_json((PACKAGE/'evidence/s1-common-runner-readiness-plan-20261008/source-evidence.json').read_bytes())['liveLedgerPinsReadOnly'];require(len(ledger_pins)==5 and all(freeze['files'].get(p['path'])==p['sha256'] for p in ledger_pins),'historical-ledger-freeze-mismatch')
    require(Path(sys.executable).resolve()==Path(request['interpreter']['path']).resolve() and sha(Path(sys.executable).read_bytes())==request['interpreter']['sha256'],'interpreter-pin-mismatch')
    require(sha(Path(EXE).read_bytes())==EXE_SHA and sha(SCHEMA.read_bytes())==grant['schemaArchiveSha256'] and sha((ROOT/'frozen-method-enums.json').read_bytes())==METHOD_ENUMS_SHA,'binary-or-schema-pin-mismatch')
    validate_cwd(profile)
    return profile,{'requestSha256':sha((ROOT/'request.json').read_bytes()),'freezeSha256':sha(freeze_bytes),'reservationSha256':sha(reservation_path.read_bytes()),'sourceCommit':request['sourceCommit'],'freezeCommit':head}


def exclusive_json(path,value):
    with Path(path).open('x',encoding='utf-8') as f:json.dump(value,f,indent=2);f.write('\n');f.flush();os.fsync(f.fileno())


def admit_rpc(method,sent_methods,stderr_triggered,thread_response_validated):
    require(not stderr_triggered,'server-stderr-activity')
    require(not thread_response_validated,'rpc-after-thread-terminal')
    require(type(sent_methods) is list and sent_methods==METHODS[:len(sent_methods)] and len(sent_methods)<len(METHODS) and method==METHODS[len(sent_methods)],'forbidden-or-out-of-order-client-method')


def consume_stderr(collector,raw,cleanup=False):
    require(not raw or not collector.triggered or cleanup,'stderr-fragment-outside-cleanup')
    collector.feed(raw)
    return 'server-stderr-activity' if collector.triggered else None


def session():
    require(sys.stdin.buffer.readline()==b'GO\n','missing-job-gate');profile,binding=load_request();out=Path(profile['externalEvidence']);job=strict_json((out/'job-assigned.json').read_bytes());require(job['workerPid']==os.getpid() and job['controllerPid']==os.getppid() and all(job.get(k)==v for k,v in binding.items()),'job-owner-or-binding-mismatch')
    pins=strict_json((ROOT/'request.json').read_bytes())['sourceFiles']
    module=lambda name:load_module(ROOT/name,pins[(ROOT/name).relative_to(REPO).as_posix()],'frozen_'+name[:-3])
    contract=strict_json((ROOT/'schema-contract.json').read_bytes());protocol=load_module(PROTOCOL,PROTOCOL_SHA,'frozen_protocol').Protocol(SCHEMA,contract);gate=module('thread_gate.py').ThreadGate(profile,protocol,contract);diagnostic=module('stderr_collector.py').StderrCollector(profile['limits']['maxStderrCaptureBytes'],profile['limits']['maxStderrDiagnosticLines']);prior=load_module(PRIOR,PRIOR_SHA,'prior_frame_guard');enums=prior.load_method_enums(ROOT/'frozen-method-enums.json');prior.validate_disabled_status_gate(enums)
    started=time.monotonic();exclusive_json(out/'started-once.json',{'key':KEY,'startedAtUnix':time.time(),**binding});messages=queue.Queue(maxsize=64);limit=threading.Event();stderr_seen=threading.Event();cleanup_started=threading.Event();lock=threading.Lock();counts={'raw':0,'stdout':0,'stderr':0};in_flight=0;sent=[];observations=[];events={};notifications=0;server=None;threads=[];reason=None;native=None;status='stopped';phase='metadata';phase_times={};preliminary=None;metadata_final=None

    def pump(stream,kind):
        nonlocal in_flight
        pending=b''
        while True:
            if kind=='stderr' and stderr_seen.is_set():
                while not cleanup_started.wait(.025):
                    if time.monotonic()>=started+50:limit.set();return
            with lock:
                allowance=min(8192,8000000-counts['raw']-in_flight)
                if allowance<=0:limit.set();return
                in_flight+=allowance
            raw=stream.read1(allowance)
            # Signal immediately after a nonempty read, before waiting for the
            # admission lock. A write already admitted can still be in flight.
            if kind=='stderr' and raw:stderr_seen.set()
            with lock:
                in_flight-=allowance;counts['raw']+=len(raw);counts[kind]+=len(raw)
                if kind=='stderr' and consume_stderr(diagnostic,raw,cleanup_started.is_set()):stderr_seen.set()
            if kind=='stderr':
                if not raw:return
                continue
            pending+=raw
            while b'\n' in pending:
                line,pending=pending.split(b'\n',1)
                if len(line)+1>1000000:limit.set();return
                try:messages.put_nowait(line+b'\n')
                except queue.Full:limit.set();return
            if len(pending)>1000000:limit.set();return
            if not raw:
                try:
                    if pending:messages.put_nowait(pending)
                    messages.put_nowait(b'')
                except queue.Full:limit.set()
                return

    def consume(raw,expected=None):
        nonlocal notifications
        require(raw and len(raw)<=1000000,'stdout-closed-or-frame-limit');frame=strict_json(raw)
        if type(frame) is dict and 'method' in frame:
            notifications+=1;require(notifications<=2048,'notification-limit');require(set(frame)<={'method','params','emittedAtMs'} and type(frame.get('method')) is str and 'params' in frame,'server-request-or-notification-envelope')
            if 'emittedAtMs' in frame:require(type(frame['emittedAtMs']) is int and -(2**63)<=frame['emittedAtMs']<2**63,'notification-timestamp')
            if frame['method']=='remoteControl/status/changed':
                decision=classify(frame,enums,0);require(decision['action']=='discard-notification',decision['reason'] or 'remote-control-not-disabled')
            else:
                require(phase=='thread','lifecycle-before-thread-request');reject_provisional(frame['params']);gate.accept_notification(frame['method'],frame['params'])
            events[frame['method']]=events.get(frame['method'],0)+1;return None
        require(expected is not None,'unsolicited-response');result=prior.validate_response_envelope(frame,expected['id']);reject_provisional(result);protocol.validate(contract['responseSchemas'][expected['method']],result);return result

    def send(rpc):
        with lock:
            admit_rpc(rpc['method'],[s['method'] for s in sent],stderr_seen.is_set(),gate.validated)
            member=contract['requestSchemas'].get(rpc['method'])
            if member:protocol.validate(member,rpc['params'])
            else:require(rpc['method']=='initialized' or (rpc['method']=='configRequirements/read' and 'params' not in rpc),'unexpected-client-params')
            entry={'method':rpc['method'],'id':rpc.get('id'),'writeCompleted':False};sent.append(entry);server.stdin.write((json.dumps(rpc)+'\n').encode());server.stdin.flush();entry['writeCompleted']=True

    def request(rpc,deadline):
        require(not stderr_seen.is_set(),'server-stderr-activity')
        while not messages.empty():consume(messages.get_nowait())
        send(rpc)
        if 'id' not in rpc:return None
        while True:
            require(not stderr_seen.is_set(),'server-stderr-activity');require(not limit.is_set() and time.monotonic()<deadline and time.monotonic()<started+50,'rpc-resource-or-deadline')
            try:raw=messages.get(timeout=.025)
            except queue.Empty:continue
            require(not stderr_seen.is_set(),'server-stderr-activity');answer=consume(raw,rpc)
            if answer is not None:return answer

    try:
        env={k:v for k,v in os.environ.items() if k not in {'OPENAI_API_KEY','CODEX_API_KEY'}};server=subprocess.Popen(profile['argv'],cwd=profile['cwd'],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE,shell=False,env=env)
        for stream,kind in [(server.stdout,'stdout'),(server.stderr,'stderr')]:
            t=threading.Thread(target=pump,args=(stream,kind),daemon=True);threads.append(t);t.start()
        for rpc in profile['rpc']:
            answer=request(rpc,started+33)
            if answer is None:continue
            safe=sanitize(rpc['id'],answer);observations.append({'id':rpc['id'],'method':rpc['method'],'responseEnvelopeValidated':True,'sanitizedResult':safe})
            if rpc['id']==1:preliminary,why=assess_config(safe);require(preliminary=='config-precheck-satisfied-awaiting-requirements',why or 'config-precheck-nonpositive')
        metadata_final,why=final_assess(observations,sent);require(metadata_final=='reported-config-and-requirements-observed',why or 'metadata-precheck-nonpositive');phase_times['metadataSeconds']=time.monotonic()-started;phase='thread';stage=time.monotonic();answer=request(profile['threadStart'],min(stage+15,started+50));gate.accept_response(answer);phase_times['threadStartSeconds']=time.monotonic()-stage;status='thread-response-observed'
    except ValueError as exc:reason=str(exc) if re.fullmatch('[a-z-]{1,100}',str(exc)) else 'invalid-or-unexpected-response'
    except BaseException:reason='local-client-exception'
    finally:
        cleanup=time.monotonic();cleanup_started.set();cleanup_deadline=min(cleanup+8,started+48)
        if server:
            try:server.stdin.close();native=server.wait(timeout=max(.001,min(2,cleanup_deadline-time.monotonic())))
            except BaseException:
                reason=reason or 'native-cleanup-required'
                try:server.kill();native=server.wait(timeout=max(.001,min(2,cleanup_deadline-time.monotonic())))
                except BaseException:reason='native-cleanup-failed'
        for t in threads:t.join(timeout=max(0,cleanup_deadline-time.monotonic()))
        if any(t.is_alive() for t in threads):reason=reason or 'drain-deadline'
        while not messages.empty():
            raw=messages.get_nowait()
            if raw:
                try:consume(raw)
                except BaseException:reason=reason or 'unexpected-final-output'
        with lock:diagnostic_result=diagnostic.finish()
        if diagnostic_result['triggered']:reason='server-stderr-activity' if reason is None else reason
        if limit.is_set() or native!=0 or time.monotonic()>=started+50:reason=reason or 'native-resource-exit-or-deadline'
        try:validate_cwd(profile)
        except BaseException:reason=reason or 'public-cwd-changed'
        if reason:status='stopped'
        result={'key':KEY,'status':status,'stopReason':reason,**binding,'sentRpc':sent,'metadataObservations':observations,'configPrecheckStatus':preliminary,'metadataFinalStatus':metadata_final,'threadBoundary':gate.finish(),'stderrDiagnostic':diagnostic_result,'eventCounts':events,'notificationCount':notifications,'nativeProcessStarted':server is not None,'nativePid':server.pid if server else None,'nativeReturnCode':native,'terminalAtUnix':time.time(),'nativeSessionWallSeconds':time.monotonic()-started,'cleanupSeconds':time.monotonic()-cleanup,'phaseTimes':phase_times,'rawBytesDiscarded':counts,'rawFramePayloadsPersisted':False,'rawStderrPersisted':False,'turnStartRequests':0,'taskToolRequests':0,'interruptRequests':0,'actorTaskReservations':0,'providerRequests':None,'providerRetries':None,'usage':None,'servingModel':None,'providerBillingStopped':None,'automaticRetries':0,'meaning':'Only prospective safe thread-start diagnostics; no task, tools, OS enforcement, historical cause or general S1.'};exclusive_json(out/'sanitized-result.json',result)
    return 0 if reason is None and status=='thread-response-observed' else 1


def controller():
    profile, binding = load_request()
    external = Path(profile["externalEvidence"])
    job_type = load_module(PROCESS, PROCESS_SHA, "existing_process_guard").WindowsJob
    started = time.monotonic(); worker = None; job = None; reason = None; code = None
    exclusive_json(external / "controller-once.json", {"key": KEY, **binding, "startedAtUnix": time.time()})
    try:
        worker = subprocess.Popen([sys.executable, str(Path(__file__).resolve()), "--session"], cwd=profile["cwd"], stdin=subprocess.PIPE, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, shell=False)
        job = job_type(worker)
        exclusive_json(external / "job-assigned.json", {"key": KEY, **binding, "controllerPid": os.getpid(), "workerPid": worker.pid, "processTreeControl": "windows-job-kill-on-close"})
        worker.stdin.write(b"GO\n"); worker.stdin.close()
        while worker.poll() is None:
            if time.monotonic()-started >= 50:
                reason = "owned-process-deadline"; job.kill(); break
            time.sleep(.01)
        code = worker.wait(timeout=10)
    except BaseException:
        reason = "controller-exception"
        if worker and worker.poll() is None:
            try:
                if job:
                    job.kill()
                else:
                    worker.kill()  # Still gated; target cannot have launched.
                code = worker.wait(timeout=10)
            except BaseException:
                reason = "controller-cleanup-failed"
    finally:
        cleanup = time.monotonic()
        if job:
            job.close()
        elapsed = time.monotonic()-started
        receipt = {"key": KEY, **binding, "workerReturnCode": code, "stopReason": reason, "controllerSeconds": elapsed, "jobCloseSeconds": time.monotonic()-cleanup, "processTreeControl": "windows-job-kill-on-close", "deadlineSeconds": 50, "controllerLimitSeconds": 65, "outerLimitSeconds": 80, "rawLogsPersisted": False, "automaticRetries": 0}
        result_path = external / "sanitized-result.json"
        receipt["terminalResultPresent"] = result_path.is_file()
        receipt["terminalResultSha256"] = sha(result_path.read_bytes()) if result_path.is_file() else None
        receipt["withinControllerBound"] = elapsed <= 65
        exclusive_json(external / "controller-receipt.json", receipt)
    return 0 if code == 0 and reason is None and elapsed <= 65 and result_path.is_file() else 1


if __name__ == "__main__":
    try:
        code = session() if sys.argv[1:] == ["--session"] else controller() if sys.argv[1:] == ["--controller"] else 2
    except BaseException:
        # No traceback or raw exception may reach a persisted log.
        code = 2
    raise SystemExit(code)
