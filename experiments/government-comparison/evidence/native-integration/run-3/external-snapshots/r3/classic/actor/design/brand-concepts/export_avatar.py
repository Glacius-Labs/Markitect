"""Export the square Markitect avatar from the canonical open-M SVG.

Requires Pillow. The narrow path parser intentionally supports only the SVG
commands used by this mark; unsupported commands fail instead of silently
changing the exported geometry.
"""

from pathlib import Path
import re
import xml.etree.ElementTree as ET

from PIL import Image, ImageDraw


HERE = Path(__file__).resolve().parent
ASSETS = HERE.parents[1] / "assets"
SOURCE = ASSETS / "markitect-mark.svg"
TOKEN = re.compile(r"[MmLlHhVvZz]|[-+]?(?:\d+(?:\.\d*)?|\.\d+)")
CANVAS = 512
OVERSAMPLE = 4
PADDING = 96


def points(path: str) -> list[tuple[float, float]]:
    tokens = TOKEN.findall(path)
    output: list[tuple[float, float]] = []
    command = ""
    x = y = 0.0
    start = (0.0, 0.0)
    i = 0

    while i < len(tokens):
        if tokens[i].isalpha():
            command = tokens[i]
            i += 1
            if command in "Zz":
                output.append(start)
                x, y = start
                command = ""
                continue

        if command in "MmLl":
            dx, dy = float(tokens[i]), float(tokens[i + 1])
            i += 2
            x, y = (x + dx, y + dy) if command.islower() else (dx, dy)
            if command in "Mm":
                start = (x, y)
                command = "l" if command == "m" else "L"
        elif command in "Hh":
            value = float(tokens[i])
            i += 1
            x = x + value if command == "h" else value
        elif command in "Vv":
            value = float(tokens[i])
            i += 1
            y = y + value if command == "v" else value
        else:
            raise ValueError(f"Unsupported SVG path command: {command!r}")

        output.append((x, y))

    return output


root = ET.parse(SOURCE).getroot()
namespace = {"svg": "http://www.w3.org/2000/svg"}
group = root.find("svg:g", namespace)
if group is None or not group.attrib.get("fill"):
    raise ValueError("Expected one colored SVG group")

polygons = [points(path.attrib["d"]) for path in group.findall("svg:path", namespace)]
if len(polygons) != 5:
    raise ValueError("Expected the open M to contain five pieces")

all_points = [point for polygon in polygons for point in polygon]
left = min(x for x, _ in all_points)
right = max(x for x, _ in all_points)
top = min(y for _, y in all_points)
bottom = max(y for _, y in all_points)
scale = (CANVAS - 2 * PADDING) / max(right - left, bottom - top)
offset_x = (CANVAS - (right - left) * scale) / 2 - left * scale
offset_y = (CANVAS - (bottom - top) * scale) / 2 - top * scale

image = Image.new("RGB", (CANVAS * OVERSAMPLE,) * 2, "#212121")
draw = ImageDraw.Draw(image)
for polygon in polygons:
    draw.polygon(
        [
            ((x * scale + offset_x) * OVERSAMPLE, (y * scale + offset_y) * OVERSAMPLE)
            for x, y in polygon
        ],
        fill=group.attrib["fill"],
    )

avatar = image.resize((CANVAS, CANVAS), Image.Resampling.LANCZOS)
avatar.save(ASSETS / "markitect-avatar-512.png", optimize=True)
avatar.resize((32, 32), Image.Resampling.LANCZOS).save(
    ASSETS / "markitect-avatar-32.png", optimize=True
)

social = Image.new("RGB", (1280, 640), "#212121")
social.paste(avatar.resize((640, 640), Image.Resampling.LANCZOS), (320, 0))
social.save(ASSETS / "markitect-social-preview.png", optimize=True)
