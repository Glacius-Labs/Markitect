import hashlib
import json
import re
import shutil
import zipfile
from pathlib import Path

import yaml

repo = Path.cwd()
r2 = Path((repo / '.artifacts/comparison-run-root.txt').read_text().strip())
r3 = Path((repo / '.artifacts/comparison-task3-root.txt').read_text().strip())
out = repo / '.artifacts/comparison-durable-staging'
out.mkdir(exist_ok=True)
r1 = Path((repo / '.artifacts/comparison-run-root-r1.txt').read_text().strip())
roots = {'R2': r2, 'R3': r3, 'R1': r1, 'MARKITECT': repo}
for i, root in enumerate(json.loads((r2 / 'operator/prepared-manifest.json').read_text())['auditRoots']):
    path = Path(root)
    if not any(path.is_relative_to(base) for base in roots.values()):
        roots['AUDIT-REPO-' + str(i)] = path
raw_index = []


def digest(data):
    return hashlib.sha256(data).hexdigest()


def normalized(value):
    if isinstance(value, dict):
        return {normalized(k): normalized(v) for k, v in value.items()}
    if isinstance(value, list):
        return [normalized(v) for v in value]
    if isinstance(value, str):
        if value.lstrip().startswith('{'):
            try:
                return json.dumps(normalized(json.loads(value)), ensure_ascii=False, indent=2)
            except (ValueError, TypeError):
                pass
        for name, root in sorted(roots.items(), key=lambda item: -len(str(item[1]))):
            for source in [str(root), str(root).replace('\\', '/'), str(root).replace('\\', '\\\\')]:
                value = re.sub(re.escape(source), '<' + name + '>', value, flags=re.I)
        # Known unrelated audit roots are represented by stable study-local aliases.
        value = re.sub(r'[A-Z]:\\Users\\[^\\]+\\\.codex\\worktrees\\([0-9a-z]+)\\Markitect', r'<AUDIT-WORKTREE-\1>', value, flags=re.I)
        value = re.sub(r'[A-Z]:\\Users\\[^\\]+\\\.codex\\AGENTS.md', '<COMMON-PARENT>/AGENTS.md', value, flags=re.I)
        value = re.sub(r'[A-Z]:\\Users\\[^\\]+\\AppData\\Local\\Temp\\markitect-comparison-reference-eval-[a-f0-9]+', '<REFERENCE-EVAL>', value, flags=re.I)
    return value


def write_json(path, data):
    target = out / path
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(json.dumps(normalized(data), ensure_ascii=False, indent=2) + '\n', encoding='utf8', newline='\n')


def preserve(source, destination):
    if not source.exists():
        return
    raw = source.read_bytes()
    label = next((n + '/' + source.relative_to(r).as_posix() for n, r in roots.items() if source.is_relative_to(r)), source.name)
    target = out / destination
    target.parent.mkdir(parents=True, exist_ok=True)
    if source.suffix == '.json':
        write_json(destination, json.loads(raw))
    elif source.suffix == '.jsonl':
        target.write_text(''.join(json.dumps(normalized(json.loads(line)), ensure_ascii=False) + '\n' for line in raw.decode('utf8').splitlines() if line), encoding='utf8', newline='\n')
    else:
        target.write_text(normalized(raw.decode('utf8')).replace('\r\n', '\n').replace('\r\n', '\n'), encoding='utf8', newline='\n')
    raw_index.append({'rawLocationAlias': label, 'rawBytes': len(raw), 'rawSha256': digest(raw), 'preserved': destination, 'normalizedSha256': digest(target.read_bytes()), 'kind': 'presentation-normalized; raw bytes remain in operator area'})


def category(path):
    if path.startswith('.telemetry/'):
        return 'telemetry'
    if path.startswith(('docs/markitect/', 'schema/', '.agents/')):
        return 'generated'
    return 'manual'


