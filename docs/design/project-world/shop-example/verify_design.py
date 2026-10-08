"""Validate this proposed design specimen, not a current Markitect project."""

from pathlib import Path, PurePosixPath
import json
import yaml


ROOT = Path(__file__).resolve().parent
PROJECT = ROOT / "project"
VERSION = "project-world/0.1"


class UniqueLoader(yaml.SafeLoader):
    pass


def unique_mapping(loader, node, deep=False):
    result = {}
    for key_node, value_node in node.value:
        key = loader.construct_object(key_node, deep=deep)
        if key in result:
            raise ValueError(f"Duplicate YAML key: {key}")
        result[key] = loader.construct_object(value_node, deep=deep)
    return result


UniqueLoader.add_constructor(
    yaml.resolver.BaseResolver.DEFAULT_MAPPING_TAG, unique_mapping
)


def require(condition, message):
    if not condition:
        raise ValueError(message)


def read(path):
    value = yaml.load(path.read_text(encoding="utf-8"), Loader=UniqueLoader)
    require(isinstance(value, dict), f"Expected mapping: {path.name}")
    require(value.get("designVersion") == VERSION, f"Unexpected design version: {path}")
    return value


def safe_path(value):
    path = PurePosixPath(value)
    require(value and not path.is_absolute() and ".." not in path.parts,
            f"Unsafe specimen path: {value}")
    require("\\" not in value and ":" not in value, f"Nonportable path: {value}")


