"""Readinglog checks. Each cites the Definitions of the snapshot it checks.

Stage 0 is 00-baseline (add and list), stage 1 adds finish and the status
filter, stage 2 adds import and stage 3 adds summary. The stored data is the
UTF-8 JSON array of the storage Definition, so malformed-data checks seed that
file directly. Error wording is free; only the envelope, exit code and
unchanged data are checked. The supplied baseline application fails the check
`missing-option` at stage 0, as the guide records (its argument parser prints a
usage error instead of the JSON envelope); the check stays because the
Definitions require the envelope for every input error.
"""
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

from oracle_kit import Registry, expect_equal, expect_keys, fail  # noqa: E402

REG = Registry("readinglog", "readinglog",
               {0: "00-baseline", 1: "01-finish-status", 2: "02-import", 3: "03-summary"})
check = REG.check

FIELDS = {"id", "title", "author", "pages", "status"}


def add(env, ident, title="Title", author="Author", pages="10", refuse=False):
    args = ["add", "--id", ident, "--title", title, "--author", author, "--pages", pages]
    return env.refuse(*args) if refuse else env.ok(*args)


def books(env, *extra):
    result = env.ok("list", *extra)
    expect_keys(result, {"books"}, "list result")
    if not isinstance(result["books"], list):
        fail("list.books must be an array")
    return result["books"]


def book_record(record, label, **expected):
    expect_keys(record, FIELDS, label)
    if type(record["pages"]) is not int:
        fail("%s: pages must be an integer, got %s" % (label, record["pages"]))
    for key, value in expected.items():
        expect_equal(record[key], value, "%s.%s" % (label, key))


def seed(env, records):
    env.write_db(json.dumps(records, ensure_ascii=False))


def stored(ident="a", title="T", author="A", pages=1, status="unread"):
    return {"id": ident, "title": title, "author": author, "pages": pages, "status": status}


# -- stage 0: the existing add/list application -----------------------------------

@check("list-empty", 0, ["list", "storage"], "A missing data file lists as an empty array and is not created")
def _(env):
    expect_equal(env.ok("list"), {"books": []}, "list on a missing database")
    if env.db.exists():
        fail("a read created the missing data file")


@check("add-record", 0, ["add", "book", "interface"], "Add returns the created unread record with exactly the book fields")
def _(env):
    book_record(add(env, "b1", "Dune", "Herbert", "412"), "added record",
                id="b1", title="Dune", author="Herbert", pages=412, status="unread")


@check("add-normalizes", 0, ["inputs", "add"], "Text input is stripped before it is stored")
def _(env):
    record = add(env, "  b1  ", "  Dune  ", "  Herbert ", "5")
    book_record(record, "added record", id="b1", title="Dune", author="Herbert")
    expect_equal(books(env), [record], "listing")


@check("insertion-order", 0, ["list", "storage", "add"], "Listing keeps insertion order across processes")
def _(env):
    records = [add(env, ident) for ident in ("c", "a", "b")]
    expect_equal(books(env), records, "listing")
    env.stored_json()


@check("duplicate-id", 0, ["add", "storage", "inputs"],
       "A duplicate id, also after normalization, is refused without replacing the record")
def _(env):
    first = add(env, "a", "First")
    add(env, "a", "Second", refuse=True)
    add(env, "  a ", "Third", refuse=True)
    expect_equal(books(env), [first], "listing")


@check("ids-case-sensitive", 0, ["inputs", "book"], "Ids are case-sensitive; titles and authors need not be unique")
def _(env):
    add(env, "a", "Same", "Same")
    add(env, "A", "Same", "Same")
    expect_equal([item["id"] for item in books(env)], ["a", "A"], "ids")


