"""Published finite CLI contract checks. Run per frozen station; never a complete verdict."""
import argparse
import csv
import json
from pathlib import Path
import subprocess
import sys
import tempfile


class CandidateFailure(RuntimeError):
    """The application failed the public process/output contract."""


def invoke(repo, db, args, error=False):
    before = db.read_bytes() if db.exists() else None
    try:
        proc = subprocess.run([sys.executable, "-B", str(repo / "app.py"), "--db", str(db), *args],
                              capture_output=True, timeout=15, cwd=repo)
    except subprocess.TimeoutExpired as exc:
        raise CandidateFailure(f"application timed out: {args}") from exc
    expected = 2 if error else 0
    stdout = proc.stdout.decode("utf-8", errors="replace")
    stderr = proc.stderr.decode("utf-8", errors="replace")
    if proc.returncode != expected:
        raise CandidateFailure(f"exit {proc.returncode}, expected {expected}; args={args}; "
                               f"stderr={stderr!r}; stdout={stdout!r}")
    try:
        value = json.loads(proc.stdout.decode("utf-8"))
    except (ValueError, UnicodeError) as exc:
        raise CandidateFailure(f"invalid UTF-8 JSON response; args={args}; "
                               f"exit={proc.returncode}; stderr={stderr!r}; stdout={stdout!r}") from exc
    if not isinstance(value, dict):
        raise CandidateFailure(f"response must be a JSON object: {args}")
    if error:
        if "error" not in value:
            raise CandidateFailure(f"expected error object: {args}")
        if (db.read_bytes() if db.exists() else None) != before:
            raise CandidateFailure(f"failed command mutated stored state: {args}")
    return value


STATION_COUNTS = {"roombook": 4, "readinglog": 4, "readinglog2": 6}


def read_output(path, kind):
    """Parse a file the application wrote: "json", "jsonl" or "csv"."""
    if not path.is_file():
        raise CandidateFailure(f"{path.name} was not written")
    try:
        text = path.read_text(encoding="utf-8")
        if kind == "csv":
            return list(csv.reader(text.splitlines(keepends=True)))
        if kind == "jsonl":
            return [json.loads(line) for line in text.splitlines() if line.strip()]
        return json.loads(text)
    except (ValueError, csv.Error) as exc:
        raise CandidateFailure(f"{path.name} is not valid UTF-8 {kind}: {exc}") from exc


def readinglog2_checks(repo, folder, db, call, check, expect, station):
    """Visible checks for the six-wave readinglog2 case. They sample the README's
    project rules; they do not cover them exhaustively."""
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


def assess(repo, case, station):
    findings = []
    if not (repo / "app.py").is_file():
        return {"case": case, "station": station, "findings": [
            {"check": "application-entrypoint", "status": "FAIL", "detail": "required app.py is missing"},
            {"check": "released-functional-checks", "status": "NOT RUN", "detail": "application cannot launch"}],
            "limitation": "Missing entrypoint; individual functional checks did not execute."}
    with tempfile.TemporaryDirectory(prefix="playground-public-check-") as folder:
        db = Path(folder) / "state.json"
        def call(*args, error=False):
            return invoke(repo, db, args, error=error)
        def check(name, body):
            try:
                body()
                findings.append({"check": name, "status": "PASS"})
            except (AssertionError, CandidateFailure, KeyError, TypeError) as error:
                findings.append({"check": name, "status": "FAIL", "detail": str(error)})
            except Exception as error:
                findings.append({"check": name, "status": "EVALUATION_ERROR", "detail": str(error)})
        def expect(actual, wanted):
            assert actual == wanted, (actual, wanted)
        def malformed():
            saved = db.read_bytes() if db.exists() else None
            try:
                db.write_text("{broken", encoding="utf-8")
                call("list", error=True)
            finally:
                if saved is None:
                    db.unlink(missing_ok=True)
                else:
                    db.write_bytes(saved)
        if case == "roombook":
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
                    path = Path(folder) / "out.csv"
                    expect(call("export", "--csv", str(path)), {"exported": 3})
                    with path.open(encoding="utf-8", newline="") as handle:
                        rows = list(csv.DictReader(handle))
                    expect(rows[0]["title"], "Meet,\nteam")
                    expect(db.read_bytes(), before)
                check("export-quotes-history-no-db-mutation", export)
            if station == 4:
                check("intentional-legacy-list", lambda: expect(call("list", "--legacy")["reservations"], call("list")["bookings"]))
        elif case == "readinglog2":
            readinglog2_checks(repo, Path(folder), db, call, check, expect, station)
        else:
            key = "entries" if station == 4 else "books"
            check("legacy-add-restart", lambda: (call("add", "--id", " a ", "--title", "Story,\npart",
                                                     "--author", " Writer ", "--pages", "120"),
                                                 expect(call("list")[key][0]["id"], "a")))
            check("duplicate-no-mutation", lambda: call("add", "--id", "a", "--title", "X", "--author", "Y", "--pages", "1", error=True))
            check("finish-idempotence", lambda: expect(call("finish", "--id", "a"), call("finish", "--id", "a")))
            check("status-filter-regression", lambda: expect(call("list", "--status", "unread"), {key: []}))
            if station >= 2:
                path = Path(folder) / "in.csv"
                path.write_text("id,title,author,pages\nb,Second,Writer,20\n", encoding="utf-8")
                check("valid-import", lambda: expect(call("import", "--csv", str(path)), {"imported": 1}))
                path_bad = Path(folder) / "bad.csv"
                path_bad.write_text("id,title,author,pages\nc,Third,Writer,20\na,Duplicate,Writer,30\n", encoding="utf-8")
                check("import-all-or-nothing", lambda: call("import", "--csv", str(path_bad), error=True))
                count_key = "entries" if station == 4 else "books"
                check("summary-count-pages", lambda: expect(call("summary"), {"authors": [{"author": "Writer", count_key: 2, "finished": 1, "pages": 140}]}))
            if station >= 3:
                check("composed-list-filter", lambda: expect(len(call("list", "--author", " Writer ", "--status", "finished")[key]), 1))
                check("unknown-author-summary", lambda: expect(call("summary", "--author", "unknown"), {"authors": []}))
                def export():
                    before = db.read_bytes()
                    path_out = Path(folder) / "out.csv"
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
        check("malformed-db-preserved", malformed)
    return {"case": case, "station": station, "findings": findings,
            "limitation": "Finite public checks; assessor must add meaningful cases and semantic review. Cascading failures retain individual evidence."}


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", type=Path, default=Path.cwd())
    parser.add_argument("--case", choices=list(STATION_COUNTS), required=True)
    parser.add_argument("--station", type=int, choices=range(1, max(STATION_COUNTS.values()) + 1), required=True)
    args = parser.parse_args()
    if args.station > STATION_COUNTS[args.case]:
        parser.error(f"case {args.case} has stations 1 to {STATION_COUNTS[args.case]}")
    result = assess(args.repo.resolve(), args.case, args.station)
    print(json.dumps(result, indent=2, ensure_ascii=False))
    sys.exit(1 if any(f["status"] != "PASS" for f in result["findings"]) else 0)
