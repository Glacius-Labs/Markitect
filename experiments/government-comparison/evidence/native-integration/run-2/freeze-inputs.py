"""Freeze exact reviewed r2 inputs; no product invocation."""
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
ROOT=Path(__file__).resolve().parents[3]
sys.path.insert(0,str(ROOT/'runtime'))
import dispatch
spec=importlib.util.spec_from_file_location('native_driver',ROOT/'run-native-integration.py')
d=importlib.util.module_from_spec(spec);spec.loader.exec_module(d)
source_commit=subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip()
if subprocess.check_output(['git','status','--porcelain'],text=True).strip():
    raise ValueError('clean corrected source commit required before freeze')
paths={Path(__file__),ROOT/'run-native-integration.py',ROOT/'public/native-integration-r2-preflight.md',
       d.CORRECTION,d.EXTERNAL/'coordinator-authority-snapshot.json',d.EXTERNAL/'history.json',d.SOURCE_GRANT,
       d.LEGACY/'coordinator-authority-snapshot.json',Path(sys.executable)}
paths.update((ROOT/'runtime'/name).resolve() for name in dispatch.runtime_pins() if not name.startswith('public/'))
paths.update((ROOT/'public'/name[len('public/'):]).resolve() for name in dispatch.runtime_pins() if name.startswith('public/'))
for path in (ROOT/'runtime').glob('test_*integration.py'):paths.add(path)
for name in ('test_native_controller.py','test_native_fixture_budget.py','test_native_role_diagnostics.py','test_government.py','test_government_roles.py','test_dispatch.py'):
    paths.add(ROOT/'runtime'/name)
for path in d.EVIDENCE.iterdir():
    if path.is_file():paths.add(path)
for arm in ('government','classic'):
    base=d.EXTERNAL/arm
    for name in ('authority.json','grant.json','protocol.json','preparation-initial.json','preparation-final.json','wrapper-diagnostics-config.json'):
        paths.add(base/name)
    request=json.loads((base/'released/request.json').read_bytes())
    paths.add(base/'released/request.json')
    paths.update(Path(item['path']) for item in request['releasedInputs'])
    product=request['product'][arm]
    paths.add(Path(product['executable']['path']))
    paths.add(Path(request['actorRepository'])/product['projectConfig']['path'])
    runtime=json.loads(Path(product['runtime']['path']).read_bytes())
    roles=[runtime['executor'],runtime['verifier']]
    if arm=='government':
        roles += [item['runner'] for item in runtime['ressorts']]
        paths.add(Path(request['actorRepository'])/'order.yaml')
    for role in roles:
        paths.update(Path(item['path']) for item in role['runtimeFiles'])
freeze={'status':'independently-reviewed-mechanical-fixture','sourceCommit':source_commit,
        'reviewSha256':d.binding(ROOT/'public/native-integration-r2-preflight.md')['sha256'],
        'historySha':'f35c8209cd8902bff17e69f05ee7716e352ee4b2',
        'inputs':[d.binding(path) for path in sorted(paths,key=str)]}
print(json.dumps(d.write_new(d.EVIDENCE/'preflight-freeze.json',freeze)))
