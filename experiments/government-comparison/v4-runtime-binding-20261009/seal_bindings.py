"""Read-only source/hash reconciliation and additive binding records; no tests."""
import datetime as dt, hashlib, json, pathlib, re, subprocess, zipfile
P=pathlib.Path(__file__).resolve().parent;WT=P.parents[2];E=P/'evidence'
def sha(p):return hashlib.sha256(pathlib.Path(p).read_bytes()).hexdigest()
def save(path,x):path.write_text(json.dumps(x,indent=2)+'\n',encoding='utf8')
def git(repo,*args):return subprocess.check_output(['git','-C',str(repo),*args])
ledger=json.loads((E/'ledger.json').read_text());repo=pathlib.Path(ledger['classicCheckout']);root=pathlib.Path(ledger['externalRoot'])
build=json.loads((E/'existing-toolchain-build.json').read_text());source=build['sourceSha'];assert git(repo,'rev-parse','HEAD').decode().strip()==source
assert not git(repo,'status','--porcelain').strip();assert sha(build['binary']['path'])==build['binary']['sha256']
assert sha(build['toolchain']['path'])==build['toolchain']['sha256']
contract=['go.mod','go.sum','internal/host/agentexec/types.go','internal/host/agentexec/json.go','internal/host/agentexec/runner.go','internal/host/agentexec/README.md',
          'internal/host/canonical_controller_execution.go','internal/host/canonical_controller_verification.go','internal/host/canonical_inference.go','internal/host/canonical_goal_modeling.go',
          'internal/host/canonical_controller.go','internal/host/cli/cli_options.go','docs/canonical-projections.md','docs/usage.md','examples/classic-commerce/runtime.template.json']
fixture=git(repo,'ls-files','examples/canonical-projection').decode().splitlines()
paths=sorted(set(contract+fixture));bindings=[]
with zipfile.ZipFile(E/'classic-main-bound-source.zip','w',zipfile.ZIP_DEFLATED) as archive:
    for name in paths:
        p=repo/name;raw=p.read_bytes();blob=git(repo,'show',source+':'+name)
        bindings.append({'path':name,'workingSha256':sha(p),'gitSha256':hashlib.sha256(blob).hexdigest(),'gitBlob':git(repo,'rev-parse',source+':'+name).decode().strip(),'bytes':len(raw)})
        archive.writestr(name,raw)
modules_text=(E/'classic-modules.stdout.log').read_text()
pins=[{'name':m[0],'version':m[1],'digest':m[2]} for m in re.findall(r'^    - pin:\n        name: (.+)\n        version: (.+)\n        digest: (.+)',modules_text,re.M)]
assert len(pins)==4
binding={'sourceSha':source,'sourceTree':git(repo,'rev-parse','HEAD^{tree}').decode().strip(),'branch':git(repo,'branch','--show-current').decode().strip(),'sourceClean':True,
         'checkout':str(repo),'binary':build['binary'],'toolchain':build['toolchain'],'modulePins':pins,'sourceFiles':bindings,
         'sourceArchive':{'path':'evidence/classic-main-bound-source.zip','sha256':sha(E/'classic-main-bound-source.zip')},
         'inspectionConfig':{'path':'examples/canonical-projection/canonical.yaml','sha256':sha(repo/'examples/canonical-projection/canonical.yaml'),'selectedRevision':source},
         'goDependencies':{'module':'github.com/Glacius-Labs/Markitect','go':'1.27.1','requirements':{'go.yaml.in/yaml/v3':'v3.0.5'},'goModSha256':sha(repo/'go.mod'),'goSumSha256':sha(repo/'go.sum')},
         'metadataCalls':ledger['metadataCalls'],'actualRoleRuntimeConfig':None,'actualNativeRoleSupport':None,'commonProfileSupport':None,
         'limits':'Binary build and read-only fixture metadata/context only. No agentexec role, native bridge, compiler service, semantic trial or release acceptance.'}
save(E/'classic-main-binding.json',binding)
excerpts=[]
for name,start,end in [('internal/host/agentexec/types.go',110,114),('internal/host/agentexec/types.go',141,146),('internal/host/agentexec/runner.go',73,104),
                       ('internal/host/agentexec/runner.go',111,139),('internal/host/agentexec/json.go',199,220),('internal/host/agentexec/json.go',353,355),
                       ('internal/host/canonical_controller_execution.go',23,33),('internal/host/canonical_controller_execution.go',125,139),
                       ('internal/host/canonical_controller_verification.go',786,788),('internal/host/canonical_inference.go',160,162),
                       ('internal/host/canonical_goal_modeling.go',139,139),('internal/host/canonical_goal_modeling.go',293,293)]:
    lines=(repo/name).read_text().splitlines();excerpts.append({'path':name,'firstLine':start,'lastLine':end,'sourceWorkingSha256':sha(repo/name),'lines':lines[start-1:end]})
actual_roots=[];forbidden=[str(repo.resolve())]
for path in [P,root/'state',root/'runtime',root/'outputs',root/'logs']:
    assert all(not path.resolve().is_relative_to(pathlib.Path(x)) for x in actual_roots+forbidden)
