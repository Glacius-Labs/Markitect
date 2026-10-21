"""The one data file of all parts: UTF-8 JSON, replaced atomically once per changing command.

The document is {"rooms": {code: seats}, "bookings": [booking, ...]}; each
part validates its own section.
"""
import json
import os
import tempfile
from pathlib import Path

from bookings import bookings
from hall.errors import Refusal
from rooms import rooms


class StoreError(Exception):
    """The data file cannot be read or written."""


def empty():
    return {"rooms": {}, "bookings": []}


def load(path):
    """Read and validate the data; a missing file is empty data."""
    path = Path(path)
    try:
        if not path.exists():
            return empty()
        state = json.loads(path.read_bytes().decode("utf-8"))
        if not isinstance(state, dict) or set(state) != {"rooms", "bookings"}:
            raise ValueError("expected an object with rooms and bookings")
        rooms.validate(state["rooms"])
        bookings.validate(state["bookings"], state["rooms"])
        return state
    except (OSError, ValueError, TypeError, KeyError, RecursionError, Refusal) as error:
        raise StoreError("the data file %s cannot be read: %s" % (path, error)) from None


def save(path, state):
    """Write the data next to the target and atomically replace the file."""
    path = Path(path)
    text = json.dumps(state, ensure_ascii=False, sort_keys=True) + "\n"
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
