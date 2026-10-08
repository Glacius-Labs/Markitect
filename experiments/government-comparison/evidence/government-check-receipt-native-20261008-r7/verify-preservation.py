"""Read-only package preservation; no experiment admission or process launch."""
import ast
import hashlib
import json
from pathlib import Path
import subprocess

EVIDENCE = Path(__file__).resolve().parent
PACKAGE = EVIDENCE.parents[1]
REPO = PACKAGE.parents[1]

def sha(raw):
    return hashlib.sha256(raw).hexdigest()

def verify():
    baseline = json.loads((EVIDENCE / 'historical-bindings.json').read_bytes())
    for item in baseline['files']:
        assert sha((PACKAGE / item['path']).read_bytes()) == item['sha256'], item['path']
    handoff = (PACKAGE / 'native-integration-handoff.md').read_bytes()
    assert sha(handoff[baseline['handoffPrefixBytes']:]) == baseline['handoffOriginalSha256']
    source_path = 'experiments/government-comparison/runtime/native_fixture_budget.py'
    old = subprocess.check_output(['git', 'show', baseline['baseCommit'] + ':' + source_path], cwd=REPO)
    current = (PACKAGE / 'runtime/native_fixture_budget.py').read_bytes()
    def check_ast(raw):
        return ast.dump(next(node for node in ast.parse(raw).body
                            if isinstance(node, ast.FunctionDef) and
                            node.name == '_validate_r4_fresh_check_receipts'))
    assert check_ast(old) == check_ast(current), 'accepted receipt-consumer correction changed'
    archive = json.loads((PACKAGE / 'evidence/government-released-binding-native-20261008-r6/external-archive.json').read_bytes())
    for item in archive['pairs']:
        assert sha(Path(item['original']).read_bytes()) == item['sha256'], item['original']
        assert sha((PACKAGE / item['copy']).read_bytes()) == item['sha256'], item['copy']
    return {'protectedHistoricalFiles': len(baseline['files']), 'historicalHandoffSuffixExact': True,
            'acceptedReceiptConsumerAstUnchanged': True, 'r6RawOriginalCopyPairs': len(archive['pairs'])}

if __name__ == '__main__':
    print(json.dumps(verify(), sort_keys=True))
