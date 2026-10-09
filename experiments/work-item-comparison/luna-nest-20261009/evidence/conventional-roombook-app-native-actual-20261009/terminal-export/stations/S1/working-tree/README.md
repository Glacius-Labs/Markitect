# Roombook

Roombook is a local command-line booking book. It stores bookings in a UTF-8
JSON file and needs only Python 3.13's standard library.

## Run

```powershell
python -B app.py --db .\roombook.json book --room "Room A" --start 2026-10-09T10:00Z --end 2026-10-09T11:00Z --title "Planning"
python -B app.py --db .\roombook.json list
python -B app.py --db .\roombook.json list --room "Room A"
```

Each successful command prints one JSON object to stdout and exits with status
0. A failed request prints `{"error":"..."}` to stdout and exits with status
2. `book` trims the room and title and rejects either when empty. Times must be
valid calendar timestamps in the exact UTC-minute form `YYYY-MM-DDTHH:MMZ`,
and the end must be later than the start. Room names are case-sensitive after
trimming.

Active bookings in the same room use half-open intervals: a booking may start
when another ends, while any actual overlap is rejected. Different rooms may
overlap. `list` includes all saved bookings, sorts by start time and then ID,
and optionally filters by the normalized room name. IDs start at 1, increase
after each successful booking, and are not reused.

The database is read afresh by each process. A missing file means an empty
book. Successful bookings write a temporary file beside the database and
atomically replace it. Invalid requests, overlap conflicts, unreadable or
malformed database files, and write failures return an error without replacing
the existing database. Concurrent independent writers are outside this
small-tool contract.

## Checks

Run the CLI regression tests and the released public station checks with:

```powershell
python -B -m unittest discover -s tests -v
python -B checks/acceptance.py --case roombook --station 1
```
