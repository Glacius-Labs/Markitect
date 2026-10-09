# Knowledge Graph validation ledger

## KG01 documentation candidate

Source reference: `1495e1be7b0046711531fa42c8407fba67b8e814`.
Candidate/remote at publication: `8f57b9c22caf24370422692ff62c0c215cba269e` on `origin/codex/knowledge-graph`.
One independent fresh Luna High Reader returned PASS with no material corrections, no edits and no executed tests. Exactly twelve questions were checked against source. README local targets exist.

| Executed check | Result | Log SHA256 |
|---|---|---|
| source-built root check at candidate | exit 0 / passed | `90470D9E4B1502F8F579D59C2C0DF42DA18828DAA954A5E00C02099EB722B801` |
| selected development engineering-change Context at candidate | exit 0 | `263499B294199DC0DF9D223B2F7D05B2B1EDC15EA05DB0C84216CA1B870DCB47` |
| fixed BASE-to-candidate Impact | exit 0; all-resources conservative expansion for new unowned documentation inputs | `4F9F3F12F90CF608DEE54157F957B239877ED957CAC1CE7D0731FCC90D1A0FEB` |
| working-tree managed-artifact helper with unchanged config | exit 0; new KG files NOT covered | `839ED6F05ACA4E57598387CEF6DFE00AAA2CC9DC674F7DFB670F0854C94C2F95` |
| git diff --cached --check | exit 0 | no log digest recorded |
| git ls-remote origin branch | exact candidate match | source fact above |

Fixed candidate snapshot digest from root check/Context: `ac2123c0e0e6297c9a90809253b4f44977c9c0adf65f51bbcdfca2addb739302`. Source-built tool digest remained `sha256:4c7b7010d8a359a4dbee00fad466323c738b04b7bfd92a0884c43c205165a911`. Accounting helper used working-tree inputs with its own snapshot digest `2af660ded37c1050f3c6e0ca4d259f4acb73e7c0789705519d39452a3359d35e`; it is not a fixed-candidate Verify result.

**Unverified ownership coverage:** `docs/work-items/knowledge-graph/**` is outside the pinned managed roots/tooling declarations. New module files will also require Integration-owned artifact configuration reconciliation. This workstream does not change the shared configuration to simulate coverage. Root and Integration were notified with the concrete paths.

No Shop cancellation execution, full Go suite, full Project Verify, real provider journey, authenticated human acceptance, Main merge, release or study result follows from KG01. Later checks must record their actual tested source separately.
