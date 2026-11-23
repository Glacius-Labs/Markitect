# Build

Type: [Step](../../../../packages/company/step.md)

## Description
The Builder carries out the story on the story's branch, named as [names in the work](../../rules/work/names.md) says, task by task, and ticks each task it has done; the task for accepting is left to whoever accepts. When the branch exists, because the planning put a spec change on it, the Builder first takes the current `main` into it and continues there; otherwise it cuts the branch from `main`. Before each review round it runs `markitect check` on the spec and the work it changed, and `markitect status` on the story's places and files, and settles what they show. It does not ask. Within the spec it decides itself and takes the simple or common solution, simple in the sense of the concept.

## Performed by
A fresh [Builder](../../agents/build.md).
