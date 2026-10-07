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
        now, limits = time.time(), self.limits
        with self.transaction() as db:
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
