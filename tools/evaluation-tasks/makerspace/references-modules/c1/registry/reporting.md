---
paths:
  - realization/reporting/**
---
# Registry: reporting

| File | Realizes | Why |
|---|---|---|
| `realization/reporting/operations.go` | [[reporting/utilization-report]] [[reporting/revenue-report]] [[reporting/reports]] [[reporting/utilization-row]] [[reporting/utilization-totals]] | Report operations: utilization and revenue |
| `realization/reporting/reporting.go` | [[reporting/reporting-interface]] [[reporting/reports]] [[reporting/utilization-row]] [[reporting/utilization-totals]] | The reporting module and the rounding and union arithmetic of the utilization report |
| `realization/reporting/reporting_test.go` | [[reporting/utilization-row]] | Test of the utilization percent rounding |
