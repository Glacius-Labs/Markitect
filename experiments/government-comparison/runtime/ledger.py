"""Transactional, no-refill shared trial accounting; unknown usage stays unknown."""
from contextlib import contextmanager
import json
from pathlib import Path
import sqlite3
import time
import uuid
import hashlib
from measurement_profile import LEGACY, OBSERVED, profile, profile_sha, limits_sha


class LimitReached(ValueError):
    pass


class Ledger:
    def __init__(self, path, trial_id, limits, *, profile_id=LEGACY):
        self.path = str(Path(path).resolve())
        self.profile_id = profile_id
        self.authority_key = None
        Path(path).parent.mkdir(parents=True, exist_ok=True)
        with self.transaction() as db:
            schema = """CREATE TABLE IF NOT EXISTS trial(id TEXT PRIMARY KEY, limits TEXT, start REAL, stopped INTEGER);
                CREATE TABLE IF NOT EXISTS tasks(id TEXT PRIMARY KEY, start REAL);
                CREATE TABLE IF NOT EXISTS attempts(id TEXT PRIMARY KEY, task TEXT, purpose TEXT, start REAL,
                    end REAL, status TEXT, turns INTEGER, tokens INTEGER, receipt TEXT);
                CREATE TABLE IF NOT EXISTS controller_runs(
                    dispatch_id TEXT PRIMARY KEY, task TEXT NOT NULL, start REAL NOT NULL,
                    end REAL, status TEXT, process_receipt TEXT, receipt_sha256 TEXT);
                CREATE TABLE IF NOT EXISTS controller_processes(
                    dispatch_id TEXT NOT NULL, sequence INTEGER NOT NULL, action TEXT NOT NULL,
                    start REAL NOT NULL, end REAL NOT NULL, status TEXT NOT NULL,
                    process_receipt TEXT NOT NULL, receipt_sha256 TEXT NOT NULL,
                    PRIMARY KEY(dispatch_id,sequence), UNIQUE(dispatch_id,action));
                CREATE TABLE IF NOT EXISTS human(seconds REAL, note TEXT);
                CREATE TABLE IF NOT EXISTS measurement_profiles(sequence INTEGER PRIMARY KEY,
                    profile_id TEXT, profile_sha TEXT, profile_json TEXT, prior_sha TEXT, approval TEXT, adopted REAL);"""
            for statement in schema.split(";"):
                if statement.strip():
                    db.execute(statement)
            prior = db.execute("SELECT id,limits FROM trial").fetchone()
            encoded = json.dumps(limits, sort_keys=True)
            if prior and prior != (trial_id, encoded):
                raise ValueError("ledger identity/profile cannot change or refill")
            if not prior:
                db.execute("INSERT INTO trial VALUES(?,?,?,0)", (trial_id, encoded, time.time()))
            if not db.execute("SELECT 1 FROM measurement_profiles").fetchone():
                db.execute("INSERT INTO measurement_profiles VALUES(1,?,?,?,?,?,?)",
                           (LEGACY, profile_sha(LEGACY), json.dumps(profile(LEGACY), sort_keys=True), None,
                            json.dumps({"kind": "original-strict-profile-preserved"}), time.time()))
            self._profile(db)
        self.limits = dict(limits)

    def _profile(self, db):
        row = db.execute("SELECT profile_id,profile_sha,profile_json FROM measurement_profiles ORDER BY sequence DESC LIMIT 1").fetchone()
        if row[1] != profile_sha(row[0]) or json.loads(row[2]) != profile(row[0]):
            raise ValueError("stored measurement profile differs from pinned definition")
        if self.profile_id is not None and row[0] != self.profile_id:
            raise ValueError("explicit measurement profile binding mismatch; adoption required")
        return json.loads(row[2])

    def adopt_profile(self, target_id, approval):
        """Explicit append-only v1→v2 adoption. No row, counter, stop or start reset."""
        with self.transaction() as db:
            self._adopt_profile(db, target_id, approval)
        self.profile_id = target_id

    def _adopt_profile(self, db, target_id, approval):
        old = self._profile(db)
        trial_id = db.execute("SELECT id FROM trial").fetchone()[0]
        expected = {"status": "approved", "trialId": trial_id, "ledgerPath": self.path,
                    "fromProfileSha256": profile_sha(LEGACY), "toProfileSha256": profile_sha(OBSERVED),
                    "limitsSha256": limits_sha(self.limits)}
        if old["profileId"] != LEGACY or target_id != OBSERVED:
            raise ValueError("only explicit successive v1 to v2 adoption is supported")
        if not isinstance(approval, dict) or any(approval.get(key) != value for key, value in expected.items()) or not approval.get("decisionRef"):
            raise ValueError("profile adoption approval identity/digests mismatch")
        if db.execute("SELECT 1 FROM attempts WHERE end IS NULL").fetchone():
            raise ValueError("active/ambiguous attempts prevent profile adoption")
        recorded_approval = {**approval, "approvalSha256": limits_sha(approval),
                             "migrationSourceSha256": hashlib.sha256(Path(__file__).read_bytes()).hexdigest()}
        db.execute("INSERT INTO measurement_profiles VALUES(2,?,?,?,?,?,?)",
                   (OBSERVED, profile_sha(OBSERVED), json.dumps(profile(OBSERVED), sort_keys=True),
                    profile_sha(LEGACY), json.dumps(recorded_approval, sort_keys=True), time.time()))

    def measurement(self):
        with self.transaction() as db:
            return self._profile(db)

    @contextmanager
    def transaction(self):
        db = sqlite3.connect(self.path, timeout=10, isolation_level=None)
        try:
            db.execute("BEGIN IMMEDIATE")
            yield db
            db.commit()
        except BaseException:
            db.rollback()
            raise
        finally:
            db.close()

    def reserve(self, task, purpose):
        """Reserve a session before launch; failed launches also consume it.

        Usage admission follows the bound measurement profile. Neither profile
        enforces an unseen provider request or an in-flight token cap.
        """
        with self.transaction() as db:
            return self._reserve(db, task, purpose)

    def _reserve(self, db, task, purpose):
        if self.profile_id is None:
            raise ValueError("recovery-only ledger cannot reserve a new session")
        measurement = self._profile(db)
        now, limits = time.time(), self.limits
        trial = db.execute("SELECT start,stopped FROM trial").fetchone()
        if trial[1] or now - trial[0] >= limits["trialWallSeconds"]:
            raise LimitReached("trial stopped/deadline")
        db.execute("INSERT OR IGNORE INTO tasks VALUES(?,?)", (task, now))
        task_start = db.execute("SELECT start FROM tasks WHERE id=?", (task,)).fetchone()[0]
        if now - task_start >= limits["taskWallSeconds"]:
            raise LimitReached("task deadline")
        all_rows = db.execute("SELECT task,end,turns,tokens FROM attempts").fetchall()
        task_rows = [row for row in all_rows if row[0] == task]
        if sum(row[1] is None for row in all_rows) >= limits["maxParallelActors"]:
            raise LimitReached("parallel session limit")
        if purpose == "repair" and "maxSemanticRepairRoundsPerTask" in limits and sum(1 for row in db.execute(
                "SELECT id FROM attempts WHERE task=? AND purpose='repair'", (task,))) >= limits["maxSemanticRepairRoundsPerTask"]:
            raise LimitReached("semantic repair limit")
        for rows, prefix in ((all_rows, "trial"), (task_rows, "task")):
            if len(rows) >= limits[prefix + "ActorCalls"]:
                raise LimitReached(prefix + " actor-session limit")
            if any(row[1] is not None and row[3] is None for row in rows):
                raise LimitReached("unknown provider usage: tokens unknown; next actor blocked")
            if measurement["unknownProviderRequestsAdmission"] == "block" and any(row[1] is not None and row[2] is None for row in rows):
                raise LimitReached("unknown provider usage: next actor blocked")
            for index, metric in ((2, "ProviderTurns"), (3, "ProviderTokens")):
                if index == 2 and not measurement["knownProviderRequestAdmissionCaps"]:
                    continue
                if sum(row[index] or 0 for row in rows) >= limits[prefix + metric]:
                    raise LimitReached(prefix + metric)
        attempt = uuid.uuid4().hex
        db.execute("INSERT INTO attempts(id,task,purpose,start) VALUES(?,?,?,?)", (attempt, task, purpose, now))
        return attempt

    def bind_dispatch(self, authority, *, maximum=None, transition=None, adoption=None):
        """Preserve the original row; append explicitly allocated successor grants."""
        with self.transaction() as db:
            measurement = self._profile(db)
            if adoption is not None:
                self._adopt_profile(db, OBSERVED, adoption)
                measurement = profile(OBSERVED)
            db.execute("CREATE TABLE IF NOT EXISTS dispatch_authority(value TEXT)")
            db.execute("""CREATE TABLE IF NOT EXISTS dispatch_authority_history(sequence INTEGER PRIMARY KEY,
                       authority_sha TEXT UNIQUE, value TEXT, profile_sha TEXT, ceiling INTEGER)""")
            db.execute("""CREATE TABLE IF NOT EXISTS dispatches(id TEXT PRIMARY KEY, execution_sha TEXT,
                       attempt TEXT UNIQUE, phase TEXT, request BLOB, command TEXT, result TEXT, measurement_profile TEXT, authority_sha TEXT)""")
            db.execute("""CREATE TABLE IF NOT EXISTS controller_runs(
                       dispatch_id TEXT PRIMARY KEY, task TEXT NOT NULL, start REAL NOT NULL,
                       end REAL, status TEXT, process_receipt TEXT, receipt_sha256 TEXT)""")
            db.execute("""CREATE TABLE IF NOT EXISTS controller_processes(
                       dispatch_id TEXT NOT NULL, sequence INTEGER NOT NULL, action TEXT NOT NULL,
                       start REAL NOT NULL, end REAL NOT NULL, status TEXT NOT NULL,
                       process_receipt TEXT NOT NULL, receipt_sha256 TEXT NOT NULL,
                       PRIMARY KEY(dispatch_id,sequence), UNIQUE(dispatch_id,action))""")
            if "measurement_profile" not in {row[1] for row in db.execute("PRAGMA table_info(dispatches)")}:
                db.execute("ALTER TABLE dispatches ADD COLUMN measurement_profile TEXT")
            if "authority_sha" not in {row[1] for row in db.execute("PRAGMA table_info(dispatches)")}:
                db.execute("ALTER TABLE dispatches ADD COLUMN authority_sha TEXT")
            value = json.dumps(authority, sort_keys=True)
            key = limits_sha(authority)
            prior = db.execute("SELECT value FROM dispatch_authority").fetchone()
            if not prior:
                db.execute("INSERT INTO dispatch_authority VALUES(?)", (value,))
                db.execute("INSERT INTO dispatch_authority_history VALUES(1,?,?,?,?)",
                           (key, value, profile_sha(measurement["profileId"]), maximum))
            elif not db.execute("SELECT 1 FROM dispatch_authority_history").fetchone():
                original = json.loads(prior[0])
                db.execute("INSERT INTO dispatch_authority_history VALUES(1,?,?,?,?)",
                           (limits_sha(original), prior[0], profile_sha(LEGACY), maximum if prior[0] == value else None))
            known = db.execute("SELECT ceiling FROM dispatch_authority_history WHERE authority_sha=?", (key,)).fetchone()
            if known:
                if known[0] is not None and maximum is not None and known[0] != maximum:
                    raise ValueError("grant ceiling cannot change or refill")
            else:
                latest = db.execute("SELECT sequence,authority_sha FROM dispatch_authority_history ORDER BY sequence DESC LIMIT 1").fetchone()
                count = db.execute("SELECT COUNT(*) FROM attempts").fetchone()[0]
                valid = (measurement["profileId"] == OBSERVED and isinstance(transition, dict)
                         and transition.get("priorAuthoritySha256") == latest[1]
                         and type(transition.get("priorActorSessions")) is int and transition["priorActorSessions"] == count
                         and type(transition.get("additionalActorSessions")) is int and transition["additionalActorSessions"] > 0
                         and type(maximum) is int and maximum == count + transition["additionalActorSessions"]
                         and maximum <= self.limits["trialActorCalls"])
                if not valid:
                    raise ValueError("dispatch authority cannot change or refill without explicit cumulative successor allocation")
                db.execute("INSERT INTO dispatch_authority_history VALUES(?,?,?,?,?)",
                           (latest[0] + 1, key, value, profile_sha(OBSERVED), maximum))
        if adoption is not None:
            self.profile_id = OBSERVED
        self.authority_key = key

    def dispatch_record(self, execution_id):
        with self.transaction() as db:
            db.row_factory = sqlite3.Row
            row = db.execute("SELECT * FROM dispatches WHERE id=?", (execution_id,)).fetchone()
            return dict(row) if row else None

    def reserve_dispatch(self, execution_id, execution_sha, raw_request, command, task, purpose, maximum):
        with self.transaction() as db:
            if db.execute("SELECT id FROM dispatches WHERE id=?", (execution_id,)).fetchone():
                raise LimitReached("dispatch already booked; use resume")
            bound = db.execute("SELECT profile_sha,ceiling FROM dispatch_authority_history WHERE authority_sha=?",
                               (self.authority_key,)).fetchone()
            if not bound or bound[0] != profile_sha(self._profile(db)["profileId"]):
                raise ValueError("new dispatch requires current profile-bound authority")
            if bound[1] is not None and maximum != bound[1]:
                raise ValueError("reservation cannot change bound grant ceiling")
            if db.execute("SELECT COUNT(*) FROM attempts").fetchone()[0] >= maximum:
                raise LimitReached("grant session limit")
            attempt = self._reserve(db, task, purpose)
            measurement = self._profile(db)
            db.execute("INSERT INTO dispatches VALUES(?,?,?,'reserved',?,?,NULL,?,?)",
                       (execution_id, execution_sha, attempt, raw_request, json.dumps(command), json.dumps(measurement, sort_keys=True), self.authority_key))
            return attempt

    def reserve_controller_dispatch(self, execution_id, execution_sha, raw_request, command,
                                    task, purpose, maximum):
        """Book the native orchestrator without pretending it is an Actor session.

        Actual configured role invocations reserve Actor attempts separately through
        the common Ledger before each delegate effect. The controller row supplies
        only an elapsed-time anchor and immutable native-process receipt.
        """
        with self.transaction() as db:
            if db.execute("SELECT id FROM dispatches WHERE id=?", (execution_id,)).fetchone():
                raise LimitReached("dispatch already booked; use resume")
            bound = db.execute("SELECT profile_sha,ceiling FROM dispatch_authority_history WHERE authority_sha=?",
                               (self.authority_key,)).fetchone()
            if not bound or bound[0] != profile_sha(self._profile(db)["profileId"]):
                raise ValueError("new dispatch requires current profile-bound authority")
            if bound[1] is not None and maximum != bound[1]:
                raise ValueError("reservation cannot change bound grant ceiling")
            now = time.time()
            trial = db.execute("SELECT start,stopped FROM trial").fetchone()
            if trial[1] or now - trial[0] >= self.limits["trialWallSeconds"]:
                raise LimitReached("trial stopped/deadline")
            db.execute("INSERT OR IGNORE INTO tasks VALUES(?,?)", (task, now))
            task_start = db.execute("SELECT start FROM tasks WHERE id=?", (task,)).fetchone()[0]
            if now - task_start >= self.limits["taskWallSeconds"]:
                raise LimitReached("task deadline")
            measurement = self._profile(db)
            db.execute("INSERT INTO dispatches VALUES(?,?,NULL,'reserved',?,?,NULL,?,?)",
                       (execution_id, execution_sha, raw_request, json.dumps(command),
                        json.dumps(measurement, sort_keys=True), self.authority_key))
            db.execute("INSERT INTO controller_runs(dispatch_id,task,start,status) VALUES(?,?,?,'reserved')",
                       (execution_id, task, now))
            return now

    def controller_elapsed(self, execution_id):
        with self.transaction() as db:
            row = db.execute("SELECT start FROM controller_runs WHERE dispatch_id=?", (execution_id,)).fetchone()
            if not row:
                raise ValueError("controller dispatch timing record is absent")
            return max(0.0, time.time() - row[0])

    def record_controller_process(self, execution_id, action, started_at, process_receipt):
        """Append one immutable native-product process receipt to a controller run."""
        if not isinstance(action, str) or not action or type(started_at) not in (int, float):
            raise ValueError("controller action and finite start time are required")
        if not isinstance(process_receipt, dict):
            raise ValueError("controller process receipt object required")
        raw = json.dumps(process_receipt, sort_keys=True)
        receipt_sha = hashlib.sha256(raw.encode()).hexdigest()
        ended = time.time()
        if ended < started_at:
            raise ValueError("controller process time moved backwards")
        with self.transaction() as db:
            controller = db.execute("SELECT end,status FROM controller_runs WHERE dispatch_id=?",
                                    (execution_id,)).fetchone()
            dispatch = db.execute("SELECT phase,attempt FROM dispatches WHERE id=?",
                                  (execution_id,)).fetchone()
            if (not controller or controller[0] is not None or controller[1] != "running" or
                    not dispatch or dispatch[0] != "launching" or dispatch[1] is not None):
                raise ValueError("active claimed controller dispatch is required")
            if db.execute("SELECT 1 FROM controller_processes WHERE dispatch_id=? AND action=?",
                          (execution_id, action)).fetchone():
                raise ValueError("controller process action already has an immutable receipt")
            sequence = db.execute("SELECT COALESCE(MAX(sequence),0)+1 FROM controller_processes WHERE dispatch_id=?",
                                  (execution_id,)).fetchone()[0]
            db.execute("INSERT INTO controller_processes VALUES(?,?,?,?,?,?,?,?)",
                       (execution_id, sequence, action, started_at, ended,
                        process_receipt.get("status", "unknown"), raw, receipt_sha))
            return {"dispatchId": execution_id, "sequence": sequence, "action": action,
                    "start": started_at, "end": ended, "status": process_receipt.get("status", "unknown"),
                    "processReceipt": process_receipt, "receiptSha256": receipt_sha}

    def finish_controller_dispatch(self, execution_id, result, process_receipt):
        """Persist the controller receipt/result without adding an Actor attempt."""
        raw_receipt = json.dumps(process_receipt, sort_keys=True)
        receipt_sha = hashlib.sha256(raw_receipt.encode()).hexdigest()
        with self.transaction() as db:
            row = db.execute("SELECT phase,result FROM dispatches WHERE id=?", (execution_id,)).fetchone()
            if not row:
                raise ValueError("unknown controller dispatch")
            if row[1] is not None:
                return json.loads(row[1])
            if row[0] not in {"launching", "recovering"}:
                raise ValueError("controller dispatch was not claimed before process completion")
            changed = db.execute("UPDATE controller_runs SET end=?,status=?,process_receipt=?,receipt_sha256=? "
                                 "WHERE dispatch_id=? AND end IS NULL",
                                 (time.time(), result["status"], raw_receipt, receipt_sha,
                                  execution_id)).rowcount
            if changed != 1:
                raise ValueError("controller timing record is already terminal")
            db.execute("UPDATE dispatches SET phase='finished',result=? WHERE id=?",
                       (json.dumps(result), execution_id))
            return result

    def verify_dispatch_binding(self, record, requested_profile):
        with self.transaction() as db:
            origin = db.execute("SELECT value FROM dispatch_authority").fetchone()
            key = record["authority_sha"] or (limits_sha(json.loads(origin[0])) if origin else None)
            stored = json.loads(record["measurement_profile"]) if record["measurement_profile"] else profile(LEGACY)
            if key != self.authority_key or stored != profile(requested_profile):
                raise ValueError("original dispatch authority/profile binding mismatch")
            binding = db.execute("SELECT value,profile_sha FROM dispatch_authority_history WHERE authority_sha=?", (key,)).fetchone()
            if not binding or binding[1] != profile_sha(requested_profile):
                raise ValueError("stored authority/profile history binding mismatch")
            if record["result"]:
                result = json.loads(record["result"])
                authority = json.loads(binding[0])
                if any(result.get(field, authority.get(field)) != authority.get(field)
                       for field in ("grantSha256", "protocolSha256")):
                    raise ValueError("stored Result authority binding mismatch")
                if (result.get("measurementProfileId", LEGACY) != requested_profile or
                        result.get("measurementProfileSha256", profile_sha(LEGACY)) != profile_sha(requested_profile)):
                    raise ValueError("stored Result profile binding mismatch")
            return stored

    def claim_dispatch(self, execution_id, phase):
        with self.transaction() as db:
            changed = db.execute("UPDATE dispatches SET phase=? WHERE id=? AND phase='reserved'",
                                 (phase, execution_id)).rowcount
            if changed == 1:
                db.execute("UPDATE controller_runs SET status='running' WHERE dispatch_id=? AND end IS NULL",
                           (execution_id,))
            return changed == 1

    def observe(self, attempt, requests, tokens):
        for value in (requests, tokens):
            if value is not None and (type(value) is not int or value < 0):
                raise ValueError("usage must be nonnegative integer or null")
        with self.transaction() as db:
            db.execute("""UPDATE attempts SET turns=CASE WHEN ? IS NULL THEN turns ELSE MAX(COALESCE(turns,0),?) END,
                       tokens=CASE WHEN ? IS NULL THEN tokens ELSE MAX(COALESCE(tokens,0),?) END WHERE id=? AND end IS NULL""",
                       (requests, requests, tokens, tokens, attempt))

    def running_bound(self, task):
        """Known aggregate counters are retrospective; no in-flight token reservation."""
        with self.transaction() as db:
            measurement = self._profile(db)
            now = time.time()
            start, stopped = db.execute("SELECT start,stopped FROM trial").fetchone()
            task_start = db.execute("SELECT start FROM tasks WHERE id=?", (task,)).fetchone()[0]
            remaining = min(self.limits["trialWallSeconds"] - (now - start),
                            self.limits["taskWallSeconds"] - (now - task_start))
            reason = "trial_stopped" if stopped else ("cumulative_wall_deadline" if remaining <= 0 else None)
            for prefix, where, args in (("trial", "", ()), ("task", " WHERE task=?", (task,))):
                for column, metric in (("turns", "ProviderTurns"), ("tokens", "ProviderTokens")):
                    if column == "turns" and not measurement["knownProviderRequestAdmissionCaps"]:
                        continue
                    total = db.execute("SELECT COALESCE(SUM(" + column + "),0) FROM attempts" + where, args).fetchone()[0]
                    if total >= self.limits[prefix + metric]:
                        reason = reason or "retrospective_" + prefix + metric
            return remaining, reason

    def complete_dispatch(self, execution_id, result, requests, tokens):
        with self.transaction() as db:
            return self._complete_dispatch(db, execution_id, result, requests, tokens)

    def abandon_reserved(self, execution_id, result):
        """Claim and terminalize an unlaunched booking in the same transaction."""
        with self.transaction() as db:
            row = db.execute("SELECT phase,result FROM dispatches WHERE id=?", (execution_id,)).fetchone()
            if row[1] is not None:
                return json.loads(row[1])
            if row[0] not in {"reserved", "recovering"}:
                return None  # Launcher won the CAS; never classify it as unlaunched.
            return self._complete_dispatch(db, execution_id, result, None, None)

    def _complete_dispatch(self, db, execution_id, result, requests, tokens):
        row = db.execute("SELECT attempt,result FROM dispatches WHERE id=?", (execution_id,)).fetchone()
        if not row:
            raise ValueError("unknown dispatch")
        if row[1] is not None:
            return json.loads(row[1])
        if row[0] is None:
            raise ValueError("nullable controller dispatch requires finish_controller_dispatch")
        prior = db.execute("SELECT turns,tokens FROM attempts WHERE id=?", (row[0],)).fetchone()
        for previous, final in zip(prior, (requests, tokens)):
            if final is not None and (type(final) is not int or final < 0):
                raise ValueError("usage must be nonnegative integer or null")
            if previous is not None and (final is None or final < previous):
                raise ValueError("observed usage cannot decrease or become unknown")
        db.execute("UPDATE attempts SET end=?,status=?,turns=?,tokens=?,receipt=? WHERE id=? AND end IS NULL",
                   (time.time(), result["status"], requests, tokens, json.dumps(result["receipts"]), row[0]))
        db.execute("UPDATE dispatches SET phase='finished',result=? WHERE id=?", (json.dumps(result), execution_id))
        return result

    def finish(self, attempt, status, *, provider_turns=None, provider_tokens=None, receipt=None):
        for value in (provider_turns, provider_tokens):
            if value is not None and (type(value) is not int or value < 0):
                raise ValueError("usage must be nonnegative integer or null")
        with self.transaction() as db:
            changed = db.execute("UPDATE attempts SET end=?,status=?,turns=?,tokens=?,receipt=? WHERE id=? AND end IS NULL",
                                 (time.time(), status, provider_turns, provider_tokens, json.dumps(receipt), attempt)).rowcount
            if changed != 1:
                raise ValueError("unknown/already finished attempt")

    def stop(self):
        with self.transaction() as db:
            db.execute("UPDATE trial SET stopped=1")

    def human(self, seconds, note):
        if not isinstance(seconds, (int, float)) or not 0 <= seconds < float("inf"):
            raise ValueError("finite nonnegative human duration required")
        with self.transaction() as db:
            db.execute("INSERT INTO human VALUES(?,?)", (seconds, note))
            total = db.execute("SELECT SUM(seconds) FROM human").fetchone()[0]
            if total >= self.limits["trialActiveHumanSeconds"]:
                db.execute("UPDATE trial SET stopped=1")
            return total

    def snapshot(self):
        with self.transaction() as db:
            db.row_factory = sqlite3.Row
            attempts = [dict(row) for row in db.execute("SELECT * FROM attempts ORDER BY start,id")]
            controller_runs = [dict(row) for row in db.execute("SELECT * FROM controller_runs ORDER BY start,dispatch_id")]
            controller_processes = [dict(row) for row in db.execute(
                "SELECT * FROM controller_processes ORDER BY dispatch_id,sequence")]
            return {"attempts": attempts, "actorSessions": len(attempts),
                    "controllerRuns": controller_runs, "controllerProcesses": controller_processes,
                    "measurementProfile": self._profile(db),
                    "profileHistory": [dict(row) for row in db.execute("SELECT * FROM measurement_profiles ORDER BY sequence")],
                    "providerTurns": None if any(row["turns"] is None for row in attempts) else sum(row["turns"] for row in attempts),
                    "providerTokens": None if any(row["tokens"] is None for row in attempts) else sum(row["tokens"] for row in attempts),
                    "hardProviderTurnTokenCap": False, "remainingActiveCalls": sum(row["end"] is None for row in attempts),
                    "humanActiveSeconds": db.execute("SELECT COALESCE(SUM(seconds),0) FROM human").fetchone()[0],
                    "stopped": bool(db.execute("SELECT stopped FROM trial").fetchone()[0])}
