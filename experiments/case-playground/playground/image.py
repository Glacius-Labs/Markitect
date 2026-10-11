"""Image pins and the image inventory (README "Updating the image pins").

container/Dockerfile is the one place for the pins: `ARG BASE_IMAGE=<image>@sha256:<index
digest>` before the one `FROM ${BASE_IMAGE}`, and `ARG DEBIAN_SNAPSHOT=<YYYYMMDDTHHMMSSZ>`,
the snapshot.debian.org time every Debian package comes from (debian and
debian-security); the host passes SOURCE_DATE_EPOCH from it. `pins` refuses a Dockerfile
without them.

The Dockerfile's last step writes INVENTORY_IN_IMAGE, one entry per line:

  markitect-playground-inventory 1
  base <BASE_IMAGE>
  snapshot <DEBIAN_SNAPSHOT>
  node <node --version>
  dpkg <name> <version> <arch>     every installed Debian package, sorted
  npm <name>@<version>             every package in both CLIs' installed trees, sorted

`check` accepts a built inventory only when every npm entry is an exact version, its base
and snapshot are the Dockerfile's, and, when a committed inventory
(container/inventory/codex-<v>-claude-<v>.txt) is given, it has the same entries. `key`
is the cache key of `host image-key`. Standard library only.
"""
from __future__ import annotations

import hashlib
import re
from datetime import datetime, timezone
from pathlib import Path

HEADER = "markitect-playground-inventory 1"
KEY_HEADER = "markitect-playground image key 1"
INVENTORY_IN_IMAGE = "/usr/share/markitect-playground/inventory.txt"
DIGEST = re.compile(r"[^\s@]+@sha256:[0-9a-f]{64}")
SNAPSHOT = re.compile(r"\d{8}T\d{6}Z")
SNAPSHOT_FORMAT = "%Y%m%dT%H%M%SZ"
# An exact semantic version, never a range, tag, URL or path.
EXACT = re.compile(r"(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?")
SNAPSHOT_URIS = ("snapshot.debian.org/archive/debian/${DEBIAN_SNAPSHOT}",
                 "snapshot.debian.org/archive/debian-security/${DEBIAN_SNAPSHOT}")
SHOWN = 20  # entries a message names at most


class ImageError(ValueError):
    """A Dockerfile without its pins, or an inventory that is unreadable, not exact or different."""


def sha256(data: bytes | str) -> str:
    return hashlib.sha256(data.encode("utf-8") if isinstance(data, str) else data).hexdigest()


def inventory_name(codex: str, claude: str) -> str:
    """The committed inventory's file name for two CLI versions (the image tag's suffix)."""
    return f"codex-{codex}-claude-{claude}.txt"


def _arg(text: str, name: str, dockerfile: Path) -> str | None:
    values = re.findall(rf"(?m)^ARG[ \t]+{name}=(\S*)[ \t]*$", text)
    if len(values) > 1:
        raise ImageError(f"{dockerfile} sets {name} more than once")
    return values[0] if values else None


def pins(dockerfile: Path) -> dict:
    """{"base", "snapshot", "sourceDateEpoch", "dockerfileSha256"} from the Dockerfile
    (instructions in upper case, so that a heredoc line such as Python's `from x import y`
    is never taken for one); raises ImageError when a pin is missing or not in its form."""
    try:
        data = dockerfile.read_bytes()
    except OSError as exc:
        raise ImageError(f"cannot read {dockerfile}: {exc}") from exc
    text = data.decode("utf-8", errors="replace")
    base = _arg(text, "BASE_IMAGE", dockerfile)
    if base is None:
        raise ImageError(f"{dockerfile} has no `ARG BASE_IMAGE=<image>@sha256:<digest>`")
    if not DIGEST.fullmatch(base):
        raise ImageError(f"{dockerfile}: BASE_IMAGE {base} is not pinned by digest (<image>@sha256:<64 hex digits>)")
    froms = list(re.finditer(r"(?m)^FROM[ \t]+(.*?)[ \t]*$", text))
    if [match.group(1) for match in froms] != ["${BASE_IMAGE}"]:
        raise ImageError(f"{dockerfile} must have exactly one FROM line, `FROM ${{BASE_IMAGE}}`")
    if re.search(r"(?m)^ARG[ \t]+BASE_IMAGE=", text).start() > froms[0].start():
        raise ImageError(f"{dockerfile}: `ARG BASE_IMAGE=...` must come before `FROM ${{BASE_IMAGE}}`")
    snapshot = _arg(text, "DEBIAN_SNAPSHOT", dockerfile)
    if snapshot is None or not SNAPSHOT.fullmatch(snapshot):
        found = "" if snapshot is None else f" (found {snapshot!r})"
        raise ImageError(f"{dockerfile} has no `ARG DEBIAN_SNAPSHOT=<YYYYMMDDTHHMMSSZ>`{found}")
    try:
        moment = datetime.strptime(snapshot, SNAPSHOT_FORMAT).replace(tzinfo=timezone.utc)
    except ValueError as exc:
        raise ImageError(f"{dockerfile}: DEBIAN_SNAPSHOT {snapshot} is not a time ({exc})") from exc
    unused = [uri for uri in SNAPSHOT_URIS if uri not in text]
    if unused:
        raise ImageError(f"{dockerfile} does not take its Debian packages from {', '.join(unused)}")
    return {"base": base, "snapshot": snapshot, "sourceDateEpoch": int(moment.timestamp()),
            "dockerfileSha256": sha256(data)}


