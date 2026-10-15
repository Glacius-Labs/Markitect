# Quellenabgleich mit aktuellem Main

Datum: 15. Oktober 2026. Feste Main-Basis: `6472d42a93c1be7e542d1fa06ad10a308dd15b8a`. Gegenstand: die im [Konsolidierungsstand](consolidation.md) fixierten Designquellen. Dies ist ein gezielter Quellenvergleich der Sprachgrenzen, keine vollständige Implementierungsplanung und kein Laufzeittest.

## Tatsächlicher Main-Vertrag

| Bereich | Befund an Main | Quelle |
| --- | --- | --- |
| Projektvokabular | Die fünf Kinds heißen Manager, Statement, Artifact, Check und Decision. Concept, Rule, UseCase, Architektur und Workflow werden über Statement-Kategorien beschrieben. | [Schema](../../../src/internal/core/projectmodel/projectmodel.go) |
| Verantwortung | Manager enthält parent, owns und instructions. Die vorgeschlagene Umbenennung zu Mandate ist nicht implementiert. | [Schema](../../../src/internal/core/projectmodel/projectmodel.go), [Dokumentations-Manager](../../../.markitect/model/documentation/manager.yaml) |
| Beschreibung | Das aktuelle Definitionsformat hat purpose; Statement besitzt spec.description. Die vorgeschlagene gemeinsame Körperform und die Symbolsyntax sind keine unterstützten Eingaben. | [Core-Typen](../../../src/internal/core/types.go), [Schema](../../../src/internal/core/projectmodel/projectmodel.go) |
| Referenzen | Eine Reference-Property bindet einen Zielkind über Target. Das Projektvokabular nutzt feste Statement-, Manager- und Check-Ziele. Der vNext-Kandidat mit gemischten Zielmengen benötigt einen gesonderten Vertrag. | [Core-Typen](../../../src/internal/core/types.go), [Schema](../../../src/internal/core/projectmodel/projectmodel.go) |
| Eingabenauswahl | project.yaml führt eine ausdrückliche modelFiles-Liste und einen Zielpfad für die generierte Dokumentation. Das ist nicht der entworfene rekursive Modellordner-Loader. | [Projektkonfiguration](../../../.markitect/project.yaml) |
| Dateiabdeckung | Das eigene Projekt verwendet coverageMode full. docs/ wird durch das Artifact documentation-pages erfasst und vom Dokumentations-Manager verantwortet. Neue Designseiten passen in diese vorhandenen Präfixe. | [Projektkonfiguration](../../../.markitect/project.yaml), [Artifact](../../../.markitect/model/documentation/documentation-pages.yaml), [Manager](../../../.markitect/model/documentation/manager.yaml) |
| Prüfungen | Check verlangt command mit mindestens einem Argument; uses bindet Statements. Eine beschriebene manuelle Prüfmethode wird damit nicht automatisch ausführbar. | [Schema](../../../src/internal/core/projectmodel/projectmodel.go) |
| Entscheidungen | Decision.subject ist ein Statement, Decision.actor ist ein Manager. Ein neuer Actor-Begriff kann diese Felder nicht allein durch Namensähnlichkeit übernehmen. | [Schema](../../../src/internal/core/projectmodel/projectmodel.go) |

Der Main-Stand enthält inzwischen die aktuellen Produktverzeichnisse Core, Infrastructure, Application und Runtime sowie den gemeinsamen operativen Record Store. Die [Architektur](../../architecture.md) erklärt diese Grenzen. Ein früher Implementierungsplan mit älteren Paketpfaden muss vor Umsetzung gegen diese Quellen erneuert werden. Die [Ursprungserklärung](../../concepts/origin-and-rationale.md) ist bereits in diesem Main enthalten.

## Folgerungen für diese Konsolidierung

Die Designunterlagen werden auf Main zusammengebracht, ohne Projektkonfiguration, bestehendes Schema oder Laufzeit zu verändern. Insbesondere werden `vNext`-Beispiele nicht als kompilierbare Main-Modelle behandelt. Es wird weder eine Schemaänderung über ein Dokumentationsbeispiel eingeschleust noch die historische Prüfung vom 11. Oktober als neuer Nachweis ausgegeben.

Die wichtigsten Umsetzungsfragen bleiben: Version und Migration, Auswahl und Bedeutung der Kinds, Körper und Symbolauflösung, Referenzzielmengen, Zusammensetzung und Realisierungsumfang, Verantwortung und Dateirechenschaft sowie Prüfmethoden. Die [Konfliktliste](consolidation.md#widersprüche-und-entscheidungen) trennt diese Fragen nach ihrem konkreten Ursprung.

Dieser Vergleich bestätigt Quellenbefunde an genau dem genannten Stand. Eine spätere Umsetzung muss ihren dann aktuellen Main erneut fixieren und passende Tests ergänzen. Er ist kein Nachweis, dass die vorgeschlagene Sprache oder eines der Beispielprojekte bereits funktioniert.
