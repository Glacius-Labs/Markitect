"""Prepare and exercise a real provider-backed Shop change via the public CLI.

Requires an existing provider sign-in. Rates are caller budget weights, not prices.
All fixtures, frozen adapter bytes and logs are written below --output.
"""
from __future__ import annotations
import argparse
import json
from pathlib import Path
import shutil
import subprocess
import uuid


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument('--source', type=Path, required=True)
    parser.add_argument('--tool', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--model', required=True)
    parser.add_argument('--stage', choices=('prepare', 'run', 'verify', 'apply'), default='prepare')
    args = parser.parse_args()
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    state_path = output / 'state.json'
    source = args.source.resolve()
    tool = args.tool.resolve()
    if args.stage == 'prepare':
        root = output / ('shop-' + uuid.uuid4().hex)
        shutil.copytree(source / 'examples/project-world', root)
        frozen = output / ('tools-' + uuid.uuid4().hex)
        frozen.mkdir()
        shutil.copy2(source / 'go.mod', frozen / 'go.mod')
        for provider in ('codex', 'claude'):
            path = Path('internal/tooling') / (provider + 'runner') / 'runner.py'
            (frozen / path).parent.mkdir(parents=True)
            shutil.copy2(source / path, frozen / path)
        def git(*words: str) -> str:
            return subprocess.check_output(['git', '-C', str(root), *words], stderr=subprocess.STDOUT, text=True).strip()
        git('init', '-b', 'codex/shop-live-packing')
        git('config', 'core.autocrlf', 'false')
        git('config', 'user.name', 'Markitect example validation')
        git('config', 'user.email', 'markitect-example@example.invalid')
        git('add', '.')
        git('commit', '-m', 'Freeze baseline Shop example')
        setup_args = ['--tool-root', str(frozen), '--provider', 'codex', '--model', args.model,
                      '--effort', 'high', '--input-micros-per-million', '20000000',
                      '--output-micros-per-million', '100000000', '--max-cost-micros', '10000000']
        invoke(tool, root, output, 'doctor', '--tool-root', str(frozen), '--provider', 'codex')
        preview = invoke(tool, root, output, 'setup', *setup_args)
        invoke(tool, root, output, 'setup', *setup_args, '--expect', preview['editPlan']['digest'], '--write')
        git('add', '.markitect/runtime.yaml')
        git('commit', '-m', 'Accept reviewed project-local runtime')
        base = git('rev-parse', 'HEAD')
        checked = invoke(tool, root, output, 'check')
        replacements = {
            '.markitect/model/commerce/sales/orders/requires-active-reservation.yaml': ('before a confirmed order can be cancelled.', 'before a confirmed or packing order can be cancelled.'),
            '.markitect/model/commerce/sales/orders/cancel-order.yaml': ('A confirmed order may be cancelled before shipment', 'A confirmed or packing order may be cancelled before shipment'),
            '.markitect/model/commerce/sales/orders/order.yaml': ('confirmed or shipped state', 'confirmed, packing or shipped state'),
            '.markitect/model/commerce/sales/orders/cancel-before-shipped.yaml': (
                'Cancellation is valid only while the order is confirmed; a shipped order cannot be cancelled.',
                'Cancellation is valid while the order is confirmed or packing; a shipped order cannot be cancelled. Reservation release remains atomic and repeated cancellation is idempotent.')}
        files = []
        for path, (old, new) in replacements.items():
            content = (root / path).read_text(encoding='utf-8')
            assert old in content
            content = content.replace(old, new)
            if path.endswith('/requires-active-reservation.yaml'):
                assert 'leaves the order confirmed by rolling back' in content
                content = content.replace('leaves the order confirmed by rolling back',
                                          'leaves the order in its original confirmed or packing state by rolling back')
            files.append({'path': path, 'content': content})
        mutation = {'apiVersion': 'project.markitect.example.org/v1alpha1', 'baseDigest': checked['projectDigest'],
                    'actor': 'user', 'goal': 'Allow cancellation during packing, preserving atomic reservation release, shipped-order rejection and idempotency.',
                    'files': files}
        draft = root / '.markitect/drafts/packing-model.json'
        draft.parent.mkdir(parents=True, exist_ok=True)
        draft.write_text(json.dumps(mutation, ensure_ascii=False), encoding='utf-8')
        edit = invoke(tool, root, output, 'edit', '--input', '.markitect/drafts/packing-model.json')
        invoke(tool, root, output, 'edit', '--input', '.markitect/drafts/packing-model.json', '--expect', edit['digest'], '--write')
        git('add', '.markitect')
        git('commit', '-m', 'Accept packing cancellation model change')
        plan = invoke(tool, root, output, 'plan', '--since', base, '--goal',
                      'Implement the accepted model change: allow cancellation during packing as well as confirmed, keep atomic reservation release and shipped-order rejection, update documentation and add a regression test.', '--write')
        assert len(plan['managers']) == 6 and len(plan['checks']) == 1, plan
        state = {'repo': str(root), 'tool': str(tool), 'model': args.model,
                 'base': base, 'head': git('rev-parse', 'HEAD'), 'plan': plan['id'],
                 'sourceRevision': subprocess.check_output(['git', '-C', str(source), 'rev-parse', 'HEAD'], text=True).strip()}
        state_path.write_text(json.dumps(state, indent=2), encoding='utf-8')
        print(json.dumps({'prepared': True, 'repo': str(root), 'plan': plan['id'], 'managers': len(plan['managers']), 'checks': len(plan['checks'])}))
        return
    state = json.loads(state_path.read_text(encoding='utf-8'))
    tool, root = Path(state['tool']), Path(state['repo'])
    if args.stage == 'run':
        result = invoke(tool, root, output, 'run', '--plan', state['plan'], '--write')
    elif args.stage == 'verify':
        result = invoke(tool, root, output, 'verify', '--run', state['plan'], '--write')
        assert result['status'] == 'verified', result
    else:
        status = invoke(tool, root, output, 'status', '--run', state['plan'])
        candidate = status['run']['candidate']['id']
        binding = ['--plan', state['plan'], '--run', state['plan'], '--candidate', candidate]
        preflight = invoke(tool, root, output, 'apply', *binding)
        exact = ['--branch', preflight['targetBranch'], '--head', preflight['expectedHead'],
                 '--worktree', preflight['expectedWorktree'], '--expect', preflight['verificationDigest'], '--write']
        probe = root / 'docs/cancellation.md'
        original = probe.read_bytes()
        try:
            probe.write_bytes(original + b'\n<!-- stale preflight probe -->\n')
            rejected = subprocess.run([str(tool), 'project', 'apply', '--repo', str(root), *binding, *exact],
                                      capture_output=True, text=True, encoding='utf-8')
            (output / 'apply-stale-probe.log').write_text(rejected.stdout + rejected.stderr, encoding='utf-8')
            assert rejected.returncode != 0 and ('stale' in rejected.stderr or 'changed' in rejected.stderr), rejected.stderr
        finally:
            probe.write_bytes(original)
        result = invoke(tool, root, output, 'apply', *binding, *exact)
        assert result['status'] == 'applied', result
    print(json.dumps({'stage': args.stage, 'status': result.get('status'), 'plan': state['plan']}))


def invoke(tool: Path, root: Path, output: Path, action: str, *words: str) -> dict:
    result = subprocess.run([str(tool), 'project', action, '--repo', str(root), *words], capture_output=True, text=True, encoding='utf-8')
    suffix = '-write' if '--write' in words else ''
    (output / (action + suffix + '.stdout.json')).write_text(result.stdout, encoding='utf-8')
    (output / (action + suffix + '.stderr.log')).write_text(result.stderr, encoding='utf-8')
    if result.returncode:
        raise RuntimeError(f'{action} exit={result.returncode}: {result.stderr[:600]}')
    return json.loads(result.stdout)


if __name__ == '__main__':
    main()
