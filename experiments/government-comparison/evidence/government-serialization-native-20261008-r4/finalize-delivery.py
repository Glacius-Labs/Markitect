"""Final additive R4 preservation and archive verification; no experimental effects."""
import hashlib
import json
from pathlib import Path
import sqlite3
import subprocess

AREA=Path(__file__).resolve().parent
ROOT=AREA.parents[1]
REPO=ROOT.parents[1]
BASE='a0bf4e48230b22fe7e1d5f1b80c7b56a0ffb6c76'
SOURCE='90a5c77d2cc7cd8103f74c83997e4d35ab838d21'
FREEZE_COMMIT='1b21c32b4a8988393922a7d3cdecc512cc174cdd'
PREFIX='experiments/government-comparison/'
NATIVE=Path('C:/Users/Consiliari/Documents/Scientist-Probes/native-metadata-fixtures-20261008/native-starts.sqlite')

def pin(path):
    return {'path':str(path.resolve()),'sha256':hashlib.sha256(path.read_bytes()).hexdigest()}

def git_equal(names,revision):
    raw=subprocess.check_output(['git','cat-file','--batch'],cwd=REPO,
        input=('\n'.join(revision+':'+n for n in names)+'\n').encode())
    offset=0
    for name in names:
        end=raw.index(b'\n',offset);size=int(raw[offset:end].split()[-1])
        assert (REPO/name).read_bytes()==raw[end+1:end+1+size],name
        offset=end+1+size+1

def rows(path,sql):
    db=sqlite3.connect(path.resolve().as_uri()+'?mode=ro',uri=True)
    try:return db.execute(sql).fetchall()
    finally:db.close()

oldnames=subprocess.check_output(['git','ls-tree','-r','--name-only',BASE,'--',
    PREFIX+'evidence/',PREFIX+'public/'],cwd=REPO,text=True).splitlines()
git_equal(oldnames,BASE)
old_handoff=subprocess.check_output(['git','show',FREEZE_COMMIT+':'+PREFIX+'native-integration-handoff.md'],cwd=REPO)
assert (ROOT/'native-integration-handoff.md').read_bytes().endswith(old_handoff)
source_names=['runtime/dispatch.py','runtime/government_roles.py','runtime/native_controller.py',
    'runtime/native_fixture_budget.py','runtime/test_native_r4_budget.py',
    'runtime/test_native_r4_chain.py','runtime/test_native_r4_checkpoint.py',
    'prepare-native-r4.py','run-native-integration-r4.py']
git_equal([PREFIX+n for n in source_names],SOURCE)
freeze=json.loads((AREA/'preflight-freeze.json').read_bytes())
for item in freeze['inputs']:assert pin(Path(item['path']))==item
pair_counts={}
for name in ['preflight-external-snapshot-manifest.json','terminal-external-snapshot-manifest.json']:
    pairs=json.loads((AREA/name).read_bytes())
    for pair in pairs:
        for item in pair.values():assert pin(Path(item['path']))==item
    pair_counts[name]=len(pairs)
r3=json.loads((ROOT/'evidence/native-integration/run-3/external-snapshot-manifest.json').read_bytes())
advanced=[]
for pair in r3:
    assert pin(Path(pair['snapshot']['path']))==pair['snapshot']
    if pin(Path(pair['original']['path']))!=pair['original']:
        advanced.append(Path(pair['original']['path']).resolve())
assert advanced==[NATIVE.resolve()]
sql='SELECT product,label,argv,claimed,finished,reserved_seconds,receipt FROM starts ORDER BY product,label'
old=rows(AREA/'historical-native-starts.sqlite',sql);current=rows(NATIVE,sql)
new=[r for r in current if r[1].startswith('government-native-serialization-r4/')]
assert [r for r in current if not r[1].startswith('government-native-serialization-r4/')]==old
assert len(new)==1 and new[0][1]=='government-native-serialization-r4/queue' and new[0][4] is not None
assert (len(current),sum(r[5] for r in current),sum(r[4] is None for r in current))==(12,1800,0)
corrections='SELECT source_key,correction_sha,source_snapshot_sha,basis_sha FROM corrections ORDER BY source_key'
old_corrections=rows(AREA/'historical-native-starts.sqlite',corrections)
assert [r for r in rows(NATIVE,corrections) if r[0]!='government-serialization-native-20261008-r4']==old_corrections
assert (AREA/'independent-post-run-review.md').is_file()
assert json.loads((AREA/'slot-release.json').read_bytes())['releaseDeclaration'] is True
summary=json.loads((AREA/'terminal-summary.json').read_bytes())
assert summary['newConsumption']=={'nativeStarts':1,'wrapperAttempts':3,'deterministicDelegates':3,'reservedSessionSeconds':150}
paths={p for p in AREA.rglob('*') if p.is_file() and '__pycache__' not in p.parts}
paths.update(ROOT/n for n in [*source_names,'native-integration-handoff.md'])
value={'grantKey':AREA.name,'sourceCommit':SOURCE,'freezeCommit':FREEZE_COMMIT,
    'historicalEvidenceAndPublicGitFilesUnchanged':len(oldnames),
    'priorHandoffSuffixByteIdentical':True,'runtimeSourcesUnchangedAfterExecution':True,
    'frozenInputsUnchanged':len(freeze['inputs']),'verifiedArchivePairs':pair_counts,
    'r3ArchivedCopiesUnchanged':len(r3),'r3OriginalCurrentNativeLedgerAdvancedOnly':str(NATIVE.resolve()),
    'historicalNativeRowsUnchanged':len(old),'historicalCorrectionsUnchanged':len(old_corrections),
    'newConsumption':summary['newConsumption'],'cumulativeConsumption':summary['cumulativeConsumption'],
    'outcome':'incomplete','resumeStarts':0,'repairOrRetry':False,'slotReleasedByScientist':True,
    's1Complete':False,'semanticAcceptance':False,'humanAcceptance':False,
    'files':[pin(p) for p in sorted(paths,key=str)]}
with (AREA/'delivery-manifest.json').open('xb') as stream:
    stream.write((json.dumps(value,sort_keys=True,separators=(',',':'))+'\n').encode())
print(json.dumps({k:value[k] for k in ['historicalEvidenceAndPublicGitFilesUnchanged',
    'frozenInputsUnchanged','verifiedArchivePairs','historicalNativeRowsUnchanged','newConsumption','cumulativeConsumption']}))
