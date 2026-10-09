import datetime
import difflib
import hashlib
import json
import shutil
import subprocess
import zipfile
from pathlib import Path

root = Path(__file__).parent
repo = Path(r'C:\Users\Consiliari\.codex\worktrees\model-driven-delivery\Markitect')
old = Path(r'C:\Users\Consiliari\AppData\Local\Temp\markitect-roombook-install-handoff-20261009')
baseline = '2b7b22bafb72de1d1ca3bf54e5870c281a171956'
source = 'bcd614a3bfcd335e0ca18850993742057103c092'
grant = Path(r'C:\Users\Consiliari\Glacius Labs\Markitect\docs\design\government\design-native-policy-preservation-source-correction-20261009-grant.json')

def sha(data):
    return hashlib.sha256(data).hexdigest()

def digest(path):
    return sha(Path(path).read_bytes())

def git(*args):
    return subprocess.check_output(['git', '-C', str(repo), *args])

def write_json(path, value):
    path.write_text(json.dumps(value, indent=2) + '\n', encoding='utf-8')

assert git('rev-parse', 'HEAD').decode().strip() == source
assert not git('status', '--porcelain=v1')
assert git('rev-parse', 'origin/codex/model-first-operations').decode().strip() == source
changed = git('diff', '--name-only', baseline, source).decode().splitlines()
assert sorted(changed) == sorted([
    'internal/tooling/codexrunner/README.md',
    'internal/tooling/codexrunner/runner.py',
    'internal/tooling/codexrunner/test_runner.py',
    'docs/validation/project-native-delivery-2026-10-09.md',
])
assert digest(old / 'handoff.json') == '0cf49b54ae019ab2ab2cb6dd3e863c995823a49137a375a9e5e4547ee8aa4cb2'
assert digest(old / 'roombook-model-installation-2b7b22ba.zip') == '1f29af68c6ec2b295c6930c76512deac93a3c565dd6f1fbef1539db5a78918b0'
old_runtime = old / 'roombook-model-install-check' / '.markitect/runtime.yaml'
old_adapter = old / 'installed-tool-root/internal/tooling/codexrunner/runner.py'
assert digest(old_runtime) == 'a99b619fda2226ccd820458b4bc501459b6ad05ba417338942074fead7a0ef98'
assert digest(old_adapter) == '0ceb15e6cfa28de4256d346d61de64a207c307405be4a3549b29bb4f380fc4ac'
runtime_bytes = old_runtime.read_bytes()
adapter = root / 'runtime-assets/runner.py'
adapter.parent.mkdir(exist_ok=True)
adapter.write_bytes(git('show', source + ':internal/tooling/codexrunner/runner.py'))
adapter_sha = digest(adapter)
assert adapter.read_bytes() == (repo / 'internal/tooling/codexrunner/runner.py').read_bytes()
assert b'"--ignore-user-config"' not in adapter.read_bytes()
assert b'"--ignore-rules"' not in adapter.read_bytes()
old_path = str(old_adapter).encode()
new_path = str(adapter).encode()
old_sha = digest(old_adapter).encode()
new_sha = adapter_sha.encode()
assert runtime_bytes.count(old_path) == 4
assert runtime_bytes.count(old_sha) == 2
replacement = runtime_bytes.replace(old_path, new_path).replace(old_sha, new_sha)
assert replacement.replace(new_path, old_path).replace(new_sha, old_sha) == runtime_bytes
runtime = root / 'runtime-replacement.yaml'
runtime.write_bytes(replacement)
(root / 'runtime-replacement.diff').write_text(''.join(difflib.unified_diff(
    runtime_bytes.decode().splitlines(keepends=True), replacement.decode().splitlines(keepends=True),
    fromfile='retained-runtime.yaml', tofile='replacement-runtime.yaml')), encoding='utf-8')
(root / 'committed-source.diff').write_bytes(git('diff', baseline, source))
shutil.copyfile(grant, root / 'source-correction-grant.json')
schema = root / 'project-schema.json'
shutil.copyfile(old / 'project-schema.json', schema)
assert digest(schema) == '30c8fe1855afe37bd424e61c969f7cbab104848f2c0d95c01d8a6bcb4795cd62'

python = Path(r'C:\Python313\python.exe')
codex = Path(r'C:\Users\Consiliari\AppData\Local\OpenAI\Codex\bin\9691020b546a15b2\codex.exe')
assert digest(python) == 'd87063e5597f257004c731b66c59c56c91038861c6877b1a3dca6b8c4e919125'
assert digest(codex) == '3553cd6e7df5a093d8cb8301cd8088a57e0971aba71ddbe0e67f7f44a15cdf68'
pin_values = [(python, digest(python)), (adapter, adapter_sha), (codex, digest(codex))]
pins = []
for role in ['agents', 'review.agents']:
    for path, value in pin_values:
        pins.append({'role': role, 'path': str(path), 'mode': '0644', 'digest': 'sha256:' + value})
