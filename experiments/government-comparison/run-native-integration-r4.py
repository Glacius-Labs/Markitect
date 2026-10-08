"""One fresh Government-only R4 case; offline preparation precedes live slot admission."""
from __future__ import annotations
import argparse
import json
import os
from pathlib import Path
import subprocess
import sys
import time
ROOT = Path(__file__).resolve().parent
sys.path.insert(0, str(ROOT / "runtime"))
import dispatch
import government
import government_integration
import government_roles
import classic
import classic_integration
import native_controller
from native_fixture_budget import FixtureBudget, DEADLINE, validate_r4_grant_binding, validate_r4_entry_gate
from process import bounded
LEGACY = Path("C:/Users/Consiliari/Documents/Scientist-Probes/native-metadata-fixtures-20261008")
EXTERNAL = Path("C:/Users/Consiliari/Documents/Scientist-Probes/native-government-serialization-20261008-r4")
EVIDENCE = ROOT / "evidence/government-serialization-native-20261008-r4"
SOURCE_GRANT = LEGACY / "released-native-grant.json"
SUCCESSOR_GRANT = EXTERNAL / "released-r4-grant.json"
R4_KEY = "government-serialization-native-20261008-r4"
GRANT_SHA = "b917f5a5eb99f0a607fd282a81acdf7b085e6a1dba14f89c8d0c32f349c82c3e"


def write_new(path, value):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    raw = value if isinstance(value, bytes) else dispatch.encoded(value) + b'\n'
    with path.open('xb') as stream:
        stream.write(raw)
    return {'path': str(path.resolve()), 'sha256': dispatch.digest(raw)}


def binding(path):
    path = Path(path).resolve(strict=True)
    return {'path': str(path), 'sha256': dispatch.digest(path.read_bytes())}


def budget(arm='government'):
    if arm != 'government':
        raise ValueError('R4 excludes Classic')
    request = json.loads((EXTERNAL / arm / 'released/request.json').read_bytes())
    return FixtureBudget(LEGACY / 'native-starts.sqlite', SOURCE_GRANT, GRANT_SHA, {'government': government.PIN['accepted']['binary']['path'], 'classic': classic.inspect_packet(classic_integration.PACKET)['binary']['path']}, request=request)


