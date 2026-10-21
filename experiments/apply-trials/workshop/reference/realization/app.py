"""Command line of the bicycle workshop: python -B app.py --db PATH COMMAND ..."""
import argparse
import json
import sys
from pathlib import Path

from billing import billing
from customers import customers
from jobs import jobs
from parts import parts
from reports import reports
from shop import store
from shop.errors import Refusal

VALUE_OPTIONS = ("--db", "--name", "--id", "--sku", "--price", "--qty", "--customer", "--bike", "--job", "--minutes",
                 "--status", "--invoice")


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

    def command(name, *required, optional=()):
        sub = commands.add_parser(name, add_help=False, allow_abbrev=False)
        for option in required:
            sub.add_argument(option, required=True)
        for option in optional:
            sub.add_argument(option)

    command("add-customer", "--name")
    command("customer", "--id")
    command("add-part", "--sku", "--name", "--price")
    command("receive", "--sku", "--qty")
    command("parts")
    command("open-job", "--customer", "--bike")
    command("add-labor", "--job", "--minutes")
    command("use-part", "--job", "--sku", "--qty")
    command("close-job", "--job")
    command("jobs", optional=("--status",))
    command("invoice", "--job")
    command("pay", "--invoice")
    command("invoices", optional=("--customer",))
    command("report")
    return parser


QUERIES = {
    "customer": lambda state, args: customers.show(state, args.id),
    "parts": lambda state, args: {"parts": parts.listing(state)},
    "jobs": lambda state, args: {"jobs": jobs.listing(state, args.status)},
    "invoices": lambda state, args: {"invoices": billing.listing(state, args.customer)},
    "report": lambda state, args: reports.report(state),
}
CHANGES = {
    "add-customer": lambda state, args: customers.add_customer(state, args.name),
    "add-part": lambda state, args: parts.add_part(state, args.sku, args.name, args.price),
    "receive": lambda state, args: parts.receive(state, args.sku, args.qty),
    "open-job": lambda state, args: jobs.open_job(state, args.customer, args.bike),
    "add-labor": lambda state, args: jobs.add_labor(state, args.job, args.minutes),
    "use-part": lambda state, args: jobs.use_part(state, args.job, args.sku, args.qty),
    "close-job": lambda state, args: jobs.close_job(state, args.job),
    "invoice": lambda state, args: billing.write_invoice(state, args.job),
    "pay": lambda state, args: billing.pay(state, args.invoice),
}


def attach_values(argv):
    """Bind the token after a value option to it, so that a value may start with a hyphen."""
    attached, tokens = [], iter(argv)
    for token in tokens:
        value = next(tokens, None) if token in VALUE_OPTIONS else None
        attached.append(token if value is None else token + "=" + value)
    return attached


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
        result = run(build_parser().parse_args(attach_values(sys.argv[1:] if argv is None else argv)))
    except (UsageError, Refusal, store.StoreError) as error:
        print(json.dumps({"error": str(error)}))
        return 2
    print(json.dumps(result))
    return 0


if __name__ == "__main__":
    sys.exit(main())