@check("add-invalid-input", 0, ["inputs", "add", "book"], "Empty text and non-positive or non-numeric pages are refused")
def _(env):
    for ident, title, author, pages in (("", "T", "A", "1"), ("   ", "T", "A", "1"), ("a", "", "A", "1"),
                                        ("a", "  ", "A", "1"), ("a", "T", "", "1"), ("a", "T", " ", "1"),
                                        ("a", "T", "A", "0"), ("a", "T", "A", "-3"), ("a", "T", "A", "abc"),
                                        ("a", "T", "A", "1.5"), ("a", "T", "A", ""), ("a", "T", "A", "true")):
        add(env, ident, title, author, pages, refuse=True)
    expect_equal(books(env), [], "listing after every refusal")


@check("missing-option", 0, ["inputs", "interface"], "A missing required option is an input error in the JSON envelope")
def _(env):
    env.refuse("add", "--id", "a", "--title", "T", "--author", "A")
    env.refuse("add")


@check("text-roundtrip", 0, ["storage", "inputs", "add"], "Non-ASCII text, commas and newlines survive storage")
def _(env):
    record = add(env, "é", "Story,\npart ☕ \"q\"", "Zöe", "120")
    expect_equal(books(env), [record], "listing")
    env.stored_json()


@check("stored-shape", 0, ["storage", "book"], "The data file is a UTF-8 JSON array of exactly the book records")
def _(env):
    first, second = add(env, "a"), add(env, "b")
    expect_equal(env.stored_json(), [first, second], "stored array")


@check("reads-leave-data", 0, ["list", "storage"], "Listing never rewrites valid stored data, whatever its formatting")
def _(env):
    env.write_db(json.dumps([stored("x"), stored("y", pages=7)], indent=4, ensure_ascii=False) + "\n\n")
    env.reads_leave_data_untouched([("list",), ("list",)])


@check("malformed-json", 0, ["storage", "interface"], "Unparseable stored data fails through the envelope and is kept")
def _(env):
    for content in (b"[{", b"not json", b"\xff\xfe\x00 bad bytes"):
        env.write_db(content)
        env.refuse("list")
        add(env, "a", refuse=True)


@check("malformed-records", 0, ["storage", "book"],
       "Stored data that violates the book shape or value rules fails through the envelope and is kept")
def _(env):
    bad = [
        {"not": "an array"},
        [stored(), stored()],  # duplicate ids
        [dict(stored(), extra=1)],
        [{key: value for key, value in stored().items() if key != "author"}],
        [stored(title="   ")],
        [stored(ident="")],
        [stored(author=7)],
        [stored(pages=0)],
        [stored(pages=-2)],
        [stored(pages=True)],
        [stored(pages="5")],
        [stored(pages=1.5)],
        [stored(status="done")],
        ["a string"],
    ]
    for records in bad:
        seed(env, records)
        env.refuse("list")
        add(env, "zzz", refuse=True)


@check("atomic-replacement", 0, ["storage"], "A mutation replaces the file atomically and leaves no temporary file")
def _(env):
    add(env, "a")
    first = env.stat()[2]
    add(env, "b")
    if env.stat()[2] == first:
        fail("the data file kept its identity: the second mutation did not replace it atomically")
    if env.leftover_temp_files():
        fail("temporary files left next to the database: %s" % env.leftover_temp_files())


# -- stage 1: finish and status filter -------------------------------------------

@check("finish-record", 1, ["finish", "book", "interface"], "Finish returns the record as finished, other fields unchanged")
def _(env):
    first = add(env, "a", "Dune", "Herbert", "412")
    other = add(env, "b")
    done = env.ok("finish", "--id", "a")
    expect_equal(done, dict(first, status="finished"), "finished record")
    expect_equal(books(env), [done, other], "listing keeps order and other records")


@check("finish-idempotent", 1, ["finish"], "Repeated finish succeeds with the same record and unchanged data")
def _(env):
    add(env, "a")
    once = env.ok("finish", "--id", "a")
    before = env.raw()
    expect_equal(env.ok("finish", "--id", "a"), once, "repeated finish")
    if env.raw() != before:
        fail("repeating finish changed the stored data")


