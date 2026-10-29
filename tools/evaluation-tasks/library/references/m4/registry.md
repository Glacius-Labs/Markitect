# Registry

| File | Realizes | Why |
|---|---|---|
| `realization/README.md` | [[realization]] [[callable-interface]] | Usage documentation: how to build, run and test the program, and its scope |
| `realization/cmd/library/main.go` | [[callable-interface]] | Entry point: serves JSON Lines on standard input and output |
| `realization/go.mod` | [[callable-interface]] | The Go module |
| `realization/internal/calendar/calendar.go` | [[inputs]] [[lending/loan-policy]] [[lending/borrow]] [[lending/renew]] | Calendar dates: parsing, adding days and the supported range |
| `realization/internal/calendar/calendar_test.go` | [[inputs]] [[lending/loan-policy]] [[lending/borrow]] [[lending/renew]] | Tests of the calendar dates |
| `realization/internal/library/library.go` | [[members/member]] [[members/register-member]] [[members/set-member-status]] [[members/inspect-member]] [[catalog/copy]] [[catalog/register-copy]] [[catalog/inspect-copy]] [[lending/loan]] [[lending/loan-policy]] [[inputs]] | Records of members, copies and loans, the loan policy values, and the member and copy operations |
| `realization/internal/library/library_test.go` | [[catalog/availability]] [[catalog/copy]] [[catalog/inspect-copy]] [[catalog/register-copy]] [[inputs]] [[lending/borrow]] [[lending/inspect-loans]] [[lending/loan]] [[lending/loan-policy]] [[lending/reminders/assess-overdue]] [[lending/renew]] [[lending/return]] [[members/eligibility]] [[members/inspect-member]] [[members/member]] [[members/register-member]] [[members/set-member-status]] [[realization]] | Tests of the domain records and operations |
| `realization/internal/library/loans.go` | [[lending/borrow]] [[lending/renew]] [[lending/return]] [[lending/inspect-loans]] [[lending/loan]] [[lending/loan-policy]] [[lending/reminders/assess-overdue]] [[members/eligibility]] [[catalog/availability]] | Lending operations: borrow, renew, return, loan inspection and the overdue assessment |
| `realization/internal/protocol/operations.go` | [[callable-interface]] [[inputs]] [[members/register-member]] [[members/set-member-status]] [[members/inspect-member]] [[catalog/register-copy]] [[catalog/inspect-copy]] [[lending/borrow]] [[lending/renew]] [[lending/return]] [[lending/inspect-loans]] [[lending/reminders/assess-overdue]] | The wire operations: their arguments and the call each makes |
| `realization/internal/protocol/protocol_test.go` | [[callable-interface]] [[catalog/inspect-copy]] [[catalog/register-copy]] [[inputs]] [[lending/borrow]] [[lending/inspect-loans]] [[lending/reminders/assess-overdue]] [[lending/renew]] [[lending/return]] [[members/inspect-member]] [[members/register-member]] [[members/set-member-status]] [[realization]] | Tests through the wire format |
| `realization/internal/protocol/request.go` | [[callable-interface]] [[inputs]] | Reading one JSON Lines request |
| `realization/internal/protocol/server.go` | [[callable-interface]] | The JSON Lines server: one response per request, in order |
| `realization/internal/protocol/views.go` | [[callable-interface]] [[members/member]] [[catalog/copy]] [[catalog/availability]] [[lending/loan]] [[lending/reminders/assess-overdue]] | The wire shape of members, copies, loans and notices |
