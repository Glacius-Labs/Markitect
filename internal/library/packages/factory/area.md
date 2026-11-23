# Area

An Area of the plan: a lasting field of work with an owner, under which features and stories are planned. Take it for every Area, each written down explicitly as a document of its own. An Area has no spec of its own: it covers parts of the product and building blocks of the architecture, which stand in the spec, and every change to them lands through a story like everything else. An Area is no part of the spec: a part is a folder of the spec with a README, an Area belongs to planning. Areas usually follow the architecture, but planning may be cut differently from what is built. In `.markitect/plan/` every Area has a folder with its features, among them the standing feature `housekeeping`.

## Purpose
The one goal of the Area, in a sentence or two, and where it connects. Example: `Lending: members borrow, renew and return copies. Connects to the catalog's interface for copies.`

## Owner
The person who owns the Area, as a link. Example: `[Ada](../people/ada.md)`.

## Covers
The parts of the product and the building blocks of the architecture this Area is responsible for, as links into the spec, one per line. Example: `[Lending](../../product/lending/README.md)`.

## Class (optional)
The class of the Area, as a link. Example: `[Cross-cutting](../classes/cross-cutting.md)`.

## Rules (optional)
Rules for this Area alone, stricter than its class's, as links or in short sentences. Example: `Lending never changes a member's data; it asks the members Area.`
