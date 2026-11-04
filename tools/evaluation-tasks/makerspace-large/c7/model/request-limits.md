---
kind: std/Rule
---
# Request limits

- **Request size and depth.** The 64 KiB request lines are 65,536 bytes, not counting the line feed. Longer lines, and lines with any JSON value nested more than 100 levels deep in arrays or objects, are not specified and are not tested.
- **Counting depth.** Levels count arrays and objects only. The outermost array or object of the line, normally the request object, is level 1, and an array or object directly inside a level-n array or object is level n + 1. So `args` is level 2, and an array or object given as an argument value is level 3. Strings, numbers, booleans and `null` add no level. A JSON value is nested more than 100 levels deep when it is an array or object at level 101 or deeper, or lies inside one. Up to level 100, all other rules apply in full. A line that is not valid JSON is answered by the invalid-request rules whatever its depth.
