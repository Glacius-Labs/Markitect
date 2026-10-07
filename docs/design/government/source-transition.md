# Government-Betriebsmodell: Übergang vom heutigen Markitect

**Designbasis:** 7. Oktober 2026; initiale Government-Implementierungsbasis ist
der geprüfte Classic-Kandidat `1ea5c76f55526fc4d721e865885436153f48b497`.
Die Quellverträge wurden mit `git show <SHA>:<Pfad>` im Worktree
`standard-operating-model` geprüft; das ist kein Claim über einen anderen
Branch-HEAD. Der Koordinator kann die Baseline nach expliziten Checks auf einen
anderen Commit wechseln. Es besteht keine Wartepflicht auf einen späteren
Architect-Stand. Dieses Dokument ist ein Design, keine Umsetzungsfreigabe.
Die [Änderungslandkarte](../delegated-engineering-change-map.md) ordnet die
Quellverträge ein; verbindliche Government-Begriffe und Modellpflichten stehen
in der [Architektur](architecture.md), Lieferfolge und Gates im
[Delivery-Plan](delivery-plan.md).

## Zielbild und Invarianten

Government ordnet technische Arbeit nach den in `architecture.md` definierten
Bereichen und Ressorts. **Bereich** ist der kanonische rekursive Typ für
hierarchische Verantwortung: Er beschreibt Zweck, Elternbereich,
verantwortete Definitionen und Artefaktumfang sowie Fähigkeiten zur Ausführung
und Prüfung. Bereiche delegieren begrenzte Aufträge nach unten; Kinder
berichten Kandidaten und Nachweise zurück; Eltern integrieren und prüfen die
Änderung. **Ressort** ist eine dauerhafte fachliche Zuständigkeit, die quer
über Bereiche und Dateien prüft, aber kein Dateiwriter ist. Technologie ist
eine Fähigkeit innerhalb der Organisation; Agenteninstanzen sind kurzlebige
Durchläufe, keine Modellobjekte.

Die Projektordnung bleibt die akzeptierte Soll-Autorität. Beobachteter
Repository-Zustand, externe/vendor-Dateien und Agentenvorschläge werden dadurch
nicht automatisch zu Soll. Jede Übernahme bindet exakte Kandidatenbytes,
Ordnungsversion, Mandate, Aufgaben und Prüfberichte. Zuständigkeit und
Delegation müssen prüfbar sein; Digest-Bindung oder ein erfolgreicher Prozess
authentifizieren keine menschliche oder organisatorische Autorität.

## Behaltene Grenzen und nötige Verschiebungen

Die neue Variante darf Classic-Zuschnitte und Schnittstellen neu ordnen, aber
die Architekturpflichten in `architecture.md` nicht aus Kompatibilitätsgründen
oder als Implementierungsvereinfachung streichen: akzeptierte Projektordnung,
Purpose, expliziter Graph, bidirektionale Modell-/Datei-Realisierung,
rekursive Ausführung und unabhängige Prüfung, Elternintegration sowie
Ressortzustimmung zum feststehenden Gesamtkandidaten. Am Baseline-Commit bleibt
`internal/core` strukturell und deterministisch:
Schema, Definition, Zweck, typisierte Referenzen, normalisiertes Modell und
Provenienz. Core darf kein Ressort-Urteil, Agentenaufruf, Laufzeitrecht oder
Promotionsentscheid werden. Der öffentliche Bereich ist ein versionierter
Modelltyp mit Zweck, Parent und expliziten Zuständigkeits-IDs; Host validiert
bereichsübergreifende Zuständigkeit und löst exakte Identitäten auf. Der
aktuelle Core-Referenzvertrag (`internal/core/types.go`,
`internal/core/compile.go`) referenziert jeweils genau einen Kind-Typ. Für
Bereiche, die beliebige Definition-Kinds verantworten, braucht es daher einen
Host-validierten Identitätswert oder einen typisierten Zuordnungsvertrag; keine
scheinbar polymorphe Core-Referenz behaupten.

