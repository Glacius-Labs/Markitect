"""Three read-only calls under the original four-call quota; no role execution."""
import datetime as dt, hashlib, json, pathlib, subprocess, time
packet=pathlib.Path(__file__).resolve().parents[1];e=packet/'evidence';lp=e/'ledger.json'
ledger=json.loads(lp.read_text());root=pathlib.Path(ledger['externalRoot']);repo=root/'classic-main'
build=json.loads((e/'existing-toolchain-build.json').read_text());binary=pathlib.Path(build['binary']['path'])
assert hashlib.sha256(binary.read_bytes()).hexdigest()==build['binary']['sha256']
sha=build['sourceSha'];config='examples/canonical-projection/canonical.yaml'
for action in ['modules','model','context']:
    assert len(ledger['metadataCalls'])<4
    now=dt.datetime.now(dt.timezone.utc);end=dt.datetime.fromisoformat(ledger['notAfterUtc'].replace('Z','+00:00'))
    timeout=min(60,(end-now).total_seconds());assert timeout>0
    argv=[str(binary),'canonical','--repo',str(repo),'--config',config,'--action',action,'--revision',sha]
    if action=='context':argv+=['--api-version','commerce.example.org/v1','--kind','UseCase','--namespace','commerce','--name','create-order']
    r={'reservedUtc':now.isoformat(),'argv':argv,'cwd':str(repo),'timeoutSeconds':timeout,'status':'reserved-before-call','binarySha256':build['binary']['sha256']}
    ledger['metadataCalls'].append(r);lp.write_text(json.dumps(ledger,indent=2)+'\n');start=time.monotonic()
    try:
        p=subprocess.run(argv,cwd=repo,stdout=subprocess.PIPE,stderr=subprocess.PIPE,timeout=timeout)
        stdout,stderr=p.stdout,p.stderr;r.update(status='completed',exitCode=p.returncode)
    except subprocess.TimeoutExpired as ex:
        stdout,stderr=ex.stdout or b'',ex.stderr or b'';r.update(status='timed-out',exitCode=None)
    for name,data in [('stdout',stdout),('stderr',stderr)]:
        out=e/f'classic-{action}.{name}.log';out.write_bytes(data);r[name]={'path':str(out.relative_to(packet)),'sha256':hashlib.sha256(data).hexdigest()}
    r.update(finishedUtc=dt.datetime.now(dt.timezone.utc).isoformat(),elapsedSeconds=time.monotonic()-start)
    lp.write_text(json.dumps(ledger,indent=2)+'\n');print(action,r['status'],r['exitCode'])
    if r['exitCode']!=0:break
assert not subprocess.check_output(['git','status','--porcelain'],cwd=repo,text=True).strip()
