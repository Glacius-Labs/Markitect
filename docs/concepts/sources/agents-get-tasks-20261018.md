# Agents get tasks; the Orchestrator owns the process (18 October 2026)

Source of [DEC-051](../register.md#dec-051-agents-get-tasks-the-orchestrator-owns-the-process).

## Evidence

In bound run R2-A1-08, a Manager work step received about 78 KB of input. The model Definitions themselves were a small part of it. The rest was:

- the Host's process rules: admission order, the write barrier, limited-delivery acceptance, the rejection allowance and owner-question mechanics;
- provenance data: record references as byte arrays, source digests, inclusion reasons, settings and profile identities.

Several of these sentences had been added to compensate for process gaps. The most recent was the reading-conflict rule of [DEC-048](../register.md#dec-048-recorded-review-cases-stand-for-every-delivery-of-a-purpose).

## Owner instruction

> Ausserdem denke ich die Agenten sollten vom Ablauf und von der Koordination gar nicht so viel wissen. Die Prompts sollten nicht unermesslich wachsen um den Prozess zu kompensieren. Stattdessen sollten die Agenten einfach klare Aufgaben bekommen und entsprechend Antworten, sodass der Orchestrator es entsprechend übernehmen und weiter machen kann. Ich befürchte wir geraten in eine Loop, wo die Prompts immer mehr von der Organisation und der Orchestrierung in die Agenten packt und daher die Ergebnisse immer fragwürdiger werden. Das soll nicht passieren.
