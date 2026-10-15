"""Local bounded package/identity experiment, not Markitect product code."""
from pathlib import Path
from copy import deepcopy
import hashlib
import importlib.util
import json
import os
import subprocess
import sys
import yaml

temp = Path(os.environ['TEMP'])
spec = importlib.util.spec_from_file_location('bootstrap_identity', temp/'markitect-bootstrap-probe.py')
kernel = importlib.util.module_from_spec(spec)
spec.loader.exec_module(kernel)
CORE='markitect/core'; VOCAB='example/connector-vocabulary'; ROOT='example/connector-project'
key=lambda package,local: json.dumps([package,local],ensure_ascii=False,separators=(',',':'))
K=key(CORE,'Kind'); P=key(CORE,'Property')
oldK,oldP=kernel.K,kernel.P
kernel.K=K; kernel.P=P
for table in [kernel.META_KIND,kernel.META_PROPERTY]:
    for p in table.values():
        if p.get('target')==oldK: p['target']=K
        if p.get('target')==oldP: p['target']=P

class Invalid(Exception): pass
def require(ok,code):
    if not ok: raise Invalid(code)
class UniqueLoader(yaml.SafeLoader): pass
def unique_mapping(loader,node,deep=False):
    loader.flatten_mapping(node); result={}
    for kn,vn in node.value:
        k=loader.construct_object(kn,deep=deep)
        require(type(k) is str and k not in result,'author.duplicate-or-invalid-key')
        result[k]=loader.construct_object(vn,deep=deep)
    return result
UniqueLoader.add_constructor(yaml.resolver.BaseResolver.DEFAULT_MAPPING_TAG,unique_mapping)
sha=lambda raw: hashlib.sha256(raw).hexdigest()

def assemble(root,lock,catalog,trusted_core_hash):
    pins={}
    for pin in lock['packages']:
        require(pin['package'] not in pins,'import.duplicate-pin')
        pins[pin['package']]=pin
    rootpkg=root['package']; require(rootpkg!=CORE,'import.core-root')
    scopes={rootpkg:root}; selected={}
    def visit(dep):
        pkg,version=dep['package'],dep['version']
        require(pkg!=rootpkg,'import.root-collision')
        if pkg in selected:
            require(selected[pkg]['version']==version,'import.version-conflict'); return
        require((pkg,version) in catalog,'import.unavailable-version')
        require(pkg in pins,'import.unlocked')
        require(pins[pkg]['version']==version,'import.lock-mismatch')
        artifact=catalog[(pkg,version)]
        require(sha(artifact['raw'])==pins[pkg]['sha256'],'import.content-mismatch')
        require(pkg!=CORE or sha(artifact['raw'])==trusted_core_hash,'import.core-not-trusted')
        require(artifact['data']['package']==pkg and artifact['data']['version']==version,'import.manifest-mismatch')
        selected[pkg]=dict(package=pkg,version=version,sha256=pins[pkg]['sha256'])
        scopes[pkg]=artifact['data']
        for child in artifact['data']['imports'].values(): visit(child)
    for dep in root['imports'].values(): visit(dep)
    require(CORE in selected,'import.missing-core')
    require(set(pins)==set(selected),'import.unused-pin')

    def valid_local(local):
        return type(local) is str and bool(local) and ':' not in local and all(p and p not in ('.','..') for p in local.split('/'))
    def resolve(package,reference):
        require(type(reference) is str,'reference.syntax')
        if ':' in reference:
            parts=reference.split(':'); require(len(parts)==2,'reference.syntax')
            alias,local=parts
            require(alias in scopes[package]['imports'],'import.unknown-alias')
            package=scopes[package]['imports'][alias]['package']
        else: local=reference
        require(valid_local(local),'reference.syntax')
        return key(package,local)

    nodes=[]; contexts={}; pending_shared=[]
    def append(package,local,kind,body,values):
        ident=key(package,local)
        nodes.append({'id':ident,'kind':kind,'body':body,'spec':values}); contexts[ident]=package
    def fields(package,owner,authored,depth=0):
        require(depth<=32,'inline.depth')
        if type(authored) is list: return [resolve(package,r) for r in authored]
        require(type(authored) is dict,'inline.shape')
        refs=[]
        for name,entry in authored.items():
            require(valid_local(name) and '/' not in name,'inline.name')
            if type(entry) is str:
                ref=resolve(package,entry); pending_shared.append((name,ref)); refs.append(ref); continue
            require(type(entry) is dict and not (entry.keys()-{'id','body','type','target','minCount','maxCount','values','properties'}),'inline.fields')
            segment=entry.get('id',name)
            require(valid_local(segment) and '/' not in segment,'inline.identity')
            local=owner+'/'+segment
            p={'name':name,**{n:deepcopy(v) for n,v in entry.items() if n not in ('id','body')}}
            if 'target' in p: p['target']=resolve(package,p['target'])
            if p.get('type')=='object' and 'properties' in p: p['properties']=fields(package,local,p['properties'],depth+1)
            append(package,local,P,entry['body'],p); refs.append(key(package,local))
        return refs

    for pkg,scope in scopes.items():
        for d in scope['definitions']:
            require(set(d)=={'id','kind','body','spec'} and valid_local(d['id']),'definition.envelope')
            kind=resolve(pkg,d['kind']); values=deepcopy(d['spec'])
            if kind==K and 'properties' in values: values['properties']=fields(pkg,d['id'],values['properties'])
            if kind==P:
                if 'target' in values: values['target']=resolve(pkg,values['target'])
                if values.get('type')=='object' and 'properties' in values: values['properties']=fields(pkg,d['id'],values['properties'])
            append(pkg,d['id'],kind,d['body'],values)
    index={}
    for d in nodes:
        require(d['id'] not in index,'definition.duplicate'); index[d['id']]=d
    for name,ref in pending_shared:
        require(ref in index and index[ref]['kind']==P and index[ref]['spec']['name']==name,'inline.shared-name')
    seed=[d for d in nodes if contexts[d['id']]==CORE]
    meta=[d for d in nodes if contexts[d['id']]!=CORE and d['kind'] in (K,P)]
    kernel.compile_design(seed,meta)
    types={i:d for i,d in index.items() if d['kind']==K}
    properties={i:d['spec'] for i,d in index.items() if d['kind']==P}
    def convert_values(package,values,refs):
        contracts={properties[r]['name']:properties[r] for r in refs}; out={}
        def single(value,p):
            if p['type']=='reference' and type(value) is str: return resolve(package,value)
            if p['type']=='object' and type(value) is dict: return convert_values(package,value,p.get('properties',[]))
            return value
        for name,value in values.items():
            if name not in contracts: out[name]=value; continue
            p=contracts[name]
            out[name]=[single(v,p) for v in value] if p['maxCount']!=1 and type(value) is list else single(value,p)
        return out
    for d in nodes:
        if d['kind'] not in (K,P) and d['kind'] in types:
            d['spec']=convert_values(contexts[d['id']],d['spec'],types[d['kind']]['spec'].get('properties',[]))
    model=kernel.compile_design(seed,[d for d in nodes if contexts[d['id']]!=CORE])
    bindings=sorted(selected.values(),key=lambda x:x['package'])
    binding_payload={'rootPackage':rootpkg,'graphDigest':model['digest'],'packages':bindings}
    snapshot=sha(json.dumps(binding_payload,sort_keys=True,separators=(',',':')).encode())
    return {'snapshotDigest':snapshot,'graphDigest':model['digest'],'bindings':bindings,'definitions':len(nodes),'edges':len(model['edges']),'index':index,'resolve':lambda r:resolve(rootpkg,r),'graph':model['edges']}

