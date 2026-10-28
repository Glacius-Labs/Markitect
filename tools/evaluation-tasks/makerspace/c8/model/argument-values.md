---
kind: std/Rule
---
# Argument values

Each argument has a field type: an identifier, text, timestamp, date or month as their definitions state, an enumeration (a JSON string equal to one of the listed values, compared exactly), or an integer.

An integer is a JSON number written as an optional minus sign followed by digits, without a fraction part or exponent (`10.0` and `1e3` are not integers), within the range its type states. Every range in this model starts at 0 or higher, so any value written with a minus sign, including `-0`, is out of range.

An argument that is missing, `null`, of the wrong JSON type or outside its format or range is refused with [[refusal-codes|`invalid_argument`]]. Each argument is checked on its own. Relationships between arguments, such as an end that is not after a start, and relationships to stored state are checked later and have their own codes.

These formats and ranges apply to arguments, including `at`. An integer argument outside its range is out of range however many digits it has. Values the program computes can lie outside these ranges; [[computed-times|computed values]] states the rules for those values.

`actor` is an identifier argument and `at` is a timestamp argument of every operation. An `actor` that is missing, `null`, not a string or not a valid identifier, such as `"bad id!"` or a string of 65 characters, is refused with `invalid_argument` at check 1, like any other argument. Only a valid identifier that names no user is refused with [[refusal-codes|`unknown_actor`]], at check 2.
