"""Command line of the depot: python -B app.py --db PATH COMMAND ..."""
import argparse
import json
import sys
from pathlib import Path

from catalog import items
from depot import store
from depot.errors import Refusal
from orders import orderbook
from reports import views
from stock import ledger


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

    add = command("add-item")
    add.add_argument("--sku", required=True)
    add.add_argument("--name", required=True)
    add.add_argument("--reorder", default="0")
    receive = command("receive")
    receive.add_argument("--sku", required=True)
    receive.add_argument("--qty", required=True)
    command("place-order").add_argument("--line", action="append", required=True)
    command("fulfil-order").add_argument("--id", required=True)
    command("cancel-order").add_argument("--id", required=True)
    command("orders").add_argument("--status")
    command("items")
    command("reorder")
    return parser


QUERIES = {
    "orders": lambda state, args: {"orders": orderbook.listing(state, args.status)},
    "items": lambda state, args: {"items": views.stock_view(state)},
    "reorder": lambda state, args: {"items": views.reorder_view(state)},
}
CHANGES = {
    "add-item": lambda state, args: items.add_item(state, args.sku, args.name, args.reorder),
    "receive": lambda state, args: ledger.receive(state, args.sku, args.qty),
    "place-order": lambda state, args: orderbook.place(state, args.line),
    "fulfil-order": lambda state, args: orderbook.fulfil(state, args.id),
    "cancel-order": lambda state, args: orderbook.cancel(state, args.id),
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
        result = run(build_parser().parse_args(argv))
    except (UsageError, Refusal, store.StoreError) as error:
        print(json.dumps({"error": str(error)}, ensure_ascii=False))
        return 2
    print(json.dumps(result, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    sys.exit(main())
