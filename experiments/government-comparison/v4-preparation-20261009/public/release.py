"""Stage-only public materialization; no task execution or Actor activation."""
import argparse, ast, hashlib, json, pathlib, shutil
ROOT=pathlib.Path(__file__).resolve().parent
def staged_checks(cutoff,condition):
    if cutoff not in range(1,7) or condition not in {'greenfield','brownfield'}:raise ValueError('unadmitted cutoff/condition')
    tree=ast.parse((ROOT/'checks.py').read_text(encoding='utf8'))
    for node in ast.walk(tree):
        if isinstance(node,ast.ClassDef) and node.name=='Checks':
            remove=({'inventory','unknown_inventory'} if cutoff<2 else set())|({'restart'} if cutoff<3 else set())
            node.body=[x for x in node.body if not isinstance(x,ast.FunctionDef) or x.name not in remove]
        if isinstance(node,ast.FunctionDef) and node.name=='run':
            out=[]
            for stmt in node.body:
                text=ast.unparse(stmt.test) if isinstance(stmt,ast.If) else ''
                if text=="condition == 'brownfield'":
                    if condition=='brownfield':out.extend(stmt.body)
                    continue
                boundary=(cutoff==1 and text=='through < 2') or (cutoff in {2,3} and text=='through < 4') or (cutoff==4 and text=='through >= 5')
                if boundary:
                    out.append(ast.Return(value=ast.Name(id='checkpoint',ctx=ast.Load())));break
                out.append(stmt)
            node.body=out
    if cutoff==1:
        for node in ast.walk(tree):
            if isinstance(node,ast.Dict):
                pairs=[(k,v) for k,v in zip(node.keys,node.values) if not isinstance(k,ast.Constant) or k.value!='inventory']
                node.keys=[k for k,v in pairs];node.values=[v for k,v in pairs]
        tree.body=[n for n in tree.body if not isinstance(n,ast.ImportFrom) or n.module!='concurrent.futures']
    tree.body=[n for n in tree.body if not isinstance(n,ast.If) or ast.unparse(n.test)!="__name__ == '__main__'"]
    main=f'''
if __name__ == '__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--base-url',required=True)
    parser.add_argument('--through-task',type=int,choices=range(1,{cutoff+1}),default={cutoff})
    parser.add_argument('--condition',choices=({condition!r},),required=True)
'''
    if cutoff>=3:main+="    parser.add_argument('--checkpoint')\n    parser.add_argument('--verify-restart')\n"
    main+="    args=parser.parse_args()\n    checks=Checks(args.base_url)\n    try:\n"
    if cutoff>=3:
        main+="        if args.verify_restart:\n            checks.restart(json.loads(Path(args.verify_restart).read_text(encoding='utf8')))\n        else:\n            checkpoint=checks.run(args.through_task,args.condition)\n            if args.checkpoint:Path(args.checkpoint).write_text(json.dumps(checkpoint,indent=2)+'\\n',encoding='utf8')\n"
    else:main+="        checks.run(args.through_task,args.condition)\n"
    main+="        print(json.dumps({'status':'passed','checks':checks.passed},indent=2))\n    except (AssertionError,OSError,KeyError,ValueError) as exc:\n        print(json.dumps({'status':'failed','passed':checks.passed,'error':str(exc)},indent=2))\n        raise SystemExit(1)\n"
    return ast.unparse(ast.fix_missing_locations(tree))+'\n'+main
def release(cutoff,destination,condition,actor_repository):
    if not pathlib.Path(destination).is_absolute() or not pathlib.Path(actor_repository).is_absolute():raise ValueError('explicit absolute paths required')
    out=pathlib.Path(destination).resolve();repo=pathlib.Path(actor_repository).resolve()
    if out.is_relative_to(repo):raise ValueError('released inputs must stay outside Actor repository')
    source=staged_checks(cutoff,condition)
    out.mkdir(parents=True,exist_ok=False)
    for name in ['brief.md','architecture.md']+(['brownfield-provenance.md'] if condition=='brownfield' else []):shutil.copyfile(ROOT/name,out/name)
    cards=sorted((ROOT/'tasks').glob('0*.md'))
    for p in cards[:cutoff]:shutil.copyfile(p,out/p.name)
    (out/'checks.py').write_text(source,encoding='utf8')
    result={'version':'v4-draft','throughTask':cutoff,'condition':condition,'inputs':[{'path':p.name,'sha256':hashlib.sha256(p.read_bytes()).hexdigest()} for p in sorted(out.iterdir())]}
    (out/'release.json').write_text(json.dumps(result,indent=2)+'\n',encoding='utf8');return result
if __name__=='__main__':
    a=argparse.ArgumentParser();a.add_argument('--through-task',type=int,required=True);a.add_argument('--destination',required=True);a.add_argument('--condition',required=True);a.add_argument('--actor-repository',required=True);x=a.parse_args()
    print(json.dumps(release(x.through_task,x.destination,x.condition,x.actor_repository),indent=2))
