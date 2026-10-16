# Product definition

## Problem and audience

A person wants a short, dependable record of unfinished work without an account, browser application or service. Tasks addresses this through a local command-line list. The intended user can open a terminal and run a Python module. It is not a team work-management system.

Success means the person can record work, recognize what remains open and return to the same list later. A correct command interface is necessary but does not itself prove usefulness; release review includes a short user walkthrough.

## First-version scope

The user can create a task, list the complete collection and mark a task completed. Each task has a stable reference, text and an open/completed state. Identical task texts are allowed. Completion changes no other task. Commands and their observable results are specified in the [command reference](cli.md), while [Domain](domain.md) defines valid task data.

Tasks are saved locally. The application has no network activity, telemetry, account system, server or third-party runtime dependency. Backup and restore use ordinary file-copy procedures rather than additional task commands.

The first version does not edit, delete or reopen tasks. It has no filters, tags, priorities, schedules, notifications, import/export feature, shared use, cross-device synchronization or automatic data migration. Asking for one of these is a scope change, not an implementation detail.

## Usage scenarios

1. A person remembers work, records it and receives its reference.
2. The person later opens a new terminal, lists tasks and identifies outstanding work.
3. After finishing work, the person marks that reference completed; the list retains the task and its completed state.
4. The person uses a separate file for another list or restores a backup after a storage problem.

The application does not organize a person's process beyond these acts. There is no assumption that list order represents priority.

## Quality goals and boundaries

| Concern | Intended behavior | How it is assessed |
|---|---|---|
| Correctness | The documented actions and errors are consistent and deterministic. | Domain, command and integration scenarios. |
| Data integrity | Confirmed changes survive an ordinary restart; a failed pre-commit save does not destroy prior data. | Real-storage restart and failure-injection tests. |
| Transparency | File selection and command outcomes are explicit; invalid data is not hidden. | CLI tests and a recovery walkthrough. |
| Simplicity | Three domain actions, a standard-library runtime and one selected file. | Architecture and scope review. |
| Portability | The same behavior on the two declared target environments. | Separate Windows and Linux evidence before release. |
| Privacy | No application network traffic or telemetry; task data remains local plaintext under OS permissions. | Source review and operations review. |
| Usability | A user can distinguish open/completed work and understand common refusals. | Human walkthrough; no productivity claim is currently established. |

The implementation reads the complete file and rewrites it for each change. It is intended for modest personal lists, not large datasets. There is no published latency or maximum-dataset guarantee before measurement. Before the first release, measure commands with a reproducible 1,000-task fixture, report environment and results, and reassess if interaction is noticeably slow; this is a design goal, not a fabricated service-level guarantee.

Normal operation means sequential commands, an ordinary local filesystem and enough resources to complete a command. Simultaneous writers, manual edits during a command, process termination mid-write, power loss, hostile changes to the filesystem and network filesystems are outside the first version's guarantees. Ordinary permission failures, invalid files and write failures remain supported error cases. See [Storage](storage.md) for the exact commit boundary.

## Delivery status

The documentation specifies the proposed 0.1 behavior. Source code, platform runs, usability observations and release packaging are pending. There is no completed release or known production installation. Release readiness is defined in [Testing](testing.md); remaining implementation work must not be mistaken for unresolved user-visible behavior.
