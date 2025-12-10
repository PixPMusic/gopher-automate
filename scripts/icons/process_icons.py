import xml.etree.ElementTree as ET
import copy

CANONICAL_PATH = "gopher_canonical.svg"
APP_ICON_PATH = "app_icon.svg"
TRAY_ICON_PATH = "tray_icon.svg"

# Namespaces
ET.register_namespace('', "http://www.w3.org/2000/svg")
ET.register_namespace('xlink', "http://www.w3.org/1999/xlink")

def load_canonical():
    tree = ET.parse(CANONICAL_PATH)
    root = tree.getroot()
    return tree, root

def create_pads(color='#333333'):
    # Create the 4 MIDI pads as a Group
    # Coordinates estimated based on ViewBox 0 0 401.98 559.472
    # Center X ~ 201. 
    g = ET.Element('{http://www.w3.org/2000/svg}g', {'id': 'midi-pads'})
    
    # User requested "Twice as big". 
    # Old Size: 45. New Size: 85.
    
    size = 85 
    gap = 14   
    
    # Calculate Centering
    # Total Width = size + gap + size = 85 + 14 + 85 = 184
    # Center X = 201
    # Start X = 201 - (184 / 2) = 201 - 92 = 109
    
    start_x = 109
    
    # Y Position
    # Needs to move up to accommodate larger height.
    # Previous Y: 300. Height was 98. End Y: 398.
    # New Height: 184. 
    # To keep center roughly similar or slightly lower on belly?
    # Let's try starting at 260. End Y = 260 + 184 = 444. 
    # The body bottom curve starts tightening around 450-500. This should fit.
    
    start_y = 260
    
    pads_coords = [
        (start_x, start_y), (start_x + size + gap, start_y),
        (start_x, start_y + size + gap), (start_x + size + gap, start_y + size + gap)
    ]
    
    for x, y in pads_coords:
        rect = ET.Element('{http://www.w3.org/2000/svg}rect', {
            'x': str(x),
            'y': str(y),
            'width': str(size),
            'height': str(size),
            'rx': '10', # Increased corner radius for larger size
            'ry': '10',
            'fill': color
        })
        g.append(rect)
    return g

def make_square(root):
    # Get original dimensions from viewBox
    vb = root.attrib.get('viewBox', '0 0 401.98 559.472').split()
    x, y, w, h = map(float, vb)
    
    target_dim = max(w, h)
    dx = (target_dim - w) / 2
    dy = (target_dim - h) / 2
    
    # Create new root with square dimensions
    new_root = ET.Element('{http://www.w3.org/2000/svg}svg', {
        'viewBox': f"0 0 {target_dim} {target_dim}",
        'width': f"{target_dim}",
        'height': f"{target_dim}",
        'version': "1.1"
    })
    
    # Wrap content in a group for centering
    content_group = ET.Element('{http://www.w3.org/2000/svg}g', {
        'transform': f"translate({dx}, {dy})"
    })
    
    for child in list(root):
        content_group.append(child)
        
    new_root.append(content_group)
    return new_root

def create_app_icon():
    tree, root = load_canonical()
    
    # 1. Update Blue Color to Official Go Blue
    for elem in root.iter():
        if 'fill' in elem.attrib and elem.attrib['fill'].upper() == '#6AD7E5':
            elem.attrib['fill'] = '#00ADD8'
            
    # 2. Add Pads - WHITE
    pads = create_pads(color='#FFFFFF')
    root.append(pads)
    
    # 3. Square the Canvas
    new_root = make_square(root)
    
    # Write new tree
    new_tree = ET.ElementTree(new_root)
    new_tree.write(APP_ICON_PATH, encoding='UTF-8', xml_declaration=True)
    print(f"Created {APP_ICON_PATH}")

