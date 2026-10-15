# Command reference

This is the intended interface of version 0.1. It is specified before application delivery. `python` denotes the selected Python 3.13 interpreter, run from the application source root.

## Invocation and common rules

```text
python -m tasks [--file PATH] add TEXT
python -m tasks [--file PATH] list
python -m tasks [--file PATH] complete ID
```

`--file` is optional, appears at most once and only before the action, and takes one nonempty path argument. Paths with spaces must be shell-quoted. A relative path is interpreted against the process working directory. Default location and directory behavior are specified in [Operations](operations.md#selecting-the-task-file). There are no environment-variable settings, interactive prompts, stdin data input or additional actions in this version.

Unknown options/actions, missing or extra arguments, repeated options, an empty path and an invalid reference spelling are invocation errors. Reference syntax is the [domain's canonical decimal syntax](domain.md); it is validated before opening a file. Validate invocation first, then action-specific text, then read/validate the complete selected file, then resolve a valid reference. Thus invalid input wins over an unrelated unreadable file, while a read failure wins over a possible unknown-reference result.

Public text is UTF-8. Each line ends in the platform's normal newline; comparison normalizes CRLF/LF only. Successful commands use stdout and leave stderr empty. Input/storage errors produce one line on stderr and no stdout. Input is never interpreted through a second shell. There is no localization or color-output contract.

## Add

Input is exactly one text argument. Normalize and validate it using [Domain](domain.md), then read the collection and allocate the next reference. Create an open task, commit the complete new collection and only then confirm.

```text
python -m tasks add "Buy milk"
Created task 1.
```

Success leaves existing tasks unchanged. Adding identical text again creates a different reference. Invalid text produces `Task text must be nonblank and single-line.` An exhausted reference range produces `No task identifiers are available.` Neither changes data or consumes a reference.

Add is not idempotent: repeating a successful add creates another task. After a missing confirmation, check the list before repeating it.

## List

List takes no action-specific arguments and is read-only. It never creates a data file or changes file contents. It displays all tasks, including completed tasks, in numerically ascending reference order.

An empty or missing file produces `No tasks.` A nonempty list starts with the header `ID<TAB>STATE<TAB>TEXT`, followed by one row per task with the same fields. `<TAB>` means one actual tab character. The reference is the value used by Complete; state uses the words from Domain. No partial list is printed if loading or validation fails.

For readability, this example shows column separation as spaces:

```text
ID    STATE        TEXT
1     completed    Buy milk
2     open         Send letter
```

## Complete

Complete takes exactly one reference. Read the file and find that task. An open task becomes completed and is saved before confirmation. Its text/reference and all other tasks stay unchanged.

```text
python -m tasks complete 1
Completed task 1.
```

Repeating the command for a completed task returns `Task 1 is already completed.` with success and no write. A valid but absent reference returns `Task 1 does not exist.` This also applies to a missing or empty data file, without creating one.

## Exit status and failures

| Exit | Meaning | stderr message |
|---|---|---|
| 0 | Successful action, including an empty list or repeated completion | Empty |
| 1 | Invalid invocation | `Invalid command or arguments.` |
| 1 | Invalid task text | `Task text must be nonblank and single-line.` |
| 1 | Unknown valid reference | `Task N does not exist.` |
| 1 | Reference capacity reached while adding | `No task identifiers are available.` |
| 2 | File cannot be read or selected path is not supported | `Cannot read task file.` |
| 2 | Invalid JSON, shape or task data | `Task file is invalid.` |
| 2 | Structurally valid file declares another format version | `Unsupported task file version.` |
| 2 | Directory preparation or save fails before commit | `Cannot write task file.` |
| 3 | Result cannot be delivered, for example a broken stdout stream | `Cannot write command output.` if stderr is usable |
| 4 | Unexpected internal defect | `Unexpected error.` if stderr is usable |

An input refusal or pre-commit storage error leaves task data unchanged. Once storage commits, output delivery or an unexpected later failure cannot undo it. Exit 3/4 or an interrupted process therefore does not establish whether a mutation happened. Output itself may be partial if the output stream fails; inspect state before retrying. See [Storage](storage.md#save-procedure-and-commit-point).

Do not emit tracebacks, task contents or successful result text on a handled error. Detailed technical diagnosis belongs in reproducible developer investigation, not a default persistent log.

## Complete example

Using a new file in an existing writable directory:

```text
python -m tasks --file demo.json list
No tasks.
python -m tasks --file demo.json add "Buy milk"
Created task 1.
python -m tasks --file demo.json complete 1
Completed task 1.
python -m tasks --file demo.json complete 1
Task 1 is already completed.
python -m tasks --file demo.json complete 99
Task 99 does not exist.
```

The last response is stderr with exit 1; the preceding responses are stdout with exit 0. A fresh process listing the same file still shows task 1 completed. These are expected results, not a record of a run.
