# C12 goal-model proposal trial 02

Run label: `c12-goal-propose-02`
Protocol binding: `operating-model-proof/v5`, commit `658866480bafa37c8c2e94b5d58fc2074655f416`, protocol SHA-256 `9b0c42914ddf5d2337ac6a20de0a666242d659f9c7cc92851c2f5c472d0e9635`. This is a separate fresh attempt; the prior invalid proposal result remains unchanged.

## Frozen bindings

- Product source: `0cf8d22fb9fb03bd87140eb01a84c79e54647ce6`.
- Build receipt: `build-receipt-goal-reference.json`, exit code 0, Go 1.27.1; CLI binary SHA-256 `3740439442b223964899fa723ac8a23ba4525e023d07383755e19d06189a7908`.
- Runner wrapper SHA-256 `7806fabc774e2127ea29977009745c49d8043ffbcba63390e3d790b82d164d57`; Codex CLI 0.130.0, `gpt-5.5`, high reasoning, Python 3.13.
- Exact unchanged goal input SHA-256 `810859a4e97616dd156e998ed5f76aa7bf63fd8c6f8f8dc883ab4ecfb088e9e9`.
- Exact original recommendation result SHA-256 `7055ae20a4c35580a489f1a26768b9fb2fae8a34751cc201deb0a2100534d20a`; recommendation digest `sha256:0e240974779360e037a0d7a59f0a3d803c4ff7eadb9acbbdf77b824db0cb3f22`.
- Exact technical selection SHA-256 `c24dcb48628bd1d54cbe2597cfa25851b4c51a6d89cf10b9502d6e33e804b44d`; selected exact Module ID `unknown-ontology-mission@1.0.0#sha256:7cbb9964571e912e416ca1e56ad9e66522d1f49a36d230216a160fb4b3f374be`.
- Pre-call runtime JSON SHA-256 `5f41207b3626ce9f076eb794afc0b79f993815f843133121ebb72711561d490c`; exact agentexec config digest `sha256:d4a22030ea003e3aafe38d1895d18936e83eb7b12892623f793f47b18f46f746`.
- Pre-call freeze manifest SHA-256 `5ee6ada1bd1e1ff0177444d5fa2b5f8967deb2dc380f8eb71310695d1fd79906`.
- Goal/catalog digests: `sha256:e1246709c39b8868cf159f233519901c538da036e963b1851708c9cc01831252` / `sha256:0b7cd9e45bc98a4ec09a8f15403b057c50e0035259ef303f4311fb3e9edce049`.
- Selected CoreSchemas digest `sha256:80a8478ffb16c1c1f079ec1cbc64ff70fc9b3ed5b6a01d04a0a082ee999fc1de`; input-binding digest `sha256:40cad91a36af5073c55c09e229e613ca2b5fe9e510dc6b21cc40508dde19cea6`; precomputed context digest `sha256:c880873728a42f0707260b3171f85508ec57f56cee574b5f3b1f43cd2ad2caa2`; precomputed request/input digest `sha256:f1097a4cb39832cefed5be1fedc4af9a11319ec2f832bf43f8cf4183d115e9ff`.

## Observed result

Exactly one fresh native `goal-propose` call ran, without retry. CLI exit code was 2; status was `incomplete`; elapsed CLI wall time 19,144 ms; Host receipt wall time 18,529 ms. The Host error was `external response does not bind to this invocation`. Receipt run ID: `f6786fd479e7afb32ae4e282aa96db8a`, outcome `incomplete`, retry count 0, accepted false, adopted false. Actual receipt input digest and context digest matched the pre-call values above. Actual config digest matched; runtime-files digest was `sha256:5898bea52150c07d81b7b0e1b690d0672da6b5ce0a76475733dd168b83e0a9b5`.

No candidate JSON or candidate digest was returned, and Core compilation was not reached. There are no diagnostics or provider-reported usage values. CLI stdout digest was `sha256:e69c4a6bfdc509aa44b28524a38fa7a939e0f605e8359a371e98ef359934d30b`; stderr was empty with digest `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`; private runner log digest was `sha256:30f04dc60e381ae05848c121c999fea1012e3947d10e84b5f3d1221461ee4604`. Raw run outputs and logs remain external under the run label. No candidate is available for technical review. This stage remains **FAIL/PARTIAL**; it establishes neither compilation nor owner acceptance, adoption, reconciliation, or verification.
