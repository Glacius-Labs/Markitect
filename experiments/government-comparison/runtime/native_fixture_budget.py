"""One finite, provider-free two-product integration allocation; no refills."""
from __future__ import annotations

import hashlib
import json
from contextlib import contextmanager
from pathlib import Path
import sqlite3
import time

KEY = "native-s1-integration-fixtures-20261008"
CORRECTION_KEY = "native-s1-corrected-integration-20261008-r2"
CORRECTION_POINTER = "threads[name=Scientist].evidence.correctedNativeIntegrationGrant"
SOURCE_THREAD = "01a11367-a781-7683-a20f-46e12614dcb4"
CORRECTION_BASIS = "f35c8209cd8902bff17e69f05ee7716e352ee4b2"
CORRECTION_SOURCE_PATH = (r"C:\Users\Consiliari\Documents\Scientist-Probes\native-metadata-fixtures-20261008-r2"
                          r"\coordinator-authority-snapshot.json")
CORRECTION_SOURCE_SHA256 = "1b2e361ce2a321149a65694b6586bab57e74388dc5486bc2d4f8cbedb2b7a75c"
DEADLINE = 38
# One product controller session and at most two concurrent role sessions, including the
# existing process wrapper's two possible five-second cleanup waits and margin.
PROCESS_SECONDS_RESERVED = 3 * (DEADLINE + 12)
CORRECTION_CUMULATIVE = {"government": {"starts": 4, "seconds": 600},
                         "classic": {"starts": 6, "seconds": 900}}
CORRECTION_PRIOR = {"government": {"starts": 2, "seconds": 300,
                                    "labels": {"inspect-constitution", "government-native-positive/queue"}},
                    "classic": {"starts": 1, "seconds": 150,
                                "labels": {"classic-native-positive/execute"}}}
CORRECTION_LABELS = {
    "government": {"government-native-corrected-r2/queue", "government-native-corrected-r2/resume"},
    "classic": {"classic-native-corrected-r2/execute", "classic-native-corrected-r2/apply",
                "classic-native-corrected-r2/verify", "classic-native-corrected-r2/audit",
                "classic-native-corrected-r2/apply-replay"}}


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def _released_binding(request, field, expected_path, expected_sha, source_key):
    binding = request.get(field)
    if (not isinstance(binding, dict) or set(binding) != {"path", "sha256", "sourceKey"} or
            binding.get("sourceKey") != source_key or
            Path(binding.get("path", "")).resolve(strict=True) != expected_path or
            binding.get("sha256") != expected_sha):
        raise ValueError(f"Request {field} does not bind the exact released source file")
    released = {str(Path(item["path"]).resolve(strict=True)): item.get("sha256")
                for item in request.get("releasedInputs", [])
                if isinstance(item, dict) and set(item) == {"path", "sha256"}
                and isinstance(item.get("path"), str)}
    if released.get(str(expected_path)) != expected_sha:
        raise ValueError(f"Request {field} must be an exact released input")


def _is_released(request, path, expected_sha):
    for item in request.get("releasedInputs", []):
        if (isinstance(item, dict) and set(item) == {"path", "sha256"} and
                isinstance(item.get("path"), str) and Path(item["path"]).resolve(strict=True) == path and
                item.get("sha256") == expected_sha):
            return True
    return False


def _strict_load(path, label):
    raw = path if isinstance(path, bytes) else Path(path).read_bytes()
    def pairs(items):
        result = {}
        for key, value in items:
            if key in result:
                raise ValueError(f"{label} has duplicate JSON keys")
            result[key] = value
        return result
    try:
        return json.loads(raw, object_pairs_hook=pairs)
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise ValueError(f"{label} is not valid JSON") from exc


