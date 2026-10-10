"""Original small Readinglog: add/list only, no backlog implementation supplied."""
import argparse
import json
from pathlib import Path
import sys


def load(path):
    books = json.loads(path.read_text(encoding="utf-8")) if path.exists() else []
    if not isinstance(books, list):
        raise ValueError("Invalid database shape")
    seen = set()
    for book in books:
        if not isinstance(book, dict) or set(book) != {"id", "title", "author", "pages", "status"}:
            raise ValueError("Invalid stored record")
        if any(not isinstance(book[key], str) or not book[key].strip() for key in ("id", "title", "author")):
            raise ValueError("Invalid stored text")
        if type(book["pages"]) is not int or book["pages"] <= 0 or book["status"] not in ("unread", "finished"):
            raise ValueError("Invalid stored value")
        if book["id"] in seen:
            raise ValueError("Duplicate stored id")
        seen.add(book["id"])
    return books


def save(path, books):
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(path.name + ".tmp")
    temporary.write_text(json.dumps(books, ensure_ascii=False) + "\n", encoding="utf-8")
    temporary.replace(path)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--db", required=True, type=Path)
    actions = parser.add_subparsers(dest="command", required=True)
    actions.add_parser("list")
    add = actions.add_parser("add")
    for field in ("id", "title", "author", "pages"):
        add.add_argument("--" + field, required=True)
    args = parser.parse_args()
    try:
        books = load(args.db)
        if args.command == "list":
            result = {"books": books}
        else:
            text = {key: getattr(args, key).strip() for key in ("id", "title", "author")}
            pages = int(args.pages)
            if not all(text.values()) or pages <= 0 or any(book["id"] == text["id"] for book in books):
                raise ValueError("Invalid or duplicate book")
            result = dict(text, pages=pages, status="unread")
            books.append(result)
            save(args.db, books)
        print(json.dumps(result, ensure_ascii=False))
        return 0
    except (ValueError, OSError, TypeError, KeyError) as error:
        print(json.dumps({"error": str(error)}))
        return 2


if __name__ == "__main__":
    sys.exit(main())
