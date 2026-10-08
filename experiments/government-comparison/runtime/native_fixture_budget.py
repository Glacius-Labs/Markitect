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
R3_KEY = "native-s1-contract-corrected-integration-20261008-r3"
R3_ACTIVE_STATUS = "Active; shared slot assigned. All other preflight and finite execution prerequisites remain mandatory."
R3_DISPATCH_IDS = {"government": "government-native-contract-corrected-r3",
                   "classic": "classic-native-contract-corrected-r3"}
R3_POINTER = "threads[name=Scientist].evidence.contractCorrectedNativeIntegrationGrant"
R3_SOURCE_THREAD = SOURCE_THREAD
R3_ENVELOPE_PATH = (r"C:\Users\Consiliari\Documents\Scientist-Probes\native-metadata-fixtures-20261008-r3"
                    r"\released-r3-a1-grant.json")
R3_ENVELOPE_SHA256 = "b45a048923102feaa2040a769281cc2c258d66be09c2ce3561238b4b74c3f932"
R3_SOURCE_PATH = (r"C:\Users\Consiliari\Documents\Scientist-Probes\native-metadata-fixtures-20261008-r3"
                  r"\coordinator-r3-a1-snapshot.json")
R3_SOURCE_SHA256 = "29595284a6e4202129d9f014aee6e1fec545d19bdfe028411687d41ec6b8fa17"
R3_COORDINATION_PATH = r"C:\Users\Consiliari\Glacius Labs\Markitect\docs\design\government\coordination-state.json"
R3_HISTORY_PATH = (Path(__file__).parents[1] / "evidence" / "native-integration" / "run-2" /
                   "external-snapshots" / "native-starts.sqlite").resolve()
R3_HISTORY_SHA256 = "c5769c0c204cfbc5d17d95320c8b2288b746aa785c16796ce0d7ab433b58fbf5"
R3_BASE_GRANT_PATH = r"C:\Users\Consiliari\Documents\Scientist-Probes\native-metadata-fixtures-20261008\released-native-grant.json"
R3_BASE_GRANT_SHA256 = "b917f5a5eb99f0a607fd282a81acdf7b085e6a1dba14f89c8d0c32f349c82c3e"
R3_BASE_PRODUCTS = {
    "government": {"name": "Government", "sourceSha": "04e225d5caee78c2a198607143863fca1e829750",
                   "binarySha256": "12241f325e4af59451e4021b31d9e6f5b829b5de06c94d35eaabfb9d663aa51f"},
    "classic": {"name": "Classic", "sourceSha": "c91363b7ac4decbe87212ff0f588b5451581a152",
                "releaseVersion": "v0.14.1",
                "binarySha256": "2cad55efad64f15d7f57638bea78312918fbf7d181c730f188b9921504da71c4"}}
R3_PRIOR_LABELS = {
    "government": {"inspect-constitution", "government-native-positive/queue",
                   "government-native-corrected-r2/queue"},
    "classic": {"classic-native-positive/execute", "classic-native-corrected-r2/execute"}}
R3_LABELS = {
    "government": {"government-native-contract-corrected-r3/queue",
                   "government-native-contract-corrected-r3/resume"},
    "classic": {"classic-native-contract-corrected-r3/execute",
                "classic-native-contract-corrected-r3/apply",
                "classic-native-contract-corrected-r3/verify",
                "classic-native-contract-corrected-r3/audit",
                "classic-native-contract-corrected-r3/apply-replay"}}
R3_CUMULATIVE = {"government": {"starts": 5, "seconds": 750},
                 "classic": {"starts": 7, "seconds": 1050}}
R4_KEY = "government-serialization-native-20261008-r4"
R4_DISPATCH_ID = "government-native-serialization-r4"
R4_POINTER = "threads[name=Scientist].evidence.governmentSerializationNativeGrant"
R4_SOURCE_THREAD = SOURCE_THREAD
R4_ENVELOPE_PATH = (r"C:\Users\Consiliari\Documents\Scientist-Probes\native-government-serialization-20261008-r4"
                    r"\released-r4-grant.json")
R4_ENVELOPE_SHA256 = "e4991c497c1ce086ee9a8bd7708409a6ec3e0de8bd114104eaef319df018e03c"
R4_SOURCE_PATH = (r"C:\Users\Consiliari\Documents\Scientist-Probes\native-government-serialization-20261008-r4"
                  r"\coordinator-r4-preflight-snapshot.json")
R4_SOURCE_SHA256 = "41bbbdf4e625d4ef018d07737b2d2decbc0b360992dfccb08ff23c4d46bec333"
R4_COORDINATION_PATH = R3_COORDINATION_PATH
R4_ACTIVE_STATUS = R3_ACTIVE_STATUS
R4_HISTORY_PATH = (Path(__file__).parents[1] / "evidence" / "government-serialization-native-20261008-r4" /
                   "historical-native-starts.sqlite").resolve()
R4_HISTORY_SHA256 = "5c08ba595cad4e6c8d4185027858a59b62024c6bc68a294b731c272130b3c097"
R4_PRIOR_LABELS = {
    ("government", "inspect-constitution"),
    ("government", "government-native-positive/queue"),
    ("classic", "classic-native-positive/execute"),
    ("government", "government-native-corrected-r2/queue"),
    ("classic", "classic-native-corrected-r2/execute"),
    ("government", "government-native-contract-corrected-r3/queue"),
    ("classic", "classic-native-contract-corrected-r3/execute"),
    ("classic", "classic-native-contract-corrected-r3/apply"),
    ("classic", "classic-native-contract-corrected-r3/verify"),
    ("classic", "classic-native-contract-corrected-r3/audit"),
    ("classic", "classic-native-contract-corrected-r3/apply-replay")}
R4_LABELS = {"government-native-serialization-r4/queue",
             "government-native-serialization-r4/resume"}
R4_CUMULATIVE = {"government": {"starts": 6, "seconds": 900},
                 "classic": {"starts": 7, "seconds": 1050}}
R4_SOURCE_SHA = "04e225d5caee78c2a198607143863fca1e829750"
R4_BINARY_SHA256 = "12241f325e4af59451e4021b31d9e6f5b829b5de06c94d35eaabfb9d663aa51f"
R4_DELEGATE_SHA256 = "e8b8e5087f994a975efc2228301cf7f51dee9de4d64b77077a0941db7a8e98b9"
R5_KEY = "government-scope-native-20261008-r5"
R5_DISPATCH_ID = "government-native-scope-r5"
R5_POINTER = "threads[name=Scientist].evidence.governmentScopeNativeGrant"
R5_ENVELOPE_PATH = (r"C:\Users\Consiliari\Documents\Scientist-Probes\native-government-scope-20261008-r5"
                    r"\released-r5-grant.json")
R5_ENVELOPE_SHA256 = "c80edc0e9864c1e332eaee81838dad0f33640aa96b226d30d14228375c493c90"
R5_SOURCE_PATH = (r"C:\Users\Consiliari\Documents\Scientist-Probes\native-government-scope-20261008-r5"
                  r"\coordinator-r5-preflight-snapshot.json")
R5_SOURCE_SHA256 = "881dd9afe3973f00dd4a04cd6d2efaa611b95f2605a952082c25ad32c3a9f5fa"
R5_COORDINATION_PATH = R3_COORDINATION_PATH
R5_ACTIVE_STATUS = "Active; Scientist slot assigned; execute only after offline preflight."
R5_HISTORY_PATH = (Path(__file__).parents[1] / "evidence" / "government-scope-native-20261008-r5" /
                   "historical-native-starts.sqlite").resolve()
R5_HISTORY_SHA256 = "dd617d58a9021fce0b11482b740d78ccb5ddd143ff0c6fa4fbe08983af7f6705"
R5_PRIOR_LABELS = R4_PRIOR_LABELS | {("government", "government-native-serialization-r4/queue")}
R5_LABELS = {"government-native-scope-r5/queue", "government-native-scope-r5/resume"}
R5_CUMULATIVE = {"government": {"starts": 7, "seconds": 1050}, "classic": {"starts": 7, "seconds": 1050}}
R5_SOURCE_SHA = R4_SOURCE_SHA
R5_BINARY_SHA256 = R4_BINARY_SHA256
R5_DELEGATE_SHA256 = R4_DELEGATE_SHA256
R5_PYTHON_PATH = r"C:\Python313\python.exe"
R5_PYTHON_SHA256 = "d87063e5597f257004c731b66c59c56c91038861c6877b1a3dca6b8c4e919125"


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


