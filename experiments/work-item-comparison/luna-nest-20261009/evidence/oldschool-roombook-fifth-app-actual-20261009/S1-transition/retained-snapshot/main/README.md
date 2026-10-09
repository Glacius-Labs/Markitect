# Roombook

A small local room reservation CLI. Greenfield: no application implementation is supplied. Use Python 3.13 standard library, no service, network, package installation or framework requirement. The public contract is a process invocation, allowing free internal structure:

`python -B app.py --db PATH COMMAND ...`

The current S1 executable surface is `book` and `list`; `cancel` and `summary` are later backlog work. Each successful command writes one JSON object to stdout and exits 0. Invalid input or a business conflict writes `{"error":"..."}` to stdout, exits 2, and leaves stored state unchanged. Error wording is free. Missing data file represents empty state; malformed existing data fails clearly without overwriting it. Use a durable UTF-8 JSON file and an atomic replace on successful mutation. A later process must observe committed data.

Commands:

- `book --room NAME --start YYYY-MM-DDTHH:MMZ --end YYYY-MM-DDTHH:MMZ --title TEXT`: returns a reservation with integer `id`, room, start, end, title and `status:"active"`. Strip room/title surrounding whitespace; nonempty required. Timestamps are exactly UTC minutes and end must be later than start. IDs start at 1, increase and are never reused.
- `list [--room NAME]`: returns `{"reservations":[...]}` sorted by start then id, including canceled records. Room identities are case-sensitive after trimming. A room filter uses the same normalization.
- `cancel --id INTEGER`: returns the existing record with `status:"canceled"`. Canceling a canceled record is an idempotent success; missing ID fails.
- `summary`: returns `{"rooms":[{"room":NAME,"active":COUNT,"minutes":TOTAL},...]}` in lexical room order. Count only active reservations and sum their exact minute durations. Empty state gives an empty list.

For one room, active reservations must not overlap; intervals are half-open, so an end equal to the next start is permitted. Other rooms do not conflict. Canceled reservations no longer block the room. Do not silently reinterpret timezone offsets or invalid calendar values. Concurrent independent writers are outside this small contract; state that limitation rather than claiming a distributed booking system.

Run the application with Python 3.13: `python -B app.py --db ./roombook.json list`. Run the S1 CLI regression suite with `python -B -m unittest discover -s tests -v`. The JSON file is created only by a successful booking, and writes use a temporary file in the same directory followed by an atomic replacement. Reads do not rewrite or migrate the file. Independent concurrent writers are not coordinated.

The backlog is in BACKLOG.md. Keep README and meaningful tests current; document run/test commands and data behavior. Merge completed work to this isolated repository's main.
