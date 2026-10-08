"""Real provider Brownfield proposal exercise; acceptance choices stay explicit."""
from __future__ import annotations
import argparse
import json
from pathlib import Path
import shutil
import subprocess
import uuid
from run import invoke


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument('--source', type=Path, required=True)
    parser.add_argument('--tool', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--model', required=True)
    parser.add_argument('--stage', choices=('prepare', 'generate', 'resolve', 'adopt'), default='prepare')
    parser.add_argument('--choices', type=Path)
    args = parser.parse_args()
    output, source, tool = args.output.resolve(), args.source.resolve(), args.tool.resolve()
    output.mkdir(parents=True, exist_ok=True)
    state_path = output / 'state.json'
    if args.stage == 'prepare':
        root = output / ('brownfield-' + uuid.uuid4().hex)
        shutil.copytree(source / 'examples/project-world', root)
        frozen = output / ('tools-' + uuid.uuid4().hex)
        frozen.mkdir()
        shutil.copy2(source / 'go.mod', frozen / 'go.mod')
        for provider in ('codex', 'claude'):
            path = Path('internal/tooling') / (provider + 'runner') / 'runner.py'
            (frozen / path).parent.mkdir(parents=True)
            shutil.copy2(source / path, frozen / path)
        (root / 'docs/legacy-cancellation.md').write_text(
            '# Legacy cancellation guide\n\nOrders in confirmed or packing state can be cancelled.\n'
            'A shipped order cannot be cancelled. Cancellation releases the active reservation.\n', encoding='utf-8')
        def git(*words: str) -> str:
            return subprocess.check_output(['git', '-C', str(root), *words], stderr=subprocess.STDOUT, text=True).strip()
        git('init', '-b', 'codex/shop-brownfield')
        git('config', 'core.autocrlf', 'false')
        git('config', 'user.name', 'Markitect example validation')
        git('config', 'user.email', 'markitect-example@example.invalid')
        git('add', '.')
        git('commit', '-m', 'Freeze intentionally contradictory Brownfield Shop')
        config = ['--tool-root', str(frozen), '--provider', 'codex', '--model', args.model, '--effort', 'high',
                  '--input-micros-per-million', '20000000', '--output-micros-per-million', '100000000', '--max-cost-micros', '5000000']
        setup = invoke(tool, root, output, 'setup', *config)
        invoke(tool, root, output, 'setup', *config, '--expect', setup['editPlan']['digest'], '--write')
        git('add', '.markitect/runtime.yaml')
        git('commit', '-m', 'Accept explicit Brownfield proposal runtime')
        head = git('rev-parse', 'HEAD')
        selected = [('orders-source', 'src/shop/orders/order.py', 'code'),
                    ('cancellation-source', 'src/shop/commerce/cancellation.py', 'code'),
                    ('inventory-source', 'src/shop/inventory/reservations.py', 'code'),
                    ('legacy-guide', 'docs/legacy-cancellation.md', 'documentation')]
        request = {'apiVersion': 'markitect.example.org/project-discovery/v1alpha1', 'id': 'shop-brownfield-example',
                   'purpose': 'Use exactly two peer adoption scopes with local IDs orders and inventory, both with parentId empty. Each scope needs its own grounded claims and at least one additive model file. Do not introduce shop, sales or other organizational container scopes; the accepted target already supplies Managers. Preserve the disagreement between legacy packing documentation and confirmed-only source as a blocking owner question. Propose additive Statements with unique imported- names under existing Manager namespaces; preserve accepted target contracts.',
                   'review': 'Finite example discovery only. Review terminology and static intent; no runtime record was selected. Owner decides adoption per scope.',
                   'commit': head, 'scopeRoots': ['src/shop', 'docs'],
                   'selected': [{'id': identity, 'path': path, 'basis': basis, 'reason': 'Explicit input for the bounded cancellation/reservation contradiction exercise.'} for identity, path, basis in selected],
                   'exclusions': [], 'unselected': [{'path': 'docs/cancellation.md', 'reason': 'Existing guide omitted from this bounded exercise.'}]}
        drafts = root / '.markitect/drafts'
        drafts.mkdir(parents=True, exist_ok=True)
        (drafts / 'discovery-request.json').write_text(json.dumps(request), encoding='utf-8')
        discovery = invoke(tool, root, output, 'discover', '--request', '.markitect/drafts/discovery-request.json', '--output', '.markitect/drafts/discovery.json')
        state = {'repo': str(root), 'tool': str(tool), 'model': args.model, 'head': head,
                 'sourceRevision': subprocess.check_output(['git', '-C', str(source), 'rev-parse', 'HEAD'], text=True).strip()}
        state_path.write_text(json.dumps(state, indent=2), encoding='utf-8')
        print(json.dumps({'prepared': True, 'repo': str(root), 'sourceCommit': head}))
        return
    state = json.loads(state_path.read_text(encoding='utf-8'))
    root, tool = Path(state['repo']), Path(state['tool'])
    common = ['--source-repo', str(root), '--revision', state['head'], '--discovery', '.markitect/drafts/discovery.json', '--report', '.markitect/drafts/distillation.json']
    if args.stage == 'generate':
        result = invoke(tool, root, output, 'distill', '--discovery', '.markitect/drafts/discovery.json', '--generate', '--write',
                        '--output', '.markitect/drafts/distillation.json', '--input-micros-per-million', '20000000',
                        '--output-micros-per-million', '100000000', '--max-cost-micros', '5000000')
    elif args.stage == 'resolve':
        if args.choices is None:
            parser.error('--stage resolve requires explicit --choices')
        shutil.copyfile(args.choices, root / '.markitect/drafts/choices.json')
        result = invoke(tool, root, output, 'resolve', *common, '--input', '.markitect/drafts/choices.json', '--output', '.markitect/drafts/resolution.json')
    else:
        result = invoke(tool, root, output, 'adopt', *common, '--resolution', '.markitect/drafts/resolution.json', '--output', '.markitect/drafts/adoption-plan.json')
        plan = json.loads((root / '.markitect/drafts/adoption-plan.json').read_text(encoding='utf-8'))
        result = invoke(tool, root, output, 'adopt', *common, '--resolution', '.markitect/drafts/resolution.json',
                        '--plan', '.markitect/drafts/adoption-plan.json', '--expect', plan['planDigest'], '--write')
    print(json.dumps({'stage': args.stage, 'result': result}))


if __name__ == '__main__':
    main()
