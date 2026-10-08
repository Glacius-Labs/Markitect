"""Static R3 fixture preparation only; never invokes a product or role runner."""
import importlib.util
import json
from pathlib import Path
import sys
import time
ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'runtime'))
import government_integration as gi
import classic_integration as ci
import government_roles as gr
import dispatch
spec = importlib.util.spec_from_file_location('native_driver', ROOT / 'run-native-integration-r3.py')
driver = importlib.util.module_from_spec(spec); spec.loader.exec_module(driver)
write = driver.write_new
external = driver.EXTERNAL
history = {
    'acceptedOfflineHandoffSha': 'f5cd8871b84f45541044110057b59f0bb1ca0804',
    'acceptedR2NegativeHandoffSha': '419775ab0ea5f421655ab3c8bd97e286ffbb6e0a',
    'nativeBudgetShaBefore': 'c5769c0c204cfbc5d17d95320c8b2288b746aa785c16796ce0d7ab433b58fbf5',
    'originalNativeBudgetPath': str(driver.LEGACY / 'native-starts.sqlite'),
    'previousNativeStarts': {'government': 3, 'classic': 2},
    'previousReservedSessionSeconds': {'government': 450, 'classic': 300},
    'previousNativeWrapperAttempts': {'government': 2, 'classic': 1},
    'previousDelegateAttempts': 0,
    'priorBudgetSnapshot': driver.binding(ROOT / 'evidence/native-integration/run-2/quota-and-ledger-summary.json'),
    'priorDeliveryValidation': driver.binding(ROOT / 'evidence/native-integration/run-2/delivery-validation.json'),
    'priorGovernmentLedger': driver.binding(driver.LEGACY / 'government/fixture-ledger.sqlite'),
    'priorClassicLedger': driver.binding(driver.LEGACY / 'fixture-ledger.sqlite'),
    'historyDisposition': 'Append native starts to original durable ledger. New role ledgers belong to fresh cases and reference, never replace, prior ledgers. Historical Government controller remains running.'}
write(external / 'history.json', history)
write(driver.EVIDENCE / 'history.json', history)
expiry = time.time() + 5400
for arm in ('government', 'classic'):
    base = external / arm
    released = base / 'released'
    if arm == 'government':
        base.mkdir()
        actor = base / 'actor'
        gi.create_disposable_repository(actor)
        assert gi.digest((actor/'government.yaml').read_bytes()) == '816041553684dc5710baf63217d435f62e9f997d1bbba4b74d9a2c4803da4325'
        marker = gi.bind_order_to_inspection(actor, 'sha256:c1d7194bd6a8961001c68cbace3f7da40d5daf1b22570026d097e6091cf63e93')
        assert marker['orderSha256'] == '7707549c484a2af0f864bb914b9b192e9352a603985f7cede093a2c32655223d'
        released.mkdir()
        for name in ('queue-state', 'run-state', 'temporary'):
            (base / 'results' / name).mkdir(parents=True)
        auth = gi.build_role_authorization(trial_id='native-contract-corrected-r3-government',dispatch_id='government-native-contract-corrected-r3',
            request_path=released/'request.json',ledger_path=base/'fixture-ledger.sqlite',runtime_path=released/'runtime.json',
            role_evidence_directory=base/'role-evidence',expires_at=expiry,python_executable=sys.executable,
            delegate_path=gi.FIXTURE_ROOT/'deterministic_delegate.py',status='approved',fixture_authorization=gi.FIXTURE_AUTHORIZATION)
        auth['maxCalls'] = 6
        write(released/'role-auth.json',auth)
        runtime = gi.build_runtime(repository=actor,base_commit=marker['baseCommit'],queue_state_directory=base/'results/queue-state',
            run_state_directory=base/'results/run-state',temporary_directory=base/'results/temporary',runtime_path=released/'runtime.json',
            authorization_path=released/'role-auth.json',authorization_raw=(released/'role-auth.json').read_bytes(),python_executable=sys.executable,
            delegate_path=gi.FIXTURE_ROOT/'deterministic_delegate.py',expected_constitution_digest=marker['constitutionDigest'])
        runtime['timeoutSeconds'] = 38
        write(released/'runtime.json',runtime)
        backlog = gi.build_backlog(runtime_path=released/'runtime.json',queue_state_directory=base/'results/queue-state')
        backlog['limits']['maxWallTimeSeconds'] = 38
        write(released/'backlog.json',backlog)
        prep = dict(marker, productGovernment={'queueStateDirectory': str(base/'results/queue-state')})
    else:
        prep = ci.prepare_fixture(ci.PACKET,base,trial_id='native-contract-corrected-r3-classic',dispatch_id='classic-native-contract-corrected-r3',
            task_id='native-classic-positive',expires_at=expiry)
    write(base/'preparation-initial.json',dict(prep,history=history))
print(json.dumps({'prepared': ['government','classic'], 'nativeStarts':0,'history':history},indent=2))
