"""Bounded design experiment, independent of the Markitect product compiler."""
from pathlib import Path
from copy import deepcopy
import hashlib
import json
import math
import random
import subprocess
import sys
import yaml

K = 'core/v1/Kind'
P = 'core/v1/Property'
TYPES = ['string', 'boolean', 'integer', 'number', 'enum', 'object', 'reference']

def descriptor(name, type_, minimum, maximum, **extra):
    return dict(name=name, type=type_, minCount=minimum, maxCount=maximum, **extra)

# This fixed grammar is the bootstrap assumption. It is not inferred from prose.
META_KIND = {'properties': descriptor('properties', 'reference', 0, -1, target=P)}
META_PROPERTY = {
    'name': descriptor('name', 'string', 1, 1),
    'type': descriptor('type', 'enum', 1, 1, values=TYPES),
    'minCount': descriptor('minCount', 'integer', 1, 1),
    'maxCount': descriptor('maxCount', 'integer', 1, 1),
    'values': descriptor('values', 'string', 0, -1),
    'target': descriptor('target', 'reference', 0, 1, target=K),
    'properties': descriptor('properties', 'reference', 0, -1, target=P),
}

class Invalid(Exception):
    pass

def require(condition, code):
    if not condition:
        raise Invalid(code)

def compile_design(trusted_seed, project):
    require(len(trusted_seed) + len(project) <= 10000, 'limit.definitions')
    index = {}
    for origin, definitions in [('seed', trusted_seed), ('project', project)]:
        for d in definitions:
            require(type(d) is dict and set(d) == {'id', 'kind', 'body', 'spec'}, 'definition.envelope')
            require(all(type(d[f]) is str and d[f].strip() for f in ('id', 'kind', 'body')), 'definition.text')
            require(type(d['spec']) is dict, 'definition.spec')
            require(origin == 'seed' or not d['id'].startswith('core/'), 'core.reserved')
            require(d['id'] not in index, 'definition.duplicate')
            index[d['id']] = d

    for d in index.values():
        require(d['kind'] in index, 'kind.unresolved')
        require(index[d['kind']]['kind'] == K, 'kind.not-kind')
    require(K in index and P in index and index[K]['kind'] == K and index[P]['kind'] == K, 'core.start')
    kinds = {i: d for i, d in index.items() if d['kind'] == K}
    properties = {i: d for i, d in index.items() if d['kind'] == P}
    edges = []

    def validate_spec(values, fields, owner, path='spec', collect=False, depth=0):
        require(depth <= 32 and type(values) is dict, 'value.object')
        require(not (values.keys() - fields.keys()), 'spec.unknown-property')
        for name, p in fields.items():
            if name not in values:
                require(p['minCount'] == 0, 'property.required')
                continue
            v = values[name]
            fieldpath = path + '.' + name
            if p['maxCount'] == 1:
                items = [(fieldpath, v)]
            else:
                require(type(v) is list, 'value.list')
                require(len(v) <= 10000 and len(v) >= p['minCount'] and (p['maxCount'] == -1 or len(v) <= p['maxCount']), 'property.cardinality')
                items = [(fieldpath + '[' + str(n) + ']', x) for n, x in enumerate(v)]
            seen = set()
            for valuepath, item in items:
                t = p['type']
                require(item is not None, 'value.null')
                if t == 'string':
                    require(type(item) is str, 'value.string')
                elif t == 'boolean':
                    require(type(item) is bool, 'value.boolean')
                elif t == 'integer':
                    require(type(item) is int, 'value.integer')
                elif t == 'number':
                    require(type(item) in (int, float) and math.isfinite(item), 'value.number')
                elif t == 'enum':
                    require(type(item) is str and item in p['values'], 'value.enum')
                elif t == 'reference':
                    require(type(item) is str and item in index, 'reference.unresolved')
                    require(index[item]['kind'] == p['target'], 'reference.target-kind')
                    require(item not in seen, 'reference.duplicate')
                    seen.add(item)
                    if collect:
                        edges.append({'from': owner, 'path': valuepath, 'to': item})
                elif t == 'object':
                    validate_spec(item, fields_for(p.get('properties', [])), owner, valuepath, collect, depth + 1)
                else:
                    raise Invalid('property.type')

    # Bootstrap validation against the fixed metacontracts precedes construction.
    for d in kinds.values():
        validate_spec(d['spec'], META_KIND, d['id'])
    for d in properties.values():
        p = d['spec']
        validate_spec(p, META_PROPERTY, d['id'])
        require(bool(p['name'].strip()), 'property.name')
        require(p['minCount'] >= 0 and p['maxCount'] >= -1 and (p['maxCount'] == -1 or p['minCount'] <= p['maxCount']), 'property.cardinality-contract')
        extras = set(p) & {'values', 'target', 'properties'}
        allowed = {'enum': {'values'}, 'reference': {'target'}, 'object': {'properties'}}.get(p['type'], set())
        require(extras <= allowed, 'property.type-fields')
        if p['type'] == 'reference':
            require('target' in p, 'property.target-required')
        if p['type'] == 'enum':
            require(bool(p.get('values')) and len(set(p['values'])) == len(p['values']), 'property.enum-values')

    def fields_for(refs, stack=()):
        fields = {}
        for ref in refs:
            require(ref in properties, 'property.not-property')
            p = properties[ref]['spec']
            require(p['name'] not in fields, 'property.duplicate-name')
            require(ref not in stack and len(stack) <= 32, 'property.object-cycle')
            if p['type'] == 'object':
                fields_for(p.get('properties', []), stack + (ref,))
            fields[p['name']] = p
        return fields

    contracts = {i: fields_for(d['spec'].get('properties', [])) for i, d in kinds.items()}
    require(contracts[K] == META_KIND and contracts[P] == META_PROPERTY, 'core.meta-contract')
    # Every node, including the seed, is also checked using the resulting graph.
    for d in index.values():
        edges.append({'from': d['id'], 'path': 'kind', 'to': d['kind']})
        validate_spec(d['spec'], contracts[d['kind']], d['id'], collect=True)
    canonical = json.dumps([index[i] for i in sorted(index)], ensure_ascii=False, sort_keys=True, separators=(',', ':')).encode()
    return {'digest': hashlib.sha256(canonical).hexdigest(), 'definitions': len(index), 'kinds': len(kinds), 'properties': len(properties), 'edges': sorted(edges, key=lambda e: (e['from'], e['path'], e['to']))}

