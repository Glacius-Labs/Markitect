# Candidate C-01 — Isolated handlers for new use cases

## Proposed rule

For new use-case behavior in this project, place its handler in the use-case folder. Existing service-centric code may remain during migration.

## Scope and classification

Project-specific hypothesis; the selected evidence does not establish a team-wide preference or prove intent.

## Supporting evidence

- E-01: The selected recent Refund folder contains a Handler.

## Counterexamples

- E-02: The older RefundService still exists; the transition note says callers are migrating.

## Qualifying evidence

- E-03: A transition note explains migration but does not state the preferred future design.

## Confidence and open question

Low confidence about normative intent. Ask the owner whether this is an approved direction for new use cases or only an observation about this fixture.