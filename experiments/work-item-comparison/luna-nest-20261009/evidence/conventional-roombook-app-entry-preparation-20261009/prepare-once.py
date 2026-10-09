"""Fresh same-byte App case and one offline ledger test batch. Zero App/CLI model calls."""
import hashlib,json,os,shutil,subprocess,sys,time
from datetime import datetime,timezone
from pathlib import Path
HERE=Path(__file__).resolve().parent; PACKET=HERE.parents[1]; ROOT=PACKET.parents[2]
sys.path.insert(0,str(PACKET/'public')); import nest; import app_ledger
primary=Path('C:/Users/Consiliari/Glacius Labs/Markitect/docs/design/government')
grant=primary/'conventional-roombook-app-entry-preparation-grant-20261009.json'
reservation=primary/'conventional-roombook-app-native-trial-reservation-20261009.json'
base=Path('C:/Users/Consiliari/Documents/Luna-Work-Item-Nests-20261009')
case='roombook-conventional-app-native'; repo=base/'cells'/case; state=base/'state'/case
def sha(p): return hashlib.sha256(p.read_bytes()).hexdigest()
def utc(): return datetime.now(timezone.utc).isoformat()
def write(name,obj): (HERE/name).write_text(json.dumps(obj,indent=2,ensure_ascii=False)+'\n',encoding='utf8',newline='\n')
assert not (HERE/'mechanical-reservation.json').exists(),'One charge only; no retry'
assert sha(grant)=='f31ba9eeeb6a74f3e64d6feb54150a54e059f41b69ca7b7f0537c5a6df3dce30'
assert sha(reservation)=='f8624ce3384b211586b71036f47ca9bf7bfa399a0011e618ee3b264187fc4643'
assert datetime.now(timezone.utc)<datetime.fromisoformat('2026-10-09T06:47:20+00:00')
assert subprocess.check_output(['git','-C',str(ROOT),'rev-parse','HEAD'],text=True).strip()=='fdabf6215da3d64576f0d7e5fa3987dc33341f8a'
shutil.copyfile(grant,HERE/'preparation-grant.json');shutil.copyfile(reservation,HERE/'trial-reservation.json')
prepared=nest.cell('roombook','conventional',base/'seeds/roombook',repo,state); app_ledger.initialize(state)
old=nest.read(PACKET/'evidence/conventional-roombook-current-native-preparation-20261009/bindings.json')
assert nest.inventory(repo)==old['preparedFileHashes'] and prepared['preparedTree']==old['preparedTree']
assert prepared['seedCommit']==old['seedCommit']=='71acaae99c826d0dce0ade81a5a84ab0aee6cfef'
source={p:sha(PACKET/p) for p in ['public/app_ledger.py','public/app-entry.md','tests/test_app_ledger.py']}
write('mechanical-reservation.json',{'batch':1,'reservedUtc':utc(),'maxWallSeconds':180,'scope':'Four focused offline App ledger tests only','sourceHashes':source,'modelOrReviewerOrActualStarts':0})
start=time.monotonic(); began=utc(); argv=[sys.executable,'-B','-m','unittest','discover','-s',str(PACKET/'tests'),'-p','test_app_ledger.py','-v']; env=os.environ.copy();env['PYTHONDONTWRITEBYTECODE']='1'
with (HERE/'mechanical.stdout.txt').open('wb') as stdout,(HERE/'mechanical.stderr.txt').open('wb') as stderr:
 proc=subprocess.Popen(argv,cwd=ROOT,env=env,stdout=stdout,stderr=stderr,creationflags=subprocess.CREATE_NEW_PROCESS_GROUP if os.name=='nt' else 0,start_new_session=os.name!='nt')
 stop=None
 try:proc.wait(timeout=150)
 except subprocess.TimeoutExpired:stop=nest.stop_owned(proc)
