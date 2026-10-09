# Roombook

Roombook is a small local room-booking CLI written for Python 3.13's standard library. It needs no service, network, package installation, or framework. Invoke it as a separate process:

`python -B app.py --db PATH COMMAND ...`

Commands:

- `book --room NAME --start YYYY-MM-DDTHH:MMZ --end YYYY-MM-DDTHH:MMZ --title TEXT` creates a reservation and returns it with an integer ID and `status:"active"`. Room and title are trimmed and must remain nonempty. Timestamps must be valid UTC calendar minutes in exactly the shown format, and end must be later than start.
- `list [--room NAME]` returns `{"reservations":[...]}` in start-time then ID order, including canceled history. Room matching is case-sensitive after trimming.
- `cancel --id INTEGER` returns the matching record with `status:"canceled"`. Repeating a cancellation succeeds with the same record and does not rewrite the database. An unknown ID or malformed ID fails.
- `summary` returns `{"rooms":[{"room":NAME,"active":COUNT,"minutes":TOTAL},...]}` with only rooms that have active reservations, sorted lexically by room name. Minutes are summed exactly from active reservation intervals. Empty state returns `{"rooms":[]}`.

For example, create a booking and then inspect it:

```powershell
python -B app.py --db .\roombook.json book --room Atlas --start 2026-10-09T10:00Z --end 2026-10-09T11:00Z --title Planning
python -B app.py --db .\roombook.json list --room Atlas
python -B app.py --db .\roombook.json summary
python -B app.py --db .\roombook.json cancel --id 1
```

Each successful command writes one JSON object to stdout and exits 0. Invalid input, an overlap, or a missing cancellation ID writes `{"error":"..."}` to stdout and exits 2; failed commands leave the database bytes unchanged. Intervals are half-open, so a booking may begin exactly when another ends. Other rooms do not conflict, and cancellation frees the interval while retaining its record in list history and ID allocation.

A missing database file means empty state. Read-only commands do not create it. A successful booking or cancellation writes UTF-8 JSON to a temporary file in the database directory and atomically replaces the database file. Malformed or unsupported stored data fails clearly and is not overwritten. IDs increase from the largest stored ID and are not reused.

Run the regression suite with `python -B -m unittest discover -s tests -v`; run the public station checks with `python -B checks/acceptance.py --repo . --case roombook --station 2`. Concurrent independent writers are outside the contract: the CLI does not coordinate simultaneous processes and is not a distributed booking system. Keep the database on a local filesystem with one writer at a time.

The finite backlog and station release are documented in `BACKLOG.md` and `STATIONS.json`.