def prepare():
    arm = 'government'
    base = EXTERNAL / arm
    released = base / 'released'
    auth = json.loads((released / 'role-auth.json').read_bytes())
    runtime = binding(released / 'runtime.json')
    auth_binding = binding(released / 'role-auth.json')
    repo = base / 'actor'
    evidence = base / 'controller-evidence'
    if evidence.exists():
        raise ValueError('R4 controller evidence must be a fresh absent directory')
    results = base / 'outer-results'
    results.mkdir(exist_ok=False)
    card = write_new(released / 'task-card.txt', 'Public deterministic native protocol fixture only. Use the selected product task and configured independent role slots. No model/provider, semantic or human acceptance claim.\n'.encode())
    mechanical = write_new(released / 'mechanical_actor.py', (ROOT / 'runtime/mechanical_actor.py').read_bytes())
    source = binding(SOURCE_GRANT)
    correction = {**binding(SUCCESSOR_GRANT), 'sourceKey': R4_KEY}
    if source['sha256'] != GRANT_SHA:
        raise ValueError('original source grant bytes changed')
    prep = json.loads((base / 'preparation-final.json').read_bytes())
    product, inputs = government_integration.product_binding(repository=repo, runtime_path=runtime['path'], runtime_raw=Path(runtime['path']).read_bytes(), backlog_path=released / 'backlog.json', backlog_raw=(released / 'backlog.json').read_bytes(), authorization_path=auth_binding['path'], authorization_raw=Path(auth_binding['path']).read_bytes(), queue_state_directory=prep['productGovernment']['queueStateDirectory'])
    revision = prep['baseCommit']
    common = json.loads((ROOT / 'public/resource-proposal.json').read_bytes())['commonLimits']
    request = {'schemaVersion': 1, 'mode': 'mechanical', 'operation': 'run_task', 'trialId': auth['trialId'], 'dispatchId': auth['dispatchId'], 'arm': arm, 'condition': 'brownfield', 'actorRepository': str(repo.resolve()), 'evidenceDirectory': str(evidence.resolve()), 'limits': common, 'wallSeconds': 38 if arm == 'government' else 180, 'baseCommit': revision, 'task': {'id': auth['taskId'], 'card': card}, 'purpose': 'task', 'releasedInputs': inputs + [source, binding(SUCCESSOR_GRANT), binding(EXTERNAL / 'coordinator-r4-preflight-snapshot.json'), binding(EXTERNAL / 'coordinator-r4-activation-snapshot.json'), binding(EXTERNAL / 'history-native-starts.sqlite'), auth['diagnostics'], card], 'prompt': card, 'mechanicalFixture': mechanical['path'], 'nativeFixtureGrant': {**source, 'sourceKey': 'native-s1-integration-fixtures-20261008'}, 'fixtureAuthorization': dict(government_roles.NATIVE_FIXTURE_AUTH), 'nativeFixtureR4Grant': correction, 'product': {arm: product}}
    request_binding = write_new(released / 'request.json', request)
    pin = dispatch.digest(dispatch.encoded(dispatch.mechanical_pin()))
    protocol = {'status': 'frozen', 'mode': 'mechanical', 'commonLimits': common, 'runnerPinSha256': pin, 'runtimeSourceSha256': dispatch.runtime_pins(), 'wrapperPythonSha256': dispatch.digest(Path(sys.executable).read_bytes()), 'fixtureSourceGrant': source, 'fixtureR4SourceGrant': correction, 'semanticAcceptance': False, 'humanAcceptance': False}
    protocol_binding = write_new(base / 'protocol.json', protocol)
    grant = {'schemaVersion': 1, 'status': 'approved', 'purpose': 's1-mechanics', 'mode': 'mechanical', 'trialId': auth['trialId'], 'notBefore': time.time() - 1, 'expiresAt': auth['expiresAt'] + 60, 'protocolSha256': protocol_binding['sha256'], 'profileSha256': dispatch.digest(dispatch.encoded(common)), 'runnerPinSha256': pin, 'ledgerPath': auth['ledgerPath'], 'resultDirectory': str(results.resolve()), 'maxActorSessions': 6, 'maxSessionWallSeconds': request['wallSeconds'], 'retrospectiveTokenThreshold': 10000, 'fixtureSourceGrant': source, 'fixtureR4SourceGrant': correction, 'authorizedRequests': [{'dispatchId': request['dispatchId'], 'executionSha256': dispatch.execution_sha(request), 'initialRequestSha256': request_binding['sha256']}]}
    grant_binding = write_new(base / 'grant.json', grant)
    return write_new(base / 'authority.json', {'request': request_binding, 'grant': grant_binding, 'protocol': protocol_binding})


def authority(arm='government'):
    if arm != 'government':
        raise ValueError('R4 excludes Classic')
    doc = json.loads((EXTERNAL / arm / 'authority.json').read_bytes())
    for item in doc.values():
        if binding(item['path']) != item:
            raise ValueError('frozen authority entry changed')
    a = dispatch.Authority(doc['grant']['path'], doc['grant']['sha256'], doc['protocol']['path'], doc['protocol']['sha256'])
    raw = Path(doc['request']['path']).read_bytes()
    r, captured = a.validate(raw)
    source = {key: r['nativeFixtureGrant'][key] for key in ('path', 'sha256')}
    if a.grant.get('fixtureSourceGrant') != source or a.protocol.get('fixtureSourceGrant') != source or a.grant.get('fixtureR4SourceGrant') != r['nativeFixtureR4Grant'] or (a.protocol.get('fixtureR4SourceGrant') != r['nativeFixtureR4Grant']) or (a.grant.get('maxActorSessions') != 6) or (a.mode != 'mechanical') or (source['sha256'] != GRANT_SHA):
        raise ValueError('compiled mechanical Authority exceeds original source allocation')
    return (a, r, raw, captured, Path(doc['request']['path']))


def validate():
    arm = 'government'
    a, r, raw, captured, request_path = authority(arm)
    bound = government.bind_request(r)
    native_controller.validate_native_fixture_grant(r, captured, bound)
    role = r['product'][arm]['roleAuthorization']
    government_roles.preflight_authorization(r, raw, a, captured, role['path'], Path(role['path']).read_bytes(), role['sha256'])
    if arm == 'government' and bound['backlogValue']['limits']['maxParallelism'] > 2:
        raise ValueError('Government fixture native parallelism exceeds source grant')
    return {'arm': arm, 'requestSha256': dispatch.digest(raw), 'validated': True, 'nativeStarts': 0, 'providerCalls': 0, 'sourcePins': dispatch.runtime_pins()}


