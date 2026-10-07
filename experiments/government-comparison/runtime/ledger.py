"""Transactional, no-refill shared trial accounting; unknown usage stays unknown."""
from contextlib import contextmanager
import json
from pathlib import Path
import sqlite3
import time
import uuid


class LimitReached(ValueError):
    pass


class Ledger:
    def __init__(self, path, trial_id, limits):
        self.path = str(path)
        Path(path).parent.mkdir(parents=True, exist_ok=True)
        with self.transaction() as db:
            schema = """CREATE TABLE IF NOT EXISTS trial(id TEXT PRIMARY KEY, limits TEXT, start REAL, stopped INTEGER);
                CREATE TABLE IF NOT EXISTS tasks(id TEXT PRIMARY KEY, start REAL);
                CREATE TABLE IF NOT EXISTS attempts(id TEXT PRIMARY KEY, task TEXT, purpose TEXT, start REAL,
                    end REAL, status TEXT, turns INTEGER, tokens INTEGER, receipt TEXT);
                CREATE TABLE IF NOT EXISTS human(seconds REAL, note TEXT);"""
            for statement in schema.split(";"):
                if statement.strip():
                    db.execute(statement)
            prior = db.execute("SELECT id,limits FROM trial").fetchone()
            encoded = json.dumps(limits, sort_keys=True)
            if prior and prior != (trial_id, encoded):
                raise ValueError("ledger identity/profile cannot change or refill")
            if not prior:
                db.execute("INSERT INTO trial VALUES(?,?,?,0)", (trial_id, encoded, time.time()))
        self.limits = dict(limits)

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

        Unknown or retrospective provider usage is never a reservation for unseen
        provider calls. A live dispatcher still needs an enforced provider-turn gate.
        """
        with self.transaction() as db:
            return self._reserve(db, task, purpose)

    def _reserve(self, db, task, purpose):
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
            if any(row[1] is not None and (row[2] is None or row[3] is None) for row in rows):
                raise LimitReached("unknown provider usage: next actor blocked")
            for index, metric in ((2, "ProviderTurns"), (3, "ProviderTokens")):
                if sum(row[index] or 0 for row in rows) >= limits[prefix + metric]:
                    raise LimitReached(prefix + metric)
        attempt = uuid.uuid4().hex
        db.execute("INSERT INTO attempts(id,task,purpose,start) VALUES(?,?,?,?)", (attempt, task, purpose, now))
        return attempt

    def bind_dispatch(self, authority):
        """One immutable grant/protocol/profile/ledger binding for this trial."""
        with self.transaction() as db:
            db.execute("CREATE TABLE IF NOT EXISTS dispatch_authority(value TEXT)")
            db.execute("""CREATE TABLE IF NOT EXISTS dispatches(id TEXT PRIMARY KEY, execution_sha TEXT,
                       attempt TEXT UNIQUE, phase TEXT, request BLOB, command TEXT, result TEXT)""")
            value = json.dumps(authority, sort_keys=True)
            prior = db.execute("SELECT value FROM dispatch_authority").fetchone()
            if prior and prior[0] != value:
                raise ValueError("dispatch authority cannot change or refill")
            if not prior:
                db.execute("INSERT INTO dispatch_authority VALUES(?)", (value,))

    def dispatch_record(self, execution_id):
        with self.transaction() as db:
            db.row_factory = sqlite3.Row
            row = db.execute("SELECT * FROM dispatches WHERE id=?", (execution_id,)).fetchone()
            return dict(row) if row else None

    def reserve_dispatch(self, execution_id, execution_sha, raw_request, command, task, purpose, maximum):
        with self.transaction() as db:
            if db.execute("SELECT id FROM dispatches WHERE id=?", (execution_id,)).fetchone():
                raise LimitReached("dispatch already booked; use resume")
            if db.execute("SELECT COUNT(*) FROM attempts").fetchone()[0] >= maximum:
                raise LimitReached("grant session limit")
            attempt = self._reserve(db, task, purpose)
            db.execute("INSERT INTO dispatches VALUES(?,?,?,'reserved',?,?,NULL)",
                       (execution_id, execution_sha, attempt, raw_request, json.dumps(command)))
            return attempt

    def claim_dispatch(self, execution_id, phase):
        with self.transaction() as db:
            return db.execute("UPDATE dispatches SET phase=? WHERE id=? AND phase='reserved'",
                              (phase, execution_id)).rowcount == 1

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
            now = time.time()
            start, stopped = db.execute("SELECT start,stopped FROM trial").fetchone()
            task_start = db.execute("SELECT start FROM tasks WHERE id=?", (task,)).fetchone()[0]
            remaining = min(self.limits["trialWallSeconds"] - (now - start),
                            self.limits["taskWallSeconds"] - (now - task_start))
            reason = "trial_stopped" if stopped else ("cumulative_wall_deadline" if remaining <= 0 else None)
            for prefix, where, args in (("trial", "", ()), ("task", " WHERE task=?", (task,))):
                for column, metric in (("turns", "ProviderTurns"), ("tokens", "ProviderTokens")):
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
            return {"attempts": attempts, "actorSessions": len(attempts),
                    "providerTurns": None if any(row["turns"] is None for row in attempts) else sum(row["turns"] for row in attempts),
                    "providerTokens": None if any(row["tokens"] is None for row in attempts) else sum(row["tokens"] for row in attempts),
                    "hardProviderTurnTokenCap": False, "remainingActiveCalls": sum(row["end"] is None for row in attempts),
                    "humanActiveSeconds": db.execute("SELECT COALESCE(SUM(seconds),0) FROM human").fetchone()[0],
                    "stopped": bool(db.execute("SELECT stopped FROM trial").fetchone()[0])}
