"""One bounded thread/start identity gate, no turns, tools, or process APIs."""
import re


def require(ok, reason):
    if not ok:
        raise ValueError(reason)


class ThreadGate:
    def __init__(self, profile, protocol, contract):
        self.profile = profile
        self.protocol = protocol
        self.contract = contract
        self.thread_id = None
        self.validated = False
        self.started_seen = False
        self.closed_seen = False
        self.pending = []
        self.observation = None

    def _thread(self, thread):
        p = self.profile['threadStart']['params']
        require(type(thread.get('id')) is str and re.fullmatch(r'[A-Za-z0-9:_-]{1,128}', thread['id']), 'thread-id-invalid')
        require(thread['cwd'] == p['cwd'] and thread['modelProvider'] == p['modelProvider'] and thread['ephemeral'] is True, 'thread-identity-mismatch')
        require(thread.get('parentThreadId') is None and thread.get('forkedFromId') is None and thread.get('model') in (None,p['model']), 'thread-model-or-parent-mismatch')
        require(thread.get('turns', []) == [], 'unexpected-existing-turn')
        return thread['id']

    def accept_response(self, result):
        require(not self.validated, 'duplicate-thread-response')
        self.protocol.validate(self.contract['responseSchemas']['thread/start'], result)
        p = self.profile['threadStart']['params']
        require(all(result[k] == p[k] for k in ['cwd','model','modelProvider','approvalPolicy']), 'thread-response-identity-or-policy-mismatch')
        sandbox = result['sandbox']; active = result.get('activePermissionProfile')
        require(sandbox.get('type') == 'readOnly' and sandbox.get('networkAccess') is not True, 'thread-response-effective-sandbox-mismatch')
        require(active is None or (active.get('id') == ':read-only' and active.get('extends') in (None,'')), 'thread-response-permission-profile-mismatch')
        self.thread_id = self._thread(result['thread']); self.validated = True
        self.observation = {'responseEnvelopeValidated':True,'gatePassed':True,'threadId':self.thread_id,'model':p['model'],'modelProvider':p['modelProvider'],'approvalPolicy':p['approvalPolicy'],'cwdMatches':True,'ephemeral':True,'sandbox':{'type':'readOnly','networkAccess':{'state':'present','value':sandbox['networkAccess']} if 'networkAccess' in sandbox else {'state':'missing'}},'activePermissionProfile':{'state':'missing'} if 'activePermissionProfile' not in result else {'state':'null'} if active is None else {'state':'present','value':{'id':':read-only','extends':active.get('extends')}}}
        pending,self.pending = self.pending,[]
        for record in pending:self._apply(record)

    def accept_notification(self, method, params):
        require(method in self.contract['notifications'], 'forbidden-or-unknown-notification')
        self.protocol.validate(self.contract['notifications'][method], params)
        if method == 'thread/started':
            record = (method,self._thread(params['thread']),None)
        else:
            status=params.get('status')
            require(method!='thread/status/changed' or (status.get('type')=='idle' or (status.get('type')=='active' and status.get('activeFlags')==[])), 'thread-status-not-allowed')
            record=(method,params['threadId'],status.get('type') if status else None)
        if not self.validated:
            require(method != 'thread/closed' and len(self.pending)<64, 'early-thread-closure-or-pending-limit')
            self.pending.append(record)
        else:self._apply(record)

    def _apply(self, record):
        method,identifier,status=record
        require(identifier==self.thread_id, 'thread-notification-id-mismatch')
        if method=='thread/started':
            require(not self.started_seen and not self.closed_seen,'duplicate-or-late-thread-started');self.started_seen=True
        elif method=='thread/closed':
            require(not self.closed_seen,'duplicate-thread-closed');self.closed_seen=True
        else:require(not self.closed_seen and status in ('idle','active'),'unexpected-active-status-without-turn')

    def finish(self):
        return {'responseValidated':self.validated,'threadId':self.thread_id,'threadStartedNotificationCorrelated':self.started_seen,'threadClosedNotificationCorrelated':self.closed_seen,'pendingLifecycleCount':len(self.pending),'reportedPermissionObservation':self.observation,'turnsRequested':0,'toolsRequested':0}