def require_freeze():
    changes = subprocess.check_output(['git', 'status', '--porcelain'], cwd=ROOT, text=True).splitlines()
    allowed = 'experiments/government-comparison/evidence/government-serialization-native-20261008-r4/'
    if any(not line[3:].replace('\\', '/').startswith(allowed) for line in changes):
        raise ValueError('R4 source and all non-R4 evidence must remain clean')
    _, request, _, _, _ = authority('government')
    admitted = validate_r4_grant_binding(request, SOURCE_GRANT, GRANT_SHA)
    validate_r4_entry_gate(admitted)
    freeze = json.loads((EVIDENCE / 'preflight-freeze.json').read_bytes())
    committed = subprocess.check_output(['git','show',
        'HEAD:experiments/government-comparison/evidence/government-serialization-native-20261008-r4/preflight-freeze.json'], cwd=ROOT)
    if committed != (EVIDENCE / 'preflight-freeze.json').read_bytes():
        raise ValueError('R4 freeze must be committed byte-identically')
    subprocess.check_call(['git','merge-base','--is-ancestor',freeze['sourceCommit'],'HEAD'],cwd=ROOT)
    if freeze.get('status') != 'independently-reviewed-mechanical-fixture':
        raise ValueError('independent final fixture preflight missing')
    if freeze.get('grantKey') != R4_KEY or freeze.get('runtimeSourceSha256') != dispatch.runtime_pins():
        raise ValueError('R4 source/grant freeze differs from the admitted candidate')
    expected = {str(Path(item['path']).resolve()): item for item in freeze['inputs']}
    required = [Path(__file__), EVIDENCE / 'independent-preflight-review.md', SUCCESSOR_GRANT]
    required += [EXTERNAL / 'government' / name for name in ('authority.json','grant.json','protocol.json','released/request.json')]
    for path in required:
        if str(path.resolve()) not in expected:
            raise ValueError('required final source/authority/preflight absent from freeze')
    for item in expected.values():
        if binding(item['path']) != item:
            raise ValueError('frozen native fixture input changed before start')


def positive_checkpoint(process, queue, report, runtime):
    """Parent stop predicate, additional to the existing strict receipt translator."""
    if (process.get('returnCode') != 0 or process.get('stopReason') is not None or
            queue.get('status') != 'complete' or queue.get('nextStep') != 'none' or queue.get('error') or
            queue.get('inFlightActors') != 0):
        raise ValueError('Queue/process is not completely positive')
    jobs = queue.get('jobs', [])
    if (len(jobs) != 1 or jobs[0].get('id') != government_integration.TASK_ID or
            jobs[0].get('state') != 'accepted-scoped' or jobs[0].get('nextStep') != 'complete'):
        raise ValueError('Queue job is not accepted-scoped/complete')
    if (report.get('status') != 'accepted-scoped' or report.get('stage') != 'complete' or
            report.get('error') or not report.get('candidateCommit') or not report.get('candidateTree')):
        raise ValueError('Run is not accepted-scoped/complete')
    checks = report.get('checks')
    if (not isinstance(checks, list) or len(checks) != len(runtime['checks']) or
            [item.get('Name') for item in checks] != [item['name'] for item in runtime['checks']] or
            any(item.get('ExitCode') != 0 for item in checks)):
        raise ValueError('Fresh configured checks are missing or failed')
    promotion = report.get('promotion', {})
    if (promotion.get('status') != 'promoted' or
            promotion.get('expectedOld') != runtime['expectedBase'] or
            promotion.get('newCommit') != report['candidateCommit'] or
            promotion.get('actualActive') != report['candidateCommit'] or
            not promotion.get('intentPath') or not promotion.get('completionPath')):
        raise ValueError('Exact promotion and durable completion are required')
    roles = government.configured_roles(runtime)
    government._validate_run_report(report, report['runId'],
        {role['slotId']: (role['phase'], role['responseRole']) for role in roles},
        require_acceptance=True,
        root_review_slots={role['slotId'] for role in roles if role['phase'] == 'review' and not role.get('area')})
    return {'candidateCommit': report['candidateCommit'], 'candidateTree': report['candidateTree'],
            'materialCandidateId': report['evidence']['materialCandidateId'],
            'evidenceId': report['evidence']['id'], 'decisionId': report['decision']['id']}


