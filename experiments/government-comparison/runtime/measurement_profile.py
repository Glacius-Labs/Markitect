"""Two explicit study measurement versions; never alter numeric trial limits."""
import hashlib
import json
from pathlib import Path

LEGACY = "strict-provider-usage-v1"
OBSERVED = "observed-native-usage-v2"
FILES = {LEGACY: "measurement-profile-v1.json", OBSERVED: "measurement-profile-v2.json"}


def profile(profile_id):
    if profile_id not in FILES:
        raise ValueError("unknown measurement profile")
    return json.loads(Path(__file__).parents[1].joinpath("public", FILES[profile_id]).read_bytes())


def limits_sha(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def profile_sha(profile_id):
    return limits_sha(profile(profile_id))
