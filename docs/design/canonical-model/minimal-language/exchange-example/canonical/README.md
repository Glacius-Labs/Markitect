# Canonical Core byte vector

[core.json](core.json) contains exactly 3769 UTF-8 bytes for markitect-core 1.0.0 under the selected [model-digest/1 contract](../../language-contract.md#canonical-bytes). It has no BOM or terminating newline.

SHA-256: 562a20613c64fb9cd3425ff39b5f9fb826b75deb153cdfac48b00fce6f5c19f1.

The design verifier compares newly computed bytes with this fixed artifact and checks the actual [lock binding](../model-lock.yaml). Its explicit canonical export command is documented in the [example guide](../README.md#canonical-bytes-and-real-content-binding). Checking does not rewrite this file or the lock.

The Core is validated in its own model and dependency scope before hashing. The Connector source still contains the separate Command/command collision and receives no canonical digest. This Python design probe does not establish Go product support or a running Connector.