write_json(root / 'replacement-pins.json', pins)

records = json.loads((root / 'review-session-records.json').read_text())
call = next(x for x in records if x['payload'].get('type') == 'function_call')
terminal = next(x for x in records if x['payload'].get('type') == 'item_completed' and x['payload']['item'].get('kind') == 'completed')
final = next(x for x in records if x['payload'].get('type') == 'agent_message')
start = datetime.datetime.fromisoformat(call['timestamp'].replace('Z', '+00:00'))
received = datetime.datetime.fromisoformat(final['timestamp'].replace('Z', '+00:00'))
review = {
    'model': 'gpt-6-luna', 'reasoningEffort': 'high', 'forkTurns': 'none',
    'returnedTask': '/root/policy_source_review',
    'returnedThreadId': None,
    'observedActivityMetadataThreadId': terminal['payload']['item']['agent_thread_id'],
    'requestRecordedUtc': call['timestamp'], 'terminalActivityUtc': terminal['timestamp'],
    'finalReceivedUtc': final['timestamp'], 'conservativeReceiptSeconds': (received - start).total_seconds(),
    'sourceChecksOnly': True, 'providerProductEvidence': False,
    'finalText': final['payload']['content'][0]['text'],
    'findingDisposition': 'Low README omission of multi_agent repaired; no post-review executable change.',
    'requestArchive': 'Original call arguments in review-session-records.json are opaque encrypted session data; no reconstructed request presented as original.',
}
assert review['conservativeReceiptSeconds'] <= 600
write_json(root / 'review-receipt.json', review)
build = json.loads((root / 'build-receipt.json').read_text(encoding='utf-8-sig'))
assert build['exitCode'] == 0
binary = root / 'markitect-bcd614a3.exe'
binary_sha = digest(binary)
assert binary_sha == digest(old / 'markitect-2b7b22ba.exe')

files = ['markitect-bcd614a3.exe', 'runtime-assets/runner.py', 'runtime-replacement.yaml',
         'runtime-replacement.diff', 'replacement-pins.json', 'project-schema.json',
         'committed-source.diff', 'source-correction-grant.json',
         'test-batch-1-adapter.txt', 'test-batch-2-entry.txt', 'review-receipt.json',
         'review-session-records.json', 'adapter-policy-delta.diff', 'native-entry-2b7b22ba.diff',
         'build-receipt.json', 'build.log', 'build-info.txt']
