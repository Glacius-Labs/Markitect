# C12 goal-propose-02 binding diagnosis

Run label: `c12-goal-propose-02`. Protocol: `operating-model-proof/v5`, commit `658866480bafa37c8c2e94b5d58fc2074655f416`, document SHA-256 `9b0c42914ddf5d2337ac6a20de0a666242d659f9c7cc92851c2f5c472d0e9635`.

## Binding comparison

The Host receipt and frozen request bind `inputDigest` to `sha256:f1097a4cb39832cefed5be1fedc4af9a11319ec2f832bf43f8cf4183d115e9ff` (71 characters). The single final model response contained `sha256:f1097a4cb39832cefed5be1fed5be1fedc4af9a11319ec2f832bf43f8cf4183d115e9ff` (78 characters). The first differing character is at zero-based position 33; the response has an extra `5be1fe` sequence.

The comparable binding fields matched: `apiVersion=markitect.example.org/agent-execution/v1alpha1`, `runId=f6786fd479e7afb32ae4e282aa96db8a`, and `role=infer`. The final response nonce was `3b14d0e61f5cc7c7860ab54d37811483`; the expected invocation nonce was not recorded in the freeze or receipt, so it cannot be independently compared.

## Attribution and outcome

The frozen wrapper (`7806fabc774e2127ea29977009745c49d8043ffbcba63390e3d790b82d164d57`) parses the final response and normalizes only `candidateJson`; it passes binding fields through unchanged. The Host therefore correctly rejected the mismatched `inputDigest` with `external response does not bind to this invocation`. This is a model final-response binding error, not a wrapper envelope assembly defect.

The one fresh invocation ended `incomplete`; no retry was made. The response did not reach candidate compilation. This diagnosis did not invoke a provider, alter the prior failed result, or modify raw evidence.

## Evidence identity

- Product source commit: `0cf8d22fb9fb03bd87140eb01a84c79e54647ce6`.
- Frozen binary SHA-256: `3740439442b223964899fa723ac8a23ba4525e023d07383755e19d06189a7908`.
- Runner wrapper SHA-256: `7806fabc774e2127ea29977009745c49d8043ffbcba63390e3d790b82d164d57`.
- Private raw log SHA-256: `30f04dc60e381ae05848c121c999fea1012e3947d10e84b5f3d1221461ee4604`.
- Final model response text SHA-256: `1de5d0c5427292ab71aba9c717d6cda759dd277cf7ca570526f122063118ec00`.
