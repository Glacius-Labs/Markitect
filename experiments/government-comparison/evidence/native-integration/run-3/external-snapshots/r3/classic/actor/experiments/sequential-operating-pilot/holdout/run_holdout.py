import argparse,hashlib,json,os,re,shutil,subprocess,sys,time,xml.etree.ElementTree as ET
from pathlib import Path
EX={"bin","obj",".git"}; SDK="10.0.103"
CS=r"""
using System.Text.Json; using Orders; using Fulfillment;
var stage=args.Single();var errors=new List<string>();int checks=0;
void C(bool b,string m){checks++;if(!b)errors.Add(m);} bool V(int q)=>q>=1&&q<=10;bool S(string? x)=>!string.IsNullOrWhiteSpace(x);
int[] qs={int.MinValue,-1,0,1,2,3,4,5,6,7,8,9,10,11,int.MaxValue};
string?[] ss={null,""," ","\t\r\n","\u00a0","\u2007","\u202f","\u3000"," \u00a0\t","SKU","  SKU\t","Café","x\u200b"};
foreach(int q in qs){var seen=new List<int>();var h=new FulfillOrderHandler(x=>seen.Add(x));bool ok=h.Execute(q);C(ok==V(q),"Fulfillment result "+q);C(seen.Count==(V(q)?1:0),"Fulfillment callback count "+q);if(V(q)&&seen.Count==1)C(seen[0]==q,"Fulfillment payload "+q);}
foreach(var s in ss)foreach(int q in qs){var seen=new List<(string? s,int q)>();var h=new CreateOrderHandler((x,n)=>seen.Add((x,n)));bool ok=h.Execute(s!,q);bool exp=S(s)&&V(q);C(ok==exp,"Orders result "+(s??"null")+" "+q);C(seen.Count==(exp?1:0),"Orders callback count");if(exp&&seen.Count==1){C(seen[0].s==s,"SKU changed");C(seen[0].q==q,"quantity changed");}}
foreach(var s in new string?[]{null,""," \t"," SKU-A\t","SKU-B"})foreach(int q in qs){var ev=new List<string>();var f=new FulfillOrderHandler(n=>ev.Add("d:"+n));var o=new CreateOrderHandler((x,n)=>{ev.Add("o:"+(x??"null")+":"+n);if(!f.Execute(n))throw new InvalidOperationException();});bool ok;try{ok=o.Execute(s!,q);}catch(Exception e){C(false,"composition threw "+e.GetType().Name);continue;}bool exp=S(s)&&V(q);C(ok==exp,"composition result");C(ev.Count==(exp?2:0),"composition count");if(exp&&ev.Count==2){C(ev[0]=="o:"+s+":"+q,"composition payload/order");C(ev[1]=="d:"+q,"dispatch payload");}}
if(stage=="callback-policy")foreach(bool orders in new[]{true,false}){
 foreach(bool sub in new[]{false,true}){
  int n=0;Action cb=()=>{n++;if(n==1){if(sub)throw new Sub();throw new InvalidOperationException();}};
  if(orders){var h=new CreateOrderHandler((_,_)=>cb());C(!h.Execute("SKU",5),"InvalidOperationException must return false");C(n==1,"InvalidOperationException callback count");C(h.Execute("SKU",5)&&n==2,"Orders same handler reusable after InvalidOperationException");}
  else{var h=new FulfillOrderHandler(_=>cb());C(!h.Execute(5),"InvalidOperationException must return false");C(n==1,"InvalidOperationException callback count");C(h.Execute(5)&&n==2,"Fulfillment same handler reusable after InvalidOperationException");}
 }
 var mark=new Marker();int count=0;Action other=()=>{count++;if(count==1)throw mark;};Exception? caught=null;
 if(orders){var h=new CreateOrderHandler((_,_)=>other());try{h.Execute("SKU",5);}catch(Exception e){caught=e;}C(ReferenceEquals(caught,mark),"Orders exception identity");C(count==1,"Orders exception retry");C(h.Execute("SKU",5)&&count==2,"Orders same handler reusable after other exception");int n=0;var r=new CreateOrderHandler((_,_)=>n++);C(!r.Execute(" \t",5)&&n==0,"rejected SKU callback");C(!r.Execute("SKU",0)&&n==0,"rejected quantity callback");}
 else{var h=new FulfillOrderHandler(_=>other());try{h.Execute(5);}catch(Exception e){caught=e;}C(ReferenceEquals(caught,mark),"Fulfillment exception identity");C(count==1,"Fulfillment exception retry");C(h.Execute(5)&&count==2,"Fulfillment same handler reusable after other exception");int n=0;var r=new FulfillOrderHandler(_=>n++);C(!r.Execute(0)&&n==0,"rejected callback");}
}
if(stage=="callback-policy"){int d=0,u=0;var f=new FulfillOrderHandler(_=>{d++;if(d==1)throw new InvalidOperationException();});var o=new CreateOrderHandler((_,q)=>{u++;if(!f.Execute(q))throw new InvalidOperationException();});C(!o.Execute("SKU",5),"composed failure");C(d==1&&u==1,"composed callback counts");C(o.Execute("SKU",5)&&d==2&&u==2,"composed same handler usable after failure");}
Console.WriteLine(JsonSerializer.Serialize(new{passed=errors.Count==0,checks,failedCheckCount=errors.Count,errors}));return errors.Count==0?0:1;
sealed class Sub:InvalidOperationException{} sealed class Marker:Exception{}
"""
def mf(r):
 d={}
 for p in sorted(r.rglob("*")):
  rel=p.relative_to(r)
  if any(x.lower() in EX for x in rel.parts):continue
  if p.is_symlink():raise RuntimeError("symlink: "+str(rel))
  if p.is_file():
   b=p.read_bytes();d[rel.as_posix()]={"sha256":hashlib.sha256(b).hexdigest(),"bytes":len(b)}
 return d
