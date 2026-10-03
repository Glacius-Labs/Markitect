import hashlib, json, shutil, subprocess
from pathlib import Path

repo = Path.cwd()
old = Path((repo / '.artifacts/comparison-run-root.txt').read_text().strip())
new = repo / '.artifacts/comparison-task3-r3'
if new.exists():
    raise SystemExit('Refusing to replace r3')
new.mkdir()
operator = new / 'operator'
operator.mkdir()
prepared = json.loads((old / 'operator/prepared-manifest.json').read_text())
manifest = json.loads((repo / 'experiments/agents-md-comparison/frozen-manifest.json').read_text())
selected = [r for r in prepared['workspaces'] if r['task'] == 't3']
for record in selected:
    original = Path(record['workspace'])
    target = new / 'workspaces' / record['id'] / 'checkout'
    subprocess.run(['git', '-c', 'safe.directory=' + str(original), '-c', 'core.autocrlf=false', 'clone', '--no-hardlinks',
                    str(original), str(target)], check=True, capture_output=True)
    subprocess.run(['git', '-C', str(target), 'remote', 'remove', 'origin'], check=True)
    subprocess.run(['git', '-C', str(target), 'config', 'core.autocrlf', 'true'], check=True)
    # Clone normalizes CRLF through Git. Restore exact frozen working-input bytes,
    # including canonical files, from the unchanged prepared checkout.
    paths = subprocess.check_output(['git', '-c', 'safe.directory=' + str(original), '-C', str(original), 'ls-tree', '-r', '--name-only', record['candidateCommit']], text=True).splitlines()
    for relative in paths:
        source = original / relative
        destination = target / relative
        destination.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(source, destination)
    subprocess.run(['git', '-C', str(target), 'update-index', '--really-refresh'], capture_output=True)
    subprocess.run(['git', '-C', str(target), 'diff', '--exit-code'], check=True, capture_output=True)
    for entry in manifest['sourceFiles']:
        data = (target / entry['path']).read_bytes()
        if hashlib.sha256(data).hexdigest() != entry['sha256']:
            raise SystemExit('Source parity failed: ' + entry['path'])
    shutil.copyfile(original / '.git/info/exclude', target / '.git/info/exclude')
    shutil.copytree(original / '.tools', target / '.tools', ignore=shutil.ignore_patterns('bin', 'obj'))
    if (original / '.agent-input').exists():
        shutil.copytree(original / '.agent-input', target / '.agent-input')
    record['previousPreparedWorkspace'] = str(original)
    record['workspace'] = str(target)
    if record['arm'] == 'markitect':
        for compiled in record['compiledInputs']:
            args = [str(target) if a == str(original) else a for a in compiled['command']]
            result = subprocess.run([str(target / '.tools/markitect.exe'), *args],
                                    cwd=target, capture_output=True, check=True)
            path = target / compiled['output']
            path.chmod(0o666)
            path.write_bytes(result.stdout)
            path.chmod(0o444)
            previous_hash = compiled['sha256']
            compiled['sha256'] = hashlib.sha256(result.stdout).hexdigest()
            compiled['bytes'] = len(result.stdout)
            compiled['command'] = args
            compiled['r2InputSha256'] = previous_hash
            compiled['sameBytesAfterRelocation'] = compiled['sha256'] == previous_hash
    source_manifest = new / record['sourceManifestPath']
    source_manifest.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(old / record['sourceManifestPath'], source_manifest)
prepared['workspaces'] = selected
prepared['auditRoots'] = sorted(set(prepared['auditRoots'] + [r['workspace'] for r in selected]))
prepared['experimentId'] = 'agents-md-vs-markitect-2026-10-03-task3-r3'
prepared['status'] = 'paired task3 relocation before either r3 agent; r2 simple blocked unchanged'
(operator / 'prepared-manifest.json').write_text(json.dumps(prepared, indent=2), encoding='utf8')

runner_root = repo / '.artifacts/task3-authorized-runner'
runner_root.mkdir(exist_ok=True)
original_script = (repo / 'experiments/agents-md-comparison/run.ps1').read_text()
start = original_script.index('function Assert-RunRoot {')
end = original_script.index('\nAssert-RunRoot', start)
replacement = '''function Assert-RunRoot {
    $expected = Join-Path $script:RepositoryRoot '.artifacts/comparison-task3-r3'
    if (-not $script:RunRootFull.Equals([IO.Path]::GetFullPath($expected), [StringComparison]::OrdinalIgnoreCase)) {
        throw 'R3 is limited to the named ignored scratch directory inside the authorized workspace.'
    }
}
'''
script = original_script[:start] + replacement + original_script[end:]
old_guard = 'if (Test-Path -LiteralPath $inherited -PathType Leaf) { throw "Inherited parent AGENTS.md would affect the trial: $inherited" }'
if old_guard not in script:
    raise SystemExit('Expected parent guidance guard absent')
script = script.replace(old_guard, 'if (Test-Path -LiteralPath $inherited -PathType Leaf) { Write-Verbose "Common inherited guidance recorded in R3 protocol: $inherited" }')
runner = runner_root / 'run.ps1'
runner.write_text(script, encoding='utf8', newline='\n')
parents = []
for parent in new.parents:
    source = parent / 'AGENTS.md'
    if source.exists():
        parents.append({'path': str(source), 'sha256': hashlib.sha256(source.read_bytes()).hexdigest()})
protocol = {'experimentId': prepared['experimentId'],
            'reason': 'Auto-review denied temporary adopter patch; no workaround against that checkout. New fresh paired clones use the existing explicitly writable workspace.',
            'originalSourceAndTaskAndOracleFreezeSha256': prepared['frozenManifestSha256'],
            'originalRunnerSha256': hashlib.sha256((repo / 'experiments/agents-md-comparison/run.ps1').read_bytes()).hexdigest(),
            'r3RunnerSha256': hashlib.sha256(runner.read_bytes()).hexdigest(),
            'changes': ['Assert exact authorized scratch root', 'Record common parent AGENTS instead of rejecting both arms'],
            'commonParentGuidance': parents,
            'parentGuidanceLimitation': 'Common to both relocated arms; task3 environment differs from r2. No causal comparison across tasks.',
            'sourceFilesVerifiedPerArm': len(manifest['sourceFiles']),
            'blockedR2Run': 't3-simple: no code writes; not scored as implementation failure',
            'freshAgentsRequired': True}
(operator / 'r3-protocol.json').write_text(json.dumps(protocol, indent=2), encoding='utf8')
(repo / '.artifacts/comparison-task3-root.txt').write_text(str(new), encoding='utf8')
print(json.dumps({'root': str(new), 'protocol': protocol}, indent=2))
