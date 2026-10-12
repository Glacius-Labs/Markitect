"""Run manifest (schema 1): one JSON file describes everything a run needs.

`load(path)` is the only validator; other modules trust its result. The case must be
one of the folders `cases.discover()` finds under the playground root (inside the run
container that is /in). `stations` defaults to the case's count. `markitect.sourceRepo`
defaults to the Git checkout that holds the playground, if it is a Markitect checkout;
the host then records it as an absolute path with the full commit
(`host.resolve_markitect`). `agent.model` and `markitect.innerModel` are full model ids,
never aliases (`registration.alias_reason`).
"""

from __future__ import annotations

import json
import re
from pathlib import Path, PurePosixPath
from typing import Any

from . import cases, registration

SCHEMA = 1
METHODS = ("conventional", "markitect")
# "fake" stands in for Codex, "fake-claude" for Claude Code (both provider-free).
AGENT_KINDS = ("codex", "fake", "claude", "fake-claude")
CLAUDE_KINDS = ("claude", "fake-claude")
ROLE_NAMES = ("manager", "reviewer", "verifier")
CODEX_EFFORTS = ("minimal", "low", "medium", "high", "xhigh", "max", "ultra")
# The image always ships both CLIs; Codex manifests may leave the Claude version out.
DEFAULT_CLAUDE_VERSION = "2.1.296"
CONTAINER_DEFAULTS = {"cpus": 4, "memory": "8g", "pidsLimit": 2048}
MARKITECT_MODULE = "github.com/Glacius-Labs/Markitect"  # what makes a checkout the default sourceRepo

_ID = re.compile(r"[a-z0-9][a-z0-9-]{0,62}")
_VERSION = re.compile(r"[0-9][0-9A-Za-z.-]{0,55}")  # both versions together form the Docker tag
_TOKEN = re.compile(r"[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}")
_EFFORT = re.compile(r"[a-z]{1,32}")
_MEMORY = re.compile(r"[1-9][0-9]*[bkmg]?")
_COMMIT = re.compile(r"[0-9a-f]{7,40}")


class ManifestError(ValueError):
    """The manifest is missing, unreadable or does not match schema 1."""


def load(path: Path, *, playground: Path | None = None) -> dict:
    path = Path(path)
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError) as exc:
        raise ManifestError(f"cannot read manifest {path}: {exc}") from exc
    except json.JSONDecodeError as exc:
        raise ManifestError(f"manifest {path} is not valid JSON: {exc}") from exc
    result = validate(data, playground=playground)
    if "markitect" in result:  # a relative or ~ sourceRepo means: from the manifest's folder
        source = Path(result["markitect"]["sourceRepo"]).expanduser()
        result["markitect"]["sourceRepo"] = str(source if source.is_absolute() else path.parent / source)
    return result


