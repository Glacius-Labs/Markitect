"""Give a bound demonstration attempt the profile names of the trial projects. Usage: bind_profiles.py ATTEMPT_DIR PYTHON_IMAGE_ID

`experiments/recursive-apply-demo/ra_bind.sh` writes a Host profile for the
Library: the agent profile `library-agent` on the Go image, a system prompt
that names the Go standard library, and the check profile `library-go`. The
trial projects declare `project-agent` and `python-tests` in their execution
settings. This rewrite renames the agent profile, puts it on the Python example
image (packaging/runtime/examples/python, built on the attempt's base image),
drops the Go sentence from the system prompt, appends the check profile
`python-tests` with that image's interpreter wrapper and records the change in
binding.txt. Like rx_python_profile.py it grants no execution authority.
"""
import hashlib
import sys
from pathlib import Path

GO_SENTENCE = " Use only the Go standard library for the realization."


def rebind(text, image):
    if "\n  project-agent:\n" in text or "\n  python-tests:\n" in text:
        raise SystemExit("the Host profile already has the trial profiles")
    for old, new in (("\n  library-agent:\n", "\n  project-agent:\n"), (GO_SENTENCE, "")):
        if text.count(old) != 1:
            raise SystemExit("expected exactly one %r in the Host profile" % old.strip())
        text = text.replace(old, new)
    first = text.index("imageID: ")  # the agent profile comes first
    end = text.index("\n", first)
    text = text[:first] + "imageID: " + image + text[end:]
    return text + f"""  python-tests:
    container:
      profileVersion: 1
      engineExecutable: /usr/bin/docker
      imageID: {image}
      executable: /opt/markitect/bin/python
      argumentPrefix: []
      env: ["PYTHONDONTWRITEBYTECODE=1"]
      limits:
        milliCPUs: 2000
        memoryBytes: 4294967296
        processes: 1024
      maxOutputBytes: 1048576
"""


def main(argv):
    attempt, image = Path(argv[1]), argv[2]
    host = attempt / "host" / "host.yaml"
    host.write_text(rebind(host.read_text(encoding="utf-8"), image), encoding="utf-8", newline="\n")
    with open(attempt / "binding.txt", "a", encoding="utf-8") as binding:
        binding.write("host_profile_note=trial profiles: project-agent on %s (Python example image), Go sentence "
                      "removed from the system prompt, python-tests check profile appended after binding\n" % image)
        binding.write("host_profile_sha256_after_rebind=%s\n" % hashlib.sha256(host.read_bytes()).hexdigest())
    print(host)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
