# Icon Generation

This directory contains the scripts and source assets used to generate the application icons for Gopher Automate.

## Contents

- `process_icons.py`: Python script that modifies the canonical Gopher SVG to create the App Icon (Blue) and Tray Icons (White/Black).
- `gopher_canonical.svg`: The official Go Gopher vector source (from `golang-samples`).

## How to Run

1. Ensure you have Python 3 and ImageMagick installed (specifically the `magick` command).
2. Run the script:
   ```bash
   python3 process_icons.py
   ```

## Output

The script generates the following in the `../../assets/` directory:

- `app_icon.png` (Blue App Icon)
- `tray_icon.png` (White Tray Icon with transparent pads)
- `tray_icon_black.png` (Black Tray Icon with transparent pads)
