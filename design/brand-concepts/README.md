# Markitect identity preview

The open M and titanium `#c4cac8` are the current product direction. The
canonical vector is [`assets/markitect-mark.svg`](../../assets/markitect-mark.svg).
The 512 px square export, 32 px size probe, and 1280 x 640 px GitHub social
preview are generated from that SVG by `export_avatar.py` using Pillow. Run
`python design/brand-concepts/export_avatar.py` after changing the vector.

[`index.html`](index.html) shows the square icon and the exact README heading
treatment. The actual repository `README.md` now uses the 512 px PNG inline
with its title, so the icon remains legible in light and dark GitHub themes.
The product mark stays separate from the Glacius Labs ice blue,
which is reserved for the parent brand. No organization avatar or remote GitHub
setting is changed by these files.

This is a local candidate, not a published release asset. The supplied image
informed the five-part geometric M but was not traced. Markitect's CLI, schemas,
and managed outputs are unaffected.