summary = {'schemaVersion': 1, 'classification': 'D: overall inconclusive; specific feasible governance capability, no measured cost advantage', 'integration': json.loads((repo / 'experiments/agents-md-comparison/integration.json').read_text()), 'runs': [], 'limits': ['One pair per task, no randomization or repeats', 'Fresh collaboration agents, backend model identity/tool/model/token counters unavailable', 'Read helper is partial instrumentation', 'Audit covers persistent known roots, not all OS writes or transient mutations', 'Task 3 relocated after auto-review denial; common parent guidance differs from Tasks 1/2', 'Task 2 old/new policy history supplied only to B', 'No database, runtime registration, human acceptance, or productivity measurement']}
for study, root, ids in [('R2', r2, ['t1-simple', 't1-markitect', 't2-simple', 't2-markitect']), ('R3', r3, ['t3-simple', 't3-markitect'])]:
    prepared = json.loads((root / 'operator/prepared-manifest.json').read_text())
    for run_id in ids:
        run_path = root / 'operator/runs' / run_id
        if not (run_path / 'run-evidence.json').exists():
            continue
        d = json.loads((run_path / 'run-evidence.json').read_text())
        prefix = f'runs/{run_id}'
        for name in ['run-evidence.json', 'agent-report.txt', 'reads.jsonl', 'workspace.diff', 'workspace.status', 'input-prompt.txt', 'external-state.json']:
            preserve(run_path / name, f'{prefix}/{name}')
        evaluation = root / 'operator/evaluation' / run_id / 'evaluation.json'
        preserve(evaluation, f'{prefix}/evaluation.json')
        record = {'runId': run_id, 'study': study, 'agentIdentity': d['agentIdentity'], 'frozenManifestSha256': d['frozenManifestSha256'], 'candidateCommit': d['candidateCommitBeforeAgent'], 'externalWriteDetected': d['externalWriteDetected'], 'readEvents': len(d['fileReadLedger']['records']), 'meteredBytes': sum(x['bytes'] for x in d['fileReadLedger']['records']), 'manualChanges': [], 'generatedChanges': [], 'telemetryChanges': [], 'assetChanges': d['frozenWorkspaceAssetChanges'], 'compiledInputChanges': d['compiledInputChanges'], 'operatorCommands': []}
        workspace = Path(d['workspace'])
        snapshots = out / prefix / 'changed-artifacts.zip'
        with zipfile.ZipFile(snapshots, 'w', compression=zipfile.ZIP_DEFLATED) as archive:
            for delta in d['workspaceDelta']:
                key = {'manual': 'manualChanges', 'generated': 'generatedChanges', 'telemetry': 'telemetryChanges'}[category(delta['path'])]
                record[key].append({k: v for k, v in delta.items() if k != 'root'})
                if category(delta['path']) != 'telemetry' and delta['change'] != 'deleted':
                    data = (workspace / delta['path']).read_bytes()
                    assert digest(data) == delta['after']['sha256'], delta['path']
                    info = zipfile.ZipInfo(delta['path'], date_time=(1980, 1, 1, 0, 0, 0))
                    info.compress_type = zipfile.ZIP_DEFLATED
                    archive.writestr(info, data)
            archive.writestr(zipfile.ZipInfo('EVIDENCE-LICENSE.txt', date_time=(1980, 1, 1, 0, 0, 0)), (repo / 'experiments/agents-md-comparison/guidance/architecture-check/LICENSE').read_bytes())
        record['changedArtifactsArchive'] = {'path': f'{prefix}/changed-artifacts.zip', 'sha256': digest(snapshots.read_bytes()), 'bytes': snapshots.stat().st_size, 'meaning': 'exact changed-file bytes, including new untracked files that ordinary git diff omits'}
        if evaluation.exists():
            e = json.loads(evaluation.read_text())
            for c in e['commands']:
                tests = re.search(r'(?:Test Run Successful|Passed!)[\s\S]*?Total tests:\s*(\d+)', c['output'])
                record['operatorCommands'].append({'name': c['name'], 'exitCode': c['exitCode'], 'rawOutputSha256': c['stdoutAndStderrSha256'], 'testCount': int(tests.group(1)) if tests else None})
        for c in d['compiledInputs']:
            preserve(workspace / c['output'], f'{prefix}/provided-{Path(c["output"]).name}')
        if d['arm'] == 'markitect':
            c = yaml.safe_load((workspace / '.agent-input/context.yaml').read_text())
            record['contextInputs'] = [{k: x[k] for k in ['key', 'path', 'hash', 'reason', 'via', 'role', 'packageName', 'packageVersion'] if k in x} for x in c['inputs']]
            record['contextBytes'] = (workspace / '.agent-input/context.yaml').stat().st_size
            post = root / 'operator/post-markitect' / run_id
            if post.exists():
                for f in post.iterdir():
                    if f.is_file():
                        preserve(f, f'{prefix}/post-markitect/{f.name}')
        summary['runs'].append(record)

for alias, root in [('R2', r2), ('R3', r3)]:
    for name in ['prepared-manifest.json', 'whole-experiment-delta.json', 'r3-protocol.json', 'setup-inventory.json', 'whole-audit-provenance.json', 'reviewer-task1-addendum.txt', 'reviewer-task2.txt', 'reviewer-task3.txt']:
        preserve(root / 'operator' / name, f'operator/{alias.lower()}-{name}')
for folder in ['policy-baseline', 'preparation/t1-markitect', 'helper-smoke']:
    directory = r2 / 'operator' / folder
    for f in directory.rglob('*'):
        if f.is_file() and f.suffix in ['.txt', '.json', '.jsonl', '.yaml']:
            preserve(f, 'operator/' + f.relative_to(r2 / 'operator').as_posix())
blocked = r2 / 'operator/runs/t3-simple'
for name in ['run-evidence.json', 'agent-report.txt', 'reads.jsonl', 'workspace.diff']:
    preserve(blocked / name, 'excluded/task3-r2-permission/' + name)

for f in (r1 / 'operator/runs/t1-simple').glob('*'):
    if f.is_file() and f.name in ['external-state.json', 'input-prompt.txt']:
        preserve(f, 'excluded/task1-r1-meter/' + f.name)
write_json('excluded-runs.json', {'r1Task1': {'status': 'interrupted after read-helper bug; unscored', 'preRunAuditSha256': digest((r1 / 'operator/runs/t1-simple/pre-run-audit.json').read_bytes()), 'rawRootAlias': 'R1'}, 'r2Task3': {'status': 'blocked by auto-review before source edits; unscored', 'sourceChanges': 0, 'rawRootAlias': 'R2/operator/runs/t3-simple'}, 'r3Preflight': {'status': 'Git-normalization and ownership setup failures before either agent; not performance evidence', 'preservedDirectories': ['MARKITECT/.artifacts/comparison-task3-r3-preflight-git-normalization', 'MARKITECT/.artifacts/comparison-task3-r3-preflight-ownership', 'MARKITECT/.artifacts/comparison-task3-r3-preflight-byte-status', 'MARKITECT/.artifacts/comparison-task3-r3-preflight-index-refresh']}})
write_json('summary.json', summary)
write_json('raw-evidence-index.json', {'schemaVersion': 1, 'normalization': 'Known absolute workspace roots replaced by aliases; original raw hashes are not normalized hashes. No full OS trace was collected.', 'files': raw_index})
print(json.dumps([{'id': r['runId'], 'reads': r['readEvents'], 'bytes': r['meteredBytes'], 'manual': len(r['manualChanges']), 'generated': len(r['generatedChanges']), 'outside': r['externalWriteDetected']} for r in summary['runs']], indent=2))
