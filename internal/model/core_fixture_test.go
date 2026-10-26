package model

// completeCoreSource is the normative §3.1 Core fixture. Keep the source YAML
// and folded body text aligned with the ten definitions in the contract.
func completeCoreSource() SourceModel {
	entries := []Entry{
		{Path: "model/markitect/core/kind.yaml", Kind: Regular, Bytes: []byte(`id: Kind
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
		{Path: "model/markitect/core/property.yaml", Kind: Regular, Bytes: []byte(`id: Property
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
		{Path: "model/markitect/core/name.yaml", Kind: Regular, Bytes: []byte(`id: name
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
		{Path: "model/markitect/core/value-type.yaml", Kind: Regular, Bytes: []byte(`id: valueType
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
		{Path: "model/markitect/core/min-count.yaml", Kind: Regular, Bytes: []byte(`id: minCount
kind: Property
body: Non-negative minimum number of present values.
spec:
  valueType: integer
  minCount: 1
  maxCount: 1
`)},
		{Path: "model/markitect/core/max-count.yaml", Kind: Regular, Bytes: []byte(`id: maxCount
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
		{Path: "model/markitect/core/values.yaml", Kind: Regular, Bytes: []byte(`id: values
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
		{Path: "model/markitect/core/target.yaml", Kind: Regular, Bytes: []byte(`id: target
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
		{Path: "model/markitect/core/properties.yaml", Kind: Regular, Bytes: []byte(`id: properties
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
		{Path: "model/markitect/core/implements.yaml", Kind: Regular, Bytes: []byte(`id: implements
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
	return SourceModel{
		Origin:       "core",
		ManifestPath: "model.yaml",
		Entries:      append(append([]Entry{{Path: "model.yaml", Kind: Regular, Bytes: []byte("formatVersion: 1\nid: markitect-core\nversion: '1.0'\nroot: model\ndependencies: {}\n")}}, entries...), emptyLockEntry()),
	}
}

func coreModelInput(definitions ...Entry) Input {
	input := Input{
		Root:         bindingSource("project", "", "{markitect-core: '1.0'}", definitions...),
		Dependencies: []SourceModel{completeCoreSource()},
	}
	// Independently fixed by replacing only version in the normative Core JSON
	// vector, rather than asking the implementation to generate its own lock.
	input.Root.Entries[len(input.Root.Entries)-1].Bytes = []byte("formatVersion: 1\nmodels:\n  markitect-core:\n    version: '1.0'\n    digest: sha256:a972d8e445a13f87a0e1c43fb635f57085926be5f73ac10c104e6778212dbed0\n")
	return input
}