def require_positive_queue():
    _, request, raw, _, _ = authority()
    evidence = Path(request['evidenceDirectory'])
    process = json.loads((evidence / 'process/process.json').read_bytes())
    translated = government.translate_queue_result(request, raw, evidence / 'process/stdout.log', process['returnCode'])
    if translated.get('status') != 'completed':
        raise ValueError('Strict Queue receipt translation is not completed')
    queue = json.loads((evidence / 'process/stdout.log').read_bytes())
    job = queue['jobs'][0]
    report = json.loads(Path(job['reportPath']).read_bytes())
    runtime = json.loads(Path(request['product']['government']['runtime']['path']).read_bytes())
    checkpoint = positive_checkpoint(process, queue, report, runtime)
    for key in ('intentPath', 'completionPath'):
        record = json.loads(Path(report['promotion'][key]).read_bytes())
        intent = record.get('intent', {}) if key == 'completionPath' else record
        if key == 'completionPath' and record.get('observedRef') != report['candidateCommit']:
            raise ValueError('Completion readback does not bind the promoted commit')
        expected = {'expectedOld': runtime['expectedBase'], 'newCommit': report['candidateCommit'],
                    'expectedTreeId': report['candidateTree'], 'materialCandidateId': checkpoint['materialCandidateId'],
                    'evidenceId': checkpoint['evidenceId'], 'decisionId': checkpoint['decisionId'],
                    'activeRef': runtime['activeRef'], 'idempotencyKey': report['runId']}
        if any(intent.get(name) != value for name, value in expected.items()):
            raise ValueError('Promotion intent/completion binding mismatch')
    active = subprocess.check_output(['git', '-C', request['actorRepository'], 'rev-parse', runtime['activeRef']], text=True).strip()
    if active != report['candidateCommit']:
        raise ValueError('Promoted active ref readback mismatch')
    return {'checkpoint': checkpoint, 'queue': queue, 'runReport': binding(job['reportPath'])}


def government_queue():
    require_freeze()
    validate()
    a, r, raw, captured, request_path = authority('government')
    argv = government.plan_request(r)['argv']
    b = budget('government')
    b.reserve('government', 'government-native-serialization-r4/queue', argv)
    result = dispatch.dispatch(request_path, a.result_directory / 'government-native-positive.json', a)
    process = Path(r['evidenceDirectory']) / 'process/process.json'
    if process.exists():
        b.finish('government', 'government-native-serialization-r4/queue', json.loads(process.read_bytes()))
    write_new(EVIDENCE / 'government/queue-result.json', result)
    snapshots = []
    for index, item in enumerate(result.get('receipts', [])):
        content = Path(item['path']).read_bytes()
        if dispatch.digest(content) != item['sha256']:
            raise ValueError('Government checkpoint receipt changed before snapshot')
        saved = write_new(EVIDENCE / 'government/queue-checkpoint' / str(index), content)
        snapshots.append({'original': item, 'snapshot': saved})
    write_new(EVIDENCE / 'government/queue-checkpoint.json', snapshots)
    return result


def government_resume():
    require_freeze()
    a, r, raw, captured, _ = authority('government')
    original = Path(r['evidenceDirectory']) / 'process/stdout.log'
    value = json.loads(original.read_bytes())
    positive = require_positive_queue()
    queue = Path(value['queueDirectory']).resolve(strict=True)
    if queue.parent != Path(r['product']['government']['queueStateDirectory']).resolve():
        raise ValueError('resume queue escaped approved parent')
    argv = government.resume_argv(r['product']['government']['executable']['path'], r['actorRepository'], r['product']['government']['backlog']['path'], queue)
    ledger = a.ledger()
    before = ledger.snapshot()
    b = budget('government')
    b.reserve('government', 'government-native-serialization-r4/resume', argv)
    fresh, fresh_captured = a.validate(raw)
    native_controller.validate_r4_entry_for_launch(fresh, fresh_captured, argv)
    process = bounded(argv, r['actorRepository'], Path(r['evidenceDirectory']) / 'resume-process', DEADLINE, env=native_controller.strip_bootstrap_environment(os.environ))
    b.finish('government', 'government-native-serialization-r4/resume', process)
    resumed = json.loads((Path(r['evidenceDirectory']) / 'resume-process/stdout.log').read_bytes())
    after = ledger.snapshot()
    result = {'process': process, 'nativeResult': resumed, 'roleStartsBefore': before['actorSessions'], 'roleStartsAfter': after['actorSessions'], 'sameQueue': resumed.get('queueDirectory') == value['queueDirectory'], 'sameNativeRoleStarts': resumed.get('actorStarts') == value.get('actorStarts'), 'noAdditionalRoles': before['actorSessions'] == after['actorSessions'], 'semanticAcceptance': False, 'humanAcceptance': False}
    result['samePersistedReport'] = resumed == value
    result['sameBoundRunReport'] = binding(positive['runReport']['path']) == positive['runReport']
    result['passed'] = (process['returnCode'] == 0 and process['stopReason'] is None and
                        all(result[key] for key in ('sameQueue','sameNativeRoleStarts','noAdditionalRoles',
                                                   'samePersistedReport','sameBoundRunReport')))
    write_new(EVIDENCE / 'government/resume-result.json', result)
    return result