def validate_r3_grant_binding(request, original_grant_path, original_grant_sha):
    """Validate the additive R3 grant against the exact original source grant and snapshot."""
    if request.get("mode") != "mechanical":
        raise ValueError("provider-free R3 allocation requires mechanical mode")
    if request.get("nativeFixtureCorrection") is not None:
        raise ValueError("R3 Request cannot reuse the closed R2 correction binding")
    original_path = Path(original_grant_path).resolve(strict=True)
    if (original_path != Path(R3_BASE_GRANT_PATH).resolve(strict=True) or
            original_grant_sha != R3_BASE_GRANT_SHA256 or sha(original_path) != R3_BASE_GRANT_SHA256):
        raise ValueError("R3 allocation requires the exact original R1 source grant")
    _released_binding(request, "nativeFixtureGrant", original_path, original_grant_sha, KEY)

    binding = request.get("nativeFixtureR3Grant")
    if (not isinstance(binding, dict) or set(binding) != {"path", "sha256", "sourceKey"} or
            binding.get("sourceKey") != R3_KEY):
        raise ValueError("exact nativeFixtureR3Grant path/SHA/key binding required")
    grant_path = Path(binding["path"])
    if grant_path.is_symlink():
        raise ValueError("R3 grant envelope cannot be a symlink")
    grant_path = grant_path.resolve(strict=True)
    grant_sha = binding["sha256"]
    if (grant_path != Path(R3_ENVELOPE_PATH).resolve(strict=True) or
            grant_sha != R3_ENVELOPE_SHA256 or sha(grant_path) != R3_ENVELOPE_SHA256):
        raise ValueError("R3 grant envelope path or digest differs from the accepted source")
    _released_binding(request, "nativeFixtureR3Grant", grant_path, grant_sha, R3_KEY)

    document = _strict_load(grant_path, "R3 grant envelope")
    if set(document) != {"grant", "sourceCoordinationPath", "sourceCoordinationSha256",
                         "sourceJsonPointer", "sourceThreadId"}:
        raise ValueError("R3 grant envelope shape mismatch")
    source_path = Path(document["sourceCoordinationPath"])
    if source_path.is_symlink():
        raise ValueError("R3 canonical source snapshot cannot be a symlink")
    source_path = source_path.resolve(strict=True)
    source_sha = document["sourceCoordinationSha256"]
    if (source_path != Path(R3_SOURCE_PATH).resolve(strict=True) or source_sha != R3_SOURCE_SHA256 or
            sha(source_path) != R3_SOURCE_SHA256 or document["sourceThreadId"] != R3_SOURCE_THREAD or
            document["sourceJsonPointer"] != R3_POINTER):
        raise ValueError("R3 canonical source path, digest, thread, or pointer mismatch")
    if not _is_released(request, source_path, source_sha):
        raise ValueError("R3 canonical source snapshot must be an exact released Request input")

    source = _strict_load(source_path, "R3 canonical coordination snapshot")
    scientist = next((item for item in source.get("threads", [])
                      if isinstance(item, dict) and item.get("name") == "Scientist"), None)
    grant = document["grant"]
    if (not isinstance(scientist, dict) or
            scientist.get("evidence", {}).get("contractCorrectedNativeIntegrationGrant") != grant or
            grant.get("key") != R3_KEY or grant.get("baseSha") != "f5cd8871b84f45541044110057b59f0bb1ca0804"):
        raise ValueError("R3 grant differs from its canonical Scientist source or frozen base")
    expected_products = {"government": R3_BASE_PRODUCTS["government"],
                         "classic": R3_BASE_PRODUCTS["classic"]}
    original_doc = _strict_load(original_path, "original R1 source grant")
    original_products = {item.get("name", "").lower(): item
                         for item in original_doc.get("grant", {}).get("products", [])
                         if isinstance(item, dict)}
    if original_products != expected_products:
        raise ValueError("R3 original product source/binary pins differ from accepted R1 products")
    expected_values = {
        "government": {"maxNativeStarts": 2, "maxWrapperAttempts": 6,
                       "maxDeterministicDelegates": 6, "maxReservedSessionSeconds": 300},
        "classic": {"maxNativeStarts": 5, "maxWrapperAttempts": 6,
                    "maxDeterministicDelegates": 6, "maxReservedSessionSeconds": 750},
        "historicalConsumed": {"nativeStarts": 5, "wrapperAttempts": 3,
                               "delegates": 0, "reservedSessionSeconds": 750},
        "maxNewNativeStarts": 7, "maxNewReservedSessionSeconds": 1050,
        "cumulativeMaxNativeStarts": 12, "cumulativeMaxReservedSessionSeconds": 1800,
        "maxParallelRoles": 2, "productsSequential": True,
        "nativeProcessDeadlineSeconds": 38,
        "controllerWindowsSeconds": {"government": 38, "classic": 180},
        "reservedSecondsPerNativeStart": 150,
        "newModelProviderCalls": 0, "metadataSessions": 0, "studyCells": 0}
    if any(grant.get(key) != value for key, value in expected_values.items()):
        raise ValueError("R3 grant limits differ from the exact finite allocation")
    product = request.get("arm")
    if product not in {"government", "classic"}:
        raise ValueError("R3 Request must identify its bounded Government or Classic product")
    if request.get("dispatchId") != R3_DISPATCH_IDS[product]:
        raise ValueError("R3 Request dispatch identity differs from the exact product grant")
    product_grant = grant[product]
    return {"path": str(grant_path), "sha256": grant_sha,
            "sourceCoordinationPath": str(source_path),
            "sourceCoordinationSha256": source_sha,
            "grantKey": R3_KEY, "grant": grant,
            "products": expected_products,
            "maxRoleStarts": product_grant["maxWrapperAttempts"],
            "maxDeterministicDelegates": product_grant["maxDeterministicDelegates"],
            "maxNativeStarts": product_grant["maxNativeStarts"],
            "maxReservedSessionSeconds": product_grant["maxReservedSessionSeconds"],
            "perProduct": {name: dict(grant[name]) for name in ("government", "classic")},
            "maxParallelRoles": 2, "productsSequential": True,
            "nativeProcessDeadlineSeconds": 38,
            "controllerWindowsSeconds": {"government": 38, "classic": 180},
            "reservedSecondsPerNativeStart": 150}


def validate_r4_grant_binding(request, original_grant_path, original_grant_sha):
    """Validate the exact additive Government serializer grant without writing a ledger."""
    if not isinstance(request, dict) or request.get("mode") != "mechanical":
        raise ValueError("R4 provider-free allocation requires a mechanical Request")
    if request.get("arm") != "government" or request.get("dispatchId") != R4_DISPATCH_ID:
        raise ValueError("R4 grant is restricted to the exact Government dispatch")
    if any(request.get(field) is not None for field in
           ("nativeFixtureCorrection", "nativeFixtureR3Grant")):
        raise ValueError("R4 Request cannot reuse a closed correction or R3 binding")
    original_path = Path(original_grant_path).resolve(strict=True)
    if (original_path != Path(R3_BASE_GRANT_PATH).resolve(strict=True) or
            original_grant_sha != R3_BASE_GRANT_SHA256 or sha(original_path) != R3_BASE_GRANT_SHA256):
        raise ValueError("R4 allocation requires the exact original R1 source grant")
    _released_binding(request, "nativeFixtureGrant", original_path, original_grant_sha, KEY)

    binding = request.get("nativeFixtureR4Grant")
    if (not isinstance(binding, dict) or set(binding) != {"path", "sha256", "sourceKey"} or
            binding.get("sourceKey") != R4_KEY):
        raise ValueError("exact nativeFixtureR4Grant path/SHA/key binding required")
    raw_path = Path(binding["path"])
    if raw_path.is_symlink():
        raise ValueError("R4 grant envelope cannot be a symlink")
    grant_path = raw_path.resolve(strict=True)
    grant_sha = binding["sha256"]
    if (grant_path != Path(R4_ENVELOPE_PATH).resolve(strict=True) or
            grant_sha != R4_ENVELOPE_SHA256 or sha(grant_path) != R4_ENVELOPE_SHA256):
        raise ValueError("R4 grant envelope path or digest differs from the accepted source")
    _released_binding(request, "nativeFixtureR4Grant", grant_path, grant_sha, R4_KEY)

    document = _strict_load(grant_path, "R4 grant envelope")
    if (not isinstance(document, dict) or set(document) !=
            {"grant", "sourceCoordinationPath", "sourceCoordinationSha256", "sourceJsonPointer", "sourceThreadId"}):
        raise ValueError("R4 grant envelope shape mismatch")
    source_path_raw = Path(document["sourceCoordinationPath"])
    if source_path_raw.is_symlink():
        raise ValueError("R4 canonical source snapshot cannot be a symlink")
    source_path = source_path_raw.resolve(strict=True)
    source_sha = document["sourceCoordinationSha256"]
    if (source_path != Path(R4_SOURCE_PATH).resolve(strict=True) or source_sha != R4_SOURCE_SHA256 or
            sha(source_path) != R4_SOURCE_SHA256 or document["sourceThreadId"] != R4_SOURCE_THREAD or
            document["sourceJsonPointer"] != R4_POINTER):
        raise ValueError("R4 canonical source path, digest, thread, or pointer mismatch")
    if not _is_released(request, source_path, source_sha):
        raise ValueError("R4 canonical source snapshot must be an exact released Request input")

    source = _strict_load(source_path, "R4 canonical coordination snapshot")
    scientist = next((item for item in source.get("threads", [])
                      if isinstance(item, dict) and item.get("name") == "Scientist"), None)
    grant = document["grant"]
    if (not isinstance(scientist, dict) or
            scientist.get("evidence", {}).get("governmentSerializationNativeGrant") != grant or
            grant.get("key") != R4_KEY or grant.get("baseSha") != "a0bf4e48230b22fe7e1d5f1b80c7b56a0ffb6c76"):
        raise ValueError("R4 grant differs from its canonical Scientist source or frozen base")

    original_doc = _strict_load(original_path, "original R1 source grant")
    original_products = {item.get("name", "").lower(): item
                         for item in original_doc.get("grant", {}).get("products", [])
                         if isinstance(item, dict)}
    expected_product = {"name": "Government", "sourceSha": R4_SOURCE_SHA,
                        "binarySha256": R4_BINARY_SHA256}
    if original_products.get("government") != expected_product:
        raise ValueError("R4 Government source/binary pins differ from the accepted R1 product")
    expected_product_grant = {**expected_product, "delegateSha256": R4_DELEGATE_SHA256}
    if grant.get("product") != expected_product_grant:
        raise ValueError("R4 product/delegate pin differs from the exact accepted Government candidate")

    executable = request.get("product", {}).get("government", {}).get("executable")
    if (not isinstance(executable, dict) or executable.get("sha256") != R4_BINARY_SHA256 or
            executable.get("sourceCommit") != R4_SOURCE_SHA or not isinstance(executable.get("path"), str)):
        raise ValueError("R4 Request must bind the exact Government executable source and digest")
    executable_path = Path(executable["path"])
    if (executable_path.is_symlink() or not executable_path.is_file() or
            sha(executable_path) != R4_BINARY_SHA256):
        raise ValueError("R4 Request Government executable differs from the exact binary pin")
    expected_values = {
        "maxFreshCases": 1, "maxNativeStarts": 2, "maxWrapperAttempts": 6,
        "maxDeterministicDelegates": 6, "maxParallelRoles": 2,
        "reservedSecondsPerNativeStart": 150, "maxNewReservedSessionSeconds": 300,
        "nativeProcessDeadlineSeconds": 38, "controllerWindowSeconds": 38,
        "historicalConsumed": {"nativeStarts": 11, "wrapperAttempts": 7,
                               "delegates": 4, "reservedSessionSeconds": 1650},
        "cumulativeMaxNativeStarts": 13, "cumulativeMaxWrapperAttempts": 13,
        "cumulativeMaxDelegates": 10, "cumulativeMaxReservedSessionSeconds": 1950,
        "newModelProviderCalls": 0, "metadataSessions": 0, "studyCells": 0,
        "classicNativeStarts": 0, "fullProductSuites": 0,
        "sequence": ["one fresh Queue",
                     "only after complete positive Queue with current checks, independent technical review and all required final assents: its associated existing Resume/Replay verification"]}
    if any(grant.get(key) != value for key, value in expected_values.items()):
        raise ValueError("R4 grant limits or sequencing differ from the exact finite Government allocation")
    return {"path": str(grant_path), "sha256": grant_sha,
            "sourceCoordinationPath": str(source_path),
            "sourceCoordinationSha256": source_sha,
            "grantKey": R4_KEY, "grant": grant, "product": expected_product_grant,
            "maxNativeStarts": grant["maxNativeStarts"],
            "maxWrapperAttempts": grant["maxWrapperAttempts"],
            "maxDeterministicDelegates": grant["maxDeterministicDelegates"],
            "maxParallelRoles": grant["maxParallelRoles"],
            "maxRoleStarts": grant["maxWrapperAttempts"],
            "maxRoleParallel": grant["maxParallelRoles"],
            "maxRoleProcessSeconds": grant["nativeProcessDeadlineSeconds"],
            "reservedSecondsPerNativeStart": grant["reservedSecondsPerNativeStart"],
            "maxNewReservedSessionSeconds": grant["maxNewReservedSessionSeconds"],
            "nativeProcessDeadlineSeconds": grant["nativeProcessDeadlineSeconds"],
            "controllerWindowSeconds": grant["controllerWindowSeconds"],
            "cumulativeMaxNativeStarts": grant["cumulativeMaxNativeStarts"],
            "cumulativeMaxWrapperAttempts": grant["cumulativeMaxWrapperAttempts"],
            "cumulativeMaxDelegates": grant["cumulativeMaxDelegates"],
            "cumulativeMaxReservedSessionSeconds": grant["cumulativeMaxReservedSessionSeconds"]}


