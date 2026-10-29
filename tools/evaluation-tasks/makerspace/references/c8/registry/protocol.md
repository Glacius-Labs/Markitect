---
paths:
  - realization/cmd/makerspace/main.go
  - realization/cmd/makerspace/main_test.go
  - realization/internal/ms/wire.go
---
# Registry: protocol

| File | Realizes | Why |
|---|---|---|
| `realization/cmd/makerspace/main.go` | [[protocol/program]] | Entry point: reads request lines from standard input and writes response lines to standard output |
| `realization/cmd/makerspace/main_test.go` | [[protocol/program]] [[protocol/responses]] [[protocol/invalid-requests]] | Test of a program session over standard input and output |
| `realization/internal/ms/wire.go` | [[protocol/requests]] [[protocol/responses]] [[protocol/invalid-requests]] [[protocol/program]] | Request parsing, invalid requests and response encoding |
