# Change request: Shifts and supervision are removed

Staff scheduling has moved to our payroll system, so nobody keeps shifts in here any more. Please remove shifts and supervision whole.

- The operations `addShift`, `cancelShift`, `listShifts` and `setSupervision` are gone, and with them the shift and supervision setting records, the identifier `shf-N` and the refusal codes `not_staff`, `shift_needed`, `shift_overlap`, `shift_too_long` and `supervisor_required`. The module `staffing` leaves the architecture.
- No family is supervised any more, so no reservation is refused for lack of cover. The other refusals of `createReservation` keep their order.
- Nobody needs to be on shift for anything any more: the refusal code `approver_not_on_shift` is gone too, and staff can act at any time.
