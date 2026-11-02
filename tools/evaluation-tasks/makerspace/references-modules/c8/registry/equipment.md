---
paths:
  - realization/equipment/**
---
# Registry: equipment

| File | Realizes | Why |
|---|---|---|
| `realization/equipment/equipment.go` | [[equipment/equipment-interface]] [[equipment/family]] [[equipment/machine]] [[equipment/maintenance-window]] [[equipment/maintenance]] [[equipment/catalog]] | Families, machines and maintenance windows, what other modules may ask of them and what equipment needs from bookings |
| `realization/equipment/operations.go` | [[equipment/add-family]] [[equipment/add-machine]] [[equipment/set-machine-rate]] [[equipment/list-machines]] [[equipment/schedule-maintenance]] [[equipment/cancel-maintenance]] [[equipment/maintenance]] [[equipment/catalog]] | Equipment operations: families, machines, rates and maintenance windows |
| `realization/equipment/views.go` | [[equipment/family]] [[equipment/machine]] [[equipment/maintenance-window]] | JSON views of the family, machine and maintenance window records |