elapsed=time.monotonic()-start; passed=proc.returncode==0 and stop is None and elapsed<=180
write('mechanical-receipt.json',{'status':'PASS' if passed else 'FAIL','batch':1,'argv':argv,'pid':proc.pid,'startedUtc':began,'endedUtc':utc(),'wallSeconds':elapsed,'exitCode':proc.returncode,'ownedStop':stop,'tests':4,'modelCalls':0,'providerCalls':0,'reviewers':0,'actualStarts':0})
assert passed,'Terminal mechanical failure; no refill'
executionBinding=(f'\nArbeitsprojekt: {repo}. Lies dessen eigene AGENTS.md und öffentliche Projektregeln; normale Dateiaufrufe richten sich auf dieses Projekt. '
 'Kein Zugriff auf andere Projekte, Studien, Chats oder Memory. Frischer App-Kontext, Luna High für alle Rollen. '
 f'Budgetledger: {state}/app-ledger.json; öffentliche Reservierungsmechanik: {PACKET}/public/app_ledger.py. '
 'Vor jedem eigenen Helfer-/Rollenstart dort reservieren; nur native collaboration-Tools mit fork_turns=none, model=gpt-6-luna, reasoning_effort=high. '
 'Maximal4gleichzeitige gemesseneAgenten/Depth2/72Starts gesamt; konkrete eigene Deadline aus Reservierung einhalten, keine persistenten Hintergrundjobs. '
 'Nur eigene Handles erfassen und koordinieren. Tatsächlich angeerbtesCWD/Instruktionsherkunft/Toolumfang und bekannte Fremdexposition ehrlich als Ausführungsreceipt erfassen, keine privaten Inhalte kopieren. '
 'Bei realer Toolpolicyblockade Arbeit terminal melden, kein Transportwechsel oder Neustart. Keine erfundenen Token-/Serving-/Kostenreceipts.')
tool={'tool':'collaboration.spawn_agent','arguments':{'task_name':'app_conventional_roombook_primary','fork_turns':'none','model':'gpt-6-luna','reasoning_effort':'high','message':nest.PROMPT+executionBinding},'cwdArgument':None,'executionAuthorized':False}
write('prospective-primary-call.json',tool)
bindings={'preparedCommit':prepared['preparedCommit'],'preparedTree':prepared['preparedTree'],'seedCommit':prepared['seedCommit'],'repo':str(repo),'stateDir':str(state),'mainCommit':nest.git(repo,'rev-parse','main'),'dirtyStatus':nest.git(repo,'status','--porcelain'),'nineInputBytesUnchanged':True,'preparedFileHashes':nest.inventory(repo),'sourceHashes':source,'cliRunnerUnchangedSha256':sha(PACKET/'public/nest.py'),'canonicalPromptSha256':hashlib.sha256(nest.PROMPT.encode()).hexdigest(),'continuationPromptSha256':hashlib.sha256(nest.CONTINUE.encode()).hexdigest(),'finalAssessorPublicPromptSha256':sha(PACKET/'public/evaluation/assessor-prompt.txt'),'appLedgerSha256':sha(state/'app-ledger.json'),'actualStarts':0,'model':'gpt-6-luna','reasoning':'high','tool':'collaboration.spawn_agent','forkTurns':'none','noCwdParameter':True,'normalAppPermissions':'Inherited App tools/protection; no profile/global config changes','inheritedCwdBasis':'App task environment supplied C:/Users/Consiliari/Glacius Labs/Markitect. Actual subagent CWD/instructions not observed before start.','parentTurnHistoryProvided':False,'privateHistoryOrMemoryIntentionallyProvided':False,'inheritedCommonInstructions':'Unknown until actual; fork none does not attest sanitization of system/developer/workspace instructions','osIsolationClaimed':False,'maxTotalMeasuredAppAgentStarts':72,'maxConcurrentMeasuredAgents':4,'maxDepth':2,'enforcement':'Cooperative case ledger, not proven platform hard enforcement','usage':None,'servingModel':None,'cost':None,'nativeCliOrAccountSetupCalls':0}
write('bindings.json',bindings)
proposal=nest.read(reservation);proposal.update(executionAuthorized=False,model='gpt-6-luna',reasoning='high',preparedCommit=prepared['preparedCommit'],repo=str(repo),stateDir=str(state),sourceBasis='fdabf6215da3d64576f0d7e5fa3987dc33341f8a',appLedgerSourceSha256=source['public/app_ledger.py'],freshAppPrimaryCall='prospective-primary-call.json',status='Disabled; requires exact Root activation and concrete Design host-job release',notAfterUtc='2026-10-09T09:12:43Z',accountSetup='Existing App account/tool configuration; no credential copying or account probe')
write('activation-proposal.json',proposal)
write('continuation-anchor.json',{'status':'PREPARED / execution disabled','sourceBasis':'fdabf6215da3d64576f0d7e5fa3987dc33341f8a','newModelOrReviewerOrActualStarts':0,'globalFullStarts':3,'next':'Handoff once; await Root activation and actual specific host-job release, then full7200s remains under09:12:43Z. No readiness/CLIprobe.'})
print(json.dumps({'status':'PASS','preparedCommit':prepared['preparedCommit'],'tree':prepared['preparedTree'],'samePreparedInputPaths':len(bindings['preparedFileHashes']),'mechanicalSeconds':elapsed,'actualStarts':0},indent=2))
