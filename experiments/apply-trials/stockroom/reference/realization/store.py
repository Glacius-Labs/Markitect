"""The data file: UTF-8 JSON {"items": [record, ...]}, replaced atomically on every change."""
import json
import os
import tempfile
from pathlib import Path

import stockroom


class StoreError(Exception):
    """The data file cannot be read or written."""


def load(path):
    """Read the data as a dict from SKU to {"name", "onHand"}; a missing file is empty data."""
    path = Path(path)
    if not path.exists():
        return {}
    try:
        document = json.loads(path.read_bytes().decode("utf-8"))
        return from_document(document)
    except (OSError, UnicodeDecodeError, ValueError, TypeError, KeyError, stockroom.StockError) as error:
        raise StoreError("the data file %s cannot be read: %s" % (path, error)) from None


def from_document(document):
    if not isinstance(document, dict) or set(document) != {"items"} or not isinstance(document["items"], list):
        raise ValueError("expected an object with an items array")
    data = {}
    for item in document["items"]:
        if not isinstance(item, dict) or set(item) != {"sku", "name", "onHand"}:
            raise ValueError("expected items with exactly sku, name and onHand")
        sku, name, on_hand = item["sku"], item["name"], item["onHand"]
        if not isinstance(sku, str) or stockroom.normalize_sku(sku) != sku or sku in data:
            raise ValueError("invalid or duplicate SKU %r" % (sku,))
        if not isinstance(name, str) or stockroom.parse_name(name) != name:
            raise ValueError("invalid name for %s" % sku)
        if type(on_hand) is not int or not 0 <= on_hand <= stockroom.LIMITS["maxOnHand"]:
            raise ValueError("invalid on-hand quantity for %s" % sku)
        data[sku] = {"name": name, "onHand": on_hand}
    return data


def save(path, data):
    """Write the data next to the target and atomically replace the file."""
    path = Path(path)
    items = [{"sku": sku, "name": data[sku]["name"], "onHand": data[sku]["onHand"]} for sku in sorted(data)]
    text = json.dumps({"items": items}, ensure_ascii=False) + "\n"
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
