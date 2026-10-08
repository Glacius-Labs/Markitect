"""Prepare the exact native spawn boundary; invokes no native tool itself.

Root must honor mayExecute and immediately use exact toolArguments, then bind
the returned Actor/tool-call IDs with record_start. Mock fixtures never authorize.
"""
import hashlib, pathlib
from dispatch_gate import require

def prepare_spawn(gate, action_id, prompt_sha256, profile_sha256):
    with gate.transaction() as state:
        require(action_id in state['actions'],'missing reservation')
        action=dict(state['actions'][action_id])
    raw=pathlib.Path(action['prompt']).read_bytes()
    require(hashlib.sha256(raw).hexdigest()==prompt_sha256,'native prompt bytes differ')
    message=raw.decode('utf8')
    admission=gate.before_native(action_id,prompt_sha256,profile_sha256)
    return {'actionId':action_id,'mayExecute':admission['nativeExecutionAuthorized'],
            'mechanicalMockOnly':admission['mechanicalMockOnly'],'tool':'collaboration.spawn_agent',
            'toolArguments':{'task_name':'v4_'+action_id,'message':message,'fork_turns':'none'},
            'effectiveEnd':admission['effectiveEnd'],'interruptRequestBy':admission['interruptRequestBy'],
            'ownStateDirectory':admission['stateDirectory'],'output':admission['output'],'logs':admission['logs'],
            'actualStopEnforcementProven':False,'osCancellation':None,'providerCancellation':None}