def validate_correction_binding(request, original_grant_path, original_grant_sha):
    """Validate the exact additive correction without creating or changing a ledger."""
    if request.get("mode") != "mechanical":
        raise ValueError("provider-free corrected fixture allocation requires mechanical mode")
    binding = request.get("nativeFixtureCorrection")
    if (not isinstance(binding, dict) or set(binding) != {"path", "sha256", "sourceKey"} or
            binding.get("sourceKey") != CORRECTION_KEY):
        raise ValueError("exact corrected fixture grant path/SHA/key binding required")
    correction_input_path = Path(binding["path"])
    if correction_input_path.is_symlink():
        raise ValueError("corrected fixture grant cannot be a symlink")
    correction_path = correction_input_path.resolve(strict=True)
    correction_sha = binding["sha256"]
    if sha(correction_path) != correction_sha:
        raise ValueError("corrected fixture grant digest mismatch")
    _released_binding(request, "nativeFixtureCorrection", correction_path, correction_sha, CORRECTION_KEY)
    doc = _strict_load(correction_path, "corrected fixture grant")
    snapshot = Path(doc.get("sourceCoordinationPath", ""))
    expected_snapshot = Path(CORRECTION_SOURCE_PATH)
    if (doc.get("sourceThreadId") != SOURCE_THREAD or doc.get("sourceJsonPointer") != CORRECTION_POINTER or
            not snapshot.is_absolute() or not snapshot.is_file() or snapshot.is_symlink() or
            snapshot.resolve(strict=True) != expected_snapshot.resolve(strict=True) or
            doc.get("sourceCoordinationSha256") != CORRECTION_SOURCE_SHA256):
        raise ValueError("corrected grant coordinator source binding mismatch")
    snapshot_raw = snapshot.read_bytes()
    snapshot_sha = hashlib.sha256(snapshot_raw).hexdigest()
    if snapshot_sha != doc.get("sourceCoordinationSha256") or snapshot_sha != CORRECTION_SOURCE_SHA256:
        raise ValueError("corrected grant coordinator snapshot digest mismatch")
    if not _is_released(request, snapshot.resolve(strict=True), snapshot_sha):
        raise ValueError("canonical corrected-grant snapshot must be an exact released Request input")
    source = _strict_load(snapshot_raw, "corrected grant coordinator snapshot")
    scientist = next((item for item in source.get("threads", [])
                      if isinstance(item, dict) and item.get("name") == "Scientist"), None)
    grant = doc.get("grant")
    if (not scientist or scientist.get("evidence", {}).get("correctedNativeIntegrationGrant") != grant or
            not isinstance(grant, dict) or grant.get("key") != CORRECTION_KEY):
        raise ValueError("corrected grant differs from its canonical Scientist source")
    original_doc = _strict_load(original_grant_path, "original fixture source grant")
    if sha(original_grant_path) != original_grant_sha:
        raise ValueError("original source grant changed before corrected allocation")
    original = original_doc.get("grant", {})
    basis = grant.get("basis", {})
    limits = grant.get("limits", {})
    expected_basis = {"negativeHandoffSha": CORRECTION_BASIS,
                      "previousAllocation": KEY,
                      "previousNativeStarts": {"government": 2, "classic": 1},
                      "previousReservedSessionSeconds": {"government": 300, "classic": 150},
                      "previousNativeWrapperAttempts": {"government": 1, "classic": 0},
                      "previousDelegateAttempts": 0}
    expected_limits = {
        "government": {"maxAdditionalNativeStarts": 2,
                       "maxAdditionalRoleInvocationsIncludingFailedWrapperStarts": 6,
                       "maxAdditionalReservedControllerAndRoleSessionSeconds": 300,
                       "cases": "One queue attempt; one directly related resume/replay only after the positive case succeeds. No repeated inspect."},
        "classic": {"maxAdditionalNativeStarts": 5,
                    "maxAdditionalRoleInvocationsIncludingFailedWrapperStarts": 6,
                    "maxAdditionalReservedControllerAndRoleSessionSeconds": 750,
                    "cases": "One Execute, guarded Apply after bound external review, fresh Verify, Audit, and stale Apply replay; later steps only after their prerequisites pass."},
        "maxParallelRoles": 2, "productsRunSequentially": True,
        "cumulativeNativeStartCeiling": {"government": 4, "classic": 6, "total": 10},
        "cumulativeReservedSessionSecondsCeiling": {"government": 600, "classic": 900, "total": 1500},
        "previousOverallNativeStartCeiling": 16,
        "previousOverallReservedSessionSecondsCeiling": 2400,
        "nativeProcessDeadlineSeconds": 38,
        "controllerWindowSeconds": {"government": 38, "classic": 180},
        "reservedSessionSecondsPerNativeStart": PROCESS_SECONDS_RESERVED}
    if basis != expected_basis or not isinstance(limits, dict):
        raise ValueError("corrected grant basis differs from the exact accepted prior allocation")
    if any(limits.get(key) != value for key, value in expected_limits.items()):
        raise ValueError("corrected grant limits differ from the exact additive allocation")
    if (grant.get("realActorCalls") != 0 or grant.get("providerCalls") != 0 or
            grant.get("metadataAppServerTrees") != 0 or grant.get("studyCells") != 0 or
            grant.get("newPurchases") is not False or grant.get("productMutations") is not False):
        raise ValueError("corrected grant exceeds its provider-free, no-mutation authorization")
    products = {item.get("name", "").lower(): item for item in original.get("products", [])
                if isinstance(item, dict)}
    if set(products) != {"government", "classic"}:
        raise ValueError("original accepted product pins are incomplete")
    return {"path": str(correction_path), "sha256": correction_sha,
            "sourceCoordinationPath": str(snapshot),
            "sourceCoordinationSha256": snapshot_sha, "grant": grant,
            "limits": expected_limits, "products": products}


