"""Data file of model 0.2.0 (oracle validation only): reads 0.1.0 files, whose items lack reorderPoint."""
import json
import os
import tempfile
from pathlib import Path

import stockroom


class StoreError(Exception):
    """The data file cannot be read or written."""


def load(path):
    path = Path(path)
    try:
        if not path.exists():
            return {}
        document = json.loads(path.read_bytes().decode("utf-8"))
        if not isinstance(document, dict) or set(document) != {"items"} or not isinstance(document["items"], list):
            raise ValueError("expected an object with an items array")
        data = {}
        for item in document["items"]:
            if not isinstance(item, dict) or not {"sku", "name", "onHand"} <= set(item) <= {"sku", "name", "onHand", "reorderPoint"}:
                raise ValueError("unexpected item fields")
            sku, point = item["sku"], item.get("reorderPoint", 0)
            if stockroom.normalize_sku(sku) != sku or sku in data or stockroom.parse_name(item["name"]) != item["name"]:
                raise ValueError("invalid item")
            if type(item["onHand"]) is not int or not 0 <= item["onHand"] <= stockroom.MAX_ON_HAND:
                raise ValueError("invalid on-hand quantity")
            if type(point) is not int or not 0 <= point <= stockroom.MAX_REORDER:
                raise ValueError("invalid reorder point")
            data[sku] = {"name": item["name"], "onHand": item["onHand"], "reorderPoint": point}
        return data
    except (OSError, ValueError, TypeError, KeyError, RecursionError, stockroom.StockError) as error:
        raise StoreError("the data file %s cannot be read: %s" % (path, error)) from None


def save(path, data):
    path = Path(path)
    text = json.dumps({"items": stockroom.list_items(data)}, ensure_ascii=False) + "\n"
    try:
        handle, temporary = tempfile.mkstemp(prefix=path.name + ".", suffix=".tmp", dir=path.parent)
        try:
            with os.fdopen(handle, "w", encoding="utf-8") as stream:
                stream.write(text)
                stream.flush()
                os.fsync(stream.fileno())
            os.replace(temporary, path)
        except BaseException:
            os.unlink(temporary)
            raise
    except OSError as error:
        raise StoreError("the data file %s cannot be written: %s" % (path, error)) from None
