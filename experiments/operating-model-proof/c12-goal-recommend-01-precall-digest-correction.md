# C12 recommendation pre-call digest correction

This immutable supplement records a provenance discrepancy for `c12-goal-recommend-01`; it does not replace or rewrite that run record.

The stage-1 freeze note recorded offline reconstructed digests before invocation:

- Goal: `sha256:e1246709c39b8868cf159f233519901c538da036e963b1851708c9cc01831252`
- Catalog: `sha256:0b7cd9e45bc98a4ec09a8f15403b057c50e0035259ef303f4311fb3e9edce049`
- Reconstructed context: `sha256:a995bdc8901c7ade3bc69d6e3db82e431889aad0da15a6a2e43481285f995fdd`
- Reconstructed request/input: `sha256:dbe0095382b9aec555bc08a77fe6ce4592eac94c79804897b6d0d97c8b332913`

The actual Host receipt preserved in the external run output matched the goal and catalog digests, but recorded different context `sha256:f50da2b3b33a72344b1a11840ae1e59eba937f36245442d80adc5b87438c1e28` and request/input `sha256:a343b609b844a9f05f316eefc014b68cf724b02f1094acf1a2d8710ead86f490`. The pre-call reconstruction was wrong. The cause has not been established; no post-run reconstruction or correction is represented as a pre-call fact. This limits the stage-1 freeze-to-invocation provenance claim. The provider result itself and all original bytes/receipts remain unchanged; there was no retry.
