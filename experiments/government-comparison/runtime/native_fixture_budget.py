"""One finite, provider-free two-product integration allocation; no refills."""
from __future__ import annotations

import hashlib
import json
from contextlib import contextmanager
from pathlib import Path
import sqlite3
import time

KEY = "native-s1-integration-fixtures-20261008"
DEADLINE = 38
# One product controller session and at most two concurrent role sessions, including the
# existing process wrapper's two possible five-second cleanup waits and margin.
PROCESS_SECONDS_RESERVED = 3 * (DEADLINE + 12)


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


class FixtureBudget:
    def __init__(self, path, grant_path, grant_sha256, binaries):
        self.path = Path(path).resolve()
        self.grant_path = Path(grant_path).resolve(strict=True)
        if sha(self.grant_path) != grant_sha256:
            raise ValueError("fixture allocation digest mismatch")
        self.grant_sha = grant_sha256
        document = json.loads(self.grant_path.read_bytes())
        snapshot = Path(document["sourceCoordinationPath"])
        if sha(snapshot) != document["sourceCoordinationSha256"]:
            raise ValueError("fixture source allocation snapshot mismatch")
        source = json.loads(snapshot.read_bytes())
        scientist = next(t for t in source["threads"] if t["name"] == "Scientist")
        grant = document["grant"]
        if (grant != scientist["evidence"]["nativeIntegrationPreparationGrant"]
                or document["sourceThreadId"] != "01a11367-a781-7683-a20f-46e12614dcb4"
                or grant["key"] != KEY
                or grant["realActorStartsAuthorized"] != 0
                or grant["providerCallsAuthorized"] != 0
                or grant["studyCellsAuthorized"] != 0
                or grant["perProduct"]["nativeCliOrControllerStartsMaximum"] != 8
                or grant["perProduct"]["deterministicRoleStartsMaximum"] != 12
                or grant["perProduct"]["maxParallel"] != 2
                or grant["perProduct"]["totalProcessSecondsMaximum"] != 1200):
            raise ValueError("fixture allocation is not the exact finite operator grant")
        self.binaries = {}
        for item in grant["products"]:
            product = item["name"].lower()
            binary = Path(binaries[product]).resolve(strict=True)
            if sha(binary) != item["binarySha256"]:
                raise ValueError("fixture binary differs from allocated candidate")
            self.binaries[product] = str(binary)
        self.path.parent.mkdir(parents=True, exist_ok=True)
        with self._db() as db:
            db.execute("CREATE TABLE IF NOT EXISTS allocation(grant_sha TEXT PRIMARY KEY)")
            rows = db.execute("SELECT grant_sha FROM allocation").fetchall()
            if rows and rows != [(grant_sha256,)]:
                raise ValueError("fixture allocation cannot be replaced or refilled")
            db.execute("INSERT OR IGNORE INTO allocation VALUES(?)", (grant_sha256,))
            db.execute("""CREATE TABLE IF NOT EXISTS starts(
                product TEXT, label TEXT, argv TEXT, claimed REAL, finished REAL,
                reserved_seconds INTEGER, receipt TEXT,
                PRIMARY KEY(product,label))""")

    def _connection(self):
        db = sqlite3.connect(self.path, timeout=10)
        db.execute("PRAGMA busy_timeout=10000")
        return db

    @contextmanager
    def _db(self):
        db = self._connection()
        try:
            with db:
                yield db
        finally:
            db.close()

    def reserve(self, product, label, argv):
        if sha(self.grant_path) != self.grant_sha:
            raise ValueError("fixture allocation changed before start")
        if (product not in self.binaries or not isinstance(label, str) or not label
                or not isinstance(argv, list) or not argv
                or str(Path(argv[0]).resolve()) != self.binaries[product]):
            raise ValueError("exact allocated product command required")
        if sha(argv[0]) != next(item["binarySha256"] for item in
                json.loads(self.grant_path.read_bytes())["grant"]["products"]
                if item["name"].lower() == product):
            raise ValueError("allocated executable changed before start")
        db = self._connection()
        try:
            db.execute("BEGIN IMMEDIATE")
            if db.execute("SELECT 1 FROM starts WHERE product=? AND label=?", (product, label)).fetchone():
                raise ValueError("native start already claimed; no retry")
            count, seconds = db.execute("SELECT COUNT(*),COALESCE(SUM(reserved_seconds),0) FROM starts WHERE product=?",
                                        (product,)).fetchone()
            if count >= 8 or seconds + PROCESS_SECONDS_RESERVED > 1200:
                raise ValueError("finite native fixture allocation exhausted")
            if db.execute("SELECT COUNT(*) FROM starts WHERE finished IS NULL").fetchone()[0] >= 2:
                raise ValueError("native fixture controller parallelism exhausted")
            db.execute("INSERT INTO starts VALUES(?,?,?,?,NULL,?,NULL)",
                       (product, label, json.dumps(argv), time.time(), PROCESS_SECONDS_RESERVED))
            db.commit()
        except BaseException:
            db.rollback()
            raise
        finally:
            db.close()

    def finish(self, product, label, receipt):
        with self._db() as db:
            row = db.execute("SELECT argv,finished FROM starts WHERE product=? AND label=?", (product,label)).fetchone()
            if not row or row[1] is not None or receipt.get("argv") != json.loads(row[0]):
                raise ValueError("native fixture receipt differs from its one-start reservation")
            db.execute("UPDATE starts SET finished=?,receipt=? WHERE product=? AND label=?",
                       (time.time(), json.dumps(receipt,sort_keys=True), product,label))

    def snapshot(self):
        with self._db() as db:
            db.row_factory = sqlite3.Row
            return {"allocationId":KEY,"grantSha256":self.grant_sha,
                    "starts":[dict(row) for row in db.execute("SELECT * FROM starts ORDER BY claimed")],
                    "deadlineSeconds":DEADLINE,"reservedSecondsPerStart":PROCESS_SECONDS_RESERVED,
                    "accountingUnit":"bounded controller and deterministic role sessions including cleanup margin",
                    "arbitraryOsDescendantWallSeconds":None,
                    "refills":0,"realActorStarts":0,"providerCalls":0,"studyCells":0}
