"""Bounded author-form normalization experiment; not product code."""
from pathlib import Path
from copy import deepcopy
import hashlib
import importlib.util
import json
import os
import subprocess
import sys
import yaml

prototype_path = Path(os.environ['TEMP']) / 'markitect-bootstrap-probe.py'
module_spec = importlib.util.spec_from_file_location('design_bootstrap', prototype_path)
bootstrap = importlib.util.module_from_spec(module_spec)
module_spec.loader.exec_module(bootstrap)
K = bootstrap.K
P = bootstrap.P

class InvalidAuthorForm(Exception):
    pass

def require(ok, code):
    if not ok:
        raise InvalidAuthorForm(code)

class UniqueLoader(yaml.SafeLoader):
    pass

def unique_mapping(loader, node, deep=False):
    loader.flatten_mapping(node)
    result = {}
    for key_node, value_node in node.value:
        key = loader.construct_object(key_node, deep=deep)
        require(isinstance(key, (str, int, float, bool, type(None))), 'author.invalid-key')
        require(key not in result, 'author.duplicate-key')
        result[key] = loader.construct_object(value_node, deep=deep)
    return result

UniqueLoader.add_constructor(yaml.resolver.BaseResolver.DEFAULT_MAPPING_TAG, unique_mapping)

def normalize(authored):
    result = []
    origins = []
    shared = []

    def properties(owner, value, depth=0):
        require(depth <= 32, 'inline.depth')
        if type(value) is not dict:
            return deepcopy(value)  # Existing canonical reference-list form.
        references = []
        for name, entry in value.items():
            require(type(name) is str and bool(name.strip()) and '/' not in name, 'inline.name')
            if type(entry) is str:
                shared.append((name, entry))
                references.append(entry)
                continue
            require(type(entry) is dict, 'inline.shape')
            require(not (entry.keys() - {'body', 'type', 'target', 'minCount', 'maxCount', 'values', 'properties'}), 'inline.unknown-field')
            require('body' in entry, 'inline.required-body')
            property_id = owner + '/' + name
            property_spec = {'name': name, **{key:deepcopy(v) for key,v in entry.items() if key != 'body'}}
            if property_spec.get('type') == 'object' and 'properties' in property_spec:
                property_spec['properties'] = properties(property_id, property_spec['properties'], depth + 1)
            result.append({'id':property_id,'kind':P,'body':entry['body'],'spec':property_spec})
            origins.append({'owner':owner,'key':name,'generatedId':property_id})
            references.append(property_id)
        return references

    for original in authored:
        node = deepcopy(original)
        if type(node) is dict and type(node.get('spec')) is dict:
            is_kind = node.get('kind') == K
            is_object_property = node.get('kind') == P and node['spec'].get('type') == 'object'
            if (is_kind or is_object_property) and 'properties' in node['spec']:
                node['spec']['properties'] = properties(node['id'], node['spec']['properties'])
        result.append(node)
    index = {}
    for node in result:
        require(node['id'] not in index, 'definition.duplicate')
        index[node['id']] = node
    for name, ref in shared:
        require(ref in index, 'inline.reference-unresolved')
        require(index[ref]['kind'] == P, 'inline.reference-not-property')
        require(index[ref]['spec'].get('name') == name, 'inline.reference-name')
    return result, origins

