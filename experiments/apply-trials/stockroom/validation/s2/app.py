"""Command line of the stockroom: python -B app.py --db PATH COMMAND ..."""
import argparse
import json
import sys
from pathlib import Path

import stockroom
import store


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
    add = commands.add_parser("add-item", add_help=False, allow_abbrev=False)
    add.add_argument("--sku", required=True)
    add.add_argument("--name", required=True)
    add.add_argument("--reorder", default="0")
    for name in ("receive", "issue"):
        movement = commands.add_parser(name, add_help=False, allow_abbrev=False)
        movement.add_argument("--sku", required=True)
        movement.add_argument("--qty", required=True)
    listing = commands.add_parser("list", add_help=False, allow_abbrev=False)
    listing.add_argument("--low", action="store_true")
    return parser


def run(args):
    path = Path(args.db)
    data = store.load(path)
    if args.command == "list":
        return {"items": stockroom.list_items(data, low=args.low)}
    if args.command == "add-item":
        record = stockroom.add_item(data, args.sku, args.name, args.reorder)
    elif args.command == "receive":
        record = stockroom.receive(data, args.sku, args.qty)
    else:
        record = stockroom.issue(data, args.sku, args.qty)
    store.save(path, data)
    return record


def main(argv=None):
    try:
        result = run(build_parser().parse_args(argv))
    except (UsageError, stockroom.StockError, store.StoreError) as error:
        print(json.dumps({"error": str(error)}))
        return 2
    print(json.dumps(result))
    return 0


if __name__ == "__main__":
    sys.exit(main())
