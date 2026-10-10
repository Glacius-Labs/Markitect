"""Public checks of the six-wave readinglog2 case; checks/acceptance.py runs them.

They sample the README's project rules; they do not cover them exhaustively."""


def checks(ctx):
    """The checks released up to `ctx.station`, in a fixed order."""
    repo, folder, db, call, check, expect, station = (ctx.repo, ctx.folder, ctx.db, ctx.call, ctx.check, ctx.expect,
                                                      ctx.station)
    invoke, read_output = ctx.invoke, ctx.read_output
    key = "entries" if station == 6 else "books"
    header = ["id", "title", "author", "pages", "status", "tags"]

    def run(path, *args, error=False):
        return invoke(repo, path, args, error=error)

    def fails(code, *args):
        expect(call(*args, error=True)["error"]["code"], code)

    def ids(*args):
        return [book["id"] for book in call("list", *args)[key]]

    def add_normalize_restart():
        call("add", "--id", " a ", "--title", 'Story,  "quoted"', "--author", " Ann \t Lee ", "--pages", "120")
        books = call("list")[key]
        expect([{name: book[name] for name in header[:5]} for book in books],
               [{"id": "a", "title": 'Story, "quoted"', "author": "Ann Lee", "pages": 120, "status": "unread"}])
    check("add-normalize-restart", add_normalize_restart)
    check("duplicate-error-code", lambda: fails("duplicate", "add", "--id", "a", "--title", "X", "--author", "Y",
                                                "--pages", "1"))
    check("audit-log-add", lambda: expect(
        [(line["op"], line["ids"]) for line in read_output(db.with_name(db.name + ".log"), "jsonl")], [("add", ["a"])]))

    def finish_idempotence():
        first = call("finish", "--id", "a")
        expect(first["status"], "finished")
        expect(call("finish", "--id", "a"), first)
    check("finish-idempotence", finish_idempotence)
    check("finish-unknown-id", lambda: fails("not_found", "finish", "--id", "missing"))
    check("status-filter", lambda: (expect(ids("--status", "unread"), []), expect(ids("--status", "finished"), ["a"])))

    if station >= 2:
        good, bad = folder / "in.csv", folder / "bad.csv"
        good.write_text("id,title,author,pages\nb,Second,Ann Lee,20\n", encoding="utf-8")
        bad.write_text("id,title,author,pages\nc,Third,Ann Lee,20\na,Again,Ann Lee,30\n", encoding="utf-8")
        check("valid-import", lambda: expect(call("import", "--csv", str(good)), {"imported": 1}))
        check("import-all-or-nothing", lambda: fails("duplicate", "import", "--csv", str(bad)))
        author = {"author": "Ann Lee", key: 2, "finished": 1, "pages": 140}
        if station >= 5:
            author["unknownPages"] = 0
        check("summary-count-pages", lambda: expect(call("summary"), {"authors": [author]}))

    if station >= 3:
        check("composed-list-filter", lambda: expect(ids("--author", " Ann  Lee ", "--status", "finished"), ["a"]))
        check("unknown-author-summary", lambda: expect(call("summary", "--author", "Unknown"), {"authors": []}))

        def tag_and_filter():
            expect(call("tag", "--id", "a", "--add", " classic ")["tags"], ["classic"])
            expect(ids("--tag", "classic", "--status", "finished"), ["a"])
        check("tag-and-list-filter", tag_and_filter)

        def export():
            before, target = db.read_bytes(), folder / "out.csv"
            expect(call("export", "--csv", str(target)), {"exported": 2})
            expect(read_output(target, "csv"), [header, ["a", 'Story, "quoted"', "Ann Lee", "120", "finished", "classic"],
                                                ["b", "Second", "Ann Lee", "20", "unread", ""]])
            expect(db.read_bytes(), before)
        check("export-quotes-tags-no-db-mutation", export)
        check("export-destination-error", lambda: fails("export_failed", "export", "--csv",
                                                        str(folder / "missing" / "out.csv")))

    if station >= 4:
        old = folder / "old.json"
        old.write_text('[{"id": "v1", "title": "Old", "author": "Ann Lee", "pages": 5, "status": "unread"}]\n',
                       encoding="utf-8")

        def v1_read_only():
            before = old.read_bytes()
            expect([book["id"] for book in run(old, "list")[key]], ["v1"])
            run(old, "summary")
            expect(old.read_bytes(), before)
        check("v1-file-read-only", v1_read_only)

        def first_mutation_writes_v2():
            run(old, "finish", "--id", "v1")
            data = read_output(old, "json")
            assert isinstance(data, dict) and set(data) == {"schemaVersion", "records"}, data
            expect(data["schemaVersion"], 2)
            expect([(book["id"], book["status"]) for book in data["records"]], [("v1", "finished")])
        check("first-mutation-writes-v2", first_mutation_writes_v2)

    if station >= 5:
        pages_db, pages_csv = folder / "pages.json", folder / "pages.csv"

        def optional_pages_summary():
            expect(run(pages_db, "add", "--id", "u", "--title", "Unknown", "--author", "Ann Lee")["pages"], None)
            run(pages_db, "add", "--id", "k", "--title", "Known", "--author", "Ann Lee", "--pages", "10")
            expect(run(pages_db, "summary"), {"authors": [{"author": "Ann Lee", key: 2, "finished": 0, "pages": 10,
                                                           "unknownPages": 1}]})
        check("optional-pages-summary", optional_pages_summary)

        def unknown_pages_export():
            expect(run(pages_db, "export", "--csv", str(pages_csv)), {"exported": 2})
            expect(read_output(pages_csv, "csv"), [header, ["u", "Unknown", "Ann Lee", "", "unread", ""],
                                                   ["k", "Known", "Ann Lee", "10", "unread", ""]])
        check("unknown-pages-empty-csv-cell", unknown_pages_export)

    if station == 6:
        check("intentional-legacy-list", lambda: expect(
            call("list", "--legacy", "--status", "finished", "--tag", "classic"),
            {"books": call("list", "--status", "finished", "--tag", "classic")["entries"]}))
        check("intentional-legacy-summary", lambda: expect(
            call("summary", "--legacy", "--author", "Ann Lee"),
            {"authors": [{"author": "Ann Lee", "books": 2, "finished": 1, "pages": 140, "unknownPages": 0}]}))
