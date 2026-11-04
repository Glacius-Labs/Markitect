---
kind: Interface
---
# Hazmat interface

What other modules may use of hazmat: its operations, the classification record and the refusal codes below. Other modules also ask hazmat, for a member, a material, a quantity and the request time, which refusal of the [[hazmat/hazard-gate]] applies to a purchase: `hazard_not_certified`, `hazard_limit_exceeded` or none. [[materials/materials-interface]] declares this need, and the protocol module supplies the answer.

- **Operations:** [[hazmat/classify-hazard]], [[hazmat/declassify-hazard]], [[hazmat/list-hazards]].
- **Uses:** [[equipment/equipment-interface]], [[materials/materials-interface]], [[training/training-interface]].
- **Records:** [[hazmat/classification]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `hazard_limit_exceeded` | The member would have bought more units of the hazardous material in the month than its monthly limit allows. |
| `hazard_not_certified` | The member has no certification valid on the date of the request time for the family of the hazardous material. |
| `not_hazardous` | The material has no classification. |
