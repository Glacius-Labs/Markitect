# Concepts

Status and relationship ownership follow [Model language](model-language.md).

<a id="concept-task"></a>
## CONCEPT-TASK — Task

Type: Concept.

A Task is one piece of work a person records. It has a reference, text and state. It has no due date, timestamp, priority, owner account or tag in version 1.

- **Reference:** a positive decimal integer, starting at 1. The reference is stable for the task's lifetime and distinct from every other stored task, including tasks with identical text. User-visible references have no leading zeros. Allocation belongs to [Add](use-cases.md#uc-add); parsing belongs to the CLI Component.
- **Text:** one line of Unicode text. Leading and trailing Unicode whitespace is removed when the task is created. The normalized text must contain a non-whitespace character and cannot contain a tab, carriage return or line feed. Text is not unique; duplicate wording is allowed. Version 1 imposes no application-specific maximum length.
- **State:** exactly `open` or `completed`. There is no intermediate completion state. Text and reference remain unchanged when a task is completed.

The stored collection contains any number of Tasks with distinct references. An empty collection is valid. The domain meaning is independent of the file encoding and programming language. Malformed stored records are handled through [safe storage](requirements.md#req-safe-storage), rather than silently normalized on read.
