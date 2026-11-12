---
kind: Kind
fields:
  For: {required: true}
  Follows: {links: Package}
parts:
  routes: std/Route
---
# Interface

An Interface Definition says how behavior is offered to programs, or to people at a command line, in a Package under `interfaces/`. **For:** who uses it. **Follows:** another Package whose Shapes, Refusals and Limits hold for it too, when there is one. Its [[std/Route|Routes]] live in the folder of its name, under `routes/`.

Shapes, Refusals and Limits that hold for a whole API or site lie in the folders `shapes/`, `refusals/` and `limits/` of its Package, and those that hold for every interface in those folders of `interfaces/`. A caller may rely on these and on the [[std/Operation|Operations]] offered, and on nothing else.

Verified when its parts are and it is offered to whom **For** names.
