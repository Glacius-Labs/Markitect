# Sample

The result of a control sample: which deviations from the spec a sampler planted in a throwaway copy of the artifacts, and the verdicts a reviewer gave on that copy without knowing them. The copy never lands; this file does, and the tool computes the hit rate from it. Each section holds values of fixed form, which the tool reads.

## Source
`job control-sample sampler`, then a comma and the day as `YYYY-MM-DD`.

## Copy
The commit or branch of the throwaway copy.

## Planted
A line `PATH: DEVIATION` for each planted deviation, the path that of a document below the spec folder.

## Verdicts
A line `PATH: VERDICT` for each document the reviewer judged on the copy, the verdict `ok`, `fail` or `skip`.