def model(before=False):
    manifest = read(PROJECT / "markitect.yaml")
    files = manifest["modelFiles"]
    require(len(files) == len(set(files)), "Duplicate selected model file")
    resources = {}
    for filename in files:
        safe_path(filename)
        replacement = ROOT / "before" / filename
        source = replacement if before and replacement.exists() else PROJECT / filename
        for resource in read(source)["resources"]:
            identity = resource["id"]
            require(identity not in resources, f"Duplicate identity: {identity}")
            require(all(resource.get(key) for key in ("kind", "owner", "area", "purpose")),
                    f"Missing ownership or purpose: {identity}")
            resources[identity] = resource

    def ref(identity, expected=None):
        require(identity in resources, f"Unresolved reference: {identity}")
        if expected:
            require(resources[identity]["kind"] == expected,
                    f"Expected {expected}: {identity}")
        return resources[identity]

    single_refs = {
        "owner": "Manager", "area": "Area", "parent": "Area",
        "manager": "Manager", "manages": "Area", "module": "Module",
        "namespace": "Namespace", "architecture": "Architecture",
        "policy": "DecisionPolicy", "workflow": "Workflow", "executive": "Manager",
    }
    list_refs = {"uses": None, "rules": "Rule", "realizes": None,
                 "requiresRealization": None, "checks": "Check"}
    for identity, resource in resources.items():
        for key, expected in single_refs.items():
            if resource.get(key) is not None:
                ref(resource[key], expected)
        for key, expected in list_refs.items():
            for target in resource.get(key, []):
                ref(target, expected)
        require(ref(resource["area"], "Area")["owner"] == resource["owner"],
                f"Role/area responsibility mismatch: {identity}")
        if resource["kind"] == "Module":
            for filename in resource["contentFiles"]:
                require(filename in files, f"Unselected module file: {filename}")
            require(resource["revision"] == "workspace", "Expected local module revision")
        if resource["kind"] == "Check":
            require(isinstance(resource["command"], list) and resource["command"],
                    f"Missing literal check arguments: {identity}")

    areas = {key: item for key, item in resources.items() if item["kind"] == "Area"}
    managers = {key: item for key, item in resources.items() if item["kind"] == "Manager"}
    require([key for key, item in areas.items() if item["parent"] is None] == [manifest["rootArea"]],
            "Expected one project root")
    require(manifest["owner"] == areas[manifest["rootArea"]]["owner"], "Root owner mismatch")
    ref(manifest["executionProcess"], "Process")
    for identity, area in areas.items():
        require(area["manager"] == area["owner"], f"Two accountable managers: {identity}")
        require(managers[area["manager"]]["manages"] == identity,
                f"Manager assignment mismatch: {identity}")
        seen = set()
        cursor = identity
        while cursor is not None:
            require(cursor not in seen, f"Cyclic area parents: {identity}")
            seen.add(cursor)
            cursor = areas[cursor]["parent"]
    require(len({item["manages"] for item in managers.values()}) == len(managers) == len(areas),
            "Expected one distinct manager role per area")

    paths = {}
    roots = []
    coverage = {}
    for identity, resource in resources.items():
        if resource["kind"] != "ArtifactGroup":
            continue
        for obligation in resource["realizes"]:
            coverage.setdefault(obligation, []).append(identity)
        for path in resource.get("knownPaths", []):
            safe_path(path)
            require(path not in paths, f"Competing artifact owner: {path}")
            paths[path] = resource["owner"]
        for path in resource.get("managedRoots", []):
            safe_path(path)
            require(path.endswith("/"), f"Expected directory root: {path}")
            for existing, owner in roots:
                require(not path.startswith(existing) and not existing.startswith(path),
                        f"Overlapping write scopes: {existing}, {path}")
            roots.append((path, resource["owner"]))
    for path, owner in paths.items():
        for prefix, scope_owner in roots:
            if path.startswith(prefix):
                require(owner == scope_owner, f"Artifact/scope owner mismatch: {path}")
    for identity, resource in resources.items():
        if resource["kind"] == "UseCase":
            require(identity in coverage, f"Use case has no declared realization: {identity}")
        for obligation in resource.get("requiresRealization", []):
            require(obligation in coverage, f"Required realization missing: {obligation}")
            require(all(resources[group].get("checks") for group in coverage[obligation]),
                    f"Required realization has no check declaration: {obligation}")

    runtime = read(PROJECT / manifest["runtimeFile"])
    runtime_roles = runtime["roles"]
    require(len(runtime_roles) == len(managers) and
            {item["manager"] for item in runtime_roles} == set(managers),
            "Runtime must assign every distinct manager")
    require(all(item["context"] == "separate" and item["verifier"] == "separate"
                for item in runtime_roles), "Shared manager/verifier context")
    require(runtime["contexts"]["includeChildTranscripts"] is False and
            runtime["contexts"]["includeDescendantContexts"] is False,
            "Child contexts must not be inherited")
    require(runtime["execution"]["start"] == "explicit", "Execution must have a start action")
    return resources, paths


def main():
    before, before_paths = model(before=True)
    after, after_paths = model()
    added = sorted(set(after) - set(before))
    changed = sorted(key for key in before.keys() & after.keys() if before[key] != after[key])
    expected_added = {
        "sales/cancelled", "sales/cancellation", "sales/cancel-before-shipped",
        "sales/cancel-order", "commerce/cancellation-releases-reservation",
    }
    require(set(added) == expected_added, f"Unexpected added definitions: {added}")
    expected_changed = {
        "sales/order", "engineering/orders-artifacts", "engineering/commerce-artifacts",
        "engineering/check-orders", "engineering/check-commerce",
    }
    require(set(changed) == expected_changed, f"Unexpected changed definitions: {changed}")
    require(before_paths == after_paths, "Design edit unexpectedly invented implementation files")
    require(all(before[key] == after[key] for key in before if key.startswith("inventory/")),
            "Inventory vocabulary unexpectedly changed")
    print(json.dumps({
        "scope": "design-specimen-only", "status": "valid",
        "baselineDefinitions": len(before), "targetDefinitions": len(after),
        "managerRoles": 5, "declaredArtifactPaths": len(after_paths),
        "added": added, "changed": changed,
        "agentsStarted": 0, "shopChecksRun": 0,
    }, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