def _lines(text: str) -> list[str]:
    return [line.strip() for line in text.splitlines() if line.strip()]


def parse(text: str) -> dict:
    """{"base", "snapshot", "node", "dpkg": [...], "npm": [(name, version), ...]}; raises
    ImageError for anything that is not an inventory."""
    lines = _lines(text)
    if not lines or lines[0] != HEADER:
        raise ImageError(f"not an image inventory: the first line is not {HEADER!r}")
    single: dict[str, str | None] = {"base": None, "snapshot": None, "node": None}
    dpkg: list[str] = []
    npm: list[tuple[str, str]] = []
    for number, line in enumerate(lines[1:], 2):
        kind, _, rest = line.partition(" ")
        rest = rest.strip()
        if kind in single:
            if single[kind] is not None:
                raise ImageError(f"line {number}: a second {kind} entry")
            if not rest or len(rest.split()) != 1:
                raise ImageError(f"line {number}: `{kind}` takes one value: {line!r}")
            single[kind] = rest
        elif kind == "dpkg":
            if len(rest.split()) != 3:
                raise ImageError(f"line {number}: `dpkg` takes name, version and architecture: {line!r}")
            dpkg.append(" ".join(rest.split()))
        elif kind == "npm":
            name, at, version = rest.rpartition("@")
            if not at or not name or name == "@" or " " in rest:
                raise ImageError(f"line {number}: `npm` takes <name>@<version>: {line!r}")
            npm.append((name, version))
        else:
            raise ImageError(f"line {number}: unknown entry {kind!r}")
    missing = [kind for kind, value in single.items() if value is None]
    if missing:
        raise ImageError(f"no {', '.join(missing)} entry")
    if not npm:
        raise ImageError("no npm entry: the CLI dependency trees are missing")
    return {**single, "dpkg": dpkg, "npm": npm}


def validate(text: str, pinned: dict) -> dict:
    """`parse`, then every npm entry an exact version and the Dockerfile's base and snapshot."""
    parsed = parse(text)
    inexact = [f"{name}@{version}" for name, version in parsed["npm"] if not EXACT.fullmatch(version)]
    if inexact:
        raise ImageError(f"{len(inexact)} entr{'y' if len(inexact) == 1 else 'ies'} of the CLI dependency trees "
                         f"{'is' if len(inexact) == 1 else 'are'} not an exact version: "
                         f"{', '.join(inexact[:SHOWN])}")
    for key in ("base", "snapshot"):
        if parsed[key] != pinned[key]:
            raise ImageError(f"its {key} is {parsed[key]}, the Dockerfile pins {pinned[key]}")
    return parsed


def differences(built: str, committed: str) -> list[str]:
    """`- entry` only in the committed inventory, `+ entry` only in the built one."""
    new, old = _lines(built), _lines(committed)
    if new == old:
        return []
    new_set, old_set = set(new), set(old)
    found = [f"- {line}" for line in old if line not in new_set] + [f"+ {line}" for line in new if line not in old_set]
    return found or ["the same entries in another order or repeated"]


def check(built: str, pinned: dict, committed: str | None) -> dict:
    """The built inventory, validated and, when a committed one is given, compared with it;
    returns the parsed built inventory. Raises ImageError naming the problem."""
    parsed = validate(built, pinned)
    if committed is None:
        return parsed
    try:
        validate(committed, pinned)
    except ImageError as exc:
        raise ImageError(f"the committed inventory: {exc}") from exc
    found = differences(built, committed)
    if found:
        more = f" (and {len(found) - SHOWN} more)" if len(found) > SHOWN else ""
        raise ImageError(f"it differs from the committed inventory in {len(found)} entr"
                         f"{'y' if len(found) == 1 else 'ies'}: {'; '.join(found[:SHOWN])}{more}")
    return parsed


def key(container: Path) -> tuple[str, list[Path]]:
    """(cache key, committed inventory files) for `container` (the folder holding
    Dockerfile and inventory/): a SHA-256 over the pinned base, the snapshot, the
    Dockerfile's SHA-256 and every committed inventory file's name and SHA-256."""
    pinned = pins(container / "Dockerfile")
    folder = container / "inventory"
    files = sorted(folder.glob("*.txt")) if folder.is_dir() else []
    material = [KEY_HEADER, f"base {pinned['base']}", f"snapshot {pinned['snapshot']}",
                f"dockerfile {pinned['dockerfileSha256']}"]
    material += [f"inventory {path.name} {sha256(path.read_bytes())}" for path in files] or ["inventory none"]
    digest = sha256("".join(line + "\n" for line in material))
    return f"playground-image-{digest}", files
