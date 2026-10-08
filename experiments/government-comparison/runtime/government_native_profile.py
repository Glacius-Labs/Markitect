"""Closed identities for the historical R5 case and its sole R6 successor."""
from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True)
class Profile:
    name: str
    key: str
    dispatch_id: str
    marker: str
    metadata_field: str
    pointer: str
    external_root: Path
    evidence_directory: Path
    envelope_sha: str
    snapshot_sha: str
    base_sha: str
    assigned_utc: str
    task_id: str

    @property
    def envelope_path(self):
        return self.external_root / f"released-{self.name}{'-a1' if self.name == 'r6' else ''}-grant.json"

    @property
    def snapshot_path(self):
        return self.external_root / f"coordinator-{self.name}{'-a1' if self.name == 'r6' else ''}-preflight-snapshot.json"

    @property
    def history_path(self):
        return self.evidence_directory / "historical-native-starts.sqlite"


_EVIDENCE = Path(__file__).parents[1] / "evidence"
R5 = Profile(
    "r5", "government-scope-native-20261008-r5", "government-native-scope-r5",
    "nativeFixtureR5Grant", "fixtureR5SourceGrant",
    "threads[name=Scientist].evidence.governmentScopeNativeGrant",
    Path(r"C:\Users\Consiliari\Documents\Scientist-Probes\native-government-scope-20261008-r5"),
    _EVIDENCE / "government-scope-native-20261008-r5",
    "c80edc0e9864c1e332eaee81838dad0f33640aa96b226d30d14228375c493c90",
    "881dd9afe3973f00dd4a04cd6d2efaa611b95f2605a952082c25ad32c3a9f5fa",
    "1d3f125d374822755936b039ff14b0fdd69e9f5f", "2026-10-08T02:00:58Z",
    "positive-overflow-release")
R6 = Profile(
    "r6", "government-released-binding-native-20261008-r6",
    "government-native-released-binding-r6", "nativeFixtureR6Grant", "fixtureR6SourceGrant",
    "threads[name=Scientist].evidence.governmentReleasedBindingNativeGrant",
    Path(r"C:\Users\Consiliari\Documents\Scientist-Probes\native-government-released-binding-20261008-r6"),
    _EVIDENCE / "government-released-binding-native-20261008-r6",
    "fc677770bdc15b3c9e69b8906b974297baad5180f391be44eabee2b53a5ac55d",
    "9db0292ff9e7bec4c3605ab2594dd9a6d2bec4a12e5fa4cad5a6a89edcc7782d",
    "165f35b3d2bd70fbcc49d54f52d479b88d54a6b7", "2026-10-08T02:27:56Z",
    "positive-overflow-release-r6")


def profile(name):
    if name == "r5":
        return R5
    if name == "r6":
        return R6
    raise ValueError("only the fixed R5 and R6 Government profiles exist")


def request_profile(request):
    selected = [item for item in (R5, R6) if request.get(item.marker) is not None]
    expected = next((item for item in (R5, R6)
                     if request.get("dispatchId") == item.dispatch_id), None)
    if not selected and expected is None:
        return None
    if (len(selected) != 1 or selected[0] != expected or request.get("arm") != "government" or
            any(request.get(key) is not None for key in
                ("nativeFixtureCorrection", "nativeFixtureR3Grant", "nativeFixtureR4Grant"))):
        raise ValueError("exact separate R5 or R6 Government Request profile required")
    return selected[0]
