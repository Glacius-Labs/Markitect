"""Capture already closed own trial; no ledger restart or candidate mutation."""
from pathlib import Path
import sys,json,shutil
from datetime import timedelta
E=Path(__file__).resolve().parent;PUBLIC=E.parent.parent/'public'
sys.path.insert(0,str(PUBLIC));import nest,app_ledger
grant=Path('C:/Users/Consiliari/Glacius Labs/Markitect/docs/design/government/design-roombook-sixth-enabled-grant-20261009.json')
g=nest.read(grant);state=Path(g['stateDir']);repo=Path(g['repo']);record=nest.read(state/'state.json');ledger=nest.read(state/'app-ledger.json')
assert record['status']=='closed' and ledger['status']=='closed' and len(ledger['entries'])==2
assert all(e['status']=='returned' for e in ledger['entries'])
assert nest.read(state/'inner-binding.json')['executionAuthorized'] is False
assert not nest.read(state/'startup-gate.json')['released'] and not nest.read(state/'final-startup-gate.json')['released']
freeze=record['freeze'];snapshot=record['snapshots'][0];diagnostics=nest.read(E/'snapshot-byte-diagnostics.json')
assert len(diagnostics)==54 and all(d['gitMatchesPrepared'] and d['lineEndingEquivalent'] for d in diagnostics)
for row in diagnostics:assert nest.sha(Path(row['folder'])/row['path'])==row['workingSha256']
candidate_checks=[]
for folder in [Path(freeze['evaluationRepo']),Path(snapshot['evaluationRepo'])]:
    assert nest.git(folder,'rev-parse','HEAD')==freeze['mainCommit'] and not nest.git(folder,'status','--porcelain')
    candidate_checks.append({'path':str(folder),'head':freeze['mainCommit'],'clean':True,'preparedRawGitBindings':27,
                            'exactWorkingRawGitBindings':25,'workingRawGitCRLFOnlyDifferences':2,'unchangedSinceAssessment':True})
assert nest.git(repo,'rev-parse','main')==freeze['mainCommit'] and nest.git(repo,'rev-parse','HEAD')==freeze['headCommit']
dirty=nest.git(repo,'status','--porcelain');assert dirty=='M .markitect/runtime.yaml'
report=state/'assessment/roombook-s1-independent-assessment.md'
assert nest.sha(report)=='f97ba9b7a9efa58395f28b47fb9d5b9f778434ee2d5019108c6f3336b31923a9'
for p in [PUBLIC/'app_ledger.py',PUBLIC/'nest.py',PUBLIC/'inner_runtime_budget.py']:
    assert nest.sha(p)==nest.sha(state/'runtime-assets'/p.name)
shutil.copyfile(report,E/'independent-final-report.md')
nest.write(E/'candidate-unchanged-verification.json',candidate_checks)
nest.write(E/'closed-app-ledger.json',ledger);nest.write(E/'closed-state.json',record)
runtime=repo/'.markitect/runtime.yaml'
nest.write(E/'terminal-resource-config-delta.json',{'bindingBeforeSha256':nest.sha(E/'terminal-inner-binding-before.json'),
 'bindingAfterSha256':nest.sha(state/'inner-binding.json'),'runtimeBeforeSha256':nest.sha(E/'terminal-runtime-before.yaml'),
 'runtimeAfterSha256':nest.sha(runtime),'originalMutableCaseDirtyStatus':dirty,
 'scope':'Only post-freeze resource metadata in original mutable checkout; both immutable candidates unchanged'})
packet=E/'private-state-packet'
for p in sorted(state.rglob('*')):
    if p.is_file() and '.git' not in p.relative_to(state).parts:
        target=packet/p.relative_to(state);target.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(p,target)
        assert nest.sha(p)==nest.sha(target)