def validate(data: Any, *, playground: Path | None = None) -> dict:
    """Return a normalized copy of a schema-1 manifest (defaults filled in).

    `playground` is the folder holding `cases/` (default: this playground)."""
    root = _object(data, "manifest", required={"schema", "id", "case", "method", "agent", "limits"},
                   optional={"stations", "container", "markitect"})
    if type(root["schema"]) is not int or root["schema"] != SCHEMA:
        raise ManifestError(f"schema: expected {SCHEMA}, got {root['schema']!r}")
    try:
        known = cases.discover(playground)
    except cases.CaseError as exc:
        raise ManifestError(f"case: {exc}") from exc
    case = _choice(root["case"], tuple(known), "case")
    count = known[case].stations
    # Waves are cumulative, so a run can only take the first N of them.
    stations = _positive_int(root["stations"], "stations") if "stations" in root else count
    if stations > count:
        raise ManifestError(f"stations: case {case} has {count} stations, got {stations}")
    result: dict[str, Any] = {
        "schema": SCHEMA,
        "id": _match(root["id"], _ID, "id"),
        "case": case,
        "stations": stations,
        "method": _choice(root["method"], METHODS, "method"),
    }

    # Claude Code as the outer agent still needs Codex in the image (Markitect's inner
    # roles), so both versions are explicit; for Codex the Claude version is optional.
    agent = _object(root["agent"], "agent", required={"kind", "codexVersion", "model", "effort", "maxSubagents"},
                    optional={"claudeVersion"})
    kind = _choice(agent["kind"], AGENT_KINDS, "agent.kind")
    if kind in CLAUDE_KINDS and "claudeVersion" not in agent:
        raise ManifestError(f"agent: missing claudeVersion (required for kind {kind!r})")
    result["agent"] = {
        "kind": kind,
        "codexVersion": _match(agent["codexVersion"], _VERSION, "agent.codexVersion"),
        "claudeVersion": _match(agent.get("claudeVersion", DEFAULT_CLAUDE_VERSION), _VERSION,
                                "agent.claudeVersion"),
        "model": _model(agent["model"], "agent.model"),
        "effort": _match(agent["effort"], _EFFORT, "agent.effort"),
        # Codex applies it per session; Claude Code has no such setting, so it is only recorded.
        "maxSubagents": _positive_int(agent["maxSubagents"], "agent.maxSubagents"),
    }
    if kind in ("codex", "fake"):
        _choice(result["agent"]["effort"], CODEX_EFFORTS, "agent.effort")

    limits = _object(root["limits"], "limits", required={"stationSeconds", "totalSeconds"})
    result["limits"] = {name: _positive_int(limits[name], f"limits.{name}")
                        for name in ("stationSeconds", "totalSeconds")}

    container = _object(root.get("container", {}), "container", optional=set(CONTAINER_DEFAULTS))
    merged = {**CONTAINER_DEFAULTS, **container}
    result["container"] = {
        "cpus": _positive_number(merged["cpus"], "container.cpus"),
        "memory": _match(merged["memory"], _MEMORY, "container.memory"),
        "pidsLimit": _positive_int(merged["pidsLimit"], "container.pidsLimit"),
    }

    if result["method"] == "markitect":
        if "markitect" not in root:
            raise ManifestError("markitect: required when method is 'markitect'")
        # Markitect's inner roles run on Codex. A Codex outer agent shares its model and effort
        # with them; a Claude Code outer agent needs them named here (product setup input).
        product = _object(root["markitect"], "markitect", required={"commit"},
                          optional={"sourceRepo", "innerModel", "innerEffort", "roles"})
        source = product["sourceRepo"] if "sourceRepo" in product else default_source_repo(playground)
        if not isinstance(source, str) or not source.strip():
            raise ManifestError("markitect.sourceRepo: expected a non-empty path string")
        result["markitect"] = {"sourceRepo": source,
                               "commit": _match(product["commit"], _COMMIT, "markitect.commit")}
        missing = sorted({"innerModel", "innerEffort"} - product.keys())
        if "roles" in product:
            roles = _object(product["roles"], "markitect.roles", optional=set(ROLE_NAMES))
            if not roles:
                raise ManifestError("markitect.roles: expected at least one role profile")
            result["markitect"]["roles"] = {name: _role_profile(roles[name], name)
                                           for name in ROLE_NAMES if name in roles}
        complete_roles = set(result["markitect"].get("roles", {})) == set(ROLE_NAMES)
        if result["agent"]["kind"] in CLAUDE_KINDS and missing and not complete_roles:
            raise ManifestError(f"markitect: missing {', '.join(missing)} (Markitect's inner roles run on "
                                "Codex, not on the Claude Code outer agent's model)")
        if "innerModel" in product:
            result["markitect"]["innerModel"] = _model(product["innerModel"], "markitect.innerModel")
        if "innerEffort" in product:
            result["markitect"]["innerEffort"] = _match(product["innerEffort"], _EFFORT, "markitect.innerEffort")
    elif "markitect" in root:
        raise ManifestError("markitect: only allowed when method is 'markitect'")
    if result["method"] == "markitect":
        for name, profile in role_profiles(result).items():
            if profile["provider"] == "codex":
                _choice(profile["effort"], CODEX_EFFORTS, f"markitect {name} Codex effort")
    return result


def _role_profile(value: Any, name: str) -> dict:
    """The existing product config RoleProfile, with explicit selection per role.

    Executable paths refer to the Linux run container, never the host filesystem.
    The product remains responsible for executable discovery and fingerprinting.
    """
    label = f"markitect.roles.{name}"
    profile = _object(value, label, required={"provider", "model", "effort"},
                      optional={"providerExecutable", "providerArgs", "providerVersion", "costMode"})
    provider = _choice(profile["provider"], ("codex", "process"), f"{label}.provider")
    result = {"provider": provider, "model": _model(profile["model"], f"{label}.model"),
              "effort": _match(profile["effort"], _EFFORT, f"{label}.effort"),
              "costMode": _choice(profile.get("costMode", "unmetered" if provider == "process" else "metered"),
                                  ("metered", "unmetered"), f"{label}.costMode")}
    if "providerExecutable" in profile:
        executable = profile["providerExecutable"]
        if (not isinstance(executable, str) or not executable.startswith("/")
                or executable != executable.strip() or "\x00" in executable
                or ".." in PurePosixPath(executable).parts):
            raise ManifestError(f"{label}.providerExecutable: expected an absolute container path")
        result["providerExecutable"] = executable
    if provider == "codex":
        _choice(result["effort"], CODEX_EFFORTS, f"{label}.effort")
        if result["costMode"] != "metered":
            raise ManifestError(f"{label}.costMode: Codex must be metered")
        if {"providerArgs", "providerVersion"} & profile.keys():
            raise ManifestError(f"{label}: providerArgs and providerVersion apply only to process executors")
    else:
        if "providerExecutable" not in result:
            raise ManifestError(f"{label}: process requires providerExecutable (absolute container path)")
        if "providerArgs" in profile:
            args = profile["providerArgs"]
            if not isinstance(args, list) or any(not isinstance(arg, str) or "\x00" in arg for arg in args):
                raise ManifestError(f"{label}.providerArgs: expected an array of strings without NUL")
            result["providerArgs"] = list(args)
        if "providerVersion" in profile:
            version = profile["providerVersion"]
            if not isinstance(version, str) or not version.strip() or version != version.strip() or "\x00" in version:
                raise ManifestError(f"{label}.providerVersion: expected a non-empty version string")
            result["providerVersion"] = version
    return result