@check("finish-id-identity", 1, ["finish", "inputs"], "The id is normalized and compared case-sensitively")
def _(env):
    add(env, "a")
    add(env, "B")
    env.ok("finish", "--id", "  a ")
    env.refuse("finish", "--id", "A")
    env.refuse("finish", "--id", "b")
    expect_equal([item["status"] for item in books(env)], ["finished", "unread"], "statuses")


@check("finish-bad-id", 1, ["finish", "interface"], "Unknown, empty and missing ids fail without changing data")
def _(env):
    add(env, "a")
    for bad in ("zzz", "", "   "):
        env.refuse("finish", "--id", bad)
    env.refuse("finish")


@check("finish-malformed-data", 1, ["finish", "storage"], "Finish on malformed data fails without overwriting it")
def _(env):
    env.write_db(b"[ broken")
    env.refuse("finish", "--id", "a")


@check("status-filter", 1, ["list"], "The status filter selects that exact status in insertion order")
def _(env):
    for ident in ("c", "a", "b", "d"):
        add(env, ident)
    env.ok("finish", "--id", "a")
    env.ok("finish", "--id", "d")
    expect_equal([item["id"] for item in books(env, "--status", "finished")], ["a", "d"], "finished")
    expect_equal([item["id"] for item in books(env, "--status", "unread")], ["c", "b"], "unread")
    expect_equal(len(books(env)), 4, "unfiltered listing keeps every status")


@check("status-filter-empty", 1, ["list"], "A filter without matches returns the empty books array")
def _(env):
    add(env, "a")
    expect_equal(env.ok("list", "--status", "finished"), {"books": []}, "finished filter")
    expect_equal(env.ok("list", "--status", "unread", ), {"books": books(env)}, "unread filter equals the list")


@check("status-filter-invalid", 1, ["list", "interface"], "An invalid status filter is an input error")
def _(env):
    add(env, "a")
    for bad in ("done", "", "FINISHED", "Unread", "all"):
        env.refuse("list", "--status", bad)


@check("status-filter-reads-leave-data", 1, ["list", "storage"], "Filtered listing never changes the stored data")
def _(env):
    add(env, "a")
    env.ok("finish", "--id", "a")
    env.reads_leave_data_untouched([("list", "--status", "finished"), ("list", "--status", "unread")])


# -- stage 2: import --------------------------------------------------------------

HEADER = "id,title,author,pages\n"


def imported(env, path):
    """The `imported` count of an import; the model only says an object with imported."""
    result = env.ok("import", "--csv", path)
    if type(result.get("imported")) is not int:
        fail("import must return an object with an integer `imported`, got %s" % result)
    return result["imported"]


@check("import-valid", 2, ["import", "book", "interface"],
       "Import appends unread books in input order after the existing ones and reports the count")
def _(env):
    first = add(env, "z", "Existing")
    path = env.write_input("in.csv", HEADER + "b,Second,Writer,20\na,First,Other,5\n")
    expect_equal(imported(env, path), 2, "imported count")
    after = books(env)
    expect_equal(after[0], first, "existing record")
    book_record(after[1], "first imported", id="b", title="Second", author="Writer", pages=20, status="unread")
    book_record(after[2], "second imported", id="a", title="First", author="Other", pages=5, status="unread")
    expect_equal(len(after), 3, "listing")
    env.stored_json()


@check("import-into-missing-data", 2, ["import", "storage"], "Import into a missing data file starts from empty state")
def _(env):
    path = env.write_input("in.csv", HEADER + "a,T,A,1\n")
    expect_equal(imported(env, path), 1, "imported count")
    expect_equal(len(books(env)), 1, "listing")


@check("import-header-only", 2, ["import"], "A file with only the valid header imports zero and changes nothing")
def _(env):
    add(env, "a")
    before = env.raw()
    expect_equal(imported(env, env.write_input("in.csv", HEADER)), 0, "imported count")
    expect_equal(len(books(env)), 1, "listing")
    if env.raw() != before:
        fail("importing zero rows changed the stored data")


