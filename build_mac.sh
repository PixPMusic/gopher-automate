#!/bin/bash

APP_NAME="GopherAutomate"
APP_ID="cc.pixp.GopherAutomate"
ICON_SOURCE="assets/app_icon.png"

echo "Building $APP_NAME with Fyne..."

# Extract version from CHANGELOG.md (first H2 like ## [0.0.1])
# We must skip named releases like [Unreleased] that don't match X.Y.Z
VERSION=$(grep -m 1 "^## \[[0-9]" CHANGELOG.md | sed -E 's/## \[([^]]+)\].*/\1/')
if [ -z "$VERSION" ]; then
    VERSION="0.0.1"
    echo "No version found in CHANGELOG, defaulting to $VERSION"
fi
echo "Detected version: $VERSION"

# Check if fyne tool is installed
if ! command -v fyne &> /dev/null; then
    echo "Installing Fyne CLI..."
    go install fyne.io/tools/cmd/fyne@latest
fi

# Package the app using Fyne CLI
echo "Packaging..."
fyne package -os darwin -icon "$ICON_SOURCE" -name "$APP_NAME" -app-id "$APP_ID" -app-version "$VERSION"



if [ $? -ne 0 ]; then
    echo "Packaging failed!"
    exit 1
fi

# Post-process Info.plist to hide from Dock
echo "Configuring as agent app (hiding from Dock)..."
PLIST="$APP_NAME.app/Contents/Info.plist"

# Use plutil to insert LSUIElement=true
# We use a temporary file approach to ensure clean XML
plutil -insert LSUIElement -bool true "$PLIST"

# Force LaunchServices refresh
echo "Forcing LaunchServices refresh..."
touch "$APP_NAME.app"

echo "Done! $APP_NAME.app created."
