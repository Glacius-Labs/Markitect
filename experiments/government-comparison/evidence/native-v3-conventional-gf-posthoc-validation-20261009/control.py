"""Separate supplemental admission; no original reservation is reused."""
import argparse, datetime as dt, hashlib, json, pathlib
ROOT=pathlib.Path(__file__).resolve().parent
def now(): return dt.datetime.now(dt.timezone.utc)
def write(p,v):
    t=p.with_suffix(p.suffix+'.tmp'); t.write_text(json.dumps(v,indent=2)+'\n',encoding='utf8'); t.replace(p)
def reserve(role,prompt):
    p=ROOT/'ledger.json'; l=json.loads(p.read_text()); t=now()
    assert t < dt.datetime.fromisoformat(l['notAfterUtc'].replace('Z','+00:00'))
    assert (ROOT/'freeze.json').exists() and len(l['activations'])<2
    assert not any(x['status'] in ['reserved','running'] for x in l['activations'])
    expected=['public-review','independent-evaluation'][len(l['activations'])]; assert role==expected
    end=min(t+dt.timedelta(seconds=600),dt.datetime.fromisoformat(l['notAfterUtc'].replace('Z','+00:00')))
    record={'id':f'posthoc-{len(l["activations"])+1:02d}','role':role,'reservedUtc':t.isoformat(),'deadlineUtc':end.isoformat(),
            'status':'reserved','actor':None,'promptPath':str(pathlib.Path(prompt).resolve()),'promptSha256':hashlib.sha256(pathlib.Path(prompt).read_bytes()).hexdigest(),'usage':None}
    l['activations'].append(record); write(p,l); return record
def update(id,status,actor=None,result=None):
    p=ROOT/'ledger.json'; l=json.loads(p.read_text()); r=next(x for x in l['activations'] if x['id']==id)
    r.update(status=status,observedUtc=now().isoformat())
    if actor:r['actor']=actor
    if result:r.update(resultPath=str(pathlib.Path(result).resolve()),resultSha256=hashlib.sha256(pathlib.Path(result).read_bytes()).hexdigest())
    write(p,l); return r
if __name__=='__main__':
    a=argparse.ArgumentParser(); a.add_argument('action',choices=['reserve','update']); a.add_argument('--role');a.add_argument('--prompt');a.add_argument('--id');a.add_argument('--status');a.add_argument('--actor');a.add_argument('--result');x=a.parse_args()
    print(json.dumps(reserve(x.role,x.prompt) if x.action=='reserve' else update(x.id,x.status,x.actor,x.result),indent=2))
