"""Close known returned roles, verify unchanged candidates, retain exact own files."""
from pathlib import Path
import sys,json,shutil,subprocess,hashlib
from datetime import datetime,timezone
E=Path(__file__).resolve().parent;PUBLIC=E.parent.parent/'public'
sys.path.insert(0,str(PUBLIC));import nest,app_ledger
grant=Path('C:/Users/Consiliari/Glacius Labs/Markitect/docs/design/government/design-roombook-sixth-enabled-grant-20261009.json')
g=nest.read(grant);state=Path(g['stateDir']);repo=Path(g['repo']);handle='/root/design_roombook_sixth_independent_final'
report=state/'assessment/roombook-s1-independent-assessment.md'
assert nest.sha(report)=='f97ba9b7a9efa58395f28b47fb9d5b9f778434ee2d5019108c6f3336b31923a9'
ledger=nest.read(state/'app-ledger.json');assert len(ledger['entries'])==2
assert ledger['entries'][0]['status']=='returned' and ledger['entries'][1]['handle']==handle
assert app_ledger.now()<app_ledger.date(ledger['entries'][1]['deadlineUtc'])
receipt={'terminalToolResultReceived':True,'receivedReceiptUtc':nest.utc(),'reportPath':str(report),'reportSha256':nest.sha(report),
 'reportedOwnJobsAllTerminal':True,'requestedModel':'gpt-6-luna','reasoning':'high','actualServingModel':None,'usage':None,'cost':None,
 'actualModelStartUtc':None,'actualModelEndUtc':None,'reportedActiveSeconds':81,
 'timingQualification':'81s is assessor report claim; rounded end time and no native execution receipt. Conservative reservation-to-receipt interval is retained separately.',
 'reportedAssessment':'R01 FAIL missing CLI; individual semantics NOT RUN; S2-S4 not assessed; no repairs/candidate changes'}
nest.write(E/'final-terminal-resource-receipt.json',receipt)
app_ledger.record(state,2,'returned',handle,receipt)
nest.write(state/'final-startup-gate.json',{'released':False,'reservation':2,'ownHandle':handle,'phase':'assessment','closedUtc':nest.utc(),'reason':'Assessment returned; trial terminal'})
binding=nest.read(state/'inner-binding.json');oldsha=nest.sha(state/'inner-binding.json')
runtime=repo/'.markitect/runtime.yaml';raw=runtime.read_bytes();assert raw.count(oldsha.encode())==2
(E/'terminal-runtime-before.yaml').write_bytes(raw);shutil.copyfile(state/'inner-binding.json',E/'terminal-inner-binding-before.json')
binding['executionAuthorized']=False;binding['activation']='Trial terminal; no further calls/slots/retries'
nest.write(state/'inner-binding.json',binding);newsha=nest.sha(state/'inner-binding.json')
after=raw.replace(oldsha.encode(),newsha.encode());assert after.replace(newsha.encode(),oldsha.encode())==raw
runtime.write_bytes(after);(E/'terminal-runtime-disabled.yaml').write_bytes(after)
shutil.copyfile(state/'inner-binding.json',E/'terminal-inner-binding-disabled.json')
app_ledger.close(state,'Sixth actual terminal: product-policy blocker S1; independent final returned; no further fullstart/refill')
record=nest.read(state/'state.json');record['status']='closed';record['closedUtc']=nest.utc();nest.write(state/'state.json',record)
freeze=record['freeze'];snapshot=record['snapshots'][0]
expected=nest.read(E.parent/'design-roombook-sixth-format-only-preparation-correction-20261009/prepared-case-inventory.json')
candidate_checks=[]
for folder in [Path(freeze['evaluationRepo']),Path(snapshot['evaluationRepo'])]:
    assert nest.git(folder,'rev-parse','HEAD')==freeze['mainCommit']
    assert not nest.git(folder,'status','--porcelain')
    actual=nest.inventory(folder);assert len(actual)==len(expected)
    for row in expected:assert actual[row['path']]['sha256']==row['sha256'],row['path']
    candidate_checks.append({'path':str(folder),'head':freeze['mainCommit'],'clean':True,'exactOriginalPreparedFiles':len(expected),'unchanged':True})
assert nest.git(repo,'rev-parse','main')==freeze['mainCommit']
assert nest.git(repo,'rev-parse','HEAD')==freeze['headCommit']
dirty=nest.git(repo,'status','--porcelain');assert dirty=='M .markitect/runtime.yaml',dirty
for source in [PUBLIC/'app_ledger.py',PUBLIC/'nest.py',PUBLIC/'inner_runtime_budget.py']:
    assert nest.sha(source)==nest.sha(state/'runtime-assets'/source.name)
shutil.copyfile(report,E/'independent-final-report.md')
nest.write(E/'candidate-unchanged-verification.json',candidate_checks)
nest.write(E/'closed-app-ledger.json',nest.read(state/'app-ledger.json'))
nest.write(E/'closed-state.json',nest.read(state/'state.json'))
nest.write(E/'terminal-resource-config-delta.json',{'bindingBeforeSha256':oldsha,'bindingAfterSha256':newsha,
 'runtimeBeforeSha256':nest.sha(E/'terminal-runtime-before.yaml'),'runtimeAfterSha256':nest.sha(runtime),
 'originalMutableCaseDirtyStatus':dirty,'scope':'Only post-freeze resource metadata in original mutable checkout; both immutable candidates untouched'})
