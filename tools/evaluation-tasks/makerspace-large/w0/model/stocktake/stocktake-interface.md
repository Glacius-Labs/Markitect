---
kind: Interface
---
# Stocktake interface

What other modules may use of stocktake: its operations, the stocktake and count records and the refusal codes below. Other modules also ask stocktake whether a material has a count in an open stocktake. [[materials/materials-interface]] declares this need, and the protocol module supplies the answer.

- **Operations:** [[stocktake/start-stocktake]], [[stocktake/record-count]], [[stocktake/close-stocktake]], [[stocktake/get-stocktake]].
- **Uses:** [[materials/materials-interface]], [[ledger/ledger-interface]].
- **Records:** [[stocktake/stocktake]], [[stocktake/count]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `nothing_counted` | The stocktake has no count. |
| `stocktake_open` | The material has a count in an open stocktake. |
| `stocktake_running` | A stocktake is already open. |
