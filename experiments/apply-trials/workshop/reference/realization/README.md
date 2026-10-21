# Bicycle workshop

A small local CLI with which one operator runs a bicycle repair workshop,
using only the Python 3.13 standard library.

## Commands

Run every command from this directory:

```text
python -B app.py --db PATH COMMAND ...
```

| Command | Effect |
| --- | --- |
| `add-customer --name NAME` | Registers a customer (ids C1, C2, ...) and returns its record. |
| `customer --id ID` | Returns `{"id", "name", "paidInvoices", "tier"}`. |
| `add-part --sku SKU --name NAME --price CENTS` | Registers a part without stock. |
| `receive --sku SKU --qty N` | Adds N units on hand. |
| `parts` | Returns `{"parts": [...]}` sorted by SKU, each `{"sku", "name", "price", "onHand", "reserved", "available"}`. |
| `open-job --customer ID --bike TEXT` | Opens a repair job (ids J1, J2, ...). |
| `add-labor --job ID --minutes N` | Adds labor to an open job. |
| `use-part --job ID --sku SKU --qty N` | Reserves N units of a part for an open job. |
| `close-job --job ID` | Closes the job; its reserved units leave the stock. |
| `jobs [--status open\|closed]` | Returns `{"jobs": [...]}` in creation order. |
| `invoice --job ID` | Writes the invoice of a closed job (INV-0001, INV-0002, ...). |
| `pay --invoice NUMBER` | Marks an invoice paid. |
| `invoices [--customer ID]` | Returns `{"invoices": [...]}` by number. |
| `report` | Returns `{"openJobs", "unpaidTotal", "paidTotal", "lowStock"}`. |

A job record is `{"id", "customer", "bike", "status", "minutes", "parts"}`
with parts as `[{"sku", "qty"}]` in the order of first use. An invoice record
is `{"number", "job", "customer", "tier", "labor", "parts", "discount", "net",
"vat", "total", "paid"}`; amounts are whole cents.

## Inputs

- Names and bicycle texts are trimmed and must have 1 to 60 characters.
- A SKU is trimmed and uppercased and must then be three letters, a hyphen and
  three digits, such as `BRK-101`.
- Numbers are ASCII digits; leading zeros keep the value. Prices are 1 to
  1000000 cents, quantities 1 to 500, labor 15 to 480 minutes in steps of 15.
  A job has at most 2400 minutes of labor.
- An option's value is the next token, even when it starts with a hyphen.

## Stock, loyalty and pricing

- Using a part on an open job reserves units; only available units (on hand
  minus reserved) can be reserved. Closing the job takes its reserved units out
  of stock. On hand never exceeds 9999.
- A customer with at least 3 paid invoices is **loyal**, otherwise
  **standard**; an invoice keeps the tier its customer had when it was written.
- Labor costs 1500 cents per 15 minutes; parts cost their price times the units
  used. A loyal customer gets 10 percent off labor and 5 percent off parts.
  Tax is 19 percent of the net amount. Each percentage is rounded to whole
  cents, an exact half to the even cent (5 percent of 1010 is 50, not 51).
- The report lists the SKUs whose available quantity is below 3.

## Output and errors

Every successful command writes one JSON object and exits 0. Invalid input,
an unknown or missing command or option, malformed data and every refusal
write `{"error": "..."}` and exit 2 without changing the stored data.

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