def main():
    base = Path(sys.argv[1])
    seed = list(yaml.safe_load_all((base / 'core.yaml').read_text(encoding='utf-8')))
    project = list(yaml.safe_load_all((base / 'connector.yaml').read_text(encoding='utf-8')))
    good = compile_design(seed, project)
    byid = lambda docs, i: next(d for d in docs if d['id'] == i)
    E = 'connector/v1/Exchange'
    exchange = 'project/v1/list-documents'
    failures = {}

    def reject(name, mutate, expected):
        variant = deepcopy(project)
        mutate(variant)
        try:
            compile_design(seed, variant)
        except Invalid as err:
            assert str(err) == expected, (name, str(err), expected)
            failures[name] = str(err)
        else:
            raise AssertionError(name + ' accepted')

    reject('duplicate-identity', lambda ds: ds.append(deepcopy(ds[0])), 'definition.duplicate')
    reject('replace-fixed-core', lambda ds: ds.append(deepcopy(seed[0])), 'core.reserved')
    reject('unknown-kind', lambda ds: byid(ds, exchange).update(kind='connector/v2/Exchange'), 'kind.unresolved')
    reject('kind-points-to-instance', lambda ds: byid(ds, exchange).update(kind='project/v1/api'), 'kind.not-kind')
    reject('property-list-points-to-instance', lambda ds: byid(ds, E)['spec']['properties'].append('project/v1/api'), 'reference.target-kind')
    def duplicate_name(ds):
        d = deepcopy(byid(ds, E+'/sender'))
        d['id'] = E+'/another-sender'
        ds.append(d)
        byid(ds, E)['spec']['properties'].append(d['id'])
    reject('duplicate-property-name', duplicate_name, 'property.duplicate-name')
    reject('property-target-is-instance', lambda ds: byid(ds, E+'/sender')['spec'].update(target='project/v1/api'), 'reference.target-kind')
    reject('property-target-missing', lambda ds: byid(ds, E+'/sender')['spec'].pop('target'), 'property.target-required')
    reject('invalid-cardinality-contract', lambda ds: byid(ds, E+'/sender')['spec'].update(minCount=2, maxCount=1), 'property.cardinality-contract')
    reject('unknown-type-operator', lambda ds: byid(ds, E+'/sender')['spec'].update(type='custom-parser'), 'value.enum')
    reject('unknown-formal-constraint', lambda ds: byid(ds, E+'/timeoutSeconds')['spec'].update(minimum=1), 'spec.unknown-property')
    reject('missing-required-property', lambda ds: byid(ds, exchange)['spec'].pop('replyChannel'), 'property.required')
    reject('wrong-reference-kind', lambda ds: byid(ds, exchange)['spec'].update(sender='project/v1/http-edge'), 'reference.target-kind')
    reject('missing-exact-version-target', lambda ds: byid(ds, E+'/sender')['spec'].update(target='connector/v2/Container'), 'reference.unresolved')
    reject('unknown-instance-field', lambda ds: byid(ds, exchange)['spec'].update(roles=['sender']), 'spec.unknown-property')
    reject('arbitrary-self-typed-node', lambda ds: ds.append({'id':'project/v1/self','kind':'project/v1/self','body':'No additional metaclass rule.','spec':{}}), 'kind.not-kind')
    shuffled = deepcopy(project)
    random.Random(0).shuffle(shuffled)
    assert compile_design(seed, shuffled)['digest'] == good['digest']
    for i in [E, E+'/sender', exchange]:
        variant = deepcopy(project)
        byid(variant, i)['body'] += ' An additional binding statement.'
        assert compile_design(seed, variant)['digest'] != good['digest']
    semantic = deepcopy(project)
    byid(semantic, exchange)['spec']['maxAttempts'] = 0
    compile_design(seed, semantic)
    semantic = deepcopy(project)
    byid(semantic, exchange)['spec']['replyChannel'] = 'project/v1/requests'
    compile_design(seed, semantic)
    # A project-defined Kind named Kind is ordinary data without core privileges.
    ordinary = deepcopy(project)
    ordinary.append({'id':'connector/v1/Kind','kind':K,'body':'An ordinary project kind.','spec':{'properties':[]}})
    ordinary.append({'id':'project/v1/ordinary','kind':'connector/v1/Kind','body':'An ordinary instance.','spec':{}})
    compile_design(seed, ordinary)
    report = {
        'scope': 'Bounded standalone design prototype; NOT the current Markitect compiler or Context/Impact/Apply; no Connector runtime executed.',
        'sourceRevision': subprocess.check_output(['git','rev-parse','HEAD'], text=True).strip(),
        'inputSha256': {n:hashlib.sha256((base/n).read_bytes()).hexdigest() for n in ['core.yaml','connector.yaml']},
        'prototypeSha256': hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
        'trustedSeedCanonicalDigest': hashlib.sha256(json.dumps(seed, sort_keys=True, ensure_ascii=False).encode()).hexdigest(),
        'baseline': 'passed', 'trustedCoreDefinitions': len(seed), 'projectDefinitions': len(project),
        'definitions': good['definitions'], 'kinds': good['kinds'], 'properties': good['properties'],
        'edges': len(good['edges']), 'modelDigest': good['digest'],
        'negativeCounterexamples': failures,
        'selfTypedKindAndSeedSelfValidation': 'passed',
        'inputOrderIndependence': 'passed', 'kindPropertyAndInstanceBodyInDigest':'passed',
        'projectKindNamedKindHasNoSpecialMeaning':'passed',
        'semanticLimits': ['maxAttempts=0 is structurally accepted', 'replyChannel=requests is structurally accepted'],
        'exchangeEdges': [e for e in good['edges'] if e['from'] == exchange],
        'kindEdges': [e for e in good['edges'] if e['from'] == E],
        'senderPropertyEdges': [e for e in good['edges'] if e['from'] == E+'/sender'],
    }
    (base/'probe.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n', encoding='utf-8')
    print(json.dumps({k:report[k] for k in ['baseline','trustedCoreDefinitions','projectDefinitions','definitions','kinds','properties','edges']}, ensure_ascii=False))
    print('Rejected counterexamples:', len(failures))

if __name__ == '__main__':
    main()
