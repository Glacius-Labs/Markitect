"""Literal stdin/stdout transport to the Scientist-owned native dispatcher.

No model call, synthetic response, parser emulation or semantic repair occurs
here. Main owns Invocation normalization and response validation/receipts.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import sys
import time

def digest(raw):
    return hashlib.sha256(raw).hexdigest()

def exclusive(path,raw):
    with Path(path).open('xb') as f:
        f.write(raw);f.flush();os.fsync(f.fileno())

def publish(path,raw):
    path=Path(path);temp=path.with_name(path.name+'.pending')
    exclusive(temp,raw)
    if path.exists():
        raise RuntimeError('refuse publication overwrite')
    os.rename(temp,path)

def transfer(raw,root,policy,clock=time.time,sleeper=time.sleep):
    # Only minimal routing metadata is inspected, never candidate semantics.
    if not raw or len(raw)>32*1024*1024:
        raise RuntimeError('invocation-size-bound')
    invocation=json.loads(raw)
    run=invocation.get('runId')
    if not isinstance(run,str) or not run or any(x not in '0123456789abcdef' for x in run):
        raise RuntimeError('unsafe-run-directory')
    if invocation.get('request',{}).get('role') not in policy['roles']:
        raise RuntimeError('role-not-reserved')
    root=Path(root)
    if not root.is_absolute() or not root.is_dir():
        raise RuntimeError('owned-mailbox-unavailable')
    # A serialized request counter charges failed starts too; stale lock refuses.
    with (root/'reservation.lock').open('x'):
        ledger=root/'requests.json'
        prior=json.loads(ledger.read_text()) if ledger.exists() else []
        if len(prior)>=policy['maxRequests']:
            raise RuntimeError('mailbox-request-quota-exhausted')
        prior.append({'runId':run,'inputSha256':digest(raw),'reservedUtcEpoch':clock()})
        replacement=ledger.with_suffix('.pending')
        exclusive(replacement,json.dumps(prior).encode())
        os.replace(replacement,ledger)
    (root/'reservation.lock').unlink()
    call=root/run
    call.mkdir(exist_ok=False)
    exclusive(call/'invocation.raw.json',raw)
    metadata={'runId':run,'nonce':invocation.get('nonce'),'inputDigest':invocation.get('inputDigest'),
              'invocationSha256':digest(raw),'bridgePid':os.getpid(),'publishedEpoch':clock()}
    exclusive(call/'request-metadata.json',json.dumps(metadata,sort_keys=True).encode())
    publish(call/'ready.json',json.dumps(metadata,sort_keys=True).encode())
    end=min(policy['notAfterEpoch'],clock()+policy['bridgeSeconds'])
    monotonic_end=time.monotonic()+max(0,end-clock())
    while clock()<end and time.monotonic()<monotonic_end:
        delivery=call/'delivery.json'
        if delivery.is_file():
            d=json.loads(delivery.read_text(encoding='utf-8'))
            if d.get('invocationSha256')!=digest(raw) or d.get('nativeActorTerminal') is not True \
                    or not d.get('nativeActorId') or not d.get('terminalReceiptSha256'):
                raise RuntimeError('missing-genuine-terminal-binding')
            response=call/'response.raw.json'
            returned=response.read_bytes()
            if not returned or len(returned)>policy['maxResponseBytes'] or digest(returned)!=d.get('responseSha256'):
                raise RuntimeError('raw-response-binding-invalid')
            terminal=(call/'native-terminal.json').read_bytes()
            if digest(terminal)!=d['terminalReceiptSha256']:
                raise RuntimeError('terminal-receipt-binding-invalid')
            publish(call/'forwarded.json',json.dumps({'responseSha256':digest(returned),
                    'nativeActorId':d['nativeActorId'],'forwardedEpoch':clock()},sort_keys=True).encode())
            return returned
        sleeper(.05)
    raise TimeoutError('own-native-delivery-deadline')

def main():
    p=argparse.ArgumentParser()
    p.add_argument('--mailbox',required=True);p.add_argument('--policy',required=True)
    a=p.parse_args()
    policy=json.loads(Path(a.policy).read_text(encoding='utf-8-sig'))
    raw=sys.stdin.buffer.read(32*1024*1024+1)
    returned=transfer(raw,a.mailbox,policy)
    sys.stdout.buffer.write(returned);sys.stdout.buffer.flush()

if __name__=='__main__':
    try:main()
    except Exception as exc:
        print('native transport incomplete: '+type(exc).__name__+': '+str(exc),file=sys.stderr)
        raise SystemExit(2)
