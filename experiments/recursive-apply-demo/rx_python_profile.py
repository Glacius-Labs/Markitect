"""Switch a bound attempt to the Python example runtime image. Usage: rx_python_profile.py ATTEMPT_DIR PYTHON_IMAGE_ID

For a Python project such as Readinglog: the agent profile runs on the Python
example image (packaging/runtime/examples/python, built on the attempt's base
image), and a check profile `readinglog-python` with that image's small
interpreter wrapper is offered, so an Extract can propose the documented
unittest command. The Go check profile stays. The change is recorded in
binding.txt."""
import pathlib
import sys

attempt, image = pathlib.Path(sys.argv[1]), sys.argv[2]
host = attempt / "host" / "host.yaml"
text = host.read_text()
first = text.index("imageID: ")  # the agent profile comes first
end = text.index("\n", first)
text = text[:first] + "imageID: " + image + text[end:]
text += f"""  readinglog-python:
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
host.write_text(text)
with open(attempt / "binding.txt", "a") as binding:
    binding.write(f"host_profile_note=agent profile on {image} (Python example image); readinglog-python check profile appended after binding\n")
print(host)
