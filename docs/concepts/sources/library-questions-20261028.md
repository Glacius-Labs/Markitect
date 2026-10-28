# Owner decisions: the open Library questions

Date: 27 and 28 October 2026 (UTC). Source: direct owner messages in the Claude Code session for lane 6 (the canonical model as a language), written in English. The owner statements are verbatim; the session's proposals they answer are summarized and labeled. The recorded meaning is the recorder's and adds no decision.

They answer the open questions of the [standard Libraries](../../design/standard-libraries.md#answered-and-open-questions), within the [canonical model track](../../design/model-track-charter.md) ([DEC-074](../register.md#dec-074-canonical-model-track-parallel-to-v1)).

## 1. The five questions

Context (summary by the recorder): the session recommended an optional flag for unordered refusal lists, a refusal-code field on Interface, letting Placement place an Invariant through an interface both implement, no wire-envelope Kind yet, and no alias-only import visibility as a Core change yet.

> Accept the recommendations on the five Library questions: unordered-refusal flag, refusals in Interface, invariant placement via an interface, no envelope Kind yet, no alias-only Core change yet. Prepare the plan, build nothing yet.

The owner then asked to go through the changes one by one.

## 2. Refusals have no order

Context (summary by the recorder): the session explained that the refusal list is read first-match, so a model must fix which refusal wins when several apply, even where the requirements are silent.

> Why does the order even matter? As i read it those are just potential problems that might occur. If the model is not clear about it, when which one applies, those should simply be definitions themselves and explain it there

The session proposed dropping the flag: the list has no order, and where precedence matters a Rule states it, as Makerspace's `order-of-checks` already does; this makes `api` a major version.

> Yes

On the name `RefusalCode`, the session offered `Error` as more familiar and noted that the name came from Makerspace's protocol.

> If its just for makerspace and not library we can go with Refusal Code. Lets discuss what the problem is

The session pointed out that the name sits in the standard `api` Library; the owner did not reopen it.

## 3. Refusal codes in an Interface

> Just think about how this is handled in code. We should do it the same way

## 4. Placing an invariant

> Okay lets look at that the same way

The session answered that in code, cross-cutting concerns sit in an outer layer, which `Placement` models, while an invariant is enforced by the code that owns the data, and proposed dropping the change.

> Yes i agree

## 5. Build

> Build the two Library changes as planned, in one PR.

## Recorded meaning

Recorder's interpretation:

- **`api` 2.0.0:** `refusals` lists the refusals an operation can return, with no order. Where precedence matters, a Rule or the operation's body states it. The flag of point 1 is not built.
- **`RefusalCode` keeps its name.**
- **`architecture` 1.1.0:** an Interface exposes refusal codes in `refusalCodes`, as a package exports its errors in code.
- **No Library change for invariants:** an Invariant lives in, and is enforced by, the Package that owns its Entities; Placement stays for cross-cutting concerns.
- **No wire-envelope Kind and no alias-only import visibility** until a real model needs them.
