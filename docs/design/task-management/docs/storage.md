# Data format and persistence

## Format version 1

The selected task file is UTF-8 JSON without a BOM. It contains exactly the top-level members `version` and `tasks`. `version` is the integer 1; `tasks` is an array. Each task object contains exactly `id`, `text` and `state`.

```json
{
  "version": 1,
  "tasks": [
    {"id": 1, "text": "Buy milk", "state": "open"},
    {"id": 2, "text": "Send letter", "state": "completed"}
  ]
}
```

`id` is a JSON integer in the [domain reference range](domain.md). Booleans and floating-point values do not qualify as integers. `text` and `state` are strings that already satisfy the domain rules; loading does not trim or repair them. References are unique across the whole array. An empty array is valid. Array order is not significant; writes order tasks numerically by reference for predictable inspection.

Reject missing/unknown members, duplicate object keys, duplicate references, invalid task properties, invalid UTF-8, a BOM, trailing non-whitespace data and nonstandard JSON constants such as NaN/Infinity. Validate root/object shapes and required fields before inspecting version compatibility; a structurally valid integer version other than 1 is unsupported. No partially valid collection is returned. An empty zero-byte file is invalid, not an empty list.

Output uses UTF-8, LF newlines, readable JSON indentation and a final newline. JSON object-member order is not an interoperability promise. There is no public import/export or manual-editing workflow; the format is documented so maintainers can verify compatibility and recover data deliberately.

## Loading and selecting a file

The default/override rules belong to [Operations](operations.md#selecting-the-task-file). A missing selected file means an empty collection; reading does not create its directory or file. Existing unreadable or invalid data is an error. A directory, symbolic-link file or non-regular file selected as the target is unsupported and not followed or replaced. This is not protection against a hostile process racing the filesystem; the application assumes a trusted local location.

All commands load and validate the same complete representation. Do not silently select a backup, temporary file or another list when the chosen file cannot be read.

## Save procedure and commit point

For a changed collection:

1. Validate the entire candidate collection and serialize its complete representation before touching the current data file.
2. If using the default location and its application directory is absent, create that directory. Explicit file overrides never cause parent-directory creation.
3. Create a uniquely named temporary regular file in the selected file's directory using exclusive creation. Write all bytes, flush, synchronize the temporary file and close it before replacement.
4. Replace the selected file with the completed temporary file. Successful replacement is the commit point.
5. Only then return a successful action result so the CLI can confirm it.

The same-directory temporary file avoids a cross-filesystem move. The implementation uses the platform's replacement primitive and must not fall back to truncating/writing the selected file in place. Python's [file replacement](https://docs.python.org/3.13/library/os.html#os.replace) and [file synchronization](https://docs.python.org/3.13/library/os.html#os.fsync) APIs inform this design; their existence does not prove this implementation or every filesystem durable.

Before commit, a reported save failure leaves an existing selected file byte-for-byte unchanged; an absent file remains absent. A newly created default directory may remain empty after failure. Clean up the invocation's temporary file where possible without touching another file. Orphan temporary files are not task files and are never loaded automatically.

After commit, no rollback is attempted because of failed confirmation or cleanup. A complete new file is authoritative. Process interruption and power loss are outside the durability guarantee; synchronization of a temporary file alone is not a promise of directory metadata survival after a system crash. Ordinary complete-command restart behavior is required and tested.

## Compatibility and recovery

Version 0.1 reads/writes format 1 only. An unsupported version is refused without rewriting or downgrading it. There is no automatic migration. A future release that changes format must document compatibility, a backup-first migration and rollback consequences before it is released.

Backups are user-controlled copies of the selected file made when commands are not running. Recovering means preserving the suspect file, restoring a known-good compatible backup and validating it with List. Never guess that a temporary file is committed state. [Operations](operations.md) owns the recovery procedure.

## Implementation notes

Python's [JSON parser](https://docs.python.org/3.13/library/json.html) permits behaviors that this format disallows by default, including repeated member names and nonstandard numeric constants. The adapter must configure or supplement parsing and then validate the whole result. Test those cases; syntactically parseable JSON is not necessarily valid task data.
