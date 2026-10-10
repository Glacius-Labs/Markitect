import unittest

from support import CliCase

HEADER = "id,title,author,pages\n"


class Import(CliCase):
    def write_csv(self, text, name="in.csv"):
        path = self.folder / name
        path.write_text(text, encoding="utf-8")
        return str(path)

    def test_appends_normalized_unread_books_in_file_order(self):
        self.add("first")
        path = self.write_csv(HEADER + 'z,"  Zed,\t Story ",Ann\u00a0 Lee,12\n\nA,Other,e\u0301mile, 3 \na,Lower,Writer,4\n')
        self.assertEqual(self.ok("import", "--csv", path), {"imported": 3})
        books = self.ok("list")["books"]
        self.assertEqual([book["id"] for book in books], ["first", "z", "A", "a"])
        self.assertEqual(books[1], {"id": "z", "title": "Zed, Story", "author": "Ann Lee", "pages": 12,
                                    "status": "unread"})
        self.assertEqual((books[2]["author"], books[2]["pages"]), ("\u00e9mile", 3))
        self.assertEqual(self.logged(), [("add", ["first"]), ("import", ["z", "A", "a"])])

    def test_header_only_imports_nothing_and_still_logs(self):
        self.assertEqual(self.ok("import", "--csv", self.write_csv(HEADER)), {"imported": 0})
        self.assertEqual(self.ok("list"), {"books": []})
        self.assertEqual(self.logged(), [("import", [])])

    def test_any_invalid_row_rejects_the_whole_file(self):
        self.add("taken")
        cases = {
            "duplicate": [HEADER + "x,T,W,1\ntaken,T,W,1\n", HEADER + "x,T,W,1\n x ,T,W,2\n",
                          HEADER + "e\u0301,T,W,1\n\u00e9,T,W,1\n"],
            "invalid_input": [HEADER + "x,T,W,1\ny,T,W,0\n", HEADER + "x,T,W,1\ny, ,W,1\n",
                              HEADER + "x,T,W,1\ny,T,W\n", HEADER + "x,T,W,1,extra\n",
                              "id,title,author\nx,T,W\n", "id,title,author,pages,status\nx,T,W,1,unread\n",
                              "", "x,T,W,1\n"],
        }
        for code, texts in cases.items():
            for text in texts:
                with self.subTest(text=text):
                    self.fails(code, "import", "--csv", self.write_csv(text))
        self.fails("invalid_input", "import", "--csv", str(self.folder / "missing.csv"))
        self.fails("invalid_input", "import")
        self.assertEqual(self.ids(), ["taken"])


if __name__ == "__main__":
    unittest.main()