Die dauerhafte Zielvorgabe entspricht weiterhin der gewünschten Repräsentation
mit exaktem Definitionsumfang, Ziel und Policies (`docs/canonical-projections.md`,
„Inputs and ownership“). Sie wird begrifflich als Zielvorgabe geführt und ist
nicht Mandat, Auftrag oder Ministerium. Bestehende Bindings und Module bleiben
Ausführungsfähigkeiten. Ein Ressort kann mehrere Zielvorgaben und Fähigkeiten
verantworten. Der Host-Loader, feste Quellenrevisionen, `core.Compile`,
begrenzter Agentenkontext, Impact-Analyse, Kandidatenbytes, Provenienz und
deterministische Diagnostik bleiben nützliche technische Fundamente.

Die heutigen Scopes koppeln Assurance an Projektionen:
`CanonicalAssuranceScope` trägt `ProjectionID`, Children und Checks in
`internal/host/canonical_controller.go`. Das Zielmodell muss Zuständigkeit und
Auftragsbaum unabhängig von Darstellung modellieren; Zielvorgaben werden
explizit an verantwortliche Bereiche gebunden. Eine Darstellung kann mehrere
Bereiche berühren, und ein Bereich mehrere Darstellungen. Explizite
Zuordnungen, Überlappungsregeln und ungelöste Zuständigkeit ersetzen eine
stillschweigende 1:1-Zuordnung.

Die Assurance-DAG- und child-first-Ausführung aus
`internal/host/assurance/assurance.go` und `run.go` ist ein möglicher
Kompositionsmechanismus, kein fertiger Government-Runner. Laufzeitkonfiguration
und Controller-Verträge sind derzeit nicht pro Bereich und pro Rolle
ausgeprägt. Erforderlich sind getrennte Verantwortungsverträge für Executor,
Integration, lokale Verifier und ressortübergreifende Prüfer. Diese binden
jeweils dieselbe Kandidatenrevision sowie Mandat und begrenzte Evidenz.
„Unabhängig“ muss als prüfbare Trennung von Eingaben/Rollen und Berichtspfaden
definiert werden; anderer Modellname allein reicht nicht. Bestehende Prozesse
laufen mit Aufruferrechten und sind keine Betriebssystem-Sandbox oder
Autorisierungsinstanz.

## Vollständigkeit vor autonomer Arbeit

Ein Government-Planer darf keinen Projektumfang aus konfigurierten Modulen,
Projektionen oder Git-Tracked-Dateien allein ableiten. Zuerst wird für eine
owner-ausgewählte vollständige Repository-Eingabe ein Inventar erzeugt, das
alle zugelassenen Pfade nach Herkunft und Rolle klassifiziert: kanonische
Quellen, explizit besessene Repräsentationen/Zielartefakte, externe oder
vendor-Eingaben, ausgeschlossene Pfade und unerklärte/unzugeordnete Dateien.
Auch untracked, ignored oder außerhalb von Git geführte, aber ausdrücklich
zugelassene Projektdateien sind sichtbar; `.git`-Objekte sind Git-Metadaten,
keine Projektartefakte. Ausgeschlossenes bleibt als Ausschluss samt Umfang
sichtbar. Kein automatisches Löschen, Umdeuten, Importieren oder Adoptieren.

Die bestehende Adoption-Vorbereitung (`docs/design/selective-adoption-handoff.md`)
bindet owner-selektierte Git-Blobs, verspricht aber keine automatische
Vollständigkeit. Government braucht eine explizite Erfassungsgrenze und
abgeschlossene Abdeckung oder weist Lücken aus; unbekannte Dateien blockieren
die Behauptung „vollständig“. Der Inventarbericht ist Evidence, keine
Kanonisierung. Brownfield-Inference darf Vorschläge samt Gegenbelegen liefern,
aber nur ein ausdrücklich autorisierter Änderungsauftrag kann eine geprüfte
Änderung der Projektordnung anstoßen.

