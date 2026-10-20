# Stockroom

A small local command line for keeping stock of items. It uses only the Python 3.13 standard library.

## Run

From this directory:

```text
python -B app.py --db PATH COMMAND ...
```

| Command | Effect |
| --- | --- |
| `add-item --sku SKU --name NAME` | Registers a new item with 0 on hand and returns it. |
| `receive --sku SKU --qty N` | Adds N to the item's on-hand quantity and returns the item. |
| `issue --sku SKU --qty N` | Subtracts N from the item's on-hand quantity and returns the item. |
| `list` | Returns `{"items": [...]}` with every item, sorted by SKU. |

An item is returned as `{"sku": "BOL-006", "name": "Bolt M6", "onHand": 750}`.

## Inputs and limits

- A SKU is trimmed and uppercased and must then be three letters, a hyphen and three digits, such as `BOL-006`. `bol-006` and `BOL-006` are the same item.
- A name is trimmed, must not be empty and has at most 60 characters. Names need not be unique.
- A quantity is written in digits only and is from 1 to 1000.
- The on-hand quantity stays from 0 to 9999. Receiving beyond 9999 and issuing more than is on hand are refused. Issuing exactly the quantity on hand leaves 0.

The limit values live in `limits.json`.

## Output and errors

Every successful command prints one JSON object and exits with 0. Invalid input, a refused change and a wrong command line print `{"error": "..."}` and exit with 2; the data file is then unchanged.

## Data file

The data is kept as UTF-8 JSON in the file given by `--db`. A missing file means no items. Each change replaces the file atomically, so a later run sees it; `list` never writes the file. A malformed file is reported and left as it is. The tool assumes one writer at a time: two processes changing the same file concurrently can lose an update.

## Tests

```text
python -B -m unittest discover -s tests
```
