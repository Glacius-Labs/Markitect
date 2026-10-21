"""Hall checks: black-box expectations of snapshot h1.

The reserve is the bookings part's choice from 0 to 2 (hall/bookings/reserve),
so no check expects a particular value: each check reads the reserve from the
room record that add-room returns, and `reserve-agreement` requires that the
rooms part reports exactly the bookable seats the bookings part admits, with
one value for every room. Error wording and the organization of the data file
are free; only the envelope, exit code, unchanged data and the specified
records are checked.
"""
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from trial_kit import Checks, expect_equal, fail, no_crash, unusable_change, unusable_query  # noqa: E402

CHECKS = Checks("hall", {"h1": {"snapshot": "h1"}})
H1 = ["h1"]
check = CHECKS.check
RESERVES = (0, 1, 2)


def add(env, code, seats, refuse=False):
    args = ["add-room", "--code", code, "--seats", str(seats)]
    return env.refuse(*args) if refuse else env.ok(*args)


def book(env, code, seats, refuse=False):
    args = ["book", "--code", code, "--seats", str(seats)]
    return env.refuse(*args) if refuse else env.ok(*args)


def collection(env, command, field, *extra):
    result = env.ok(command, *extra)
    if set(result) != {field} or not isinstance(result[field], list):
        fail("`%s` must return exactly {\"%s\": [...]}, got %s" % (command, field, result))
    return result[field]


def rooms(env):
    return collection(env, "rooms", "rooms")


def bookings(env, *extra):
    return collection(env, "bookings", "bookings", *extra)


def reserve_of(record):
    """The reserve a room record shows; it must be one of the model's values."""
    if not isinstance(record, dict) or set(record) != {"code", "seats", "bookable"}:
        fail("a room record has exactly code, seats and bookable, got %s" % record)
    if type(record["seats"]) is not int or type(record["bookable"]) is not int:
        fail("seats and bookable are integers, got %s" % record)
    reserve = record["seats"] - record["bookable"]
    if reserve not in RESERVES:
        fail("the room record shows a reserve of %d, the model allows 0 to 2: %s" % (reserve, record))
    return reserve


def room(env, code, seats):
    record = add(env, code, seats)
    reserve = reserve_of(record)
    return record, reserve


# -- every profile --------------------------------------------------------------------

@check("empty-state", H1, ["hall/storage", "hall/rooms/list-rooms", "hall/bookings/list-bookings"],
       "A missing data file lists no rooms and no bookings")
def _(env, p):
    expect_equal(rooms(env), [], "rooms without a data file")
    expect_equal(bookings(env), [], "bookings without a data file")


@check("add-room-record", H1, ["hall/rooms/add-room", "hall/rooms/room", "hall/rooms/room-code",
                                "hall/bookings/reserve", "hall/inputs"],
       "add-room returns the record with the normalized code, the seats and the bookable seats")
def _(env, p):
    record, reserve = room(env, " b202 ", "040")
    expect_equal(record, {"code": "B202", "seats": 40, "bookable": 40 - reserve}, "added room")


@check("room-inputs", H1, ["hall/rooms/room-code", "hall/rooms/add-room", "hall/inputs"],
       "Malformed or duplicate codes and invalid seats are refused; 3 and 500 seats are allowed")
def _(env, p):
    add(env, "A101", 10)
    for code in ("A10", "AB101", "1101", "A1011", "A-101", "", "   ", "Ä101", "A10１", "a 101", " a101 "):
        add(env, code, 10, refuse=True)
    for seats in ("2", "501", "0", "-3", "+3", "3.0", "1e2", " 5", "5 ", "", "x", "５"):
        add(env, "C303", seats, refuse=True)
    expect_equal(add(env, "C303", "3")["seats"], 3, "3 seats")
    expect_equal(add(env, "D404", "500")["seats"], 500, "500 seats")


@check("rooms-listing", H1, ["hall/rooms/list-rooms", "hall/rooms/room", "hall/storage"],
       "rooms lists every room record sorted by code, and persists across processes")
def _(env, p):
    added = [add(env, code, seats) for code, seats in (("c300", 10), ("A100", 500), ("B200", 3))]
    expect_equal(rooms(env), sorted(added, key=lambda record: record["code"]), "rooms")


@check("reserve-agreement", H1, ["hall/bookings/reserve", "hall/bookings/book", "hall/rooms/room",
                                  "hall/rooms/list-rooms"],
       "Every room shows the same reserve, and book admits exactly the bookable seats the room record shows")
def _(env, p):
    reserves = set()
    for code, seats in (("A101", 3), ("B202", 10), ("C303", 500)):
        record, reserve = room(env, code, seats)
        reserves.add(reserve)
        bookable = record["bookable"]
        if bookable > 1:
            book(env, code, bookable - 1)
        book(env, code, 1)
        book(env, code, 1, refuse=True)
    if len(reserves) != 1:
        fail("rooms show different reserves %s; the model requires one value for every room" % sorted(reserves))
    listed = {record["code"]: reserve_of(record) for record in rooms(env)}
    expect_equal(sorted(set(listed.values())), sorted(reserves), "reserve in the rooms listing")


@check("readme-reserve", H1, ["hall/bookings/reserve", "hall/realization"],
       "The README states the reserve that the room records show")
def _(env, p):
    _, reserve = room(env, "A101", 10)
    readme = env.realization / "README.md"
    if not readme.is_file():
        fail("realization/README.md does not exist")
    lines = [line for line in readme.read_text(encoding="utf-8", errors="replace").splitlines()
             if "reserve" in line.lower()]
    if not any(re.search(r"(?<![0-9])%d(?![0-9])" % reserve, line) for line in lines):
        fail("no README line about the reserve states its value %d: %s" % (reserve, lines[:5]))


