"""Roombook checks. Each cites the Definitions of the snapshot it checks.

Stage 1 is 01-book-list, stage 2 adds cancel, stage 3 adds summary. The
expectations follow the Definitions and the guide's recorded owner choice:
summary groups only active reservations and omits a room whose whole history is
canceled. Error wording is free; only the envelope, exit code and unchanged
data are checked. Stored data is inspected only as the storage Definition
states it: UTF-8 JSON at --db, atomic replacement, nothing rewritten by reads.
"""
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

from oracle_kit import Registry, expect_equal, expect_keys, fail  # noqa: E402

REG = Registry("roombook", "roombook", {1: "01-book-list", 2: "02-cancel", 3: "03-summary"})
check = REG.check

FIELDS = {"id", "room", "start", "end", "title", "status"}
DAY = "2026-10-09"


def book(env, room, start, end, title="Meeting", refuse=False):
    args = ["book", "--room", room, "--start", start, "--end", end, "--title", title]
    return env.refuse(*args) if refuse else env.ok(*args)


def at(hour, minute=0, day=DAY):
    return "%sT%02d:%02dZ" % (day, hour, minute)


def check_record(record, label, **expected):
    expect_keys(record, FIELDS, label)
    if type(record["id"]) is not int:
        fail("%s: id must be an integer, got %s" % (label, record["id"]))
    for key, value in expected.items():
        expect_equal(record[key], value, "%s.%s" % (label, key))


def listing(env, *extra):
    result = env.ok("list", *extra)
    expect_keys(result, {"reservations"}, "list result")
    if not isinstance(result["reservations"], list):
        fail("list.reservations must be an array")
    return result["reservations"]


# -- stage 1: book and list ------------------------------------------------------

@check("list-empty", 1, ["list", "storage", "interface"], "A missing data file is empty state")
def _(env):
    expect_equal(env.ok("list"), {"reservations": []}, "list on a missing database")
    expect_equal(env.ok("list", "--room", "A"), {"reservations": []}, "filtered list on a missing database")


@check("book-record", 1, ["book", "reservation", "interface"],
       "Booking returns the created active record with the first integer id and normalized text")
def _(env):
    record = book(env, "  A  ", at(10), at(11), "  Standup  ")
    check_record(record, "booked record", id=1, room="A", start=at(10), end=at(11), title="Standup", status="active")


@check("persist-across-processes", 1, ["storage", "list", "book"],
       "A later process observes the committed record in a durable UTF-8 JSON file")
def _(env):
    record = book(env, "A", at(10), at(11))
    expect_equal(listing(env), [record], "list after booking")
    env.stored_json()


@check("ids-increase", 1, ["reservation", "book"], "Ids start at 1 and increase over successive creations")
def _(env):
    ids = [book(env, room, at(10), at(11))["id"] for room in ("A", "B", "C")]
    expect_equal(ids[0], 1, "first id")
    if not (ids[0] < ids[1] < ids[2]):
        fail("ids must increase, got %s" % ids)


@check("list-order", 1, ["list"], "Listing is sorted by start, then by id")
def _(env):
    late = book(env, "A", at(12), at(13))
    first = book(env, "B", at(10), at(11))
    tie = book(env, "C", at(10), at(10, 30))
    early = book(env, "A", at(9), at(10))
    expect_equal([item["id"] for item in listing(env)], [early["id"], first["id"], tie["id"], late["id"]],
                 "order of ids")


@check("list-room-filter", 1, ["list", "inputs"],
       "The room filter is trimmed and case-sensitive, keeps the ordering and may match nothing")
def _(env):
    later = book(env, "A", at(12), at(13))
    earlier = book(env, "A", at(9), at(10))
    book(env, "a", at(9), at(10))
    book(env, "B", at(9), at(10))
    expect_equal([item["id"] for item in listing(env, "--room", "  A ")], [earlier["id"], later["id"]],
                 "filter ' A '")
    expect_equal(len(listing(env, "--room", "a")), 1, "case-sensitive filter 'a'")
    expect_equal(listing(env, "--room", "Nowhere"), [], "filter without matches")


@check("room-identity", 1, ["inputs", "availability"],
       "Rooms are trimmed and case-sensitive: a differently cased room does not conflict")
def _(env):
    book(env, "Room", at(10), at(11))
    other = book(env, "room", at(10), at(11))
    expect_equal(other["room"], "room", "second room")
    book(env, "  Room  ", at(12), at(13))