def freeze():
    if subprocess.check_output(['git', 'status', '--porcelain'], cwd=ROOT):
        raise ValueError('commit the independently reviewed preflight before freezing')
    validate()
    _, request, _, _, _ = authority()
    paths = {Path(__file__), ROOT / 'prepare-native-r4.py', SUCCESSOR_GRANT,
             EXTERNAL / 'coordinator-r4-preflight-snapshot.json', EXTERNAL / 'coordinator-r4-activation-snapshot.json',
             EVIDENCE / 'independent-preflight-review.md', EVIDENCE / 'host-success-contract.md',
             EVIDENCE / 'historical-native-starts.sqlite'}
    paths.update((EXTERNAL / 'government').rglob('*.json'))
    paths.update(ROOT / 'runtime' / name for name in dispatch.runtime_pins() if (ROOT / 'runtime' / name).is_file())
    paths.add(ROOT / 'runtime/government_integration.py')
    runtime = json.loads(Path(request['product']['government']['runtime']['path']).read_bytes())
    for role in government.configured_roles(runtime):
        paths.update(Path(item['path']) for item in role['runtimeFiles'])
    paths.update(Path(item['path']) for item in request['releasedInputs'])
    paths.add(Path(request['product']['government']['executable']['path']))
    paths.add(Path(government_integration.FIXTURE_ROOT / 'deterministic_delegate.py'))
    paths.update((EXTERNAL / 'government/actor').rglob('*'))
    paths = {p for p in paths if p.is_file() and '.git' not in p.parts}
    value = {'grantKey': R4_KEY, 'status': 'independently-reviewed-mechanical-fixture',
             'sourceCommit': subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),
             'runtimeSourceSha256': dispatch.runtime_pins(), 'inputs': [binding(p) for p in sorted(paths,key=str)],
             'nativeStarts': 0, 'semanticAcceptance': False, 'humanAcceptance': False}
    return write_new(EVIDENCE / 'preflight-freeze.json', value)


def run_once():
    # A call before activation is rejected without consuming the one-case latch.
    require_freeze()
    write_new(EXTERNAL / 'case-claimed.json', {'grantKey': R4_KEY, 'claimedUtc': time.time()})
    outcome = {'grantKey': R4_KEY, 'status': 'incomplete', 'noRetry': True,
               'semanticAcceptance': False, 'humanAcceptance': False, 's1Complete': False}
    try:
        raw = Path('C:/Users/Consiliari/Glacius Labs/Markitect/docs/design/government/coordination-state.json').read_bytes()
        write_new(EVIDENCE / 'coordinator-start-snapshot.json', raw)
        result = government_queue()
        outcome['queueStatus'] = result.get('status')
        if result.get('status') == 'completed':
            outcome['positiveQueue'] = require_positive_queue()
            resumed = government_resume()
            outcome['resumePassed'] = resumed['passed']
            if resumed['passed']:
                outcome['status'] = 'bounded-mechanics-passed'
    except Exception as exc:
        outcome['error'] = str(exc)
    finally:
        write_new(EVIDENCE / 'terminal-result.json', outcome)
    return outcome


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=('prepare','validate','freeze','run-once'))
    args = parser.parse_args()
    print(json.dumps({'prepare':prepare, 'validate':validate, 'freeze':freeze, 'run-once':run_once}[args.action](), sort_keys=True))
