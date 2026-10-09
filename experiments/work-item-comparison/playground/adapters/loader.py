"""Explicit, hash-pinned adapter loading; there is no plugin auto-discovery."""
from __future__ import annotations

import hashlib
import importlib
import importlib.abc
import importlib.util
import json
import re
import sys
from pathlib import Path

from .contract import AdapterError, validate_descriptor


_MANIFEST_KEYS = {"schema", "factory", "configPath", "sourcePins"}
_FACTORY = re.compile(r"^([A-Za-z_]\w*(?:\.[A-Za-z_]\w*)*):([A-Za-z_]\w*)$")
_SHA256 = re.compile(r"^[0-9a-f]{64}$")
_LOADED_PINNED_MODULES = {}


class _VerifiedSourceLoader(importlib.abc.SourceLoader):
    """Import a pinned module only from freshly rehashed source bytes, never pyc."""

    def __init__(self, fullname, source_path, expected_digest):
        self.fullname = fullname
        self.source_path = Path(source_path).resolve()
        self.expected_digest = expected_digest
        self.compiled_digest = None
        self.compiled_path = None

    def get_filename(self, fullname):
        if fullname != self.fullname:
            raise ImportError("verified loader requested for another module")
        return str(self.source_path)

    def is_package(self, fullname):
        return fullname == self.fullname and self.source_path.name == "__init__.py"

    def get_data(self, path):
        candidate = Path(path)
        if candidate.is_symlink() or candidate.resolve() != self.source_path:
            raise ImportError("verified loader source path changed")
        try:
            source = candidate.read_bytes()
        except OSError as exc:
            raise ImportError(f"cannot read pinned source at import time: {candidate}") from exc
        actual = hashlib.sha256(source).hexdigest()
        if actual != self.expected_digest:
            raise ImportError(f"pinned source changed before compilation: {candidate}")
        return source

    def get_code(self, fullname):
        if fullname != self.fullname:
            raise ImportError("verified loader requested for another module")
        source = self.get_data(str(self.source_path))
        code = compile(source, str(self.source_path), "exec", dont_inherit=True)
        if Path(code.co_filename).resolve() != self.source_path:
            raise ImportError("compiled code origin differs from pinned source")
        self.compiled_digest = hashlib.sha256(source).hexdigest()
        self.compiled_path = Path(code.co_filename).resolve()
        return code


class _PinnedModuleFinder(importlib.abc.MetaPathFinder):
    def __init__(self, pinned_modules):
        self.pinned_modules = pinned_modules
        self.loaders = {}

    def find_spec(self, fullname, path=None, target=None):
        binding = self.pinned_modules.get(fullname)
        if binding is None:
            return None
        source_path, expected_digest = binding
        loader = _VerifiedSourceLoader(fullname, source_path, expected_digest)
        self.loaders[fullname] = loader
        spec = importlib.util.spec_from_loader(fullname, loader,
                                               is_package=loader.is_package(fullname))
        spec.origin = source_path
        spec.has_location = True
        return spec


def _absolute_path(value, field):
    if isinstance(value, Path):
        value = str(value)
    if not isinstance(value, str) or not Path(value).is_absolute():
        raise AdapterError(f"{field} must be an absolute path")
    path = Path(value)
    if path.is_symlink():
        raise AdapterError(f"{field} cannot be a symlink")
    return path.resolve()