@check("overlap-refused", 1, ["availability", "book", "inputs"],
       "Intersecting active intervals in the same room are refused without changing data")
def _(env):
    book(env, "A", at(10), at(11))
    for start, end in ((at(10, 30), at(11, 30)), (at(9, 30), at(10, 30)), (at(10, 15), at(10, 45)),
                       (at(9), at(12)), (at(10), at(11))):
        book(env, "A", start, end, refuse=True)
    book(env, " A ", at(10, 30), at(11, 30), refuse=True)  # the normalized room is the same room
    expect_equal(len(listing(env)), 1, "reservations after refusals")


@check("adjacent-allowed", 1, ["availability"], "Intervals are half-open: touching ends are allowed")
def _(env):
    book(env, "A", at(10), at(11))
    book(env, "A", at(11), at(12))
    book(env, "A", at(9), at(10))
    expect_equal(len(listing(env)), 3, "reservations")


@check("other-rooms-free", 1, ["availability", "book"], "Overlapping intervals in other rooms are permitted")
def _(env):
    book(env, "A", at(10), at(11))
    book(env, "B", at(10), at(11))
    book(env, "C", at(10, 30), at(10, 45))
    expect_equal(len(listing(env)), 3, "reservations")


@check("invalid-timestamps", 1, ["inputs", "interface"],
       "Timestamps must be exactly YYYY-MM-DDTHH:MMZ, valid calendar values, with no reinterpretation")
def _(env):
    good_end, good_start = at(23, 0, "2030-01-01"), at(1, 0, "2030-01-01")
    for bad in ("2026-10-09T10:00:00Z", "2026-10-09T10:00+01:00", "2026-10-09 10:00Z", "2026-10-09T10:00",
                "2026-10-09", "2026-10-09T10:00z", "2026-02-30T10:00Z", "2026-13-01T10:00Z",
                "2026-10-09T24:00Z", "2026-10-09T10:60Z", "2026-02-29T10:00Z", "26-10-09T10:00Z",
                "2026-10-09T1:00Z", "", "tomorrow"):
        book(env, "A", bad, good_end, refuse=True)
        book(env, "A", good_start, bad, refuse=True)
    expect_equal(listing(env), [], "reservations after every refusal")
    leap = book(env, "A", "2028-02-29T10:00Z", "2028-02-29T11:00Z")
    check_record(leap, "leap day", start="2028-02-29T10:00Z")


@check("end-after-start", 1, ["inputs"], "The end must be later than the start")
def _(env):
    book(env, "A", at(10), at(10), refuse=True)
    book(env, "A", at(11), at(10), refuse=True)
    book(env, "A", at(10), at(10, 1))


@check("required-text", 1, ["inputs"], "Room and title must be nonempty after trimming")
def _(env):
    for room, title in (("", "T"), ("   ", "T"), ("A", ""), ("A", "   ")):
        book(env, room, at(10), at(11), title, refuse=True)
    expect_equal(listing(env), [], "reservations after every refusal")


@check("missing-option", 1, ["interface", "inputs"], "A missing required option is an input error in the JSON envelope")
def _(env):
    env.refuse("book", "--room", "A", "--start", at(10), "--end", at(11))
    env.refuse("book", "--start", at(10), "--end", at(11), "--title", "T")


@check("text-roundtrip", 1, ["storage", "inputs", "book"],
       "Non-ASCII text, commas and newlines survive storage and later processes")
def _(env):
    title = "Meet,\nteam ☕ Café \"quoted\""
    record = book(env, "Büro", at(10), at(11), title)
    expect_equal(record["title"], title, "title in the booking result")
    expect_equal(listing(env), [record], "listing after a later process")
    env.stored_json()


@check("reads-leave-data", 1, ["list", "storage"], "Listing never changes the stored data")
def _(env):
    book(env, "A", at(10), at(11))
    book(env, "B", at(12), at(13))
    env.reads_leave_data_untouched([("list",), ("list", "--room", "A"), ("list", "--room", "none")])


@check("malformed-data", 1, ["storage", "interface"],
       "Malformed stored data fails clearly through the envelope and is not overwritten")
def _(env):
    for content in (b'{"reservations": [', b"not json at all", b"\xff\xfe\x00 bad bytes"):
        env.write_db(content)
        env.refuse("list")
        env.refuse("book", "--room", "A", "--start", at(10), "--end", at(11), "--title", "T")


