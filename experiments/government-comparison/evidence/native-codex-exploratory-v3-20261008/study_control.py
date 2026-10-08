"""Small durable admission/stage recorder for Scientist's native tool dispatcher."""
import argparse
import datetime as dt
import hashlib
import json
from pathlib import Path
import subprocess
import time

ROOT = Path(__file__).resolve().parent
LEDGER = ROOT / 'ledger.json'


def now():
    return dt.datetime.now(dt.timezone.utc)


def utc(value):
    return dt.datetime.fromisoformat(value.replace('Z', '+00:00'))


def write(path, value):
    temporary = path.with_suffix(path.suffix + '.tmp')
    temporary.write_text(json.dumps(value, indent=2) + '\n', encoding='utf-8')
    temporary.replace(path)


def load():
    return json.loads(LEDGER.read_text())


def reserve(trial, task, role, actor=None, prompt=None):
    ledger = load()
    instant = now()
    assert instant < utc(ledger['overallDeadlineUtc'])
    cell = next(c for c in ledger['cells'] if c['trialId'] == trial)
    assert type(task) is int and 1 <= task <= 6
    assert cell.get('status') not in ['closed', 'contaminated']
    if 'startedUtc' not in cell:
        cell.update(startedUtc=instant.isoformat(), deadlineUtc=min(instant + dt.timedelta(seconds=7200), utc(ledger['overallDeadlineUtc'])).isoformat(), startedMonotonic=time.monotonic(), status='running', tasks={})
    assert instant < utc(cell['deadlineUtc'])
    assert time.monotonic() - cell['startedMonotonic'] < 7200
    if str(task) not in cell['tasks'] and task > 1:
        previous = cell['tasks'].get(str(task-1))
        assert previous and previous['status'] == 'closed'
        if task == 6:
            assert previous['outcome'] in ['complete', 'passed']
    stage = cell['tasks'].setdefault(str(task), {'startedUtc': instant.isoformat(), 'startedMonotonic': time.monotonic(), 'deadlineUtc': min(instant + dt.timedelta(seconds=1200), utc(cell['deadlineUtc'])).isoformat(), 'status': 'running'})
    assert stage['status'] != 'contaminated' and instant < utc(stage['deadlineUtc'])
    assert stage['status'] != 'closed' or role == 'independent-evaluation'
    assert time.monotonic() - stage['startedMonotonic'] < 1200
    calls = ledger['trialActivations']
    assert len(calls) < 432
    assert sum(x['trialId'] == trial for x in calls) < 72
    assert sum(x['trialId'] == trial and x['task'] == task for x in calls) < 12
    assert sum(x['status'] in ['reserved', 'running'] for x in calls) < 4
    record = {'reservationId': f'activation-{len(calls)+1:04d}', 'trialId': trial, 'task': task,
              'role': role, 'reservedUtc': instant.isoformat(), 'reservedMonotonic': time.monotonic(),
              'actorId': actor, 'status': 'reserved', 'usage': None,
              'deadlineUtc': stage['deadlineUtc'], 'remainingTaskActivations': 11-sum(x['trialId']==trial and x['task']==task for x in calls)}
    if prompt:
        data = Path(prompt).read_bytes()
        record.update(promptPath=str(Path(prompt).resolve()), promptSha256=hashlib.sha256(data).hexdigest())
    calls.append(record)
    write(LEDGER, ledger)
    with (ROOT / 'dispatch-events.jsonl').open('a', encoding='utf-8') as events:
        events.write(json.dumps({'event': 'reserved-before-native-call', **record}) + '\n')
    return record


def update(reservation, actor=None, status=None, result=None):
    ledger = load()
    matches = ledger['trialActivations'] + ledger['preparationActivations']
    record = next(x for x in matches if x['reservationId'] == reservation)
    if actor:
        record['actorId'] = actor
    if status:
        record['status'] = status
        record['observedUtc'] = now().isoformat()
        if status in ['completed', 'failed', 'interrupted']:
            record['endedMonotonic'] = time.monotonic()
    if result:
        record['resultPath'] = str(Path(result).resolve())
        record['resultSha256'] = hashlib.sha256(Path(result).read_bytes()).hexdigest()
    write(LEDGER, ledger)
    with (ROOT / 'dispatch-events.jsonl').open('a', encoding='utf-8') as events:
        events.write(json.dumps({'event': 'activation-observation', **record}) + '\n')
    return record


def snapshot(trial, task, repository, status):
    ledger = load()
    cell = next(c for c in ledger['cells'] if c['trialId'] == trial)
    stage = cell['tasks'][str(task)]
    candidate = subprocess.check_output(['git', '-C', repository, 'rev-parse', 'HEAD'], text=True).strip()
    dirty = subprocess.check_output(['git', '-C', repository, 'status', '--porcelain'], text=True).strip()
    assert not dirty, 'Actor must freeze a committed candidate before next stage'
    stage.update(status='closed', outcome=status, candidateSha=candidate, frozenUtc=now().isoformat(), endedMonotonic=time.monotonic())
    write(LEDGER, ledger)
    return stage


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    sub = parser.add_subparsers(dest='action', required=True)
    r = sub.add_parser('reserve'); r.add_argument('--trial', required=True); r.add_argument('--task', type=int, required=True); r.add_argument('--role', required=True); r.add_argument('--actor'); r.add_argument('--prompt')
    u = sub.add_parser('update'); u.add_argument('--reservation', required=True); u.add_argument('--actor'); u.add_argument('--status'); u.add_argument('--result')
    s = sub.add_parser('snapshot'); s.add_argument('--trial', required=True); s.add_argument('--task', type=int, required=True); s.add_argument('--repository', required=True); s.add_argument('--status', required=True)
    args = vars(parser.parse_args()); action = args.pop('action')
    print(json.dumps({'reserve': reserve, 'update': update, 'snapshot': snapshot}[action](**args)))
