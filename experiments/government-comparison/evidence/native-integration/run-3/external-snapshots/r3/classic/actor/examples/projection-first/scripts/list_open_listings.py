from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any


EXPECTED_FILTER_STATUS = "Open"
EXPECTED_SORT_FIELDS = ["openedAt", "id"]
EXPECTED_SORT_ORDER = "ascending"
EXPECTED_OUTPUT_FORMAT = "json-stdout"


def load_view(path: str) -> dict[str, Any]:
    with Path(path).open("r", encoding="utf-8") as stream:
        view = json.load(stream)
    if not isinstance(view, dict):
        raise ValueError("view must be a JSON object")
    return view


def view_spec(view: dict[str, Any]) -> dict[str, Any]:
    spec = view.get("spec")
    if not isinstance(spec, dict):
        raise ValueError("view spec must be a JSON object")
    if spec.get("filterStatus") != EXPECTED_FILTER_STATUS:
        raise ValueError("unsupported filterStatus")
    if spec.get("sortFields") != EXPECTED_SORT_FIELDS:
        raise ValueError("unsupported sortFields")
    if spec.get("sortOrder") != EXPECTED_SORT_ORDER:
        raise ValueError("unsupported sortOrder")
    if spec.get("outputFormat") != EXPECTED_OUTPUT_FORMAT:
        raise ValueError("unsupported outputFormat")
    return spec


def read_request() -> dict[str, Any]:
    request = json.load(sys.stdin)
    if not isinstance(request, dict):
        raise ValueError("stdin must be a JSON object")
    items = request.get("items")
    if not isinstance(items, list):
        raise ValueError("stdin must contain an items array")
    for item in items:
        if not isinstance(item, dict):
            raise ValueError("each item must be a JSON object")
    return request


def open_listings(items: list[dict[str, Any]], filter_status: str) -> list[dict[str, Any]]:
    selected = [item for item in items if item.get("status") == filter_status]
    return sorted(selected, key=lambda item: (item.get("openedAt", ""), item.get("id", "")))


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Render the finite ListingView JSON projection.")
    parser.add_argument("--view", required=True, help="Path to resources/list-open-listings.yaml")
    args = parser.parse_args(argv)

    try:
        spec = view_spec(load_view(args.view))
        request = read_request()
        rows = open_listings(request["items"], spec["filterStatus"])
        json.dump({"items": rows}, sys.stdout, separators=(",", ":"))
        sys.stdout.write("\n")
        return 0
    except Exception as exc:
        print(f"list-open-listings: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