def main():
    base=Path(sys.argv[1])
    load=lambda n: yaml.load((base/n).read_text(encoding='utf-8'),Loader=UniqueLoader)
    catalog={}
    for name in ['core-1.0.0.yaml','vocabulary-1.0.0.yaml','vocabulary-1.1.0.yaml','vocabulary-2.0.0.yaml']:
        raw=(base/name).read_bytes(); data=yaml.load(raw.decode(),Loader=UniqueLoader)
        catalog[(data['package'],data['version'])]={'data':data,'raw':raw}
    trusted=sha((base/'core-1.0.0.yaml').read_bytes())
    locks=load('locks.yaml'); roots={s:load('project-'+s+'.yaml') for s in ['a','b','c']}
    results={s:assemble(roots[s],locks[s],catalog,trusted) for s in roots}
    prop=key(VOCAB,'Exchange/sender'); exchange=key(ROOT,'list-documents')
    a,b,c=[results[s] for s in ['a','b','c']]
    assert set(a['index'])==set(b['index'])==set(c['index'])
    assert a['graph']==b['graph'] and a['snapshotDigest']!=b['snapshotDigest']
    assert b['index'][prop]['spec']['name']=='sender' and c['index'][prop]['spec']['name']=='requester'
    assert c['resolve']('connector:Exchange/sender')==prop and prop in c['index']
    assert c['index'][exchange]['spec']['requester']==key(ROOT,'api')
    failures={}
    def reject(name,root,lock,cat,expected):
        try: assemble(root,lock,cat,trusted)
        except (Invalid,kernel.Invalid) as err:
            assert str(err)==expected,(name,str(err),expected); failures[name]=str(err)
        else: raise AssertionError(name+' accepted')
    reject('version-2-with-old-project-field',load('project-c-stale.yaml'),locks['c'],catalog,'spec.unknown-property')
    reject('new-import-with-old-lock',roots['b'],locks['a'],catalog,'import.lock-mismatch')
    unknown=deepcopy(roots['a']); unknown['imports']['connector']['version']='1.9.0'
    reject('unavailable-exact-version',unknown,locks['a'],catalog,'import.unavailable-version')
    conflict=deepcopy(roots['a']); conflict['imports']['anotherVersion']={'package':VOCAB,'version':'2.0.0'}
    reject('two-versions-of-one-package',conflict,locks['a'],catalog,'import.version-conflict')
    corrupt=deepcopy(catalog); corrupt[(VOCAB,'1.0.0')]['raw']+=b'\n# changed bytes\n'
    reject('same-version-different-content',roots['a'],locks['a'],corrupt,'import.content-mismatch')
    noalias=deepcopy(roots['a']); next(d for d in noalias['definitions'] if d['id']=='list-documents')['kind']='unknown:Exchange'
    reject('unknown-import-alias',noalias,locks['a'],catalog,'import.unknown-alias')
    unqualified=deepcopy(roots['a']); next(d for d in unqualified['definitions'] if d['id']=='list-documents')['kind']='Exchange'
    reject('no-global-search-for-unqualified-kind',unqualified,locks['a'],catalog,'kind.unresolved')
    def repackage(cat,pkg,version,lock):
        item=cat[(pkg,version)]; item['raw']=yaml.safe_dump(item['data'],sort_keys=False).encode()
        next(p for p in lock['packages'] if p['package']==pkg)['sha256']=sha(item['raw'])
    foreign=deepcopy(catalog); foreignlock=deepcopy(locks['a'])
    next(d for d in foreign[(VOCAB,'1.0.0')]['data']['definitions'] if d['id']=='Exchange')['spec']['properties']['sender']['target']='connector:Container'
    repackage(foreign,VOCAB,'1.0.0',foreignlock)
    reject('library-cannot-use-roots-import-alias',roots['a'],foreignlock,foreign,'import.unknown-alias')
    wrong=deepcopy(roots['c']); next(d for d in wrong['definitions'] if d['id']=='list-documents')['spec']['requester']='connector:Container'
    reject('kind-definition-is-not-container-instance',wrong,locks['c'],catalog,'reference.target-kind')

    # Changing the root-local alias preserves all normalized targets and bindings.
    aliased=deepcopy(roots['a']); aliased['imports']['wire']=aliased['imports'].pop('connector')
    for d in aliased['definitions']:
        if d['kind'].startswith('connector:'): d['kind']='wire:'+d['kind'].split(':',1)[1]
    alias_result=assemble(aliased,locks['a'],catalog,trusted)
    assert alias_result['snapshotDigest']==a['snapshotDigest']

    # Forgetting the stable ID creates a new Property; it need not break a model without old references.
    lost=deepcopy(catalog); lostlock=deepcopy(locks['c'])
    next(d for d in lost[(VOCAB,'2.0.0')]['data']['definitions'] if d['id']=='Exchange')['spec']['properties']['requester'].pop('id')
    repackage(lost,VOCAB,'2.0.0',lostlock)
    lost_result=assemble(roots['c'],lostlock,lost,trusted)
    assert prop not in lost_result['index'] and key(VOCAB,'Exchange/requester') in lost_result['index']

    def summary(result):
        return {n:result[n] for n in ['snapshotDigest','graphDigest','bindings','definitions','edges']}
    files=sorted(p for p in base.glob('*.yaml'))
    report={
        'scope':'Bounded standalone package-binding, reference-resolution and stable-inline-ID probe using the earlier independent structural prototype; NOT product loader support or Connector execution.',
        'sourceRevision':subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip(),
        'inputSha256':{p.name:sha(p.read_bytes()) for p in files},
        'resolverPrototypeSha256':sha(Path(__file__).read_bytes()),
        'structuralPrototypeSha256':sha((temp/'markitect-bootstrap-probe.py').read_bytes()),
        'canonicalIdentityEncoding':'JSON pair [package,id]; alias strings are resolved in the declaring package before structural validation.',
        'states':{s:summary(results[s]) for s in results},
        'negativeCounterexamples':failures,
        'sameLogicalIdentitiesAcrossThreeStates':True,
        'version11ChangesBodyButNotFieldsOrReferenceEdges':True,
        'newRuntimeTimingObligationVerified':False,
        'samePropertyIdentityAfterFieldRename':json.loads(prop),
        'propertyFieldNames':{'a':a['index'][prop]['spec']['name'],'b':b['index'][prop]['spec']['name'],'c':c['index'][prop]['spec']['name']},
        'referenceToPropertyResolvesInAllStates':all(r['resolve']('connector:Exchange/sender')==prop and prop in r['index'] for r in results.values()),
        'aliasRenamePreservesSnapshotDigest':True,
        'forgottenStableIdCounterexample':'The migrated project remains structurally valid, but Exchange/sender is absent and Exchange/requester is new; old references would be unresolved.',
        'limitations':['No generic body-link parser or free-text rewrite.','No package registry, network fetching, signatures or production input hardening.','No semantic completeness, backward compatibility or runtime conformance proof.'],
    }
    (base/'probe.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
    print(json.dumps({'states':{s:{'definitions':results[s]['definitions'],'edges':results[s]['edges']} for s in results},'rejectedCounterexamples':len(failures),'propertyNames':report['propertyFieldNames'],'stablePropertyIdentity':json.loads(prop),'aliasRenamePreservesBinding':True}))

if __name__=='__main__': main()
