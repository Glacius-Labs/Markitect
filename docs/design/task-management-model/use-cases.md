# Use Cases

These records own local caller-visible behavior. Shared input meaning is owned by [Task](concepts.md#concept-task); the explicit `applies-to` edges select the shared Requirements. References below navigate to those owners and do not duplicate their definitions.

<a id="uc-add"></a>
## UC-ADD — Add a task

Type: UseCase. Actor: the local user. Trigger: `tasks [--file PATH] add TEXT`.

**Input.** Exactly one text argument. Shell quoting is used to pass spaces as one argument. Normalize and validate it according to the Task Concept before reading the task file.

**Flow.** Read the selected collection. Allocate reference `1` for an empty collection, otherwise one greater than the highest stored reference. Create a Task with the normalized text and initial state `open`. Store the complete next collection, then return the created reference. A failed store does not consume a reference because no counter outside the saved collection is advanced.

**Success.** stdout is `Created task N.` where `N` is the new reference. Existing tasks are unchanged. Duplicate text creates a distinct task.

**Refusal.** Invalid Task text produces `Task text must be nonblank and single-line.` with the input-failure outcome from REQ-ERRORS. A valid input followed by a file failure uses the shared storage outcome. Argument-count/syntax errors are rejected by the CLI before this flow. No rejected operation creates a task.

<a id="uc-list"></a>
## UC-LIST — List tasks

Type: UseCase. Actor: the local user. Trigger: `tasks [--file PATH] list`.

**Input.** No task-specific arguments.

**Flow.** Read the selected collection and order all Tasks by their reference numerically ascending. This is a read-only action; do not save, normalize or create a task file.

**Success.** For an empty collection, stdout is `No tasks.` Otherwise, stdout begins with `ID<TAB>STATE<TAB>TEXT` and contains one line per task in the same three-column order. `<TAB>` here means one tab character, not the printed word. State is the Task Concept's state word; text is the normalized stored text. No filtering or hiding of completed tasks occurs. The displayed reference is the value accepted by Complete.

**Failure.** File failures use the shared storage outcome and do not produce a partial list. Argument-count/syntax errors are rejected by the CLI before this flow.

<a id="uc-complete"></a>
## UC-COMPLETE — Complete a task

Type: UseCase. Actor: the local user. Trigger: `tasks [--file PATH] complete ID`.

**Input.** Exactly one reference using the Task Concept's user-visible syntax. The CLI validates the syntax before reading the file.

**Flow.** Read the selected collection and find the task with that reference. For an open task, change only its state to `completed`, store the complete next collection, then confirm the result. For an already completed task, keep the collection unchanged and do not write the file.

**Success.** Completing an open task produces `Completed task N.` Completing an already completed task produces `Task N is already completed.` Both are successful outcomes. Other tasks, the selected task's text and its reference are unchanged.

**Refusal.** A syntactically valid reference absent from the collection produces `Task N does not exist.` with the input-failure outcome from REQ-ERRORS. This includes an empty or missing task file; no new file is created. File failures use the shared storage outcome. Argument-count/syntax errors are rejected by the CLI before this flow.

## One complete interaction

This is an expected future interaction, not execution evidence. It assumes the selected file is initially absent and normalizes newline conventions.

```text
$ tasks list
No tasks.
$ tasks add "Buy milk"
Created task 1.
$ tasks add "Buy milk"
Created task 2.
$ tasks complete 1
Completed task 1.
$ tasks list
ID<TAB>STATE<TAB>TEXT
1<TAB>completed<TAB>Buy milk
2<TAB>open<TAB>Buy milk
$ tasks complete 1
Task 1 is already completed.
$ tasks complete 99
Task 99 does not exist.
```

The final line is stderr with exit code `1`; earlier responses are stdout with code `0`. On a later process invocation against the same file, the list remains the same. A changed working directory without an explicit file selection addresses a different default file; it is not data loss in the first file.
