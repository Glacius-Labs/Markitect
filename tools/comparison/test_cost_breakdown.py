"""Attribution of an Apply's tokens, tool calls and time to Manager sessions."""

import json
from pathlib import Path
import tempfile
import unittest

import cost_breakdown

HIERARCHY = [{"Package": ".", "Parent": "", "Children": ["lending"]},
             {"Package": "lending", "Parent": ".", "Children": ["lending/reminders"]},
             {"Package": "lending/reminders", "Parent": "lending", "Children": None}]


def transcript(path, package, mandate, total, tools, start="2026-10-25T09:00:00.1234567Z", end="2026-10-25T09:01:30.5Z"):
    instructions = "\n".join(["Markitect harness v2", "", "Complete Manager hierarchy:", json.dumps(HIERARCHY), "",
                              "Manager:", json.dumps({"Package": package}), "", "Mandate:", json.dumps(mandate), ""])
    events = [(start, {"method": "thread/start", "params": {"developerInstructions": instructions}}),
              (start, {"method": "turn/start", "params": {}}),
              (start, {"method": "thread/tokenUsage/updated", "params": {"tokenUsage": {"total": {"totalTokens": 1, "outputTokens": 1}}}})]
    events += [(start, {"method": "item/completed", "params": {"item": item}}) for item in tools]
    events.append((end, {"method": "thread/tokenUsage/updated", "params": {"tokenUsage": {"total": total}}}))
    with path.open("w", encoding="utf-8") as stream:
        for at, message in events:
            stream.write(json.dumps({"at": at, "direction": "receive", "data": json.dumps(message) + "\n"}) + "\n")


class CostBreakdownTests(unittest.TestCase):
    def test_sessions_are_attributed_and_grouped(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            transcript(root / "a.jsonl", ".", "assessment", {"totalTokens": 100, "inputTokens": 90, "cachedInputTokens": 80, "outputTokens": 10},
                       [{"type": "dynamicToolCall", "tool": "delegate"}, {"type": "dynamicToolCall", "tool": "await"}])
            transcript(root / "b.jsonl", "lending/reminders", "implement", {"totalTokens": 50, "inputTokens": 45, "outputTokens": 5},
                       [{"type": "commandExecution"}, {"type": "fileChange"}, {"type": "agentMessage"}])
            result = cost_breakdown.breakdown([root])
        rows = {row["transcript"]: row for row in result["sessions"]}
        self.assertEqual((rows["a.jsonl"]["package"], rows["a.jsonl"]["mandate"], rows["a.jsonl"]["depth"]), (".", "assessment", 0))
        self.assertEqual((rows["b.jsonl"]["depth"], rows["b.jsonl"]["totalTokens"], rows["b.jsonl"]["turns"]), (2, 50, 1))
        # The last usage event is the session's total; the earlier one is superseded.
        self.assertEqual(result["total"]["totalTokens"], 150)
        self.assertEqual(result["by_mandate"]["assessment"]["tool_calls"], 2)
        self.assertEqual(result["tool_calls"], {"await": 1, "commandExecution": 1, "delegate": 1, "fileChange": 1})
        self.assertEqual(result["by_depth"]["2"]["outputTokens"], 5)
        self.assertEqual(rows["a.jsonl"]["seconds"], 90.4)


if __name__ == "__main__":
    unittest.main()
