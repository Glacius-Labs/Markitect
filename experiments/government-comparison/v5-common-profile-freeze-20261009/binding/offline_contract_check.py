"""ONE OFFLINE MECHANICAL CHECK. Synthetic wire values only; never readiness/product evidence."""
import ast, copy, importlib.util, json, pathlib, tempfile, time
ROOT = pathlib.Path(__file__).resolve().parents[1]
ADAPTER = ROOT / "public-source/internal/tooling/codexrunner/runner.py"
def deny(*args, **kwargs):
    raise AssertionError("Forbidden process/provider/CLI operation in offline check")
spec = importlib.util.spec_from_file_location("frozen_codex_adapter", ADAPTER)
adapter = importlib.util.module_from_spec(spec)
spec.loader.exec_module(adapter)
adapter.subprocess.Popen = deny
adapter.subprocess.run = deny
adapter.check_version = deny
adapter.resolve_codex = deny
def closed(properties):
    return {"type":"object","additionalProperties":False,"properties":properties,"required":list(properties)}
text={"type":"string"}
texts={"type":"array","items":text}
delegation=closed({"managerId":text,"goal":text})
rework=closed({"managerId":text,"goal":text,"reason":text})
task_properties={"status":{"type":"string","enum":["complete","partial","blocked","failed","no-op"]},"summary":text,
"delegations":{"type":"array","items":delegation},"reworkRequests":{"type":"array","items":rework},"integrated":{"type":"boolean"},
"questions":texts,"risks":texts,"resolvedQuestions":texts,"resolvedRisks":texts,"escalateTo":text}
review_schema=closed({"status":{"type":"string","enum":["pass","fail"]},"summary":text,
"findings":{"type":"array","items":closed({"path":text,"expectation":text,"grounding":text})}})
def invocation(kind, phase="work"):
    schema=copy.deepcopy(closed(task_properties) if kind=="projectrun-task/v1" else review_schema)
    if kind=="projectrun-task/v1":schema["properties"]["integrated"]["enum"]=[phase=="integrate"]
    return {"apiVersion":"markitect.example.org/agent-execution/v1alpha1","runId":"0"*32,"nonce":"1"*32,
      "inputDigest":"sha256:"+"2"*64,"request":{"role":"executor","sourceRevision":"3"*40,"modelDigest":"sha256:"+"4"*64,
      "modulePin":"synthetic","projectionId":"synthetic","scopeIds":["SYNTHETIC-SCOPE"],"policyIds":[],"artifacts":[],
      "context":{"kind":kind,"phase":phase,"responseSchema":schema}}}
def response(inv, report):
    return {"apiVersion":inv["apiVersion"],"runId":inv["runId"],"nonce":inv["nonce"],"inputDigest":inv["inputDigest"],
      "role":inv["request"]["role"],"outcome":"proposed","candidateFiles":[],"candidateJson":None,
      "reportJson":json.dumps(report),"evidenceRefs":[],"verifierObservations":[],"uncertainty":[]}
def must_reject(fn):
    try:fn()
    except adapter.AdapterError:return
    raise AssertionError("Expected closed DTO rejection")
start=time.monotonic()
observations=[]
for phase in ["work","integrate"]:
    inv=adapter.validate_invocation(invocation("projectrun-task/v1",phase))
    report={"status":"complete","summary":"Synthetic serialization only","delegations":[],"reworkRequests":[],
      "integrated":phase=="integrate","questions":[],"risks":[],"resolvedQuestions":[],"resolvedRisks":[],"escalateTo":""}
    out=adapter.normalize_codex_response(response(inv,report),inv)
    assert out["reportJson"]==report
    invalid=copy.deepcopy(report);invalid["unknownField"]="synthetic"
    must_reject(lambda:adapter.normalize_codex_response(response(inv,invalid),inv))
    observations.append("synthetic Manager "+phase+" closed report normalization/rejection")
inv=adapter.validate_invocation(invocation("projectrun-review/v1"))
review={"status":"pass","summary":"Synthetic serialization only","findings":[]}
out=adapter.normalize_codex_response(response(inv,review),inv)
assert out["reportJson"]==review and out["candidateFiles"]==[]
bad=response(inv,review);bad["candidateFiles"]=[{"path":"synthetic.txt","mode":"0644","content":"synthetic"}]
must_reject(lambda:adapter.normalize_codex_response(bad,inv))
badreport={"status":"fail","summary":"Synthetic","findings":[]}
must_reject(lambda:adapter.normalize_codex_response(response(inv,badreport),inv))
observations.append("synthetic Reviewer executor/context dispatch, no-write and verdict consistency")
vinv=invocation("projectrun-task/v1")
vinv["request"]["role"]="verifier"
vinv["request"]["context"]={"requiredSubjects":["SYNTHETIC-SUBJECT"]}
v=response(vinv,None);v["reportJson"]=None;v["outcome"]="passed"
v["verifierObservations"]=[{"subject":"SYNTHETIC-SUBJECT","outcome":"passed","detail":"Synthetic DTO, no real verification"}]
out=adapter.normalize_codex_response(v,vinv)
assert "reportJson" not in out and out["verifierObservations"][0]["subject"]=="SYNTHETIC-SUBJECT"
observations.append("synthetic true Verifier distinct wire DTO; Host coverage validation NOT executed")
with tempfile.TemporaryDirectory(prefix="scientist-v5-wire-only-") as tmp:
    collector=adapter.EventCollector(pathlib.Path(tmp)/"synthetic-events.jsonl")
    assert collector.telemetry() is None
    collector.record_line(json.dumps({"type":"turn.completed","usage":{"input_tokens":123,"output_tokens":7,"cached_input_tokens":11}}).encode()+b"\n")
    mapped=collector.telemetry()
    assert mapped=={"source":"provider-reported","inputTokens":123,"outputTokens":7,"cachedTokens":11}
    collector.close()
observations.append("synthetic event keys map to input/output/cached tokens; missing usage remains None, no actual telemetry")
assert adapter.model_config_args({"model_reasoning_effort":"high"})==["--config",'model_reasoning_effort="high"']
must_reject(lambda:adapter.model_config_args({"sandbox":"danger-full-access"}))
tree=ast.parse(ADAPTER.read_text(encoding="utf-8"))
launch=next(n for n in tree.body if isinstance(n,ast.FunctionDef) and n.name=="launch_codex")
argv=next(n.value for n in ast.walk(launch) if isinstance(n,ast.Assign) and any(isinstance(t,ast.Name) and t.id=="argv" for t in n.targets))
constants=[n.value for n in argv.elts if isinstance(n,ast.Constant)]
for token in ["--ignore-user-config","--sandbox","read-only","--ignore-rules","--ephemeral","--json","--skip-git-repo-check","--disable","plugins","shell_tool","unified_exec"]:
    assert token in constants
assert "Codex invoked a tool despite its configured tool restrictions." in ADAPTER.read_text()
observations.append("existing security argv and restricted modelOptions preserved, no subprocess executed")
result={"classification":"PASS / offline mechanical mapping only","synthetic":True,"actualProviderUsage":None,
"actualProductOrProviderCalls":0,"hostGoValidatorsExecuted":False,"productReadinessOrEquivalence":False,
"seconds":time.monotonic()-start,"observations":observations}
print(json.dumps(result))