receipt=nest.read(E/'final-terminal-resource-receipt.json');start=app_ledger.date(ledger['startedUtc']);closed=app_ledger.now()
terminal={'status':'CLOSED sixth actual; S1 product-policy blocker/R01 failure; no replacement',
 'ledgerClosedUtc':ledger['closedUtc'],'captureClosedUtc':closed.isoformat(),'firstReservationUtc':ledger['startedUtc'],
 'trialWallSecondsThroughCapture':(closed-start).total_seconds(),'absoluteRootEndUtc':g['notAfterUtc'],
 'originalCellEndUtc':(start+timedelta(seconds=7200)).isoformat(),'globalFullStartsConsumed':6,'globalMaxFullStarts':6,
 'actualThisGrant':1,'ledgerEntriesConsumed':2,'outerImplementationStarts':1,'innerLedgerStarts':0,'helperStarts':0,'freshIndependentFinalStarts':1,
 'allEntriesTerminalReturned':True,'ledgerClosed':True,'nativeProductModelRoleCalls':'0 reported and 0 inner ledger entries; not hard provider telemetry',
 'actualSource':g['measuredPrototypeSource'],'binarySha256':g['binarySha256'],'adapterSha256':g['genuineAdapterSha256'],
 'wrapperSha256':g['wrapperSha256'],'frameworkSource':g['publicSourcePin'],'preparedCommit':g['preparedCommit'],'preparedTree':g['preparedTree'],
 'mainCommit':freeze['mainCommit'],'featureHead':freeze['headCommit'],'featureNotMerged':True,'implementationFreezeClean':freeze['dirtyStatus']=='',
 'postFreezeOriginalMutableDirty':dirty,'postFreezeDirtyCause':'Final-parent/terminal-disable resource metadata only; frozen raw/main unaffected',
 'retainedStations':['S1'],'S1R01':'FAIL missing app.py; detailed semantics NOT RUN','S2':'NOT RUN','S3':'NOT RUN; team not reached','S4':'NOT RUN; rename not reached',
 'primaryReportedBlocker':'project briefings/readiness: briefing bundle is invalid: incomplete definition identity',
 'independentFinalReportSha256':nest.sha(report),'assessorReportedActiveSeconds':81,
 'assessorConservativeReservationToReceiptSeconds':(app_ledger.date(receipt['receivedReceiptUtc'])-app_ledger.date(ledger['entries'][1]['reservedUtc'])).total_seconds(),
 'assessorTimingQualification':'Reported81s/rounded end are not exact native execution; conservative receipt interval separate.',
 'assessmentChecks':'PublicS1harness attempts6checks plus direct missing CLI. One shared launcher failure; no application or first-party tests executed.',
 'harnessReportingLimit':'JSON decode before exit/stderr check obscures missing entrypoint; six parse errors are not six independent behavior diagnoses.',
 'generatedViewFinding':'Initial copied owner projection lists present common files absent. Proven snapshot drift; product/runtime causal attribution unknown.',
 'snapshotQualification':'All54rawGit match original prepared; 50Working match rawGit, 4Working differ CRLF only. Assessor before/after Working unchanged. Original capture assertion failure retained, no edits/rescore/tests.',
 'knownOwnJobs':'Primary/final report all known jobs terminal; both returned. No global inventory or escaped-provider containment proof.',
 'requestedProfile':'Both actual Approles gpt-6-luna/high/forknone. Native inner CLI read-only/tools-limited difference declared but not reached.',
 'servingModel':None,'tokens':None,'cost':None,'humanTime':None,'humanAcceptance':None,
 'setupConservativeFreshSeconds':963,'priorOwnerSetupSecondsSeparatelyRetained':1385.854584,'studySetup900SecondsPass':False,
 'comparisonScope':'Exploratory alpha actual with unequal setup effort; no fair full-effort/efficiency winner.',
 'rawActorCliOutputs':'Not separately exported; copied messages, WORKLOG and own state retained. Original tool outputs in actor transcript, no reproduction.',
 'resourceMechanicsQualification':'Final pipe-drain path NOT RUN/not exact-final independent review; genuine adapter/provider compatibility not reached.',
 'noPrivateFinalFeedbackToImplementer':True,'noProductRepairOrMainReleaseReadinessHumanSuccessClaim':True}
nest.write(E/'terminal-summary.json',terminal)
nest.write(E/'terminal-retained-file-manifest.json',[{'path':p.relative_to(E).as_posix(),'bytes':p.stat().st_size,'sha256':nest.sha(p)}
 for p in sorted(E.rglob('*')) if p.is_file() and p.name!='terminal-retained-file-manifest.json'])
print(json.dumps({'captureClosedUtc':closed.isoformat(),'trialWallSecondsThroughCapture':terminal['trialWallSecondsThroughCapture'],
 'ledgerClosedUtc':ledger['closedUtc'],'terminalSummarySha256':nest.sha(E/'terminal-summary.json'),
 'manifestSha256':nest.sha(E/'terminal-retained-file-manifest.json'),'manifestFiles':len(nest.read(E/'terminal-retained-file-manifest.json'))}))
