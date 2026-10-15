# Source project description

This is the original ordinary-language example developed in the owner conversation before extracting statement kinds. It is an input record, not a second normative specification. Current candidate obligations live in the seven-kind records linked from the [package map](README.md). Later editorial choices are identified in [Decisions](decisions.md).

## Original description

**Projektbeschreibung: Lokale Aufgabenverwaltung**

**Zweck und Umfang**

Die Anwendung hilft einer Person, ihre offenen Aufgaben im Blick zu behalten. Sie wird lokal auf einem Rechner über die Kommandozeile bedient. Aufgaben bleiben nach dem Beenden und Neustarten erhalten.

Die erste Fassung unterstützt das Anlegen, Auflisten und Erledigen von Aufgaben. Gemeinsame Nutzung, Synchronisierung und Erinnerungen liegen außerhalb ihres Umfangs.

**Verhalten und Nutzung**

Eine Aufgabe besitzt einen Text und den Zustand „offen“ oder „erledigt“.

Beim Anlegen gibt der Benutzer einen Text ein. Die Anwendung legt die Aufgabe als offen an und bestätigt den Vorgang. Ein leerer oder ausschließlich aus Leerzeichen bestehender Text wird mit einer verständlichen Meldung abgelehnt.

Beim Auflisten zeigt die Anwendung alle Aufgaben mit ihrem Text und Zustand. Jede Aufgabe lässt sich eindeutig auswählen. Gibt es keine Aufgaben, zeigt die Anwendung dies an.

Der Benutzer kann eine offene Aufgabe als erledigt markieren. Eine bereits erledigte Aufgabe bleibt erledigt. Wird eine unbekannte Aufgabe ausgewählt, erscheint eine verständliche Fehlermeldung.

**Aufbau**

Die Anwendung hat drei Verantwortungsbereiche:

- Die Bedienung nimmt Eingaben entgegen und zeigt Ergebnisse an.
- Die Aufgabenverwaltung führt die angebotenen Aktionen aus.
- Die Speicherung liest und schreibt Aufgaben in einer lokalen Datei.

Diese Aufteilung beschreibt die Verantwortlichkeiten. Ihre konkrete Umsetzung in Dateien oder Programmeinheiten bleibt offen.

**Repository und Betrieb**

Das Repository enthält den Quellcode, automatisierte Tests, eine Bedienungsanleitung und die Hinweise für Entwicklungsagenten.

Die Bedienungsanleitung erklärt Einrichtung, Start, verfügbare Aktionen und den Speicherort der Aufgaben. Die Agentenhinweise verweisen auf die maßgebliche Projektbeschreibung und den Entwicklungsablauf.

Kann die Anwendung gespeicherte Aufgaben nicht lesen oder Änderungen nicht speichern, meldet sie den Fehler. Sie bestätigt eine Änderung erst nach erfolgreicher Speicherung. Unlesbare vorhandene Daten werden nicht durch einen leeren Aufgabenbestand überschrieben.

**Entwicklung und Prüfung**

Änderungen berücksichtigen das beschriebene Verhalten, die Tests und die betroffene Dokumentation. Neue Fähigkeiten oder geänderte Anforderungen werden in der Projektbeschreibung festgehalten.

Automatisierte Tests prüfen die angebotenen Aktionen und ihre Fehlerfälle. Ein durchgehender Test legt eine Aufgabe an, beendet die Anwendung, startet sie erneut und prüft, dass die Aufgabe weiterhin vorhanden ist.

**Begründungen und offene Punkte**

Die erste Fassung bleibt klein, damit ihr Verhalten und ihre Umsetzung leicht nachvollziehbar sind. Die lokale Speicherung ermöglicht die Nutzung ohne zusätzlichen Dienst.

Offen sind die Programmiersprache, die genauen Befehle und das Dateiformat. Diese Entscheidungen müssen das beschriebene Verhalten erfüllen.