@check("atomic-replacement", 1, ["storage"],
       "A mutation replaces the file atomically and leaves no temporary file behind")
def _(env):
    book(env, "A", at(10), at(11))
    first = env.stat()[2]
    book(env, "B", at(10), at(11))
    if env.stat()[2] == first:
        fail("the data file kept its identity: the second mutation did not replace it atomically")
    if sorted(path.name for path in env.dbdir.iterdir()) != [env.db.name]:
        fail("leftover files next to the database: %s" % sorted(path.name for path in env.dbdir.iterdir()))


# -- stage 2: cancel -------------------------------------------------------------

@check("cancel-record", 2, ["cancel", "interface"],
       "Cancel returns the same record with status canceled and changes nothing else", )
def _(env):
    first = book(env, " Lab ", at(10), at(11), "  Café, 1 ")
    other = book(env, "B", at(10), at(11))
    canceled = env.ok("cancel", "--id", first["id"])
    expect_equal(canceled, dict(first, status="canceled"), "canceled record")
    expect_equal(listing(env), sorted([canceled, other], key=lambda item: item["id"]), "listing after cancel")


@check("cancel-idempotent", 2, ["cancel"], "Repeated cancellation succeeds with the same record and unchanged data")
def _(env):
    record = book(env, "A", at(10), at(11))
    once = env.ok("cancel", "--id", record["id"])
    before = env.raw()
    expect_equal(env.ok("cancel", "--id", record["id"]), once, "repeated cancel")
    if env.raw() != before:
        fail("repeating cancel changed the stored data")


@check("cancel-bad-id", 2, ["cancel", "interface"], "Unknown and invalid ids fail without changing data")
def _(env):
    book(env, "A", at(10), at(11))
    for bad in ("99", "0", "-1", "abc", "1.5", ""):
        env.refuse("cancel", "--id", bad)
    env.refuse("cancel")


@check("history-kept", 2, ["cancel", "list", "reservation"],
       "Canceled records stay in the listing and the room filter, in order")
def _(env):
    one = book(env, "A", at(12), at(13))
    two = book(env, "A", at(10), at(11))
    env.ok("cancel", "--id", one["id"])
    items = listing(env)
    expect_equal([(item["id"], item["status"]) for item in items],
                 [(two["id"], "active"), (one["id"], "canceled")], "ordered history")
    expect_equal(len(listing(env, "--room", "A")), 2, "room filter keeps canceled records")


@check("cancel-releases-interval", 2, ["cancel", "availability"],
       "A canceled interval no longer blocks; a new active one blocks again")
def _(env):
    first = book(env, "A", at(10), at(11))
    env.ok("cancel", "--id", first["id"])
    again = book(env, "A", at(10), at(11))
    book(env, "A", at(10, 30), at(11, 30), refuse=True)
    states = {item["id"]: item["status"] for item in listing(env)}
    expect_equal(states, {first["id"]: "canceled", again["id"]: "active"}, "statuses after rebooking")


@check("ids-never-reused", 2, ["reservation", "cancel"], "A canceled record's id is never reused")
def _(env):
    ids = [book(env, "A", at(8 + n), at(9 + n))["id"] for n in range(3)]
    env.ok("cancel", "--id", ids[-1])
    later = book(env, "B", at(10), at(11))["id"]
    env.ok("cancel", "--id", later)
    latest = book(env, "B", at(10), at(11))["id"]
    seen = ids + [later, latest]
    if len(set(seen)) != len(seen) or not (ids[-1] < later < latest):
        fail("ids must stay unique and increasing after cancellation, got %s" % seen)


@check("cancel-keeps-others", 2, ["cancel"], "Cancellation preserves every other record")
def _(env):
    records = [book(env, "A", at(8 + n), at(9 + n), "T%d" % n) for n in range(3)]
    env.ok("cancel", "--id", records[1]["id"])
    after = listing(env)
    expect_equal([after[0], after[2]], [records[0], records[2]], "untouched records")
    expect_equal(after[1]["status"], "canceled", "canceled record")


@check("cancel-malformed-data", 2, ["cancel", "storage"], "Cancel on malformed data fails without overwriting it")
def _(env):
    env.write_db(b"{ broken")
    env.refuse("cancel", "--id", "1")


# -- stage 3: summary ------------------------------------------------------------

