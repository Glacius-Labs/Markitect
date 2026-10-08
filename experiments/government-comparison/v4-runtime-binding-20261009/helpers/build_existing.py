"""One explicitly separately granted build; no fallback or toolchain download."""
import datetime as dt, hashlib, json, os, pathlib, subprocess, time
HERE=pathlib.Path(__file__).resolve().parents[1]
e=HERE/'evidence'
state=json.loads(pathlib.Path('C:/Users/Consiliari/Glacius Labs/Markitect/docs/design/government/coordination-state.json').read_text(encoding='utf-8-sig'))
grant=next(x for x in state['threads'] if x['name']=='Scientist')['evidence']['v4ClassicExistingToolchainBuildCorrectionGrant']
receipt=e/'existing-toolchain-build.json'
assert not receipt.exists(), 'separate one-attempt reservation already exists'
(e/'existing-toolchain-build-grant.json').write_text(json.dumps(grant,indent=2)+'\n')
go=pathlib.Path('C:/Users/Consiliari/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.windows-amd64/bin/go.exe')
root=pathlib.Path('C:/Users/Consiliari/Documents/Scientist-v4-runtime-binding-20261009')
repo=root/'classic-main';binary=root/'runtime/markitect.exe'
env=os.environ.copy();env['GOTOOLCHAIN']='local'
version=subprocess.check_output([str(go),'version'],env=env,text=True,timeout=10).strip()
assert version=='go version go1.27.1 windows/amd64'
assert subprocess.check_output(['git','rev-parse','HEAD'],cwd=repo,text=True).strip()==grant['sourceSha']
assert not subprocess.check_output(['git','status','--porcelain'],cwd=repo,text=True).strip()
now=dt.datetime.now(dt.timezone.utc);end=dt.datetime.fromisoformat(grant['notAfterUtc'].replace('Z','+00:00'))
timeout=min(600,(end-now).total_seconds());assert timeout>0
argv=[str(go),'build','-mod=readonly','-trimpath','-o',str(binary),'./cmd/markitect']
r={'grantKey':grant['key'],'reservedUtc':now.isoformat(),'status':'reserved-before-build','argv':argv,'cwd':str(repo),'timeoutSeconds':timeout,'environmentOverride':{'GOTOOLCHAIN':'local'},'toolchain':{'path':str(go),'version':version,'sha256':hashlib.sha256(go.read_bytes()).hexdigest()},'sourceSha':grant['sourceSha']}
receipt.write_text(json.dumps(r,indent=2)+'\n');start=time.monotonic()
try:
    p=subprocess.run(argv,cwd=repo,env=env,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,timeout=timeout)
    output=p.stdout;r.update(status='completed',exitCode=p.returncode)
except subprocess.TimeoutExpired as ex:
    output=ex.stdout or b'';r.update(status='timed-out',exitCode=None)
log=e/'existing-toolchain-build.log';log.write_bytes(output)
r.update(finishedUtc=dt.datetime.now(dt.timezone.utc).isoformat(),elapsedSeconds=time.monotonic()-start,logSha256=hashlib.sha256(output).hexdigest(),binary=None)
if r['exitCode']==0:
    r['binary']={'path':str(binary),'sha256':hashlib.sha256(binary.read_bytes()).hexdigest(),'bytes':binary.stat().st_size}
receipt.write_text(json.dumps(r,indent=2)+'\n')
print(json.dumps(r,indent=2))
