"""Shared Unicode naming and White_Space profile for bounded language probes."""
from __future__ import annotations

import unicodedata
from typing import Any

UNICODE_VERSION = "15.1.0"
# Unicode 15.1.0 White_Space, shared by body and body-link display validation.
WHITE_SPACE = frozenset("\t\n\v\f\r \x85\xa0\u1680\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u2028\u2029\u202f\u205f\u3000")


class ProbeLimitError(ValueError):
    pass


def check_unicode_version(required: str = UNICODE_VERSION) -> None:
    if unicodedata.unidata_version != required:
        raise ValueError(f"Unicode table version must be {required}; runtime has {unicodedata.unidata_version}")


def require_nfc(value: Any, category: str) -> None:
    if not isinstance(value, str) or not value:
        raise ValueError(f"{category} must be nonempty text")
    if not unicodedata.is_normalized("NFC", value):
        raise ValueError(f"{category} {value!r} must already be NFC; no automatic correction")


def name_key(value: str) -> str:
    """Unicode canonical caseless matching (D145), only for names/diagnostics."""
    return unicodedata.normalize("NFD", unicodedata.normalize("NFD", value).casefold())


def word_matches(word: str, style: str, *, digit_start: bool = False) -> bool:
    # Python's version-bound identifier predicate uses XID_Start/XID_Continue.
    # Remove connector punctuation, including underscore, to preserve our styles.
    identifier = word.isidentifier() or (digit_start and bool(word) and word[0].isdecimal() and ("x" + word).isidentifier())
    if not identifier or any(unicodedata.category(char) == "Pc" for char in word):
        return False
    letters = [char for char in word if unicodedata.category(char).startswith("L")]
    cased = [char for char in letters if char.lower() != char.upper()]
    if style == "lower":
        return all(char.islower() for char in cased)
    if cased:
        initial_upper = cased[0].isupper() or unicodedata.category(cased[0]) == "Lt"
        if initial_upper != (style == "pascal"):
            return False
    previous_upper = False
    for char in word:
        if unicodedata.category(char).startswith("M"):
            continue
        upper = char.isupper() or unicodedata.category(char) == "Lt"
        if upper and previous_upper:
            return False
        previous_upper = upper
    return True


def name_matches(value: Any, style: str, separator: str | None = None) -> bool:
    if not isinstance(value, str) or not value or not unicodedata.is_normalized("NFC", value):
        return False
    components = value.split(separator) if separator else [value]
    for component in components:
        words = component.split("-") if style == "kebab" else [component]
        if any(not word_matches(word, "lower" if style == "kebab" else style,
                                digit_start=style == "kebab" and index > 0) for index, word in enumerate(words)):
            return False
    return True


def filename_matches(value: Any) -> bool:
    return isinstance(value, str) and value.endswith(".yaml") and name_matches(value[:-5], "kebab", None)


def require_model_label(value: Any, category: str) -> None:
    """Common grammar for model IDs and exact version labels."""
    require_nfc(value, category)
    sections = value.replace("-", ".").split(".")
    if (not all(word_matches(section, "lower", digit_start=True) for section in sections)
            or any(char.lower() != char.upper() and not char.islower() for char in value)):
        raise ValueError(f"{category} {value!r} must use lowercase Unicode label sections separated by . or -; no automatic correction")
