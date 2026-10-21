# Hall

A small local room booking CLI for one operator, using only the Python 3.13
standard library.

## Commands

Run every command from this directory:

```text
python -B app.py --db PATH COMMAND ...
```

| Command | Effect |
| --- | --- |
| `add-room --code CODE --seats N` | Registers a room and returns its record. |
| `rooms` | Returns `{"rooms": [...]}`, every room record sorted by code. |
| `book --code CODE --seats N` | Books N seats of an existing room and returns the booking record. |
| `bookings [--code CODE]` | Returns `{"bookings": [...]}` sorted by id, optionally only those of one room. |

A room record is `{"code": "A101", "seats": 10, "bookable": 8}`. A booking
record is `{"id": 1, "code": "A101", "seats": 2}`; ids start at 1 and grow by
one.

## Inputs and the reserve

- A room code is trimmed and uppercased and must then be one letter A-Z and
  three digits, such as `A101`. `a101` and ` A101 ` name the same room.
- Numbers are ASCII digits without sign, spaces, fraction or exponent; leading
  zeros keep the value. A room has 3 to 500 seats, a booking 1 to 500.
- The reserve is **2**: every room holds back 2 of its seats, so its bookable
  seats are its seats minus 2. A booking is admitted only while the seats of
  all bookings of that room stay within its bookable seats; bookings of other
  rooms do not count.

## Output and errors

Every successful command writes one JSON object and exits 0. Invalid input,
an unknown or missing command or option, and every refusal write
`{"error": "..."}` and exit 2 without changing the stored data.

## Data file

All data is UTF-8 JSON in the file named by `--db`. A missing file is empty
data; its directory is never created. Malformed data is reported and never
overwritten. Each successful change replaces the file atomically; queries never
write it. There is no guarantee for several independent processes writing the
same file at the same time.

## Tests

```text
python -B -m unittest discover -s . -p "test_*.py"
```