class FixtureBudget:
    def __init__(self, path, grant_path, grant_sha256, binaries, *, request=None):
        self.path = Path(path).resolve()
        self.grant_path = Path(grant_path).resolve(strict=True)
        if sha(self.grant_path) != grant_sha256:
            raise ValueError("fixture allocation digest mismatch")
        self.grant_sha = grant_sha256
        document = _strict_load(self.grant_path, "fixture source grant")
        snapshot = Path(document["sourceCoordinationPath"])
        if sha(snapshot) != document["sourceCoordinationSha256"]:
            raise ValueError("fixture source allocation snapshot mismatch")
        source = _strict_load(snapshot, "fixture source coordination snapshot")
        scientist = next(t for t in source["threads"] if t["name"] == "Scientist")
        grant = document["grant"]
        if (grant != scientist["evidence"]["nativeIntegrationPreparationGrant"]
                or document["sourceThreadId"] != SOURCE_THREAD
                or grant["key"] != KEY
                or grant["realActorStartsAuthorized"] != 0
                or grant["providerCallsAuthorized"] != 0
                or grant["studyCellsAuthorized"] != 0
                or grant["perProduct"]["nativeCliOrControllerStartsMaximum"] != 8
                or grant["perProduct"]["deterministicRoleStartsMaximum"] != 12
                or grant["perProduct"]["maxParallel"] != 2
                or grant["perProduct"]["totalProcessSecondsMaximum"] != 1200):
            raise ValueError("fixture allocation is not the exact finite operator grant")
        if request is not None:
            _released_binding(request, "nativeFixtureGrant", self.grant_path, grant_sha256, KEY)
        self.binaries = {}
        for item in grant["products"]:
            product = item["name"].lower()
            binary = Path(binaries[product]).resolve(strict=True)
            if sha(binary) != item["binarySha256"]:
                raise ValueError("fixture binary differs from allocated candidate")
            self.binaries[product] = str(binary)
        self.correction = None
        if request is not None and request.get("nativeFixtureCorrection") is not None:
            self.correction = self._validate_correction(request)
        if self.correction is not None and not self.path.is_file():
            raise ValueError("corrected allocation requires the original nonempty native-start history")
        if self.correction is None:
            self.path.parent.mkdir(parents=True, exist_ok=True)
        else:
            original_document = _strict_load(self.grant_path, "original fixture source grant")
            self.original_snapshot_path = Path(original_document["sourceCoordinationPath"])
            self.original_snapshot_sha = original_document["sourceCoordinationSha256"]
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
            if self.correction is not None:
                db.execute("""CREATE TABLE IF NOT EXISTS corrections(
                    source_key TEXT PRIMARY KEY, correction_sha TEXT NOT NULL,
                    source_snapshot_sha TEXT NOT NULL, basis_sha TEXT NOT NULL)""")
                self._validate_prior_history(db)
                identity = (CORRECTION_KEY, self.correction["sha256"],
                            self.correction["sourceCoordinationSha256"], CORRECTION_BASIS)
                rows = db.execute("SELECT source_key,correction_sha,source_snapshot_sha,basis_sha FROM corrections").fetchall()
                if rows and rows != [identity]:
                    raise ValueError("corrected allocation history is already bound to a different correction")
                db.execute("INSERT OR IGNORE INTO corrections VALUES(?,?,?,?)", identity)
            elif db.execute("SELECT 1 FROM sqlite_master WHERE type='table' AND name='corrections'").fetchone():
                if db.execute("SELECT 1 FROM corrections LIMIT 1").fetchone():
                    raise ValueError("corrected history requires its exact Request-bound correction")

    def _validate_correction(self, request):
        corrected = validate_correction_binding(request, self.grant_path, self.grant_sha)
        for product, item in corrected["products"].items():
            binary = self.binaries.get(product)
            if not binary or sha(binary) != item.get("binarySha256"):
                raise ValueError("corrected allocation must retain original accepted product binaries")
        return corrected

    def _validate_prior_history(self, db):
        allocation = db.execute("SELECT grant_sha FROM allocation").fetchall()
        if allocation != [(self.grant_sha,)]:
            raise ValueError("corrected allocation must append to the original source-grant ledger")
        rows = db.execute("SELECT product,label,finished,reserved_seconds FROM starts ORDER BY product,label").fetchall()
        historical = {(product, label) for product, value in CORRECTION_PRIOR.items()
                      for label in value["labels"]}
        historical_rows = [row for row in rows if (row[0], row[1]) in historical]
        expected = {(product, label, PROCESS_SECONDS_RESERVED) for product, label in historical}
        observed = {(product, label, seconds) for product, label, _finished, seconds in historical_rows}
        if (len(historical_rows) != 3 or observed != expected or
                any(finished is None for product, label, finished, seconds in historical_rows)):
            raise ValueError("corrected allocation requires the exact three immutable prior native starts")
        for product, value in CORRECTION_PRIOR.items():
            product_rows = [row for row in historical_rows if row[0] == product]
            if len(product_rows) != value["starts"] or sum(row[3] for row in product_rows) != value["seconds"]:
                raise ValueError("prior native start/time history differs from the correction basis")
        new_rows = [row for row in rows if (row[0], row[1]) not in historical]
        if any(product not in CORRECTION_LABELS or label not in CORRECTION_LABELS[product]
               for product, label, _finished, _seconds in new_rows):
            raise ValueError("corrected native history contains an unallocated start label")
        limits = CORRECTION_CUMULATIVE
        for product in limits:
            product_rows = [row for row in rows if row[0] == product]
            new_product_rows = [row for row in new_rows if row[0] == product]
            if (len(product_rows) > limits[product]["starts"] or
                    len(new_product_rows) > limits[product]["starts"] - CORRECTION_PRIOR[product]["starts"] or
                    sum(row[3] for row in product_rows) > limits[product]["seconds"] or
                    any(row[3] != PROCESS_SECONDS_RESERVED for row in product_rows)):
                raise ValueError("corrected native history exceeds its cumulative start/time ceiling")
        if any(row[0] not in limits for row in new_rows):
            raise ValueError("corrected native history contains an unknown product")
        active = [row for row in new_rows if row[2] is None]
        if len(active) > 1:
            raise ValueError("corrected native history exceeds maxParallel=1")
        corrections = db.execute("SELECT source_key,correction_sha,source_snapshot_sha,basis_sha FROM corrections").fetchall()
        expected_identity = (CORRECTION_KEY, self.correction["sha256"],
                             self.correction["sourceCoordinationSha256"], CORRECTION_BASIS)
        if corrections and corrections != [expected_identity]:
            raise ValueError("corrected allocation history is bound to another source decision")

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
        if self.correction is not None:
            if (self.original_snapshot_path.is_symlink() or
                    sha(self.original_snapshot_path) != self.original_snapshot_sha or
                    Path(self.correction["path"]).is_symlink() or
                    Path(self.correction["sourceCoordinationPath"]).is_symlink() or
                    sha(self.correction["path"]) != self.correction["sha256"] or
                    sha(self.correction["sourceCoordinationPath"]) != self.correction["sourceCoordinationSha256"]):
                raise ValueError("corrected allocation or canonical source changed before start")
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
            if self.correction is None:
                start_limit, seconds_limit, parallel_limit = 8, 1200, 2
            else:
                allocation = CORRECTION_CUMULATIVE.get(product)
                if allocation is None:
                    raise ValueError("unknown corrected allocation product")
                start_limit, seconds_limit, parallel_limit = allocation["starts"], allocation["seconds"], 1
                historical = {(item, prior_label) for item, value in CORRECTION_PRIOR.items()
                              for prior_label in value["labels"]}
                history = db.execute("SELECT product,label FROM starts ORDER BY claimed").fetchall()
                correction_products = [item for item, prior_label in history
                                       if (item, prior_label) not in historical]
                product_order = []
                for item in correction_products:
                    if not product_order or product_order[-1] != item:
                        product_order.append(item)
                if len(product_order) > 1 and product_order[-1] != product:
                    raise ValueError("corrected products must run sequentially without interleaving")
            if count >= start_limit or seconds + PROCESS_SECONDS_RESERVED > seconds_limit:
                raise ValueError("finite native fixture allocation exhausted")
            active = db.execute("SELECT COUNT(*) FROM starts WHERE finished IS NULL").fetchone()[0]
            if active >= parallel_limit:
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
            if self.correction is None:
                allocation = {"allocationId": KEY, "grantSha256": self.grant_sha,
                              "nativeStartLimits": {"government": 8, "classic": 8},
                              "reservedSecondsPerProduct": 1200, "maxParallel": 2}
            else:
                allocation = {"allocationId": CORRECTION_KEY, "grantSha256": self.grant_sha,
                              "correctionSha256": self.correction["sha256"],
                              "sourceCoordinationSha256": self.correction["sourceCoordinationSha256"],
                              "cumulativeNativeStartCeiling": {key: value["starts"] for key, value in CORRECTION_CUMULATIVE.items()},
                              "cumulativeReservedSessionSecondsCeiling": {key: value["seconds"] for key, value in CORRECTION_CUMULATIVE.items()},
                              "maxParallel": 1, "productsRunSequentially": True}
            return {**allocation,
                    "starts":[dict(row) for row in db.execute("SELECT * FROM starts ORDER BY claimed")],
                    "deadlineSeconds":DEADLINE,"reservedSecondsPerStart":PROCESS_SECONDS_RESERVED,
                    "accountingUnit":"bounded controller and deterministic role sessions including cleanup margin",
                    "arbitraryOsDescendantWallSeconds":None,
                    "refills":0,"realActorStarts":0,"providerCalls":0,"studyCells":0}
