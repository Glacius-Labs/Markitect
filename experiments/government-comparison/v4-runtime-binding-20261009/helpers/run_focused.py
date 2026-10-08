"""Pre-reserve each of at most two mechanical test process invocations."""
import datetime as dt, hashlib, json, os, pathlib, subprocess, sys, time
p=pathlib.Path(__file__).resolve().parents[1];lp=p/'evidence/ledger.json';ledger=json.loads(lp.read_text())
assert len(ledger['testInvocations'])<2
now=dt.datetime.now(dt.timezone.utc);end=dt.datetime.fromisoformat(ledger['notAfterUtc'].replace('Z','+00:00'))
timeout=min(120,(end-now).total_seconds());assert timeout>0
argv=[sys.executable,'-m','unittest','discover','-s','helpers','-p','test_binding.py','-v']
row={'reservedUtc':now.isoformat(),'argv':argv,'cwd':str(p),'timeoutSeconds':timeout,'status':'reserved-before-test','kind':'synthetic-mechanics-no-native-callbacks'}
ledger['testInvocations'].append(row);lp.write_text(json.dumps(ledger,indent=2)+'\n')
env=os.environ.copy();env['PYTHONDONTWRITEBYTECODE']='1';start=time.monotonic()
try:
    r=subprocess.run(argv,cwd=p,env=env,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,timeout=timeout)
    raw=r.stdout;row.update(status='completed',exitCode=r.returncode)
except subprocess.TimeoutExpired as ex:
    raw=ex.stdout or b'';row.update(status='timed-out',exitCode=None)
log=p/'evidence'/f'focused-test-{len(ledger["testInvocations"])}.log';log.write_bytes(raw)
row.update(finishedUtc=dt.datetime.now(dt.timezone.utc).isoformat(),elapsedSeconds=time.monotonic()-start,log=str(log.relative_to(p)),logSha256=hashlib.sha256(raw).hexdigest())
lp.write_text(json.dumps(ledger,indent=2)+'\n');print(raw.decode('utf8',errors='replace'));print(json.dumps(row,indent=2))
