# Live Shop acceptance exercise

This harness uses the public `markitect project` CLI and an existing signed-in Codex account. It starts real provider calls, including separately configured implementer/reviewer sessions and Manager integration, for `--stage run` and an explicitly requested `--stage repair`; preparation performs local discovery, reviewed runtime setup, an exact model edit and impact planning. The selected model is explicit; there is no fallback. The caller authorizes provider usage and supplies budget weights. The weights in this finite experiment are estimates, not provider prices or a billing cap.

Build a source candidate and keep its adapter bytes fixed for the whole exercise. From the Markitect source root:

```powershell
go build -o "$env:TEMP\markitect-live.exe" ./cmd/markitect
python -B experiments/project-world-live/run.py --source . --tool "$env:TEMP\markitect-live.exe" --output "$env:TEMP\shop-live-proof" --model MODEL
python -B experiments/project-world-live/run.py --source . --tool "$env:TEMP\markitect-live.exe" --output "$env:TEMP\shop-live-proof" --model MODEL --stage run
python -B experiments/project-world-live/run.py --source . --tool "$env:TEMP\markitect-live.exe" --output "$env:TEMP\shop-live-proof" --model MODEL --stage verify
python -B experiments/project-world-live/run.py --source . --tool "$env:TEMP\markitect-live.exe" --output "$env:TEMP\shop-live-proof" --model MODEL --stage apply
```

Preparation creates a disposable Git repository, freezes both source adapters outside it, and records the exact source revision and plan in `state.json`. No adopting checkout or global configuration is changed. It changes four accepted model statements: cancellation must work during packing, as well as while confirmed. `plan --since` must select six Managers and one declared check without a manually supplied Manager list. Runtime setup pins Python, the adapter and the native provider executable for implementation and read-only review. Standard setup enables finite local review rounds and Manager-directed rework within the same cumulative run budget. Review findings and follow-up candidates remain in the durable run record; invocation counts depend on the actual findings, not a fixed happy-path expectation.

If the CLI discovered through npm or `PATH` differs from the one supplied by your agent host, pass `--provider-executable ABSOLUTE_NATIVE_CLI_PATH` during preparation. For a Codex app session that exposes `CODEX_CLI_PATH`, this can be `--provider-executable "$env:CODEX_CLI_PATH"`. `state.json` records the selected executable/version; later stages use the frozen runtime and do not rediscover a provider. Both harnesses support this option. Confirm model compatibility on that exact CLI/account; a rejected model is never replaced automatically.

The Apply stage first changes one disposable documentation file after preflight, checks rejection of the stale binding, restores the exact original bytes, then applies the verified candidate.

If a required declared check fails, inspect `project status --repo DISPOSABLE_SHOP_PATH --run PLAN_ID`. An explicit `--stage repair` returns that known check failure through the same Manager tree, using the same run identity, original deadline and cumulative start/cost ledger. It does not apply files or change accepted model/check obligations. After successful repair, run `--stage verify` again before Apply. The default setup permits one repair round; unknown provider outcomes, stale inputs, exhausted limits and an unchanged failed candidate remain blocked. Keep the binary and adapter bytes fixed throughout the lifecycle.

After Apply, read the disposable checkout path from `state.json` and run the independent acceptance checker:

```powershell
python -B experiments/project-world-live/acceptance.py --repo DISPOSABLE_SHOP_PATH
```

This checker stays outside the adopting checkout and is never supplied to the Managers. It constructs its own SQLite database and tests confirmed/packing cancellation, zero/duplicate/pre-released reservation rollback, an injected database failure during release, repeat idempotency and shipped-order rejection. Run it before implementation as a negative control: the unmodified fixture must fail the new packing requirement. The declared fixture suite and this independent checker cover finite examples; passing them does not prove completeness of the model or every possible implementation behavior.

Public CLI reports are saved beneath the output directory. Provider logs remain in the run's private log store and must not be published as raw transcripts. `controlled-local` constrains proposals and guarded Apply; it does not provide operating-system isolation. See the [validation record](../../docs/validation/project-world-delivery.md) for the exact tested candidates and observed provider behavior.


The companion `brownfield.py` prepares another disposable Shop with a deliberately contradictory legacy guide. Use the same source/tool/output/model flags and stages `prepare`, then `generate`. The generated report must retain the disagreement as a question and keep static source observations separate from documented intent. Review the report before supplying explicit choices with `--stage resolve --choices CHOICES.json`; the choices transport contains `actor`, `authorityClaim`, `decisionReference`, `questions` and `scopes`. `--stage adopt` previews and applies the exact resolved model-only proposal. Choose adoption and deferral deliberately; the harness never invents those decisions. Source code and documentation must stay byte-for-byte unchanged by adoption. There is no runtime evidence in this discovery, so generated claims must not assert that execution occurred.
