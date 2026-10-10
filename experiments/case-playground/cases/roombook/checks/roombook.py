"""Public checks of the roombook case; checks/acceptance.py runs them."""
import csv


def checks(ctx):
    """The checks released up to `ctx.station`, in a fixed order."""
    call, check, expect, station, db, folder = ctx.call, ctx.check, ctx.expect, ctx.station, ctx.db, ctx.folder
    key = "bookings" if station == 4 else "reservations"
    args = ("--room", " A ", "--start", "2026-10-09T10:00Z", "--end", "2026-10-09T11:00Z", "--title", "Meet,\nteam")
    check("empty", lambda: expect(call("list"), {key: []}))
    check("create-normalize-and-restart", lambda: (call("book", *args), expect(len(call("list")[key]), 1),
                                                  expect(call("list")[key][0]["room"], "A")))
    check("overlap-no-mutation", lambda: call("book", *args, error=True))
    check("adjacent-interval", lambda: call("book", "--room", "A", "--start", "2026-10-09T11:00Z",
                                           "--end", "2026-10-09T11:30Z", "--title", "Next"))
    check("invalid-calendar", lambda: call("book", "--room", "B", "--start", "2026-02-30T10:00Z",
                                           "--end", "2026-03-01T10:00Z", "--title", "Bad", error=True))
    if station >= 2:
        check("cancel-idempotence", lambda: expect(call("cancel", "--id", "1"), call("cancel", "--id", "1")))
        check("cancel-frees-room-and-no-id-reuse", lambda: expect(call("book", *args)["id"], 3))
        check("active-report", lambda: expect(call("summary"), {"rooms": [{"room": "A", "active": 2, "minutes": 90}]}))
    if station >= 3:
        check("status-room-filter", lambda: expect(len(call("list", "--status", "canceled", "--room", " A ")[key]), 1))
        check("unknown-room-report", lambda: expect(call("summary", "--room", "Unknown"), {"rooms": []}))

        def export():
            before = db.read_bytes()
            path = folder / "out.csv"
            expect(call("export", "--csv", str(path)), {"exported": 3})
            with path.open(encoding="utf-8", newline="") as handle:
                rows = list(csv.DictReader(handle))
            expect(rows[0]["title"], "Meet,\nteam")
            expect(db.read_bytes(), before)
        check("export-quotes-history-no-db-mutation", export)
    if station == 4:
        check("intentional-legacy-list", lambda: expect(call("list", "--legacy")["reservations"], call("list")["bookings"]))