def _read_manifest(path):
    try:
        value = json.loads(Path(path).read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise AdapterError(f"cannot read adapter manifest: {exc}") from exc
    if not isinstance(value, dict) or set(value) != _MANIFEST_KEYS or value.get("schema") != 1:
        raise AdapterError("adapter manifest must contain exactly schema, factory, configPath, sourcePins")
    return value


def _verify_pins(raw_pins):
    if not isinstance(raw_pins, dict) or not raw_pins:
        raise AdapterError("manifest sourcePins must be a nonempty mapping")
    pins = {}
    for raw_path, expected in raw_pins.items():
        path = _absolute_path(raw_path, "source pin path")
        if not isinstance(expected, str) or not _SHA256.fullmatch(expected):
            raise AdapterError("source pin must be a lowercase SHA-256 digest")
        try:
            actual = hashlib.sha256(path.read_bytes()).hexdigest()
        except OSError as exc:
            raise AdapterError(f"cannot read pinned adapter source: {path}") from exc
        if actual != expected:
            raise AdapterError(f"adapter source pin mismatch: {path}")
        pins[str(path)] = expected
    return pins


def _pinned_module_names(import_root, pins):
    names = {}
    for raw_path, digest in pins.items():
        path = Path(raw_path)
        try:
            relative = path.relative_to(import_root)
        except ValueError:
            continue
        if relative.name == "__init__.py":
            parts = relative.parent.parts
        elif relative.suffix == ".py":
            parts = relative.with_suffix("").parts
        else:
            continue
        if parts:
            names[".".join(parts)] = (str(path.resolve()), digest)
    return names


def load_adapter(manifest_path, config_path=None):
    """Verify all declared source pins before importing the explicitly named factory."""
    manifest = _read_manifest(manifest_path)
    factory = manifest["factory"]
    match = _FACTORY.fullmatch(factory) if isinstance(factory, str) else None
    if not match:
        raise AdapterError("factory must be an explicit module:Class reference")
    module_name, class_name = match.groups()
    pins = _verify_pins(manifest["sourcePins"])

    module_rel = Path(*module_name.split("."))
    candidates = [Path(name) for name in pins if Path(name).as_posix().endswith(module_rel.as_posix() + ".py")]
    if not candidates:
        raise AdapterError("factory module source must be included in sourcePins")
    if len(candidates) != 1:
        raise AdapterError("factory module source is ambiguous in sourcePins")
    module_path = candidates[0]
    package_parts = module_name.split(".")[:-1]
    package_init = module_path.parent
    for _part in reversed(package_parts):
        init_path = package_init / "__init__.py"
        if str(init_path.resolve()) not in pins:
            raise AdapterError("every factory package __init__.py must be included in sourcePins")
        package_init = package_init.parent
    import_root = module_path
    for _ in module_name.split("."):
        import_root = import_root.parent
    pinned_modules = _pinned_module_names(import_root, pins)
    if pinned_modules.get(module_name, (None,))[0] != str(module_path.resolve()):
        raise AdapterError("factory module does not resolve within the manifest import root")
    for cached_name, (expected_path, expected_digest) in pinned_modules.items():
        cached_module = sys.modules.get(cached_name)
        if cached_module is None:
            continue
        cached_binding = _LOADED_PINNED_MODULES.get(cached_name)
        if cached_binding is None:
            raise AdapterError(f"factory import name is already loaded outside the pinned loader: {cached_name}")
        cached_path, cached_digest = cached_binding
        if cached_path != expected_path or cached_digest != expected_digest:
            raise AdapterError(f"cached factory module source changed since pinned load: {cached_name}")
    finder = _PinnedModuleFinder(pinned_modules)
    sys.path.insert(0, str(import_root))
    sys.meta_path.insert(0, finder)
    try:
        module = importlib.import_module(module_name)
        imported_path = Path(getattr(module, "__file__", "")).resolve()
        if imported_path != module_path:
            raise AdapterError(f"import resolved factory module outside its pinned source file: {imported_path} != {module_path}")
        factory_loader = getattr(getattr(module, "__spec__", None), "loader", None)
        factory_binding = pinned_modules[module_name]
        if (not isinstance(factory_loader, _VerifiedSourceLoader) or
                factory_loader.source_path != Path(factory_binding[0]) or
                factory_loader.compiled_path != Path(factory_binding[0]) or
                factory_loader.compiled_digest != factory_binding[1]):
            raise AdapterError("factory code was not compiled from verified pinned source bytes")
        for pinned_name, (pinned_path, pinned_digest) in pinned_modules.items():
            loaded = sys.modules.get(pinned_name)
            if loaded is None:
                continue
            loaded_path = Path(getattr(loaded, "__file__", "")).resolve()
            if loaded_path != Path(pinned_path):
                raise AdapterError(f"pinned dependency imported from a different source file: {pinned_name}")
            loaded_loader = getattr(getattr(loaded, "__spec__", None), "loader", None)
            if (not isinstance(loaded_loader, _VerifiedSourceLoader) or
                    loaded_loader.source_path != Path(pinned_path) or
                    loaded_loader.compiled_path != Path(pinned_path) or
                    loaded_loader.compiled_digest != pinned_digest):
                raise AdapterError(f"pinned dependency was not compiled from verified bytes: {pinned_name}")
            _LOADED_PINNED_MODULES[pinned_name] = (pinned_path, pinned_digest)
        constructor = getattr(module, class_name, None)
        if not callable(constructor):
            raise AdapterError("factory class is not callable")
        chosen_config = config_path if config_path is not None else manifest["configPath"]
        config = _absolute_path(chosen_config, "configPath")
        adapter = constructor(config_path=config)
        bind_pins = getattr(adapter, "bind_manifest_source_pins", None)
        if callable(bind_pins):
            bind_pins(dict(pins))
    except AdapterError:
        raise
    except Exception as exc:
        raise AdapterError(f"cannot construct adapter factory: {type(exc).__name__}: {exc}") from exc
    finally:
        try:
            sys.meta_path.remove(finder)
        except ValueError:
            pass
        try:
            sys.path.remove(str(import_root))
        except ValueError:
            pass

    if not all(callable(getattr(adapter, name, None)) for name in
               ("describe", "setup", "ensure_runtime", "start", "resume", "status", "cancel", "close")):
        raise AdapterError("adapter factory does not implement the schema 1 API")
    descriptor = validate_descriptor(adapter.describe())
    descriptor_pins = descriptor.get("sourcePins", {})
    normalized_descriptor_pins = {}
    for path, digest in descriptor_pins.items():
        normalized_descriptor_pins[str(_absolute_path(path, "descriptor source pin"))] = digest
    if normalized_descriptor_pins != pins:
        raise AdapterError("adapter descriptor sourcePins differ from the verified manifest pins")
    return adapter
