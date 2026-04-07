# color-converter

Converts colors between HEX, RGB, HSL, and HSV formats.
Generates accessible contrast ratios and luminance.
No extra packages needed — uses Python stdlib colorsys.


Version: 1.0.0

## Usage

```yaml
run:
  component:
    name: color-converter
    with:
      color: "" # Input color value. HEX (#rrggbb or #rgb), RGB (r,g,b), HSL (h,s%,l%), or color name.  # required
      from_format: "" # Source format: 'hex', 'rgb', 'hsl', 'hsv', or 'auto' (default).
      to_format: "" # Target format: 'all' (default), 'hex', 'rgb', 'hsl', 'hsv'.
```

## Install

```bash
kdeps component install color-converter
```
