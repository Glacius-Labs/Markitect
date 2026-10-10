"""Public checks of the readinglog case; checks/acceptance.py runs them."""
import csv


def checks(ctx):
    """The checks released up to `ctx.station`, in a fixed order."""
    call, check, expect, station, db, folder = ctx.call, ctx.check, ctx.expect, ctx.station, ctx.db, ctx.folder
    key = "entries" if station == 4 else "books"
    check("legacy-add-restart", lambda: (call("add", "--id", " a ", "--title", "Story,\npart",
                                             "--author", " Writer ", "--pages", "120"),
                                         expect(call("list")[key][0]["id"], "a")))
    check("duplicate-no-mutation", lambda: call("add", "--id", "a", "--title", "X", "--author", "Y", "--pages", "1", error=True))
    check("finish-idempotence", lambda: expect(call("finish", "--id", "a"), call("finish", "--id", "a")))
    check("status-filter-regression", lambda: expect(call("list", "--status", "unread"), {key: []}))
    if station >= 2:
        path = folder / "in.csv"
        path.write_text("id,title,author,pages\nb,Second,Writer,20\n", encoding="utf-8")
        check("valid-import", lambda: expect(call("import", "--csv", str(path)), {"imported": 1}))
        path_bad = folder / "bad.csv"
        path_bad.write_text("id,title,author,pages\nc,Third,Writer,20\na,Duplicate,Writer,30\n", encoding="utf-8")
        check("import-all-or-nothing", lambda: call("import", "--csv", str(path_bad), error=True))
        count_key = "entries" if station == 4 else "books"
        check("summary-count-pages", lambda: expect(call("summary"), {"authors": [{"author": "Writer", count_key: 2, "finished": 1, "pages": 140}]}))
    if station >= 3:
        check("composed-list-filter", lambda: expect(len(call("list", "--author", " Writer ", "--status", "finished")[key]), 1))
        check("unknown-author-summary", lambda: expect(call("summary", "--author", "unknown"), {"authors": []}))

        def export():
            before = db.read_bytes()
            path_out = folder / "out.csv"
            expect(call("export", "--csv", str(path_out)), {"exported": 2})
            with path_out.open(encoding="utf-8", newline="") as handle:
                rows = list(csv.DictReader(handle))
            expect(rows[0]["title"], "Story,\npart")
            expect(rows[0]["status"], "finished")
            expect(db.read_bytes(), before)
        check("export-quotes-status-no-db-mutation", export)
    if station == 4:
        check("intentional-legacy-list", lambda: expect(call("list", "--legacy")["books"], call("list")["entries"]))
        check("intentional-legacy-summary", lambda: expect(call("summary", "--legacy")["authors"][0]["books"], 2))