## Technische Abhängigkeiten der Lieferfolge

Die verbindlichen Funktionsschritte G1–G5 und Studiengates S1–S2 sind im
[Delivery-Plan](delivery-plan.md) definiert. Diese Quellnotiz führt keinen
zweiten Lieferplan; sie hält fest, welche vorhandenen Mechaniken die jeweilige
Stufe tragen können und wo neue Verträge nötig sind.

- **G1 — Modell und Repository-Abdeckung:** Bereichs-/Ressortmodell,
  Zuständigkeit, Parent, Modell-/Datei-Zuordnung, vollständiges klassifiziertes
  Inventar und lesender Plan. `canonical_load`, `core.Compile`, Context und
  Impact sind mögliche Baseline-Bausteine; kein Agentenlauf oder Besitz wird
  daraus abgeleitet.
- **G2 — einzelner vollständiger Auftrag:** zuerst einen Bereich bis zu
  tatsächlicher Kandidatenerstellung, eigenem Review und geschützter
  Übernahme führen. Dafür sind Candidate-Bindung, ressortweite finale Urteile
  und Promotion-Guard neue Verträge; heutiges `controller-apply` reicht nicht.
  Ein interner DAG-/Routing-Prototyp darf vorbereiten, was G3 verallgemeinert,
  ersetzt aber nicht diesen End-to-End-Nachweis.
- **G3 — Rekursion und Zusammensetzung:** anschließend Eltern mit mindestens
  zwei fachlichen Kindern, getrennten Kontexten, echter Zusammenführung und
  eigener Elternprüfung. Assurance-DAG/child-first-Ausführung können als
  Mechanik geprüft werden; Scopes, Rollen und Integrationspflichten müssen
  unabhängig von Zielvorgaben modelliert werden.
- **G4 — Modellpflege und Konflikt:** Vorschläge mit Impact, Belegen und
  Unsicherheit; autorisierte Modelländerung und ungelöste Konflikte gemäß
  aktiver Architekturordnung. Beobachtete Drift allein wird nie Soll.
- **G5 — Dauerbetrieb:** persistente Queue, Ressourcenbegrenzung,
  Wiederaufnahme, Idempotenz und sichere Promotion nach Unterbrechung.
  Endliche Controller-Aktionen belegen diese Eigenschaften nicht.
- **S1/S2 — Readiness und Vergleich:** erst die Kriterien des Delivery-Plans
  erfüllen, dann die drei Arme über Greenfield und Brownfield vergleichen.
  Abdeckung, Fehler, Eingriffe, Reviews, Zeit und Pflegekosten separat messen;
  keine Nutzenbehauptung aus Fixtures ableiten.

## Risiken und Verifikation

Hauptrisiken sind unvollständiges Inventar, überlappende oder fehlende
Zuständigkeit, zu große Agentenkontexte, Ressortkonflikte, ein gemeinsamer
Executor/Verifier unter bloß getrennten Namen, veraltete Stimmen nach
Kandidatenänderung, nichtatomare Promotion und automatische Kanonisierung
beobachteten Ist-Zustands. Verifikation muss deshalb vollständige
Pfadklassifikation gegen eine owner-definierte Eingabe, exakte Source- und
Mandatsdigests, Scope- und DAG-Grenzen, negativen Review, Kandidatenwechsel,
gleichzeitige aktive Änderungen sowie Abbruch/Wiederaufnahme abdecken.

Jede Lieferstufe bekommt eigene Source-/Gate-Belege und einen klaren
Capability-Status: geplant, implementiert, getestet, im Laufzeitversuch
beobachtet oder vom Owner angenommen. Die veröffentlichte Classic-Variante
bleibt währenddessen eigenständig prüf- und releasefähig. Government ist eine
experimentelle Architekturvariante; dieses Dokument entscheidet nicht, dass
sie die Produktidentität oder den Classic-Vertrag ersetzt.
