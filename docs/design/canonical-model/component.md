# Component

A **Component** describes a part of the system, its purpose and its responsibilities.

A Component can be a logical part or have a runtime or deployment role. It can contain other Components; a separate process or deployment is not required. For example, Task Logic is a logical Component, while a backend service can be a runtime Component.

Design decision: the owner chose this general, nestable meaning on 15 October 2026 in the [consolidated design discussion](consolidation.md#1-component-allgemeiner-systemteil-oder-laufzeitbaustein). The additional purposes of System and Module remain to be clarified.

Current review status: the owner subsequently marked Component versus Module, together with System, as [provisional terms](consolidation.md#arbeitsziel-und-wackelkandidaten). The general meaning above is the discussion basis; whether both Component and Module are needed is open and is the next design priority.
