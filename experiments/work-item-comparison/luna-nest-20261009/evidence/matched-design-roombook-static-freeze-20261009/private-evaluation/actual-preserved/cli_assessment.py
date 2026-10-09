import csv, json, subprocess, sys, tempfile
from pathlib import Path
PY = r'C:/Python313/python.exe'
ROOT = Path(r'C:/Users/Consiliari/Documents/Luna-Work-Item-Nests-20261009/state/roombook-conventional-oldschool-app/stations')
def call(repo, db, *args, expect=0):
    p = subprocess.run([PY, '-B', str(repo/'app.py'), '--db', str(db), *args], text=True, encoding='utf-8', capture_output=True)
    assert p.returncode == expect, (repo.name, args, p.returncode, p.stdout, p.stderr)
    assert p.stderr == '', (args, p.stderr)
    obj = json.loads(p.stdout)
    if expect == 2: assert set(obj) == {'error'} and obj['error']
    return obj

def check_common(n):
  repo=ROOT/f'S{n}'/'main'
  with tempfile.TemporaryDirectory(prefix=f'roombook-S{n}-') as tmp:
    d=Path(tmp)/'db.json'
    # Empty reads do not create state; each successful write is visible to a new process.
    r=call(repo,d,'list'); assert r.get('reservations',r.get('bookings')) == [] and not d.exists()
    a=call(repo,d,'book','--room',' Atlas ','--start','2026-10-09T10:00Z','--end','2026-10-09T11:00Z','--title',' Plan ')
    assert a == {'id':1,'room':'Atlas','start':'2026-10-09T10:00Z','end':'2026-10-09T11:00Z','title':'Plan','status':'active'}
    before=d.read_bytes()
    call(repo,d,'book','--room','Atlas','--start','2026-10-09T10:59Z','--end','2026-10-09T11:30Z','--title','overlap',expect=2); assert d.read_bytes()==before
    b=call(repo,d,'book','--room','Atlas','--start','2026-10-09T11:00Z','--end','2026-10-09T11:30Z','--title','next'); assert b['id']==2
    c=call(repo,d,'book','--room','atlas','--start','2026-10-09T10:15Z','--end','2026-10-09T10:45Z','--title','other room'); assert c['id']==3
    before=d.read_bytes()
    call(repo,d,'book','--room','Atlas','--start','2026-10-09T12:00+00:00','--end','2026-10-09T13:00Z','--title','offset',expect=2); assert d.read_bytes()==before
    call(repo,d,'book','--room','Atlas','--start','2026-02-30T12:00Z','--end','2026-02-30T13:00Z','--title','bad date',expect=2); assert d.read_bytes()==before
    listing=call(repo,d,'list'); key='bookings' if n==4 else 'reservations'; rows=listing[key]
    assert [x['id'] for x in rows]==[1,3,2],rows
    assert call(repo,d,'list','--room',' Atlas ')[key]==[rows[0],rows[2]]
    assert d.read_bytes()==before
    return repo

def check_s2(repo):
  with tempfile.TemporaryDirectory(prefix='roombook-S2-') as tmp:
    d=Path(tmp)/'db.json'
    call(repo,d,'book','--room','Z','--start','2026-10-09T10:00Z','--end','2026-10-09T10:30Z','--title','a')
    call(repo,d,'book','--room','A','--start','2026-10-09T10:00Z','--end','2026-10-09T11:00Z','--title','b')
    first=call(repo,d,'cancel','--id','1'); assert first['status']=='canceled'
    before=d.read_bytes(); second=call(repo,d,'cancel','--id','1'); assert second==first and d.read_bytes()==before
    call(repo,d,'cancel','--id','99',expect=2); assert d.read_bytes()==before
    assert call(repo,d,'summary')['rooms']==[{'room':'A','active':1,'minutes':60}]

def check_s3(repo):
  with tempfile.TemporaryDirectory(prefix='roombook-S3-') as tmp:
    d=Path(tmp)/'db.json'; out=Path(tmp)/'out.csv'
    x=call(repo,d,'book','--room',' Atlas ','--start','2026-10-09T10:00Z','--end','2026-10-09T11:00Z','--title','line,one\nline two')
    call(repo,d,'book','--room','Atlas','--start','2026-10-09T11:00Z','--end','2026-10-09T11:15Z','--title','two')
    call(repo,d,'book','--room','Other','--start','2026-10-09T10:00Z','--end','2026-10-09T10:30Z','--title','third')
    call(repo,d,'cancel','--id','2'); before=d.read_bytes()
    assert [r['id'] for r in call(repo,d,'list','--room','Atlas','--status','active').get('reservations',call(repo,d,'list','--room','Atlas','--status','active').get('bookings'))]==[1]
    assert [r['id'] for r in call(repo,d,'list','--room','Atlas','--status','canceled').get('reservations',call(repo,d,'list','--room','Atlas','--status','canceled').get('bookings'))]==[2]
    assert call(repo,d,'summary','--room',' unknown ')=={'rooms':[]}
    assert call(repo,d,'summary','--room',' Atlas ')=={'rooms':[{'room':'Atlas','active':1,'minutes':60}]}
    assert call(repo,d,'export','--csv',str(out))=={'exported':3}
    assert d.read_bytes()==before
    rows=list(csv.reader(out.open(encoding='utf-8',newline='')))
    assert rows[0]==['id','room','start','end','title','status'] and rows[1][4]=='line,one\nline two' and rows[3][5]=='canceled'
    call(repo,d,'export','--csv',str(d),expect=2); assert d.read_bytes()==before

def check_s4(repo):
  with tempfile.TemporaryDirectory(prefix='roombook-S4-') as tmp:
    d=Path(tmp)/'db.json'
    prior={'reservations':[{'id':7,'room':'Atlas','start':'2026-10-09T10:00Z','end':'2026-10-09T11:00Z','title':'old','status':'active'}]}
    raw=json.dumps(prior,indent=2).encode()+b'\n'; d.write_bytes(raw)
    assert call(repo,d,'list')=={'bookings':prior['reservations']}; assert d.read_bytes()==raw
    assert call(repo,d,'list','--legacy','--room',' Atlas ','--status','active')=={'reservations':prior['reservations']}; assert d.read_bytes()==raw
    assert call(repo,d,'summary')=={'rooms':[{'room':'Atlas','active':1,'minutes':60}]}
    z=call(repo,d,'book','--room','Atlas','--start','2026-10-09T11:00Z','--end','2026-10-09T11:30Z','--title','new'); assert z['id']==8
    stored=json.loads(d.read_text()); assert set(stored)=={'reservations'} and len(stored['reservations'])==2
    before=d.read_bytes(); call(repo,d,'list'); assert d.read_bytes()==before
    assert call(repo,d,'export','--csv',str(Path(tmp)/'out.csv'))=={'exported':2}

if __name__=='__main__':
  for n in range(1,5):
    repo=check_common(n); print(f'S{n} common: PASS')
    if n>=2: check_s2(repo); print(f'S{n} S2 behavior: PASS')
    if n>=3: check_s3(repo); print(f'S{n} S3 behavior: PASS')
    if n==4: check_s4(repo); print('S4 rename/storage compatibility: PASS')