def create_tray_body(filename, color):
    tree, root = load_canonical()
    
    old_children = list(root)
    for child in old_children:
        root.remove(child)
        
    body_group = ET.Element('{http://www.w3.org/2000/svg}g')
    
    # Process all elements to function as a solid silhouette of the given color
    for elem in old_children:
        has_fill = False
        if 'fill' in elem.attrib and elem.attrib['fill'] != 'none':
            has_fill = True
        elif 'fill' not in elem.attrib:
            has_fill = True
            
        if has_fill:
            elem.set('fill', color)
        
        if 'stroke' in elem.attrib:
            elem.set('stroke', color)
            
        for sub in elem.iter():
            if sub == elem: continue
            
            s_has_fill = False
            if 'fill' in sub.attrib and sub.attrib['fill'] != 'none':
                s_has_fill = True
            elif 'fill' not in sub.attrib:
                s_has_fill = True 
            
            if s_has_fill:
                sub.set('fill', color)
            
            if 'stroke' in sub.attrib:
                sub.set('stroke', color)

        body_group.append(elem)
        
    root.append(body_group)
    
    # Square the canvas
    new_root = make_square(root)
    
    new_tree = ET.ElementTree(new_root)
    new_tree.write(filename, encoding='UTF-8', xml_declaration=True)
    print(f"Created {filename}")

def create_tray_layers():
    # 1. GENERATE TRAY BODY (White)
    create_tray_body("tray_body.svg", "white")
    
    # 2. GENERATE TRAY BODY (Black)
    create_tray_body("tray_body_black.svg", "black")
    
    # 3. GENERATE PADS (For Cutout)
    tree_pads, root_pads = load_canonical()
    for c in list(root_pads):
        root_pads.remove(c)
        
    pads_g = create_pads(color='black') 
    root_pads.append(pads_g)
    
    # Square the canvas for pads too so alignment matches
    new_root_pads = make_square(root_pads)
    
    new_tree_pads = ET.ElementTree(new_root_pads)
    new_tree_pads.write("tray_pads.svg", encoding='UTF-8', xml_declaration=True)
    print("Created tray_pads.svg")

import subprocess
import os

# ... imports ...

def render_svg(svg_path, png_path):
    # Use magick to convert SVG to PNG
    try:
        subprocess.run(["magick", "-background", "none", svg_path, png_path], check=True)
        print(f"Rendered {png_path}")
    except subprocess.CalledProcessError as e:
        print(f"Error rendering {svg_path}: {e}")

def composite_tray_icon(body_png, pads_png, output_png):
    # Use magick composite DstOut to subtract pads from body
    try:
        subprocess.run([
            "magick", 
            body_png, 
            pads_png, 
            "-compose", "DstOut", 
            "-composite", 
            output_png
        ], check=True)
        print(f"Composited {output_png}")
    except subprocess.CalledProcessError as e:
        print(f"Error compositing {output_png}: {e}")

def cleanup_temp_files(files):
    for f in files:
        if os.path.exists(f):
            os.remove(f)
            print(f"Removed temp file {f}")

if __name__ == "__main__":
    # 1. Create App Icon (SVG)
    create_app_icon() 
    
    # 2. Create Tray Layers (SVGs)
    create_tray_layers()
    
    # 3. Render Layers to PNG
    # Note: SVGs and intermediate PNGs remain in CWD (scripts/icons) but are cleaned up.
    # App Icon: Render directly to assets
    render_svg("app_icon.svg", "../../assets/app_icon.png")
    
    render_svg("tray_body.svg", "tray_body.png")
    render_svg("tray_body_black.svg", "tray_body_black.png")
    render_svg("tray_pads.svg", "tray_pads.png")
    
    # 4. Composite Final Tray Icons
    # Output directly to assets
    composite_tray_icon("tray_body.png", "tray_pads.png", "../../assets/tray_icon.png")             # Light Mode
    composite_tray_icon("tray_body_black.png", "tray_pads.png", "../../assets/tray_icon_black.png") # Dark Mode
    
    # 5. Cleanup
    cleanup_temp_files([
        "app_icon.svg", 
        "tray_body.svg", "tray_body.png",
        "tray_body_black.svg", "tray_body_black.png",
        "tray_pads.svg", "tray_pads.png"
    ])
    
    print("Icon generation complete.")
