"""Transport a real product Invocation to a counted native role; never decide it."""
import argparse
import datetime as dt
import hashlib
import json
from pathlib import Path
import sys
import time

MAX_BYTES = 8 * 1024 * 1024


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--mailbox', required=True)
    parser.add_argument('--trial', required=True)
    parser.add_argument('--task', type=int, required=True)
    parser.add_argument('--deadline', required=True)
    args = parser.parse_args()
    deadline = dt.datetime.fromisoformat(args.deadline.replace('Z', '+00:00')).timestamp()
    raw = sys.stdin.buffer.read(MAX_BYTES + 1)
    if len(raw) > MAX_BYTES:
        raise ValueError('Invocation exceeds bridge bound')
    invocation = json.loads(raw)
    if invocation.get('apiVersion') != 'markitect.example.org/agent-execution/v1alpha1':
        raise ValueError('Unsupported native Invocation')
    run_id = invocation['runId']
    if not isinstance(run_id, str) or not run_id.isalnum() or len(run_id) > 128:
        raise ValueError('Unsafe native run identity')
    mailbox = Path(args.mailbox)
    if not mailbox.is_absolute():
        raise ValueError('Absolute own-cell mailbox required')
    mailbox.mkdir(parents=True, exist_ok=True)
    directory = mailbox / run_id
    directory.mkdir(exist_ok=False)
    (directory / 'invocation.json').write_bytes(raw)
    created = dt.datetime.now(dt.timezone.utc)
    expires = min(deadline, created.timestamp() + 580)
    request = {'trialId': args.trial, 'task': args.task,
               'invocationSha256': digest(raw), 'createdUtc': created.isoformat(),
               'expiresUtc': dt.datetime.fromtimestamp(expires, dt.timezone.utc).isoformat(),
               'runId': run_id, 'role': invocation['request']['role']}
    (directory / 'request.json').write_text(json.dumps(request, indent=2) + '\n')
    response = directory / 'response.json'
    while time.time() < expires:
        if response.is_file():
            answer = response.read_bytes()
            if len(answer) > MAX_BYTES:
                raise ValueError('Response exceeds bridge bound')
            value = json.loads(answer)
            for key in ['apiVersion', 'runId', 'nonce', 'inputDigest']:
                if value.get(key) != invocation.get(key):
                    raise ValueError('Role response identity mismatch')
            if value.get('role') != invocation['request']['role']:
                raise ValueError('Role mismatch')
            receipt = {**request, 'responseSha256': digest(answer),
                       'completedUtc': dt.datetime.now(dt.timezone.utc).isoformat(),
                       'semanticFieldsSuppliedByNativeRole': True, 'usage': value.get('usage')}
            (directory / 'transport-receipt.json').write_text(json.dumps(receipt, indent=2) + '\n')
            sys.stdout.buffer.write(answer)
            return 0
        time.sleep(.2)
    (directory / 'timeout.json').write_text(json.dumps({'status': 'timeout', **request}) + '\n')
    return 2


if __name__ == '__main__':
    try:
        raise SystemExit(main())
    except Exception as exc:
        print(type(exc).__name__ + ': ' + str(exc), file=sys.stderr)
        raise SystemExit(2)
