# Archivierte Entwurfsproben

Diese Quellen waren bei den Designgesprächen temporäre Prüfprogramme. Sie sind hier bytegetreu archiviert, damit eine andere Session die Berichte nachvollziehen kann. Sie sind keine Markitect-Produktkomponenten und keine vollständige Implementierung des aktuellen [Sprachvertrags](../language-contract.md).

Die Python-Prototypen haben dieselben Inhaltsdigests wie die in bootstrap/probe.json, bootstrap/compact-probe.json und identities/probe.json gebundenen Quellen. Die beiden Go-Programme und inputs.json archivieren den älteren Durchstich gegen den bestehenden Core nach Adaption von definition auf purpose. Die Archive werden nicht nachträglich an neuere Designentscheidungen angepasst.

Vom Repositoryroot mit Python und PyYAML:

```text
python -B docs/design/canonical-model/minimal-language/probes/replay.py
```

Mit der laut CONTRIBUTING passenden Go-Version zusätzlich:

```text
python -B docs/design/canonical-model/minimal-language/probes/replay.py --with-go
```

Der Replay prüft Eingabe-/Prototypdigests und führt die Proben in einem neuen temporären Arbeitsverzeichnis aus. Neu erzeugte Python-Berichte werden bis auf sourceRevision mit den historischen Berichten verglichen. Bei den älteren Go-Proben wird deren erzeugter Berichtsteil mit dem gespeicherten Bericht verglichen; seine später ergänzten Provenienzfelder bleiben erhalten. Die Originale werden nicht verändert. Unveränderte Ergebnisse am aktuellen Head sind eine erneute begrenzte Prüfung, keine rückwirkende Änderung der historischen Evidenz.

Die Python-Archive erwarten ihren damaligen Bootstrap-Import unter TEMP. Der Replay stellt genau diese Umgebung nur für seine Kindprozesse bereit. Ein Go-Overlay legt den virtuellen Einstieg unter src/cmd/markitect an, damit die bestehenden internal-Imports gültig bleiben; am Repository wird dabei keine Quelldatei erzeugt. Die Go-Quellen sind deshalb als .go.txt gespeichert und werden nicht versehentlich zu einem neuen Go-Paket.

Der Compilerprototyp versteht keine vollständige Körperlinkgrammatik und keinen semantischen Index. Die M3/M4-Reviews im Sprachvertrag sind weiterhin angenommene Ergebnisse eines Designfalls. Es wird kein Connector, kein Broker und kein Agentenprovider gestartet.
