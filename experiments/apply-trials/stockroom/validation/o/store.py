"""Data file under the overspecified model (oracle validation only): {SKU: {name, onHand}}, two-space indent."""
import json
import os
import tempfile
from pathlib import Path

import stockroom
from limits import MAX_ON_HAND


class StoreError(Exception):
    """The data file cannot be read or written."""


def load(path):
    path = Path(path)
    try:
        if not path.exists():
            return {}
        document = json.loads(path.read_bytes().decode("utf-8"))
        if not isinstance(document, dict):
            raise ValueError("expected an object keyed by SKU")
        data = {}
        for sku, item in document.items():
            if stockroom.normalize_sku(sku) != sku or not isinstance(item, dict) or list(item) != ["name", "onHand"]:
                raise ValueError("invalid item %r" % sku)
            if stockroom.parse_name(item["name"]) != item["name"]:
                raise ValueError("invalid name")
            if type(item["onHand"]) is not int or not 0 <= item["onHand"] <= MAX_ON_HAND:
                raise ValueError("invalid on-hand quantity")
            data[sku] = {"name": item["name"], "onHand": item["onHand"]}
        return data
    except (OSError, ValueError, TypeError, KeyError, RecursionError, stockroom.StockError) as error:
        raise StoreError("the data file %s cannot be read: %s" % (path, error)) from None


def save(path, data):
    path = Path(path)
    document = {sku: {"name": data[sku]["name"], "onHand": data[sku]["onHand"]} for sku in sorted(data)}
    text = json.dumps(document, ensure_ascii=False, indent=2) + "\n"
    try:
        handle, temporary = tempfile.mkstemp(prefix=path.name + ".", suffix=".tmp", dir=path.parent)
        try:
            with os.fdopen(handle, "w", encoding="utf-8", newline="\n") as stream:
                stream.write(text)
                stream.flush()
                os.fsync(stream.fileno())
            os.replace(temporary, path)
        except BaseException:
            os.unlink(temporary)
            raise
    except OSError as error:
        raise StoreError("the data file %s cannot be written: %s" % (path, error)) from None
