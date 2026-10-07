"""Known deterministic external fixture, never a model or product-arm simulation."""
import argparse
import json
from pathlib import Path
import subprocess
import sys
import time

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("case", choices=["success", "no_usage", "sleep", "child"])
    parser.add_argument("dispatch_id")
    args = parser.parse_args()
    with Path("launches.jsonl").open("a", encoding="utf-8") as out:
        out.write(json.dumps({"dispatchId": args.dispatch_id}) + "\n")
    if args.case == "child":
        subprocess.Popen([sys.executable, "-c", "import time; from pathlib import Path; time.sleep(2); Path('escaped-child').write_text('bad')"])
    if args.case in {"sleep", "child"}:
        time.sleep(30)
    print(json.dumps({"type": "mechanical.completed", "dispatchId": args.dispatch_id}), flush=True)
    if args.case == "success":
        print(json.dumps({"type": "mechanical.usage", "providerRequests": 1, "input_tokens": 3,
                          "output_tokens": 2, "counterScope": "synthetic-only"}), flush=True)