def rooms(env):
    result = env.ok("summary")
    expect_keys(result, {"rooms"}, "summary result")
    if not isinstance(result["rooms"], list):
        fail("summary.rooms must be an array")
    for entry in result["rooms"]:
        expect_keys(entry, {"room", "active", "minutes"}, "summary entry")
        if type(entry["active"]) is not int or type(entry["minutes"]) is not int:
            fail("active and minutes must be integers, got %s" % entry)
    return result["rooms"]


@check("summary-empty", 3, ["summary", "storage"], "Empty state yields an empty rooms array")
def _(env):
    expect_equal(env.ok("summary"), {"rooms": []}, "summary of a missing database")


@check("summary-counts-minutes", 3, ["summary", "reservation"],
       "Per room: the active count and the exact minutes, in lexical room order")
def _(env):
    book(env, "B", at(9), at(9, 45))
    book(env, "A", at(10), at(11))
    book(env, "A", at(12), at(12, 30))
    expect_equal(rooms(env), [{"room": "A", "active": 2, "minutes": 90}, {"room": "B", "active": 1, "minutes": 45}],
                 "summary")


@check("summary-active-only", 3, ["summary", "cancel"], "Canceled reservations count neither as active nor as minutes")
def _(env):
    keep = book(env, "A", at(10), at(11))
    drop = book(env, "A", at(12), at(14))
    env.ok("cancel", "--id", drop["id"])
    expect_equal(rooms(env), [{"room": "A", "active": 1, "minutes": 60}], "summary")
    expect_equal(keep["status"], "active", "kept record")


@check("summary-omits-canceled-rooms", 3, ["summary", "cancel", "list"],
       "A room whose whole history is canceled is omitted, history stays listed")
def _(env):
    only = book(env, "Gone", at(10), at(11))
    live = book(env, "Live", at(10), at(11))
    env.ok("cancel", "--id", only["id"])
    expect_equal(rooms(env), [{"room": "Live", "active": 1, "minutes": 60}], "summary with a fully canceled room")
    env.ok("cancel", "--id", live["id"])
    expect_equal(env.ok("summary"), {"rooms": []}, "summary when everything is canceled")
    expect_equal(len(listing(env)), 2, "canceled history in list")


@check("summary-lexical-order", 3, ["summary"], "Rooms are in lexical order, not insertion or numeric order")
def _(env):
    for room in ("Zed", "A2", "A10", "Baker"):
        book(env, room, at(10), at(11))
    expect_equal([entry["room"] for entry in rooms(env)], ["A10", "A2", "Baker", "Zed"], "room order")


@check("summary-exact-minutes", 3, ["summary", "reservation", "inputs"],
       "Minutes are exact UTC-minute durations across midnight, days and a leap day")
def _(env):
    book(env, "Night", at(23, 30), at(1, 15, "2026-10-10"))
    book(env, "Days", "2026-10-09T08:00Z", "2026-10-12T08:01Z")
    book(env, "Leap", "2028-02-28T12:00Z", "2028-03-01T12:00Z")
    expect_equal(rooms(env), [{"room": "Days", "active": 1, "minutes": 3 * 1440 + 1},
                              {"room": "Leap", "active": 1, "minutes": 2 * 1440},
                              {"room": "Night", "active": 1, "minutes": 105}], "summary")


@check("summary-room-identity", 3, ["summary", "inputs"], "Rooms are grouped by trimmed, case-sensitive identity")
def _(env):
    book(env, " Hall ", at(8), at(9))
    book(env, "Hall", at(9), at(10))
    book(env, "hall", at(8), at(9, 30))
    found = {entry["room"]: (entry["active"], entry["minutes"]) for entry in rooms(env)}
    expect_equal(found, {"Hall": (2, 120), "hall": (1, 90)}, "rooms")


@check("summary-rebooking", 3, ["summary", "availability", "cancel"], "A rebooked interval counts once the old one is canceled")
def _(env):
    first = book(env, "A", at(10), at(11))
    env.ok("cancel", "--id", first["id"])
    book(env, "A", at(10), at(11))
    expect_equal(rooms(env), [{"room": "A", "active": 1, "minutes": 60}], "summary")


@check("summary-reads-leave-data", 3, ["summary", "storage"], "Summary never changes the stored data")
def _(env):
    book(env, "A", at(10), at(11))
    env.reads_leave_data_untouched([("summary",), ("summary",)])


@check("summary-malformed-data", 3, ["summary", "storage", "interface"],
       "Summary on malformed data fails through the envelope without overwriting it")
def _(env):
    env.write_db(b'{"reservations": [')
    env.refuse("summary")
