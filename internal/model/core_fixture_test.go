package model

// completeCoreSource is the normative §3.1 Core fixture. Keep the source YAML
// and folded body text aligned with the nine definitions in the contract.
func completeCoreSource() SourceModel {
	entries := []Entry{
		{Path: "model/markitect/core/kind.yaml", Kind: Regular, Bytes: []byte(`id: Kind
kind: Kind
body: >-
  Deklariert einen Typ. properties benennt die erlaubten Felder
  der spec seiner Definitionen. Der body einer Typdefinition beschreibt
  deren gemeinsame Bedeutung und geltende Festlegungen.
spec:
  properties:
    - properties
`)},
		{Path: "model/markitect/core/property.yaml", Kind: Regular, Bytes: []byte(`id: Property
kind: Kind
body: >-
  Deklariert einen Feldvertrag mit Werttyp und Kardinalität.
  Der body beschreibt die Bedeutung seiner Werte oder Beziehungen.
  Bei type object benennt properties die erlaubten Felder des Objektwerts.
spec:
  properties:
    - name
    - type
    - minCount
    - maxCount
    - values
    - target
    - properties
`)},
		{Path: "model/markitect/core/name.yaml", Kind: Regular, Bytes: []byte(`id: name
kind: Property
body: >-
  Optionaler ausdrücklich abweichender nichtleerer Feldname.
  Ohne Angabe gilt die vollständige lokale ID der Property als Feldname.
  Der effektive Feldname muss innerhalb jedes verwendenden Vertrags
  eindeutig sein.
spec:
  type: string
  minCount: 0
  maxCount: 1
`)},
		{Path: "model/markitect/core/type.yaml", Kind: Regular, Bytes: []byte(`id: type
kind: Property
body: Der vom Core verstandene Werttyp dieser Property.
spec:
  type: enum
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
body: Nichtnegative Mindestanzahl vorhandener Werte.
spec:
  type: integer
  minCount: 1
  maxCount: 1
`)},
		{Path: "model/markitect/core/max-count.yaml", Kind: Regular, Bytes: []byte(`id: maxCount
kind: Property
body: >-
  Höchstanzahl vorhandener Werte. Der Wert -1 erlaubt beliebig viele Werte
  innerhalb der Werkzeuggrenzen. Andere Werte müssen mindestens minCount
  entsprechen. Genau 1 bezeichnet einen Einzelwert; andere Höchstwerte
  eine Liste.
spec:
  type: integer
  minCount: 1
  maxCount: 1
`)},
		{Path: "model/markitect/core/values.yaml", Kind: Regular, Bytes: []byte(`id: values
kind: Property
body: >-
  Erlaubte Stringwerte einer enum-Property.
  Bei type enum ist eine nichtleere Liste eindeutiger Werte erforderlich.
  Bei anderen Werttypen ist diese Angabe nicht zulässig.
spec:
  type: string
  minCount: 0
  maxCount: -1
`)},
		{Path: "model/markitect/core/target.yaml", Kind: Regular, Bytes: []byte(`id: target
kind: Property
body: >-
  Der verlangte Kind der referenzierten Definitionen.
  Bei type reference ist diese Angabe erforderlich.
  Bei anderen Werttypen ist sie nicht zulässig.
spec:
  type: reference
  target: Kind
  minCount: 0
  maxCount: 1
`)},
		{Path: "model/markitect/core/properties.yaml", Kind: Regular, Bytes: []byte(`id: properties
kind: Property
body: >-
  Referenziert die erlaubten Feldverträge des hier beschriebenen Objekts.
  Die effektiven Feldnamen müssen innerhalb dieses Vertrags eindeutig sein.
spec:
  type: reference
  target: Property
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
	input.Root.Entries[len(input.Root.Entries)-1].Bytes = []byte("formatVersion: 1\nmodels:\n  markitect-core:\n    version: '1.0'\n    digest: sha256:b912d62b4e7ecdb181291f1dafa712e8a0f407ad35d336ac2c3420a523ec3de4\n")
	return input
}
