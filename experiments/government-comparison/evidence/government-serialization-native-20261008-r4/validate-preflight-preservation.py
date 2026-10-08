"""Read-only historical preservation check before the single R4 case."""
import hashlib
import json
from pathlib import Path
import sqlite3
import subprocess

AREA = Path(__file__).resolve().parent
ROOT = AREA.parents[1]
REPO = ROOT.parents[1]
BASE = 'a0bf4e48230b22fe7e1d5f1b80c7b56a0ffb6c76'
PREFIX = 'experiments/government-comparison/'
names = subprocess.check_output(['git','ls-tree','-r','--name-only',BASE,'--',
    PREFIX+'evidence/',PREFIX+'public/'],cwd=REPO,text=True).splitlines()
raw = subprocess.check_output(['git','cat-file','--batch'],cwd=REPO,
    input=('\n'.join(BASE+':'+n for n in names)+'\n').encode())
offset=0
for name in names:
    end=raw.index(b'\n',offset); size=int(raw[offset:end].split()[-1])
    assert (REPO/name).read_bytes()==raw[end+1:end+1+size],name
    offset=end+1+size+1
old=subprocess.check_output(['git','show',BASE+':'+PREFIX+'native-integration-handoff.md'],cwd=REPO)
assert (ROOT/'native-integration-handoff.md').read_bytes().endswith(old)
manifest=json.loads((ROOT/'evidence/native-integration/run-3/external-snapshot-manifest.json').read_bytes())
for pair in manifest:
    for item in pair.values():
        assert hashlib.sha256(Path(item['path']).read_bytes()).hexdigest()==item['sha256']
native=Path('C:/Users/Consiliari/Documents/Scientist-Probes/native-metadata-fixtures-20261008/native-starts.sqlite')
assert native.read_bytes()==(AREA/'historical-native-starts.sqlite').read_bytes()
db=sqlite3.connect(native.resolve().as_uri()+'?mode=ro',uri=True)
try:
    counts=db.execute('SELECT COUNT(*),SUM(reserved_seconds),SUM(finished IS NULL) FROM starts').fetchone()
finally:
    db.close()
assert counts==(11,1650,0)
value={'acceptedBase':BASE,'historicalEvidenceAndPublicFilesUnchanged':len(names),
       'priorHandoffSuffixByteIdentical':True,'r3ArchiveOriginalAndCopyPairsUnchanged':len(manifest),
       'nativeStarts':11,'reservedSessionSeconds':1650,'unfinishedNativeReservations':0,
       'nativeLedgerSha256':hashlib.sha256(native.read_bytes()).hexdigest(),'newExperimentalStarts':0}
with (AREA/'preservation-preflight.json').open('xb') as stream:
    stream.write((json.dumps(value,sort_keys=True,separators=(',',':'))+'\n').encode())
print(json.dumps(value))