file_pins = [{'path': name, 'sha256': digest(root / name)} for name in files]
handoff = {
    'schemaVersion': 1, 'status': 'source-corrected replacement assets; not activated',
    'source': {'repository': str(repo), 'branch': 'codex/model-first-operations',
               'baselineSha': baseline, 'sha': source, 'clean': True, 'pushed': True,
               'changedFiles': changed, 'allGoAndSchemaImplementationUnchanged': True},
    'grant': {'key': 'design-native-policy-preservation-source-correction-20261009',
              'issuedUtc': '2026-10-09T09:42:18Z', 'notAfterUtc': '2026-10-09T10:42:18Z',
              'maxActiveSeconds': 3600, 'builds': 1, 'targetedTestBatches': 2,
              'sourceReviewers': 1, 'newInstallations': 0, 'productJobsAdded': 0,
              'providerOrTransportProbes': 0, 'newStudyAllocation': None},
    'policy': {'removedActualArguments': ['--ignore-user-config', '--ignore-rules'],
               'normalCliRulesAndUserConfigEnabled': True, 'readOnlySandbox': True,
               'disabledFeatures': ['plugins', 'shell_tool', 'unified_exec', 'multi_agent'],
               'adapterTimeoutRangeSeconds': [1, 600], 'hostRoleTimeoutSeconds': 300,
               'model': 'gpt-6-luna', 'reasoningEffort': 'high',
               'normalAmbientInstructionsAbsentClaimed': False, 'osIsolationClaimed': False},
    'build': {**build, 'sourceSha': source, 'binary': str(binary), 'binarySha256': binary_sha,
              'identicalToRetainedBaselineBinary': True, 'release': False},
    'runtime': {'path': str(runtime), 'sha256': digest(runtime), 'adapterPath': str(adapter),
                'adapterSha256': adapter_sha, 'pins': pins, 'activated': False,
                'delta': 'Only four adapter path occurrences and two adapter digests replaced; all other bytes preserved.',
                'perRunLimits': {'maxDepth': 2, 'maxStarts': 24, 'maxRetries': 1, 'maxParallel': 1,
                                 'maxDurationSeconds': 1200, 'reviewRounds': 3, 'managerRounds': 2},
                'cliVersion': '0.162.0-alpha.2', 'newVersionProbe': False},
    'schema': {'path': str(schema), 'sha256': digest(schema),
               'provenance': 'Retained source-2b schema bytes; exact source delta leaves every schema implementation file unchanged.'},
    'validation': {'adapterTests': {'count': 41, 'seconds': 1.107, 'result': 'pass', 'actualProviderCall': False},
                   'onboardingRegression': {'seconds': 0.311, 'result': 'pass'},
                   'rootAndNestedHelpRegressions': {'seconds': 0.555, 'result': 'pass'},
                   'review': review, 'diffCheck': 'pass',
                   'baselineHostedCI': {'runId': 37908655579, 'sha': baseline, 'result': 'success'},
                   'replacementHostedCI': {'runId': 37914238147, 'sha': source, 'result': 'in_progress at checkpoint'}},
    'retainedEvidence': {'oldHandoff': str(old / 'handoff.json'), 'oldHandoffSha256': digest(old / 'handoff.json'),
                         'oldArchiveSha256': digest(old / 'roombook-model-installation-2b7b22ba.zip'),
                         'oldAdapterSha256': digest(old_adapter), 'oldRuntimeSha256': digest(old_runtime),
                         'oldInstallationTargetSha': 'd7a004172f98ebb6bc59bf7e93f9c709fb2ebc86',
                         'unchanged': True, 'oldRuntimeMustNotStartNative': True},
    'remainingGates': ['Complete replacement source hosted gates',
                       'External owner accepts/activates exact replacement runtime and verifies shared counters/deadlines',
                       'Fresh native product or study start requires a separate explicit allocation',
                       'No successful ordinary native delivery, Scientist acceptance, main integration or release established'],
    'files': file_pins,
    'sealedUtc': datetime.datetime.now(datetime.timezone.utc).isoformat(),
}
write_json(root / 'handoff.json', handoff)
note = f'''# Native policy-preservation source handoff

Source `{source}` is clean and pushed on `codex/model-first-operations`; PR #89 remains draft.
The actual adapter no longer passes `--ignore-user-config` or `--ignore-rules`.
Normal CLI policy remains enabled; explicit read-only/tool restrictions, finite role timeouts,
Luna High and invocation-bound response validation remain in place.

Both allowed targeted test batches passed: 41 adapter tests and focused onboarding/help regressions.
The sole fresh Luna-High source review completed in {review['conservativeReceiptSeconds']:.3f} seconds;
its low documentation finding is repaired. These tests used no actual provider call.
The one production build passed. Its SHA-256 is `{binary_sha}`; the Go binary is byte-identical
because the source delta changes only the Python adapter, its tests and documentation.
Baseline source-2b hosted CI passed; replacement run 37914238147 is in progress at this checkpoint.

Replacement adapter SHA-256: `{adapter_sha}`.
Replacement runtime SHA-256: `{digest(runtime)}`.
`runtime-replacement.yaml` changes only the adapter path and digest for both roles.
It is a prepared configuration artifact and has not been installed or activated.
The six pins are checked against the actual existing file bytes without launching Python/Codex roles.
The schema is copied from the unchanged schema implementation's retained source-2b output.

The old Roombook installation and immutable evidence remain unchanged; its policy-suppressing
adapter must not be used for a new native start. No new installation, provider/transport probe,
product job or trial was started. Three prior product jobs remain closed with zero inner starts.
Shared study counters, deadlines and any future allocation belong to the external coordinator.
The sixth cell remains unallocated; this source correction provides no readiness or release claim.
'''
(root / 'HANDOFF.md').write_text(note, encoding='utf-8')
archive = root / 'native-policy-replacement-bcd614a3.zip'
with zipfile.ZipFile(archive, 'w', zipfile.ZIP_DEFLATED) as output:
    for name in files + ['handoff.json', 'HANDOFF.md']:
        output.write(root / name, name)
summary = {'sourceSha': source, 'handoffSha256': digest(root / 'handoff.json'),
           'archivePath': str(archive), 'archiveSha256': digest(archive),
           'binarySha256': binary_sha, 'adapterSha256': adapter_sha, 'runtimeSha256': digest(runtime),
           'sealedUtc': handoff['sealedUtc']}
write_json(root / 'handoff-summary.json', summary)
print(json.dumps(summary, indent=2))