def validate_r4_entry_gate(validated_grant):
    """Require the live explicit Scientist R4 assignment immediately before consumption."""
    if (not isinstance(validated_grant, dict) or validated_grant.get("grantKey") != R4_KEY or
            validated_grant.get("grant", {}).get("key") != R4_KEY or
            validated_grant.get("sourceCoordinationPath") != R4_SOURCE_PATH or
            validated_grant.get("sourceCoordinationSha256") != R4_SOURCE_SHA256):
        raise ValueError("validated R4 grant provenance required for the entry gate")
    live_path = Path(R4_COORDINATION_PATH)
    if live_path.is_symlink():
        raise ValueError("live R4 coordination state cannot be a symlink")
    live_path = live_path.resolve(strict=True)
    raw = live_path.read_bytes()
    source = _strict_load(raw, "live R4 coordination state")
    scientist = next((item for item in source.get("threads", [])
                      if isinstance(item, dict) and item.get("name") == "Scientist"), None)
    live_grant = scientist.get("evidence", {}).get("governmentSerializationNativeGrant") if scientist else None
    frozen_source = _strict_load(validated_grant["sourceCoordinationPath"], "frozen R4 coordination source")
    frozen_scientist = next((item for item in frozen_source.get("threads", [])
                             if isinstance(item, dict) and item.get("name") == "Scientist"), None)
    frozen_grant = (frozen_scientist.get("evidence", {}).get("governmentSerializationNativeGrant")
                    if frozen_scientist else None)
    if not isinstance(live_grant, dict) or not isinstance(frozen_grant, dict):
        raise ValueError("live Scientist R4 grant is missing")
    slot = source.get("fullSuiteSlot")
    if not isinstance(slot, dict) or slot.get("owner") != "Scientist":
        raise ValueError("R4 fullSuiteSlot is not explicitly assigned to Scientist")
    slot_keys = [(key, slot[key]) for key in ("assignmentKey", "key", "grantKey") if key in slot]
    if len(slot_keys) != 1 or slot_keys[0][1] != R4_KEY:
        raise ValueError("R4 fullSuiteSlot must carry exactly one matching explicit grant key")
    if live_grant.get("status") != R4_ACTIVE_STATUS:
        raise ValueError("live Scientist R4 grant status is not the exact active assignment status")
    if not isinstance(live_grant.get("sentUtc"), str) or not live_grant["sentUtc"]:
        raise ValueError("live Scientist R4 grant sentUtc must remain an explicit timestamp")
    slot_assigned_utc = slot.get("assignedUtc")
    if (not isinstance(slot_assigned_utc, str) or not slot_assigned_utc or
            not isinstance(live_grant.get("slotAssignedUtc"), str) or
            live_grant["slotAssignedUtc"] != slot_assigned_utc):
        raise ValueError("R4 grant slotAssignedUtc must exactly match the live fullSuiteSlot assignment time")
    mutable = {"status", "sentUtc", "slotAssignedUtc"}
    if ({key: value for key, value in live_grant.items() if key not in mutable} !=
            {key: value for key, value in frozen_grant.items() if key not in mutable}):
        raise ValueError("live Scientist R4 grant differs from the frozen grant outside activation fields")
    return {"coordinationPath": str(live_path), "coordinationSha256": hashlib.sha256(raw).hexdigest(),
            "slotOwner": "Scientist", "slotKey": slot_keys[0][1], "grantKey": R4_KEY}



def validate_r5_grant_binding(request, original_grant_path, original_grant_sha):
    """Validate the exact additive R5 Government scope grant without writing a ledger."""
    if not isinstance(request, dict) or request.get("mode") != "mechanical":
        raise ValueError("R5 provider-free allocation requires a mechanical Request")
    if request.get("arm") != "government" or request.get("dispatchId") != R5_DISPATCH_ID:
        raise ValueError("R5 grant is restricted to the exact Government dispatch")
    if any(request.get(field) is not None for field in
           ("nativeFixtureCorrection", "nativeFixtureR3Grant", "nativeFixtureR4Grant")):
        raise ValueError("R5 Request cannot reuse a closed correction, R3, or R4 binding")
    original_path = Path(original_grant_path).resolve(strict=True)
    if (original_path != Path(R3_BASE_GRANT_PATH).resolve(strict=True) or
            original_grant_sha != R3_BASE_GRANT_SHA256 or sha(original_path) != R3_BASE_GRANT_SHA256):
        raise ValueError("R5 allocation requires the exact original R1 source grant")
    _released_binding(request, "nativeFixtureGrant", original_path, original_grant_sha, KEY)

    binding = request.get("nativeFixtureR5Grant")
    if (not isinstance(binding, dict) or set(binding) != {"path", "sha256", "sourceKey"} or
            binding.get("sourceKey") != R5_KEY):
        raise ValueError("exact nativeFixtureR5Grant path/SHA/key binding required")
    raw_path = Path(binding["path"])
    if raw_path.is_symlink():
        raise ValueError("R5 grant envelope cannot be a symlink")
    grant_path = raw_path.resolve(strict=True)
    grant_sha = binding["sha256"]
    if (grant_path != Path(R5_ENVELOPE_PATH).resolve(strict=True) or
            grant_sha != R5_ENVELOPE_SHA256 or sha(grant_path) != R5_ENVELOPE_SHA256):
        raise ValueError("R5 grant envelope path or digest differs from the accepted source")
    _released_binding(request, "nativeFixtureR5Grant", grant_path, grant_sha, R5_KEY)

    document = _strict_load(grant_path, "R5 grant envelope")
    if (not isinstance(document, dict) or set(document) !=
            {"grant", "sourceCoordinationPath", "sourceCoordinationSha256", "sourceJsonPointer", "sourceThreadId"}):
        raise ValueError("R5 grant envelope shape mismatch")
    source_path_raw = Path(document["sourceCoordinationPath"])
    if source_path_raw.is_symlink():
        raise ValueError("R5 canonical source snapshot cannot be a symlink")
    source_path = source_path_raw.resolve(strict=True)
    source_sha = document["sourceCoordinationSha256"]
    if (source_path != Path(R5_SOURCE_PATH).resolve(strict=True) or source_sha != R5_SOURCE_SHA256 or
            sha(source_path) != R5_SOURCE_SHA256 or document["sourceThreadId"] != SOURCE_THREAD or
            document["sourceJsonPointer"] != R5_POINTER):
        raise ValueError("R5 canonical source path, digest, thread, or pointer mismatch")
    if not _is_released(request, source_path, source_sha):
        raise ValueError("R5 canonical source snapshot must be an exact released Request input")

    source = _strict_load(source_path, "R5 canonical coordination snapshot")
    scientist = next((item for item in source.get("threads", [])
                      if isinstance(item, dict) and item.get("name") == "Scientist"), None)
    grant = document["grant"]
    if (not isinstance(scientist, dict) or
            scientist.get("evidence", {}).get("governmentScopeNativeGrant") != grant or
            grant.get("key") != R5_KEY or grant.get("baseSha") != "1d3f125d374822755936b039ff14b0fdd69e9f5f"):
        raise ValueError("R5 grant differs from its canonical Scientist source or frozen base")

    original_doc = _strict_load(original_path, "original R1 source grant")
    original_products = {item.get("name", "").lower(): item
                         for item in original_doc.get("grant", {}).get("products", [])
                         if isinstance(item, dict)}
    expected_product = {"name": "Government", "sourceSha": R5_SOURCE_SHA,
                        "binarySha256": R5_BINARY_SHA256}
    if original_products.get("government") != expected_product:
        raise ValueError("R5 Government source/binary pins differ from the accepted R1 product")
    expected_product_grant = {**expected_product, "delegateSha256": R5_DELEGATE_SHA256}
    if grant.get("product") != expected_product_grant:
        raise ValueError("R5 product/delegate pin differs from the exact accepted Government candidate")

    executable = request.get("product", {}).get("government", {}).get("executable")
    if (not isinstance(executable, dict) or executable.get("sha256") != R5_BINARY_SHA256 or
            executable.get("sourceCommit") != R5_SOURCE_SHA or not isinstance(executable.get("path"), str)):
        raise ValueError("R5 Request must bind the exact Government executable source and digest")
    executable_path = Path(executable["path"])
    if (executable_path.is_symlink() or not executable_path.is_file() or
            sha(executable_path) != R5_BINARY_SHA256):
        raise ValueError("R5 Request Government executable differs from the exact binary pin")
    python_pin = grant.get("python")
    python_path = Path(R5_PYTHON_PATH)
    if (python_pin != {"path": "C:/Python313/python.exe", "sha256": R5_PYTHON_SHA256} or
            not python_path.is_file() or python_path.is_symlink() or sha(python_path) != R5_PYTHON_SHA256):
        raise ValueError("R5 Python executable differs from the exact Python313 pin")

    expected_values = {
        "maxFreshCases": 1, "maxNativeStarts": 2, "maxWrapperAttempts": 6,
        "maxDeterministicDelegates": 6, "maxParallelRoles": 2,
        "reservedSecondsPerNativeStart": 150, "maxNewReservedSessionSeconds": 300,
        "nativeProcessDeadlineSeconds": 38, "controllerWindowSeconds": 38,
        "historicalConsumed": {"nativeStarts": 12, "wrapperAttempts": 10,
                               "delegates": 7, "reservedSessionSeconds": 1800},
        "cumulativeMaxNativeStarts": 14, "cumulativeMaxWrapperAttempts": 16,
        "cumulativeMaxDelegates": 13, "cumulativeMaxReservedSessionSeconds": 2100,
        "newModelProviderCalls": 0, "metadataSessions": 0, "studyCells": 0,
        "classicNativeStarts": 0, "fullProductSuites": 0,
        "historicalRealUsage": {"actorStarts": 5, "knownTokenSubtotal": 53331, "totalTokens": None},
        "sequence": ["one fresh Queue",
                     "only after complete positive native AND outer result, fresh actual checks, independent technical review, all required final assents and promotion: associated existing Resume/Replay verification"]}
    if any(grant.get(key) != value for key, value in expected_values.items()):
        raise ValueError("R5 grant limits or sequencing differ from the exact finite Government allocation")
    return {"path": str(grant_path), "sha256": grant_sha,
            "sourceCoordinationPath": str(source_path), "sourceCoordinationSha256": source_sha,
            "grantKey": R5_KEY, "grant": grant, "product": expected_product_grant,
            "maxNativeStarts": grant["maxNativeStarts"],
            "maxWrapperAttempts": grant["maxWrapperAttempts"],
            "maxDeterministicDelegates": grant["maxDeterministicDelegates"],
            "maxParallelRoles": grant["maxParallelRoles"],
            "maxRoleStarts": grant["maxWrapperAttempts"], "maxRoleParallel": grant["maxParallelRoles"],
            "maxRoleProcessSeconds": grant["nativeProcessDeadlineSeconds"],
            "reservedSecondsPerNativeStart": grant["reservedSecondsPerNativeStart"],
            "maxNewReservedSessionSeconds": grant["maxNewReservedSessionSeconds"],
            "nativeProcessDeadlineSeconds": grant["nativeProcessDeadlineSeconds"],
            "controllerWindowSeconds": grant["controllerWindowSeconds"],
            "cumulativeMaxNativeStarts": grant["cumulativeMaxNativeStarts"],
            "cumulativeMaxWrapperAttempts": grant["cumulativeMaxWrapperAttempts"],
            "cumulativeMaxDelegates": grant["cumulativeMaxDelegates"],
            "cumulativeMaxReservedSessionSeconds": grant["cumulativeMaxReservedSessionSeconds"]}


