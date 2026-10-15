# Tasks

Tasks is a small, local command-line application for keeping track of personal work. Add a task, see the whole list and mark work completed. Data stays on your computer and remains available between runs.

**Project status:** version 0.1 is a pre-implementation design. This repository contains its documentation; application source, executable tests and release packages have not been supplied yet. Commands below specify the intended interface and become runnable when the implementation is delivered. No supported release or successful application test run is claimed.

## What the first version does

- Create a task with a text and a stable number.
- List all tasks, including completed tasks.
- Mark a task completed; repeating that action is harmless.
- Keep tasks in a local file without an account or online service.

Editing, deleting, reopening, priorities, dates, reminders, filtering, synchronization and shared use are outside the first version. One person runs commands sequentially against a file.

## Intended setup

The baseline is Python 3.13.x with its standard library, on Windows 11 or Ubuntu 24.04. These are target environments to verify before release. No package installation, database or background service is required. Obtain the complete application source folder and run commands from its root. In these examples, `python` means the Python 3.13 interpreter; on a machine that names it `python3.13`, substitute that executable throughout.

```text
python --version
python -m tasks add "Buy milk"
python -m tasks list
python -m tasks complete 1
```

Expected list after adding the first task:

```text
ID    STATE    TEXT
1     open     Buy milk
```

The actual column separators are tabs, illustrated here as spacing. Text output is UTF-8. Shell quoting passes a task containing spaces as one argument.

## Where tasks are kept

The default is `.tasks/tasks.json` inside the current operating-system user's home directory. It does not change when you move between working directories. Reading an absent file shows an empty list; the default directory/file is created on the first successful change.

Select another file before the command:

```text
python -m tasks --file "./personal-tasks.json" list
```

A relative override is relative to the working directory. Its parent directory must already exist. All invocations must select the same file to work on the same list. Use an ordinary local file, not a network share or a symbolic-link target.

## Failures and data care

Invalid input does not change tasks. An unreadable, damaged or unsupported data file is reported as an error and is never silently reset. Successful data-changing commands confirm only after storage has committed. A missing confirmation after interruption or output failure does not prove that nothing was saved; inspect the list before repeating an add command.

Tasks are stored as plaintext, not encrypted. Do not enter secrets. Keep backups of important lists and do not run two writers against the same file. [Operations](docs/operations.md) explains backup, restore, updates and recovery.

## Read more

| Reader task | Document |
|---|---|
| Understand the purpose, scope and quality goals | [Product](docs/product.md) |
| Use commands or interpret an error | [Command reference](docs/cli.md) |
| Understand task meaning and validation | [Domain](docs/domain.md) |
| Understand the implementation design | [Architecture](docs/architecture.md) |
| Inspect the persistent format and save procedure | [Storage](docs/storage.md) |
| Back up, restore or update | [Operations](docs/operations.md) |
| Develop or review a change | [Contributing](CONTRIBUTING.md) |
| Understand verification and release readiness | [Testing](docs/testing.md) |
| Understand why these choices were made | [Decisions](docs/decisions.md) |

[Documentation map](docs/README.md) · [Changelog](CHANGELOG.md)