packet=E/'private-state-packet'
for p in sorted(state.rglob('*')):
    if p.is_file() and '.git' not in p.relative_to(state).parts:
        target=packet/p.relative_to(state);target.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(p,target)
        assert nest.sha(p)==nest.sha(target)
ledger=nest.read(state/'app-ledger.json');closed=app_ledger.now();start=app_ledger.date(ledger['startedUtc'])
terminal={'status':'CLOSED sixth actual; S1 product-policy blocker and R01 failure; no replacement',
 'closedUtc':closed.isoformat(),'firstReservationUtc':ledger['startedUtc'],'trialWallSecondsThroughClosure':(closed-start).total_seconds(),
 'absoluteRootEndUtc':g['notAfterUtc'],'originalCellEndUtc':(start+__import__('datetime').timedelta(seconds=7200)).isoformat(),
 'globalFullStartsConsumed':6,'globalMaxFullStarts':6,'actualThisGrant':1,'ledgerEntriesConsumed':2,
 'outerImplementationStarts':1,'innerLedgerStarts':0,'helperStarts':0,'freshIndependentFinalStarts':1,
 'allEntriesTerminalReturned':all(e['status']=='returned' for e in ledger['entries']),'ledgerClosed':ledger['status']=='closed',
 'nativeProductModelRoleCalls':'0 reported by primary and 0 inner ledger entries; cooperative measurement, not provider telemetry',
 'actualSource':g['measuredPrototypeSource'],'binarySha256':g['binarySha256'],'adapterSha256':g['genuineAdapterSha256'],
 'wrapperSha256':g['wrapperSha256'],'frameworkSource':g['publicSourcePin'],'preparedCommit':g['preparedCommit'],'preparedTree':g['preparedTree'],
 'mainCommit':freeze['mainCommit'],'featureHead':freeze['headCommit'],'featureNotMerged':True,'implementationFreezeClean':freeze['dirtyStatus']=='',
 'postFreezeOriginalMutableDirty':dirty,'postFreezeDirtyCause':'Resource-only final-parent/terminal-disable metadata; frozen raw/main unaffected',
 'retainedStations':['S1'],'S1R01':'FAIL missing app.py on frozen main; detailed semantics NOT RUN',
 'S2':'NOT RUN','S3':'NOT RUN; actual team criterion not reached','S4':'NOT RUN; rename not reached',
 'observedPrimaryReportedBlocker':'project briefings/readiness: briefing bundle is invalid: incomplete definition identity',
 'independentFinalReportSha256':nest.sha(report),'assessorReportedActiveSeconds':81,
 'assessorConservativeReservationToTerminalReceiptSeconds':(app_ledger.date(receipt['receivedReceiptUtc'])-app_ledger.date(ledger['entries'][1]['reservedUtc'])).total_seconds(),
 'assessorTimingQualification':'No native exact start/end or serving proof; report interval claim and conservative receipt interval distinct.',
 'assessmentChecks':'One existing S1 public harness attempted six checks and direct missing-entrypoint CLI; all share launcher failure. No app code/tests executed.',
 'assessmentHarnessReportingLimit':'JSON decode precedes exit/stderr check; six reported parse errors are not six independent semantic failures.',
 'generatedViewFinding':'Some present common files listed absent in initial copied owner projection. Proven snapshot drift; causal source/runtime attribution unknown.',
 'knownOwnJobs':'Primary and final reported every known own command job terminal; both returned. No global inventory or escaped-provider containment proof.',
 'requestedProfile':'gpt-6-luna/high/fork none for both actual App roles; native inner read-only/tools-limited genuine CLI differs from normal App but was not reached',
 'servingModel':None,'tokens':None,'cost':None,'humanTime':None,'humanAcceptance':None,
 'setupConservativeFreshSeconds':963,'priorOwnerSetupSecondsSeparatelyRetained':1385.854584,'studySetup900SecondsPass':False,
 'qualityEffortComparison':'Exploratory alpha actual with unequal setup effort; no full fair efficiency/effort winner',
 'rawActorCliOutputs':'Not exported separately; copied handoff/worklog and own state artifacts retained, original actor tool outputs remain in task transcript. No reproduction/canary.',
 'sourceMechanicsQualification':'Final exceptional pipe-drain path NOT RUN/not exact-final independent subagent review; real adapter/provider compatibility was not reached.',
 'noProductRepairOrPrivateFinalFeedbackToImplementer':True,'noMainReleaseReadinessOrHumanSuccessClaim':True}
nest.write(E/'terminal-summary.json',terminal)
nest.write(E/'terminal-retained-file-manifest.json',[{'path':p.relative_to(E).as_posix(),'bytes':p.stat().st_size,'sha256':nest.sha(p)}
 for p in sorted(E.rglob('*')) if p.is_file() and p.name!='terminal-retained-file-manifest.json'])
print(json.dumps({'closedUtc':closed.isoformat(),'trialWallSeconds':terminal['trialWallSecondsThroughClosure'],
 'ledgerEntries':len(ledger['entries']),'terminalSummarySha256':nest.sha(E/'terminal-summary.json'),
 'manifestSha256':nest.sha(E/'terminal-retained-file-manifest.json'),'manifestFiles':len(nest.read(E/'terminal-retained-file-manifest.json'))}))