def validate_r5_entry_gate(validated_grant):
    """Require the exact live Scientist R5 assignment before each consumption."""
    if (not isinstance(validated_grant, dict) or validated_grant.get("grantKey") != R5_KEY or
            validated_grant.get("grant", {}).get("key") != R5_KEY or
            validated_grant.get("sourceCoordinationPath") != R5_SOURCE_PATH or
            validated_grant.get("sourceCoordinationSha256") != R5_SOURCE_SHA256):
        raise ValueError("validated R5 grant provenance required for the entry gate")
    live_path = Path(R5_COORDINATION_PATH)
    if live_path.is_symlink():
        raise ValueError("live R5 coordination state cannot be a symlink")
    live_path = live_path.resolve(strict=True)
    raw = live_path.read_bytes()
    source = _strict_load(raw, "live R5 coordination state")
    scientist = next((item for item in source.get("threads", [])
                      if isinstance(item, dict) and item.get("name") == "Scientist"), None)
    live_grant = scientist.get("evidence", {}).get("governmentScopeNativeGrant") if scientist else None
    frozen_source = _strict_load(validated_grant["sourceCoordinationPath"], "frozen R5 coordination source")
    frozen_scientist = next((item for item in frozen_source.get("threads", [])
                             if isinstance(item, dict) and item.get("name") == "Scientist"), None)
    frozen_grant = (frozen_scientist.get("evidence", {}).get("governmentScopeNativeGrant")
                    if frozen_scientist else None)
    if not isinstance(live_grant, dict) or not isinstance(frozen_grant, dict):
        raise ValueError("live Scientist R5 grant is missing")
    if live_grant != frozen_grant or live_grant != validated_grant["grant"]:
        raise ValueError("live Scientist R5 grant differs from the exact frozen active grant")
    slot = source.get("fullSuiteSlot")
    frozen_slot = frozen_source.get("fullSuiteSlot")
    if not isinstance(slot, dict) or slot.get("owner") != "Scientist":
        raise ValueError("R5 fullSuiteSlot is not explicitly assigned to Scientist")
    slot_keys = [(key, slot[key]) for key in ("assignmentKey", "key", "grantKey") if key in slot]
    if len(slot_keys) != 1 or slot_keys[0][1] != R5_KEY:
        raise ValueError("R5 fullSuiteSlot must carry exactly one matching explicit grant key")
    if slot != frozen_slot:
        raise ValueError("live R5 fullSuiteSlot differs from the exact frozen Scientist assignment")
    if (slot.get("assignedUtc") != "2026-10-08T02:00:58Z" or
            live_grant.get("slotAssignedUtc") != slot.get("assignedUtc")):
        raise ValueError("R5 grant slotAssignedUtc must match the exact live Scientist slot assignment")
    return {"coordinationPath": str(live_path), "coordinationSha256": hashlib.sha256(raw).hexdigest(),
            "slotOwner": "Scientist", "slotKey": slot_keys[0][1], "grantKey": R5_KEY}


def validate_r3_entry_gate(validated_grant):
    """Require a live, exact R3 full-suite assignment before any R3 consumption."""
    if (not isinstance(validated_grant, dict) or validated_grant.get("grantKey") != R3_KEY or
            validated_grant.get("grant", {}).get("key") != R3_KEY or
            validated_grant.get("sourceCoordinationPath") != R3_SOURCE_PATH or
            validated_grant.get("sourceCoordinationSha256") != R3_SOURCE_SHA256):
        raise ValueError("validated R3 grant provenance required for the entry gate")
    live_path = Path(R3_COORDINATION_PATH)
    if live_path.is_symlink():
        raise ValueError("live R3 coordination state cannot be a symlink")
    live_path = live_path.resolve(strict=True)
    raw = live_path.read_bytes()
    source = _strict_load(raw, "live R3 coordination state")
    scientist = next((item for item in source.get("threads", [])
                      if isinstance(item, dict) and item.get("name") == "Scientist"), None)
    live_grant = scientist.get("evidence", {}).get("contractCorrectedNativeIntegrationGrant") if scientist else None
    frozen_source = _strict_load(validated_grant["sourceCoordinationPath"], "frozen R3 coordination source")
    frozen_scientist = next((item for item in frozen_source.get("threads", [])
                             if isinstance(item, dict) and item.get("name") == "Scientist"), None)
    frozen_grant = (frozen_scientist.get("evidence", {}).get("contractCorrectedNativeIntegrationGrant")
                    if frozen_scientist else None)
    if not isinstance(live_grant, dict) or not isinstance(frozen_grant, dict):
        raise ValueError("live Scientist R3 grant is missing")
    slot = source.get("fullSuiteSlot")
    if not isinstance(slot, dict) or slot.get("owner") != "Scientist":
        raise ValueError("R3 fullSuiteSlot is not explicitly assigned to Scientist")
    slot_keys = [(key, slot[key]) for key in ("assignmentKey", "key", "grantKey") if key in slot]
    if len(slot_keys) != 1 or slot_keys[0][1] != R3_KEY:
        raise ValueError("R3 fullSuiteSlot must carry exactly one matching explicit grant key")
    if live_grant.get("status") != R3_ACTIVE_STATUS:
        raise ValueError("live Scientist R3 grant status is not the exact active assignment status")
    if not isinstance(live_grant.get("sentUtc"), str) or not live_grant["sentUtc"]:
        raise ValueError("live Scientist R3 grant sentUtc must remain an explicit timestamp")
    slot_assigned_utc = slot.get("assignedUtc")
    grant_assigned_utc = live_grant.get("slotAssignedUtc")
    if (not isinstance(slot_assigned_utc, str) or not slot_assigned_utc or
            not isinstance(grant_assigned_utc, str) or not grant_assigned_utc or
            grant_assigned_utc != slot_assigned_utc):
        raise ValueError("R3 grant slotAssignedUtc must exactly match the live fullSuiteSlot assignment time")
    mutable_keys = {"status", "sentUtc", "slotAssignedUtc"}
    if ({key: value for key, value in live_grant.items() if key not in mutable_keys} !=
            {key: value for key, value in frozen_grant.items() if key not in mutable_keys}):
        raise ValueError("live Scientist R3 grant differs from the frozen grant outside authorized activation fields")
    if live_grant.get("key") != R3_KEY:
        raise ValueError("live Scientist R3 grant key mismatch")
    return {"coordinationPath": str(live_path),
            "coordinationSha256": hashlib.sha256(raw).hexdigest(),
            "slotOwner": "Scientist", "slotKey": slot_keys[0][1], "grantKey": R3_KEY}


