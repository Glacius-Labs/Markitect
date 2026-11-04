---
kind: std/Rule
internal: true
---
# Records in responses

A `result` object has exactly the results its operation lists, and every object inside it has exactly the fields stated for its type. There are never fewer or additional members. Nullable fields are present with the value `null` when they have no value. Lists are JSON arrays; an empty list is `[]`, never `null`. Amounts, counts and minutes are JSON integers. The order of members inside a JSON object is free; the order of list elements is fixed by [[list-order|the order of listed results]].
