"""Release final r2 input bytes after focused source corrections are frozen."""
import importlib.util
import json
from pathlib import Path
import sys
ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT/'runtime'))
import dispatch
import government_integration as gi
import classic_integration as ci
spec=importlib.util.spec_from_file_location('native_driver', ROOT/'run-native-integration.py')
d=importlib.util.module_from_spec(spec);spec.loader.exec_module(d)
for arm in ('government','classic'):
    base=d.EXTERNAL/arm; released=base/'released'
    prep=json.loads((base/'preparation-initial.json').read_bytes())
    correction={**d.binding(d.CORRECTION),'sourceKey':d.CORRECTION_KEY}
    diagnostic=d.write_new(base/'wrapper-diagnostics-config.json',{'arm':arm,'correction':correction,'requestPath':str(released/'request.json')})
    auth=json.loads((released/'role-auth.json').read_bytes())
    auth['maxCalls']=6;auth['diagnostics']=diagnostic
    auth_raw=dispatch.encoded(auth)+b'\n';(released/'role-auth.json').write_bytes(auth_raw)
    runtime=json.loads((released/'runtime.json').read_bytes())
    roles=[runtime['executor'],runtime['verifier']]
    if arm=='government':roles += [item['runner'] for item in runtime['ressorts']]
    for role in roles:
        # Only static, not-yet-frozen prepared inputs are being finalized.
        args=role['args']
        args[args.index('--authorization-sha256')+1]=dispatch.digest(auth_raw)
        args.extend(['--diagnostics-config',diagnostic['path'],'--diagnostics-sha256',diagnostic['sha256']])
        files=role['runtimeFiles']
        files.append(gi.runtime_file(diagnostic['path']))
        role['runtimeFiles']=[gi.runtime_file(item['path']) for item in files]
        assert len(role['runtimeFiles'])<=32
    (released/'runtime.json').write_bytes(dispatch.encoded(runtime)+b'\n')
    prep['roleAuthorizationSha256']=dispatch.digest(auth_raw)
    prep['runtimeSha256']=dispatch.digest((released/'runtime.json').read_bytes())
    prep['runtimeSourceSha256']=dispatch.runtime_pins()
    prep['correction']=correction
    prep['diagnostics']=diagnostic
    d.write_new(base/'preparation-final.json',prep)
    print(json.dumps({'arm':arm,'authority':d.prepare(arm),'nativeStarts':0},sort_keys=True))
