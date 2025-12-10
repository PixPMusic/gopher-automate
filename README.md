# GopherAutomate

> **Note:** This project is currently in development and is not yet ready for production use. It's a re-implementation of a previously-very-hacked-together tool I used to maintain at a previous job, but I no longer have access to the original code. As it's not needed for my daily workflow, progress will be slow, but I'm trying to make it far more user friendly this time around.
> Currently, only the lighting features have been implemented, and it only supports Novation's Launchpad S and Mini MK3--others may work, but I haven't tested them, nor have I reviewed their programmers guides.

A cross-platform automation tool using MIDI triggers, built with Go and [Fyne](https://fyne.io/).

Configure your MIDI controllers to trigger actions and manage device layouts—all from the system tray.

## Features

- **System Tray Integration** - Runs quietly in the background, accessible from your menu bar
- **MIDI Device Management** - Detect and configure connected MIDI devices
- **Layout Configuration** - Create and switch between custom control layouts
- **Persistent Settings** - Your configuration is saved and restored automatically

## Installation

### Pre-built Binaries

Download the latest release for your platform from the [Releases](https://github.com/PixPMusic/gopher-automate/releases) page.

If you encounter an "The app is damaged and can't be opened. You should move it to the Trash." error when running the pre-built macOS app, try running this first:

```bash
xattr -d com.apple.quarantine GopherAutomate.app
```

The app likely isn't damaged, but is instead triggering Gatekeeper. The error message is a bit misleading, as it should read "This app is from an unidentified developer." and let you open it by using `System Settings > Security & Privacy > General`, but this seems to not always work.

### Build from Source

Requires Go 1.24+ and the [Fyne CLI](https://docs.fyne.io/started/).

```bash
# Clone the repository
git clone https://github.com/PixPMusic/gopher-automate.git
cd gopher-automate

# Install dependencies
go mod download

# Build for your platform
# macOS
./build_mac.sh

# Or use fyne directly for other platforms
fyne package -os darwin -icon assets/icon.png -name GopherAutomate
fyne package -os windows -icon assets/icon.png -name GopherAutomate
fyne package -os linux -icon assets/icon.png -name GopherAutomate
```

## Usage

1. Launch **GopherAutomate** - it will appear in your system tray
2. Click the tray icon and select **Open** to configure your MIDI devices
3. Connect your MIDI controller and select it from the device list
4. Configure your layout and save

## Roadmap

- [x] ~~Device Management~~
- [x] ~~Layout Editor~~
- [ ] AppleScript Support
- [ ] Shell Command Support
- [ ] Inter-App Communication

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Update `CHANGELOG.md` with your changes
4. Commit your changes (`git commit -m 'feat: add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

## License

MIT License - see [LICENSE](LICENSE) for details.
