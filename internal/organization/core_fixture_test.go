package organization_test

import "github.com/Glacius-Labs/Markitect/internal/model"

// Captured from internal/model/core_fixture_test.go at fc56336b29e98c06e92d57377b67fc7a9663ed12.
// Tests carry their own Go-only Core source; no runtime Docs dependency.
// coreSource is the normative §3.1 Core fixture. Keep the source YAML
// and folded body text aligned with the nine definitions in the contract.
func coreSource() model.SourceModel {
	entries := []model.Entry{
		{Path: "model/markitect/core/kind.yaml", Kind: model.Regular, Bytes: []byte(`id: Kind
kind: Kind
body: >-
  Deklariert einen Typ. properties benennt die erlaubten Felder
  der spec seiner Definitionen. Der body einer Typdefinition beschreibt
  deren gemeinsame Bedeutung und geltende Festlegungen.
spec:
  properties:
    - properties
`)},
		{Path: "model/markitect/core/property.yaml", Kind: model.Regular, Bytes: []byte(`id: Property
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
		{Path: "model/markitect/core/name.yaml", Kind: model.Regular, Bytes: []byte(`id: name
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
		{Path: "model/markitect/core/type.yaml", Kind: model.Regular, Bytes: []byte(`id: type
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
		{Path: "model/markitect/core/min-count.yaml", Kind: model.Regular, Bytes: []byte(`id: minCount
kind: Property
body: Nichtnegative Mindestanzahl vorhandener Werte.
spec:
  type: integer
  minCount: 1
  maxCount: 1
`)},
		{Path: "model/markitect/core/max-count.yaml", Kind: model.Regular, Bytes: []byte(`id: maxCount
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
		{Path: "model/markitect/core/values.yaml", Kind: model.Regular, Bytes: []byte(`id: values
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
		{Path: "model/markitect/core/target.yaml", Kind: model.Regular, Bytes: []byte(`id: target
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
		{Path: "model/markitect/core/properties.yaml", Kind: model.Regular, Bytes: []byte(`id: properties
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
	return model.SourceModel{
		Origin:       "core",
		ManifestPath: "model.yaml",
		Entries:      append(append([]model.Entry{{Path: "model.yaml", Kind: model.Regular, Bytes: []byte("formatVersion: 1\nid: markitect-core\nversion: '1.0'\nroot: model\ndependencies: {}\n")}}, entries...), emptyLock()),
	}
}
