# Registry

| File | Realizes | Why |
|---|---|---|
| `realization/README.md` | [[documentation]] [[project]] | How to build, run and test the program, an example session and the scope |
| `realization/go.mod` | [[project]] | The Go module |
| `realization/policies/ids.go` | [[values]] [[protocol/initial-state]] | Counters for the generated identifiers of a module's records |
| `realization/policies/money.go` | [[money]] | Charge amount rounded half up once per line |
| `realization/policies/operation.go` | [[Operation]] [[argument-values]] | The vocabulary in which a module describes its operations: argument types, permissions, results and refusals |
| `realization/policies/order.go` | [[list-order]] | Ascending order of map keys by ASCII byte value |
| `realization/policies/policies_test.go` | [[money]] [[time]] [[computed-times]] | Tests of charge rounding and of day, month and interval arithmetic |
| `realization/policies/refusal.go` | [[refusal-codes]] | The refusal: a code and a reason |
| `realization/policies/roles.go` | [[authorization]] | The roles and their place in a permission string |
| `realization/policies/time.go` | [[time]] [[computed-times]] | Time scale, day and month arithmetic and half-open interval overlap |
| `realization/policies/values.go` | [[values]] [[identifier-limits]] [[integer-limits]] | Value formats: identifiers, text, timestamps, dates and months, parsed and formatted |
