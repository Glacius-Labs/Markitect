import json, hashlib, importlib.util, os, shutil, subprocess, sys, time
from datetime import datetime, timezone
from pathlib import Path
root=Path('C:/Users/Consiliari/.codex/worktrees/government-scientist/Markitect')
packet=root/'experiments/work-item-comparison/luna-nest-20261009'
primary=Path('C:/Users/Consiliari/Glacius Labs/Markitect/docs/design/government')
activation=primary/'conventional-roombook-explicit-workspace-root-activation-20261009.json'
grant=primary/'conventional-roombook-explicit-workspace-enabled-grant-20261009.json'
out=packet/'evidence/conventional-roombook-explicit-workspace-actual-20261009'
base=Path('C:/Users/Consiliari/Documents/Luna-Work-Item-Nests-20261009')
state=base/'state/roombook-conventional-explicit-workspace'
home=base/'homes/roombook-conventional-explicit-workspace'
exe=Path('C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/9691020b546a15b2/codex.exe')
def utc(): return datetime.now(timezone.utc).isoformat()
def sha(p): return hashlib.sha256(p.read_bytes()).hexdigest()
def write(p,v): p.write_text(json.dumps(v,indent=2,ensure_ascii=False)+'\n',encoding='utf8',newline='\n')
assert sha(activation)=='fe061ed89850801fcd1a8fa7c645559052e9d9d29b51615a43c8a4fe2013d26c'
assert sha(grant)=='c83bb57027f265dd82a69f17f247cd53da6c4987f434261635b46b3be7907fcc'
assert sha(packet/'public/nest.py')=='45241de9e157394cf2dd83412ebd3d86ce83764f1b791447030e5d82f320d635'
assert subprocess.check_output(['git','-C',str(root),'rev-parse','HEAD'],text=True).strip()=='76a63bdeb0e740241db20ddbd08d4547154acd84'
assert not subprocess.check_output(['git','-C',str(root),'status','--porcelain']).strip()
settings=json.loads(grant.read_text(encoding='utf8'))
now=datetime.now(timezone.utc); end=datetime.fromisoformat(settings['notAfterUtc'].replace('Z','+00:00'))
assert (end-now).total_seconds()>=7200
assert not out.exists(); out.mkdir()
shutil.copyfile(activation,out/'root-activation.json'); shutil.copyfile(grant,out/'execution-grant.json')
spec=importlib.util.spec_from_file_location('nest',packet/'public/nest.py'); nest=importlib.util.module_from_spec(spec);spec.loader.exec_module(nest)
s=nest.read(state/'state.json'); assert s['status']=='prepared' and not s['sessions']
nest.admission(s,settings,exe,'implementation')
setup=json.loads((state/'ordinary-setup-receipt.json').read_text(encoding='utf-8-sig'))
assert setup['accountMaterialCopiedOpaque'] is True
assert (now-datetime.fromisoformat(setup['startedUtc'])).total_seconds()<=900
shutil.copyfile(state/'ordinary-setup-receipt.json',out/'setup-receipt.json')
env=os.environ.copy();env['CODEX_HOME']=str(home);env['PYTHONDONTWRITEBYTECODE']='1'
cmd=[sys.executable,'-B',str(packet/'public/nest.py'),'drive',str(state),str(grant),str(exe)]
with (state/'driver.stdout.txt').open('wb') as stdout,(state/'driver.stderr.txt').open('wb') as stderr:
 proc=subprocess.Popen(cmd,cwd=root,env=env,stdout=stdout,stderr=stderr,creationflags=subprocess.CREATE_NEW_PROCESS_GROUP if os.name=='nt' else 0,start_new_session=os.name!='nt')
 write(out/'driver-launch.json',{'driverPid':proc.pid,'driverSupervisorPid':os.getpid(),'driverStartedUtc':utc(),'argv':cmd,'grantSha256':sha(grant),'acceptedSourceSha':'76a63bdeb0e740241db20ddbd08d4547154acd84','nativeHome':str(home),'stateDir':str(state),'newMaxFullStarts':1,'globalStartsBefore':2,'nativeStartNotYetObserved':True})
 print(json.dumps({'driverPid':proc.pid,'supervisorPid':os.getpid(),'startedUtc':utc(),'stateDir':str(state)}),flush=True)
 stop=None
 try: proc.wait(timeout=min(7200,(end-datetime.now(timezone.utc)).total_seconds()))
 except subprocess.TimeoutExpired: stop=nest.stop_owned(proc)
 write(state/'driver-exit.json',{'exitCode':proc.returncode,'endedUtc':utc(),'ownedStop':stop})
 print(json.dumps({'driverExit':proc.returncode,'terminalUtc':utc(),'stop':stop}),flush=True)
 sys.exit(proc.returncode if proc.returncode is not None else 1)