@check("book-record", H1, ["hall/bookings/book", "hall/bookings/booking", "hall/rooms/room-code"],
       "book takes a normalized room code, returns the booking record and numbers ids from 1")
def _(env, p):
    room(env, "A101", 20)
    room(env, "B202", 20)
    expect_equal(book(env, " a101 ", "002"), {"id": 1, "code": "A101", "seats": 2}, "first booking")
    expect_equal(book(env, "b202", 1), {"id": 2, "code": "B202", "seats": 1}, "second booking")
    expect_equal(book(env, "A101", 1)["id"], 3, "third id")


@check("book-inputs", H1, ["hall/bookings/book", "hall/rooms/room-code", "hall/inputs"],
       "Unknown rooms, malformed codes and invalid seats are refused without a change")
def _(env, p):
    book(env, "A101", 1, refuse=True)
    room(env, "A101", 500)
    for code in ("B202", "A10", "AB101", "", "Ä101"):
        book(env, code, 1, refuse=True)
    for seats in ("0", "501", "-1", "+1", "1.0", " 1", "", "x", "５"):
        book(env, "A101", seats, refuse=True)
    expect_equal(bookings(env), [], "bookings after refusals")


@check("rooms-independent", H1, ["hall/bookings/book", "hall/bookings/reserve"],
       "Bookings of one room do not count against another room")
def _(env, p):
    first, _ = room(env, "A101", 10)
    second, _ = room(env, "B202", 10)
    book(env, "A101", first["bookable"])
    expect_equal(book(env, "B202", second["bookable"])["seats"], second["bookable"], "the other room is still free")


@check("bookings-listing", H1, ["hall/bookings/list-bookings", "hall/rooms/room-code", "hall/storage"],
       "bookings lists by id, filters by a normalized existing code and refuses an unknown one")
def _(env, p):
    room(env, "A101", 20)
    room(env, "B202", 20)
    made = [book(env, "B202", 1), book(env, "A101", 2), book(env, "B202", 3)]
    expect_equal(bookings(env), made, "all bookings by id")
    expect_equal(bookings(env, "--code", " b202"), [made[0], made[2]], "bookings of B202")
    expect_equal(bookings(env, "--code", "A101"), [made[1]], "bookings of A101")
    env.refuse("bookings", "--code", "C303")
    env.refuse("bookings", "--code", "C3")


@check("reads-untouched", H1, ["hall/storage", "hall/rooms/list-rooms", "hall/bookings/list-bookings"],
       "Queries never write the data file")
def _(env, p):
    room(env, "A101", 10)
    book(env, "A101", 1)
    env.reads_leave_data_untouched([["rooms"], ["bookings"], ["bookings", "--code", "a101"]])


@check("refusal-keeps-data", H1, ["hall/storage", "hall/interface", "hall/bookings/book"],
       "A refused change leaves the data file byte for byte unchanged")
def _(env, p):
    record, _ = room(env, "A101", 3)
    book(env, "A101", record["bookable"])
    book(env, "A101", 1, refuse=True)
    add(env, "a101", 5, refuse=True)


@check("malformed-kept", H1, ["hall/storage", "hall/interface"],
       "Malformed data is reported through the error object and never overwritten")
def _(env, p):
    for content in ("{", "[{", "\xff"):  # not JSON at all: the organization of valid JSON is free
        env.write_db(content.encode("latin-1") if content == "\xff" else content)
        before = env.raw()
        add(env, "A101", 10, refuse=True)
        env.refuse("rooms")
        expect_equal(env.raw() == before, True, "malformed data kept")


@check("missing-directory", H1, ["hall/storage", "hall/interface"],
       "Without its directory, a change fails cleanly and a query is empty")
def _(env, p):
    env.db = env.root / "absent" / "db.json"
    add(env, "A101", 10, refuse=True)
    expect_equal(rooms(env), [], "rooms in a missing directory")
    expect_equal(bookings(env), [], "bookings in a missing directory")
    env.refuse("bookings", "--code", "A101")  # the query sees empty data, so the room does not exist
    if (env.root / "absent").exists():
        fail("the data directory was created")


@check("usage-errors", H1, ["hall/interface"], "Command-line errors use the error object and exit 2")
def _(env, p):
    for args in ((), ("unknown",), ("add-room", "--code", "A101"), ("add-room", "--seats", "3"), ("book",),
                 ("rooms", "--help"), ("--help",), ("bookings", "--code"), ("rooms", "extra")):
        env.refuse(*args)


@check("edge-robustness", H1, ["hall/interface", "hall/inputs", "hall/storage", "hall/rooms/room-code"],
       "Extreme inputs fail through the error object, never with a crash")
def _(env, p):
    room(env, "A101", 10)
    add(env, "B202", "9" * 5000, refuse=True)
    book(env, "A101", "9" * 5000, refuse=True)
    expect_equal(book(env, "a101", "0" * 5000 + "1")["seats"], 1, "book 5000 leading zeros and 1")
    book(env, "A101\udcff", 1, refuse=True)
    add(env, "x" * 5000, 3, refuse=True)
    no_crash(env, "bookings", "--code", "\udcff")
    env.write_db("[" * 100000)
    env.refuse("rooms")
    unusable_query(env, "rooms")
    unusable_change(env, "add-room", "--code", "C303", "--seats", "3")


@check("no-leftovers", H1, ["hall/storage"], "Atomic replacement leaves no temporary files")
def _(env, p):
    room(env, "A101", 10)
    book(env, "A101", 1)
    add(env, "A101", 10, refuse=True)
    leftovers = env.leftover_temp_files()
    if leftovers:
        fail("temporary files left beside the data file: %s" % leftovers)
