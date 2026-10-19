# A rejection that finds fewer gaps does not count (19 October 2026)

Source of [DEC-054](../register.md#dec-054-a-rejection-that-finds-fewer-gaps-does-not-count).

## Evidence

Two bound library runs with gpt-6-luna ended Incomplete at the rejection limit of three, while the rejected Manager converged. Each run had no owner question and no Host fault.

- R2-A1-10 (Main 441366f5) stopped after 23.5 minutes and 39 steps. Three consecutive local reviews rejected `library` with 9, 9 and 4 of 11 recorded cases missing. No candidate was accepted in between, so [DEC-050](../register.md#dec-050-an-accepted-delivery-restarts-the-rejection-count) did not restart the count.
- R2-A1-11 (Main 8391b3de) stopped after 33 minutes and 44 steps. Three consecutive local reviews rejected `catalog`, with these gaps, counted as missing cases plus blocking findings: 6 + 1, 4 + 3, then 0 + 1. The last review judged every recorded case evidenced. Its one blocking finding asked for a test that borrows the second same-title copy, not only one that shows it is available.

## Proposal

The Architect proposed: a rejection whose review finds fewer gaps than the task's last rejection for the same purpose shows that the work converges, so it does not count toward the limit. The limit then stops only work that does not improve. The finite shared execution budget still bounds the total cost.

## Owner decision

The Architect asked:

> R2-A1-11 ist wieder am Ablehnungslimit gescheitert. catalog lag zuletzt bei 0 von 6 fehlenden Fällen, und es blieb nur ein blockierender Befund übrig. Lücken über die drei Reviews: 7 → 7 → 1. Soll eine Ablehnung, die weniger Lücken findet als die vorige, nicht mehr gegen das Limit zählen?

The owner chose:

> Ja, Fortschritt zählt nicht (Recommended)
