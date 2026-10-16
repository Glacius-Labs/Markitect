# Unicode 15.1.0 data provenance

These files are the pinned Unicode Character Database inputs for the Markitect
v1 name profile. The source archive is the official Unicode 15.1.0 UCD archive:
<https://www.unicode.org/Public/zipped/15.1.0/UCD.zip>

The archive SHA-256 was verified before extraction:

```text
cb1c663d053926500cd501229736045752713a066bd75802098598b7a7056177  UCD-15.1.0.zip
```

Only the inputs used by the generator and conformance corpus are vendored:

| File | Use | SHA-256 |
| --- | --- | --- |
| `UnicodeData.txt` | General categories, canonical decompositions, and combining classes | `2fc713e6a31a87c4850a37fe2caffa4218180fadb5de86b43a143ddb4581fb86` |
| `CaseFolding.txt` | Full default case folding (statuses C and F) | `4e55acfdc32825a22e87670e9056a3bf94ad7c5400065778e9e10f8314372bcf` |
| `DerivedCoreProperties.txt` | XID_Start, XID_Continue, Lowercase, and Uppercase | `f55d0db69123431a7317868725b1fcbf1eab6b265d756d1bd7f0f6d9f9ee108b` |
| `DerivedNormalizationProps.txt` | Full_Composition_Exclusion | `8875dccee2bc1a7c1fe568a3b502a9e78c9e0495afd96b6568b4294d0ed1f7e1` |
| `NormalizationTest.txt` | The official normalization conformance corpus | `871238e37e3be0696ec2bd0891119a041b052da1a84485eda05a5438724b223e` |

The license snapshot was obtained from
<https://www.unicode.org/license.txt> and its SHA-256 was verified as
`e7a93b009565cfce55919a381437ac4db883e9da2126fa28b91d12732bc53d96`.
The complete snapshot is `UNICODE-LICENSE-V3.txt`. Generated Go source carries
the full Unicode notice because production source packaging includes the Go
tables but excludes these text inputs.

Run `python internal/model/generate_unicode.py` from the repository root to
regenerate `unicode_tables.go` and `unicode_corpus_test.go`. Run the same
command with `--check` to verify byte-for-byte reproducibility. The generator
checks every input and license SHA-256 before producing output. It uses only
these checked-in files and Python's standard library; it does not consult host
Unicode tables, locale settings, or the network.

The generated tables include Unicode 15.1.0 XID and category ranges, canonical
decomposition and composition mappings, canonical combining classes, and full
default case-fold mappings. Composition mappings omit every scalar in
Full_Composition_Exclusion. Test corpus constants hold gzip-compressed,
base64-encoded exact source files so conformance tests remain self-contained
when only Go source files are packaged.
