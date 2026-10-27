# Owner decisions: repository structure

Date: 24 October 2026. Source: the owner's direct message in a Claude Code session, after the R1 inventory ([repository structure](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/development/repository-structure.md), PR #680) listed six decisions for the owner in its §11, each with the assistant's recommendation. The owner statement is verbatim.

## Owner decision (verbatim)

> Yes, I agree with all six recommendations: archive the 873 files including the marketing plan, don't translate archived records, protect the archive refs, pin the six old commits, adjust the tag Rule instead of restoring tags, and move the internal packages in step 5.

## Recorded meaning

1. R2 removes the 873 files classified as archive from Main, including `docs/strategy/marketing-plan.md`, the test projects and the accepted-but-later designs. They stay reachable through a named archive point, its bundle and the index page `docs/archive.md`.
2. Archived records are not translated. DEC-069 applies to the documentation on Main.
3. A repository ruleset protects `refs/heads/archive/**` against deletion and force-push. The coordinator created it on 24 October 2026 as ruleset "Protect archive refs".
4. The six commits from before the 23 October history reset that remaining files cite (`5ff32069`, `1f4a1ede`, `3ebe3c6e`, `bc4b49e7`, `0c28097c` and `7dbd599c`) are pinned with archive refs.
5. The tags are not restored. The self-model Rule `tooling/maintenance-boundaries` is changed so that it no longer claims tag preservation. The retained release evidence and the archive refs keep the needed history.
6. Package moves inside `internal/` happen in step 5 together with the self-model Package cut, not in R2.