@check("import-normalizes", 2, ["import", "inputs"], "Imported text is normalized like add")
def _(env):
    path = env.write_input("in.csv", HEADER + "  a ,  Title ,  Writer  ,7\n")
    env.ok("import", "--csv", path)
    book_record(books(env)[0], "imported", id="a", title="Title", author="Writer", pages=7)


@check("import-exact-header", 2, ["import"], "The header must be exactly id,title,author,pages")
def _(env):
    add(env, "keep")
    for name, content in (("order", "title,id,author,pages\nb,T,A,1\n"), ("missing", "id,title,author\nb,T,A\n"),
                          ("extra", "id,title,author,pages,note\nb,T,A,1,x\n"),
                          ("extra-header-only", "id,title,author,pages,note\nb,T,A,1\n"),
                          ("extra-first", "note,id,title,author,pages\nx,b,T,A,1\n"),("names", "id,name,author,pages\nb,T,A,1\n"),
                          ("case", "ID,title,author,pages\nb,T,A,1\n"), ("empty-file", ""),
                          ("rows-only", "b,T,A,1\n")):
        env.refuse("import", "--csv", env.write_input(name + ".csv", content))
    expect_equal(len(books(env)), 1, "listing")


@check("import-all-or-nothing", 2, ["import", "inputs", "storage"],
       "Any invalid row rejects the whole import without changing data")
def _(env):
    add(env, "keep")
    rows = ("b,T,A,0", "b,T,A,-1", "b,T,A,x", "b,T,A,1.5", "b,T,A,", ",T,A,1", "b,,A,1", "b,T,,1", "b, ,A,1",
            "b,T,A", "b,T,A,1,extra", "keep,T,A,1")
    for number, row in enumerate(rows):
        content = HEADER + "ok%d,Good,Writer,3\n%s\n" % (number, row)
        env.refuse("import", "--csv", env.write_input("bad%d.csv" % number, content))
    expect_equal(len(books(env)), 1, "listing")


@check("import-duplicates", 2, ["import", "inputs"], "Duplicate ids within the file or against stored books reject everything")
def _(env):
    add(env, "a")
    env.refuse("import", "--csv", env.write_input("one.csv", HEADER + "b,T,A,1\nb,U,A,2\n"))
    env.refuse("import", "--csv", env.write_input("two.csv", HEADER + "b,T,A,1\n  b ,U,A,2\n"))
    env.refuse("import", "--csv", env.write_input("three.csv", HEADER + "c,T,A,1\n a ,U,A,2\n"))
    expect_equal(len(books(env)), 1, "listing")


@check("import-ids-case-sensitive", 2, ["import", "inputs"], "Imported ids differing only in case are distinct")
def _(env):
    expect_equal(imported(env, env.write_input("in.csv", HEADER + "a,T,A,1\nA,T,A,1\n")), 2, "imported count")


@check("import-csv-quoting", 2, ["import"], "Quoted commas and newlines, UTF-8 text and CRLF line ends are read as CSV")
def _(env):
    content = 'id,title,author,pages\r\na,"Story,\npart ☕","Zöe ""Z""",12\r\nb,Plain,Writer,3\r\n'
    path = env.write_input("in.csv", content.encode("utf-8"))
    expect_equal(imported(env, path), 2, "imported count")
    first = books(env)[0]
    expect_equal((first["title"], first["author"]), ("Story,\npart ☕", 'Zöe "Z"'), "quoted fields")


@check("import-bad-file", 2, ["import", "interface"], "A missing file, a directory or invalid UTF-8 fails through the envelope")
def _(env):
    add(env, "keep")
    env.refuse("import", "--csv", str(env.inputs / "absent.csv"))
    env.refuse("import", "--csv", str(env.inputs))
    env.refuse("import", "--csv", env.write_input("latin.csv", HEADER.encode() + b"a,Caf\xe9,A,1\n"))
    env.refuse("import")
    expect_equal(len(books(env)), 1, "listing")


