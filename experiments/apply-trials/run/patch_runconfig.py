"""Give a trial run configuration a neutral repository identity. Usage: patch_runconfig.py RUN_CONFIG IDENTITY

The demonstration harness names the repository after the Library; Agents get no hint from the trial name.
"""
import json
import sys
from pathlib import Path

path, identity = Path(sys.argv[1]), sys.argv[2]
config = json.loads(path.read_text())
config["repositoryIdentity"] = identity
path.write_text(json.dumps(config, indent=1))
