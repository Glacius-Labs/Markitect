"""Pure pilot transport from one validated Host invocation to a fresh Luna High agent."""
import argparse, hashlib, importlib.util, json, os, pathlib, re, sys, time


def digest(data):
    return 'sha256:' + hashlib.sha256(data).hexdigest()


def put(path, data):
    with path.open('xb') as stream:
        stream.write(data)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--queue-root', required=True)
    ap.add_argument('--prompt-adapter', required=True)
    args = ap.parse_args()
    spec = importlib.util.spec_from_file_location('prompt_adapter', args.prompt_adapter)
    adapter = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(adapter)
    raw = sys.stdin.buffer.read(32 * 1024 * 1024 + 1)
    if len(raw) > 32 * 1024 * 1024:
        raise ValueError('invocation too large')
    invocation = adapter.validate_invocation(adapter.strict_loads(raw))
    run = invocation['runId']
    if not re.fullmatch(r'[A-Za-z0-9-]{1,128}', run):
        raise ValueError('unsafe run ID')
    configuration = adapter.strict_loads(os.environ['MARKITECT_AGENT_CONFIG_JSON'])
    if (configuration.get('model') != 'gpt-6-luna' or not isinstance(configuration.get('modelOptions'), dict)
            or configuration['modelOptions'].get('model_reasoning_effort') != 'high'
            or configuration.get('providerVersion') != 'unavailable'):
        raise ValueError('runtime configuration does not match the declared Luna High pilot')
    job = pathlib.Path(args.queue_root) / run
    job.mkdir(parents=True, exist_ok=False)
    put(job / 'invocation.json', raw)
    prompt = adapter.make_prompt(invocation).encode('utf-8')
    put(job / 'prompt.txt', prompt)
    put(job / 'response-schema.json', json.dumps(adapter.RESPONSE_SCHEMA, indent=2).encode())
    request = invocation['request']
    put(job / 'ready.json', json.dumps({
        'runId': run,
        'role': request['role'],
        'projectionId': request['projectionId'],
        'modelDigest': request['modelDigest'],
        'sourceRevision': request['sourceRevision'],
        'inputDigest': invocation['inputDigest'],
        'promptDigest': digest(prompt),
        'modelConfigured': configuration['model'],
        'reasoningConfigured': 'high',
        'providerVersion': 'unavailable',
        'dispatchFile': 'dispatch.json',
        'dispatchRequiredFields': ['agentId', 'model', 'reasoningEffort'],
        'responseFile': 'response.json',
    }, indent=2).encode())
    deadline = time.monotonic() + 540
    response_file = job / 'response.json'
    dispatch_file = job / 'dispatch.json'
    while not response_file.is_file() or not dispatch_file.is_file():
        if time.monotonic() >= deadline:
            raise TimeoutError('desktop subagent dispatch or response missing within pilot bound')
        time.sleep(0.3)

    dispatch_raw = dispatch_file.read_bytes()
    dispatch = adapter.strict_loads(dispatch_raw)
    if not isinstance(dispatch, dict) or set(dispatch) != {'agentId', 'model', 'reasoningEffort'}:
        raise ValueError('dispatch.json has an unsupported shape')
    agent_id = dispatch.get('agentId')
    if not isinstance(agent_id, str) or not agent_id.strip():
        raise ValueError('dispatch.json requires the actual root-created collaboration agentId')
    if dispatch.get('model') != 'gpt-6-luna' or dispatch.get('reasoningEffort') != 'high':
        raise ValueError('dispatch.json must attest the configured gpt-6-luna/high task')

    response_raw = response_file.read_bytes()
    if len(response_raw) > 8 * 1024 * 1024:
        raise ValueError('response too large')
    put(job / 'raw-response.json', response_raw)
    normalized = adapter.normalize_codex_response(adapter.strict_loads(response_raw))
    if not isinstance(normalized, dict):
        raise ValueError('normalized response must be an object')
    normalized_raw = (json.dumps(normalized, ensure_ascii=False, separators=(',', ':')) + '\n').encode('utf-8')
    put(job / 'normalized-response.json', normalized_raw)

    request = invocation['request']
    echoed = {
        'apiVersion': invocation['apiVersion'],
        'runId': invocation['runId'],
        'nonce': invocation['nonce'],
        'role': request['role'],
        'inputDigest': invocation['inputDigest'],
    }
    for key, expected in echoed.items():
        if normalized.get(key) != expected:
            raise ValueError('subagent response does not echo invocation ' + key)

    response = normalized
    emitted = (json.dumps(response, ensure_ascii=False, separators=(',', ':')) + chr(10)).encode('utf-8')
    put(job / 'emitted-response.json', emitted)
    dispatch_link = {
        'path': 'dispatch.json',
        'digest': digest(dispatch_raw),
        'agentId': agent_id,
        'model': dispatch['model'],
        'reasoningEffort': dispatch['reasoningEffort'],
    }
    private_log = {
        'type': 'pilot.desktop-subagent-bridge',
        'runId': run,
        'inputDigest': invocation['inputDigest'],
        'requestDigest': digest(raw),
        'promptDigest': digest(prompt),
        'rawResponseDigest': digest(response_raw),
        'normalizedResponseDigest': digest(normalized_raw),
        'emittedResponseDigest': digest(emitted),
        'projectionId': request['projectionId'],
        'modelDigest': request['modelDigest'],
        'sourceRevision': request['sourceRevision'],
        'dispatchMetadata': dispatch_link,
        'configuredModel': 'gpt-6-luna',
        'configuredEffort': 'high',
        'responseModifiedByBridge': False,
        'providerVersion': 'unavailable',
    }
    log = pathlib.Path(os.environ['MARKITECT_AGENT_PRIVATE_LOG'])
    put(log, (json.dumps(private_log, ensure_ascii=False, separators=(',', ':')) + '\n').encode('utf-8'))
    sys.stdout.buffer.write(emitted)
    sys.stdout.buffer.flush()


if __name__ == '__main__':
    try:
        main()
    except Exception as exc:
        print(type(exc).__name__ + ': ' + str(exc), file=sys.stderr)
        raise SystemExit(2)