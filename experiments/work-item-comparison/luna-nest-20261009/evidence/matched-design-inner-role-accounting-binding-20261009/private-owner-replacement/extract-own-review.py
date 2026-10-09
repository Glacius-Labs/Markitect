import json
from pathlib import Path

root = Path(__file__).parent
session = Path(r'C:\Users\Consiliari\.codex\sessions\2026\10\08\rollout-2026-10-08T19-12-07-01a11c80-37aa-7fe0-9586-35d916ce6561.jsonl')
records = []
with session.open(encoding='utf-8') as handle:
    for line in handle:
        row = json.loads(line)
        timestamp = row.get('timestamp', '')
        if not ('2026-10-09T09:42:18' <= timestamp < '2026-10-09T09:52:00'):
            continue
        payload = row.get('payload', {})
        if row.get('type') not in ('response_item', 'event_msg'):
            continue
        encoded = json.dumps(payload)
        if 'policy_source_review' in encoded:
            records.append(row)
(root / 'review-session-records.json').write_text(json.dumps(records, indent=2), encoding='utf-8')
for row in records:
    payload = row.get('payload', {})
    print(row['timestamp'], row['type'], payload.get('type'), payload.get('name', ''), json.dumps(payload)[:400])
