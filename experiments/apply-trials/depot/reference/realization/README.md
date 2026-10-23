# Depot

A small local command line for a depot: a catalog of items, the stock on hand, customer orders that reserve stock, and stock reports. It uses only the Python 3.13 standard library.

## Run

From this directory:

```text
python -B app.py --db PATH COMMAND ...
```

| Command | Effect | Returns |
| --- | --- | --- |
| `add-item --sku SKU --name NAME [--reorder N]` | Adds an item to the catalog; the reorder point defaults to 0. | `{"sku", "name", "reorderPoint"}` |
| `receive --sku SKU --qty N` | Adds N units to the item's stock on hand. | `{"sku", "onHand"}` |
| `place-order --line SKU:QTY [--line SKU:QTY ...]` | Places an open order of 1 to 10 lines and reserves their quantities. | the order |
| `fulfil-order --id N` | Fulfils an open order and takes its lines out of stock. | the order |
| `cancel-order --id N` | Cancels an open order and releases its reservation; cancelling again changes nothing. | the order |
| `orders [--status open\|fulfilled\|cancelled]` | Lists orders by id. | `{"orders": [...]}` |
| `items` | Shows every item with `onHand`, `reserved` and `available`, sorted by SKU. | `{"items": [...]}` |
| `reorder` | Lists items whose available quantity is below their reorder point, with the `shortfall`, largest first. | `{"items": [...]}` |

An order is returned as `{"id": 1, "status": "open", "lines": [{"sku": "BOL-006", "qty": 8}]}`. Ids start at 1 and count up; a refused order uses no id.

## Inputs and rules

- A SKU is trimmed and uppercased and must then be three letters, a hyphen and three digits, such as `BOL-006`, in every command and order line.
- A name is trimmed, must not be empty and has at most 60 characters.
- A quantity is written in digits only and is from 1 to 500; a reorder point from 0 to 9999; an order id at least 1.
- Stock on hand stays from 0 to 9999; receiving beyond 9999 is refused.
- Reserved is what open orders hold; available is on hand minus reserved. An order is placed only when every line fits the available quantity and no item appears twice; otherwise nothing is placed.
- Fulfilled and cancelled orders reserve nothing and cannot be fulfilled again; a fulfilled order cannot be cancelled.

## Output and errors

Every successful command prints one JSON object and exits with 0. Invalid input, a refused change and a wrong command line print `{"error": "..."}` and exit with 2; the data file is then unchanged.

## Data file

All data is kept as UTF-8 JSON in the one file given by `--db`. A missing file means an empty depot. The file's directory is not created: without it, changes fail and the queries show an empty depot. Each changing command replaces the file atomically, so a later run sees all of its changes; the queries never write the file. A malformed file is reported and left as it is. The tool assumes one writer at a time: two processes changing the same file concurrently can lose an update.

## Code and tests

`app.py` reads the command line; `depot/` holds the shared input rules and the data file, `catalog/`, `stock/`, `orders/` and `reports/` one part each, with its tests beside it. `tests/` holds the end-to-end tests. Run every test with:

```text
python -B -m unittest discover -s . -p "test_*.py"
```
