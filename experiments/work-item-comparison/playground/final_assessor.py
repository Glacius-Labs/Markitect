"""Fresh public-spec assessment, with no implementer session or private results."""
from datetime import datetime, timezone
import json
from pathlib import Path
import threading
import shutil
import subprocess
import hashlib


def assess(plan, choice, executable, candidate, audit, expires):
    from conventional.backends import run
    remaining=(expires-datetime.now(timezone.utc)).total_seconds()
    if remaining<=0:
        return {"state":"NOT RUN","reason":"trajectory capture/assessment deadline exhausted"}
    folder=audit/"independent-final-assessment"
    folder.mkdir()
    assessed=folder/"candidate"
    shutil.copytree(candidate,assessed)
    def manifest(root):
        return {str(p.relative_to(root)):hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(root.rglob("*")) if p.is_file() and ".git" not in p.relative_to(root).parts}
    bound=manifest(candidate)
    if manifest(assessed)!=bound:
        raise ValueError("final assessor candidate differs from frozen main")
    def git(*args):
        subprocess.run(["git","-C",str(assessed),*args],check=True,capture_output=True,timeout=60)
    git("init","-b","main")
    git("config","user.name","Playground Assessor Setup")
    git("config","user.email","playground-assessor@example.invalid")
    git("add","--all")
    git("diff","--cached","--check")
    git("commit","-m","Bind frozen public candidate for independent assessment")
    options=dict(choice["config"]["runtimeOptions"])
    options.update(sandbox="read-only",nativeMaxConcurrentAgents=1)
    prompt=("Pruefe diesen eingefrorenen Kandidaten unabhaengig anhand seiner oeffentlichen README.md, BACKLOG.md, "
        "STATIONS.json, QUALITY.md und Projektregeln fuer alle vier freigegebenen Stationen. Du bist ein frischer finaler "
        "Assessor. Implementiere und repariere nichts, veraendere keine Dateien und starte keine Helfer. Nutze nur diesen "
        "Kandidaten und seine oeffentlichen Anforderungen; keine Implementer-Transkripte, private Bewertungen, alte "
        "Versuchsergebnisse, andere Varianten oder Credentials. Fuehre passende bestehende Tests und eigene relevante "
        "Grenz-/Regressionsfaelle aus. Berichte PASS/FAIL/NOT RUN/EVALUATION_ERROR oder unknown mit konkreter Evidenz, "
        "verbleibende Funktions-/Regel-/Persistenz-/Rename-/Dokumentationsprobleme, S3-Teamnachweise und Grenzen. "
        "Ein nativer Erfolg oder Testpass ist keine menschliche Abnahme. Verwende gpt-6-luna high wie gebunden. "
        "Antworte mit einer nachvollziehbaren finalen Bewertung, ohne Reparaturauftrag.")
    spec={"backend":"codex-cli","command":[str(executable),"-c","agents.enabled=false","-c","features.multi_agent=false"],"cwd":str(assessed),"model":choice["config"]["model"],
          "effort":choice["config"]["effort"],"runtimeOptions":options,"timeoutSeconds":min(5400,remaining),
          "requestTimeoutSeconds":min(600,remaining)}
    binding={"schema":1,"startedAt":datetime.now(timezone.utc).isoformat(),"spec":spec,"prompt":prompt,
             "sourceScope":"fresh native session; frozen candidate and public declared requirements only",
             "candidateFilePins":bound,"frozenMainPath":str(candidate),"startRequests":1,"authority":plan["id"],"humanAcceptance":"not established"}
    (folder/"binding.json").write_bytes((json.dumps(binding,indent=2)+"\n").encode())
    (folder/"prompt.txt").write_bytes(prompt.encode())
    def emit(event):
        event={"observedAt":datetime.now(timezone.utc).isoformat(),**event}
        with (folder/"events.jsonl").open("ab") as stream:
            stream.write((json.dumps(event,ensure_ascii=False)+"\n").encode())
            stream.flush()
    print(json.dumps({"event":"fresh-independent-assessor-start","case":choice["case"]}),flush=True)
    try:
        result=run(spec,prompt,None,emit,threading.Event())
    except Exception as exc:
        result={"state":"uncertain","reason":f"{type(exc).__name__}: {exc}","replay":"forbidden"}
    result["candidateFilesUnchanged"]=manifest(assessed)==bound
    result.update(endedAt=datetime.now(timezone.utc).isoformat(),humanAcceptance="not established")
    (folder/"result.json").write_bytes((json.dumps(result,indent=2)+"\n").encode())
    return result
