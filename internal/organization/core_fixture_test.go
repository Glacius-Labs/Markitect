package organization_test

import "github.com/Glacius-Labs/Markitect/internal/model"

// Captured from internal/model/core_fixture_test.go with the English bodies of markitect-core 1.2.0.
// Tests carry their own Go-only Core source; no runtime Docs dependency.
// coreSource is the normative §3.1 Core fixture. Keep the source YAML
// and folded body text aligned with the ten definitions in the contract.
func coreSource() model.SourceModel {
	entries := []model.Entry{
		{Path: "model/markitect/core/kind.yaml", Kind: model.Regular, Bytes: []byte(`id: Kind
kind: Kind
body: >-
  Declares a type. properties names the permitted fields of the spec
  of its definitions. The body of a type definition describes their
  shared meaning and applicable stipulations. implements names interface
  Kinds; the type declares every Property of each in properties.
spec:
  properties:
    - properties
    - implements
`)},
		{Path: "model/markitect/core/property.yaml", Kind: model.Regular, Bytes: []byte(`id: Property
kind: Kind
body: >-
  Declares a field contract with value type and cardinality.
  The body describes the meaning of its values or relationships.
  For value type object, properties names the permitted fields of the object value.
spec:
  properties:
    - name
    - valueType
    - minCount
    - maxCount
    - values
    - target
    - properties
`)},
		{Path: "model/markitect/core/name.yaml", Kind: model.Regular, Bytes: []byte(`id: name
kind: Property
body: >-
  Optional explicitly deviating non-empty field name.
  If omitted, the complete local ID of the Property applies as the field name.
  The effective field name must be unique within every contract
  that uses it.
spec:
  valueType: string
  minCount: 0
  maxCount: 1
`)},
		{Path: "model/markitect/core/value-type.yaml", Kind: model.Regular, Bytes: []byte(`id: valueType
kind: Property
body: The value type of this Property, understood by the Core.
spec:
  valueType: enum
  minCount: 1
  maxCount: 1
  values:
    - string
    - boolean
    - integer
    - number
    - enum
    - object
    - reference
`)},
		{Path: "model/markitect/core/min-count.yaml", Kind: model.Regular, Bytes: []byte(`id: minCount
kind: Property
body: Non-negative minimum number of present values.
spec:
  valueType: integer
  minCount: 1
  maxCount: 1
`)},
		{Path: "model/markitect/core/max-count.yaml", Kind: model.Regular, Bytes: []byte(`id: maxCount
kind: Property
body: >-
  Maximum number of present values. The value -1 permits any number of values
  within the tool limits. Other values must be at least minCount.
  Exactly 1 denotes a single value; other maximum values denote
  a list.
spec:
  valueType: integer
  minCount: 1
  maxCount: 1
`)},
		{Path: "model/markitect/core/values.yaml", Kind: model.Regular, Bytes: []byte(`id: values
kind: Property
body: >-
  Permitted string values of an enum Property.
  For value type enum, a non-empty list of distinct values is required.
  For other value types, this entry is not permitted.
spec:
  valueType: string
  minCount: 0
  maxCount: -1
`)},
		{Path: "model/markitect/core/target.yaml", Kind: model.Regular, Bytes: []byte(`id: target
kind: Property
body: >-
  The Kind required of the referenced definitions.
  Definitions of a Kind that implements the required Kind, directly
  or through other interfaces, also satisfy it.
  For value type reference, this entry is required.
  For other value types, it is not permitted.
spec:
  valueType: reference
  target: Kind
  minCount: 0
  maxCount: 1
`)},
		{Path: "model/markitect/core/properties.yaml", Kind: model.Regular, Bytes: []byte(`id: properties
kind: Property
body: >-
  References the permitted field contracts of the object described here.
  The effective field names must be unique within this contract.
spec:
  valueType: reference
  target: Property
  minCount: 0
  maxCount: -1
`)},
		{Path: "model/markitect/core/implements.yaml", Kind: model.Regular, Bytes: []byte(`id: implements
kind: Property
body: >-
  References the interface Kinds a Kind implements. The Kind must declare
  every Property of each interface in its own properties, and its
  definitions satisfy references that require the interface, directly
  or indirectly. The implements graph must be acyclic.
spec:
  valueType: reference
  target: Kind
  minCount: 0
  maxCount: -1
`)},
	}
	return model.SourceModel{
		Origin:       "core",
		ManifestPath: "model.yaml",
		Entries:      append(append([]model.Entry{{Path: "model.yaml", Kind: model.Regular, Bytes: []byte("formatVersion: 1\nid: markitect-core\nversion: '1.0'\nroot: model\ndependencies: {}\n")}}, entries...), emptyLock()),
	}
}
