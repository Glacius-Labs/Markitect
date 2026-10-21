"""Command line of the room booking hall: python -B app.py --db PATH COMMAND ..."""
import argparse
import json
import sys
from pathlib import Path

from bookings import bookings
from hall import store
from hall.errors import Refusal
from rooms import rooms


class UsageError(Exception):
    """The command line itself is invalid."""


class Parser(argparse.ArgumentParser):
    """Reports usage errors as exceptions instead of printing and exiting."""

    def error(self, message):
        raise UsageError(message)


def build_parser():
    parser = Parser(prog="app.py", add_help=False, allow_abbrev=False)
    parser.add_argument("--db", required=True)
    commands = parser.add_subparsers(dest="command", required=True, parser_class=Parser)

    def command(name):
        return commands.add_parser(name, add_help=False, allow_abbrev=False)

    add = command("add-room")
    add.add_argument("--code", required=True)
    add.add_argument("--seats", required=True)
    command("rooms")
    book = command("book")
    book.add_argument("--code", required=True)
    book.add_argument("--seats", required=True)
    command("bookings").add_argument("--code")
    return parser


QUERIES = {
    "rooms": lambda state, args: {"rooms": rooms.listing(state)},
    "bookings": lambda state, args: {"bookings": bookings.listing(state, args.code)},
}
CHANGES = {
    "add-room": lambda state, args: rooms.add_room(state, args.code, args.seats),
    "book": lambda state, args: bookings.book(state, args.code, args.seats),
}


def run(args):
    path = Path(args.db)
    state = store.load(path)
    if args.command in QUERIES:
        return QUERIES[args.command](state, args)
    result = CHANGES[args.command](state, args)
    store.save(path, state)
    return result


def main(argv=None):
    try:
        result = run(build_parser().parse_args(sys.argv[1:] if argv is None else argv))
    except (UsageError, Refusal, store.StoreError) as error:
        print(json.dumps({"error": str(error)}))
        return 2
    print(json.dumps(result))
    return 0


if __name__ == "__main__":
    sys.exit(main())
