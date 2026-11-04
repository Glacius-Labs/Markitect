---
kind: std/Concept
---
# Values

The formats of the values that requests, records and results carry. Arguments are checked against them as [[argument-values]] states.

| Value | JSON type and range | Meaning |
| --- | --- | --- |
| identifier | string | A JSON string of 1 to 64 characters from `A`–`Z`, `a`–`z`, `0`–`9`, `.`, `_` and `-`, whose first character is a letter or digit. Identifiers are compared exactly and are case-sensitive: `Ada` and `ada` are different identifiers. |
| text | string | A JSON string that contains at least one character without the Unicode `White_Space` property. It is stored and returned exactly as supplied, without trimming. |
| timestamp | string | A JSON string `YYYY-MM-DDTHH:MMZ` naming a valid UTC calendar date and time from `2000-01-01T00:00Z` to `2099-12-31T23:59Z`, for example `2026-03-02T09:00Z`. Hours are `00` to `23`, minutes `00` to `59`; there are no seconds. |
| date | string | A JSON string `YYYY-MM-DD` naming a valid calendar date from `2000-01-01` to `2099-12-31`. `2026-02-29` is not valid; `2028-02-29` is. |
| month | string | A JSON string `YYYY-MM` from `2000-01` to `2099-12`. |
| clock time | string | A time of day `HH:MM`. |
| count | integer at least `0` | A count, quantity or sequence number; never negative. |
| cent amount | integer at least `0`, in cents | An amount of money; never negative. |
| minute count | integer at least `0`, in minutes | A number of minutes; never negative. |
| percent | integer at least `0` | A discount in percent. |
| flag | boolean | A yes-or-no value. |
| operation name | string | The name of an operation, as in a request's `op`. |
| audit target | string | A plain string that identifies the record an operation acted on. A target that joins two values with `/` can be up to 129 characters long and need not meet the identifier format. |

Units: cents (`ct`): One cent; all amounts are integer numbers of cents; minute (`min`): One minute of time.