DEADLINE=0
def run(cmd,cwd,env,log):
 t=time.monotonic();p=subprocess.run(cmd,cwd=cwd,env=env,text=True,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,timeout=max(1,DEADLINE-time.monotonic()));sec=round(time.monotonic()-t,3);log.write_text(p.stdout,encoding="utf8");return p,sec
def main():
 global DEADLINE;DEADLINE=time.monotonic()+600
 p=argparse.ArgumentParser();p.add_argument("--fixture",required=True,type=Path);p.add_argument("--stage",required=True,choices=("sku","callback-policy"));p.add_argument("--output",required=True,type=Path);a=p.parse_args()
 src=a.fixture.resolve(strict=True);out=a.output.resolve()
 if out.exists():raise SystemExit("output must be absent")
 roots={n:src/"src"/n for n in ("Orders","Fulfillment")}
 for n,r in roots.items():
  if not (r/(n+".csproj")).is_file():raise SystemExit("missing project "+str(r/(n+".csproj")))
 before={n:mf(r) for n,r in roots.items()};dot=shutil.which("dotnet")
 if not dot:raise SystemExit("dotnet absent")
 v=subprocess.run([dot,"--version"],text=True,stdout=subprocess.PIPE).stdout.strip()
 if v!=SDK:raise SystemExit(f"expected SDK {SDK}, got {v}")
 out.mkdir(parents=True);w=out/"isolated";w.mkdir()
 for n,r in roots.items():
  d=w/n
  for x in sorted(r.rglob("*")):
   rel=x.relative_to(r)
   if any(z.lower() in EX for z in rel.parts):continue
   if x.is_symlink():raise RuntimeError("symlink: "+str(rel))
   y=d/rel
   if x.is_dir():y.mkdir(parents=True,exist_ok=True)
   elif x.is_file():y.parent.mkdir(parents=True,exist_ok=True);shutil.copy2(x,y)
 app=w/"Behavior";app.mkdir();(app/"Program.cs").write_text(CS,encoding="utf8")
 (app/"Behavior.csproj").write_text("""<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><OutputType>Exe</OutputType><TargetFramework>net10.0</TargetFramework><ImplicitUsings>enable</ImplicitUsings><Nullable>enable</Nullable></PropertyGroup><ItemGroup><ProjectReference Include="../Orders/Orders.csproj"/><ProjectReference Include="../Fulfillment/Fulfillment.csproj"/></ItemGroup></Project>""",encoding="utf8")
 cfg=w/"NuGet.Config";cfg.write_text('<?xml version="1.0"?><configuration><packageSources><clear /></packageSources></configuration>',encoding="utf8")
 env=os.environ.copy();env.update(DOTNET_CLI_TELEMETRY_OPTOUT="1",DOTNET_NOLOGO="1",NUGET_PACKAGES=str(w/"packages"))
 mh=hashlib.sha256(json.dumps(before,sort_keys=True,separators=(",",":")).encode()).hexdigest()
 static={}; io_re=re.compile(r"\b(?:System\.IO|File|Directory|FileStream|HttpClient|WebRequest|Socket|TcpClient|UdpClient)\b")
 for n,root in roots.items():
  refs=[];packages=[];io=[];parse_errors=[]
  for proj in sorted(root.rglob("*.csproj")):
   try: xml=ET.parse(proj).getroot()
   except Exception as e: parse_errors.append({"project":proj.relative_to(root).as_posix(),"error":f"{type(e).__name__}: {e}"});continue
   local=lambda x:x.tag.split("}")[-1]
   for node in xml.iter():
    if local(node) in ("PackageReference","PackageDownload"):
     packages.append({"project":proj.relative_to(root).as_posix(),"include":node.attrib.get("Include",node.attrib.get("Update",""))})
    elif local(node)=="ProjectReference":
     raw=node.attrib.get("Include",node.attrib.get("Update","")); condition=node.attrib.get("Condition","")
     item={"project":proj.relative_to(root).as_posix(),"include":raw,"condition":condition}
     if condition or not raw or any(c in raw for c in "$%@*?;"):
      item["status"]="not_established"
     else:
      target=(proj.parent/Path(raw)).resolve()
      try: target.relative_to(root.resolve(strict=True)); internal=True
      except ValueError: internal=False
      item["resolved_path"]=str(target)
      item["status"]="internal" if internal and target.is_file() else ("external" if not internal else "missing")
     refs.append(item)
  for f in sorted(root.rglob("*.cs")):
   if any(z.lower() in EX for z in f.relative_to(root).parts):continue
   try: lines=f.read_text(encoding="utf-8-sig").splitlines()
   except UnicodeDecodeError: continue
   for ln,line in enumerate(lines,1):
    if io_re.search(line):io.append({"file":f.relative_to(root).as_posix(),"line":ln,"text":line.strip()[:200]})
  reject=bool(packages or parse_errors or any(x["status"]!="internal" for x in refs))
  static[n]={"project_references":refs,"package_references":packages,"project_parse_errors":parse_errors,"io_text_mentions_advisory":io,"status":"rejected" if reject else "passed"}
 disallowed=any(x["status"]=="rejected" for x in static.values())
 r={"schema":"markitect-sequence-holdout/v1","status":"running","stage":a.stage,"fixture":str(src),"dotnet_sdk":v,"source_before":before,"source_manifest_sha256":mh,"static_review":static,"static_review_rejected":disallowed,"commands":[],"limitations":{"scope":"Copied .cs files in temporary console; includes same-area helper .cs files. Declared behavioral observations only.","static":"External or unresolved ProjectReferences and any project PackageReference/PackageDownload reject the run; same-area project references are allowed when resolved under that copied area. IO API text matches are advisory only. XML scans do not establish imported/conditional/transitive build behavior or absence of hidden IO.","callback_effects":"No rollback claim for callback-owned effects."}}
 cmd=[dot,"restore",str(app/"Behavior.csproj"),"--configfile",str(cfg),"--ignore-failed-sources"];r["commands"].append({"argv":cmd,"cwd":str(w)});z,t=run(cmd,w,env,out/"restore.log");r["restore"]={"exit_code":z.returncode,"seconds":t}
 if z.returncode==0:
  cmd=[dot,"build",str(app/"Behavior.csproj"),"--no-restore","--nologo"];r["commands"].append({"argv":cmd,"cwd":str(w)});z,t=run(cmd,w,env,out/"build.log");r["build"]={"exit_code":z.returncode,"seconds":t}
 if r.get("build",{}).get("exit_code")==0:
  cmd=[dot,str(app/"bin/Debug/net10.0/Behavior.dll"),a.stage];r["commands"].append({"argv":cmd,"cwd":str(w)});z,t=run(cmd,w,env,out/"behavior.log");r["behavior"]={"exit_code":z.returncode,"seconds":t,"stdout":z.stdout.strip()}
  try:r["behavior"]["result"]=json.loads(z.stdout.strip().splitlines()[-1])
  except Exception:r["behavior"]["result"]=None
 after={n:mf(x) for n,x in roots.items()};r["source_after"]=after;r["source_unchanged"]=before==after
 r["status"]="passed" if not disallowed and r.get("build",{}).get("exit_code")==0 and r.get("behavior",{}).get("exit_code")==0 and r.get("behavior",{}).get("result",{}).get("passed") and r["source_unchanged"] else "failed"
 r["finish_utc"]=time.strftime("%Y-%m-%dT%H:%M:%SZ",time.gmtime());(out/"result.json").write_text(json.dumps(r,indent=2),encoding="utf8")
 keys=("schema","status","stage","fixture","dotnet_sdk","source_manifest_sha256","static_review","static_review_rejected","commands","restore","build","behavior","source_unchanged","limitations");(out/"summary.yaml").write_text(json.dumps({k:r.get(k) for k in keys},indent=2),encoding="utf8")
 print(json.dumps({"status":r["status"],"result":str(out/"result.json"),"summary":str(out/"summary.yaml")}));return 0 if r["status"]=="passed" else 1
if __name__=="__main__":
 try:raise SystemExit(main())
 except Exception as e:print(json.dumps({"schema":"markitect-sequence-holdout/v1","status":"failed","error":f"{type(e).__name__}: {e}"}),file=sys.stderr);raise