@contextmanager
def _readonly_db(path):
    target = Path(path).resolve(strict=True)
    db = sqlite3.connect(target.as_uri() + "?mode=ro", uri=True, timeout=10)
    try:
        db.execute("PRAGMA query_only=ON")
        yield db
    finally:
        db.close()


def _history_rows(path):
    with _readonly_db(path) as db:
        allocation = db.execute("SELECT grant_sha FROM allocation ORDER BY grant_sha").fetchall()
        starts = db.execute("SELECT product,label,argv,claimed,finished,reserved_seconds,receipt "
                            "FROM starts ORDER BY product,label").fetchall()
        corrections = db.execute("SELECT source_key,correction_sha,source_snapshot_sha,basis_sha "
                                 "FROM corrections ORDER BY source_key").fetchall()
        return allocation, starts, corrections


def _validate_r4_fresh_check_receipts(expected_checks, observed_checks):
    """Match configured lowercase definitions to Go encoding/json GateResult fields."""
    if not isinstance(expected_checks, list) or not expected_checks or not isinstance(observed_checks, list):
        raise ValueError("R4 accepted run report lacks fresh configured checks")
    expected = {(item.get("name"), item.get("tool")) for item in expected_checks
                if isinstance(item, dict)}
    observed = [(item.get("Name"), item.get("Tool")) for item in observed_checks
                if isinstance(item, dict)]
    if (not expected or len(expected) != len(expected_checks) or
            len(observed) != len(observed_checks) or len(observed) != len(expected) or
            set(observed) != expected or len(set(observed)) != len(observed)):
        raise ValueError("R4 fresh checks differ from the exact configured check names/tools")
    for check in observed_checks:
        if (type(check.get("ExitCode")) is not int or check["ExitCode"] != 0 or
                type(check.get("Milliseconds")) is not int or check["Milliseconds"] < 0 or
                type(check.get("TimeoutMilliseconds")) is not int or
                check["TimeoutMilliseconds"] <= 0 or
                check["Milliseconds"] > check["TimeoutMilliseconds"]):
            raise ValueError("R4 fresh check name/exit/time receipt is not a passing bounded check")
    return True


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
        self.r3 = None
        self.r4 = None
        self.r5 = None
        if (request is not None and request.get("dispatchId") in set(R3_DISPATCH_IDS.values()) and
                request.get("nativeFixtureR3Grant") is None):
            raise ValueError("exact R3 fixture grant is required for the corrected R3 dispatch")
        if request is not None and request.get("dispatchId") == R4_DISPATCH_ID and request.get("nativeFixtureR4Grant") is None:
            raise ValueError("exact R4 fixture grant is required for the Government serialization R4 dispatch")
        if request is not None and request.get("dispatchId") == R5_DISPATCH_ID and request.get("nativeFixtureR5Grant") is None:
            raise ValueError("exact R5 fixture grant is required for the Government scope R5 dispatch")
        if request is not None and request.get("nativeFixtureCorrection") is not None:
            self.correction = self._validate_correction(request)
        if request is not None and request.get("nativeFixtureR4Grant") is not None:
            self.r4 = validate_r4_grant_binding(request, self.grant_path, self.grant_sha)
            if self.correction is not None or request.get("nativeFixtureR3Grant") is not None:
                raise ValueError("R4 Request cannot combine closed correction grants")
            self.r4_request = request
            if not self.path.is_file():
                raise ValueError("R4 requires the existing immutable native-start history")
            self._validate_r4_prior_history_readonly()
            return
        if request is not None and request.get("nativeFixtureR5Grant") is not None:
            self.r5 = validate_r5_grant_binding(request, self.grant_path, self.grant_sha)
            if (self.correction is not None or request.get("nativeFixtureR3Grant") is not None or
                    request.get("nativeFixtureR4Grant") is not None):
                raise ValueError("R5 Request cannot combine closed correction grants")
            self.r5_request = request
            if not self.path.is_file():
                raise ValueError("R5 requires the existing immutable native-start history")
            self._validate_r5_prior_history_readonly()
            return
        if request is not None and request.get("nativeFixtureR3Grant") is not None:
            self.r3 = validate_r3_grant_binding(request, self.grant_path, self.grant_sha)
            if self.correction is not None:
                raise ValueError("R3 Request cannot combine the closed R2 correction and R3 grant")
            self.r3_request = request
            if not self.path.is_file():
                raise ValueError("R3 requires the existing immutable native-start history")
            self._validate_r3_prior_history_readonly()
            return
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

    @staticmethod
    def _r3_identity(validated_grant):
        history_basis = hashlib.sha256(json.dumps(
            validated_grant["grant"]["historicalConsumed"], sort_keys=True,
            separators=(",", ":")).encode("utf-8")).hexdigest()
        return (R3_KEY, validated_grant["sha256"],
                validated_grant["sourceCoordinationSha256"], history_basis)

    def _validate_r3_rows(self, allocation, starts, corrections):
        if allocation != [(self.grant_sha,)]:
            raise ValueError("R3 must append to the original R1 native-start allocation")
        old_allocation, old_starts, old_corrections = _history_rows(R3_HISTORY_PATH)
        if hashlib.sha256(R3_HISTORY_PATH.read_bytes()).hexdigest() != R3_HISTORY_SHA256:
            raise ValueError("immutable R2 native-start snapshot digest mismatch")
        old_labels = {(product, label) for product, labels in R3_PRIOR_LABELS.items() for label in labels}
        old_rows = [row for row in starts if (row[0], row[1]) in old_labels]
        expected_old_rows = [row for row in old_starts if (row[0], row[1]) in old_labels]
        if old_allocation != [(R3_BASE_GRANT_SHA256,)] or len(expected_old_rows) != 5 or old_rows != expected_old_rows:
            raise ValueError("R3 historical rows differ from the immutable five-row R2 snapshot")
        prior_corrections = [row for row in corrections if row[0] != R3_KEY]
        if prior_corrections != old_corrections or len(old_corrections) != 1 or old_corrections[0][0] != CORRECTION_KEY:
            raise ValueError("R3 must preserve the exact existing R2 correction record")
        new_rows = [row for row in starts if (row[0], row[1]) not in old_labels]
        if any(product not in R3_LABELS or label not in R3_LABELS[product]
               for product, label, _argv, _claimed, _finished, _seconds, _receipt in new_rows):
            raise ValueError("R3 history contains an unallocated native start label")
        identity = self._r3_identity(self.r3)
        r3_corrections = [row for row in corrections if row[0] == R3_KEY]
        if r3_corrections and r3_corrections != [identity]:
            raise ValueError("R3 history is bound to a different source grant or snapshot")
        for product, limits in R3_CUMULATIVE.items():
            product_rows = [row for row in starts if row[0] == product]
            new_product_rows = [row for row in new_rows if row[0] == product]
            prior_count = len([row for row in expected_old_rows if row[0] == product])
            if (prior_count != (3 if product == "government" else 2) or
                    len(product_rows) > limits["starts"] or
                    len(new_product_rows) > limits["starts"] - prior_count or
                    sum(row[5] for row in product_rows) > limits["seconds"] or
                    any(row[5] != PROCESS_SECONDS_RESERVED for row in product_rows)):
                raise ValueError("R3 history exceeds its cumulative native start/session ceiling")
        ordered_products = []
        new_rows_by_claim = sorted(new_rows, key=lambda row: row[3])
        for product, _label, _argv, _claimed, _finished, _seconds, _receipt in new_rows_by_claim:
            if not ordered_products or ordered_products[-1] != product:
                ordered_products.append(product)
        if len(ordered_products) > 1 and ordered_products != ["government", "classic"]:
            raise ValueError("R3 products must run sequentially in Government then Classic order")
        if len([row for row in new_rows if row[4] is None]) > 1:
            raise ValueError("R3 history exceeds maxParallel=1 native controller allocation")

    @staticmethod
    def _r5_identity(validated_grant):
        return (R5_KEY, validated_grant["sha256"], validated_grant["sourceCoordinationSha256"],
                validated_grant["grant"]["baseSha"])

    @staticmethod
    def _r4_identity(validated_grant):
        history_basis = hashlib.sha256(json.dumps(
            validated_grant["grant"]["historicalConsumed"], sort_keys=True,
            separators=(",", ":")).encode("utf-8")).hexdigest()
        return (R4_KEY, validated_grant["sha256"],
                validated_grant["sourceCoordinationSha256"], history_basis)

    def _r4_validate_queue_success(self, queue_row):
        """Re-run the product translator against exact bound Request/stdout before Resume."""
        receipt_raw = queue_row[6].encode("utf-8") if isinstance(queue_row[6], str) else b"{}"
        receipt = _strict_load(receipt_raw, "R4 Queue process receipt")
        if (not isinstance(receipt, dict) or type(receipt.get("returnCode")) is not int or
                receipt.get("returnCode") != 0):
            raise ValueError("R4 resume requires a successful queue process")
        government_product = self.r4_request.get("product", {}).get("government", {})
        role_binding = government_product.get("roleAuthorization")
        if (not isinstance(role_binding, dict) or set(role_binding) != {"path", "sha256"} or
                not isinstance(role_binding.get("path"), str)):
            raise ValueError("R4 queue requires the exact Request roleAuthorization binding")
        auth_path = Path(role_binding["path"])
        if auth_path.is_symlink() or sha(auth_path) != role_binding["sha256"]:
            raise ValueError("R4 queue roleAuthorization path/digest changed")
        auth = _strict_load(auth_path, "R4 roleAuthorization")
        request_path = auth.get("requestPath") if isinstance(auth, dict) else None
        if not isinstance(request_path, str):
            raise ValueError("R4 roleAuthorization does not bind the exact Request path")
        request_raw = Path(request_path).read_bytes()
        if _strict_load(request_raw, "R4 Request") != self.r4_request:
            raise ValueError("R4 exact Request bytes differ from the admitted Request")
        evidence = Path(self.r4_request.get("evidenceDirectory", "")).resolve(strict=True)
        stdout_path = evidence / "process" / "stdout.log"
        try:
            from government import translate_queue_result
            translated = translate_queue_result(
                self.r4_request, request_raw, stdout_path, receipt["returnCode"])
        except (OSError, ValueError, KeyError) as exc:
            raise ValueError(f"R4 Queue native result is not positively translatable: {exc}") from exc
        if (translated.get("status") != "completed" or
                translated.get("government", {}).get("queueStatus") != "complete"):
            raise ValueError("R4 Queue translator did not report completed/complete")
        jobs = translated.get("government", {}).get("nativeJobStates", [])
        if (len(jobs) != 1 or jobs[0].get("id") != self.r4_request.get("task", {}).get("id") or
                jobs[0].get("state") not in {"accepted-scoped", "accepted-complete"}):
            raise ValueError("R4 Queue translator did not bind one accepted Request job")
        reports = [item for item in translated.get("receipts", [])
                   if item.get("kind") == "government-run-report"]
        if len(reports) != 1:
            raise ValueError("R4 Queue requires one translated native run report")
        report_path = Path(reports[0]["path"]).resolve(strict=True)
        report_raw = report_path.read_bytes()
        if hashlib.sha256(report_raw).hexdigest() != reports[0].get("sha256"):
            raise ValueError("R4 translated run report digest changed")
        report = _strict_load(report_raw, "R4 native run report")
        promotion = report.get("promotion") if isinstance(report, dict) else None
        if (not isinstance(report, dict) or report.get("status") != "accepted-scoped" or
                report.get("stage") != "complete" or
                not isinstance(promotion, dict) or promotion.get("status") != "promoted"):
            raise ValueError("R4 run report must be accepted-scoped/complete with promoted status")
        completion_receipts = [item for item in translated.get("receipts", [])
                               if item.get("kind") == "government-promotion-completion"]
        if len(completion_receipts) != 1:
            raise ValueError("R4 promoted run must bind its promotion-completion receipt")
        runtime_binding = government_product.get("runtime", {})
        if (not isinstance(runtime_binding, dict) or not isinstance(runtime_binding.get("path"), str) or
                sha(runtime_binding["path"]) != runtime_binding.get("sha256")):
            raise ValueError("R4 fresh-check runtime binding is missing or changed")
        runtime = _strict_load(runtime_binding["path"], "R4 native runtime")
        expected_checks = runtime.get("checks") if isinstance(runtime, dict) else None
        observed_checks = report.get("checks")
        _validate_r4_fresh_check_receipts(expected_checks, observed_checks)
        kinds = {item.get("kind") for item in translated.get("receipts", [])}
        if "government-decision" not in kinds or not any(
                isinstance(kind, str) and kind.startswith("government-vote:") for kind in kinds):
            raise ValueError("R4 accepted queue lacks explicit validated final vote/decision receipts")
        return translated

    def _validate_r4_rows(self, allocation, starts, corrections):
        if allocation != [(self.grant_sha,)]:
            raise ValueError("R4 must append to the original R1 native-start allocation")
        if hashlib.sha256(R4_HISTORY_PATH.read_bytes()).hexdigest() != R4_HISTORY_SHA256:
            raise ValueError("immutable R4 eleven-start history snapshot digest mismatch")
        old_allocation, old_starts, old_corrections = _history_rows(R4_HISTORY_PATH)
        old_rows = [row for row in starts if (row[0], row[1]) in R4_PRIOR_LABELS]
        expected_old_rows = [row for row in old_starts if (row[0], row[1]) in R4_PRIOR_LABELS]
        if (old_allocation != [(R3_BASE_GRANT_SHA256,)] or len(expected_old_rows) != 11 or
                old_rows != expected_old_rows):
            raise ValueError("R4 historical starts differ from the immutable eleven-row snapshot")
        prior_corrections = [row for row in corrections if row[0] != R4_KEY]
        if prior_corrections != old_corrections:
            raise ValueError("R4 must preserve all existing R2/R3 correction records exactly")
        new_rows = [row for row in starts if (row[0], row[1]) not in R4_PRIOR_LABELS]
        if any(product != "government" or label not in R4_LABELS for
               product, label, _argv, _claimed, _finished, _seconds, _receipt in new_rows):
            raise ValueError("R4 history contains an unallocated non-Government start")
        identity = self._r4_identity(self.r4)
        r4_corrections = [row for row in corrections if row[0] == R4_KEY]
        if r4_corrections and r4_corrections != [identity]:
            raise ValueError("R4 history is bound to a different source grant or snapshot")
        if (len(new_rows) > 2 or sum(row[5] for row in new_rows) > 300 or
                any(row[5] != PROCESS_SECONDS_RESERVED for row in new_rows)):
            raise ValueError("R4 history exceeds its two-start/300-second finite allocation")
        total_rows = [*expected_old_rows, *new_rows]
        if (len(total_rows) > 13 or sum(row[5] for row in total_rows) > 1950 or
                sum(row[0] == "government" for row in total_rows) > 6 or
                sum(row[5] for row in total_rows if row[0] == "government") > 900 or
                sum(row[0] == "classic" for row in total_rows) > 7 or
                sum(row[5] for row in total_rows if row[0] == "classic") > 1050):
            raise ValueError("R4 history exceeds cumulative native-start/session ceilings")
        new_ordered = sorted(new_rows, key=lambda row: row[3])
        if [row[1] for row in new_ordered] not in ([], ["government-native-serialization-r4/queue"],
                                                   ["government-native-serialization-r4/queue",
                                                    "government-native-serialization-r4/resume"]):
            raise ValueError("R4 must reserve its queue before its single associated resume")
        if len(new_ordered) == 2:
            queue_receipt = json.loads(new_ordered[0][6] or "{}")
            if (new_ordered[0][4] is None or type(queue_receipt.get("returnCode")) is not int or
                    queue_receipt.get("returnCode") != 0 or new_ordered[1][3] < new_ordered[0][4]):
                raise ValueError("R4 resume requires a completed successful queue process")
            self._r4_validate_queue_success(new_ordered[0])
        if len([row for row in new_rows if row[4] is None]) > 1:
            raise ValueError("R4 native controller parallelism is exhausted")

    def _validate_r4_prior_history_readonly(self):
        if hashlib.sha256(R4_HISTORY_PATH.read_bytes()).hexdigest() != R4_HISTORY_SHA256:
            raise ValueError("immutable R4 eleven-start history snapshot digest mismatch")
        with _readonly_db(self.path) as db:
            allocation = db.execute("SELECT grant_sha FROM allocation ORDER BY grant_sha").fetchall()
            starts = db.execute("SELECT product,label,argv,claimed,finished,reserved_seconds,receipt "
                                "FROM starts ORDER BY product,label").fetchall()
            corrections = db.execute("SELECT source_key,correction_sha,source_snapshot_sha,basis_sha "
                                     "FROM corrections ORDER BY source_key").fetchall()
        self._validate_r4_rows(allocation, starts, corrections)

    def _reserve_r4(self, product, label, argv):
        if product != "government" or label not in R4_LABELS:
            raise ValueError("native start label is not allocated by the exact R4 Government grant")
        db = self._connection()
        try:
            db.execute("BEGIN IMMEDIATE")
            allocation = db.execute("SELECT grant_sha FROM allocation ORDER BY grant_sha").fetchall()
            starts = db.execute("SELECT product,label,argv,claimed,finished,reserved_seconds,receipt "
                                "FROM starts ORDER BY product,label").fetchall()
            corrections = db.execute("SELECT source_key,correction_sha,source_snapshot_sha,basis_sha "
                                     "FROM corrections ORDER BY source_key").fetchall()
            self._validate_r4_rows(allocation, starts, corrections)
            if db.execute("SELECT 1 FROM starts WHERE product=? AND label=?", (product, label)).fetchone():
                raise ValueError("R4 native start already claimed; no retry")
            new_rows = [row for row in starts if (row[0], row[1]) not in R4_PRIOR_LABELS]
            expected_next = ("government-native-serialization-r4/queue" if not new_rows else
                             "government-native-serialization-r4/resume")
            if label != expected_next:
                raise ValueError("R4 permits one queue followed only by its associated resume")
            if label == "government-native-serialization-r4/resume":
                queue_row = next((row for row in new_rows
                                  if row[1] == "government-native-serialization-r4/queue"), None)
                if queue_row is None or queue_row[4] is None:
                    raise ValueError("R4 resume requires a completed successful queue process")
                self._r4_validate_queue_success(queue_row)
            count, seconds = db.execute("SELECT COUNT(*),COALESCE(SUM(reserved_seconds),0) "
                                        "FROM starts WHERE product='government'").fetchone()
            if count >= R4_CUMULATIVE["government"]["starts"] or seconds + PROCESS_SECONDS_RESERVED > R4_CUMULATIVE["government"]["seconds"]:
                raise ValueError("finite R4 Government native allocation exhausted")
            if db.execute("SELECT COUNT(*) FROM starts WHERE finished IS NULL").fetchone()[0] >= 1:
                raise ValueError("R4 native controller parallelism exhausted")
            identity = self._r4_identity(self.r4)
            existing = db.execute("SELECT source_key,correction_sha,source_snapshot_sha,basis_sha "
                                  "FROM corrections WHERE source_key=?", (R4_KEY,)).fetchone()
            if existing is None:
                db.execute("INSERT INTO corrections VALUES(?,?,?,?)", identity)
            elif tuple(existing) != identity:
                raise ValueError("R4 ledger record differs from its validated source grant")
            db.execute("INSERT INTO starts VALUES(?,?,?,?,NULL,?,NULL)",
                       (product, label, json.dumps(argv), time.time(), PROCESS_SECONDS_RESERVED))
            db.commit()
        except BaseException:
            db.rollback()
            raise
        finally:
            db.close()

    def _r5_validate_queue_success(self, queue_row):
        """Re-run the product translator against exact bound Request/stdout before Resume."""
        receipt_raw = queue_row[6].encode("utf-8") if isinstance(queue_row[6], str) else b"{}"
        receipt = _strict_load(receipt_raw, "R5 Queue process receipt")
        if (not isinstance(receipt, dict) or type(receipt.get("returnCode")) is not int or
                receipt.get("returnCode") != 0):
            raise ValueError("R5 resume requires a successful queue process")
        government_product = self.r5_request.get("product", {}).get("government", {})
        role_binding = government_product.get("roleAuthorization")
        if (not isinstance(role_binding, dict) or set(role_binding) != {"path", "sha256"} or
                not isinstance(role_binding.get("path"), str)):
            raise ValueError("R5 queue requires the exact Request roleAuthorization binding")
        auth_path = Path(role_binding["path"])
        if auth_path.is_symlink() or sha(auth_path) != role_binding["sha256"]:
            raise ValueError("R5 queue roleAuthorization path/digest changed")
        auth = _strict_load(auth_path, "R5 roleAuthorization")
        request_path = auth.get("requestPath") if isinstance(auth, dict) else None
        if not isinstance(request_path, str):
            raise ValueError("R5 roleAuthorization does not bind the exact Request path")
        request_raw = Path(request_path).read_bytes()
        if _strict_load(request_raw, "R5 Request") != self.r5_request:
            raise ValueError("R5 exact Request bytes differ from the admitted Request")
        evidence = Path(self.r5_request.get("evidenceDirectory", "")).resolve(strict=True)
        stdout_path = evidence / "process" / "stdout.log"
        try:
            from government import translate_queue_result
            translated = translate_queue_result(
                self.r5_request, request_raw, stdout_path, receipt["returnCode"])
        except (OSError, ValueError, KeyError) as exc:
            raise ValueError(f"R5 Queue native result is not positively translatable: {exc}") from exc
        if (translated.get("status") != "completed" or
                translated.get("government", {}).get("queueStatus") != "complete"):
            raise ValueError("R5 Queue translator did not report completed/complete")
        jobs = translated.get("government", {}).get("nativeJobStates", [])
        if (len(jobs) != 1 or jobs[0].get("id") != self.r5_request.get("task", {}).get("id") or
                jobs[0].get("state") not in {"accepted-scoped", "accepted-complete"}):
            raise ValueError("R5 Queue translator did not bind one accepted Request job")
        reports = [item for item in translated.get("receipts", [])
                   if item.get("kind") == "government-run-report"]
        if len(reports) != 1:
            raise ValueError("R5 Queue requires one translated native run report")
        report_path = Path(reports[0]["path"]).resolve(strict=True)
        report_raw = report_path.read_bytes()
        if hashlib.sha256(report_raw).hexdigest() != reports[0].get("sha256"):
            raise ValueError("R5 translated run report digest changed")
        report = _strict_load(report_raw, "R5 native run report")
        promotion = report.get("promotion") if isinstance(report, dict) else None
        if (not isinstance(report, dict) or report.get("status") != "accepted-scoped" or
                report.get("stage") != "complete" or
                not isinstance(promotion, dict) or promotion.get("status") != "promoted"):
            raise ValueError("R5 run report must be accepted-scoped/complete with promoted status")
        completion_receipts = [item for item in translated.get("receipts", [])
                               if item.get("kind") == "government-promotion-completion"]
        if len(completion_receipts) != 1:
            raise ValueError("R5 promoted run must bind its promotion-completion receipt")
        runtime_binding = government_product.get("runtime", {})
        if (not isinstance(runtime_binding, dict) or not isinstance(runtime_binding.get("path"), str) or
                sha(runtime_binding["path"]) != runtime_binding.get("sha256")):
            raise ValueError("R5 fresh-check runtime binding is missing or changed")
        runtime = _strict_load(runtime_binding["path"], "R5 native runtime")
        expected_checks = runtime.get("checks") if isinstance(runtime, dict) else None
        observed_checks = report.get("checks")
        _validate_r4_fresh_check_receipts(expected_checks, observed_checks)
        kinds = {item.get("kind") for item in translated.get("receipts", [])}
        if "government-decision" not in kinds or not any(
                isinstance(kind, str) and kind.startswith("government-vote:") for kind in kinds):
            raise ValueError("R5 accepted queue lacks explicit validated final vote/decision receipts")
        return translated

    def _validate_r5_rows(self, allocation, starts, corrections):
        if allocation != [(self.grant_sha,)]:
            raise ValueError("R5 must append to the original R1 native-start allocation")
        if hashlib.sha256(R5_HISTORY_PATH.read_bytes()).hexdigest() != R5_HISTORY_SHA256:
            raise ValueError("immutable R5 twelve-start history snapshot digest mismatch")
        old_allocation, old_starts, old_corrections = _history_rows(R5_HISTORY_PATH)
        old_rows = [row for row in starts if (row[0], row[1]) in R5_PRIOR_LABELS]
        expected_old_rows = [row for row in old_starts if (row[0], row[1]) in R5_PRIOR_LABELS]
        if (old_allocation != [(R3_BASE_GRANT_SHA256,)] or len(expected_old_rows) != 12 or
                old_rows != expected_old_rows):
            raise ValueError("R5 historical starts differ from the immutable twelve-row snapshot")
        prior_corrections = [row for row in corrections if row[0] != R5_KEY]
        if prior_corrections != old_corrections:
            raise ValueError("R5 must preserve all existing R2/R3 correction records exactly")
        new_rows = [row for row in starts if (row[0], row[1]) not in R5_PRIOR_LABELS]
        if any(product != "government" or label not in R5_LABELS for
               product, label, _argv, _claimed, _finished, _seconds, _receipt in new_rows):
            raise ValueError("R5 history contains an unallocated non-Government start")
        identity = self._r5_identity(self.r5)
        r5_corrections = [row for row in corrections if row[0] == R5_KEY]
        if r5_corrections and r5_corrections != [identity]:
            raise ValueError("R5 history is bound to a different source grant or snapshot")
        if (len(new_rows) > 2 or sum(row[5] for row in new_rows) > 300 or
                any(row[5] != PROCESS_SECONDS_RESERVED for row in new_rows)):
            raise ValueError("R5 history exceeds its two-start/300-second finite allocation")
        total_rows = [*expected_old_rows, *new_rows]
        if (len(total_rows) > 14 or sum(row[5] for row in total_rows) > 2100 or
                sum(row[0] == "government" for row in total_rows) > 7 or
                sum(row[5] for row in total_rows if row[0] == "government") > 1050 or
                sum(row[0] == "classic" for row in total_rows) > 7 or
                sum(row[5] for row in total_rows if row[0] == "classic") > 1050):
            raise ValueError("R5 history exceeds cumulative native-start/session ceilings")
        new_ordered = sorted(new_rows, key=lambda row: row[3])
        if [row[1] for row in new_ordered] not in ([], ["government-native-scope-r5/queue"],
                                                   ["government-native-scope-r5/queue",
                                                    "government-native-scope-r5/resume"]):
            raise ValueError("R5 must reserve its queue before its single associated resume")
        if len(new_ordered) == 2:
            queue_receipt = json.loads(new_ordered[0][6] or "{}")
            if (new_ordered[0][4] is None or type(queue_receipt.get("returnCode")) is not int or
                    queue_receipt.get("returnCode") != 0 or new_ordered[1][3] < new_ordered[0][4]):
                raise ValueError("R5 resume requires a completed successful queue process")
            self._r5_validate_queue_success(new_ordered[0])
        if len([row for row in new_rows if row[4] is None]) > 1:
            raise ValueError("R5 native controller parallelism is exhausted")

    def _validate_r5_prior_history_readonly(self):
        if hashlib.sha256(R5_HISTORY_PATH.read_bytes()).hexdigest() != R5_HISTORY_SHA256:
            raise ValueError("immutable R5 twelve-start history snapshot digest mismatch")
        with _readonly_db(self.path) as db:
            allocation = db.execute("SELECT grant_sha FROM allocation ORDER BY grant_sha").fetchall()
            starts = db.execute("SELECT product,label,argv,claimed,finished,reserved_seconds,receipt "
                                "FROM starts ORDER BY product,label").fetchall()
            corrections = db.execute("SELECT source_key,correction_sha,source_snapshot_sha,basis_sha "
                                     "FROM corrections ORDER BY source_key").fetchall()
        self._validate_r5_rows(allocation, starts, corrections)

    def _reserve_r5(self, product, label, argv):
        if product != "government" or label not in R5_LABELS:
            raise ValueError("native start label is not allocated by the exact R5 Government grant")
        db = self._connection()
        try:
            db.execute("BEGIN IMMEDIATE")
            allocation = db.execute("SELECT grant_sha FROM allocation ORDER BY grant_sha").fetchall()
            starts = db.execute("SELECT product,label,argv,claimed,finished,reserved_seconds,receipt "
                                "FROM starts ORDER BY product,label").fetchall()
            corrections = db.execute("SELECT source_key,correction_sha,source_snapshot_sha,basis_sha "
                                     "FROM corrections ORDER BY source_key").fetchall()
            self._validate_r5_rows(allocation, starts, corrections)
            if db.execute("SELECT 1 FROM starts WHERE product=? AND label=?", (product, label)).fetchone():
                raise ValueError("R5 native start already claimed; no retry")
            new_rows = [row for row in starts if (row[0], row[1]) not in R5_PRIOR_LABELS]
            expected_next = ("government-native-scope-r5/queue" if not new_rows else
                             "government-native-scope-r5/resume")
            if label != expected_next:
                raise ValueError("R5 permits one queue followed only by its associated resume")
            if label == "government-native-scope-r5/resume":
                queue_row = next((row for row in new_rows
                                  if row[1] == "government-native-scope-r5/queue"), None)
                if queue_row is None or queue_row[4] is None:
                    raise ValueError("R5 resume requires a completed successful queue process")
                self._r5_validate_queue_success(queue_row)
            count, seconds = db.execute("SELECT COUNT(*),COALESCE(SUM(reserved_seconds),0) "
                                        "FROM starts WHERE product='government'").fetchone()
            if count >= R5_CUMULATIVE["government"]["starts"] or seconds + PROCESS_SECONDS_RESERVED > R5_CUMULATIVE["government"]["seconds"]:
                raise ValueError("finite R5 Government native allocation exhausted")
            if db.execute("SELECT COUNT(*) FROM starts WHERE finished IS NULL").fetchone()[0] >= 1:
                raise ValueError("R5 native controller parallelism exhausted")
            identity = self._r5_identity(self.r5)
            existing = db.execute("SELECT source_key,correction_sha,source_snapshot_sha,basis_sha "
                                  "FROM corrections WHERE source_key=?", (R5_KEY,)).fetchone()
            if existing is None:
                db.execute("INSERT INTO corrections VALUES(?,?,?,?)", identity)
            elif tuple(existing) != identity:
                raise ValueError("R5 ledger record differs from its validated source grant")
            db.execute("INSERT INTO starts VALUES(?,?,?,?,NULL,?,NULL)",
                       (product, label, json.dumps(argv), time.time(), PROCESS_SECONDS_RESERVED))
            db.commit()
        except BaseException:
            db.rollback()
            raise
        finally:
            db.close()

    def _validate_r3_prior_history_readonly(self):
        if hashlib.sha256(R3_HISTORY_PATH.read_bytes()).hexdigest() != R3_HISTORY_SHA256:
            raise ValueError("immutable R2 native-start snapshot digest mismatch")
        with _readonly_db(self.path) as db:
            allocation = db.execute("SELECT grant_sha FROM allocation ORDER BY grant_sha").fetchall()
            starts = db.execute("SELECT product,label,argv,claimed,finished,reserved_seconds,receipt "
                                "FROM starts ORDER BY product,label").fetchall()
            corrections = db.execute("SELECT source_key,correction_sha,source_snapshot_sha,basis_sha "
                                     "FROM corrections ORDER BY source_key").fetchall()
        self._validate_r3_rows(allocation, starts, corrections)

    def preflight_snapshot(self):
        """Read-only allocation/history view; reports slot readiness without reserving."""
        if self.r5 is not None:
            with _readonly_db(self.path) as db:
                db.row_factory = sqlite3.Row
                rows = [dict(row) for row in db.execute("SELECT * FROM starts ORDER BY claimed")]
            try:
                gate = {"ready": True, **validate_r5_entry_gate(self.r5)}
            except (OSError, ValueError) as exc:
                gate = {"ready": False, "reason": str(exc)}
            return {"allocationId": R5_KEY, "grantSha256": self.r5["sha256"],
                    "sourceCoordinationSha256": self.r5["sourceCoordinationSha256"],
                    "cumulativeNativeStartCeiling": {key: value["starts"] for key, value in R5_CUMULATIVE.items()},
                    "cumulativeReservedSessionSecondsCeiling": {key: value["seconds"] for key, value in R5_CUMULATIVE.items()},
                    "maxParallel": 1, "productsRunSequentially": True, "starts": rows,
                    "entryGate": gate, "deadlineSeconds": 38,
                    "reservedSecondsPerStart": PROCESS_SECONDS_RESERVED,
                    "accountingUnit": "bounded Government controller and deterministic role sessions including cleanup margin",
                    "arbitraryOsDescendantWallSeconds": None,
                    "refills": 0, "realActorStarts": 0, "providerCalls": 0, "studyCells": 0}
        if self.r4 is not None:
            with _readonly_db(self.path) as db:
                db.row_factory = sqlite3.Row
                rows = [dict(row) for row in db.execute("SELECT * FROM starts ORDER BY claimed")]
            try:
                gate = {"ready": True, **validate_r4_entry_gate(self.r4)}
            except (OSError, ValueError) as exc:
                gate = {"ready": False, "reason": str(exc)}
            return {"allocationId": R4_KEY, "grantSha256": self.r4["sha256"],
                    "sourceCoordinationSha256": self.r4["sourceCoordinationSha256"],
                    "cumulativeNativeStartCeiling": {key: value["starts"] for key, value in R4_CUMULATIVE.items()},
                    "cumulativeReservedSessionSecondsCeiling": {key: value["seconds"] for key, value in R4_CUMULATIVE.items()},
                    "maxParallel": 1, "productsRunSequentially": True,
                    "starts": rows, "entryGate": gate, "deadlineSeconds": 38,
                    "reservedSecondsPerStart": PROCESS_SECONDS_RESERVED,
                    "accountingUnit": "bounded Government controller and deterministic role sessions including cleanup margin",
                    "arbitraryOsDescendantWallSeconds": None,
                    "refills": 0, "realActorStarts": 0, "providerCalls": 0, "studyCells": 0}
        if self.r3 is None:
            return self.snapshot()
        with _readonly_db(self.path) as db:
            db.row_factory = sqlite3.Row
            rows = [dict(row) for row in db.execute("SELECT * FROM starts ORDER BY claimed")]
        try:
            gate = {"ready": True, **validate_r3_entry_gate(self.r3)}
        except (OSError, ValueError) as exc:
            gate = {"ready": False, "reason": str(exc)}
        return {"allocationId": R3_KEY, "grantSha256": self.r3["sha256"],
                "sourceCoordinationSha256": self.r3["sourceCoordinationSha256"],
                "cumulativeNativeStartCeiling": {key: value["starts"] for key, value in R3_CUMULATIVE.items()},
                "cumulativeReservedSessionSecondsCeiling": {key: value["seconds"] for key, value in R3_CUMULATIVE.items()},
                "maxParallel": 1, "productsRunSequentially": True,
                "starts": rows, "entryGate": gate, "deadlineSeconds": 38,
                "reservedSecondsPerStart": PROCESS_SECONDS_RESERVED,
                "accountingUnit": "bounded controller and deterministic role sessions including cleanup margin",
                "arbitraryOsDescendantWallSeconds": None,
                "refills": 0, "realActorStarts": 0, "providerCalls": 0, "studyCells": 0}

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
        if self.r5 is not None:
            self.r5 = validate_r5_grant_binding(self.r5_request, self.grant_path, self.grant_sha)
            if (sha(self.r5["path"]) != self.r5["sha256"] or
                    sha(self.r5["sourceCoordinationPath"]) != self.r5["sourceCoordinationSha256"]):
                raise ValueError("R5 grant or frozen source changed before start")
            validate_r5_entry_gate(self.r5)
        if self.r4 is not None:
            self.r4 = validate_r4_grant_binding(self.r4_request, self.grant_path, self.grant_sha)
            if (sha(self.r4["path"]) != self.r4["sha256"] or
                    sha(self.r4["sourceCoordinationPath"]) != self.r4["sourceCoordinationSha256"]):
                raise ValueError("R4 grant or frozen source changed before start")
            # Check the live assignment before opening or mutating the start ledger.
            validate_r4_entry_gate(self.r4)
        if self.r3 is not None:
            if (sha(self.grant_path) != R3_BASE_GRANT_SHA256 or
                    sha(self.r3["path"]) != self.r3["sha256"] or
                    sha(self.r3["sourceCoordinationPath"]) != self.r3["sourceCoordinationSha256"]):
                raise ValueError("R3 grant or frozen source changed before start")
            # The live gate precedes opening or mutating the native-start database.
            validate_r3_entry_gate(self.r3)
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
        if self.r5 is not None:
            self._reserve_r5(product, label, argv)
            return
        if self.r4 is not None:
            self._reserve_r4(product, label, argv)
            return
        db = self._connection()
        try:
            db.execute("BEGIN IMMEDIATE")
            if db.execute("SELECT 1 FROM starts WHERE product=? AND label=?", (product, label)).fetchone():
                raise ValueError("native start already claimed; no retry")
            if self.r3 is not None:
                allocation_rows = db.execute("SELECT grant_sha FROM allocation ORDER BY grant_sha").fetchall()
                start_rows = db.execute("SELECT product,label,argv,claimed,finished,reserved_seconds,receipt "
                                        "FROM starts ORDER BY product,label").fetchall()
                correction_rows = db.execute("SELECT source_key,correction_sha,source_snapshot_sha,basis_sha "
                                             "FROM corrections ORDER BY source_key").fetchall()
                self._validate_r3_rows(allocation_rows, start_rows, correction_rows)
                if product not in R3_LABELS or label not in R3_LABELS[product]:
                    raise ValueError("native start label is not allocated by the exact R3 product grant")
                limits = R3_CUMULATIVE[product]
                count, seconds = db.execute("SELECT COUNT(*),COALESCE(SUM(reserved_seconds),0) "
                                            "FROM starts WHERE product=?", (product,)).fetchone()
                if count >= limits["starts"] or seconds + PROCESS_SECONDS_RESERVED > limits["seconds"]:
                    raise ValueError("finite R3 native fixture allocation exhausted")
                if db.execute("SELECT COUNT(*) FROM starts WHERE finished IS NULL").fetchone()[0] >= 1:
                    raise ValueError("R3 native controller parallelism exhausted")
                history = db.execute("SELECT product,label FROM starts ORDER BY claimed").fetchall()
                prior = {(item, prior_label) for item, labels in R3_PRIOR_LABELS.items()
                         for prior_label in labels}
                r3_products = [item for item, prior_label in history if (item, prior_label) not in prior]
                product_order = []
                for item in r3_products:
                    if not product_order or product_order[-1] != item:
                        product_order.append(item)
                if ((not product_order and product != "government") or
                        (product == "government" and "classic" in product_order) or
                        (product == "classic" and product_order not in (["government"],
                                                                        ["government", "classic"]))):
                    raise ValueError("R3 products must run sequentially in Government then Classic order")
                identity = self._r3_identity(self.r3)
                row = db.execute("SELECT source_key,correction_sha,source_snapshot_sha,basis_sha "
                                 "FROM corrections WHERE source_key=?", (R3_KEY,)).fetchone()
                if row is None:
                    db.execute("INSERT INTO corrections VALUES(?,?,?,?)", identity)
                elif tuple(row) != identity:
                    raise ValueError("R3 correction record differs from the validated source grant")
                db.execute("INSERT INTO starts VALUES(?,?,?,?,NULL,?,NULL)",
                           (product, label, json.dumps(argv), time.time(), PROCESS_SECONDS_RESERVED))
                db.commit()
                return
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
        if self.r5 is not None:
            return self.preflight_snapshot()
        if self.r4 is not None:
            return self.preflight_snapshot()
        if self.r3 is not None:
            return self.preflight_snapshot()
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