def role_profiles(manifest: dict) -> dict[str, dict]:
    """Resolved requested profiles; runtime.yaml records what config actually wrote."""
    product, agent = manifest.get("markitect") or {}, manifest.get("agent") or {}
    default = {"provider": "codex", "model": product.get("innerModel", agent.get("model")),
               "effort": product.get("innerEffort", agent.get("effort")), "costMode": "metered"}
    selected = product.get("roles") or {}
    return {name: dict(selected.get(name, default)) for name in ROLE_NAMES}


def uses_native_roles(manifest: dict) -> bool:
    return manifest.get("method") == "markitect" and any(
        profile["provider"] == "codex" for profile in role_profiles(manifest).values())


def _model(value: Any, label: str) -> str:
    """A full model id (registration.alias_reason): an alias could name the same model as
    a reviewer under another name."""
    model = _match(value, _TOKEN, label)
    reason = registration.alias_reason(model)
    if reason:
        raise ManifestError(f"{label}: {reason}")
    return model


def default_source_repo(playground: Path | None = None) -> str:
    """The Git checkout that holds the playground, as an absolute path, when it is a
    Markitect checkout (its go.mod declares the product's module)."""
    start = Path(playground or cases.ROOT).resolve()
    for folder in (start, *start.parents):
        if (folder / ".git").exists():
            if go_module(folder) != MARKITECT_MODULE:
                raise ManifestError(f"markitect.sourceRepo: missing, and the Git checkout {folder} that holds the "
                                    f"playground is not a Markitect checkout (its go.mod does not declare module "
                                    f"{MARKITECT_MODULE}); set markitect.sourceRepo to a Markitect checkout")
            return str(folder)
    raise ManifestError(f"markitect.sourceRepo: missing, and no Git checkout holds the playground at {start} "
                        "to use as the default; set markitect.sourceRepo to a Markitect checkout")


def go_module(folder: Path) -> str | None:
    """The module path that `folder/go.mod` declares, or None."""
    try:
        text = (Path(folder) / "go.mod").read_text(encoding="utf-8")
    except (OSError, UnicodeError):
        return None
    for line in text.splitlines():
        words = line.split("//", 1)[0].split()
        if len(words) == 2 and words[0] == "module":
            return words[1].strip('"`')
    return None


def _object(value: Any, where: str, *, required: set[str] = frozenset(),
            optional: set[str] = frozenset()) -> dict:
    if not isinstance(value, dict):
        raise ManifestError(f"{where}: expected a JSON object")
    missing = sorted(required - value.keys())
    if missing:
        raise ManifestError(f"{where}: missing {', '.join(missing)}")
    unknown = sorted(value.keys() - required - optional)
    if unknown:
        raise ManifestError(f"{where}: unknown field(s) {', '.join(unknown)}")
    return value


def _match(value: Any, pattern: re.Pattern[str], where: str) -> str:
    if not isinstance(value, str) or not pattern.fullmatch(value):
        raise ManifestError(f"{where}: {value!r} does not match {pattern.pattern}")
    return value


def _choice(value: Any, choices: tuple[str, ...], where: str) -> str:
    if value not in choices:
        raise ManifestError(f"{where}: expected one of {', '.join(choices)}, got {value!r}")
    return value


def _positive_int(value: Any, where: str) -> int:
    if not isinstance(value, int) or isinstance(value, bool) or value <= 0:
        raise ManifestError(f"{where}: expected a positive integer, got {value!r}")
    return value


def _positive_number(value: Any, where: str) -> int | float:
    if not isinstance(value, (int, float)) or isinstance(value, bool) or not value > 0:
        raise ManifestError(f"{where}: expected a positive number, got {value!r}")
    return value
