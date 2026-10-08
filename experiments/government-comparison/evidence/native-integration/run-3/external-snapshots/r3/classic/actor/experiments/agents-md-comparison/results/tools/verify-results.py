"""Check preserved study integrity; does not rerun or accept agent implementations."""
import hashlib
import json
import re
import zipfile
from pathlib import Path

import yaml

repo = Path(__file__).resolve().parents[4]
experiment = repo / 'experiments/agents-md-comparison'
results = experiment / 'results'


def sha(data):
    return hashlib.sha256(data).hexdigest()


def load(path):
    return json.loads(path.read_text(encoding='utf8'))


manifest = load(experiment / 'frozen-manifest.json')
assert sha((experiment / 'frozen-manifest.json').read_bytes()) == 'e085ec47e2da0a62e07bf1d526be5dab9a35c7c53cf9d5e7cde185cb31125c3e'
for record in manifest['frozenFiles']:
    assert sha((repo / record['path']).read_bytes()) == record['sha256'], record['path']

index = load(results / 'raw-evidence-index.json')
for record in index['files']:
    assert sha((results / record['preserved']).read_bytes()) == record['normalizedSha256'], record['preserved']

summary = load(results / 'summary.json')
expected = {
    't1-simple': (16, 21386, 2, 0, 10),
    't1-markitect': (21, 76291, 6, 0, 10),
    't2-simple': (16, 32275, 4, 0, 14),
    't2-markitect': (19, 96233, 7, 74, 13),
    't3-simple': (18, 27149, 3, 0, 10),
    't3-markitect': (25, 194573, 6, 3, 10),
}
assert {r['runId'] for r in summary['runs']} == set(expected)
for run in summary['runs']:
    assert not run['externalWriteDetected']
    assert not run['compiledInputChanges']
    evidence = load(results / 'runs' / run['runId'] / 'run-evidence.json')
    assert not evidence['fileReadLedger']['helperMutated']
    assert evidence['model'] is None and evidence['toolEvents'] is None and evidence['tokenUsage'] is None
    commands = run['operatorCommands']
    assert all(c['exitCode'] == 0 for c in commands)
    total = next(c['testCount'] for c in commands if c['name'] == 'linked-application-architecture-tests')
    measured = (run['readEvents'], run['meteredBytes'], len(run['manualChanges']), len(run['generatedChanges']), total)
    assert measured == expected[run['runId']], run['runId']
    archive = run['changedArtifactsArchive']
    archive_path = results / archive['path']
    assert sha(archive_path.read_bytes()) == archive['sha256']
    with zipfile.ZipFile(archive_path) as z:
        required = {'EVIDENCE-LICENSE.txt'}
        for change in run['manualChanges'] + run['generatedChanges']:
            if change['change'] != 'deleted':
                required.add(change['path'])
                assert sha(z.read(change['path'])) == change['after']['sha256'], change['path']
        assert set(z.namelist()) == required

impact = yaml.safe_load((results / 'runs/t2-markitect/provided-impact.yaml').read_text())
assert impact['directPolicySubjectCount'] == len(impact['directPolicySubjects']) == 2
assert impact['affectedCount'] == len(impact['affected']) == 69
assert len(impact['policyChanges']) == 4
failed = [c for c in impact['policyChanges'] if c['candidate']['status'] == 'failed']
assert len(failed) == 2
assert {c['subject'] for c in failed} == set(impact['directPolicySubjects'])
assert all(c['base']['status'] == 'not-applicable' for c in impact['policyChanges'])
for row in load(results / 'operator/policy-baseline/results.json'):
    assert row['exitCode'] == 1
for name, exits in [('t1-markitect', [1, 0]), ('t2-markitect', [0, 0]), ('t3-markitect', [0, 0, 0])]:
    assert [r['exitCode'] for r in load(results / f'runs/{name}/post-markitect/results.json')] == exits

r3 = load(results / 'operator/r3-whole-experiment-delta.json')
assert len(r3) == 14
assert all(x['root'].startswith('<R3>') for x in r3)
for path in results.rglob('*'):
    if path.is_file() and path.suffix != '.zip' and '__pycache__' not in path.parts:
        data = path.read_text(encoding='utf8')
        assert not re.search(r'[A-Z]:[\\/]+Users[\\/]+[^\\/<>]+[\\/]', data), path
print('Passed: frozen inputs, normalized evidence hashes, six exact changed-file archives, measured counts, failing-policy delta, passing checks and known-root audit boundaries.')