def main():
    base = Path(sys.argv[1])
    load = lambda name: list(yaml.load_all((base/name).read_text(encoding='utf-8'), Loader=UniqueLoader))
    seed = load('core.yaml')
    explicit = load('connector.yaml')
    compact = load('connector-compact.yaml')
    normalized, origins = normalize(compact)
    canonical = lambda nodes: json.dumps(sorted(nodes,key=lambda d:d['id']),sort_keys=True,ensure_ascii=False,separators=(',',':'))
    assert canonical(normalized) == canonical(explicit)
    a = bootstrap.compile_design(seed, explicit)
    b = bootstrap.compile_design(seed, normalized)
    assert a == b
    failures = {}
    E = 'connector/v1/Exchange'
    find = lambda ds: next(d for d in ds if d['id'] == E)

    def reject(name, mutate, expected):
        ds = deepcopy(compact)
        mutate(ds)
        try:
            normalize(ds)
        except InvalidAuthorForm as err:
            assert str(err) == expected, (name, str(err), expected)
            failures[name] = str(err)
        else:
            raise AssertionError(name + ' accepted')

    reject('inline-id-override', lambda ds: find(ds)['spec']['properties']['sender'].update(id='another'), 'inline.unknown-field')
    reject('inline-kind-override', lambda ds: find(ds)['spec']['properties']['sender'].update(kind=K), 'inline.unknown-field')
    reject('inline-name-override', lambda ds: find(ds)['spec']['properties']['sender'].update(name='receiver'), 'inline.unknown-field')
    reject('missing-inline-body', lambda ds: find(ds)['spec']['properties']['sender'].pop('body'), 'inline.required-body')
    reject('invalid-inline-name', lambda ds: find(ds)['spec']['properties'].update({'bad/name':deepcopy(find(ds)['spec']['properties']['sender'])}), 'inline.name')
    reject('invalid-inline-shape', lambda ds: find(ds)['spec']['properties'].update(sender=None), 'inline.shape')
    reject('derived-id-collision', lambda ds: ds.append(deepcopy(next(d for d in explicit if d['id']==E+'/sender'))), 'definition.duplicate')
    reject('shared-property-alias', lambda ds: find(ds)['spec']['properties'].update(sender=E+'/receiver'), 'inline.reference-name')
    try:
        list(yaml.load_all('properties:\n  sender: {}\n  sender: {}\n', Loader=UniqueLoader))
    except InvalidAuthorForm as err:
        assert str(err) == 'author.duplicate-key'
        failures['duplicate-yaml-key'] = str(err)
    else:
        raise AssertionError('duplicate YAML key accepted')

    # Sharing uses a reference to the existing Property, with the same field name.
    reused = deepcopy(compact)
    reused.append({'id':'connector/v1/OtherExchange','kind':K,'body':'A type using the same sender contract.','spec':{'properties':{'sender':E+'/sender'}}})
    shared, _ = normalize(reused)
    bootstrap.compile_design(seed, shared)
    assert sum(d['id']==E+'/sender' for d in shared)==1

    # A domain field named properties remains domain data; only the metacontexts normalize.
    nested = deepcopy(compact)
    nested.append({'id':'connector/v1/Example','kind':K,'body':'An example with a domain field named properties.','spec':{'properties':{'properties':{'body':'A structured domain value.','type':'object','minCount':1,'maxCount':1,'properties':{'text':{'body':'Domain text.','type':'string','minCount':1,'maxCount':1}}}}}})
    value = {'id':'project/v1/example','kind':'connector/v1/Example','body':'A concrete example.','spec':{'properties':{'text':'keep me'}}}
    nested.append(value)
    nested_normalized, _ = normalize(nested)
    assert next(d for d in nested_normalized if d['id']==value['id']) == value
    bootstrap.compile_design(seed, nested_normalized)

    report = {
        'scope':'Bounded standalone normalization and design-compiler probe; NOT current Markitect product support.',
        'sourceRevision':subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip(),
        'inputSha256':{n:hashlib.sha256((base/n).read_bytes()).hexdigest() for n in ['core.yaml','connector.yaml','connector-compact.yaml']},
        'normalizerPrototypeSha256':hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
        'bootstrapPrototypeSha256':hashlib.sha256(prototype_path.read_bytes()).hexdigest(),
        'baseline':'passed','compactTopLevelDefinitions':len(compact),'generatedPropertyDefinitions':len(origins),'normalizedProjectDefinitions':len(normalized),
        'sameNormalizedDefinitions':'passed','sameModelDigest':'passed','sameEdges':'passed',
        'modelDigest':b['digest'],'definitionsWithFixedCore':b['definitions'],'edges':len(b['edges']),
        'negativeCounterexamples':failures,'sharedPropertyWithoutCopy':'passed','nestedObjectNormalization':'passed','domainFieldNamedPropertiesUnchanged':'passed',
        'derivedPropertyOrigins':origins,
    }
    (base/'compact-probe.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
    print(json.dumps({k:report[k] for k in ['baseline','compactTopLevelDefinitions','generatedPropertyDefinitions','sameNormalizedDefinitions','sameModelDigest','sameEdges','definitionsWithFixedCore','edges']}))
    print('Rejected author-form counterexamples:',len(failures))

if __name__=='__main__':
    main()
