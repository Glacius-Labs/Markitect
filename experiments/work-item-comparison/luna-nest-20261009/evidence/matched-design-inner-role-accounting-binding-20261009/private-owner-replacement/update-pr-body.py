from pathlib import Path

root = Path(__file__).parent
original = Path(r'C:\Users\Consiliari\AppData\Local\Temp\markitect-native-workflow-20261009\pr-body.md').read_text(encoding='utf-8')
body = original.replace('Source `2b7b22ba` has a fresh hosted run, with final platform gates still pending.',
                        'Source `2b7b22ba` has completed hosted CI on Linux and Windows; replacement source requires its own final gates.')
body = body.replace('its hosted CI remains pending. It has no further real-agent proof.',
                    'its hosted CI passed on Linux and Windows. It has no further real-agent proof.')
body = body.replace('Source remains clean/pushed at `2b7b22ba`, and its existing CI continues without replacing the prior failed native proof pin.',
                    'That installation handoff retains its clean/pushed source-2b pin and subsequently successful hosted CI, without replacing the prior failed native proof pin.')
body += '''

A separate bounded source correction at `bcd614a3bfcd335e0ca18850993742057103c092` removes actual `--ignore-user-config` and `--ignore-rules` from the native Python adapter. Normal CLI policy stays enabled, while read-only/tool restrictions, finite timeout, Luna High and invocation-bound response validation remain. Both allowed test batches passed: 41 mocked adapter regressions and focused model-first onboarding/root/nested-help checks. The sole fresh read-only Luna-High source review found only an omitted `multi_agent` README entry, now repaired. No provider call or new product job was made. New hosted run: https://github.com/Glacius-Labs/Markitect/actions/runs/37914238147 (pending at this checkpoint).

The one permitted production build passed and is byte-identical to the source-2b Go binary because only Python/tests/docs changed. The replacement adapter SHA-256 is `1284ea4e187cf68ab8a9c15062366cadc750d63a9deb5b833214ec05bcc56ba6`; replacement runtime SHA-256 is `c3d4204490a9d1d5230cc151f99a63dcb8357bcc425022d040c48a0888afc44e`. Exact six runtime pins preserve other settings; the replacement configuration has not been installed or activated. The old installation/archive and failed-job evidence remain unchanged and must not supply the old adapter for a new native start. The machine-readable replacement handoff is `%TEMP%/markitect-native-policy-preservation-20261009/handoff.json`, SHA-256 `2b5615e9231208990f8a4748bb6d022dda9b2d5201b9e851f205574e469f1c76`; replacement archive SHA-256 is `1eb592b4560204ed799eeaae4386c559a483cb1e1a27aecce2a2857e458f933a`. No installation, provider/transport probe, fourth product proof, study allocation or readiness claim was added. Draft status remains until the outstanding actual delivery and acceptance gates close.
'''
(root / 'pr-body.md').write_text(body, encoding='utf-8')