@check("import-malformed-data", 2, ["import", "storage"], "Import on malformed data fails without overwriting it")
def _(env):
    env.write_db(b"[ broken")
    env.refuse("import", "--csv", env.write_input("in.csv", HEADER + "a,T,A,1\n"))


@check("import-then-finish", 2, ["import", "finish", "list"], "Imported books behave like added ones")
def _(env):
    env.ok("import", "--csv", env.write_input("in.csv", HEADER + "a,T,A,1\nb,T,A,2\n"))
    env.ok("finish", "--id", "b")
    expect_equal([(item["id"], item["status"]) for item in books(env)], [("a", "unread"), ("b", "finished")], "listing")
    add(env, "a", refuse=True)


# -- stage 3: summary -------------------------------------------------------------

def authors(env):
    result = env.ok("summary")
    if not isinstance(result.get("authors"), list):
        fail("summary.authors must be an array")
    for entry in result["authors"]:
        expect_keys(entry, {"author", "books", "finished", "pages"}, "summary entry")
        if any(type(entry[key]) is not int for key in ("books", "finished", "pages")):
            fail("books, finished and pages must be integers, got %s" % entry)
    return result["authors"]


@check("summary-empty", 3, ["summary", "storage"], "Empty state yields an empty authors array and creates no file")
def _(env):
    expect_equal(authors(env), [], "summary of a missing database")
    if env.db.exists():
        fail("summary created the missing data file")


@check("summary-totals", 3, ["summary", "book"],
       "Per author: all books, the finished ones and all pages, in lexical author order")
def _(env):
    add(env, "1", author="Zed", pages="100")
    add(env, "2", author="Adams", pages="30")
    add(env, "3", author="Zed", pages="20")
    add(env, "4", author="Adams", pages="5")
    env.ok("finish", "--id", "3")
    env.ok("finish", "--id", "4")
    expect_equal(authors(env), [{"author": "Adams", "books": 2, "finished": 1, "pages": 35},
                                {"author": "Zed", "books": 2, "finished": 1, "pages": 120}], "summary")


@check("summary-lexical-order", 3, ["summary"], "Authors are in lexical order, not insertion or numeric order")
def _(env):
    for number, author in enumerate(("Zed", "A2", "A10", "Baker")):
        add(env, str(number), author=author)
    expect_equal([entry["author"] for entry in authors(env)], ["A10", "A2", "Baker", "Zed"], "author order")


@check("summary-author-identity", 3, ["summary", "inputs"], "Authors are grouped by their stored (trimmed), case-sensitive value")
def _(env):
    add(env, "1", author="  Writer ", pages="1")
    add(env, "2", author="Writer", pages="2")
    add(env, "3", author="writer", pages="4")
    found = {entry["author"]: (entry["books"], entry["pages"]) for entry in authors(env)}
    expect_equal(found, {"Writer": (2, 3), "writer": (1, 4)}, "authors")


@check("summary-after-import-and-finish", 3, ["summary", "import", "finish"], "Summary reflects imported and finished books")
def _(env):
    add(env, "1", author="W", pages="10")
    env.ok("import", "--csv", env.write_input("in.csv", HEADER + "2,T,W,20\n3,T,V,5\n"))
    env.ok("finish", "--id", "2")
    expect_equal(authors(env), [{"author": "V", "books": 1, "finished": 0, "pages": 5},
                                {"author": "W", "books": 2, "finished": 1, "pages": 30}], "summary")


@check("summary-reads-leave-data", 3, ["summary", "storage"], "Summary never rewrites the stored data")
def _(env):
    env.write_db(json.dumps([stored("x"), stored("y", pages=7, status="finished")], indent=3) + "\n")
    env.reads_leave_data_untouched([("summary",), ("summary",)])


@check("summary-malformed-data", 3, ["summary", "storage", "interface"],
       "Summary on malformed data fails through the envelope without overwriting it")
def _(env):
    env.write_db(b"[ broken")
    env.refuse("summary")
    seed(env, [stored(pages=0)])
    env.refuse("summary")
