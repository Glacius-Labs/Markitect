"""One additional authorized batch. Existing fixtures unchanged; no product/native model process."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import time
from datetime import datetime, timezone

HERE = Path(__file__).resolve().parent
PACKET = HERE.parents[1]
REPO = PACKET.parents[2]
BASE = "429c52d8a37fa60ca4393a3501a495128b4d2e88"
GRANT_SOURCE = Path("C:/Users/Consiliari/Glacius Labs/Markitect/docs/design/government/luna-nest-newline-mechanical-validation-grant-20261009.json")

def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def write(path, value):
    path.write_text(json.dumps(value, indent=2, ensure_ascii=False) + "\n", encoding="utf-8", newline="\n")

def git(*args):
    return subprocess.check_output(["git", "-C", str(REPO), *args], timeout=30).strip()

if (HERE / "reservation.json").exists():
    raise SystemExit("One reservation already exists: no retry")
grant = json.loads(GRANT_SOURCE.read_text(encoding="utf-8-sig"))
now = datetime.now(timezone.utc)
deadline = datetime.fromisoformat(grant["absoluteNotAfterUtc"].replace("Z", "+00:00"))
if git("rev-parse", "HEAD").decode() != BASE or grant["baseSourceSha"] != BASE:
    raise SystemExit("Wrong immutable base")
if (deadline - now).total_seconds() < 210:
    raise SystemExit("Full test/cleanup margin cannot fit the grant")
shutil.copyfile(GRANT_SOURCE, HERE / "grant.json")
original = [p for p in sorted(PACKET.rglob("*")) if p.is_file() and HERE not in p.parents]
before = {str(p.relative_to(PACKET).as_posix()): digest(p) for p in original}
existing_checkout_line_endings = []
for relative, expected in before.items():
    # Every original tracked blob must still match its source checkpoint byte-for-byte.
    path_in_repo = (PACKET / relative).relative_to(REPO).as_posix()
    tracked = git("ls-files", "--error-unmatch", path_in_repo)
    if not tracked:
        raise SystemExit("Unexpected prior untracked file")
    object_bytes = subprocess.check_output(["git", "-C", str(REPO), "show", BASE + ":" + path_in_repo], timeout=30)
    if hashlib.sha256(object_bytes).hexdigest() != expected:
        local_bytes = (PACKET / relative).read_bytes()
        if not relative.startswith(("public/", "tests/")) and local_bytes.replace(b"\r\n", b"\n") == object_bytes:
            existing_checkout_line_endings.append(relative)
        else:
            raise SystemExit("Original working/raw-Git binding changed: " + relative)
write(HERE / "reservation.json", {"grantId": grant["grantId"], "grantSha256": digest(HERE / "grant.json"),
      "reservedUtc": now.isoformat(), "baseSourceSha": BASE, "additionalBatchesReserved": 1,
      "maxTestWallSeconds": 180, "originalFileHashes": before,
      "existingCheckoutLineEndingDifference": existing_checkout_line_endings,
      "caseActors": 0, "providerProbes": 0})
write(HERE / "continuation-anchor.json", {"status": "one additional batch reserved before launch",
      "oldPreparationBatches": 2, "oldPreparationFailedBatches": 2, "oldReviewers": 1,
      "newReviewerStarts": 0, "newMechanicalBatches": 1, "actualFullStarts": 0,
      "noRepeat": True, "deadlineUtc": grant["absoluteNotAfterUtc"]})
ast = """
param([string]$Launcher)
$tokensForValidation = $null
$errorsForValidation = $null
[System.Management.Automation.Language.Parser]::ParseFile($Launcher,[ref]$tokensForValidation,[ref]$errorsForValidation) | Out-Null
[ordered]@{errors=@($errorsForValidation).Count;purpose='AST only; launcher not executed'} | ConvertTo-Json -Compress
if (@($errorsForValidation).Count -ne 0) { exit 1 }
"""
(HERE / "ast-only.ps1").write_text(ast, encoding="utf-8", newline="\n")
commands = [
    ("existing-eight-fixtures", [sys.executable, "-B", "-m", "unittest", "discover", "-s", str(PACKET / "tests"), "-v"]),
    ("existing-brownfield-baseline", [sys.executable, "-B", "-m", "unittest", "discover", "-s", str(PACKET / "public/cases/readinglog/tests"), "-v"]),
    ("existing-oldschool-ast", [shutil.which("pwsh"), "-NoProfile", "-File", str(HERE / "ast-only.ps1"),
                               "-Launcher", str(PACKET / "public/conventional/oldschool/Invoke-NestTurn.ps1")])
]
started = time.monotonic()
end = started + 180
receipts = []
for name, argv in commands:
    if time.monotonic() >= end - 30:
        receipts.append({"name": name, "status": "NOT RUN", "reason": "Aggregate active/cleanup margin exhausted"})
        break
    record = {"name": name, "argv": argv, "startedUtc": datetime.now(timezone.utc).isoformat()}
    with (HERE / (name + ".stdout.txt")).open("wb") as out, (HERE / (name + ".stderr.txt")).open("wb") as err:
        proc = subprocess.Popen(argv, cwd=REPO, stdout=out, stderr=err,
                                creationflags=subprocess.CREATE_NEW_PROCESS_GROUP if os.name == "nt" else 0,
                                start_new_session=os.name != "nt")
        record["pid"] = proc.pid
        try:
            proc.wait(timeout=max(1, end - time.monotonic() - 30))
            record.update(exitCode=proc.returncode, status="PASS" if proc.returncode == 0 else "FAIL")
        except subprocess.TimeoutExpired:
            record["status"] = "TIME LIMIT"
            if os.name == "nt":
                try:
                    stop = subprocess.run(["taskkill", "/PID", str(proc.pid), "/T", "/F"], capture_output=True, timeout=15)
                    record["ownedStopExitCode"] = stop.returncode
                    proc.wait(timeout=min(10, max(1, end - time.monotonic())))
                    record["localExitConfirmed"] = True
                except (subprocess.TimeoutExpired, OSError):
                    record["localExitConfirmed"] = False
            else:
                os.killpg(proc.pid, 9)
                proc.wait(timeout=10)
                record["localExitConfirmed"] = True
    record["endedUtc"] = datetime.now(timezone.utc).isoformat()
    record["stdoutSha256"] = digest(HERE / (name + ".stdout.txt"))
    record["stderrSha256"] = digest(HERE / (name + ".stderr.txt"))
    receipts.append(record)
    if record["status"] != "PASS":
        break  # terminal, no retry or further test charge
after = {relative: digest(PACKET / relative) for relative in before}
unchanged = before == after
passed = len(receipts) == 3 and all(r["status"] == "PASS" for r in receipts) and unchanged
result = {"grantId": grant["grantId"], "baseSourceSha": BASE, "additionalBatch": 1,
          "status": "PASS" if passed else "FAIL", "aggregateTestWallSeconds": time.monotonic() - started,
          "originalFilesUnchanged": unchanged, "receipts": receipts, "caseActors": 0,
          "productActors": 0, "providerProbes": 0, "productBuilds": 0, "actualFullStarts": 0,
          "newReviewerStarts": 0, "cost": None, "tokens": None,
          "scope": "Mechanical fixtures + original baseline + PS AST; no native case execution or acceptance"}
write(HERE / "receipt.json", result)
write(HERE / "continuation-anchor.json", {"status": result["status"] + "; additive checkpoint/handoff pending",
      "oldPreparationBatches": 2, "oldPreparationFailedBatches": 2, "oldReviewers": 1,
      "newReviewerStarts": 0, "newMechanicalBatches": 1, "actualFullStarts": 0,
      "noRepeat": True, "deadlineUtc": grant["absoluteNotAfterUtc"], "next": "Commit/push exact additive evidence and callback once"})
print(json.dumps(result, indent=2))
sys.exit(0 if passed else 1)
