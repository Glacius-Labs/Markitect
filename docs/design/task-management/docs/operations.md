# Setup, data care and releases

These procedures target the documented first release. There is no released executable or application test evidence yet. They must be exercised against the delivered source on both target environments before release.

## Local setup and execution

Use Python 3.13.x on the target Windows 11 or Ubuntu 24.04 environment. Obtain a complete versioned application source folder and retain its version identifier. Do not combine source files from different releases. No pip install, account, database, service or downloaded runtime package is required after Python itself is installed.

From the source root, verify the selected interpreter with `python --version`, then invoke the commands in the [reference](cli.md). Replace `python` with the actual Python 3.13 executable name if needed. The current directory must contain the delivered `tasks/` package for `python -m tasks` to find it.

The source directory is needed for module discovery, but the default task file is outside it. Removing or replacing application source does not intentionally delete the task file. An explicit override inside a source directory is the user's responsibility during upgrades.

## Selecting the task file

The default is `Path.home() / ".tasks" / "tasks.json"` for the OS user running the process. On Windows, this is normally a path such as `C:\Users\Alice\.tasks\tasks.json`; on Linux, `/home/alice/.tasks/tasks.json`. These examples do not override the actual home directory. Running as another OS user selects that user's default file.

If the home directory cannot be resolved, fail reading the selected file rather than guessing a working-directory fallback. Read-only commands do not create `.tasks`. The first mutation may create it. Its directory must otherwise be usable; failed directory preparation is a save failure.

`--file PATH` before the action replaces the default for that invocation only. Resolve relative paths against the current directory; an override's parent must already exist. Do not save an implicit global preference. A missing override file is an empty collection; a missing parent prevents saving. Different selected files are separate lists with independent references.

Use trusted, ordinary local files. Network shares, symbolic-link targets, concurrent commands and edits by another program are unsupported. Stop commands before copying, replacing or inspecting private task data with editing tools.

## Backup and restore

For important data, make a backup after a work session and before upgrading:

1. Stop all commands using the selected file.
2. Identify the actual file, including any override. Copy it to a separate protected backup location; keep its format/version and the application version with the backup record.
3. Confirm that the copy exists and is readable. Test restoration with a copy selected through `--file`, never by experimenting on the sole original.

To restore, preserve the current file under another name, copy the compatible backup to the selected location and run List. Confirm expected references, texts and states before making another change. A nonexistent file needs no backup, but a zero-byte or corrupt file must be preserved for investigation rather than mistaken for no data.

There is no automatic backup rotation, cloud copy, encryption or secure deletion. Backup frequency and retention are the user's choice. Removing a file or source folder is not secure erasure.

## Troubleshooting

| Symptom | Check and recovery |
|---|---|
| Python cannot find `tasks` | Use the complete application source folder and run from its root with the intended interpreter. |
| Unexpectedly empty list | Check OS user and any `--file` argument. List must not create a missing file. Do not assume another list was erased. |
| Read failure | Check path, regular-file status and read permissions. Preserve existing data; do not replace it with an empty file. |
| Invalid task file | Preserve it, test a compatible backup with `--file`, then restore deliberately. A valid JSON document may still violate the task format. |
| Unsupported version | Use the application version compatible with that data or a format-1 backup. Never force a downgrade by editing only the version number. |
| Save failure | Check the parent directory, write/replace permissions, file locks and free space. An ordinary failure before commit leaves prior data intact. |
| Confirmation missing or output error | List the same file before repeating an add; the mutation may already have committed. |
| Orphan temporary files | Stop commands and preserve anything needed for investigation. A confirmed selected file remains authoritative; do not restore from a temporary file automatically. |

Unexpected defects require the source version, Python/platform versions, action, exit status and a minimized synthetic reproduction. Do not attach real task contents unless deliberately reviewed for disclosure.

## Privacy and file access

Task text is plaintext in storage, backups and command output. Command arguments may also appear in terminal history or process listings. Do not use Tasks for credentials or other secrets. OS account/directory permissions are the protection boundary; the application does not authenticate users or promise encryption.

Use a private home directory and protected backup location. Replacement can create a new file with permissions inherited from its directory; do not assume custom permissions on an old file are preserved unchanged. The application must not deliberately broaden OS permissions, but permission behavior must be assessed on both targets. `.gitignore` protects only paths covered by a particular repository and cannot protect arbitrary overrides.

## Source release and update

The planned delivery is a versioned source archive, initially `tasks-0.1.0.zip`, with the package, documentation, tests and changelog. There is no registry installer or background deployment. The release location is not established because no release exists; do not invent a download URL.

Before publishing, run the [release criteria](testing.md#release-readiness), record the exact source revision, update the package version/changelog and create an archive from the clean source. Include a SHA-256 checksum with its actual filename. Unpack the archive into a clean directory and exercise setup, tests and the command walkthrough on both targets before distributing it. Exclude task files, backups, temporary files and interpreter caches.

For an update, back up the selected data, obtain a complete verified source archive, unpack separately, confirm data-format compatibility and run the new version against a backup copy first. Then use it with the original selected file. Keep the previous source and backup until satisfied. A rollback is permitted only when the older application accepts the existing data format; source replacement does not migrate data.
