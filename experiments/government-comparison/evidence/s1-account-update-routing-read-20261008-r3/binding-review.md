# Independent request and freeze binding review

I independently checked the R3 request and freeze against the committed source and live authority. I found no binding blocker.

The committed source is `d5997d6392ae3f6397bd45a21e3ed4946b1b768c`. The request binds that source; all 217 request source pins match both current file bytes and their Git blobs at `HEAD`. All 232 frozen input hashes match current bytes. The request hash is `6c8c15827f790ae47b971b738ed355a94a91a446d5e7915ddc84b536dc815f73`; the freeze hash is `8953d04f970d87ccc0a5c26381def7cc28f5c761209214ce081cbb71f9e83ebe`. Request, freeze, and authorization-grant/profile hashes cross-reference exactly, and the freeze was created after grant issuance.

The copied grant matches the current canonical Scientist grant semantically. The active Scientist slot has the same key, owner, grant issue time, and assignment time, with no release fields. Its readiness-window binding matches the current window: ordinal 2, final allocation, one remaining considered attempt, one already consumed/reserved in the resumed block, and the recorded deadline/latest-start limits. Initial actual counts for this new allocation remain zero. The external evidence root is absent, so there is no reservation or run artifact.

I checked the five read-only historical ledger pins against the freeze. The frozen executable and schema archive hashes match the live grant. The pinned interpreter and the real R2 client/`run_once.py` bases are present in the binding. The account observer pin matches the accepted byte-identical observer. The two authorized public CWD files match their frozen/current hashes; direct read-only directory metadata also matched the grant’s volume serial, file index, attributes, same-file identity, resolved path, and exact two-entry inventory for both spellings.

No tests or processes were run for this review. This review confirms source and binding consistency only; it does not satisfy the runtime reservation or final live pre-launch checks.
