"""Offline structural preflight for pinned Classic Main agentexec v1alpha1.

No model execution, semantic interpretation, evidence manufacture or DTO repair.
Product parsing/verification remains authoritative. Design requires its own
explicit compatible public contract before this profile can be selected.
"""
import argparse
import hashlib
import json
import pathlib
import re

API='markitect.example.org/agent-execution/v1alpha1'
MODES={'0600','0644','0755'}
ARRAYS={'candidateFiles','evidenceRefs','verifierObservations','uncertainty'}
FIELDS={'apiVersion','runId','nonce','role','inputDigest','outcome',*ARRAYS,'candidateJson','usage'}
class Invalid(ValueError): pass
class Pairs(list): pass
def need(ok,message):
    if not ok:raise Invalid(message)
def strict(data):
    need(0<len(data)<=32<<20,'JSON byte bound')
    def convert(value,opaque=False):
        if isinstance(value,Pairs):
            out={};seen=set()
            for key,child in value:
                folded=key if opaque else key.casefold()
                need(folded not in seen,'duplicate or aliased JSON key');seen.add(folded)
                out[key]=convert(child,opaque or key in {'candidateJson','context'})
            return out
        if isinstance(value,list):return [convert(x,opaque) for x in value]
        return value
    try:
        parsed=json.loads(data.decode('utf8'),object_pairs_hook=Pairs,parse_constant=lambda _: (_ for _ in ()).throw(Invalid('non-finite number')))
        return convert(parsed)
    except (UnicodeError,json.JSONDecodeError) as exc:raise Invalid('invalid UTF-8 JSON') from exc
def text(value,limit=4096):
    if not isinstance(value,str):return False
    try:return 0<len(value.encode('utf8'))<=limit
    except UnicodeError:return False
def path(value):
    need(text(value) and not any(ord(c)<32 or ord(c)==127 or c in '\\:<>?*|' for c in value) and not value.startswith('/'),'nonportable path')
    parts=value.split('/')
    need(all(p and p not in {'.','..'} and p.casefold()!='.git' and not p.endswith((' ','.')) for p in parts),'unsafe path component')
    reserved={'con','prn','aux','nul',*[f'com{i}' for i in range(1,10)],*[f'lpt{i}' for i in range(1,10)]}
    need(all(p.split('.')[0].casefold() not in reserved for p in parts),'reserved path component')
    return value
