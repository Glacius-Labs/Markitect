"""CCF metadata-only read inside the pinned Python runtime: login, catalog, account and quota.

No thread/start or turn/start is sent. Only the dedicated read-only volume at
/auth is read; credential bytes and account identifiers never reach stdout/stderr; only plan/type and quota metadata are reported.
"""
import datetime
import base64
import json
import os
from pathlib import Path
import selectors
import subprocess
import sys
import time

requested_model = sys.argv[1] if len(sys.argv) == 2 else 'gpt-6.1-sol'
if requested_model not in ('gpt-6.1-sol', 'gpt-6-luna'):
    raise SystemExit('unsupported metadata model')
result = {"kind": "metadata-only", "generationRequests": 0,
          "requestedModel": requested_model, "requestedEffort": "high",
          "credentialSource": "mk-codex-runtime-auth (read-only)",
          "checkedAtUtc": datetime.datetime.now(datetime.timezone.utc).isoformat()}
peer = None
path = None
try:
    login = json.loads(Path('/auth/auth.json').read_bytes())
    assert login.get('auth_mode') == 'chatgpt' and login.get('OPENAI_API_KEY') is None
    assert all(login.get('tokens', {}).get(key) for key in ('id_token', 'access_token', 'account_id'))
    token = login['tokens']['access_token']
    claim = token.split('.')[1]
    expires = json.loads(base64.urlsafe_b64decode(claim + '=' * (-len(claim) % 4))).get('exp')
    result['accessExpiresUtc'] = datetime.datetime.fromtimestamp(expires, datetime.timezone.utc).isoformat()
    result['accessValiditySeconds'] = int(expires - time.time())
    result['validForSeriesCeiling'] = result['accessValiditySeconds'] >= 8100
    result['hasApiKey'] = False
    result['refreshTokenDelivered'] = False
    login['tokens']['refresh_token'] = ''
    home = Path('/tmp/ccf-codex-metadata')
    home.mkdir(mode=0o700)
    path = home / 'auth.json'
    path.write_text(json.dumps(login))
    path.chmod(0o600)
    env = {key: os.environ[key] for key in ('PATH', 'HOME', 'LANG', 'SSL_CERT_FILE', 'SSL_CERT_DIR')
           if key in os.environ}
    env['CODEX_HOME'] = str(home)
    status = subprocess.run(['/usr/local/bin/codex', 'login', 'status'],
                            capture_output=True, timeout=15, env=env)
    result['loginStatusExit'] = status.returncode
    result['loginReportsChatGPT'] = b'ChatGPT' in status.stdout + status.stderr
    assert status.returncode == 0 and result['loginReportsChatGPT']
    argv = ['/usr/local/bin/codex', 'app-server', '--disable', 'apps', '--disable', 'memories',
            '--disable', 'multi_agent', '--disable', 'image_generation', '--disable', 'browser_use',
            '--disable', 'computer_use', '-c', 'cli_auth_credentials_store="file"',
            '-c', 'model_provider="openai"']
    peer = subprocess.Popen(argv, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                            stderr=subprocess.DEVNULL, env=env)
    selector = selectors.DefaultSelector()
    selector.register(peer.stdout, selectors.EVENT_READ)
    def send(value):
        peer.stdin.write(json.dumps(value).encode() + b'\n')
        peer.stdin.flush()
    def response(request_id):
        end = time.monotonic() + 30
        while time.monotonic() < end:
            if not selector.select(max(0, end - time.monotonic())):
                break
            raw = peer.stdout.readline()
            if not raw:
                raise RuntimeError('metadata connection ended')
            value = json.loads(raw)
            if value.get('id') == request_id:
                assert 'error' not in value
                return value['result']
        raise TimeoutError('metadata response timeout')
    send({'id': 1, 'method': 'initialize', 'params': {
        'clientInfo': {'name': 'markitect-metadata', 'title': 'Markitect metadata', 'version': '1'},
        'capabilities': {'experimentalApi': True}}})
    response(1)
    send({'method': 'initialized', 'params': {}})
    send({'id': 2, 'method': 'model/list', 'params': {'includeHidden': True, 'limit': 100}})
    catalog = response(2)
    models = catalog.get('data', [])
    selected = [item for item in models if item.get('model') == requested_model
                or item.get('id') == requested_model]
    result['catalogCount'] = len(models)
    result['modelListed'] = bool(selected)
    result['catalogHasMore'] = catalog.get('nextCursor') is not None
    result['selected'] = [{key: item.get(key) for key in
                           ('id', 'model', 'supportedReasoningEfforts', 'defaultReasoningEffort')}
                          for item in selected]
    send({'id': 3, 'method': 'account/read', 'params': {'refreshToken': False}})
    account = response(3)
    result['account'] = {key: (account.get('account') or {}).get(key) for key in ('type', 'planType')}
    send({'id': 4, 'method': 'account/rateLimits/read', 'params': {}})
    quota = response(4)
    result['quota'] = {key: quota.get(key) for key in ('rateLimits', 'rateLimitsByLimitId')}
    result['inferenceAccessProved'] = False
    result['catalogMayBeBundledOrCached'] = True
except Exception as error:
    result['preflightErrorType'] = type(error).__name__
finally:
    if peer is not None:
        peer.terminate()
        try:
            peer.wait(timeout=5)
        except subprocess.TimeoutExpired:
            peer.kill()
            peer.wait(timeout=5)
    result['appServerStopped'] = peer is None or peer.poll() is not None
    try:
        if path is not None:
            path.unlink(missing_ok=True)
        result['temporaryCredentialRemoved'] = path is None or not path.exists()
    except OSError:
        result['temporaryCredentialRemoved'] = False
    print(json.dumps(result, indent=2))
sys.exit(0 if result.get('modelListed') and result.get('appServerStopped') and result.get('temporaryCredentialRemoved') else 4)