save(E/'audit-root-binding.json',{'sourceSha':source,'scope':'inspected canonical Executor/Verifier/Inference/goal-modeling callers; role invocation not run',
     'actualInputRootsFromBoundCallers':actual_roots,'executorContextHasWorkspace':False,'executorAllowedRootsMeaning':'logical output target allowlist, not filesystem InputRoots',
     'immutableInputs':'full acquired source revision/model and explicit serialized request artifacts; command executable and declared runtime files fingerprinted independently',
     'conservativeAdditionalForbiddenRoots':forbidden,'boundHelperRoot':str(P/'helpers'),'boundOutputAndLogRoot':str(E),'dispatcherStateRoot':str(root/'state/dispatch'),
     'futureActorPaths':'<dispatcherStateRoot>/actors/<unique action ID>/{response.json,logs}',
     'noAuditNarrowing':True,'osIsolation':False,'actualNativeStopTimerProven':False,'osCancellation':None,'providerCancellation':None,'excerpts':excerpts})
old=P.parent/'v4-preparation-20261009';accepted=json.loads((old/'evidence/source-manifest.json').read_text());preserved=[]
for row in accepted['files']:
    path=old/row['path'];assert sha(path)==row['sha256'];rel=path.relative_to(WT).as_posix();assert git(WT,'show','c7d55105a1f0183d6517e263d15c0fadef7d4b2d:'+rel)==path.read_bytes();preserved.append({'path':rel,'sha256':sha(path)})
manifest=old/'evidence/source-manifest.json';rel=manifest.relative_to(WT).as_posix();assert git(WT,'show','c7d55105a1f0183d6517e263d15c0fadef7d4b2d:'+rel)==manifest.read_bytes();preserved.append({'path':rel,'sha256':sha(manifest)})
older=json.loads((old/'evidence/readonly-preservation-check.json').read_text())['retainedEvidenceFiles']
for row in older:assert sha(WT/row['path'])==row['sha256']
freeze=json.loads((P.parent/'evidence/native-codex-exploratory-v3-20261008/freeze.json').read_text());count=0
for name,h in freeze['files'].items():assert sha(name)==h;count+=1
for cell in freeze['cells']:
    for name,h in cell['supportPins'].items():assert sha(name)==h;count+=1
private=json.loads((old/'evidence/private-archive-manifest.json').read_text())
assert sha(old/'evidence/private-preparation.zip')==private['archiveSha256']
for row in private['files']:assert sha(pathlib.Path(private['sourceLocation'])/row['path'])==row['sha256']
save(E/'preservation-check.json',{'checkedUtc':dt.datetime.now(dt.timezone.utc).isoformat(),'acceptedV4FilesIncludingManifest':preserved,'olderEvidenceFiles':older,
     'originalV3PinChecks':count,'privateFilesCheckedWithoutExecution':len(private['files']),'privateArchiveSha256':private['archiveSha256'],'mutations':0})
proposal=old/'public/common-profile-proposal.json'
grant=json.loads((E/'authorization-grant.json').read_text())
cfg={'kind':'offline-preparation','key':grant['key'],'issued':dt.datetime.fromisoformat(grant['issuedUtc'].replace('Z','+00:00')).timestamp(),
     'end':dt.datetime.fromisoformat(grant['notAfterUtc'].replace('Z','+00:00')).timestamp(),'executionAllowed':False,'admissionComplete':False,'cells':[],
     'maxActivations':0,'maxConcurrent':4,'maxActiveCells':2,'profileSha256':sha(proposal),'profileStatus':'proposal only; actual common tool/role support not admitted',
     'actualInputRootsFromBoundClassicCallers':actual_roots,'conservativeAdditionalForbiddenRoots':forbidden,'classicBinarySha256':build['binary']['sha256']}
save(P/'dispatch-config.json',cfg)
admission=json.loads((old/'public/candidate-admission.json').read_text());admission['version']='v4-runtime-binding';admission['status']='NOT ADMITTED; binary/metadata and dispatch mechanics bound'
admission['classicMain'].update(binarySha256=build['binary']['sha256'],inspectionConfigSha256=binding['inspectionConfig']['sha256'],modulePins=pins,
     gap='Exact Main source/binary/inspection config and modules bound. Real common-profile agent runtime configuration and role support remain unbound; no native role launched.')
admission['dispatch']={'configSha256':sha(P/'dispatch-config.json'),'executionAllowed':False,'boundary':'helpers/native_boundary.py prepares exact spawn args, no tool invoked','mechanicalTests':42,'actualNativeTimerEnforcement':None}
save(P/'candidate-admission.json',admission)
communication=json.loads(pathlib.Path('C:/Users/Consiliari/Glacius Labs/Markitect/docs/design/government/coordination-state.json').read_text(encoding='utf-8-sig'))['communicationPolicy']
save(E/'human-callback-authorization.json',{k:communication[k] for k in ['effectiveUtc','authorizationSource','userQuote','callbackTargetThreadId','callbackHostId','events','unchangedReports','acknowledgmentLoops']})
ledger.update(status='closed-qualified-preparation-not-admitted',closedUtc=dt.datetime.now(dt.timezone.utc).isoformat(),separateBuildCorrectionReceipt='evidence/existing-toolchain-build.json',finalMechanicalTestsPassed=42,
              limits='No real native role/timer/cancellation, matched profile, Design adapter, trial or human acceptance. No further executions under these consumed mechanical quotas.')
save(E/'ledger.json',ledger)
print(json.dumps({'sourceFiles':len(bindings),'modulePins':len(pins),'acceptedV4Preserved':len(preserved),'olderEvidencePreserved':len(older),'originalV3PinChecks':count,'privateFilesPreserved':len(private['files']),'binarySha256':build['binary']['sha256']},indent=2))
