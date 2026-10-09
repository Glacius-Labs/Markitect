"""Case-local reservations for manually invoked native collaboration tools.

No model/tool invocation API. Bounds depend on cooperating App actors; no OS isolation.
"""
from datetime import datetime, timedelta, timezone
from pathlib import Path
import nest

def now(): return datetime.now(timezone.utc)
def date(value): return datetime.fromisoformat(value.replace('Z', '+00:00'))

def initialize(state_dir):
    state_dir=Path(state_dir)
    with nest.locked(state_dir):
        target=state_dir/'app-ledger.json'
        if target.exists(): raise ValueError('Existing ledger; no reset')
        nest.write(target, {'status':'prepared','startedUtc':None,'grantSha256':None,
                            'implementationClosed':False,'entries':[],
                            'usage':None,'servingModel':None,'cost':None,
                            'enforcement':'Cooperative native-tool reservations; not a sandbox'})

def reserve(state_dir, grant_path, kind, depth=1, parent=None, continuation_handle=None, at=None):
    """Reserve BEFORE spawn_agent/idle followup_task; failed calls still consume a slot."""
    state_dir=Path(state_dir); at=at or now()
    with nest.locked(state_dir):
        ledger=nest.read(state_dir/'app-ledger.json'); state=nest.read(state_dir/'state.json')
        grant=nest.read(grant_path); digest=nest.sha(grant_path)
        if not grant.get('executionAuthorized') or grant.get('model')!='gpt-6-luna' or grant.get('reasoning')!='high':
            raise ValueError('Exact enabled Luna High grant required')
        if grant.get('preparedCommit')!=state['preparedCommit'] or grant.get('repo')!=state['repo']:
            raise ValueError('Wrong prepared case binding')
        if ledger['grantSha256'] and ledger['grantSha256']!=digest: raise ValueError('Grant drift; no refill')
        if ledger['status']=='closed': raise ValueError('Terminal cell; no retry')
        if kind not in ('implementation','helper','assessment','assessment-helper'): raise ValueError('Unknown role')
        if depth not in (1,2): raise ValueError('Measured depth limited to two')
        entries=ledger['entries']; active=[e for e in entries if e['status'] in ('reserved','started')]
        if any(e['status']=='unknown-stop' for e in entries): raise ValueError('Unreconciled own actor stop')
        if len(active)>=4 or len(entries)>=72: raise ValueError('App concurrency/start envelope consumed')
        if depth==2 and not any(e['reservation']==parent and e in active and e['depth']==1 for e in entries):
            raise ValueError('An active own measured parent is required')
        final=kind in ('assessment','assessment-helper')
        if not ledger['startedUtc']:
            if kind!='implementation' or (date(grant['notAfterUtc'])-at).total_seconds()<7200:
                raise ValueError('Fresh primary needs full 7200-second window')
            if nest.git(state['repo'],'rev-parse','HEAD')!=state['preparedCommit'] or nest.git(state['repo'],'status','--porcelain'):
                raise ValueError('Prepared project drift')
            ledger['startedUtc']=at.isoformat(); ledger['grantSha256']=digest; ledger['status']='active'
        start=date(ledger['startedUtc']); station=state['stationIndex']
        end=min(date(grant['notAfterUtc']),start+timedelta(seconds=6600 if final else 5400))
        if not final: end=min(end,start+timedelta(seconds=1350*(station+1)))
        if final:
            if not state.get('freeze') or (kind=='assessment' and active): raise ValueError('Assessment requires freeze and terminal own handles')
            if kind=='assessment-helper' and not any(e in active and e['kind']=='assessment' and e['reservation']==parent for e in entries):
                raise ValueError('Final helper requires active own assessor')
            if kind=='assessment' and any(e['kind']=='assessment' for e in entries): raise ValueError('One fresh final assessor only')
            prior=next((e for e in entries if e['kind']=='assessment'),None)
            end=min(end,date(prior['reservedUtc'])+timedelta(seconds=1200) if prior else at+timedelta(seconds=1200))
        elif ledger['implementationClosed'] or state.get('freeze'):
            raise ValueError('Implementation is terminal')
        if kind=='implementation':
            outer=[e for e in entries if e['kind']=='implementation']
            if len(outer)>=8 or sum(e['stationIndex']==station for e in outer)>=2:
                raise ValueError('Outer/station entry envelope consumed')
        if (end-at).total_seconds()<=30: raise ValueError('Phase window exhausted; no extension')
        if continuation_handle and not any(e.get('handle')==continuation_handle and e['status']=='returned' and e['kind']==kind for e in entries):
            raise ValueError('Continuation must name own returned handle')
        entry={'reservation':len(entries)+1,'kind':kind,'depth':depth,'parent':parent,
               'stationIndex':station,'reservedUtc':at.isoformat(),'deadlineUtc':end.isoformat(),
               'status':'reserved','handle':None,'continuationHandle':continuation_handle,
               'model':'gpt-6-luna','reasoning':'high','forkTurns':'none' if not continuation_handle else 'own context',
               'usage':None,'servingModel':None,'cost':None,'toolResult':None}
        entries.append(entry); nest.write(state_dir/'app-ledger.json',ledger)
        state['status']='running'; nest.write(state_dir/'state.json',state)
        return entry

def record(state_dir, reservation, status, handle=None, receipt=None):
    if status not in ('started','returned','call-failed','interrupted','unknown-stop'): raise ValueError('Unknown outcome')
    state_dir=Path(state_dir)
    with nest.locked(state_dir):
        ledger=nest.read(state_dir/'app-ledger.json'); entry=next(e for e in ledger['entries'] if e['reservation']==reservation)
        if entry['status'] not in ('reserved','started'): raise ValueError('Entry already terminal')
        if handle and entry['handle'] and handle!=entry['handle']: raise ValueError('Own handle binding differs')
        if handle: entry['handle']=handle
        entry.update(status=status,observedUtc=now().isoformat(),toolResult=receipt)
        nest.write(state_dir/'app-ledger.json',ledger)
        state=nest.read(state_dir/'state.json')
        state['status']='unreconciled_stop' if any(e['status']=='unknown-stop' for e in ledger['entries']) else (
            'running' if any(e['status'] in ('reserved','started') for e in ledger['entries']) else 'returned')
        nest.write(state_dir/'state.json',state)

def close_implementation(state_dir, reason):
    state_dir=Path(state_dir)
    with nest.locked(state_dir):
        ledger=nest.read(state_dir/'app-ledger.json'); ledger.update(implementationClosed=True,implementationStopReason=reason)
        nest.write(state_dir/'app-ledger.json',ledger)

def close(state_dir, reason):
    state_dir=Path(state_dir)
    with nest.locked(state_dir):
        ledger=nest.read(state_dir/'app-ledger.json')
        if any(e['status'] in ('reserved','started','unknown-stop') for e in ledger['entries']): raise ValueError('Unreconciled own handles')
        ledger.update(status='closed',implementationClosed=True,closedUtc=now().isoformat(),terminalReason=reason)
        nest.write(state_dir/'app-ledger.json',ledger)
