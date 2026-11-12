---
kind: Kind
fields:
  For: {required: true}
  Shows: {required: true}
  All actions:
parts:
  actions: std/Action
  refusals: std/Refusal
  displays: std/Display
---
# Page

A Page Definition states one page of a site as the person who uses it meets it, under `pages/` of the site's Package. **For:** the roles that may open it, in backticks, and how it is reached. **Shows:** what the person sees, named by what the [[std/Operation|Operations]] it reads return; beyond that a page shows only what the configuration holds, such as the name of the business.

Its [[std/Action|Actions]] and [[std/Refusal|Refusals]] live in the folder of its name, under `actions/` and `refusals/`; further statements on what it shows are [[std/Display|Displays]] under `displays/`. **All actions:** what holds for every one of its Actions, where something does. Layout, wording and looks are free unless a part states them.

Verified when it shows what is stated and its parts are verified.
