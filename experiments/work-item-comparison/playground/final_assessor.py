"""Fresh public-spec assessment, with no implementer session or private results."""
from datetime import datetime, timezone
import json
from pathlib import Path
import threading
import shutil
import subprocess
import hashlib


def assess(plan, choice, executable, candidate, audit, expires, *, completion_target="main_merge"):
    from conventional.backends import run
    remaining=(expires-datetime.now(timezone.utc)).total_seconds()
    if remaining<=0:
        return {"state":"NOT RUN","reason":"trajectory capture/assessment deadline exhausted"}
    folder=audit/"independent-final-assessment"
    folder.mkdir()
    assessed=folder/"candidate"
    shutil.copytree(candidate,assessed)
    supplement_name = ".playground-assessment-public"
    def manifest(root, *, exclude_supplement=False):
        result = {}
        for path in sorted(root.rglob("*")):
            relative = path.relative_to(root)
            if (not path.is_file() or ".git" in relative.parts or
                    (exclude_supplement and relative.parts and relative.parts[0] == supplement_name)):
                continue
            result[relative.as_posix()] = hashlib.sha256(path.read_bytes()).hexdigest()
        return result
    bound=manifest(candidate)
    if manifest(assessed)!=bound:
        raise ValueError("final assessor candidate differs from its frozen assessment candidate")
    workspace_mode = completion_target == "workspace_snapshot"
    if completion_target not in {"main_merge", "workspace_snapshot"}:
        raise ValueError("unsupported completion target")
    if workspace_mode and (not bound or choice.get("config", {}).get("backend") != "codex-app-server"):
        raise ValueError("workspace final assessment requires a nonempty candidate and the declared AppServer backend")
    if workspace_mode:
        capture_record_path = candidate.parent / "snapshot.json"
        capture_record = json.loads(capture_record_path.read_text(encoding="utf-8"))
        candidate_binding = capture_record.get("assessmentCandidate")
        if (capture_record.get("completionTarget") != completion_target or
                not isinstance(candidate_binding, dict) or
                candidate_binding.get("manifest") != bound or
                (candidate.parent / candidate_binding.get("path", "")).resolve() != candidate.resolve()):
            raise ValueError("workspace final assessment candidate differs from its frozen snapshot binding")
    assessment_public = None
    assessment_public_manifest = None
    if workspace_mode:
        if (assessed / supplement_name).exists():
            raise ValueError("candidate collides with reserved assessment requirements directory")
        initial_public = audit / "initial-public"
        if not initial_public.is_dir() or initial_public.is_symlink():
            raise ValueError("workspace assessment requires the pre-actor initial public workspace snapshot")
        initial_binding_path = audit / "initial-public-binding.json"
        initial_binding = json.loads(initial_binding_path.read_text(encoding="utf-8"))
        initial_manifest = manifest(initial_public)
        manifest_hash = hashlib.sha256(
            (json.dumps(initial_manifest, sort_keys=True, separators=(",", ":")) + "\n").encode("utf-8")).hexdigest()
        if (initial_binding.get("kind") != "prepared_public_workspace" or
                initial_binding.get("fileManifest") != initial_manifest or
                initial_binding.get("fileManifestSha256") != manifest_hash):
            raise ValueError("pre-actor initial public workspace differs from its binding")
        assessment_public = assessed / supplement_name
        assessment_public.mkdir()
        for source in sorted(initial_public.rglob("*")):
            relative = source.relative_to(initial_public)
            if ".git" in relative.parts:
                continue
            if source.is_symlink():
                raise ValueError("frozen public requirements contain a symlink")
            destination = assessment_public / relative
            if source.is_dir():
                destination.mkdir(parents=True, exist_ok=True)
            elif source.is_file():
                destination.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(source, destination)
        assessment_public_manifest = manifest(assessment_public)
        if (not assessment_public_manifest or
                initial_manifest != assessment_public_manifest):
            raise ValueError("frozen initial public requirements snapshot is empty")
    if not workspace_mode:
        def git(*args):
            subprocess.run(["git","-C",str(assessed),*args],check=True,capture_output=True,timeout=60)
        git("init","-b","main")
        git("config","user.name","Playground Assessor Setup")
        git("config","user.email","playground-assessor@example.invalid")
        git("add","--all")
        git("diff","--cached","--check")
        git("commit","-m","Bind frozen public candidate for independent assessment")
    options=dict(choice["config"]["runtimeOptions"])
    options.update(sandbox="read-only",approvalPolicy="never",nativeMaxConcurrentAgents=1,
                   scopedGitApproval=False,allowLoginShell=False)
    options.pop("gitApprovalShell", None)
    prompt=("Pruefe diesen eingefrorenen Kandidaten unabhaengig anhand der separaten unveraenderlichen "
        "Anfangsanforderungen im Ordner .playground-assessment-public sowie des Kandidaten und seiner aktuellen "
        "oeffentlichen README.md, BACKLOG.md, "
        "STATIONS.json, QUALITY.md und Projektregeln fuer alle vier freigegebenen Stationen. Du bist ein frischer finaler "
        "Assessor. Implementiere und repariere nichts, veraendere keine Dateien und starte keine Helfer. Nutze nur diesen "
        "Anfangsanforderungen als verbindliche Basis und die im Kandidaten aktuelle Stationserklaerung; keine "
        "Implementer-Transkripte, private Bewertungen, alte "
        "Versuchsergebnisse, andere Varianten oder Credentials. Fuehre passende bestehende Tests und eigene relevante "
        "Grenz-/Regressionsfaelle aus. Berichte PASS/FAIL/NOT RUN/EVALUATION_ERROR oder unknown mit konkreter Evidenz, "
        "verbleibende Funktions-/Regel-/Persistenz-/Rename-/Dokumentationsprobleme, S3-Teamnachweise und Grenzen. "
        "Ein nativer Erfolg oder Testpass ist keine menschliche Abnahme. Verwende gpt-6-luna high wie gebunden. "
        "Antworte mit einer nachvollziehbaren finalen Bewertung, ohne Reparaturauftrag.")
    backend = "codex-app-server" if workspace_mode else "codex-cli"
    command = [str(executable), "-c", "agents.enabled=false", "-c", "features.multi_agent=false"]
    final_budget = plan.get("freshFinalSeconds", 5400) if workspace_mode else 5400
    spec={"backend":backend,"command":command,"cwd":str(assessed),"model":choice["config"]["model"],
          "effort":choice["config"]["effort"],"runtimeOptions":options,"timeoutSeconds":min(final_budget,remaining),
          "requestTimeoutSeconds":min(600,remaining)}
    binding={"schema":1,"startedAt":datetime.now(timezone.utc).isoformat(),"spec":spec,"prompt":prompt,
             "sourceScope":"fresh native session; frozen candidate and public declared requirements only",
             "completionTarget":completion_target,
             "assessmentCandidate":{"path":str(candidate),"manifest":bound},
             "frozenInitialPublicRequirements":{"path":str(assessment_public) if assessment_public else None,
                 "manifest":assessment_public_manifest,
                 "sourceBindingPath":str(initial_binding_path) if workspace_mode else None,
                 "sourceBindingSha256":hashlib.sha256(initial_binding_path.read_bytes()).hexdigest() if workspace_mode else None},
             "candidateFilePins":bound,"frozenCandidatePath":str(candidate),"startRequests":1,"authority":plan["id"],"humanAcceptance":"not established"}
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
    result["candidateFilesUnchanged"]=manifest(assessed,exclude_supplement=workspace_mode)==bound
    if workspace_mode:
        result["frozenInitialPublicRequirementsUnchanged"]=manifest(assessment_public)==assessment_public_manifest
        result["initialPublicWorkspaceBindingUnchanged"]=(
            hashlib.sha256(initial_binding_path.read_bytes()).hexdigest() == binding["frozenInitialPublicRequirements"]["sourceBindingSha256"] and
            manifest(initial_public) == initial_manifest)
        if (result["candidateFilesUnchanged"] is not True or
                result["frozenInitialPublicRequirementsUnchanged"] is not True or
                result["initialPublicWorkspaceBindingUnchanged"] is not True):
            result["nativeState"] = result.get("state")
            result.update(state="failed", reason="candidate or frozen initial public requirements changed during assessment")
    result.update(endedAt=datetime.now(timezone.utc).isoformat(),humanAcceptance="not established")
    (folder/"result.json").write_bytes((json.dumps(result,indent=2)+"\n").encode())
    return result
