# Pre-cutover source archive and restoration

Created and verified before any S1 source removal under the [source implementation approval](../concepts/sources/v1-source-implementation-approval-20261023.md). This archive preserves the integrated source before Extract, Cockpit/UI, Authoring and Setup retirement; it is not a release or runtime-readiness claim.

- Repository: `https://github.com/Glacius-Labs/Markitect.git`.
- Retained remote branch: `archive/v1-pre-cutover-20261023`.
- Exact commit: `447ac3b152de79b52de80513e14dc94349a060bd`.
- Exact Git tree: `6bd59e02d929dc94fe82f799cefc0da18ff7984c`.
- Independent local bundle: `C:\Users\Consiliari\Glacius Labs\Markitect-archives\v1-pre-cutover-20261023\markitect-447ac3b.bundle`.
- Bundle SHA256: `6336E2E48E46F9FA35EB278C66C3BCBA9EB058664951676B56A8336CB40124BB`.

The remote branch was pushed without force and read back at the exact commit. Keep this named archive ref and the bundle; ordinary implementation cleanup must not delete them. It is a source snapshot, not a dependency on dangling historical objects. The bundle records a complete history and the `refs/heads/main` reference. The source snapshot commit has no parents; this does not establish ancestry to pre-reset historical SHAs.

## Restore independently from the bundle

Choose a new empty destination outside existing checkouts. Verify the bundle hash first, then explicitly select its `main` branch (the bundle does not supply a default HEAD):

```powershell
$bundlePath = 'C:\Users\Consiliari\Glacius Labs\Markitect-archives\v1-pre-cutover-20261023\markitect-447ac3b.bundle'
Get-FileHash -Algorithm SHA256 -LiteralPath $bundlePath
git clone --branch main $bundlePath 'C:\path\to\new-restored-markitect'
git -C 'C:\path\to\new-restored-markitect' fsck --full
git -C 'C:\path\to\new-restored-markitect' rev-parse HEAD
git -C 'C:\path\to\new-restored-markitect' rev-parse 'HEAD^{tree}'
git -C 'C:\path\to\new-restored-markitect' status --short
```

Require the commit/tree above and an empty status. The actual restoration check lives beside the bundle in `restore-check`: checkout, `git fsck --full`, exact commit/tree and clean status succeeded. It is not an implementation worktree. Bundle verification succeeded independently of its restored checkout. A copy of this guide is retained beside the bundle as `RESTORE.md`.

## Restore from the named remote reference

```text
git clone --branch archive/v1-pre-cutover-20261023 https://github.com/Glacius-Labs/Markitect.git NEW_EMPTY_DESTINATION
```

Verify the same exact commit/tree and clean status. If the reference has moved or is unavailable, use the verified bundle. Do not reset or overwrite an active implementation checkout to restore history. This archive contains committed source only, not local credentials, operator state, provider histories or uncommitted planning documents. The approved plan and later source changes have their own reviewed commits.
