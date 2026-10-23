# An accepted delivery restarts the rejection count (18 October 2026)

Source of [DEC-050](../register.md#dec-050-an-accepted-delivery-restarts-the-rejection-count).

## Evidence

Bound run R2-A1-08 (Main a9f62311) ended Incomplete after 8945 s, because `library` reached the local rejection limit of three. Each of its three limited deliveries was rejected once. Each rejection was repaired, and the next candidate passed: two deliveries were accepted, and the third rejection came on the third delivery. The limit counted rejections over the whole run, so a Manager that converged on every delivery was stopped.

## Proposal

The Architect proposed to the owner that the limit count rejections since the task's last accepted delivery. The purpose of the limit is to stop work that does not converge, and an accepted delivery shows convergence. The finite shared execution budget still bounds the total cost.

## Owner decision

The owner answered the proposal:

> Mache es so wie du vorschlägst.