def validate(invocation_bytes,response_bytes,allowed_paths,allowed_modes=None):
    invocation=strict(invocation_bytes);r=strict(response_bytes)
    need(isinstance(invocation,dict) and isinstance(r,dict),'object envelope required')
    need(set(invocation)=={'apiVersion','runId','nonce','inputDigest','request'},'closed invocation DTO')
    need(set(r)<=FIELDS and FIELDS-{'candidateJson','usage'}<=set(r),'closed response DTO')
    req=invocation.get('request',{});need(isinstance(req,dict),'request object required');role=req.get('role')
    need(isinstance(req,dict) and set(req)=={'role','sourceRevision','modelDigest','modulePin','projectionId','scopeIds','policyIds','context','artifacts'},'closed request DTO')
    need(isinstance(req['context'],dict),'opaque context object required')
    for key in ['sourceRevision','modelDigest','modulePin','projectionId']:need(text(req[key]),'request identity field: '+key)
    need(re.fullmatch('sha256:[0-9a-f]{64}',req['modelDigest']) is not None,'model digest shape')
    for key in ['scopeIds','policyIds']:
        need(isinstance(req[key],list) and len(req[key])<=128 and all(text(x) for x in req[key]),'request ID array: '+key)
    need(isinstance(req['artifacts'],list) and len(req['artifacts'])<=128,'request artifact array')
    for artifact in req['artifacts']:
        need(isinstance(artifact,dict) and set(artifact)=={'path','mode','digest','content'},'closed request artifact DTO')
        path(artifact['path']);need(artifact['mode'] in MODES and text(artifact['digest']) and isinstance(artifact['content'],str),'request artifact shape')
    need(role in {'executor','verifier','infer'},'unadmitted role')
    for key in ['apiVersion','runId','nonce','inputDigest']:
        need(r.get(key)==invocation.get(key) and text(r.get(key)),'identity mismatch: '+key)
    need(r['apiVersion']==API and r['role']==role,'API/role mismatch')
    need(re.fullmatch('[0-9a-f]{32}',r['runId']) is not None and re.fullmatch('[0-9a-f]{32}',r['nonce']) is not None,'identity shape')
    need(re.fullmatch('sha256:[0-9a-f]{64}',r['inputDigest']) is not None,'digest shape')
    for key in ARRAYS:need(isinstance(r[key],list) and len(r[key])<=128,'required bounded array: '+key)
    outcomes={'executor':{'proposed','failed','incomplete','escalated'},'verifier':{'passed','failed','incomplete','escalated'},'infer':{'proposed','failed','incomplete','escalated'}}
    need(r['outcome'] in outcomes[role],'role outcome')
    if role=='executor':
        need(not r['verifierObservations'] and 'candidateJson' not in r,'executor has foreign-role output')
        need(r['outcome']!='proposed' or bool(r['candidateFiles']),'empty proposal')
    elif role=='verifier':
        need(not r['candidateFiles'] and 'candidateJson' not in r,'verifier has candidate output')
        need(r['outcome'] not in {'passed','failed'} or bool(r['verifierObservations']),'missing verifier observations')
    else:
        need(not r['candidateFiles'] and not r['verifierObservations'],'infer has foreign-role output')
        need(r['outcome']!='proposed' or isinstance(r.get('candidateJson'),dict),'missing inference object')
    if 'candidateJson' in r:need(isinstance(r['candidateJson'],dict) and len(json.dumps(r['candidateJson']).encode())<=8<<20,'inference object bound')
    permitted={path(x) for x in allowed_paths};modes=set(allowed_modes or MODES);need(modes<=MODES and bool(modes),'unadmitted candidate mode policy')
    seen=set();size=0
    for f in r['candidateFiles']:
        need(isinstance(f,dict) and set(f)=={'path','mode','content'},'closed candidate-file DTO')
        name=path(f['path']);need(name in permitted,'outside explicit write allowlist');need(name.casefold() not in seen,'duplicate path alias');seen.add(name.casefold())
        need(f['mode'] in modes,'unsupported candidate mode');need(isinstance(f['content'],str),'candidate content string')
        try:size+=len(f['content'].encode('utf8'))
        except UnicodeError as exc:raise Invalid('candidate content invalid UTF-8') from exc
        need(size<=8<<20,'candidate bytes bound')
    supplied=set(req.get('scopeIds',[]))|set(req.get('policyIds',[]))|{x['path'] for x in req.get('artifacts',[])}
    refs=r['evidenceRefs'];need(all(text(x) and x in supplied for x in refs),'unsupplied evidence reference');need(len(refs)==len(set(refs)),'duplicate evidence reference')
    for o in r['verifierObservations']:
        need(isinstance(o,dict) and set(o)=={'subject','outcome','detail'},'closed observation DTO')
        need(text(o['subject']) and text(o['detail']) and o['outcome'] in {'passed','failed','incomplete','escalated'},'invalid observation')
    need(all(text(x) for x in r['uncertainty']),'invalid uncertainty')
    usage=r.get('usage')
    if usage is not None:
        fields={'source','inputTokens','outputTokens','cachedTokens','toolCalls'}
        need(isinstance(usage,dict) and set(usage)<=fields and usage.get('source')=='provider-reported','invalid usage source')
        values=[usage.get(k) for k in fields-{'source'}]
        need(any(v is not None for v in values) and all(v is None or type(v) is int and 0<=v<2**63 for v in values),'invalid usage counts')
    return {'structuralPreflight':'passed','profile':'classic-main-agentexec-v1alpha1','responseSha256':hashlib.sha256(response_bytes).hexdigest(),
            'invocationSha256':hashlib.sha256(invocation_bytes).hexdigest(),'semanticAcceptance':False,'productValidationRequired':True}
def publish(invocation_path,response_path,output_path,allowed_paths,forbidden_roots):
    need(bool(forbidden_roots),'explicit audited roots required for publication')
    raw=pathlib.Path(response_path).read_bytes();result=validate(pathlib.Path(invocation_path).read_bytes(),raw,allowed_paths)
    out=pathlib.Path(output_path).resolve()
    need(all(not out.is_relative_to(pathlib.Path(x).resolve()) for x in forbidden_roots),'publish target is inside audited workspace')
    need(not out.exists(),'response publication already exists')
    temporary=out.with_name(out.name+'.tmp');need(not temporary.exists(),'temporary response exists')
    with temporary.open('xb') as stream:stream.write(raw)
    temporary.replace(out)
    return result
if __name__=='__main__':
    a=argparse.ArgumentParser();a.add_argument('--invocation',required=True);a.add_argument('--response',required=True);a.add_argument('--allowlist',required=True);a.add_argument('--publish');a.add_argument('--forbidden-root',action='append',default=[]);x=a.parse_args()
    allow=json.loads(pathlib.Path(x.allowlist).read_text())
    result=publish(x.invocation,x.response,x.publish,allow,x.forbidden_root) if x.publish else validate(pathlib.Path(x.invocation).read_bytes(),pathlib.Path(x.response).read_bytes(),allow)
    print(json.dumps(result,sort_keys=True))
