# Markitect production assessment

**Recorded:** 2026-10-02; v0.12.0 evidence added 2026-10-03. This dated assessment records release evidence and distinguishes it from source behavior. It reports only the Markitect source, its distributions, and source-owned gates; project adoption and acceptance are outside its scope.

## v0.12.0 published release

The immutable [v0.12.0 release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.12.0), release ID `402288212`, binds source commit [`83b715c`](https://github.com/Glacius-Labs/Markitect/commit/83b715c415e39af79715cd8a40b3c79237a4c850), following integration of [PR 54](https://github.com/Glacius-Labs/Markitect/pull/54). It ships the bounded `same-target` Domain assertion: two statically named paths, each one or two singleton relation steps, compared by canonical resource identity. The v0.11.0 release remains immutable historical evidence.

| Evidence | Exact basis | Result |
|---|---|---|
| Commit-bound Windows/Linux quality and release workflow | [Run 37090219114](https://github.com/Glacius-Labs/Markitect/actions/runs/37090219114), attempt 1, source `83b715c415e39af79715cd8a40b3c79237a4c850` | Windows and Linux gates passed for the exact release source |
| Release publisher | Release `402288212` | `published-verified`; immutable release and all four assets and attestations verified for the exact source and workflow run |
| Public distribution metadata | `markitect-release distribution --tag v0.12.0 --repo . --write` | Returned successfully after public release verification; generated the README's marked install/example blocks and the three canonical v0.12.0 WinGet manifests |
| Public Windows binary | Separately downloaded public asset | SHA-256 matched the attested Windows asset and `version` reported `Markitect 0.12.0 (windows/amd64)`; software-architecture and delivery-target-equality fixture checks passed |
| Public fixture check basis | Software and Delivery example directories in the inspected working tree | Checks ran with `provisional: true`; these results exercise release behavior against current local fixture inputs and are not immutable consumer-snapshot verification |

| Asset | Verified SHA-256 |
|---|---|
| `markitect-v0.12.0-bundle.zip` | `577b05e57817a4623ddbdb1fa4e29dbfe6aad044640e3bac0a1e3edb49584ef0` |
| `markitect-v0.12.0-linux-amd64` | `f2072ded99d709b8267124d1460e6b1c1b5afd45c3b70ba7c723a82e17152297` |
| `markitect-v0.12.0-provenance.yaml` | `f093da2c40773bea1f30c8203714bc4e343a348cd5ffb1fe58bb23aa3a3e2f9c` |
| `markitect-v0.12.0-windows-amd64.exe` | `b03424560caa460322e3580785d6abc9dfbf2137df878e03ddedcf76b3a8fa47` |

These release checks establish the shipped assertion and its declared fixture behavior. They do not establish real-project adoption, productivity improvement, lower context use, fewer defects, market demand, or general business benefit. The separate real-project validation exercise must report its own observed evidence and limits.

## v0.11.0 published release

The immutable [v0.11.0 release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.11.0), release ID `402071906`, binds source commit [`a86903f`](https://github.com/Glacius-Labs/Markitect/commit/a86903f1e94cada583a225800fa42b2273755fbd), merged from [PR 51](https://github.com/Glacius-Labs/Markitect/pull/51). It adds resource-scoped relation counts, deterministic per-subject PolicyResults, narrowly scoped source-bound exceptions, generated Domain contracts, and provider-independent discovery and constitution-change authoring workflows. Architecture contracts remain explicit versioned Domain packages. The annotated tag is unsigned; immutable-release and asset attestations were verified separately.

| Evidence | Exact basis | Result |
|---|---|---|
| Reviewed candidate and PR CI | [PR 51](https://github.com/Glacius-Labs/Markitect/pull/51), candidate `9acb01fc8beadcf0b13b04422eecb33aca494e98`; [run 37048194132](https://github.com/Glacius-Labs/Markitect/actions/runs/37048194132) | Windows and Linux passed; merged source has the same tree `4caec0d5e05ac85f5ba54602b3f99815eb04a304` |
| Main CI and source artifact | [Run 37048677509](https://github.com/Glacius-Labs/Markitect/actions/runs/37048677509), source `a86903f1e94cada583a225800fa42b2273755fbd` | All three jobs passed; actual downloaded artifact contained bootstrap, lock, source archive, .NET adapter, and both new embedded workflows; the archive's SHA-256 matched lock value `2b15b4f979ab66b22ed58ad09eaad489319b22dd61235f501bf9fd2374bf3c5b` |
| Tagged release | [Run 37049243207](https://github.com/Glacius-Labs/Markitect/actions/runs/37049243207), attempt 1 | Source quality, preflight, Windows/Linux bundle installation and smoke, and asset assembly passed |
| Owner publisher | Release `402071906` | `published-verified`; exact run artifact, tag, immutable release, and all four asset attestations verified; bundle bytes matched the local fixed-source build |
| Public distribution metadata | `markitect-release distribution --tag v0.11.0 --repo . --write` | Returned `written` after verifying public assets, attestations, provenance, workflow attempt, and digests; generated README pins and three canonical v0.11.0 WinGet manifests |
| Public Windows binary | Separate anonymous public download | Digest matched the attested Windows asset; reported `Markitect 0.11.0 (windows/amd64)`; constitution check/model, discovery check, and bundled notices passed |
| WinGet metadata validation | `winget validate --manifest packaging/winget/GlaciusLabs.Markitect/0.11.0` | Native manifest validation succeeded; public catalog availability and installation remain separate |

| Asset | Verified SHA-256 |
|---|---|
| `markitect-v0.11.0-bundle.zip` | `ba46ab6327bde9df31f420a4023bd7c826725085a911ace4e1287d7ff9d2eccf` |
| `markitect-v0.11.0-linux-amd64` | `5a3e4de0449e266191080a04e4c1da0cc2bfa09e88ee99965b03cce58e7fdb90` |
| `markitect-v0.11.0-windows-amd64.exe` | `4e563a19a2e572b0bfc940a24ffee38cf08509639e46a68aa60f2dfadd82f6d8` |
| `markitect-v0.11.0-provenance.yaml` | `bf5c78f7ba9c5b7c3358de255eaf2cdbcf7efac58e28aaea9a14d2463f5ea43f` |

The constitution fixture exercises a pinned v1-to-v2 policy migration, impact, a missing Validator, a source-bound temporary exception visible in results/context/views, and removal after adding the Validator. Invalid, stale, expired, and unneeded exceptions fail; structural and collection-wide errors remain unwaivable. Constraint digests include referenced relation semantics. The package writer regression checks physical archive bytes while preserving virtual members and unrelated local files.

The discovery fixture is synthetic. It demonstrates fixed Git-source and selected-file bindings, separate candidate/evidence/decision bytes, and a read-only integrity helper. It does not establish successful AI inference, authenticated human approval, adoption, measured consumer effort, or business benefit. Real-project validation remains future work. WinGet manifests are release metadata; Microsoft catalog availability and package-manager installation are not claimed. Earlier tags and releases remain unchanged.

## v0.10.0 published release

The immutable [v0.10.0 release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.10.0), release ID `401991913`, binds source commit [`8882d8f`](https://github.com/Glacius-Labs/Markitect/commit/8882d8f8b9eee184036c1dec4b4c51422608d1b1), merged from [PR 49](https://github.com/Glacius-Labs/Markitect/pull/49). The release extends the bundled resource vocabulary with project-owned, versioned domain definitions and a validated semantic model; it adds conservative reconciliation plans and an explicit external command-adapter protocol with optional configured apply capability. The separate .NET reference adapter is read-only. This is a deliberate pre-1.0 language and model change, and no compatibility guarantee is claimed. The annotated tag is unsigned; immutable-release and asset attestations were verified separately.

| Evidence | Exact basis | Result |
|---|---|---|
| Reviewed candidate and PR CI | [PR 49](https://github.com/Glacius-Labs/Markitect/pull/49), candidate `44f14040a26330d7c395654835ac81f4f039da72`; [run 37034873229](https://github.com/Glacius-Labs/Markitect/actions/runs/37034873229) | Windows and Linux passed; merged release source has the same tree `f9a0336a8d3bbc01555b775c56dc2086a3987ea9` |
| Main CI | [Run 37035301668](https://github.com/Glacius-Labs/Markitect/actions/runs/37035301668) | All three jobs passed at main source `8882d8f8b9eee184036c1dec4b4c51422608d1b1`; actual downloaded main source artifact contained the hidden bootstrap, lock file, source archive, and standalone adapter/compiler sources |
| Tagged release | [Run 37035389068](https://github.com/Glacius-Labs/Markitect/actions/runs/37035389068), attempt 1 | Quality, preflight, Windows/Linux installation and bundle smoke, and asset assembly passed |
| Owner publisher | Release `401991913` | `published-verified`; exact run artifact, tag, immutable release, and all four asset attestations were verified |
| Public distribution metadata | `markitect-release distribution --tag v0.10.0 --repo . --write` | Returned `written` after verifying public assets, attestations, provenance, workflow attempt, and payload digests; generated README install/try blocks and three canonical v0.10.0 WinGet manifests |
| Native Windows binary | Exact attested Windows asset | Reported `Markitect 0.10.0 (windows/amd64)`; minimal, repository-layout, and canonical-engineering fixtures passed, including semantic model validation and bundled notices |

| Asset | Verified SHA-256 |
|---|---|
| `markitect-v0.10.0-bundle.zip` | `4633b6257d3bc0f571e212906eb96464e78022dc7b79bddd1137abe59c31eaa4` |
| `markitect-v0.10.0-linux-amd64` | `520b35bee308ce06f3e7d539ea593da25e6b3adf609baa0262b27c130f4e25a6` |
| `markitect-v0.10.0-windows-amd64.exe` | `da230c607a8779c882f825944353f629def22ff9f84be21eeb6f39c2e7f0f230` |
| `markitect-v0.10.0-provenance.yaml` | `c8f43c2c3906622a78696fa13ea63862d9ae4d2f4a2bedd1cda9a3098a4ce8a1` |

The public Windows binary downloaded separately matched the verified Windows asset digest above and passed the canonical-engineering fixture. The annotated tag has no signature block and is unsigned. WinGet manifests are generated release metadata; Microsoft catalog availability and package-manager installation are not claimed. No benchmark, business-benefit result, or adopting-project acceptance was established. Earlier tags and releases remain unchanged.

## v0.9.1 published release

The immutable [v0.9.1 release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.9.1), release ID `401496097`, binds [source commit `eb1b978`](https://github.com/Glacius-Labs/Markitect/commit/eb1b9781da6459d796c4e8fe108b6a8260c75b8e), merged from [PR 46](https://github.com/Glacius-Labs/Markitect/pull/46). It corrects source-relative prose navigation after v0.9.0 relocated Markdown views. Known local typed YAML destinations and unambiguous former companion aliases point to the selected Markdown view; ordinary documents/images retain their actual source target, and provider inline prose keeps canonical YAML destinations. Existing unmarked Markdown files take precedence over companion aliases. The projection consumes fixed snapshot files and infers no dependencies or artifact declarations. The annotated tag is unsigned; immutable release and asset attestations were verified separately.

| Evidence | Exact basis | Result |
|---|---|---|
| Source gates | [PR CI 36954684689](https://github.com/Glacius-Labs/Markitect/actions/runs/36954684689) and [main CI 36954870512](https://github.com/Glacius-Labs/Markitect/actions/runs/36954870512) | Windows/Linux gates passed; reviewed candidate `6afd17cdfc6e9e64a0cc405c976a015d0f8948e8` and release source share tree `98e5dfc62da9880eb69f23ebc584803e5ad3ac57` |
| Tagged release | [Run 36955087269](https://github.com/Glacius-Labs/Markitect/actions/runs/36955087269), attempt 1 | Source, deterministic bundle and installed bootstrap gates passed on Windows/Linux; four exact-source assets assembled |
| Owner publisher | Release `401496097` | `published-verified`; exact run artifact, tag, payload digests, immutable release and all four asset attestations verified |
| Public distribution metadata | `markitect-release distribution --tag v0.9.1 --repo . --write` | Public downloads, attestations, provenance, workflow attempt and digests verified; README install blocks and three versioned WinGet manifests generated |
| Native Windows binary | Exact attested Windows asset | Reported `Markitect 0.9.1 (windows/amd64)`; navigation, minimal and provider-only layout fixtures passed, with embedded notices present |
| Main source artifact | Actual download from main run 36954870512 | Hidden bootstrap and tool files were present in the downloaded source artifact |
| Published native benchmark | [Run 36955512621](https://github.com/Glacius-Labs/Markitect/actions/runs/36955512621) | Attested v0.9.1 and v0.9.0 binaries completed 40 runs per platform with zero nonzero exits; Windows/Linux raw data and summaries retained |

| Asset | Verified SHA-256 |
|---|---|
| `markitect-v0.9.1-bundle.zip` | `aab429352a5fe884d26a498c4b0e45ae881709ddc10484a90ea5e1bbb4edb9b3` |
| `markitect-v0.9.1-linux-amd64` | `37e5de60e9a5ca5216f359ca2a332f55811118f2b84932c456ee1001b739b81c` |
| `markitect-v0.9.1-windows-amd64.exe` | `792e12c8ff44ad44ffb515bac381fcd1d171cb9a0a46a43e11a135bf17abe38e` |
| `markitect-v0.9.1-provenance.yaml` | `716d5bd3508b205ecce89039da6b8fd03b5a425b38ef32fe9a120569e1ee4023` |

Both binaries use identical fixture v2 bytes, digest `37387124b579687fc250caf6e943f672b3e44a3700d86fbe5f3051867f1f2463`, on each platform, so the report marks the workloads comparable. These first-run and three later fresh-process measurements retain runner/cache noise and establish no general speed gain. The bounded prose destination scanner is not a full Markdown parser or a validator of every ordinary link target. WinGet metadata does not establish Microsoft catalog availability. Consumer upgrades and adopting-project acceptance remain project-owned. Earlier tags and releases are unchanged.

## v0.9.0 published release

The immutable [v0.9.0 release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.9.0), release ID `401463997`, binds [source commit `ff5ffd5`](https://github.com/Glacius-Labs/Markitect/commit/ff5ffd5e3b538be58341fdb7983490895201d2b1), merged from [PR 43](https://github.com/Glacius-Labs/Markitect/pull/43). It makes generic Markdown views an explicit target under `docs/markitect/<area>/`, routes provider outputs directly to canonical inputs, and separates consumer bootstrap files from the complete `.markitect/tool/` pin set. Bundle schema v2 deliberately replaces the earlier layout contract. The annotated tag is unsigned; the immutable release and every asset attestation were verified separately.

| Evidence | Exact basis | Result |
|---|---|---|
| Source gates | [PR CI 36947731879](https://github.com/Glacius-Labs/Markitect/actions/runs/36947731879) and [main CI 36947965582](https://github.com/Glacius-Labs/Markitect/actions/runs/36947965582) | Windows/Linux source gates passed; candidate and release source share tree `153f501172ae8b9a3145774e20c14e87546e282d` |
| Tagged release | [Run 36948247745](https://github.com/Glacius-Labs/Markitect/actions/runs/36948247745), attempt 1 | Source, deterministic bundle and installed bootstrap gates passed on Windows/Linux; four exact-source assets assembled |
| Owner publisher | Release `401463997` | `published-verified`; exact run artifact, tag, payload digests, immutable release and each asset attestation verified |
| Public distribution metadata | `markitect-release distribution --tag v0.9.0 --repo . --write` | Public downloads, attestations, provenance, workflow attempt and digests verified; README install blocks and three versioned WinGet manifests generated |
| Native Windows binary | Exact attested Windows asset | Reported `Markitect 0.9.0 (windows/amd64)`; minimal and provider-only layout fixtures passed; init preview wrote nothing, default Area write and resulting Project check passed |
| Main source artifact | Download from main run 36947965582 | Hidden bootstrap and tool files were present in the actual downloaded source artifact |
| Published native benchmark | [Run 36948742024](https://github.com/Glacius-Labs/Markitect/actions/runs/36948742024) | Attested v0.9.0/v2 and v0.8.1/v1 binaries completed 40 runs per platform with zero nonzero exits; Windows/Linux raw data and summaries retained |

| Asset | Verified SHA-256 |
|---|---|
| `markitect-v0.9.0-bundle.zip` | `987f667a570ede674a845b7ff06e0aa76e5f52c83456800840384cdaa39a5b71` |
| `markitect-v0.9.0-linux-amd64` | `22c8198d608f01d01069329ed2dbd9ca784479d80ba72a17c63a0ef2d90d2449` |
| `markitect-v0.9.0-windows-amd64.exe` | `aee5fdd2e1ed91bdefd84c423bbc56213c9015c7de0bd91c00d4d182b6eea281` |
| `markitect-v0.9.0-provenance.yaml` | `de5ae4b572fd895fdef3f06dedb057af341881f6b77c22fba77f176bfae5de53` |

Benchmark fixture v1 remains byte-identical. The transition uses fixture digests `9f3bb4442d0544b02b941a90c2d46689ea1d990c7d227254b70cb5301299949d` (v1) and `37387124b579687fc250caf6e943f672b3e44a3700d86fbe5f3051867f1f2463` (v2) on both platforms. Different workloads are explicitly non-comparable, so the report withholds percentage changes and establishes no speed gain. Versioned WinGet metadata does not establish Microsoft catalog availability or a v0.9.0 package-manager installation. Consumer upgrades and adopting-project acceptance remain project-owned.

## v0.8.1 published release

The immutable [v0.8.1 release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.8.1), release ID `401377439`, binds [source commit `a8f4eb2`](https://github.com/Glacius-Labs/Markitect/commit/a8f4eb2b4257cb877f891d51f582c6f08ce25389), merged from [PR 39](https://github.com/Glacius-Labs/Markitect/pull/39). It separates resolved snapshot values, content digests and deterministic comparison from Git acquisition. In-memory snapshots with opaque IDs exercise parsing, context, review and conservative impact behavior; the public Git CLI, YAML revision fields, digest algorithm and Git safety guards retain their contracts. The annotated tag is unsigned; the immutable release and every asset attestation were verified separately.

| Evidence | Exact basis | Result |
|---|---|---|
| Source gates | [PR CI 36929152765](https://github.com/Glacius-Labs/Markitect/actions/runs/36929152765) and [main CI 36929541072](https://github.com/Glacius-Labs/Markitect/actions/runs/36929541072) | Windows/Linux source gates passed; candidate and release source share tree `9621e48261a7748581079d97588ffaecb366a39e` |
| Tagged release | [Run 36929587756](https://github.com/Glacius-Labs/Markitect/actions/runs/36929587756), attempt 1 | Source, deterministic bundle and bundle install/bootstrap gates passed on Windows/Linux; four exact-source assets assembled |
| Owner publisher | Release `401377439` | `published-verified`; owner preflight, exact run artifact, tag, payload digests, immutable release and each asset attestation were verified |
| Public distribution metadata | `markitect-release distribution --tag v0.8.1 --repo . --write` | Public release, each asset attestation, provenance, successful workflow attempt and downloaded digests were checked; README install blocks and three versioned WinGet manifests were generated |
| Native Windows binary | Exact attested Windows asset | Reported `Markitect 0.8.1 (windows/amd64)` and passed the executable repository-layout fixture |
| Published native benchmark | [Run 36930381247](https://github.com/Glacius-Labs/Markitect/actions/runs/36930381247) | Windows/Linux fixture v1 completed for attested v0.8.1 and v0.8.0 binaries; raw data and summaries retained as run artifacts |

| Asset | Verified SHA-256 |
|---|---|
| `markitect-v0.8.1-bundle.zip` | `f0b42b564cf280f490b98e441c3a5ede649aa0a29db3287a13cbcc5051fbc7b0` |
| `markitect-v0.8.1-linux-amd64` | `2519814945606eb08cdc710576f9b4b032c91ad05f44d17025c891601298a642` |
| `markitect-v0.8.1-windows-amd64.exe` | `6804409617bf0bb007df42710a3a609b996525d4d20289160981abf933ed6c71` |
| `markitect-v0.8.1-provenance.yaml` | `c43146398fda780cea76b15587c8b7f0e1d475ec9ac24021f0f2c295559b214a` |

The generated WinGet manifests are versioned metadata; they do not establish Microsoft catalog availability or a v0.8.1 package-manager installation. Benchmark timings are diagnostic and establish no speed gain. New snapshot providers, consumer upgrades and adopting-project acceptance remain separate work.

## v0.8.0 published release

The immutable [v0.8.0 release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.8.0), release ID `401351835`, binds [source commit `b8b81f0`](https://github.com/Glacius-Labs/Markitect/commit/b8b81f0739d23894592ba5b76ee91a25e7dea6d4), merged from [PR 36](https://github.com/Glacius-Labs/Markitect/pull/36). It recommends responsibility-owned canonical resources under `.markitect/areas/` and makes `.markitect/areas/<namespace>` the default for new `init` plans that omit `--path`. Existing Projects, explicit paths, generated companions, provider outputs and pinned distribution paths retain their contracts. The annotated tag is unsigned; the immutable release and every asset attestation were verified separately.

| Evidence | Exact basis | Result |
|---|---|---|
| Source gates | [PR CI 36923568400](https://github.com/Glacius-Labs/Markitect/actions/runs/36923568400) and [main CI 36924066559](https://github.com/Glacius-Labs/Markitect/actions/runs/36924066559) | Windows/Linux source gates passed; candidate and release source share tree `2d535943fb32787adb037853f77d0c7ecdfb5a51` |
| Tagged release | [Run 36924579615](https://github.com/Glacius-Labs/Markitect/actions/runs/36924579615), attempt 1 | Source, deterministic bundle and bundle install/bootstrap gates passed on Windows/Linux; the executable layout fixture and omitted-path default were checked |
| Owner publisher | Release `401351835` | `published-verified`; owner preflight, exact run artifact, tag, payload digests, immutable release and each asset attestation were verified |
| Public distribution metadata | `markitect-release distribution --tag v0.8.0 --repo . --write` | Public release, each asset attestation, provenance, successful workflow attempt and downloaded digests were checked; README install blocks and three versioned WinGet manifests were generated |
| Native Windows binary | Exact attested Windows asset | Reported `Markitect 0.8.0 (windows/amd64)`; layout fixture passed; omitted-path initialization preview wrote nothing, explicit write created the default Area, and the resulting Project passed `check` |
| Published native benchmark | [Run 36925736027](https://github.com/Glacius-Labs/Markitect/actions/runs/36925736027) | Windows/Linux fixture v1 completed for the attested v0.8.0 and v0.7.0 binaries; raw data and summaries retained as run artifacts |

| Asset | Verified SHA-256 |
|---|---|
| `markitect-v0.8.0-bundle.zip` | `3c8fbb78b5467fdd5e16e21db11ffa72995dbd894f8bd47a5cd4bc0b651d24d8` |
| `markitect-v0.8.0-linux-amd64` | `b764576514556f23c06a6e4bf41e96cdd8f0361a0132c697045053aaa2dc40cd` |
| `markitect-v0.8.0-windows-amd64.exe` | `85757530e1f5ac3a50d3173a5c9e7f6c0401fadc40e0d1b7ce3876dadd7a3c62` |
| `markitect-v0.8.0-provenance.yaml` | `d7477e521f6d908fa8d92111cc8c16f4a455e3da322cf6a83da3f34c6b337b1b` |

The generated WinGet manifests are versioned metadata; they do not establish Microsoft catalog availability or a v0.8.0 package-manager installation. Benchmark timings are diagnostic and establish no speed gain. Consumer pins, project-local migrations and human acceptance remain owned by adopting repositories.

## v0.7.0 published release

The immutable [v0.7.0 release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.7.0), release ID `401286207`, binds [source commit `857a77f`](https://github.com/Glacius-Labs/Markitect/commit/857a77f86b3a4056c7973fed49d12ce2e6bf9b28), merged from [PR 31](https://github.com/Glacius-Labs/Markitect/pull/31). Its fixed-run context request binds a committed task snapshot and explicitly selected source paths to one full Git revision, reporting selected hashes and missing inputs. This records product capability only; no adopting-project implementation effectiveness or time/cost savings are claimed here. The annotated tag is unsigned; the immutable release and its asset attestations were verified separately.

| Evidence | Exact basis | Result |
|---|---|---|
| Source gates | [PR CI 36911228378](https://github.com/Glacius-Labs/Markitect/actions/runs/36911228378) | Windows/Linux source gates passed for the exact reviewed source tree; its tree is identical to the release commit |
| Tagged release | [Run 36913950360](https://github.com/Glacius-Labs/Markitect/actions/runs/36913950360), attempt 1 | Windows/Linux source and bundle install smoke gates passed; exact-source assets and provenance were assembled |
| Owner publisher | Release `401286207` | `published-verified`; owner preflight, run artifact, tag, payload digests, immutable release, and each asset attestation were verified |
| Public distribution metadata | `markitect-release distribution --tag v0.7.0 --repo . --write` | Public immutable release and asset attestations were checked; README install blocks and three versioned WinGet manifests were generated from the attested assets |
| Public downloads | Versioned unauthenticated Windows/Linux download URLs | Distribution verification matched the four downloaded asset digests to provenance and the successful release run |

| Asset | Verified SHA-256 |
|---|---|
| `markitect-v0.7.0-bundle.zip` | `461ae101e3cc50f15b315382162fff1f75f4d91266a0e96790871e80b93c9db5` |
| `markitect-v0.7.0-linux-amd64` | `384718c6574b28ffdcce121a17bda167d963996c3c954e0c27a33493cd9725c8` |
| `markitect-v0.7.0-windows-amd64.exe` | `d797dbbd26d2c40b819b1519dd554701c407505091c9494baa30ff1447a4be0f` |
| `markitect-v0.7.0-provenance.yaml` | `08f0ab103e33f0f8acbf5a1428ccc85b6e552ec5ed11d22136886610f4cba0f6` |

The generated WinGet manifests are local versioned metadata; no community catalog submission or package-manager installation result is claimed. The release gates establish the Markitect source and package checks that ran, not adopting-project acceptance.

## v0.6.0 published release

The immutable [v0.6.0 release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.6.0), release ID `401237667`, binds [source commit 810a86f](https://github.com/Glacius-Labs/Markitect/commit/810a86ffe13aef065f38e70ae49d40911f447af2) from [PR 27](https://github.com/Glacius-Labs/Markitect/pull/27). It ships canonical-owner authoring guidance, optional snapshot router validation and the provider diagnostic/shared-agent-link corrections integrated after v0.5.0. The tag is annotated and unsigned; GitHub release and all asset attestations were verified separately.

| Evidence | Exact basis | Result |
|---|---|---|
| Source gates | [PR CI 36905108369](https://github.com/Glacius-Labs/Markitect/actions/runs/36905108369) | Windows/Linux tests, vet, build, schema, executable examples and package smoke passed |
| Tagged release | [Run 36905582096](https://github.com/Glacius-Labs/Markitect/actions/runs/36905582096), attempt 1 | Deterministic bundle and both-platform bundle install/bootstrap gates passed; four exact-source assets assembled |
| Owner publisher | Release `401237667` | `published-verified`: owner plan approved, run artifact/tag/manifest checked, immutable release and every asset attestation verified |
| Public downloads | Versioned unauthenticated Windows/Linux download URLs | Bytes matched the attested hashes; Windows executable reported `Markitect 0.6.0 (windows/amd64)` |
| Published native benchmark | [Run 36906676251](https://github.com/Glacius-Labs/Markitect/actions/runs/36906676251) | Windows/Linux fixture v1 completed for attested v0.6.0 and v0.5.0 binaries; raw JSON and summaries retained as run artifacts |

| Asset | Verified SHA-256 |
|---|---|
| `markitect-v0.6.0-bundle.zip` | `6fc7ad77b13e3b25d5666f7136a7a631b62bfcd931a5454610b96de12a2981d1` |
| `markitect-v0.6.0-linux-amd64` | `52f017f191a21ef352e602da15e7db7ea5d71b0fa0d0fdcaae9d166def00948c` |
| `markitect-v0.6.0-windows-amd64.exe` | `7bd08165e11e2cf315307c49b65d7ab08f61d75dc8a178ad229bfea450067280` |
| `markitect-v0.6.0-provenance.yaml` | `b13009121cab1e748db5afe5c04a39a9c298e7d794233301ba4f2f8b18aa19ae` |

Local Windows tests exposed transient file-sharing locks during fixture cleanup. The release candidate retries only Windows access-denied/sharing-violation cleanup failures within a bounded two-second window and preserves the final error. The complete local test suite and both platform gates passed afterward. This changes test cleanup, not production refusal or write behavior.

Benchmark timings remain diagnostic and do not establish a speed gain or an adopting project's semantic/human acceptance. Router checking establishes snapshot navigation structure; the actual canonical-owner authoring decision remains a separate observed acceptance case. Consumer pins and project-local acceptance records belong in the adopting repository.

## v0.5.0 published release

The immutable [v0.5.0 release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.5.0), release ID 401138862, points to [source commit 7883734](https://github.com/Glacius-Labs/Markitect/commit/78837346d0cb27c95b3a10f0a469b2709b757a83) from [PR 17](https://github.com/Glacius-Labs/Markitect/pull/17). It adds explicitly selected provider adapters and opt-in sourced consistency checks. The tag object itself is unsigned; the immutable-release attestation was verified separately.

| Gate | Evidence | Result |
|---|---|---|
| Source CI | [Main run 36887739125](https://github.com/Glacius-Labs/Markitect/actions/runs/36887739125) | Windows and Linux source checks passed at the release commit |
| Release CI | [Run 36889429215](https://github.com/Glacius-Labs/Markitect/actions/runs/36889429215) | Source checks, bundle build and installation smoke passed on Windows and Linux; run artifact contained four exact-source assets |
| Published release | Release 401138862 | Release is immutable and not a draft; tag target is the source commit; `gh release verify` and all four local `gh release verify-asset` checks succeeded |
| Native Windows asset | Attested executable | `version` reported 0.5.0 and `check --repo examples/minimal` passed |
| Post-publication benchmark | [Run 36890129330](https://github.com/Glacius-Labs/Markitect/actions/runs/36890129330) | Failed during job setup because one upload-artifact action pin was one character short; no measurements were made |
| Recovery benchmark | [Run 36891661794](https://github.com/Glacius-Labs/Markitect/actions/runs/36891661794) | Repaired workflow ran on main for the same immutable v0.5.0 tag; Windows and Linux fixture v1 passed and each uploaded raw JSON and a summary |

| Asset | SHA-256 |
|---|---|
| `markitect-v0.5.0-bundle.zip` | `25c995f40a8e704197ee4245dc8daf37255adfb2dc2af6524c25d9b638c97da0` |
| `markitect-v0.5.0-linux-amd64` | `e9eaa87fa36bb30578aaed0bef5bae2639689b39ac5bcd35c2d0632323d794cc` |
| `markitect-v0.5.0-windows-amd64.exe` | `6f5b98dad2ea6003bc0368800691af70084a44d2c225c26fe12d120bcda27178` |
| `markitect-v0.5.0-provenance.yaml` | `1f0713059bc6962e41d1b1dc4013e1ec479ed69924582e550608552c53e20281` |

The verified provenance names source commit 78837346d0cb27c95b3a10f0a469b2709b757a83 and workflow run 36889429215. The failed benchmark is separate from release validity. The recovery run measured the same attested v0.5.0 and v0.4.1 binaries without changing either release; its short timings and range differences do not establish a speed gain or regression. An adopting project's installed pin and acceptance remain unverified here.

## v0.4.1 published release

The immutable [`v0.4.1` release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.4.1), release ID `401051061`, points to source commit [`f424432`](https://github.com/Glacius-Labs/Markitect/commit/f4244324b04d4dd748ae13cea43e2455b251a5de) from [PR 12](https://github.com/Glacius-Labs/Markitect/pull/12). The release adds Apache-2.0 licensing and repository guidance; product behavior remains that of v0.4.0.

| Gate | Evidence | Result |
|---|---|---|
| Candidate and main CI | [Main run 36874976547](https://github.com/Glacius-Labs/Markitect/actions/runs/36874976547) | Windows and Linux source checks passed at the release commit |
| Release CI | [Run 36875604747](https://github.com/Glacius-Labs/Markitect/actions/runs/36875604747) | Source gates, bundle and installation smoke passed on Windows and Linux |
| Owner publisher | Release `401051061` | Exact run artifact and tag checked; release published immutable; a repeat `gh release verify` and each of the four `gh release verify-asset` commands succeeded |
| Native Windows asset | Released executable, digest below | `version` reported 0.4.1; `licenses` showed third-party notices and the Markitect Apache-2.0 reference |

| Asset | SHA-256 |
|---|---|
| `markitect-v0.4.1-bundle.zip` | `440e0829e405c4a793eaa02fe102cdabc6d596ed142d3d472f35182d37c13eab` |
| `markitect-v0.4.1-linux-amd64` | `a02d0eb0ada42d95012da0965d471e6e31c4f5d607ac849fb8bd29d85b9b65a3` |
| `markitect-v0.4.1-windows-amd64.exe` | `419f61e99a3ee09c4d87624b44755bacbcb796244cffe7a1fa148da4c79a35f1` |
| `markitect-v0.4.1-provenance.yaml` | `5bfcf67cb3f30a16f5ca7a879bfd1533762000319133e702a171e2db01798c56` |

The publisher's immediate post-publication attestation request returned an error, reporting `published-unverified`. The subsequent direct GitHub release and asset verification succeeded without changing the immutable release. This records the transient verification result instead of claiming that the publisher itself completed successfully.

## v0.4.0 published release

The immutable [`v0.4.0` release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.4.0) was verified against source commit [`1f0d17b`](https://github.com/Glacius-Labs/Markitect/commit/1f0d17bff22143afc6b2b63b4ea8f17486c818e6), release ID `400944013`. Its tree matches reviewed candidate `d7bb86434559bce52a9f7d8078d656c04ecdb024` from [PR 9](https://github.com/Glacius-Labs/Markitect/pull/9).

| Gate | Evidence | Result |
|---|---|---|
| Candidate CI | [Run 36860632114](https://github.com/Glacius-Labs/Markitect/actions/runs/36860632114) | Windows and Linux passed, including packaged commands and project initialization |
| Main CI | [Run 36860951014](https://github.com/Glacius-Labs/Markitect/actions/runs/36860951014) | Passed at the release source commit |
| Release CI | [Run 36861035636](https://github.com/Glacius-Labs/Markitect/actions/runs/36861035636) | Source gates, deterministic bundle and installation/bootstrap smoke passed on both platforms |
| Owner publisher | Release `400944013` | `published-verified`: exact workflow artifact, tag/commit, immutable release and all four asset attestations verified |
| Native Windows asset | Downloaded release executable, digest below | Version/notices, read-only init preview, initialization and structural check passed; fixed verification correctly returned incomplete evidence for the new Project without declared checks |

| Asset | SHA-256 |
|---|---|
| `markitect-v0.4.0-bundle.zip` | `7966d53a1fe81e2b65d1896f267d708ea6ab95dc08f041476664193a5f10d4c3` |
| `markitect-v0.4.0-linux-amd64` | `8487c3f3258992e89d890ddf036945ce4f026f504d3ed7d80b133b4c3b576608` |
| `markitect-v0.4.0-windows-amd64.exe` | `67d62b1ebdd1224dc18591c72c98c7d8e3f395cedd137ffff1ed3cd66eebc719` |
| `markitect-v0.4.0-provenance.yaml` | `0a852d3b89a5430eaf882dfc6b0bebd53d8748cce6a8030da5302da366fd35a5` |

The earlier [candidate run 36860209819](https://github.com/Glacius-Labs/Markitect/actions/runs/36860209819) failed the packaged notices command because the bootstrap incorrectly inserted `--repo`. The repair preserves repository-independent notices and help arguments and adds regression coverage. That failed result is retained; only the later candidate passed the final gate.

An independent review found and confirmed the Windows short-path repair at `070f51c1828a123530167d985890039579b28068`; its six reviewed implementation/test files are unchanged in the release. The coordinating reviewer inspected the later notices and bootstrap follow-ups. The [authoring pilot](authoring-pilot.md) was interrupted by a provider usage limit after two completed actors, and its planned blind semantic review did not run. Those measurements make no productivity or cost claim and are not release attestations.

## v0.3.1 published release

At the time this assessment was recorded, the immutable [`v0.3.1` release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.3.1) was verified against source commit [`52f0bd6`](https://github.com/Glacius-Labs/Markitect/commit/52f0bd64ce64759be10ffdee180c2cf7c939d49b), release ID `400495147`. Main CI [run 36787681871](https://github.com/Glacius-Labs/Markitect/actions/runs/36787681871) and Release CI [run 36787725843](https://github.com/Glacius-Labs/Markitect/actions/runs/36787725843) passed for that release source. This release predates the minimal initialization behavior described in the current source.

## Project initialization behavior

Version 0.4.0 adds minimal project initialization. Preview is read-only; write recomputes and validates its plan, requires a named non-protected branch, and creates only the Project file and one area README. Structural checking is not complete verification: an initialized project with no owner-declared checks must still receive `incomplete-evidence` from `verify`. The product also provides a code-to-documentation example, bundled third-party notices, and corrected Windows short/long-path handling. Multi-file writes remain non-transactional and the resource API remains alpha.

## v0.2.0 release

The immutable [`v0.2.0` release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.2.0) is verified against source commit [`8ff0af4`](https://github.com/Glacius-Labs/Markitect/commit/8ff0af4). GitHub Actions [run 36777150514](https://github.com/Glacius-Labs/Markitect/actions/runs/36777150514) passed for that exact release source. This confirms distribution availability; it does not establish acceptance by an adopting project.

## v0.3.0 source scope

This version adds deterministic offline content archives, exact Project pins, explicit exports, origin-aware dependencies and context, and read-only imported resources. It also recognizes orphaned aggregate rule-adapter outputs during drift checks. The package source and consumer examples exercise the public CLI; the archive, graph, context, impact and review tests cover integrity failures and boundary isolation.

The release requires a reviewed fixed commit, the complete Windows/Linux CI matrix, deterministic bundle and installation smoke tests, and verified immutable release assets. The release's provenance asset records the exact source and hosted run; use that record for publication status rather than treating this source assessment as a release receipt. See [Content packages](content-packages.md) for supported behavior and [Operations](operations.md) for the publication procedure.

### v0.3.1 correction

The cycle detector also rejects an implementation that needs a Contract bound directly back to itself. Version 0.3.0 omitted that self-selected execution relationship and could accept this invalid graph. Regression tests cover both Project-local and package-local bindings; existing immutable releases remain unchanged.

## Historical v0.1.0 source and release evidence

The release is bound to merge commit [`45e0017a3d25d2abd464de934b5ef3a802732d0a`](https://github.com/Glacius-Labs/Markitect/commit/45e0017a3d25d2abd464de934b5ef3a802732d0a), whose tree matches reviewed candidate [`4adb45964a172a77a1355c6ea5ac2596c325b90d`](https://github.com/Glacius-Labs/Markitect/commit/4adb45964a172a77a1355c6ea5ac2596c325b90d).

| Gate | Evidence | Result |
|---|---|---|
| Pull request CI | [Run 36762498178](https://github.com/Glacius-Labs/Markitect/actions/runs/36762498178) | Windows and Linux passed |
| Main CI | [Run 36762893903](https://github.com/Glacius-Labs/Markitect/actions/runs/36762893903) | Passed for the merge commit |
| Release build and smoke tests | [Run 36762948855](https://github.com/Glacius-Labs/Markitect/actions/runs/36762948855) | Source gates and bundle/bootstrap checks passed on both platforms |
| Immutable release | [`v0.1.0`](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.1.0), release ID `400355906` | Published 2026-09-30; tag resolves to the merge commit; release and all four asset attestations verified |

The immutable release contains the complete v0.1.0 distribution. Its outer bundle digest is `9bb4d2badedf806faa9bf2150a0bcd78551b1f69238d53e7b58f49282a4fd6b6` and the nested source archive digest is `c9bc71b0939982e58d2edfe14e18fb5221c0d947dcac4e672e90f2ae612a2628`.

| Asset | SHA-256 |
|---|---|
| `markitect-v0.1.0-bundle.zip` | `9bb4d2badedf806faa9bf2150a0bcd78551b1f69238d53e7b58f49282a4fd6b6` |
| `markitect-v0.1.0-linux-amd64` | `2c338b3601efe54026d5266e592f69cb3e677bfba051f21182d837229a98581f` |
| `markitect-v0.1.0-windows-amd64.exe` | `a6dab3a59c83c29a0f1317982516177be848055b024e07455cafe025847f3431` |
| `markitect-v0.1.0-provenance.yaml` | `41af323fef7b1c1040c25541eb989616724446c5c3222e21fc00046bf2d1cb6a` |

The provenance records source tag/commit, workflow run and attempt, toolchain, and payload digests. It is self-declared; GitHub's immutable-release attestation is authoritative for the tag, commit, and attached assets. GitHub's autogenerated source archive is not the Markitect distribution bundle.

## Product boundary at v0.1.0

The release contained the Go CLI, typed YAML resources and schemas, a dependency graph, structural queries, fixed-snapshot context and impact, review-evidence eligibility, generic and explicitly selected provider views, core authoring resources, and the versioned release installer. The source and bootstrap ran on Windows amd64 and Linux amd64 with Go 1.27.1 or later. The native release binary could preview or apply installation without Go.

The checks establish the product gates that ran. They do not prove prose is true or complete, infer a project's policy, certify an agent's runtime behavior, or constitute an adopting project's decision. The v0.1.0 assets above remain an immutable historical pin; source version 0.2.0 and later versions do not alter them.

## Historical publication route

The tagged workflow built and tested the exact source and uploaded a run-scoped transport artifact. It did not publish a GitHub Release. The Actions `GITHUB_TOKEN` could not read the admin-only immutable-release setting (HTTP 403), so an authorized owner used the local Go publisher and existing GitHub CLI authentication. No personal access token was stored in workflow secrets. The publisher checked the run artifact, exact tag target, asset digests, and immutable result before reporting success.
